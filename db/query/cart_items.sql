-- name: GetCartItems :many
SELECT id, cart_id, product_id, quantity, unit_price
FROM cart_items
WHERE cart_id = $1;

-- name: UpsertCartItem :one
INSERT INTO cart_items (cart_id, product_id, quantity, unit_price)
VALUES ($1, $2, $3, $4)
ON CONFLICT (cart_id, product_id)
DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity
RETURNING id, cart_id, product_id, quantity, unit_price;

-- name: UpdateCartItemQty :one
UPDATE cart_items
SET quantity = $3
WHERE cart_id = $1 AND product_id = $2
RETURNING id, cart_id, product_id, quantity, unit_price;

-- name: DeleteCartItem :execrows
DELETE FROM cart_items
WHERE cart_id = $1 AND product_id = $2;

-- name: ClearCart :exec
DELETE FROM cart_items WHERE cart_id = $1;

-- name: GetProductPrice :one
SELECT price FROM products WHERE id = $1;
