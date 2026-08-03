CREATE TABLE IF NOT EXISTS admins (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username            VARCHAR(50)  NOT NULL UNIQUE REFERENCES users(username) ON DELETE CASCADE,
    display_name        VARCHAR(255) NOT NULL,
    level_rights        INTEGER      NOT NULL DEFAULT 1
                                                        CHECK (level_rights in (1, 2, 3)),
    total_moderation    INTEGER      NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admins_username ON admins(username);
