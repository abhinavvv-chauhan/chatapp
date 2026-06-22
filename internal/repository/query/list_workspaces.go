package query

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
)

const listWorkspacesByOwner = `-- name: ListWorkspacesByOwner :many
SELECT id, name, slug, owner_id, created_at, updated_at FROM workspaces
WHERE owner_id = $1
ORDER BY name ASC
`

func (q *Queries) ListWorkspacesByOwner(ctx context.Context, ownerID pgtype.UUID) ([]Workspace, error) {
	rows, err := q.db.Query(ctx, listWorkspacesByOwner, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Workspace
	for rows.Next() {
		var i Workspace
		if err := rows.Scan(&i.ID, &i.Name, &i.Slug, &i.OwnerID, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}