-- name: CreateOrder :one
INSERT INTO orders (username)
VALUES ($1)
RETURNING id, uid, username, status, total_amount, created_at, updated_at;

-- name: GetOrderByID :one
SELECT id, uid, username, status, total_amount, created_at, updated_at
FROM orders
WHERE id = $1;

-- name: GetOrderByUID :one
SELECT id, uid, username, status, total_amount, created_at, updated_at
FROM orders
WHERE uid = $1;

-- name: ListOrdersByUsername :many
SELECT id, uid, username, status, total_amount, created_at, updated_at
FROM orders
WHERE username = sqlc.arg(username)
  AND (sqlc.arg(filter_status)::text = '' OR status = sqlc.arg(filter_status))
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_) OFFSET sqlc.arg(offset_);

-- name: CountOrdersByUsername :one
SELECT COUNT(*)
FROM orders
WHERE username = sqlc.arg(username)
  AND (sqlc.arg(filter_status)::text = '' OR status = sqlc.arg(filter_status));

-- name: UpdateOrderStatus :one
UPDATE orders
SET status     = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, uid, username, status, total_amount, created_at, updated_at;

-- name: UpdateOrderStatusWithOldStatus :one
WITH old_order AS (
    SELECT oldo.id, oldo.status
    FROM orders as oldo
    WHERE oldo.id = $1
),
     updated AS (
UPDATE orders as o
SET status = $2,
    updated_at = NOW()
WHERE o.id = $1
    RETURNING o.id, o.uid, o.username, o.status, o.total_amount, o.created_at, o.updated_at
)
SELECT
    updated.*,
    old_order.status AS old_status
FROM updated
         JOIN old_order ON updated.id = old_order.id;

-- name: UpdateOrderTotal :one
UPDATE orders
SET total_amount = $2,
    updated_at   = NOW()
WHERE id = $1
RETURNING id, uid, username, status, total_amount, created_at, updated_at;
