BEGIN;

DROP TRIGGER IF EXISTS trg_user_history_no_update ON user_history;
DROP FUNCTION IF EXISTS user_history_forbid_update();
DROP TABLE IF EXISTS user_history;
DROP TYPE IF EXISTS user_history_action;

COMMIT;
