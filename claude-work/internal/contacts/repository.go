package contacts

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	contactsdb "github.com/blaz/serve/internal/contacts/db"
)

type contactKey struct {
	userID uuid.UUID
	name   string
}

type memRepo struct {
	mu   sync.RWMutex
	data map[contactKey]Contact
}

// NewRepository returns a new in-memory Repo.
func NewRepository() Repo {
	return &memRepo{data: make(map[contactKey]Contact)}
}

func (r *memRepo) Save(_ context.Context, c Contact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := contactKey{userID: c.UserID, name: c.Name}
	if _, exists := r.data[k]; exists {
		return ErrConflict
	}
	r.data[k] = c
	return nil
}

func (r *memRepo) All(_ context.Context, userID uuid.UUID) ([]Contact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Contact
	for k, c := range r.data {
		if k.userID == userID {
			out = append(out, c)
		}
	}
	if out == nil {
		out = []Contact{}
	}
	return out, nil
}

func (r *memRepo) FindByName(_ context.Context, userID uuid.UUID, name string) (Contact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.data[contactKey{userID: userID, name: name}]
	if !ok {
		return Contact{}, ErrNotFound
	}
	return c, nil
}

func (r *memRepo) Remove(_ context.Context, userID uuid.UUID, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := contactKey{userID: userID, name: name}
	if _, ok := r.data[k]; !ok {
		return ErrNotFound
	}
	delete(r.data, k)
	return nil
}

// pgRepo is a Postgres-backed Repo using sqlc-generated queries.
type pgRepo struct {
	q *contactsdb.Queries
}

// NewPGRepository returns a Repo backed by a pgx connection pool.
func NewPGRepository(pool *pgxpool.Pool) Repo {
	return &pgRepo{q: contactsdb.New(pool)}
}

// addressToDB encodes an Address as a single string for storage.
// Format: "street|city|state|zip|country".  Empty address → "".
func addressToDB(a Address) string {
	if a == (Address{}) {
		return ""
	}
	return a.Street + "|" + a.City + "|" + a.State + "|" + a.Zip + "|" + a.Country
}

// addressFromDB decodes the stored string back into an Address.
func addressFromDB(s string) Address {
	if s == "" {
		return Address{}
	}
	parts := strings.SplitN(s, "|", 5)
	if len(parts) != 5 {
		return Address{}
	}
	return Address{Street: parts[0], City: parts[1], State: parts[2], Zip: parts[3], Country: parts[4]}
}

func (r *pgRepo) Save(ctx context.Context, c Contact) error {
	err := r.q.SaveContact(ctx, contactsdb.SaveContactParams{
		UserID:  c.UserID,
		Name:    c.Name,
		Phone:   c.Phone,
		Email:   c.Email,
		Address: addressToDB(c.Address),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (r *pgRepo) All(ctx context.Context, userID uuid.UUID) ([]Contact, error) {
	rows, err := r.q.AllContacts(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Contact, len(rows))
	for i, row := range rows {
		out[i] = Contact{UserID: row.UserID, Name: row.Name, Phone: row.Phone, Email: row.Email, Address: addressFromDB(row.Address)}
	}
	return out, nil
}

func (r *pgRepo) FindByName(ctx context.Context, userID uuid.UUID, name string) (Contact, error) {
	row, err := r.q.FindContactByName(ctx, contactsdb.FindContactByNameParams{UserID: userID, Name: name})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Contact{}, ErrNotFound
		}
		return Contact{}, err
	}
	return Contact{UserID: row.UserID, Name: row.Name, Phone: row.Phone, Email: row.Email, Address: addressFromDB(row.Address)}, nil // FindContactByNameRow has same fields
}

func (r *pgRepo) Remove(ctx context.Context, userID uuid.UUID, name string) error {
	if _, err := r.FindByName(ctx, userID, name); err != nil {
		return err
	}
	return r.q.RemoveContact(ctx, contactsdb.RemoveContactParams{UserID: userID, Name: name})
}
