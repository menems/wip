package users

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Repository is the port the users service depends on.
// Implementations live in repository.go.
type Repository interface {
	// List returns a page of users matching the filter, plus the total count.
	List(ctx context.Context, filter UserFilter) ([]*User, int, error)

	// FindByID returns the user with the given ID.
	// Returns ErrNotFound if the user does not exist.
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)

	// FindByEmail returns the user with the given email address.
	// Returns ErrNotFound if no such user exists.
	FindByEmail(ctx context.Context, email string) (*User, error)

	// Create inserts a new user and assigns them the specified role.
	// Returns ErrEmailConflict if the email is already taken.
	Create(ctx context.Context, user *User, roleID uuid.UUID) error

	// Update applies name/email changes and replaces the user's role assignment.
	// Returns ErrEmailConflict if the new email is already taken by another user.
	// Returns ErrNotFound if the user does not exist.
	Update(ctx context.Context, user *User, roleID uuid.UUID) error

	// UpdatePassword persists a new bcrypt password hash for the user.
	// Returns ErrNotFound if the user does not exist.
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error

	// SetActive sets is_active on the user and returns the updated record.
	// Returns ErrNotFound if the user does not exist.
	SetActive(ctx context.Context, id uuid.UUID, active bool) (*User, error)

	// CountActiveSystemRoleUsers returns the number of active users who hold
	// at least one system role. Used to enforce the last-admin guard.
	CountActiveSystemRoleUsers(ctx context.Context) (int, error)
}

// MinPasswordLength is the minimum allowed plaintext password length.
const MinPasswordLength = 8

// Service implements the application use cases for user management.
// It depends only on the Repository port — no HTTP or DB types.
type Service struct {
	repo       Repository
	bcryptCost int // configurable so tests can use bcrypt.MinCost
}

// NewService constructs a user Service.
// bcryptCost should be 12 in production and bcrypt.MinCost in tests.
func NewService(repo Repository, bcryptCost int) *Service {
	return &Service{repo: repo, bcryptCost: bcryptCost}
}

// List returns a paginated, filtered list of users and the total count.
func (s *Service) List(ctx context.Context, filter UserFilter) ([]*User, int, error) {
	users, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("users: list: %w", err)
	}
	return users, total, nil
}

// Get returns the user with the given ID.
// Returns ErrNotFound if the user does not exist.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("users: get: %w", err)
	}
	return user, nil
}

// Create creates a new user with the given fields and assigns them a role.
// Returns ErrEmailConflict if the email is already taken.
func (s *Service) Create(ctx context.Context, req CreateParams) (*User, error) {
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("users: create: hash password: %w", err)
	}

	user := &User{
		ID:           uuid.New(),
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		Name:         strings.TrimSpace(req.Name),
		PasswordHash: string(hash),
		IsActive:     true,
	}

	if err = s.repo.Create(ctx, user, req.RoleID); err != nil {
		return nil, fmt.Errorf("users: create: %w", err)
	}

	// Return the fully hydrated user (with roles).
	return s.repo.FindByID(ctx, user.ID)
}

// Update applies name/email changes and replaces the user's role assignment.
// Returns ErrNotFound if the user does not exist.
// Returns ErrEmailConflict if the new email is already used by another user.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateParams) (*User, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("users: update: %w", err)
	}

	existing.Email = strings.ToLower(strings.TrimSpace(req.Email))
	existing.Name = strings.TrimSpace(req.Name)

	if err = s.repo.Update(ctx, existing, req.RoleID); err != nil {
		return nil, fmt.Errorf("users: update: %w", err)
	}

	return s.repo.FindByID(ctx, id)
}

// Deactivate sets is_active=false on the user.
// Returns ErrNotFound if the user does not exist.
// Returns ErrLastAdmin if the user is the last active admin in the system.
func (s *Service) Deactivate(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("users: deactivate: %w", err)
	}

	// Guard: if this is the last active admin, refuse.
	if user.IsActive && user.hasSystemRole() {
		count, err := s.repo.CountActiveSystemRoleUsers(ctx)
		if err != nil {
			return nil, fmt.Errorf("users: deactivate: count admins: %w", err)
		}
		if count <= 1 {
			return nil, ErrLastAdmin
		}
	}

	updated, err := s.repo.SetActive(ctx, id, false)
	if err != nil {
		return nil, fmt.Errorf("users: deactivate: %w", err)
	}
	return updated, nil
}

// Reactivate sets is_active=true on the user.
// Returns ErrNotFound if the user does not exist.
func (s *Service) Reactivate(ctx context.Context, id uuid.UUID) (*User, error) {
	updated, err := s.repo.SetActive(ctx, id, true)
	if err != nil {
		return nil, fmt.Errorf("users: reactivate: %w", err)
	}
	return updated, nil
}

// ResetPassword updates the user's password hash.
// Returns ErrNotFound if the user does not exist.
// Returns a validation error if the password is shorter than MinPasswordLength.
func (s *Service) ResetPassword(ctx context.Context, id uuid.UUID, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}

	// Confirm the user exists before hashing.
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return fmt.Errorf("users: reset password: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("users: reset password: hash: %w", err)
	}

	if err = s.repo.UpdatePassword(ctx, id, string(hash)); err != nil {
		return fmt.Errorf("users: reset password: %w", err)
	}
	return nil
}

// validatePassword returns an error if the password does not meet requirements.
func validatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("users: password must be at least %d characters: %w",
			MinPasswordLength, ErrValidation)
	}
	return nil
}

// ErrValidation is a sentinel used internally for input validation failures.
// Handlers detect this to return 400 VALIDATION_ERROR.
var ErrValidation = fmt.Errorf("validation error")
