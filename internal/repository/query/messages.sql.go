package query

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createMessage = `-- name: CreateMessage :one
INSERT INTO messages (channel_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING id, channel_id, user_id, content, created_at, updated_at
`

type CreateMessageParams struct {
	ChannelID pgtype.UUID `json:"channel_id"`
	UserID    pgtype.UUID `json:"user_id"`
	Content   string      `json:"content"`
}

func (q *Queries) CreateMessage(ctx context.Context, arg CreateMessageParams) (Message, error) {
	row := q.db.QueryRow(ctx, createMessage, arg.ChannelID, arg.UserID, arg.Content)
	var i Message
	err := row.Scan(
		&i.ID,
		&i.ChannelID,
		&i.UserID,
		&i.Content,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listMessagesByChannel = `-- name: ListMessagesByChannel :many
SELECT id, channel_id, user_id, content, created_at, updated_at FROM messages
WHERE channel_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`

type ListMessagesByChannelParams struct {
	ChannelID pgtype.UUID `json:"channel_id"`
	Limit     int32       `json:"limit"`
	Offset    int32       `json:"offset"`
}

func (q *Queries) ListMessagesByChannel(ctx context.Context, arg ListMessagesByChannelParams) ([]Message, error) {
	rows, err := q.db.Query(ctx, listMessagesByChannel, arg.ChannelID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Message
	for rows.Next() {
		var i Message
		if err := rows.Scan(
			&i.ID,
			&i.ChannelID,
			&i.UserID,
			&i.Content,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}