-- name: CreateUser :exec
INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, $3, $4);

-- name: FindUserByEmail :one
SELECT id, name, email, password_hash FROM users WHERE email = $1;
