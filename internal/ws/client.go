package ws

import (
	"context"
	"encoding/json" 
	"log/slog"
	"time"
	"strings"
	"regexp"

	"github.com/abhinavvv-chauhan/chat-app/internal/repository/query"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/abhinavvv-chauhan/chat-app/internal/worker"
)

var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]+)`)

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	userID    string
	username  string 
	channelID string
	queries   *query.Queries
}

type WSMessage struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Content  string `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			slog.Info("User disconnected", "user_id", c.userID)
			break
		}

		content := string(message)

		if strings.HasPrefix(content, "[POLL_UPDATE=") {
			msgObj := WSMessage{
				UserID:    c.userID,
				Username:  c.username,
				Content:   content,
				CreatedAt: time.Now(),
			}
			msgBytes, _ := json.Marshal(msgObj)
			go c.detectMentionsAndNotify(message)
			c.hub.broadcast <- BroadcastMessage{
				ChannelID: c.channelID,
				Data:      msgBytes,
			}
			
			continue 
		}

		var userUUID, channelUUID pgtype.UUID
		userUUID.Scan(c.userID)
		channelUUID.Scan(c.channelID)

		savedMsg, err := c.queries.CreateMessage(context.Background(), query.CreateMessageParams{
			ChannelID: channelUUID,
			UserID:    userUUID,
			Content:   content, 
		})
		if err != nil {
			slog.Error("Failed to save message to database", "error", err)
			continue 
		}

		msgObj := WSMessage{
			UserID:    c.userID,
			Username:  c.username,
			Content:   savedMsg.Content,
			CreatedAt: savedMsg.CreatedAt.Time,
		}
		msgBytes, _ := json.Marshal(msgObj)
		c.hub.broadcast <- BroadcastMessage{
			ChannelID: c.channelID,
			Data:      msgBytes,
		}
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.conn.Close()
	}()

	for message := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			return
		}
	}
}

func (c *Client) detectMentionsAndNotify(messageData []byte) {
	text := string(messageData)
	matches := mentionRegex.FindAllStringSubmatch(text, -1)

	if len(matches) == 0 {
		return
	}

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		mentionedUser := match[1]

		// Prevent the server from sending you a push notification when you tag yourself
		if mentionedUser == c.username {
			continue
		}

		slog.Info("Mention detected!", "mentioned", mentionedUser, "by", c.username)

		// Package the push notification job
		task := worker.Task{
			Type: "web_push",
			Payload: worker.PushPayload{
				// Note: We will dynamically fetch the subscription from the DB in the next step
				Message: map[string]string{
					"title": "Mentioned by " + c.username,
					"body":  text,
					"url":   "/?channel=" + c.channelID,
				},
			},
		}

		// Non-blocking channel send! 
		// If 10,000 mentions happen at once and the queue fills up, 
		// this safely drops the notification instead of crashing the server.
		select {
		case c.hub.WorkerPool.TaskQueue <- task:
			slog.Debug("Push notification queued to worker pool")
		default:
			slog.Warn("Worker pool queue is full, dropping push notification")
		}
	}
}