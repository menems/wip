package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// allowedSortColumns is the allowlist for the sort_by query parameter.
// Any value not in this map falls back to created_at.
var allowedSortColumns = map[string]string{
	"created_at": "u.created_at",
	"name":       "u.name",
	"email":      "u.email",
}

// DBRepository implements Repository against PostgreSQL via pgx.
type DBRepository struct {
	pool *pgxpool.Pool
}

// NewDBRepository constructs a DBRepository backed by the given pool.
func NewDBRepository(pool *pgxpool.Pool) *DBRepository {
	return &DBRepository{pool: pool}
}

// List returns a page of users matching the filter and the total matching count.
func (r *DBRepository) List(ctx context.Context, filter UserFilter) ([]*User, int, error) {
	sortCol := allowedSortColumns[filter.SortBy]
	if sortCol == "" {
		sortCol = "u.created_at"
	}
	sortDir := "DESC"
	if strings.ToLower(filter.SortDir) == "asc" {
		sortDir = "ASC"
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	offset := (page - 1) * perPage

	// Count query
	countArgs := []any{}
	countWhere := "WHERE 1=1"
	if filter.Search != "" {
		countArgs = append(countArgs, "%"+filter.Search+"%")
		countWhere += fmt.Sprintf(" AND (u.name ILIKE $%d OR u.email ILIKE $%d)", len(countArgs), len(countArgs))
	}

	var total int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(DISTINCT u.id) FROM users u "+countWhere, countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("users: repo: count: %w", err)
	}

	// Data query — uses same WHERE, adds ORDER BY + pagination
	dataArgs := append([]any{}, countArgs...)
	dataArgs = append(dataArgs, perPage, offset)
	limitPlaceholder := fmt.Sprintf("$%d", len(dataArgs)-1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(dataArgs))

	dataQuery := fmt.Sprintf(`
		SELECT DISTINCT u.id, u.email, u.name, u.password_hash, u.is_active, u.created_at, u.updated_at
		FROM users u
		%s
		ORDER BY %s %s
		LIMIT %s OFFSET %s
	`, countWhere, sortCol, sortDir, limitPlaceholder, offsetPlaceholder)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("users: repo: list query: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("users: repo: scan user: %w", err)
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("users: repo: list rows: %w", err)
	}

	// Load roles for all users
	for _, u := range users {
		if err = r.loadRoles(ctx, u); err != nil {
			return nil, 0, err
		}
	}

	return users, total, nil
}

// FindByID returns the user with the given ID including their roles.
func (r *DBRepository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, name, password_hash, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`, id)

	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("users: repo: find by id: %w", err)
	}

	if err = r.loadRoles(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// FindByEmail returns the user with the given email including their roles.
func (r *DBRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, name, password_hash, is_active, created_at, updated_at
		FROM users WHERE email = $1
	`, email)

	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("users: repo: find by email: %w", err)
	}

	if err = r.loadRoles(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Create inserts the user and assigns them to roleID in a single transaction.
func (r *DBRepository) Create(ctx context.Context, user *User, roleID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("users: repo: create: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, email, name, password_hash, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, user.ID, user.Email, user.Name, user.PasswordHash, user.IsActive)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailConflict
		}
		return fmt.Errorf("users: repo: create: insert user: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
	`, user.ID, roleID)
	if err != nil {
		return fmt.Errorf("users: repo: create: assign role: %w", err)
	}

	return tx.Commit(ctx)
}

// Update applies name/email changes and replaces the user's role assignment.
func (r *DBRepository) Update(ctx context.Context, user *User, roleID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("users: repo: update: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `
		UPDATE users SET email = $1, name = $2 WHERE id = $3
	`, user.Email, user.Name, user.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailConflict
		}
		return fmt.Errorf("users: repo: update: exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Replace role assignment
	if _, err = tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, user.ID); err != nil {
		return fmt.Errorf("users: repo: update: delete roles: %w", err)
	}
	if _, err = tx.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, user.ID, roleID,
	); err != nil {
		return fmt.Errorf("users: repo: update: insert role: %w", err)
	}

	return tx.Commit(ctx)
}

// UpdatePassword sets a new bcrypt hash on the given user.
func (r *DBRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`, passwordHash, id,
	)
	if err != nil {
		return fmt.Errorf("users: repo: update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetActive sets is_active on the user and returns the updated record.
func (r *DBRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) (*User, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET is_active = $1 WHERE id = $2`, active, id,
	)
	if err != nil {
		return nil, fmt.Errorf("users: repo: set active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.FindByID(ctx, id)
}

// CountActiveSystemRoleUsers returns the count of active users who hold any system role.
func (r *DBRepository) CountActiveSystemRoleUsers(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles ro ON ro.id = ur.role_id
		WHERE u.is_active = true AND ro.is_system = true
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("users: repo: count active admins: %w", err)
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// DB value objects
// ---------------------------------------------------------------------------

// userRecord mirrors exactly the columns returned by every users SELECT.
// It is the DB adapter's internal representation and must not escape the
// repository layer.
type userRecord struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// scan populates the record from a rowScanner (pgx.Row or pgx.Rows).
func (r *userRecord) scan(row rowScanner) error {
	return row.Scan(
		&r.ID, &r.Email, &r.Name, &r.PasswordHash,
		&r.IsActive, &r.CreatedAt, &r.UpdatedAt,
	)
}

// toDomain converts the DB record to a domain User.
// Roles are left empty; callers must hydrate them separately via loadRoles.
func (r *userRecord) toDomain() *User {
	return &User{
		ID:           r.ID,
		Email:        r.Email,
		Name:         r.Name,
		PasswordHash: r.PasswordHash,
		IsActive:     r.IsActive,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// roleRecord mirrors the columns returned by the roles SELECT.
type roleRecord struct {
	ID       uuid.UUID
	Name     string
	IsSystem bool
}

// toDomain converts the DB record to a domain Role.
func (r *roleRecord) toDomain() Role {
	return Role{ID: r.ID, Name: r.Name, IsSystem: r.IsSystem}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// rowScanner is satisfied by pgx.Row and pgx.Rows, letting scanUser work for both.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*User, error) {
	var rec userRecord
	if err := rec.scan(row); err != nil {
		return nil, err
	}
	return rec.toDomain(), nil
}

func (r *DBRepository) loadRoles(ctx context.Context, u *User) error {
	rows, err := r.pool.Query(ctx, `
		SELECT ro.id, ro.name, ro.is_system
		FROM roles ro
		JOIN user_roles ur ON ur.role_id = ro.id
		WHERE ur.user_id = $1
		ORDER BY ro.name
	`, u.ID)
	if err != nil {
		return fmt.Errorf("users: repo: load roles: %w", err)
	}
	defer rows.Close()

	u.Roles = []Role{}
	for rows.Next() {
		var rec roleRecord
		if err = rows.Scan(&rec.ID, &rec.Name, &rec.IsSystem); err != nil {
			return fmt.Errorf("users: repo: scan role: %w", err)
		}
		u.Roles = append(u.Roles, rec.toDomain())
	}
	return rows.Err()
}

// isUniqueViolation reports whether err is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
