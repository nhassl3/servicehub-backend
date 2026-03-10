-- name: CreateSeller :one
INSERT INTO sellers (username, display_name, description)
VALUES ($1, $2, $3)
RETURNING id, username, display_name, description, avatar_url, rating, total_sales, created_at, updated_at;

-- name: GetSellerByProductId :one
SELECT username
FROM sellers
WHERE id=(SELECT p.seller_id FROM products p WHERE p.id=$1);

-- name: GetSeller :one
SELECT id, username, display_name, description, avatar_url, rating, total_sales, created_at, updated_at
FROM sellers
WHERE (sqlc.narg('username')::varchar IS NULL OR username = sqlc.narg('username')::varchar)
  AND (sqlc.narg('seller_id')::uuid IS NULL OR id = sqlc.narg('seller_id')::uuid)
  AND (sqlc.narg('username')::varchar IS NOT NULL OR sqlc.narg('seller_id')::uuid IS NOT NULL);

-- name: UpdateSeller :one
UPDATE sellers
SET display_name = $2,
    description  = $3,
    avatar_url   = $4,
    updated_at   = NOW()
WHERE username = $1
RETURNING id, username, display_name, description, avatar_url, rating, total_sales, created_at, updated_at;

-- name: IncreaseTotalSalesByProductId :exec
UPDATE sellers
SET total_sales = total_sales + $2
WHERE id = (SELECT p.seller_id FROM products p WHERE p.id = $1);

-- name: UpdateSellerRating :exec
UPDATE sellers
SET rating     = (
    SELECT COALESCE(AVG(p.rating), 0)
    FROM products p
    WHERE p.seller_id = $1
),
    updated_at = NOW()
WHERE id = $1;

-- name: SellerExistsByUsername :one
SELECT EXISTS(SELECT 1 FROM sellers WHERE username = $1);
