package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sassai/backend/internal/user"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgerrcode"
)

// UserRepository implements user.Repository using the sqlc-generated Queries.
type UserRepository struct {
	q *Queries
}

// NewUserRepository constructs a UserRepository backed by the connection pool.
func NewUserRepository(pool *Pool) *UserRepository {
	return &UserRepository{q: New(pool)}
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, string, error) {
	row, err := r.q.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", user.ErrNotFound
		}
		return nil, "", fmt.Errorf("find user by email: %w", err)
	}
	return toUser(row.ID, row.Email, row.Name, row.AvatarUrl, row.CreatedAt, row.UpdatedAt), row.PasswordHash, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	row, err := r.q.FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return toUser(row.ID, row.Email, row.Name, row.AvatarUrl, row.CreatedAt, row.UpdatedAt), nil
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash, name string) (*user.User, error) {
	row, err := r.q.CreateUser(ctx, CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, user.ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return toUser(row.ID, row.Email, row.Name, row.AvatarUrl, row.CreatedAt, row.UpdatedAt), nil
}

func (r *UserRepository) Update(ctx context.Context, id string, params user.UpdateParams) (*user.User, error) {
	row, err := r.q.UpdateUser(ctx, UpdateUserParams{
		ID:        id,
		Name:      params.Name,
		AvatarUrl: params.AvatarURL,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}
	return toUser(row.ID, row.Email, row.Name, row.AvatarUrl, row.CreatedAt, row.UpdatedAt), nil
}

func toUser(id, email, name, avatarURL string, createdAt, updatedAt pgtype.Timestamptz) *user.User {
	return &user.User{
		ID:        id,
		Email:     email,
		Name:      name,
		AvatarURL: avatarURL,
		CreatedAt: toTime(createdAt),
		UpdatedAt: toTime(updatedAt),
	}
}

func toTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}
