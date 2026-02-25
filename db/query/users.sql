-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, full_name)
VALUES ($1, $2, $3, $4)
RETURNING username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at;

-- name: GetUserByUsername :one
SELECT username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at
FROM users
WHERE username = $1;

-- name: GetUserByEmail :one
SELECT username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByUID :one
SELECT username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at
FROM users
WHERE uid = $1;

-- name: UserExistsByUsername :one
SELECT EXISTS(SELECT 1 FROM users WHERE username = $1);

-- name: UserExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);

-- name: UpdateUser :one
UPDATE users
SET full_name  = $2,
    avatar_url = $3,
    updated_at = NOW()
WHERE username = $1
RETURNING username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at;

-- name: UpdatePassword :one
UPDATE users
SET password_hash = sqlc.arg(new_password),
    updated_at = NOW()
WHERE username = sqlc.arg(username)
RETURNING username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at;


-- name: SetUserRole :one
UPDATE users
SET role       = $2,
    updated_at = NOW()
WHERE username = $1
RETURNING username, uid, email, password_hash, full_name, avatar_url, role, is_active, created_at, updated_at;
