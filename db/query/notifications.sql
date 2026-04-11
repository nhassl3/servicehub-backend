-- name: CreateNotification :one
insert into notifications (username, message, group_of_message) VALUES ($1, $2, $3) returning id;

-- name: GetNotification :one
select * from notifications 