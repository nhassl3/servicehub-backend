-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, full_name)
VALUES ($1, $2, $3, $4)
RETURNING username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at, email_reset_at;

-- name: GetUserByUsername :one
SELECT username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at, email_reset_at
FROM users
WHERE username = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at, email_reset_at
FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByUID :one
SELECT username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at, email_reset_at
FROM users
WHERE uid = $1 LIMIT 1;

-- name: UserExistsByUsername :one
SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 LIMIT 1);

-- name: UserExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 LIMIT 1);

-- name: UpdateUser :one
UPDATE users
SET full_name  = $2,
    avatar_url = $3,
    updated_at = NOW()
WHERE username = $1
RETURNING username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at, email_reset_at;

-- name: UpdatePassword :one
UPDATE users
SET password_hash = sqlc.arg(new_password),
    updated_at = NOW()
WHERE username = sqlc.arg(username)
RETURNING username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at, email_reset_at;


-- name: SetUserRole :one
UPDATE users
SET role       = $2,
    updated_at = NOW()
WHERE username = $1
RETURNING username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at, email_reset_at;

-- name: VerifyEmailAccount :one
UPDATE users
SET is_active=true,
    updated_at=NOW()
WHERE (sqlc.narg(email)::varchar IS NULL OR email = sqlc.narg(email)::varchar)
    AND (sqlc.narg(username)::varchar IS NULL OR username = sqlc.narg(username)::varchar)
RETURNING username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at, email_reset_at;

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
  AND is_active = FALSE
  AND email_reset_at IS NULL
RETURNING username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at, email_reset_at;
