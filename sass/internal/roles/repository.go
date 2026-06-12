package roles

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/menems/sass/db/sqlc"
)

// DBRepository implements Repository against PostgreSQL via sqlc.
type DBRepository struct {
	pool *pgxpool.Pool
	q    *sqlcdb.Queries
}

// NewDBRepository constructs a DBRepository backed by the given pool.
func NewDBRepository(pool *pgxpool.Pool) *DBRepository {
	return &DBRepository{pool: pool, q: sqlcdb.New(pool)}
}

// List returns all roles with their permissions, ordered by name.
func (r *DBRepository) List(ctx context.Context) ([]*Role, error) {
	rows, err := r.q.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("roles: repo: list: %w", err)
	}
	roles := make([]*Role, 0, len(rows))
	for _, row := range rows {
		role := &Role{
			ID:          uuid.UUID(row.ID.Bytes),
			Name:        row.Name,
			Description: row.Description,
			IsSystem:    row.IsSystem,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		}
		if err = r.loadPermissions(ctx, role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// FindByID returns the role with the given ID including its permissions.
func (r *DBRepository) FindByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	row, err := r.q.FindRoleByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("roles: repo: find by id: %w", err)
	}
	role := &Role{
		ID:          uuid.UUID(row.ID.Bytes),
		Name:        row.Name,
		Description: row.Description,
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
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

	qtx := r.q.WithTx(tx)
	err = qtx.CreateRole(ctx, sqlcdb.CreateRoleParams{
		ID:          pgtype.UUID{Bytes: role.ID, Valid: true},
		Name:        role.Name,
		Description: &role.Description,
		IsSystem:    role.IsSystem,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrNameConflict
		}
		return fmt.Errorf("roles: repo: create: insert: %w", err)
	}
	if err = insertPermissions(ctx, qtx, role.ID, role.Permissions); err != nil {
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

	qtx := r.q.WithTx(tx)
	affected, err := qtx.UpdateRole(ctx, sqlcdb.UpdateRoleParams{
		Name:        role.Name,
		Description: &role.Description,
		ID:          pgtype.UUID{Bytes: role.ID, Valid: true},
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrNameConflict
		}
		return fmt.Errorf("roles: repo: update: exec: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	if err = qtx.DeletePermissionsForRole(ctx, pgtype.UUID{Bytes: role.ID, Valid: true}); err != nil {
		return fmt.Errorf("roles: repo: update: delete permissions: %w", err)
	}
	if err = insertPermissions(ctx, qtx, role.ID, role.Permissions); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Delete removes the role record (and cascades to role_permissions via FK).
func (r *DBRepository) Delete(ctx context.Context, id uuid.UUID) error {
	affected, err := r.q.DeleteRole(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return fmt.Errorf("roles: repo: delete: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// CountUsersWithRole returns the number of users assigned to the given role.
func (r *DBRepository) CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int, error) {
	count, err := r.q.CountUsersWithRole(ctx, pgtype.UUID{Bytes: roleID, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("roles: repo: count users with role: %w", err)
	}
	return int(count), nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (r *DBRepository) loadPermissions(ctx context.Context, role *Role) error {
	rows, err := r.q.LoadPermissionsForRole(ctx, pgtype.UUID{Bytes: role.ID, Valid: true})
	if err != nil {
		return fmt.Errorf("roles: repo: load permissions for %s: %w", role.ID, err)
	}
	role.Permissions = make([]Permission, 0, len(rows))
	for _, row := range rows {
		role.Permissions = append(role.Permissions, Permission{
			Resource: row.Resource,
			Action:   row.Action,
		})
	}
	return nil
}

func insertPermissions(ctx context.Context, q *sqlcdb.Queries, roleID uuid.UUID, perms []Permission) error {
	for _, p := range perms {
		err := q.InsertPermission(ctx, sqlcdb.InsertPermissionParams{
			ID:       pgtype.UUID{Bytes: uuid.New(), Valid: true},
			RoleID:   pgtype.UUID{Bytes: roleID, Valid: true},
			Resource: p.Resource,
			Action:   p.Action,
		})
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
