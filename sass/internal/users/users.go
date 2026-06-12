// Package users implements user management (CRUD, lifecycle, password management).
// It follows hexagonal architecture: domain types and port interfaces live here;
// the application layer (service.go) depends only on split Repository interfaces;
// adapters (repository.go, handler.go) implement and consume them.
package users

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors that callers must handle.
var (
	// ErrNotFound is returned when the requested user does not exist.
	ErrNotFound = errors.New("not found")

	// ErrEmailConflict is returned when a user with the same email already exists.
	ErrEmailConflict = errors.New("email already in use")

	// ErrLastAdmin is returned when trying to deactivate the last active system-role user.
	ErrLastAdmin = errors.New("cannot deactivate the last active admin")

	// ErrInvalidPassword is returned when the supplied current password is incorrect.
	ErrInvalidPassword = errors.New("current password is incorrect")
)

// Role is a slim view of a role as needed by the users layer.
type Role struct {
	ID       uuid.UUID
	Name     string
	IsSystem bool
}

// User is the full user record including roles.
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

// UserFilter controls pagination and search for List.
type UserFilter struct {
	Search  string
	Page    int
	PerPage int
	SortBy  string
	SortDir string
}

// CreateRequest carries validated fields for creating a new user.
type CreateRequest struct {
	Email    string    // validated by handler
	Name     string    // validated by handler
	Password string    // plain-text; hashed by service
	RoleID   uuid.UUID // role to assign
}

// UpdateRequest carries validated fields for updating an existing user.
type UpdateRequest struct {
	Name   string
	Email  string
	RoleID uuid.UUID // replaces the current role assignment
}
