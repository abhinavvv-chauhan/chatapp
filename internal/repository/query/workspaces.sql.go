package query

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
)

const createWorkspace = `-- name: CreateWorkspace :one
INSERT INTO workspaces (name, slug, owner_id)
VALUES ($1, $2, $3)
RETURNING id, name, slug, owner_id, created_at, updated_at
`

type CreateWorkspaceParams struct {
	Name    string      `json:"name"`
	Slug    string      `json:"slug"`
	OwnerID pgtype.UUID `json:"owner_id"`
}

func (q *Queries) CreateWorkspace(ctx context.Context, arg CreateWorkspaceParams) (Workspace, error) {
	row := q.db.QueryRow(ctx, createWorkspace, arg.Name, arg.Slug, arg.OwnerID)
	var i Workspace
	err := row.Scan(&i.ID, &i.Name, &i.Slug, &i.OwnerID, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}