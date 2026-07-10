package query

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type JoinRequest struct {
	ID          pgtype.UUID `json:"id"`
	WorkspaceID pgtype.UUID `json:"workspace_id"`
	UserID      pgtype.UUID `json:"user_id"`
	Username    string      `json:"username"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
}

const createJoinRequest = `INSERT INTO workspace_join_requests (workspace_id, user_id) VALUES ($1, $2) ON CONFLICT (workspace_id, user_id) DO UPDATE SET status = 'pending', updated_at = CURRENT_TIMESTAMP`

func (q *Queries) CreateJoinRequest(ctx context.Context, workspaceID pgtype.UUID, userID pgtype.UUID) error {
	_, err := q.db.Exec(ctx, createJoinRequest, workspaceID, userID)
	return err
}

const getPendingRequests = `SELECT r.id, r.workspace_id, r.user_id, u.username, r.status, r.created_at FROM workspace_join_requests r JOIN users u ON r.user_id = u.id WHERE r.workspace_id = $1 AND r.status = 'pending' ORDER BY r.created_at DESC`

func (q *Queries) GetPendingJoinRequests(ctx context.Context, workspaceID pgtype.UUID) ([]JoinRequest, error) {
	rows, err := q.db.Query(ctx, getPendingRequests, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []JoinRequest
	for rows.Next() {
		var i JoinRequest
		if err := rows.Scan(
			&i.ID,
			&i.WorkspaceID,
			&i.UserID,
			&i.Username,
			&i.Status,
			&i.CreatedAt,
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

const updateJoinRequestStatus = `UPDATE workspace_join_requests SET status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING user_id, workspace_id`

type UpdateJoinRequestRow struct {
	UserID      pgtype.UUID `json:"user_id"`
	WorkspaceID pgtype.UUID `json:"workspace_id"`
}

func (q *Queries) UpdateJoinRequestStatus(ctx context.Context, requestID pgtype.UUID, status string) (UpdateJoinRequestRow, error) {
	row := q.db.QueryRow(ctx, updateJoinRequestStatus, requestID, status)
	var i UpdateJoinRequestRow
	err := row.Scan(&i.UserID, &i.WorkspaceID)
	return i, err
}
