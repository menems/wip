package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBRepository implements Repository against a PostgreSQL database via pgx.
type DBRepository struct {
	pool *pgxpool.Pool
}

// NewDBRepository constructs a DBRepository backed by the given connection pool.
func NewDBRepository(pool *pgxpool.Pool) *DBRepository {
	return &DBRepository{pool: pool}
}

// FindUserByEmail loads a user and their assigned roles by email address.
// Returns ErrNotFound if no user with that email exists.
func (r *DBRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	user, err := r.scanUser(ctx, `
		SELECT id, email, name, password_hash, is_active, created_at
		FROM users
		WHERE email = $1
	`, email)
	if err != nil {
		return nil, fmt.Errorf("auth: repo: find by email: %w", err)
	}
	if err = r.loadRoles(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// FindUserByID loads a user and their assigned roles by primary key.
// Returns ErrNotFound if the user does not exist.
func (r *DBRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := r.scanUser(ctx, `
		SELECT id, email, name, password_hash, is_active, created_at
		FROM users
		WHERE id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("auth: repo: find by id: %w", err)
	}
	if err = r.loadRoles(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// SaveRefreshToken persists a new refresh token record.
func (r *DBRepository) SaveRefreshToken(ctx context.Context, token *RefreshToken) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("auth: repo: save refresh token: %w", err)
	}
	return nil
}

// FindRefreshToken retrieves a token record by its SHA-256 hash.
// Returns ErrNotFound if no matching record exists.
func (r *DBRepository) FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash)

	var t RefreshToken
	var revokedAt *time.Time

	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("auth: repo: find refresh token: %w", err)
	}

	t.RevokedAt = revokedAt
	return &t, nil
}

// RevokeRefreshToken sets revoked_at=now() on the given token record.
func (r *DBRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("auth: repo: revoke refresh token: %w", err)
	}
	return nil
}

// scanUser executes a query that returns a single user row and maps it to a User.
// Returns ErrNotFound if the query returns no rows.
func (r *DBRepository) scanUser(ctx context.Context, query string, args ...any) (*User, error) {
	row := r.pool.QueryRow(ctx, query, args...)

	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &u, nil
}

// loadRoles populates u.Roles by querying user_roles and roles.
func (r *DBRepository) loadRoles(ctx context.Context, u *User) error {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.name
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`, u.ID)
	if err != nil {
		return fmt.Errorf("auth: repo: load roles for user %s: %w", u.ID, err)
	}
	defer rows.Close()

	u.Roles = []Role{}
	for rows.Next() {
		var role Role
		if err = rows.Scan(&role.ID, &role.Name); err != nil {
			return fmt.Errorf("auth: repo: scan role: %w", err)
		}
		u.Roles = append(u.Roles, role)
	}

	return rows.Err()
}
