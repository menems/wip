-- name: FindUserByEmail :one
-- Returns the full row including password_hash so the auth service can verify credentials.
SELECT id, email, password_hash, name, avatar_url, created_at, updated_at
FROM users
WHERE email = $1;

-- name: FindUserByID :one
SELECT id, email, name, avatar_url, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, name)
VALUES ($1, $2, $3)
RETURNING id, email, name, avatar_url, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET name       = $2,
    avatar_url = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING id, email, name, avatar_url, created_at, updated_at;
