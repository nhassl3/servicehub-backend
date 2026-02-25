-- name: GetWishlistItems :many
SELECT id, username, product_id, created_at
FROM wishlists
WHERE username = $1
ORDER BY created_at DESC;

-- name: AddWishlistItem :one
INSERT INTO wishlists (username, product_id)
VALUES ($1, $2)
ON CONFLICT (username, product_id) DO NOTHING
RETURNING id, username, product_id, created_at;

-- name: RemoveWishlistItem :execrows
DELETE FROM wishlists
WHERE username = $1 AND product_id = $2;

-- name: WishlistItemExists :one
SELECT EXISTS(SELECT 1 FROM wishlists WHERE username = $1 AND product_id = $2);
