package user

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/menems/saas/internal/user/db"
)

// Repository wraps db.Queries and implements Store.
type Repository struct {
	q *db.Queries
}

// NewRepository creates a new Repository backed by the given queries.
func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

// Create persists a new user.
func (r *Repository) Create(ctx context.Context, u User) (User, error) {
	now := time.Now()
	row, err := r.q.CreateUser(ctx, db.CreateUserParams{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
		CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	return dbUserToDomain(row), nil
}

// GetByID retrieves a user by their unique ID.
func (r *Repository) GetByID(ctx context.Context, id string) (User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return dbUserToDomain(row), nil
}

// GetByEmail retrieves a user by their email address.
func (r *Repository) GetByEmail(ctx context.Context, email string) (User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return dbUserToDomain(row), nil
}

// List returns all users ordered by creation time descending.
func (r *Repository) List(ctx context.Context) ([]User, error) {
	rows, err := r.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = dbUserToDomain(row)
	}
	return users, nil
}

// Update modifies an existing user's mutable fields.
func (r *Repository) Update(ctx context.Context, u User) (User, error) {
	row, err := r.q.UpdateUser(ctx, db.UpdateUserParams{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      string(u.Role),
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return User{}, err
	}
	return dbUserToDomain(row), nil
}

// Delete removes a user by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteUser(ctx, id)
}

// dbUserToDomain converts a sqlc-generated db.User to the domain User type.
func dbUserToDomain(u db.User) User {
	return User{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		Role:         Role(u.Role),
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}
