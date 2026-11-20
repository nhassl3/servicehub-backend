-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, full_name)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUser :one
SELECT *
FROM users
WHERE (sqlc.narg(username)::varchar IS NOT NULL AND username=sqlc.narg(username))
    OR (sqlc.narg(email)::varchar IS NOT NULL AND email=sqlc.narg(email))
    OR (sqlc.narg(uid)::uuid IS NOT NULL AND uid=sqlc.narg(uid)) LIMIT 1;

-- name: UserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE
                                      (sqlc.narg(email)::varchar IS NOT NULL AND email=sqlc.narg(email))
                                  OR (sqlc.narg(username)::varchar IS NOT NULL AND username=sqlc.narg(username)));

-- name: UpdateUser :one
UPDATE users
SET full_name  = $2,
    avatar_url = $3,
    updated_at = NOW()
WHERE username = $1
RETURNING *;

-- name: UpdatePassword :one
UPDATE users
SET password_hash = sqlc.arg(new_password),
    updated_at = NOW()
WHERE username = sqlc.arg(username)
RETURNING *;

-- name: SetUserRole :one
UPDATE users
SET role       = $2,
    updated_at = NOW()
WHERE username = $1
RETURNING *;

-- name: VerifyEmailAccount :one
UPDATE users
SET is_active=true,
    updated_at=NOW()
WHERE (sqlc.narg(email)::varchar IS NULL OR email = sqlc.narg(email)::varchar)
    AND (sqlc.narg(username)::varchar IS NULL OR username = sqlc.narg(username)::varchar)
RETURNING *;

-- name: ConfirmEmailReset :one
-- Email can be changed only once and only while the account is still unverified
-- (the 72-hour window is enforced in the service layer). The whole switch is
-- atomic: either the account gets the new email, is activated and the one-shot
-- flag is consumed, or nothing happens.
UPDATE users
SET email = sqlc.arg(new_email),
    is_active = TRUE,
    email_reset_at = NOW(),
    updated_at = NOW()
WHERE username = sqlc.arg(username)
RETURNING *;

-- name: OAuthLogin :one
INSERT INTO users (username, email, password_hash)
VALUES ($1, $2, $3)
    ON CONFLICT (username) DO UPDATE SET username = EXCLUDED.username RETURNING *;
