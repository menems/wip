package roles

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBRepository implements Repository against PostgreSQL via pgx.
type DBRepository struct {
	pool *pgxpool.Pool
}

// NewDBRepository constructs a DBRepository backed by the given pool.
func NewDBRepository(pool *pgxpool.Pool) *DBRepository {
	return &DBRepository{pool: pool}
}

// List returns all roles with their permissions, ordered by name.
func (r *DBRepository) List(ctx context.Context) ([]*Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, COALESCE(description, ''), is_system, created_at, updated_at
		FROM roles
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("roles: repo: list: %w", err)
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, fmt.Errorf("roles: repo: scan: %w", err)
		}
		roles = append(roles, role)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("roles: repo: list rows: %w", err)
	}

	for _, role := range roles {
		if err = r.loadPermissions(ctx, role); err != nil {
			return nil, err
		}
	}
	return roles, nil
}

// FindByID returns the role with the given ID including its permissions.
func (r *DBRepository) FindByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, COALESCE(description, ''), is_system, created_at, updated_at
		FROM roles WHERE id = $1
	`, id)

	role, err := scanRole(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("roles: repo: find by id: %w", err)
	}

	if err = r.loadPermissions(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// Create inserts the role and its permissions in a single transaction.
func (r *DBRepository) Create(ctx context.Context, role *Role) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("roles: repo: create: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO roles (id, name, description, is_system)
		VALUES ($1, $2, $3, $4)
	`, role.ID, role.Name, role.Description, role.IsSystem)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrNameConflict
		}
		return fmt.Errorf("roles: repo: create: insert: %w", err)
	}

	if err = insertPermissions(ctx, tx, role.ID, role.Permissions); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Update replaces the role's name/description and fully replaces its permissions.
func (r *DBRepository) Update(ctx context.Context, role *Role) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("roles: repo: update: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `
		UPDATE roles SET name = $1, description = $2 WHERE id = $3
	`, role.Name, role.Description, role.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrNameConflict
		}
		return fmt.Errorf("roles: repo: update: exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Full replacement of permissions
	if _, err = tx.Exec(ctx,
		`DELETE FROM role_permissions WHERE role_id = $1`, role.ID,
	); err != nil {
		return fmt.Errorf("roles: repo: update: delete permissions: %w", err)
	}

	if err = insertPermissions(ctx, tx, role.ID, role.Permissions); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Delete removes the role record (and cascades to role_permissions via FK).
func (r *DBRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("roles: repo: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CountUsersWithRole returns the number of users assigned to the given role.
func (r *DBRepository) CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_roles WHERE role_id = $1`, roleID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("roles: repo: count users with role: %w", err)
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRole(row rowScanner) (*Role, error) {
	var role Role
	var createdAt, updatedAt time.Time
	err := row.Scan(
		&role.ID, &role.Name, &role.Description, &role.IsSystem,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	role.CreatedAt = createdAt
	role.UpdatedAt = updatedAt
	return &role, nil
}

func (r *DBRepository) loadPermissions(ctx context.Context, role *Role) error {
	rows, err := r.pool.Query(ctx, `
		SELECT resource, action
		FROM role_permissions
		WHERE role_id = $1
		ORDER BY resource, action
	`, role.ID)
	if err != nil {
		return fmt.Errorf("roles: repo: load permissions for %s: %w", role.ID, err)
	}
	defer rows.Close()

	role.Permissions = []Permission{}
	for rows.Next() {
		var p Permission
		if err = rows.Scan(&p.Resource, &p.Action); err != nil {
			return fmt.Errorf("roles: repo: scan permission: %w", err)
		}
		role.Permissions = append(role.Permissions, p)
	}
	return rows.Err()
}

// tx is the minimal interface shared by pgxpool.Pool and pgx.Tx for Exec.
type txExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func insertPermissions(ctx context.Context, tx txExecer, roleID uuid.UUID, perms []Permission) error {
	for _, p := range perms {
		_, err := tx.Exec(ctx, `
			INSERT INTO role_permissions (id, role_id, resource, action)
			VALUES ($1, $2, $3, $4)
		`, uuid.New(), roleID, p.Resource, p.Action)
		if err != nil {
			return fmt.Errorf("roles: repo: insert permission (%s:%s): %w", p.Resource, p.Action, err)
		}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
