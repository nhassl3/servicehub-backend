-- name: UpsertCart :one
INSERT INTO carts (username)
VALUES ($1)
ON CONFLICT (username) DO UPDATE SET username = EXCLUDED.username
RETURNING id, username;
