CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid(),
    username VARCHAR NOT NULL,
    refresh_token VARCHAR NOT NULL,
    user_agent VARCHAR NOT NULL,
    client_ip VARCHAR NOT NULL,
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (NOW())
);

ALTER TABLE sessions ADD FOREIGN KEY (username) REFERENCES users (username);