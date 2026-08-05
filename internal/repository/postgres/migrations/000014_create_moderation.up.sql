CREATE TABLE IF NOT EXISTS moderation (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id     UUID         NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    product_id   UUID         NOT NULL UNIQUE REFERENCES products(id) ON DELETE CASCADE,
    active       BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_uid ON moderation(admin_id);
CREATE INDEX idx_admin_product_id ON moderation(product_id);