package user

import (
	"context"
	"fmt"
)

// Get returns a user by ID.
func (s *Service) Get(ctx context.Context, id string) (User, error) {
	u, err := s.store.GetByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}
