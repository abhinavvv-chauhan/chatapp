package query

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
)

const listChannelsByWorkspace = `-- name: ListChannelsByWorkspace :many
SELECT id, workspace_id, name, is_private, created_at, updated_at FROM channels
WHERE workspace_id = $1
ORDER BY name ASC
`

func (q *Queries) ListChannelsByWorkspace(ctx context.Context, workspaceID pgtype.UUID) ([]Channel, error) {
	rows, err := q.db.Query(ctx, listChannelsByWorkspace, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Channel
	for rows.Next() {
		var i Channel
		if err := rows.Scan(&i.ID, &i.WorkspaceID, &i.Name, &i.IsPrivate, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}