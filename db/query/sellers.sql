-- name: CreateSeller :one
INSERT INTO sellers (username, display_name, description)
VALUES ($1, $2, $3)
RETURNING id, username, display_name, description, avatar_url, rating, total_sales, created_at, updated_at;

-- name: GetSellerByUsername :one
SELECT id, username, display_name, description, avatar_url, rating, total_sales, created_at, updated_at
FROM sellers
WHERE username = $1;

-- name: UpdateSeller :one
UPDATE sellers
SET display_name = $2,
    description  = $3,
    avatar_url   = $4,
    updated_at   = NOW()
WHERE username = $1
RETURNING id, username, display_name, description, avatar_url, rating, total_sales, created_at, updated_at;

-- name: SellerExistsByUsername :one
SELECT EXISTS(SELECT 1 FROM sellers WHERE username = $1);
