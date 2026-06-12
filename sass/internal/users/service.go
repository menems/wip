package users

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Repository interfaces — consumer-side, defined here where they are used.
// Each covers one cohesive concern and has ≤3 methods.
// ---------------------------------------------------------------------------

// userReader is the read-query port the service depends on.
type userReader interface {
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, filter UserFilter) ([]*User, int, error)
}

// userWriter is the mutation port the service depends on.
type userWriter interface {
	Create(ctx context.Context, user *User, roleID uuid.UUID) error
	Update(ctx context.Context, user *User, roleID uuid.UUID) error
}

// userLifecycle is the lifecycle-management port the service depends on.
type userLifecycle interface {
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	SetActive(ctx context.Context, id uuid.UUID, active bool) (*User, error)
	CountActiveSystemRoleUsers(ctx context.Context) (int, error)
}

// ---------------------------------------------------------------------------
// UserService
// ---------------------------------------------------------------------------

// UserService implements the application use cases for user management.
// It depends only on the three Repository ports — no HTTP or DB types.
type UserService struct {
	reader    userReader
	writer    userWriter
	lifecycle userLifecycle
}

// NewUserService constructs a UserService backed by the three repository ports.
func NewUserService(r userReader, w userWriter, lc userLifecycle) *UserService {
	return &UserService{reader: r, writer: w, lifecycle: lc}
}

// List returns a page of users matching the filter and the total matching count.
func (s *UserService) List(ctx context.Context, filter UserFilter) ([]*User, int, error) {
	users, total, err := s.reader.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("users: service: list: %w", err)
	}
	return users, total, nil
}

// Get returns the user with the given ID.
// Returns ErrNotFound if the user does not exist.
func (s *UserService) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.reader.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("users: service: get: %w", err)
	}
	return user, nil
}

// Create hashes the password, inserts the user, and assigns the role.
// Returns ErrEmailConflict if the email is already in use.
func (s *UserService) Create(ctx context.Context, req CreateRequest) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("users: service: create: hash password: %w", err)
	}

	user := &User{
		ID:           uuid.New(),
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: string(hash),
		IsActive:     true,
	}

	if err = s.writer.Create(ctx, user, req.RoleID); err != nil {
		return nil, fmt.Errorf("users: service: create: %w", err)
	}

	created, err := s.reader.FindByID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("users: service: create: reload: %w", err)
	}
	return created, nil
}

// Update applies name/email changes and replaces the user's role assignment.
// Returns ErrNotFound if the user does not exist.
// Returns ErrEmailConflict if the new email is already in use.
func (s *UserService) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*User, error) {
	user, err := s.reader.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("users: service: update: %w", err)
	}

	user.Name = req.Name
	user.Email = req.Email

	if err = s.writer.Update(ctx, user, req.RoleID); err != nil {
		return nil, fmt.Errorf("users: service: update: %w", err)
	}

	updated, err := s.reader.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("users: service: update: reload: %w", err)
	}
	return updated, nil
}

// SetActive activates or deactivates the user.
// Returns ErrLastAdmin when deactivating the last active system-role user.
func (s *UserService) SetActive(ctx context.Context, id uuid.UUID, active bool) (*User, error) {
	if !active {
		user, err := s.reader.FindByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("users: service: set active: %w", err)
		}
		if hasSystemRole(user) {
			count, err := s.lifecycle.CountActiveSystemRoleUsers(ctx)
			if err != nil {
				return nil, fmt.Errorf("users: service: set active: count admins: %w", err)
			}
			if count <= 1 {
				return nil, ErrLastAdmin
			}
		}
	}

	updated, err := s.lifecycle.SetActive(ctx, id, active)
	if err != nil {
		return nil, fmt.Errorf("users: service: set active: %w", err)
	}
	return updated, nil
}

// ChangePassword verifies the old password and sets a new bcrypt hash.
// Returns ErrNotFound if the user does not exist.
// Returns ErrInvalidPassword if oldPwd does not match the stored hash.
func (s *UserService) ChangePassword(ctx context.Context, id uuid.UUID, oldPwd, newPwd string) error {
	user, err := s.reader.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("users: service: change password: %w", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPwd)); err != nil {
		return ErrInvalidPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("users: service: change password: hash: %w", err)
	}

	if err = s.lifecycle.UpdatePassword(ctx, id, string(hash)); err != nil {
		return fmt.Errorf("users: service: change password: %w", err)
	}
	return nil
}

// Delete removes the user (soft-delete: deactivates with last-admin guard).
// Returns ErrNotFound if the user does not exist.
// Returns ErrLastAdmin when trying to remove the last active system-role user.
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	user, err := s.reader.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("users: service: delete: %w", err)
	}

	if hasSystemRole(user) {
		count, err := s.lifecycle.CountActiveSystemRoleUsers(ctx)
		if err != nil {
			return fmt.Errorf("users: service: delete: count admins: %w", err)
		}
		if count <= 1 {
			return ErrLastAdmin
		}
	}

	if _, err = s.lifecycle.SetActive(ctx, id, false); err != nil {
		return fmt.Errorf("users: service: delete: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func hasSystemRole(u *User) bool {
	for _, r := range u.Roles {
		if r.IsSystem {
			return true
		}
	}
	return false
}
