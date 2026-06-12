// Package users implements user management: listing, creation, editing,
// deactivation, reactivation, and admin-initiated password reset.
// It follows hexagonal architecture: domain types and port interfaces live here.
package users

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors that callers must handle.
var (
	// ErrNotFound is returned when a user does not exist.
	ErrNotFound = errors.New("not found")

	// ErrEmailConflict is returned when the requested email is already in use.
	ErrEmailConflict = errors.New("email already in use")

	// ErrLastAdmin is returned when deactivating the user would leave no active
	// admin (system-role holder) in the system.
	ErrLastAdmin = errors.New("cannot deactivate the last active admin")
)

// Role is a slim view of a role as needed by user management.
type Role struct {
	ID       uuid.UUID
	Name     string
	IsSystem bool // used for last-admin guard; not exposed in API
}

// User is the full user record including role assignments.
type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	IsActive     bool
	Roles        []Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// hasSystemRole reports whether the user holds at least one system role.
func (u *User) hasSystemRole() bool {
	for _, r := range u.Roles {
		if r.IsSystem {
			return true
		}
	}
	return false
}

// UserFilter carries query parameters for the list endpoint.
type UserFilter struct {
	Search  string // substring match against name and email
	SortBy  string // column name; validated against an allowlist in the repository
	SortDir string // "asc" or "desc"
	Page    int    // 1-based
	PerPage int    // max 100
}

// CreateParams carries validated fields for creating a new user.
type CreateParams struct {
	Email    string
	Name     string
	Password string
	RoleID   uuid.UUID
}

// UpdateParams carries validated fields for updating an existing user.
type UpdateParams struct {
	Email  string
	Name   string
	RoleID uuid.UUID
}

// UserService is the application port that the HTTP handler depends on.
// The concrete implementation lives in service.go; adapters and tests may
// provide their own implementations.
type UserService interface {
	// List returns a paginated, filtered list of users and the total count.
	List(ctx context.Context, filter UserFilter) ([]*User, int, error)

	// Get returns the user with the given ID.
	// Returns ErrNotFound if the user does not exist.
	Get(ctx context.Context, id uuid.UUID) (*User, error)

	// Create creates a new user and assigns them the specified role.
	// Returns ErrEmailConflict if the email is already taken.
	Create(ctx context.Context, cmd CreateParams) (*User, error)

	// Update applies name/email changes and replaces the user's role assignment.
	// Returns ErrNotFound if the user does not exist.
	// Returns ErrEmailConflict if the new email is already used by another user.
	Update(ctx context.Context, id uuid.UUID, cmd UpdateParams) (*User, error)

	// Deactivate sets is_active=false on the user.
	// Returns ErrNotFound if the user does not exist.
	// Returns ErrLastAdmin if the user is the last active admin in the system.
	Deactivate(ctx context.Context, id uuid.UUID) (*User, error)

	// Reactivate sets is_active=true on the user.
	// Returns ErrNotFound if the user does not exist.
	Reactivate(ctx context.Context, id uuid.UUID) (*User, error)

	// ResetPassword updates the user's password hash.
	// Returns ErrNotFound if the user does not exist.
	// Returns a validation error if the password is shorter than MinPasswordLength.
	ResetPassword(ctx context.Context, id uuid.UUID, password string) error
}
