-- name: AuthFindUserByEmail :one
SELECT id, email, name, password_hash, is_active, created_at
FROM users WHERE email = $1;

-- name: AuthFindUserByID :one
SELECT id, email, name, password_hash, is_active, created_at
FROM users WHERE id = $1;

-- name: SaveRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4);

-- name: FindRefreshToken :one
SELECT id, user_id, token_hash, expires_at, revoked_at
FROM refresh_tokens WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1;

-- name: LoadRolesForAuthUser :many
SELECT r.id, r.name
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.name;
