-- name: CreateMessage :one
INSERT INTO messages (channel_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMessagesByChannel :many
SELECT * FROM messages
WHERE channel_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;