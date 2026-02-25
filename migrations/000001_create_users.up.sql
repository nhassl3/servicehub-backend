CREATE TABLE IF NOT EXISTS users (
    username      VARCHAR(50)  PRIMARY KEY,
    uid           UUID         NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name     VARCHAR(255) NOT NULL DEFAULT '',
    avatar_url    TEXT         NOT NULL DEFAULT '',
    role          VARCHAR(20)  NOT NULL DEFAULT 'buyer'
                  CHECK (role IN ('buyer', 'seller', 'admin')),
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_uid   ON users(uid);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role  ON users(role);
