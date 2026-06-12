package user

import "context"

// Service is the inbound port for user use cases.
// The HTTP adapter depends on this interface, not the concrete type.
type Service interface {
	GetByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, id string, params UpdateParams) (*User, error)
}

// userService is the application-layer implementation of Service.
type userService struct {
	repo Repository
}

// NewService constructs a UserService backed by the given Repository.
func NewService(repo Repository) Service {
	return &userService{repo: repo}
}

func (s *userService) GetByID(ctx context.Context, id string) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *userService) Update(ctx context.Context, id string, params UpdateParams) (*User, error) {
	return s.repo.Update(ctx, id, params)
}
