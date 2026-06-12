package roles

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Service implements the application use cases for role management.
// It depends only on the Repository port — no HTTP or DB types.
type Service struct {
	repo Repository
}

// NewService constructs a roles Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns all roles with their permissions.
func (s *Service) List(ctx context.Context) ([]*Role, error) {
	roles, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("roles: list: %w", err)
	}
	return roles, nil
}

// Get returns the role with the given ID.
// Returns ErrNotFound if the role does not exist.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Role, error) {
	role, err := s.repo.FindByID(ctx, id)
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

	if err := s.repo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("roles: create: %w", err)
	}

	return s.repo.FindByID(ctx, role.ID)
}

// Update fully replaces a role's name, description, and permission set.
// Returns ErrNotFound if the role does not exist.
// Returns ErrNameConflict if the new name is already used by another role.
// Returns a validation error for any invalid (resource, action) combination.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Role, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("roles: update: %w", err)
	}

	if err = validatePermissions(req.Permissions); err != nil {
		return nil, err
	}

	existing.Name = strings.TrimSpace(req.Name)
	existing.Description = strings.TrimSpace(req.Description)
	existing.Permissions = deduplicatePermissions(req.Permissions)

	if err = s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("roles: update: %w", err)
	}

	return s.repo.FindByID(ctx, id)
}

// Delete removes the role.
// Returns ErrNotFound if the role does not exist.
// Returns ErrSystemRole if the role is a system role.
// Returns ErrRoleInUse if the role is currently assigned to any user.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("roles: delete: %w", err)
	}

	if role.IsSystem {
		return ErrSystemRole
	}

	count, err := s.repo.CountUsersWithRole(ctx, id)
	if err != nil {
		return fmt.Errorf("roles: delete: count users: %w", err)
	}
	if count > 0 {
		return ErrRoleInUse
	}

	if err = s.repo.Delete(ctx, id); err != nil {
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
var ErrValidation = fmt.Errorf("validation error")
