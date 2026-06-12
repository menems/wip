package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/menems/sass/db/sqlc"
)

// DBRepository implements Repository against a PostgreSQL database via sqlc.
type DBRepository struct {
	pool *pgxpool.Pool
	q    *sqlcdb.Queries
}

// NewDBRepository constructs a DBRepository backed by the given connection pool.
func NewDBRepository(pool *pgxpool.Pool) *DBRepository {
	return &DBRepository{pool: pool, q: sqlcdb.New(pool)}
}

// FindUserByEmail loads a user and their assigned roles by email address.
// Returns ErrNotFound if no user with that email exists.
func (r *DBRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	row, err := r.q.AuthFindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("auth: repo: find by email: %w", err)
	}
	user := authRowToUser(row.ID, row.Email, row.Name, row.PasswordHash, row.IsActive, row.CreatedAt)
	if err = r.loadRoles(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// FindUserByID loads a user and their assigned roles by primary key.
// Returns ErrNotFound if the user does not exist.
func (r *DBRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row, err := r.q.AuthFindUserByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("auth: repo: find by id: %w", err)
	}
	user := authRowToUser(row.ID, row.Email, row.Name, row.PasswordHash, row.IsActive, row.CreatedAt)
	if err = r.loadRoles(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// SaveRefreshToken persists a new refresh token record.
func (r *DBRepository) SaveRefreshToken(ctx context.Context, token *RefreshToken) error {
	err := r.q.SaveRefreshToken(ctx, sqlcdb.SaveRefreshTokenParams{
		ID:        pgtype.UUID{Bytes: token.ID, Valid: true},
		UserID:    pgtype.UUID{Bytes: token.UserID, Valid: true},
		TokenHash: token.TokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: token.ExpiresAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("auth: repo: save refresh token: %w", err)
	}
	return nil
}

// FindRefreshToken retrieves a token record by its SHA-256 hash.
// Returns ErrNotFound if no matching record exists.
func (r *DBRepository) FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	row, err := r.q.FindRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("auth: repo: find refresh token: %w", err)
	}

	t := &RefreshToken{
		ID:        uuid.UUID(row.ID.Bytes),
		UserID:    uuid.UUID(row.UserID.Bytes),
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
	}
	if row.RevokedAt.Valid {
		revokedAt := row.RevokedAt.Time
		t.RevokedAt = &revokedAt
	}
	return t, nil
}

// RevokeRefreshToken sets revoked_at=now() on the given token record.
func (r *DBRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	if err := r.q.RevokeRefreshToken(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return fmt.Errorf("auth: repo: revoke refresh token: %w", err)
	}
	return nil
}

// loadRoles populates u.Roles via the sqlc-generated LoadRolesForAuthUser query.
func (r *DBRepository) loadRoles(ctx context.Context, u *User) error {
	rows, err := r.q.LoadRolesForAuthUser(ctx, pgtype.UUID{Bytes: u.ID, Valid: true})
	if err != nil {
		return fmt.Errorf("auth: repo: load roles for user %s: %w", u.ID, err)
	}
	u.Roles = make([]Role, 0, len(rows))
	for _, row := range rows {
		u.Roles = append(u.Roles, Role{
			ID:   uuid.UUID(row.ID.Bytes),
			Name: row.Name,
		})
	}
	return nil
}

// authRowToUser maps the common auth SELECT columns to a domain User.
func authRowToUser(id pgtype.UUID, email, name, passwordHash string, isActive bool, createdAt pgtype.Timestamptz) *User {
	return &User{
		ID:           uuid.UUID(id.Bytes),
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		IsActive:     isActive,
		CreatedAt:    createdAt.Time,
	}
}
