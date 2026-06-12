package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Create validates the user and password, hashes the password, assigns an ID, and persists.
func (s *Service) Create(ctx context.Context, u User, password string) (User, error) {
	if err := u.Validate(); err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	if err := ValidatePassword(password); err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return User{}, fmt.Errorf("create user: hash password: %w", err)
	}

	u.ID = uuid.NewString()
	u.PasswordHash = hash

	created, err := s.store.Create(ctx, u)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}
