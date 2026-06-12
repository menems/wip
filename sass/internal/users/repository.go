package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/menems/sass/db/sqlc"
)

// allowedSortColumns is the allowlist for the sort_by query parameter.
// Any value not in this map falls back to created_at.
var allowedSortColumns = map[string]string{
	"created_at": "u.created_at",
	"name":       "u.name",
	"email":      "u.email",
}

// DBRepository implements Repository against PostgreSQL via sqlc.
type DBRepository struct {
	pool *pgxpool.Pool
	q    *sqlcdb.Queries
}

// NewDBRepository constructs a DBRepository backed by the given pool.
func NewDBRepository(pool *pgxpool.Pool) *DBRepository {
	return &DBRepository{pool: pool, q: sqlcdb.New(pool)}
}

// resolveFilter normalises pagination and sort parameters from a UserFilter.
func resolveFilter(filter UserFilter) (sortCol, sortDir string, page, perPage, offset int) {
	sortCol = allowedSortColumns[filter.SortBy]
	if sortCol == "" {
		sortCol = "u.created_at"
	}
	sortDir = "DESC"
	if strings.ToLower(filter.SortDir) == "asc" {
		sortDir = "ASC"
	}
	page = filter.Page
	if page < 1 {
		page = 1
	}
	perPage = filter.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	offset = (page - 1) * perPage
	return
}

// List returns a page of users matching the filter and the total matching count.
// Dynamic ORDER BY and ILIKE search are not covered by sqlc; this method stays hand-written.
func (r *DBRepository) List(ctx context.Context, filter UserFilter) ([]*User, int, error) {
	sortCol, sortDir, _, perPage, offset := resolveFilter(filter)

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

	var pgxUsers []*User
	for rows.Next() {
		u, err := scanListUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("users: repo: scan user: %w", err)
		}
		pgxUsers = append(pgxUsers, u)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("users: repo: list rows: %w", err)
	}

	for _, u := range pgxUsers {
		if err = r.loadRoles(ctx, u); err != nil {
			return nil, 0, err
		}
	}

	return pgxUsers, total, nil
}

// FindByID returns the user with the given ID including their roles.
func (r *DBRepository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row, err := r.q.FindUserByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("users: repo: find by id: %w", err)
	}
	u := sqlcUserToUser(row)
	if err = r.loadRoles(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// FindByEmail returns the user with the given email including their roles.
func (r *DBRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	row, err := r.q.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("users: repo: find by email: %w", err)
	}
	u := sqlcUserToUser(row)
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

	qtx := r.q.WithTx(tx)
	err = qtx.CreateUser(ctx, sqlcdb.CreateUserParams{
		ID:           pgtype.UUID{Bytes: user.ID, Valid: true},
		Email:        user.Email,
		Name:         user.Name,
		PasswordHash: user.PasswordHash,
		IsActive:     user.IsActive,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailConflict
		}
		return fmt.Errorf("users: repo: create: insert user: %w", err)
	}
	err = qtx.AssignRole(ctx, sqlcdb.AssignRoleParams{
		UserID: pgtype.UUID{Bytes: user.ID, Valid: true},
		RoleID: pgtype.UUID{Bytes: roleID, Valid: true},
	})
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

	qtx := r.q.WithTx(tx)
	affected, err := qtx.UpdateUser(ctx, sqlcdb.UpdateUserParams{
		Email: user.Email,
		Name:  user.Name,
		ID:    pgtype.UUID{Bytes: user.ID, Valid: true},
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailConflict
		}
		return fmt.Errorf("users: repo: update: exec: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	if err = qtx.DeleteUserRoles(ctx, pgtype.UUID{Bytes: user.ID, Valid: true}); err != nil {
		return fmt.Errorf("users: repo: update: delete roles: %w", err)
	}
	if err = qtx.AssignRole(ctx, sqlcdb.AssignRoleParams{
		UserID: pgtype.UUID{Bytes: user.ID, Valid: true},
		RoleID: pgtype.UUID{Bytes: roleID, Valid: true},
	}); err != nil {
		return fmt.Errorf("users: repo: update: insert role: %w", err)
	}
	return tx.Commit(ctx)
}

// UpdatePassword sets a new bcrypt hash on the given user.
func (r *DBRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	affected, err := r.q.UpdatePassword(ctx, sqlcdb.UpdatePasswordParams{
		PasswordHash: passwordHash,
		ID:           pgtype.UUID{Bytes: id, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("users: repo: update password: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// SetActive sets is_active on the user and returns the updated record.
func (r *DBRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) (*User, error) {
	affected, err := r.q.SetActive(ctx, sqlcdb.SetActiveParams{
		IsActive: active,
		ID:       pgtype.UUID{Bytes: id, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("users: repo: set active: %w", err)
	}
	if affected == 0 {
		return nil, ErrNotFound
	}
	return r.FindByID(ctx, id)
}

// CountActiveSystemRoleUsers returns the count of active users who hold any system role.
func (r *DBRepository) CountActiveSystemRoleUsers(ctx context.Context) (int, error) {
	count, err := r.q.CountActiveSystemRoleUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("users: repo: count active admins: %w", err)
	}
	return int(count), nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (r *DBRepository) loadRoles(ctx context.Context, u *User) error {
	rows, err := r.q.LoadRolesForUser(ctx, pgtype.UUID{Bytes: u.ID, Valid: true})
	if err != nil {
		return fmt.Errorf("users: repo: load roles: %w", err)
	}
	u.Roles = make([]Role, 0, len(rows))
	for _, row := range rows {
		u.Roles = append(u.Roles, Role{
			ID:       uuid.UUID(row.ID.Bytes),
			Name:     row.Name,
			IsSystem: row.IsSystem,
		})
	}
	return nil
}

// sqlcUserToUser maps a sqlcdb.User to a domain User (roles left empty).
func sqlcUserToUser(u sqlcdb.User) *User {
	return &User{
		ID:           uuid.UUID(u.ID.Bytes),
		Email:        u.Email,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

// scanListUser scans a single row from the hand-written List query.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanListUser(row rowScanner) (*User, error) {
	var (
		id           pgtype.UUID
		email, name  string
		passwordHash string
		isActive     bool
		createdAt    pgtype.Timestamptz
		updatedAt    pgtype.Timestamptz
	)
	if err := row.Scan(&id, &email, &name, &passwordHash, &isActive, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return &User{
		ID:           uuid.UUID(id.Bytes),
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		IsActive:     isActive,
		CreatedAt:    createdAt.Time,
		UpdatedAt:    updatedAt.Time,
	}, nil
}

// isUniqueViolation reports whether err is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
