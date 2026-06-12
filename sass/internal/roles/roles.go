// Package roles implements role and permission management.
// It follows hexagonal architecture: domain types and port interfaces live here.
package roles

import (
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
	Resource string
	Action   string
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
	ID          uuid.UUID
	Name        string
	Description string
	IsSystem    bool
	Permissions []Permission
	CreatedAt   time.Time
	UpdatedAt   time.Time
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

