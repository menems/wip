package auth

import (
	"context"
	"errors"

	"github.com/menems/saas/pkg/authz"
)

// Sentinel errors for auth domain operations.
var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrValidation      = errors.New("auth validation failed")
)

// Claims is an alias for authz.Claims — JWT payload extracted by the interceptor.
type Claims = authz.Claims

// ContextWithClaims returns a new context with the given claims attached.
// Re-exported from pkg/authz for convenience.
var ContextWithClaims = authz.ContextWithClaims

// ClaimsFromContext extracts the authenticated Claims from the context.
// Re-exported from pkg/authz for convenience.
var ClaimsFromContext = authz.ClaimsFromContext

// AuthService defines the domain operations for authentication.
type AuthService interface {
	// Login authenticates a user by email and password, returning a signed JWT token.
	Login(ctx context.Context, email, password string) (string, error)

	// VerifyToken validates a JWT token and returns the embedded claims.
	VerifyToken(ctx context.Context, token string) (Claims, error)
}
