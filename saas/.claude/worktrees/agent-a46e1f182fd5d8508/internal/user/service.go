package user

import "context"

// Store defines the persistence operations the service depends on.
type Store interface {
	Create(ctx context.Context, u User) (User, error)
	GetByID(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	List(ctx context.Context) ([]User, error)
	Update(ctx context.Context, u User) (User, error)
	Delete(ctx context.Context, id string) error
}

// PasswordHasher abstracts password hashing so it can be mocked in tests.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// Service implements the user domain operations.
type Service struct {
	store  Store
	hasher PasswordHasher
}

// NewService creates a new user service.
func NewService(store Store, hasher PasswordHasher) *Service {
	return &Service{store: store, hasher: hasher}
}
