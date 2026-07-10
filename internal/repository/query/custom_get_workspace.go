package query

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const getWorkspaceByID = `SELECT id, name, slug, owner_id, created_at, updated_at FROM workspaces WHERE id = $1`

func (q *Queries) GetWorkspaceByID(ctx context.Context, id pgtype.UUID) (Workspace, error) {
	row := q.db.QueryRow(ctx, getWorkspaceByID, id)
	var i Workspace
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Slug,
		&i.OwnerID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
