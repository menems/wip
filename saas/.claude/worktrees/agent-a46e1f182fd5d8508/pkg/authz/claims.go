package authz

import (
	"context"
	"fmt"
	"time"
)

// Claims holds the data extracted from a verified JWT token.
type Claims struct {
	UserID    string
	Email     string
	Role      string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

// claimsKey is the context key for storing authenticated Claims.
type claimsKey struct{}

// ContextWithClaims returns a new context with the given claims attached.
func ContextWithClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, c)
}

// ClaimsFromContext extracts the authenticated Claims from the context.
// Returns an error if no claims are present.
func ClaimsFromContext(ctx context.Context) (Claims, error) {
	c, ok := ctx.Value(claimsKey{}).(Claims)
	if !ok {
		return Claims{}, fmt.Errorf("no claims in context")
	}
	return c, nil
}
