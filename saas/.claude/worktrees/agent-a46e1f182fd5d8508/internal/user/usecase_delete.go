package user

import (
	"context"
	"fmt"
)

// Delete removes a user by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
