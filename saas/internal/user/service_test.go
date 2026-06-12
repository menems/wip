package user_test

import (
	"context"

	"github.com/menems/saas/internal/user"
)

// mockStore implements user.Store for testing.
type mockStore struct {
	createFn     func(ctx context.Context, u user.User) (user.User, error)
	getByIDFn    func(ctx context.Context, id string) (user.User, error)
	getByEmailFn func(ctx context.Context, email string) (user.User, error)
	listFn       func(ctx context.Context) ([]user.User, error)
	updateFn     func(ctx context.Context, u user.User) (user.User, error)
	deleteFn     func(ctx context.Context, id string) error
}

func (m *mockStore) Create(ctx context.Context, u user.User) (user.User, error) {
	return m.createFn(ctx, u)
}

func (m *mockStore) GetByID(ctx context.Context, id string) (user.User, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockStore) GetByEmail(ctx context.Context, email string) (user.User, error) {
	return m.getByEmailFn(ctx, email)
}

func (m *mockStore) List(ctx context.Context) ([]user.User, error) {
	return m.listFn(ctx)
}

func (m *mockStore) Update(ctx context.Context, u user.User) (user.User, error) {
	return m.updateFn(ctx, u)
}

func (m *mockStore) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

// mockHasher implements user.PasswordHasher for testing.
type mockHasher struct {
	hashFn    func(password string) (string, error)
	compareFn func(hash, password string) error
}

func (m *mockHasher) Hash(password string) (string, error) {
	return m.hashFn(password)
}

func (m *mockHasher) Compare(hash, password string) error {
	return m.compareFn(hash, password)
}
