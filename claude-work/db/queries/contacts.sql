-- name: SaveContact :exec
INSERT INTO contacts (user_id, name, phone, email, address) VALUES ($1, $2, $3, $4, $5);

-- name: AllContacts :many
SELECT user_id, name, phone, email, address FROM contacts WHERE user_id = $1;

-- name: FindContactByName :one
SELECT user_id, name, phone, email, address FROM contacts WHERE user_id = $1 AND name = $2;

-- name: RemoveContact :exec
DELETE FROM contacts WHERE user_id = $1 AND name = $2;
