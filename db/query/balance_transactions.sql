-- name: CreateBalanceTx :one
INSERT INTO balance_transactions (username, type, amount, comment)
VALUES ($1, $2, $3, $4)
RETURNING id, username, type, amount, comment, created_at;

-- name: ListBalanceTxByUsername :many
SELECT id, username, type, amount, comment, created_at
FROM balance_transactions
WHERE username = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountBalanceTxByUsername :one
SELECT COUNT(*) FROM balance_transactions WHERE username = $1;
