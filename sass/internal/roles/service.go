package roles

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Repository interfaces — consumer-side, defined here where they are used.
// Each covers one cohesive concern and has ≤3 methods.
// ---------------------------------------------------------------------------

// roleReader is the read-query port the service depends on.
type roleReader interface {
	List(ctx context.Context) ([]*Role, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Role, error)
}

// roleWriter is the mutation port the service depends on.
type roleWriter interface {
	Create(ctx context.Context, role *Role) error
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// roleCounter is the user-count port the service depends on.
type roleCounter interface {
	CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int, error)
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service implements the application use cases for role management.
// It depends only on the three repository ports — no HTTP or DB types.
type Service struct {
	reader  roleReader
	writer  roleWriter
	counter roleCounter
}

// NewService constructs a roles Service backed by the three repository ports.
func NewService(r roleReader, w roleWriter, c roleCounter) *Service {
	return &Service{reader: r, writer: w, counter: c}
}

// List returns all roles with their permissions.
func (s *Service) List(ctx context.Context) ([]*Role, error) {
	roles, err := s.reader.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("roles: list: %w", err)
	}
	return roles, nil
}

// Get returns the role with the given ID.
// Returns ErrNotFound if the role does not exist.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Role, error) {
	role, err := s.reader.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("roles: get: %w", err)
	}
	return role, nil
}

// Create creates a new role with the given permissions.
// Returns ErrNameConflict if the name is already taken.
// Returns a validation error for any invalid (resource, action) combination.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Role, error) {
	if err := validatePermissions(req.Permissions); err != nil {
		return nil, err
	}

	role := &Role{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		IsSystem:    false,
		Permissions: deduplicatePermissions(req.Permissions),
	}

	if err := s.writer.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("roles: create: %w", err)
	}

	return s.reader.FindByID(ctx, role.ID)
}

// Update fully replaces a role's name, description, and permission set.
// Returns ErrNotFound if the role does not exist.
// Returns ErrNameConflict if the new name is already used by another role.
// Returns a validation error for any invalid (resource, action) combination.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Role, error) {
	existing, err := s.reader.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("roles: update: %w", err)
	}

	if err = validatePermissions(req.Permissions); err != nil {
		return nil, err
	}

	existing.Name = strings.TrimSpace(req.Name)
	existing.Description = strings.TrimSpace(req.Description)
	existing.Permissions = deduplicatePermissions(req.Permissions)

	if err = s.writer.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("roles: update: %w", err)
	}

	return s.reader.FindByID(ctx, id)
}

// Delete removes the role.
// Returns ErrNotFound if the role does not exist.
// Returns ErrSystemRole if the role is a system role.
// Returns ErrRoleInUse if the role is currently assigned to any user.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	role, err := s.reader.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("roles: delete: %w", err)
	}

	if role.IsSystem {
		return ErrSystemRole
	}

	count, err := s.counter.CountUsersWithRole(ctx, id)
	if err != nil {
		return fmt.Errorf("roles: delete: count users: %w", err)
	}
	if count > 0 {
		return ErrRoleInUse
	}

	if err = s.writer.Delete(ctx, id); err != nil {
		return fmt.Errorf("roles: delete: %w", err)
	}
	return nil
}

// validatePermissions returns an error if any permission is outside the valid set.
func validatePermissions(perms []Permission) error {
	for _, p := range perms {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("roles: invalid permission: %w: %w", err, ErrValidation)
		}
	}
	return nil
}

// deduplicatePermissions removes duplicate (resource, action) pairs.
func deduplicatePermissions(perms []Permission) []Permission {
	seen := make(map[string]struct{}, len(perms))
	out := make([]Permission, 0, len(perms))
	for _, p := range perms {
		key := p.Resource + ":" + p.Action
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}

// ErrValidation is a sentinel used internally for input validation failures.
var ErrValidation = errors.New("validation error")
