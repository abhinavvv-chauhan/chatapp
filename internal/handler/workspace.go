package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/abhinavvv-chauhan/chat-app/internal/repository/query"
	"github.com/abhinavvv-chauhan/chat-app/internal/server/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type WorkspaceHandler struct {
	queries *query.Queries
}

func NewWorkspaceHandler(q *query.Queries) *WorkspaceHandler {
	return &WorkspaceHandler{queries: q}
}

type CreateWorkspaceReq struct {
	Name string `json:"name"`
}

type CreateChannelReq struct {
	Name string `json:"name"`
}

func (h *WorkspaceHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspaceID")
	var workspaceUUID pgtype.UUID
	if err := workspaceUUID.Scan(workspaceID); err != nil {
		http.Error(w, "Invalid workspace ID", http.StatusBadRequest)
		return
	}

	var req CreateChannelReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	safeName := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))

	channel, err := h.queries.CreateChannel(r.Context(), query.CreateChannelParams{
		WorkspaceID: workspaceUUID,
		Name:        safeName,
		IsPrivate:   false,
	})
	if err != nil {
		http.Error(w, "Failed to create channel", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(channel)
}

func (h *WorkspaceHandler) CreateWorkspaceWithGeneralChannel(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkspaceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	userIDStr := r.Context().Value(middleware.UserIDKey).(string)

	var userUUID pgtype.UUID
	userUUID.Scan(userIDStr)

	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	workspace, err := h.queries.CreateWorkspace(r.Context(), query.CreateWorkspaceParams{
		Name:    req.Name,
		Slug:    slug,
		OwnerID: userUUID,
	})
	if err != nil {
		http.Error(w, "Failed to create workspace", http.StatusInternalServerError)
		return
	}

	channel, err := h.queries.CreateChannel(r.Context(), query.CreateChannelParams{
		WorkspaceID: workspace.ID,
		Name:        "general",
		IsPrivate:   false,
	})
	if err != nil {
		http.Error(w, "Workspace created, but failed to create general channel", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = h.queries.JoinWorkspace(r.Context(), workspace.ID, userUUID)
	if err != nil {
		slog.Error("Failed to auto-join creator to workspace", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Success!",
		"workspace":  workspace,
		"channel_id": channel.ID,
	})
}

func (h *WorkspaceHandler) GetMyWorkspaces(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Context().Value(middleware.UserIDKey).(string)
	var userUUID pgtype.UUID
	userUUID.Scan(userIDStr)

	workspaces, err := h.queries.ListJoinedWorkspaces(r.Context(), userUUID)
	if err != nil {
		http.Error(w, "Failed to retrieve workspaces", http.StatusInternalServerError)
		return
	}

	if workspaces == nil {
		workspaces = []query.Workspace{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workspaces)
}

func (h *WorkspaceHandler) GetWorkspaceChannels(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspaceID")
	var workspaceUUID pgtype.UUID
	if err := workspaceUUID.Scan(workspaceID); err != nil {
		http.Error(w, "Invalid workspace ID format", http.StatusBadRequest)
		return
	}

	channels, err := h.queries.ListChannelsByWorkspace(r.Context(), workspaceUUID)
	if err != nil {
		http.Error(w, "Failed to retrieve channels", http.StatusInternalServerError)
		return
	}

	if channels == nil {
		channels = []query.Channel{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

func (h *WorkspaceHandler) JoinWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspaceID")
	var workspaceUUID pgtype.UUID
	if err := workspaceUUID.Scan(workspaceID); err != nil {
		http.Error(w, "Invalid workspace ID format", http.StatusBadRequest)
		return
	}

	userIDStr := r.Context().Value(middleware.UserIDKey).(string)
	var userUUID pgtype.UUID
	userUUID.Scan(userIDStr)

	err := h.queries.JoinWorkspace(r.Context(), workspaceUUID, userUUID)
	if err != nil {
		http.Error(w, "Failed to join workspace", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "joined successfully"}`))
}

func (h *WorkspaceHandler) GetDiscoverWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := h.queries.ListAllWorkspaces(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve public workspaces", http.StatusInternalServerError)
		return
	}

	if workspaces == nil {
		workspaces = []query.Workspace{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workspaces)
}
