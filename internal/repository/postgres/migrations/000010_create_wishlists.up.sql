CREATE TABLE IF NOT EXISTS wishlists (
    id         BIGSERIAL   PRIMARY KEY,
    username   VARCHAR(50) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    product_id UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (username, product_id)
);

CREATE INDEX idx_wishlists_username ON wishlists(username);
