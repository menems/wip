package middleware

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBPermissionLoader implements PermissionLoader against PostgreSQL.
// It computes the union of all permissions across all roles assigned to the user.
type DBPermissionLoader struct {
	pool *pgxpool.Pool
}

// NewDBPermissionLoader constructs a DBPermissionLoader backed by the given pool.
func NewDBPermissionLoader(pool *pgxpool.Pool) *DBPermissionLoader {
	return &DBPermissionLoader{pool: pool}
}

// LoadPermissions queries the union of role_permissions for all roles assigned
// to the given user. Duplicate (resource, action) pairs are deduplicated in SQL.
func (l *DBPermissionLoader) LoadPermissions(ctx context.Context, userID uuid.UUID) ([]Permission, error) {
	rows, err := l.pool.Query(ctx, `
		SELECT DISTINCT rp.resource, rp.action
		FROM role_permissions rp
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("middleware: load permissions for user %s: %w", userID, err)
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err = rows.Scan(&p.Resource, &p.Action); err != nil {
			return nil, fmt.Errorf("middleware: scan permission: %w", err)
		}
		perms = append(perms, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("middleware: iterate permissions: %w", err)
	}

	return perms, nil
}
