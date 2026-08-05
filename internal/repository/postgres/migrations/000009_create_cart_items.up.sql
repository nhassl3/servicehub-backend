CREATE TABLE IF NOT EXISTS cart_items (
    id         BIGSERIAL    PRIMARY KEY,
    cart_id    BIGINT       NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_id UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity   INTEGER      NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit_price NUMERIC(12,2) NOT NULL,
    UNIQUE (cart_id, product_id)
);

CREATE INDEX idx_cart_items_cart_id ON cart_items(cart_id);
