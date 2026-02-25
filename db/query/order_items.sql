-- name: CreateOrderItem :one
INSERT INTO order_items (order_id, product_id, quantity, unit_price)
VALUES ($1, $2, $3, $4)
RETURNING id, order_id, product_id, quantity, unit_price, total_price;

-- name: GetOrderItems :many
SELECT id, order_id, product_id, quantity, unit_price, total_price
FROM order_items
WHERE order_id = $1;
