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

	messages, err := h.queries.GetMessagesWithUser(r.Context(), channelUUID, limit, 0)
	if err != nil {
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	if messages == nil {
		messages = []query.MessageWithUser{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}