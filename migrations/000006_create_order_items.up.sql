CREATE TABLE IF NOT EXISTS order_items (
    id          BIGSERIAL    PRIMARY KEY,
    order_id    UUID         NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id  UUID         NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity    INTEGER      NOT NULL CHECK (quantity > 0),
    unit_price  NUMERIC(12,2) NOT NULL,
    total_price NUMERIC(12,2) GENERATED ALWAYS AS (quantity * unit_price) STORED
);

CREATE INDEX idx_order_items_order_id   ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
