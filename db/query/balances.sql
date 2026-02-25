-- name: UpsertBalance :one
INSERT INTO balances (username)
VALUES ($1)
ON CONFLICT (username) DO UPDATE SET username = EXCLUDED.username
RETURNING username, amount, updated_at;

-- name: GetBalance :one
SELECT username, amount, updated_at
FROM balances
WHERE username = $1;

-- name: AddToBalance :one
INSERT INTO balances (username, amount)
VALUES ($1, $2)
ON CONFLICT (username)
DO UPDATE SET amount     = balances.amount + EXCLUDED.amount,
              updated_at = NOW()
RETURNING username, amount, updated_at;

-- name: DeductFromBalance :one
UPDATE balances
SET amount     = amount - $2,
    updated_at = NOW()
WHERE username = $1
RETURNING username, amount, updated_at;

-- name: GetBalanceForUpdate :one
SELECT username, amount
FROM balances
WHERE username = $1
FOR UPDATE;
