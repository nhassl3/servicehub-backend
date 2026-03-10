-- name: GetAdminById :one
SELECT id, username, display_name, level_rights, total_moderation, created_at, updated_at FROM admins WHERE id=$1;

-- name: GetAdminByUsername :one
SELECT id, username, display_name, level_rights, total_moderation, created_at, updated_at FROM admins WHERE username=$1;

-- name: GetAdminUsernameByProductId :one
SELECT username FROM admins
                WHERE id=(SELECT admin_id FROM moderation WHERE product_id=$1);
