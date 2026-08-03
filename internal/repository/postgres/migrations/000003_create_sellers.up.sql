CREATE TABLE IF NOT EXISTS sellers (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username     VARCHAR(50)  NOT NULL UNIQUE REFERENCES users(username) ON DELETE CASCADE,
    display_name VARCHAR(255) NOT NULL,
    description  TEXT         NOT NULL DEFAULT '',
    avatar_url   TEXT         NOT NULL DEFAULT '',
    rating       NUMERIC(3,2) NOT NULL DEFAULT 0.00,
    total_sales  INTEGER      NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sellers_username ON sellers(username);
