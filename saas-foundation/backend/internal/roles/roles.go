// Package roles implements role and permission management.
// It follows hexagonal architecture: domain types and port interfaces live here.
package roles

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors that callers must handle.
var (
	// ErrNotFound is returned when a role does not exist.
	ErrNotFound = errors.New("not found")

	// ErrNameConflict is returned when the requested name is already in use.
	ErrNameConflict = errors.New("role name already in use")

	// ErrSystemRole is returned when attempting to delete a system role.
	ErrSystemRole = errors.New("cannot delete a system role")

	// ErrRoleInUse is returned when attempting to delete a role that is
	// currently assigned to one or more users.
	ErrRoleInUse = errors.New("role is assigned to one or more users")
)

// validPermissions defines every allowed (resource, action) combination.
// Any permission outside this set is rejected at the service layer.
var validPermissions = map[string][]string{
	"users":      {"read", "write", "delete"},
	"roles":      {"read", "write", "delete"},
	"audit_logs": {"read"},
}

// Permission is a single (resource, action) authorisation grant.
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// Validate reports whether the permission is in the allowed set.
func (p Permission) Validate() error {
	actions, ok := validPermissions[p.Resource]
	if !ok {
		return fmt.Errorf("unknown resource %q", p.Resource)
	}
	for _, a := range actions {
		if a == p.Action {
			return nil
		}
	}
	return fmt.Errorf("action %q is not valid for resource %q", p.Action, p.Resource)
}

// Role is the full role record including its permission set.
type Role struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsSystem    bool         `json:"is_system"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// CreateRequest carries validated fields for creating a new role.
type CreateRequest struct {
	Name        string
	Description string
	Permissions []Permission
}

// UpdateRequest carries validated fields for updating an existing role.
// Permissions are fully replaced on every update.
type UpdateRequest struct {
	Name        string
	Description string
	Permissions []Permission
}

// Repository is the port the roles service depends on.
type Repository interface {
	// List returns all roles with their permissions.
	List(ctx context.Context) ([]*Role, error)

	// FindByID returns the role with the given ID.
	// Returns ErrNotFound if the role does not exist.
	FindByID(ctx context.Context, id uuid.UUID) (*Role, error)

	// Create inserts a new role with its initial permission set.
	// Returns ErrNameConflict if the name is already taken.
	Create(ctx context.Context, role *Role) error

	// Update applies name/description changes and fully replaces permissions.
	// Returns ErrNotFound if the role does not exist.
	// Returns ErrNameConflict if the new name is taken by a different role.
	Update(ctx context.Context, role *Role) error

	// Delete removes the role. The caller must check guards before calling.
	// Returns ErrNotFound if the role does not exist.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountUsersWithRole returns the number of users assigned to the given role.
	CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int, error)
}
