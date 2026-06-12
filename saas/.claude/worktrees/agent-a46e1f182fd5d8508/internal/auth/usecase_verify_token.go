package auth

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// VerifyToken validates a JWT token and returns the embedded claims.
func (s *Service) VerifyToken(_ context.Context, tokenStr string) (Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(_ *jwt.Token) (any, error) {
		return s.signingKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		return Claims{}, fmt.Errorf("verify token: %w", err)
	}

	c, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return Claims{}, fmt.Errorf("verify token: invalid claims")
	}

	return Claims{
		UserID:    c.Subject,
		Email:     c.Email,
		Role:      c.Role,
		ExpiresAt: c.ExpiresAt.Time,
		IssuedAt:  c.IssuedAt.Time,
	}, nil
}
