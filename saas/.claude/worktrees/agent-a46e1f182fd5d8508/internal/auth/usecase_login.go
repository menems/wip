package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Login authenticates a user by email and password, returning a signed JWT token.
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.finder.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}

	if err := s.comparer.Compare(u.PasswordHash, password); err != nil {
		return "", fmt.Errorf("login: invalid credentials: %w", err)
	}

	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenDuration)),
		},
		Email: u.Email,
		Role:  string(u.Role),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.signingKey)
	if err != nil {
		return "", fmt.Errorf("login: sign token: %w", err)
	}

	return signed, nil
}
