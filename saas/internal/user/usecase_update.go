package user

import (
	"context"
	"fmt"
)

// Update validates and persists changes to a user.
func (s *Service) Update(ctx context.Context, u User) (User, error) {
	if err := u.Validate(); err != nil {
		return User{}, fmt.Errorf("update user: %w", err)
	}

	updated, err := s.store.Update(ctx, u)
	if err != nil {
		return User{}, fmt.Errorf("update user: %w", err)
	}
	return updated, nil
}
