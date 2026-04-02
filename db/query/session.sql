-- name: CreateSession :one
INSERT INTO sessions (username, refresh_token, user_agent, client_ip, is_blocked, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (username) DO UPDATE
    SET refresh_token=excluded.refresh_token,
        user_agent=excluded.user_agent,
        client_ip=excluded.client_ip,
        expires_at=excluded.expires_at,
        created_at=NOW()
RETURNING *;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE username=$1;

-- name: GetSession :one
SELECT id, username, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at
FROM sessions
WHERE refresh_token=$1 LIMIT 1;

-- name: GetSessionByUsername :one
SELECT id, username, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at
FROM sessions
WHERE username=$1 LIMIT 1;