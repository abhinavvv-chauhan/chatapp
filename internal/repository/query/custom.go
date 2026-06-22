package query

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type MessageWithUser struct {
	ID        pgtype.UUID      `json:"id"`
	ChannelID pgtype.UUID      `json:"channel_id"`
	UserID    pgtype.UUID      `json:"user_id"`
	Username  string           `json:"username"`
	Content   string           `json:"content"`
	CreatedAt pgtype.Timestamp `json:"created_at"`
}

const getMessagesWithUser = `
SELECT m.id, m.channel_id, m.user_id, u.username, m.content, m.created_at
FROM messages m
JOIN users u ON m.user_id = u.id
WHERE m.channel_id = $1
ORDER BY m.created_at DESC
LIMIT $2 OFFSET $3
`

func (q *Queries) GetMessagesWithUser(ctx context.Context, channelID pgtype.UUID, limit int32, offset int32) ([]MessageWithUser, error) {
	rows, err := q.db.Query(ctx, getMessagesWithUser, channelID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MessageWithUser
	for rows.Next() {
		var i MessageWithUser
		if err := rows.Scan(&i.ID, &i.ChannelID, &i.UserID, &i.Username, &i.Content, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, nil
}
func (q *Queries) GetUsername(ctx context.Context, id pgtype.UUID) (string, error) {
	var username string
	err := q.db.QueryRow(ctx, "SELECT username FROM users WHERE id = $1", id).Scan(&username)
	return username, err
}

const listAllWorkspaces = `
SELECT id, name, slug, owner_id, created_at, updated_at
FROM workspaces
ORDER BY created_at ASC
`

func (q *Queries) ListAllWorkspaces(ctx context.Context) ([]Workspace, error) {
	rows, err := q.db.Query(ctx, listAllWorkspaces)
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
	return items, nil
}

const joinWorkspace = `
INSERT INTO workspace_members (workspace_id, user_id) 
VALUES ($1, $2) 
ON CONFLICT DO NOTHING
`

func (q *Queries) JoinWorkspace(ctx context.Context, workspaceID pgtype.UUID, userID pgtype.UUID) error {
	_, err := q.db.Exec(ctx, joinWorkspace, workspaceID, userID)
	return err
}

const listJoinedWorkspaces = `
SELECT w.id, w.name, w.slug, w.owner_id, w.created_at, w.updated_at
FROM workspaces w
JOIN workspace_members wm ON w.id = wm.workspace_id
WHERE wm.user_id = $1
ORDER BY w.name ASC
`

func (q *Queries) ListJoinedWorkspaces(ctx context.Context, userID pgtype.UUID) ([]Workspace, error) {
	rows, err := q.db.Query(ctx, listJoinedWorkspaces, userID)
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
	return items, nil
}

type PollOption struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Votes    int    `json:"votes"`
	HasVoted bool   `json:"has_voted"`
}

type PollData struct {
	ID         string       `json:"id"`
	Question   string       `json:"question"`
	Options    []PollOption `json:"options"`
	TotalVotes int          `json:"total_votes"`
}

func (q *Queries) CreatePollWithOptions(ctx context.Context, question string, options []string) (string, error) {
	var pollID string
	err := q.db.QueryRow(ctx, "INSERT INTO polls (question) VALUES ($1) RETURNING id", question).Scan(&pollID)
	if err != nil {
		return "", err
	}

	for _, opt := range options {
		_, err = q.db.Exec(ctx, "INSERT INTO poll_options (poll_id, text) VALUES ($1, $2)", pollID, opt)
		if err != nil {
			return "", err
		}
	}
	return pollID, nil
}

func (q *Queries) VotePoll(ctx context.Context, pollID, optionID, userID string) error {
	_, err := q.db.Exec(ctx, `
		INSERT INTO poll_votes (poll_id, option_id, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (poll_id, user_id)
		DO UPDATE SET option_id = $2
	`, pollID, optionID, userID)
	return err
}

func (q *Queries) GetPollData(ctx context.Context, pollID string, userID string) (*PollData, error) {
	var question string
	err := q.db.QueryRow(ctx, "SELECT question FROM polls WHERE id = $1", pollID).Scan(&question)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.Query(ctx, `
		SELECT po.id, po.text,
			   COUNT(pv.user_id) as vote_count,
			   COALESCE(BOOL_OR(pv.user_id = $2), false) as has_voted
		FROM poll_options po
		LEFT JOIN poll_votes pv ON po.id = pv.option_id
		WHERE po.poll_id = $1
		GROUP BY po.id, po.text
		ORDER BY po.id ASC
	`, pollID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []PollOption
	totalVotes := 0
	for rows.Next() {
		var o PollOption
		rows.Scan(&o.ID, &o.Text, &o.Votes, &o.HasVoted)
		totalVotes += o.Votes
		options = append(options, o)
	}

	return &PollData{ID: pollID, Question: question, Options: options, TotalVotes: totalVotes}, nil
}
