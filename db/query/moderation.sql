-- name: GetModerationByProductId :one
SELECT id, admin_id, product_id, active, created_at, updated_at FROM moderation WHERE product_id=$1;

-- name: GetModerationByAdminId :one
SELECT id, admin_id, product_id, active, created_at, updated_at FROM moderation WHERE admin_id=$1;

-- name: ListActiveProducts :many
SELECT id, admin_id, product_id, active, created_at, updated_at FROM moderation
WHERE active=true
ORDER BY product_id
LIMIT $1 OFFSET $2;
