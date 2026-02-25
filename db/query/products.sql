-- name: CreateProduct :one
INSERT INTO products (seller_id, category_id, title, description, price, tags)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, seller_id, category_id, title, description, price, tags, status, sales_count, rating, reviews_count, created_at, updated_at;

-- name: GetProductByID :one
SELECT id, seller_id, category_id, title, description, price, tags, status, sales_count, rating, reviews_count, created_at, updated_at
FROM products
WHERE id = $1;

-- name: ListProducts :many
SELECT id, seller_id, category_id, title, description, price, tags, status, sales_count, rating, reviews_count, created_at, updated_at
FROM products
WHERE (sqlc.narg('seller_id')::uuid IS NULL OR seller_id = sqlc.narg('seller_id')::uuid)
  AND (sqlc.narg('category_id')::int IS NULL OR category_id = sqlc.narg('category_id')::int)
  AND (sqlc.narg('min_price')::numeric IS NULL OR price >= sqlc.narg('min_price')::numeric)
  AND (sqlc.narg('max_price')::numeric IS NULL OR price <= sqlc.narg('max_price')::numeric)
  AND status = sqlc.arg('status')
ORDER BY created_at DESC
LIMIT sqlc.arg('limit_') OFFSET sqlc.arg('offset_');

-- name: CountListProducts :one
SELECT COUNT(*)
FROM products
WHERE (sqlc.narg('seller_id')::uuid IS NULL OR seller_id = sqlc.narg('seller_id')::uuid)
  AND (sqlc.narg('category_id')::int IS NULL OR category_id = sqlc.narg('category_id')::int)
  AND (sqlc.narg('min_price')::numeric IS NULL OR price >= sqlc.narg('min_price')::numeric)
  AND (sqlc.narg('max_price')::numeric IS NULL OR price <= sqlc.narg('max_price')::numeric)
  AND status = sqlc.arg('status');

-- name: SearchProducts :many
SELECT id, seller_id, category_id, title, description, price, tags, status, sales_count, rating, reviews_count, created_at, updated_at
FROM products
WHERE fts @@ plainto_tsquery('english', sqlc.arg('query'))
  AND status = 'active'
ORDER BY ts_rank(fts, plainto_tsquery('english', sqlc.arg('query'))) DESC
LIMIT sqlc.arg('limit_') OFFSET sqlc.arg('offset_');

-- name: CountSearchProducts :one
SELECT COUNT(*)
FROM products
WHERE fts @@ plainto_tsquery('english', $1)
  AND status = 'active';

-- name: UpdateProduct :one
UPDATE products
SET title       = $2,
    description = $3,
    price       = $4,
    tags        = $5,
    status      = $6,
    updated_at  = NOW()
WHERE id = $1
RETURNING id, seller_id, category_id, title, description, price, tags, status, sales_count, rating, reviews_count, created_at, updated_at;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = $1;

-- name: IncrementProductSales :exec
UPDATE products
SET sales_count = sales_count + $2
WHERE id = $1;

-- name: UpdateProductRating :exec
UPDATE products
SET rating = $2
WHERE id = $1;

-- name: GetProductSellerID :one
SELECT seller_id FROM products WHERE id = $1;
