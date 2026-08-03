CREATE TABLE IF NOT EXISTS balances (
    username   VARCHAR(50)   NOT NULL UNIQUE REFERENCES users(username) ON DELETE CASCADE,
    amount     NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (amount >= 0),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_balances_username ON balances(username);
