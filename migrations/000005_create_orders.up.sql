CREATE TABLE IF NOT EXISTS orders (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    uid          UUID         NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    username     VARCHAR(50)  NOT NULL REFERENCES users(username) ON DELETE RESTRICT,
    status       VARCHAR(20)  NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending', 'paid', 'delivered', 'cancelled')),
    total_amount NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_username ON orders(username);
CREATE INDEX idx_orders_status   ON orders(status);
CREATE INDEX idx_orders_uid      ON orders(uid);
