package query

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
)

const createChannel = `-- name: CreateChannel :one
INSERT INTO channels (workspace_id, name, is_private, channel_type)
VALUES ($1, $2, $3, $4)
RETURNING id, workspace_id, name, is_private, channel_type, created_at, updated_at
`

type CreateChannelParams struct {
	WorkspaceID pgtype.UUID `json:"workspace_id"`
	Name        string      `json:"name"`
	IsPrivate   bool        `json:"is_private"`
	ChannelType string      `json:"channel_type"`
}

func (q *Queries) CreateChannel(ctx context.Context, arg CreateChannelParams) (Channel, error) {
	row := q.db.QueryRow(ctx, createChannel, arg.WorkspaceID, arg.Name, arg.IsPrivate, arg.ChannelType)
	var i Channel
	err := row.Scan(&i.ID, &i.WorkspaceID, &i.Name, &i.IsPrivate, &i.ChannelType, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}