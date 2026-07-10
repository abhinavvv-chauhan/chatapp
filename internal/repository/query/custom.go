package query

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type ReactionJSON struct {
	Emoji  string      `json:"emoji"`
	UserID pgtype.UUID `json:"user_id"`
}

type MessageWithUser struct {
	ID             pgtype.UUID      `json:"id"`
	ChannelID      pgtype.UUID      `json:"channel_id"`
	UserID         pgtype.UUID      `json:"user_id"`
	Username       string           `json:"username"`
	Content        string           `json:"content"`
	ParentID       pgtype.UUID      `json:"parent_id"`
	IsEdited       bool             `json:"is_edited"`
	CreatedAt      pgtype.Timestamp `json:"created_at"`
	Reactions      []ReactionJSON   `json:"reactions"`
	ParentUsername *string          `json:"parent_username,omitempty"`
	ParentContent  *string          `json:"parent_content,omitempty"`
}

const getMessagesWithUser = `
SELECT 
    m.id, m.channel_id, m.user_id, u.username, m.content, m.parent_id, 
    m.updated_at > m.created_at as is_edited, m.created_at,
    pu.username as parent_username, pm.content as parent_content
FROM messages m
JOIN users u ON m.user_id = u.id
LEFT JOIN messages pm ON m.parent_id = pm.id
LEFT JOIN users pu ON pm.user_id = pu.id
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
		if err := rows.Scan(&i.ID, &i.ChannelID, &i.UserID, &i.Username, &i.Content, &i.ParentID, &i.IsEdited, &i.CreatedAt, &i.ParentUsername, &i.ParentContent); err != nil {
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

const listDiscoverWorkspaces = `
SELECT id, name, slug, owner_id, created_at, updated_at
FROM workspaces
WHERE owner_id != $1
  AND id NOT IN (
    SELECT workspace_id FROM workspace_members WHERE user_id = $1
  )
ORDER BY created_at ASC
`

func (q *Queries) ListDiscoverWorkspaces(ctx context.Context, userID pgtype.UUID) ([]Workspace, error) {
	rows, err := q.db.Query(ctx, listDiscoverWorkspaces, userID)
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

const addReaction = `
INSERT INTO message_reactions (message_id, user_id, emoji) 
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
`
func (q *Queries) AddReaction(ctx context.Context, messageID pgtype.UUID, userID pgtype.UUID, emoji string) error {
	_, err := q.db.Exec(ctx, addReaction, messageID, userID, emoji)
	return err
}

const removeReaction = `
DELETE FROM message_reactions 
WHERE message_id = $1 AND user_id = $2 AND emoji = $3
`
func (q *Queries) RemoveReaction(ctx context.Context, messageID pgtype.UUID, userID pgtype.UUID, emoji string) error {
	_, err := q.db.Exec(ctx, removeReaction, messageID, userID, emoji)
	return err
}

const getReactions = `
SELECT message_id, user_id, emoji, created_at 
FROM message_reactions
WHERE message_id = ANY($1)
`
func (q *Queries) GetReactions(ctx context.Context, messageIDs []pgtype.UUID) ([]MessageReaction, error) {
	rows, err := q.db.Query(ctx, getReactions, messageIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MessageReaction
	for rows.Next() {
		var i MessageReaction
		if err := rows.Scan(&i.MessageID, &i.UserID, &i.Emoji, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, nil
}

const updateReadState = `
INSERT INTO channel_read_states (channel_id, user_id, last_read_at)
VALUES ($1, $2, CURRENT_TIMESTAMP)
ON CONFLICT (channel_id, user_id) DO UPDATE 
SET last_read_at = CURRENT_TIMESTAMP
`
func (q *Queries) UpdateReadState(ctx context.Context, channelID pgtype.UUID, userID pgtype.UUID) error {
	_, err := q.db.Exec(ctx, updateReadState, channelID, userID)
	return err
}

const getUnreadCounts = `
SELECT c.id AS channel_id, 
       COUNT(m.id) AS unread_count
FROM channels c
LEFT JOIN channel_read_states crs ON c.id = crs.channel_id AND crs.user_id = $1
LEFT JOIN messages m ON m.channel_id = c.id AND m.created_at > COALESCE(crs.last_read_at, '1970-01-01'::timestamp)
WHERE c.workspace_id = $2
GROUP BY c.id
`
type UnreadCountRow struct {
	ChannelID   pgtype.UUID
	UnreadCount int64
}
func (q *Queries) GetUnreadCounts(ctx context.Context, userID pgtype.UUID, workspaceID pgtype.UUID) ([]UnreadCountRow, error) {
	rows, err := q.db.Query(ctx, getUnreadCounts, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []UnreadCountRow
	for rows.Next() {
		var i UnreadCountRow
		if err := rows.Scan(&i.ChannelID, &i.UnreadCount); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, nil
}

const getMissedMessages = `
WITH read_state AS (
    SELECT last_read_at FROM channel_read_states WHERE channel_id = $1 AND user_id = $2
),
missed AS (
    SELECT 
        m.id, m.channel_id, m.user_id, u.username, m.content, m.parent_id, 
        m.updated_at > m.created_at as is_edited, m.created_at,
        pu.username as parent_username, pm.content as parent_content
    FROM messages m
    JOIN users u ON m.user_id = u.id
    LEFT JOIN messages pm ON m.parent_id = pm.id
    LEFT JOIN users pu ON pm.user_id = pu.id
    WHERE m.channel_id = $1 
      AND (
          (EXISTS(SELECT 1 FROM read_state) AND m.created_at > (SELECT last_read_at FROM read_state))
          OR NOT EXISTS(SELECT 1 FROM read_state)
      )
    ORDER BY m.created_at DESC
    LIMIT 100
)
SELECT * FROM missed ORDER BY created_at ASC;
`

func (q *Queries) GetMissedMessages(ctx context.Context, channelID pgtype.UUID, userID pgtype.UUID) ([]MessageWithUser, error) {
	rows, err := q.db.Query(ctx, getMissedMessages, channelID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MessageWithUser
	for rows.Next() {
		var i MessageWithUser
		if err := rows.Scan(&i.ID, &i.ChannelID, &i.UserID, &i.Username, &i.Content, &i.ParentID, &i.IsEdited, &i.CreatedAt, &i.ParentUsername, &i.ParentContent); err != nil {
			return nil, err
		}
		items = append(items, i)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
