package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// authUserFinder is the user-lookup port the service depends on.
type authUserFinder interface {
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	FindUserByID(ctx context.Context, id uuid.UUID) (*User, error)
}

// authTokenStore is the token-persistence port the service depends on.
type authTokenStore interface {
	SaveRefreshToken(ctx context.Context, token *RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
}

// Service implements the application use cases for authentication.
// It depends only on the two repository ports and TokenConfig — no HTTP or DB types.
type Service struct {
	users  authUserFinder
	tokens authTokenStore
	cfg    TokenConfig
}

// NewService constructs an auth Service with the given repository ports and token config.
func NewService(uf authUserFinder, ts authTokenStore, cfg TokenConfig) *Service {
	return &Service{users: uf, tokens: ts, cfg: cfg}
}

// Login authenticates a user by email and password.
// Returns ErrInvalidCredentials for unknown email or wrong password.
// Returns ErrAccountDeactivated if the account is inactive.
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.users.FindUserByEmail(ctx, email)
	if err != nil {
		// Map ErrNotFound to ErrInvalidCredentials to avoid user enumeration.
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth: login: find user: %w", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountDeactivated
	}

	return s.issueTokens(ctx, user)
}

// Refresh validates a raw refresh token, rotates it, and issues a new token pair.
// Returns ErrTokenInvalid if the token is missing, expired, or revoked.
func (s *Service) Refresh(ctx context.Context, rawToken string) (*LoginResult, error) {
	hash := hashToken(rawToken)

	record, err := s.tokens.FindRefreshToken(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrTokenInvalid
		}
		return nil, fmt.Errorf("auth: refresh: find token: %w", err)
	}

	if !record.IsValid() {
		return nil, ErrTokenInvalid
	}

	user, err := s.users.FindUserByID(ctx, record.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth: refresh: find user: %w", err)
	}

	// Revoke the old token before issuing a new one (rotation).
	if err = s.tokens.RevokeRefreshToken(ctx, record.ID); err != nil {
		return nil, fmt.Errorf("auth: refresh: revoke old token: %w", err)
	}

	return s.issueTokens(ctx, user)
}

// Logout revokes the refresh token associated with rawToken.
// It is idempotent: if the token is already revoked or not found, no error is returned.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	hash := hashToken(rawToken)

	record, err := s.tokens.FindRefreshToken(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil // Already gone — treat as success.
		}
		return fmt.Errorf("auth: logout: find token: %w", err)
	}

	if record.RevokedAt != nil {
		return nil // Already revoked.
	}

	if err = s.tokens.RevokeRefreshToken(ctx, record.ID); err != nil {
		return fmt.Errorf("auth: logout: revoke token: %w", err)
	}

	return nil
}

// Me returns the authenticated user's profile by ID.
// Returns ErrNotFound if the user does not exist.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := s.users.FindUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth: me: %w", err)
	}
	return user, nil
}

// issueTokens creates a new access JWT and a new opaque refresh token for the user.
func (s *Service) issueTokens(ctx context.Context, user *User) (*LoginResult, error) {
	accessToken, err := s.signAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("auth: sign access token: %w", err)
	}

	rawRefresh, err := generateOpaqueToken()
	if err != nil {
		return nil, fmt.Errorf("auth: generate refresh token: %w", err)
	}

	record := &RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: time.Now().Add(s.cfg.RefreshTTL),
	}

	if err = s.tokens.SaveRefreshToken(ctx, record); err != nil {
		return nil, fmt.Errorf("auth: save refresh token: %w", err)
	}

	return &LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

// claims is the JWT payload structure.
type claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Name  string `json:"name"`
}

// signAccessToken creates a signed HS256 JWT for the given user.
// A random JWTID (jti) is included so that tokens issued within the same second
// are always distinct — important for token rotation and future blacklisting.
func (s *Service) signAccessToken(user *User) (string, error) {
	now := time.Now()
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTTL)),
		},
		Email: user.Email,
		Name:  user.Name,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(s.cfg.Secret)
}

// SignToken creates a signed access JWT for the given user fields.
// It is primarily used by tests in other packages that need a real, verifiable
// token without going through a full login flow.
func (s *Service) SignToken(userID uuid.UUID, email, name string) (string, error) {
	return s.signAccessToken(&User{ID: userID, Email: email, Name: name})
}

// VerifyAccessToken parses and validates an HS256 JWT, returning the subject UUID.
// This is used by the JWT middleware (task 1.5); it lives here to keep all JWT
// logic in one place.
func (s *Service) VerifyAccessToken(tokenString string) (uuid.UUID, error) {
	var c claims
	token, err := jwt.ParseWithClaims(tokenString, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.cfg.Secret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, fmt.Errorf("auth: invalid access token: %w", err)
	}

	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("auth: token subject is not a UUID: %w", err)
	}

	return id, nil
}

// generateOpaqueToken creates a cryptographically random 32-byte token,
// returned as a URL-safe base64 string suitable for use in a cookie.
func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// hashToken returns the lowercase hex SHA-256 of the raw token string.
// Only the hash is stored in the database; the raw token lives in the cookie.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
