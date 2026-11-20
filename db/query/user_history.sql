-- name: CreateUserHistory :one
INSERT INTO user_history (
    username, actor_username, action, changes, ip_address, user_agent, request_id
) VALUES (
             $1, $2, $3, $4, $5, $6, $7
         )
    RETURNING *;

-- name: GetUserHistory :one
SELECT * FROM user_history WHERE id = $1 LIMIT 1;

-- Keyset-пагинация: первая страница — before_id = NULL,
-- следующая — before_id = id последней записи предыдущей страницы.
-- name: ListUserHistory :many
SELECT * FROM user_history
WHERE username = @username
  AND (sqlc.narg('action')::user_history_action IS NULL OR action = sqlc.narg('action'))
  AND (sqlc.narg('before_id')::bigint IS NULL OR id < sqlc.narg('before_id'))
ORDER BY id DESC
    LIMIT @page_size;

-- name: ListUserHistoryByActor :many
SELECT * FROM user_history
WHERE actor_username = @actor_username
  AND (sqlc.narg('before_id')::bigint IS NULL OR id < sqlc.narg('before_id'))
ORDER BY id DESC
    LIMIT @page_size;

-- name: CountUserHistory :one
SELECT count(*) FROM user_history WHERE username = $1 LIMIT 1;

-- Retention: удаляет записи старше указанной даты.
-- name: DeleteUserHistoryOlderThan :execrows
DELETE FROM user_history WHERE created_at < $1;
