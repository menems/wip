package auth_test

import (
	"context"

	"github.com/menems/saas/internal/user"
)

// mockUserFinder implements auth.UserFinder for testing.
type mockUserFinder struct {
	getByEmailFn func(ctx context.Context, email string) (user.User, error)
}

func (m *mockUserFinder) GetByEmail(ctx context.Context, email string) (user.User, error) {
	return m.getByEmailFn(ctx, email)
}

// mockPasswordComparer implements auth.PasswordComparer for testing.
type mockPasswordComparer struct {
	compareFn func(hash, password string) error
}

func (m *mockPasswordComparer) Compare(hash, password string) error {
	return m.compareFn(hash, password)
}
