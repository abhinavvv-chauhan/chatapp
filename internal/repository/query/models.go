package query

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Channel struct {
	ID          pgtype.UUID        `json:"id"`
	WorkspaceID pgtype.UUID        `json:"workspace_id"`
	Name        string             `json:"name"`
	IsPrivate   bool               `json:"is_private"`
	CreatedAt   pgtype.Timestamptz `json:"created_at"`
	UpdatedAt   pgtype.Timestamptz `json:"updated_at"`
}

type Message struct {
	ID        pgtype.UUID        `json:"id"`
	ChannelID pgtype.UUID        `json:"channel_id"`
	UserID    pgtype.UUID        `json:"user_id"`
	Content   string             `json:"content"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

type User struct {
	ID           pgtype.UUID        `json:"id"`
	Email        string             `json:"email"`
	PasswordHash string             `json:"password_hash"`
	Username     string             `json:"username"`
	AvatarUrl    pgtype.Text        `json:"avatar_url"`
	CreatedAt    pgtype.Timestamptz `json:"created_at"`
	UpdatedAt    pgtype.Timestamptz `json:"updated_at"`
}

type Workspace struct {
	ID        pgtype.UUID        `json:"id"`
	Name      string             `json:"name"`
	Slug      string             `json:"slug"`
	OwnerID   pgtype.UUID        `json:"owner_id"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}