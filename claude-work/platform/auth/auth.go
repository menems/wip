package auth

import (
	"context"

	"github.com/google/uuid"
)

type contextKey struct{}

// TokenLookup resolves a session token to a user ID.
type TokenLookup interface {
	FindUserByToken(ctx context.Context, token string) (uuid.UUID, error)
}

// WithUserID returns a context carrying the given user ID.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// UserIDFromContext extracts the user ID injected by the auth interceptor.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(contextKey{}).(uuid.UUID)
	return id, ok
}
