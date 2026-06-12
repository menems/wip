package users

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	usersdb "github.com/blaz/serve/internal/users/db"
)

type userRecord struct {
	user User
	hash string
}

type memRepo struct {
	mu       sync.RWMutex
	users    map[uuid.UUID]userRecord
	byEmail  map[string]uuid.UUID
	sessions map[string]uuid.UUID
}

// NewRepository returns a new in-memory Repo.
func NewRepository() Repo {
	return &memRepo{
		users:    make(map[uuid.UUID]userRecord),
		byEmail:  make(map[string]uuid.UUID),
		sessions: make(map[string]uuid.UUID),
	}
}

func (r *memRepo) CreateUser(_ context.Context, u User, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byEmail[u.Email]; exists {
		return ErrConflict
	}
	r.users[u.ID] = userRecord{user: u, hash: hash}
	r.byEmail[u.Email] = u.ID
	return nil
}

func (r *memRepo) FindByEmail(_ context.Context, email string) (User, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byEmail[email]
	if !ok {
		return User{}, "", ErrNotFound
	}
	rec := r.users[id]
	return rec.user, rec.hash, nil
}

func (r *memRepo) CreateSession(_ context.Context, userID uuid.UUID, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[token] = userID
	return nil
}

func (r *memRepo) FindUserByToken(_ context.Context, token string) (uuid.UUID, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.sessions[token]
	if !ok {
		return uuid.UUID{}, ErrNotFound
	}
	return id, nil
}

// pgRepo persists users in Postgres; sessions remain in-memory.
type pgRepo struct {
	q        *usersdb.Queries
	mu       sync.RWMutex
	sessions map[string]uuid.UUID
}

// NewPGRepository returns a Repo backed by a pgx connection pool.
func NewPGRepository(pool *pgxpool.Pool) Repo {
	return &pgRepo{
		q:        usersdb.New(pool),
		sessions: make(map[string]uuid.UUID),
	}
}

func (r *pgRepo) CreateUser(ctx context.Context, u User, hash string) error {
	err := r.q.CreateUser(ctx, usersdb.CreateUserParams{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: hash,
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

func (r *pgRepo) FindByEmail(ctx context.Context, email string) (User, string, error) {
	row, err := r.q.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, "", ErrNotFound
		}
		return User{}, "", err
	}
	return User{ID: row.ID, Name: row.Name, Email: row.Email}, row.PasswordHash, nil
}

func (r *pgRepo) CreateSession(_ context.Context, userID uuid.UUID, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[token] = userID
	return nil
}

func (r *pgRepo) FindUserByToken(_ context.Context, token string) (uuid.UUID, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.sessions[token]
	if !ok {
		return uuid.UUID{}, ErrNotFound
	}
	return id, nil
}
