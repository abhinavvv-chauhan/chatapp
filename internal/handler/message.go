package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/abhinavvv-chauhan/chat-app/internal/repository/query"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type MessageHandler struct {
	queries *query.Queries
}

func NewMessageHandler(q *query.Queries) *MessageHandler {
	return &MessageHandler{queries: q}
}

func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "channelID")
	var channelUUID pgtype.UUID
	if err := channelUUID.Scan(channelID); err != nil {
		http.Error(w, "Invalid channel ID format", http.StatusBadRequest)
		return
	}

	limit := int32(50)
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}

	offset := int32(0)
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if parsed, err := strconv.Atoi(offsetParam); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	messages, err := h.queries.GetMessagesWithUser(r.Context(), channelUUID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	if messages == nil {
		messages = []query.MessageWithUser{}
	} else if len(messages) > 0 {
		var msgIDs []pgtype.UUID
		for _, m := range messages {
			msgIDs = append(msgIDs, m.ID)
		}
		
		reactions, _ := h.queries.GetReactions(r.Context(), msgIDs)
		
		// Map reactions to messages
		reactionMap := make(map[pgtype.UUID][]query.ReactionJSON)
		for _, r := range reactions {
			reactionMap[r.MessageID] = append(reactionMap[r.MessageID], query.ReactionJSON{
				Emoji:  r.Emoji,
				UserID: r.UserID,
			})
		}
		
		for i := range messages {
			if r, ok := reactionMap[messages[i].ID]; ok {
				messages[i].Reactions = r
			} else {
				messages[i].Reactions = []query.ReactionJSON{}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}