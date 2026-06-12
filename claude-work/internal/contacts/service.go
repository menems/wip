package contacts

import (
	"context"

	"github.com/google/uuid"
)

// Repo is the persistence interface required by the service.
type Repo interface {
	Save(ctx context.Context, c Contact) error
	All(ctx context.Context, userID uuid.UUID) ([]Contact, error)
	FindByName(ctx context.Context, userID uuid.UUID, name string) (Contact, error)
	Remove(ctx context.Context, userID uuid.UUID, name string) error
}

type svc struct{ repo Repo }

// NewService returns a Service backed by the provided Repo.
func NewService(r Repo) Service { return &svc{repo: r} }

func (s *svc) Add(ctx context.Context, userID uuid.UUID, c Contact) error {
	c.UserID = userID
	return s.repo.Save(ctx, c)
}

func (s *svc) List(ctx context.Context, userID uuid.UUID) ([]Contact, error) {
	return s.repo.All(ctx, userID)
}

func (s *svc) GetByName(ctx context.Context, userID uuid.UUID, name string) (Contact, error) {
	return s.repo.FindByName(ctx, userID, name)
}

func (s *svc) Delete(ctx context.Context, userID uuid.UUID, name string) error {
	return s.repo.Remove(ctx, userID, name)
}
