CREATE TABLE IF NOT EXISTS products (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id     UUID         NOT NULL REFERENCES sellers(id) ON DELETE CASCADE,
    category_id   INTEGER      NOT NULL REFERENCES categories(id),
    title         VARCHAR(255) NOT NULL,
    description   TEXT         NOT NULL DEFAULT '',
    price         NUMERIC(12,2) NOT NULL CHECK (price >= 0),
    tags          TEXT[]       NOT NULL DEFAULT '{}',
    status        VARCHAR(20)  NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'inactive', 'draft')),
    sales_count   INTEGER      NOT NULL DEFAULT 0,
    rating        NUMERIC(3,2) NOT NULL DEFAULT 0.00,
    reviews_count INTEGER      NOT NULL DEFAULT 0,
    fts           TSVECTOR     GENERATED ALWAYS AS (
                      to_tsvector('english', title || ' ' || description)
                  ) STORED,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_seller_id   ON products(seller_id);
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_status      ON products(status);
CREATE INDEX idx_products_fts         ON products USING GIN(fts);
CREATE INDEX idx_products_tags        ON products USING GIN(tags);
