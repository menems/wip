// Package jwt contains a driven adapter that implements auth.TokenService using JWT.
package jwt

import (
	"fmt"
	"os"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

type claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	gojwt.RegisteredClaims
}

// TokenService implements auth.TokenService using HMAC-signed JWTs.
type TokenService struct{}

// New constructs a TokenService.
func New() *TokenService {
	return &TokenService{}
}

func secret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "dev-secret-change-in-production"
	}
	return []byte(s)
}

// Generate creates a signed JWT for the given user.
func (s *TokenService) Generate(userID, email string) (string, error) {
	c := claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  gojwt.NewNumericDate(time.Now()),
		},
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, c)
	signed, err := token.SignedString(secret())
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Validate parses and validates a JWT, returning the embedded user identity.
func (s *TokenService) Validate(tokenStr string) (string, string, error) {
	token, err := gojwt.ParseWithClaims(tokenStr, &claims{}, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret(), nil
	})
	if err != nil {
		return "", "", fmt.Errorf("parse token: %w", err)
	}
	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return "", "", fmt.Errorf("invalid token")
	}
	return c.UserID, c.Email, nil
}
