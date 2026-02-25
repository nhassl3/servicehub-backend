CREATE TABLE IF NOT EXISTS reviews (
    id         BIGSERIAL   PRIMARY KEY,
    product_id UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    username   VARCHAR(50) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    rating     SMALLINT    NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment    TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, username)
);

CREATE INDEX idx_reviews_product_id ON reviews(product_id);
CREATE INDEX idx_reviews_rating     ON reviews(rating);
