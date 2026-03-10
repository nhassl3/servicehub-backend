CREATE TABLE IF NOT EXISTS balance_transactions (
    id         BIGSERIAL     PRIMARY KEY,
    username   VARCHAR(50)   NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    type       VARCHAR(20)   NOT NULL CHECK (type IN ('deposit', 'withdraw', 'commission', 'profit')),
    amount     NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    comment    TEXT          NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_balance_tx_username_created ON balance_transactions(username, created_at DESC);
