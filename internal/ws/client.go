package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"time"

	"github.com/abhinavvv-chauhan/chat-app/internal/repository/query"
	"github.com/abhinavvv-chauhan/chat-app/internal/worker"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
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
	Type           string    `json:"type"`
	MessageID      string    `json:"message_id,omitempty"`
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	IsEdited       bool      `json:"is_edited,omitempty"`
	ParentID       string    `json:"parent_id,omitempty"`
	ParentUsername *string   `json:"parent_username,omitempty"`
	ParentContent  *string   `json:"parent_content,omitempty"`
}

type IncomingEvent struct {
	Type           string  `json:"type"`
	ChannelID      string  `json:"channel_id,omitempty"`
	MessageID      string  `json:"message_id,omitempty"` 
	Content        string  `json:"content,omitempty"`    
	TargetUser     string  `json:"target_user,omitempty"`
	ParentID       string  `json:"parent_id,omitempty"`
	Emoji          string  `json:"emoji,omitempty"`
	ParentUsername *string `json:"parent_username,omitempty"`
	ParentContent  *string `json:"parent_content,omitempty"`
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
		var event IncomingEvent
		
		// Attempt to parse incoming data as a JSON event (Typing, Edits, Deletes)
		if err := json.Unmarshal(message, &event); err == nil {
			
			// 1. Typing Indicators
			if event.Type == "typing" || event.Type == "stopped_typing" {
				outPayload := map[string]string{
					"type":       event.Type,
					"user_id":    c.userID,
					"username":   c.username,
					"channel_id": c.channelID,
				}
				outBytes, _ := json.Marshal(outPayload)
				c.hub.broadcast <- BroadcastMessage{
					ChannelID: c.channelID,
					Data:      outBytes,
				}
				continue
			}

			// 1.5 WebRTC Signaling (Mesh Network)
			if event.Type == "webrtc_offer" || event.Type == "webrtc_answer" || event.Type == "webrtc_ice_candidate" || event.Type == "voice_join" || event.Type == "voice_leave" {
				// Relay the signal to everyone in the channel, but the frontend will ignore it 
				// if the target_user_id doesn't match their own ID (except for voice_join/leave).
				outPayload := map[string]string{
					"type":        event.Type,
					"user_id":     c.userID,
					"username":    c.username,
					"target_user": event.TargetUser,
					"content":     event.Content, // Holds stringified SDP or ICE candidate
				}
				outBytes, _ := json.Marshal(outPayload)
				c.hub.broadcast <- BroadcastMessage{
					ChannelID: c.channelID,
					Data:      outBytes,
				}
				continue
			}

			// Parse User ID for Edit/Delete ownership verification
			var userUUID pgtype.UUID
			userUUID.Scan(c.userID)

			// 2. Deleting Messages
			if event.Type == "message_delete" && event.MessageID != "" {
				var msgUUID pgtype.UUID
				msgUUID.Scan(event.MessageID)

				err := c.queries.DeleteMessage(context.Background(), query.DeleteMessageParams{
					ID:     msgUUID,
					UserID: userUUID,
				})

				if err == nil {
					outBytes, _ := json.Marshal(map[string]string{
						"type":       "message_delete",
						"message_id": event.MessageID,
					})
					c.hub.broadcast <- BroadcastMessage{ChannelID: c.channelID, Data: outBytes}
				} else {
					slog.Error("Failed to delete message", "error", err)
				}
				continue
			}

			// 3. Editing Messages
			if event.Type == "message_edit" && event.MessageID != "" && event.Content != "" {
				var msgUUID pgtype.UUID
				msgUUID.Scan(event.MessageID)

				updatedMsg, err := c.queries.UpdateMessage(context.Background(), query.UpdateMessageParams{
					ID:      msgUUID,
					Content: event.Content,
					UserID:  userUUID,
				})

				if err == nil {
					editPayload := WSMessage{
						Type:      "message_edit",
						MessageID: event.MessageID, 
						Content:   updatedMsg.Content,
						IsEdited:  true,
					}
					outBytes, _ := json.Marshal(editPayload)
					c.hub.broadcast <- BroadcastMessage{ChannelID: c.channelID, Data: outBytes}
				} else {
					slog.Error("Failed to edit message (may be past 15 min limit)", "error", err)
				}
				continue
			}
			
			// 4. Reactions
			if event.Type == "reaction_add" && event.MessageID != "" && event.Emoji != "" {
				var msgUUID pgtype.UUID
				msgUUID.Scan(event.MessageID)
				err := c.queries.AddReaction(context.Background(), msgUUID, userUUID, event.Emoji)
				if err == nil {
					outBytes, _ := json.Marshal(map[string]string{
						"type":       "reaction_add",
						"message_id": event.MessageID,
						"user_id":    c.userID,
						"emoji":      event.Emoji,
					})
					c.hub.broadcast <- BroadcastMessage{ChannelID: c.channelID, Data: outBytes}
				}
				continue
			}

			if event.Type == "reaction_remove" && event.MessageID != "" && event.Emoji != "" {
				var msgUUID pgtype.UUID
				msgUUID.Scan(event.MessageID)
				err := c.queries.RemoveReaction(context.Background(), msgUUID, userUUID, event.Emoji)
				if err == nil {
					outBytes, _ := json.Marshal(map[string]string{
						"type":       "reaction_remove",
						"message_id": event.MessageID,
						"user_id":    c.userID,
						"emoji":      event.Emoji,
					})
					c.hub.broadcast <- BroadcastMessage{ChannelID: c.channelID, Data: outBytes}
				}
				continue
			}

			// 5. Read Receipts
			if event.Type == "mark_read" {
				var chanUUID pgtype.UUID
				chanUUID.Scan(c.channelID)
				c.queries.UpdateReadState(context.Background(), chanUUID, userUUID)
				continue
			}
		}

		// 6. Standard Chat Message Processing
		// If it's a JSON "message", use its fields. Otherwise, treat the raw string as content.
		messageContent := content
		if event.Type == "message" {
			messageContent = event.Content
		}

		var userUUID, channelUUID pgtype.UUID
		userUUID.Scan(c.userID)
		channelUUID.Scan(c.channelID)

		var parentUUID pgtype.UUID
		if event.ParentID != "" {
			parentUUID.Scan(event.ParentID)
		}

		savedMsg, err := c.queries.CreateMessage(context.Background(), query.CreateMessageParams{
			ChannelID: channelUUID,
			UserID:    userUUID,
			Content:   messageContent,
			ParentID:  parentUUID,
		})
		if err != nil {
			slog.Error("Failed to save message to database", "error", err)
			continue
		}

		// Safely extract the UUID string representation from the database object
		idVal, _ := savedMsg.ID.Value()
		idStr, _ := idVal.(string)

		msgObj := WSMessage{
			Type:           "message",
			MessageID:      idStr, 
			UserID:         c.userID,
			Username:       c.username,
			Content:        savedMsg.Content,
			CreatedAt:      savedMsg.CreatedAt.Time,
			ParentID:       event.ParentID,
			ParentUsername: event.ParentUsername,
			ParentContent:  event.ParentContent,
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

		if mentionedUser == c.username {
			continue
		}

		slog.Info("Mention detected!", "mentioned", mentionedUser, "by", c.username)

		task := worker.Task{
			Type: "web_push",
			Payload: worker.PushPayload{
				Message: map[string]string{
					"title": "Mentioned by " + c.username,
					"body":  text,
					"url":   "/?channel=" + c.channelID,
				},
			},
		}

		select {
		case c.hub.WorkerPool.TaskQueue <- task:
			slog.Debug("Push notification queued to worker pool")
		default:
			slog.Warn("Worker pool queue is full, dropping push notification")
		}
	}
}