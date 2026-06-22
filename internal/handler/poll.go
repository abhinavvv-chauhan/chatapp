package handler

import (
	"encoding/json"
	"net/http"

	"github.com/abhinavvv-chauhan/chat-app/internal/repository/query"
	"github.com/abhinavvv-chauhan/chat-app/internal/server/middleware"
	"github.com/go-chi/chi/v5"
)

type PollHandler struct {
	queries *query.Queries
}

func NewPollHandler(q *query.Queries) *PollHandler {
	return &PollHandler{queries: q}
}

type CreatePollReq struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

func (h *PollHandler) CreatePoll(w http.ResponseWriter, r *http.Request) {
	var req CreatePollReq
	json.NewDecoder(r.Body).Decode(&req)
	pollID, _ := h.queries.CreatePollWithOptions(r.Context(), req.Question, req.Options)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": pollID})
}

func (h *PollHandler) GetPoll(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")
	userID := r.Context().Value(middleware.UserIDKey).(string)
	
	data, _ := h.queries.GetPollData(r.Context(), pollID, userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

type VoteReq struct {
	OptionID string `json:"option_id"`
}

func (h *PollHandler) Vote(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")
	userID := r.Context().Value(middleware.UserIDKey).(string)
	var req VoteReq
	json.NewDecoder(r.Body).Decode(&req)

	h.queries.VotePoll(r.Context(), pollID, req.OptionID, userID)
	w.WriteHeader(http.StatusOK)
}