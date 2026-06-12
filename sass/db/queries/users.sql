-- name: FindUserByID :one
SELECT id, email, name, password_hash, is_active, created_at, updated_at
FROM users WHERE id = $1;

-- name: FindUserByEmail :one
SELECT id, email, name, password_hash, is_active, created_at, updated_at
FROM users WHERE email = $1;

-- name: CreateUser :exec
INSERT INTO users (id, email, name, password_hash, is_active)
VALUES ($1, $2, $3, $4, $5);

-- name: AssignRole :exec
INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2);

-- name: UpdateUser :execrows
UPDATE users SET email = $1, name = $2 WHERE id = $3;

-- name: DeleteUserRoles :exec
DELETE FROM user_roles WHERE user_id = $1;

-- name: UpdatePassword :execrows
UPDATE users SET password_hash = $1 WHERE id = $2;

-- name: SetActive :execrows
UPDATE users SET is_active = $1 WHERE id = $2;

-- name: CountActiveSystemRoleUsers :one
SELECT COUNT(DISTINCT u.id)::int
FROM users u
JOIN user_roles ur ON ur.user_id = u.id
JOIN roles ro ON ro.id = ur.role_id
WHERE u.is_active = true AND ro.is_system = true;

-- name: LoadRolesForUser :many
SELECT ro.id, ro.name, ro.is_system
FROM roles ro
JOIN user_roles ur ON ur.role_id = ro.id
WHERE ur.user_id = $1 ORDER BY ro.name;
