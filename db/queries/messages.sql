-- name: CreateMessage :one
INSERT INTO messages (channel_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMessagesByChannel :many
SELECT * FROM messages
WHERE channel_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMessagesPaginated :many
SELECT id, channel_id, user_id, content, created_at
FROM messages
WHERE channel_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateMessage :one
UPDATE messages
SET content = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND user_id = $3
RETURNING id, channel_id, user_id, content, created_at, updated_at;

-- name: DeleteMessage :exec
DELETE FROM messages
WHERE id = $1 AND user_id = $2;