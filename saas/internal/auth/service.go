package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/menems/saas/internal/user"
)

// UserFinder retrieves a user by email for authentication.
type UserFinder interface {
	GetByEmail(ctx context.Context, email string) (user.User, error)
}

// PasswordComparer verifies a password against a hash.
type PasswordComparer interface {
	Compare(hash, password string) error
}

// Service implements AuthService.
type Service struct {
	finder        UserFinder
	comparer      PasswordComparer
	signingKey    []byte
	tokenDuration time.Duration
}

// NewService creates a new auth service.
func NewService(finder UserFinder, comparer PasswordComparer, signingKey []byte, tokenDuration time.Duration) *Service {
	return &Service{
		finder:        finder,
		comparer:      comparer,
		signingKey:    signingKey,
		tokenDuration: tokenDuration,
	}
}

// jwtClaims embeds jwt.RegisteredClaims with custom fields.
type jwtClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Role  string `json:"role"`
}
