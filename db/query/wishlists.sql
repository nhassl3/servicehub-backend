-- name: GetWishlistItems :many
SELECT id, username, product_id, created_at
FROM wishlists
WHERE username = $1
ORDER BY created_at DESC;

-- name: ToggleWishlistItem :one
WITH deleted AS (
DELETE FROM wishlists dw
WHERE dw.username = $1 AND dw.product_id = $2
    RETURNING id, username, product_id, created_at
), inserted AS (
INSERT INTO wishlists as iw (iw.username, iw.product_id)
SELECT $1, $2
WHERE NOT EXISTS (SELECT 1 FROM deleted)
    RETURNING id, username, product_id, created_at
    )
SELECT id, username, product_id, created_at, TRUE AS added
FROM inserted
UNION ALL
SELECT id, username, product_id, created_at, FALSE AS added
FROM deleted;

-- name: RemoveWishlistItem :execrows
DELETE FROM wishlists
WHERE username = $1 AND product_id = $2;

-- name: WishlistItemExists :one
SELECT EXISTS(SELECT 1 FROM wishlists WHERE username = $1 AND product_id = $2);
