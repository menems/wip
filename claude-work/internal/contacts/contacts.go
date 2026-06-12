package contacts

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Sentinel errors returned by service and repository operations.
var (
	ErrNotFound = errors.New("contact not found")
	ErrConflict = errors.New("contact already exists")
)

// Address is a mailing address; zero value means no address provided.
type Address struct {
	Street  string
	City    string
	State   string
	Zip     string
	Country string
}

// Contact is the domain type for an address book entry.
type Contact struct {
	UserID  uuid.UUID
	Name    string
	Phone   string
	Email   string
	Address Address
}

// Service is the interface for contact business logic.
type Service interface {
	Add(ctx context.Context, userID uuid.UUID, c Contact) error
	List(ctx context.Context, userID uuid.UUID) ([]Contact, error)
	GetByName(ctx context.Context, userID uuid.UUID, name string) (Contact, error)
	Delete(ctx context.Context, userID uuid.UUID, name string) error
}
