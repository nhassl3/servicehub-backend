-- name: CreateNotification :one
insert into notifications (username, message, group_of_message) VALUES ($1, $2, (select id from notification_group where slug=$3)) returning id;

-- name: GetNotification :one
select * from notifications where id=$1;

-- name: ListNotification :many
select * from notifications n
    where
        (sqlc.narg('to_user_id')::uuid is null or n.username=(select username from users u where u.id=sqlc.narg('to_user_id')))
        AND (sqlc.narg('to_user_username')::varchar is null or n.username=sqlc.narg('to_user_username'))
ORDER BY n.created_at DESC
LIMIT 10 OFFSET sqlc.arg('offset');