-- name: GetAdmin :one
SELECT *
FROM admins
WHERE (sqlc.arg(username)::varchar IS NULL OR username = sqlc.arg(username)::varchar)
  AND (sqlc.arg(admin_id)::uuid IS NULL OR id = sqlc.arg(admin_id)::uuid)
  AND (sqlc.arg(username)::varchar IS NOT NULL OR sqlc.arg(admin_id)::uuid IS NOT NULL) LIMIT 1;

-- name: GetAdminUsernameByProductId :one
SELECT username FROM admins
                WHERE id=(SELECT admin_id FROM moderation WHERE product_id=$1);

-- name: GetAdminForUpdate :one
SELECT *
FROM admins
WHERE (sqlc.arg(username)::varchar IS NULL OR username = sqlc.arg(username)::varchar)
  AND (sqlc.arg(admin_id)::uuid IS NULL OR id = sqlc.arg(admin_id)::uuid)
  AND (sqlc.arg(username)::varchar IS NOT NULL OR sqlc.arg(admin_id)::uuid IS NOT NULL) FOR UPDATE;

-- name: UpdateAdmin :one
UPDATE admins
    SET display_name = $1,
        level_rights = $2,
        total_moderation = $3,
        avatar_url = $4,
        updated_at = NOW()
WHERE username = $1
RETURNING *;

-- name: AdminExistsByUsername :one
SELECT EXISTS(SELECT 1 FROM admins WHERE username = $1);

-- name: IncreaseTotalModerates :exec
UPDATE admins SET total_moderation = total_moderation+sqlc.arg(total_moderates) WHERE username = $1;

-- name: CreateAdmin :one
INSERT INTO admins (username, display_name, level_rights)
VALUES ($1, $2, $3)
RETURNING *;
