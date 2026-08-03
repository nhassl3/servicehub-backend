CREATE TABLE IF NOT EXISTS carts (
    id         BIGSERIAL   PRIMARY KEY,
    username   VARCHAR(50) NOT NULL UNIQUE REFERENCES users(username) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_carts_username ON carts(username);
