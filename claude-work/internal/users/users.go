package users

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Sentinel errors.
var (
	ErrNotFound       = errors.New("user not found")
	ErrConflict       = errors.New("user already exists")
	ErrBadCredentials = errors.New("invalid credentials")
)

// User is the domain type for a registered account.
type User struct {
	ID    uuid.UUID
	Name  string
	Email string
}

// Service is the interface for user business logic.
type Service interface {
	Register(ctx context.Context, name, email, password string) (User, error)
	SignIn(ctx context.Context, email, password string) (string, error)
	FindUserByToken(ctx context.Context, token string) (uuid.UUID, error)
}
