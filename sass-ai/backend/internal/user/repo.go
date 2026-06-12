package user

import "context"

// Repository is the outbound port for user persistence.
// Implementations live in adapter/postgres.
type Repository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	// FindByEmail returns the user together with its password hash so the
	// auth service can verify credentials without exposing the hash on User.
	FindByEmail(ctx context.Context, email string) (*User, string, error)
	Create(ctx context.Context, email, passwordHash, name string) (*User, error)
	Update(ctx context.Context, id string, params UpdateParams) (*User, error)
}
