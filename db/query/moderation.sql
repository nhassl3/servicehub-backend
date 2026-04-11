-- name: GetModeration :one
SELECT * FROM moderation WHERE id=$1 LIMIT 1;

-- name: Total :one
SELECT COUNT(*)
FROM moderation
WHERE (sqlc.narg('admin_id')::uuid IS NULL OR admin_id = sqlc.narg('admin_id')::uuid)
    AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
    AND active = $1;

-- name: ListModerationItems :many
SELECT
    p.id as product_id,
    p.seller_id,
    p.category_id,
    p.title,
    p.description,
    p.price,
    p.tags,
    p.status,
    p.sales_count,
    p.rating,
    p.reviews_count,
    p.created_at as product_created_at,
    p.updated_at as product_updated_at,
    m.id as moderation_id,
    m.admin_id as moderation_admin_id,
    a.username as admin_username,
    m.active as moderation_active,
    m.created_at as moderation_created_at,
    m.updated_at as moderation_updated_at
FROM products p
         LEFT JOIN moderation m ON p.id = m.product_id
         LEFT JOIN admins   a ON a.id = m.admin_id
WHERE
    (sqlc.narg('admin_id')::uuid IS NULL AND p.status = 'draft')
   OR
    (m.admin_id = sqlc.narg('admin_id')::uuid AND m.active = true)
ORDER BY m.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CreateModeration :one
INSERT INTO moderation (product_id, admin_id, active)
VALUES ($1, $2, true)
ON CONFLICT (product_id)
DO UPDATE SET active = true, admin_id = excluded.admin_id, updated_at = NOW()
RETURNING *;

-- name: Release :exec
WITH updated AS (
    UPDATE moderation m SET active = false, status = sqlc.arg('status'), updated_at = NOW()
    WHERE m.product_id = sqlc.arg('product_id')
    RETURNING m.product_id, m.status
)
DELETE FROM moderation a
WHERE a.product_id = (SELECT u.product_id FROM updated u WHERE u.status = 'draft');

-- clean up redis lock storage for next methods (Approve, Reject)

-- name: Approve :exec
UPDATE products SET status='active' WHERE id=$1;

-- name: Reject :exec
UPDATE products SET status='inactive' WHERE id=$1;

