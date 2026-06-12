package user

import (
	"context"
	"fmt"
)

// List returns all users.
func (s *Service) List(ctx context.Context) ([]User, error) {
	users, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}
