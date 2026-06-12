-- name: CreateUser :one
INSERT INTO users (id, email, name, password_hash, role, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, email, name, password_hash, role, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, name, password_hash, role, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, name, password_hash, role, created_at, updated_at
FROM users
WHERE email = $1;

-- name: ListUsers :many
SELECT id, email, name, password_hash, role, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: UpdateUser :one
UPDATE users
SET email = $2, name = $3, role = $4, updated_at = $5
WHERE id = $1
RETURNING id, email, name, password_hash, role, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
