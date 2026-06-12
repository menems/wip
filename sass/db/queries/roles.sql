-- name: ListRoles :many
SELECT id, name, COALESCE(description, '') AS description, is_system, created_at, updated_at
FROM roles ORDER BY name;

-- name: FindRoleByID :one
SELECT id, name, COALESCE(description, '') AS description, is_system, created_at, updated_at
FROM roles WHERE id = $1;

-- name: CreateRole :exec
INSERT INTO roles (id, name, description, is_system) VALUES ($1, $2, $3, $4);

-- name: UpdateRole :execrows
UPDATE roles SET name = $1, description = $2 WHERE id = $3;

-- name: DeleteRole :execrows
DELETE FROM roles WHERE id = $1;

-- name: LoadPermissionsForRole :many
SELECT resource, action FROM role_permissions
WHERE role_id = $1 ORDER BY resource, action;

-- name: DeletePermissionsForRole :exec
DELETE FROM role_permissions WHERE role_id = $1;

-- name: InsertPermission :exec
INSERT INTO role_permissions (id, role_id, resource, action)
VALUES ($1, $2, $3, $4);

-- name: CountUsersWithRole :one
SELECT COUNT(*)::int FROM user_roles WHERE role_id = $1;
