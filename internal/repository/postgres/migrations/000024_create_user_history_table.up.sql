BEGIN;

CREATE TYPE user_history_action AS ENUM (
    'unspecified',
    'created',
    'updated',
    'deleted',
    'restored',
    'login',
    'password_changed',
    'email_changed',
    'role_changed'
);

CREATE TABLE IF NOT EXISTS user_history (
    id          BIGSERIAL PRIMARY KEY,
    username    VARCHAR(50)              NOT NULL REFERENCES users (username) ON DELETE CASCADE,
    -- Дефолтное значение 'system' означает, что обновление пользователя произведено системой
    actor_username    VARCHAR(50)              REFERENCES users (username) ON DELETE SET NULL DEFAULT 'system',
    action      user_history_action NOT NULL,
    changes     JSONB               NOT NULL DEFAULT '{}'::jsonb,
    ip_address  INET,
    user_agent  TEXT,
    request_id  TEXT,
    created_at  TIMESTAMPTZ         NOT NULL DEFAULT now(),

    CONSTRAINT user_history_changes_is_object CHECK (jsonb_typeof(changes) = 'object')
);

CREATE INDEX idx_user_history_username_id   ON user_history (username, id DESC);
CREATE INDEX idx_user_history_user_action   ON user_history (username, action, id DESC);
CREATE INDEX idx_user_history_actor_id      ON user_history (actor_username) WHERE actor_username <> 'system';
CREATE INDEX idx_user_history_created_at    ON user_history (created_at);

COMMENT ON TABLE  user_history         IS 'Append-only журнал изменений пользователя';
COMMENT ON COLUMN user_history.changes IS 'Diff вида {"field": {"old": ..., "new": ...}}. Секреты (пароли, токены) не хранить';

-- История неизменяема: UPDATE запрещён. DELETE оставлен для retention-очистки.
CREATE FUNCTION user_history_forbid_update() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'user_history is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_user_history_no_update
    BEFORE UPDATE ON user_history
    FOR EACH ROW EXECUTE FUNCTION user_history_forbid_update();

COMMIT;
