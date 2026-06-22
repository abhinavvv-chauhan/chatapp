package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/abhinavvv-chauhan/chat-app/internal/repository/query"
	"github.com/abhinavvv-chauhan/chat-app/pkg/password"
	"github.com/abhinavvv-chauhan/chat-app/pkg/token"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuthHandler struct {
	queries   *query.Queries
	jwtSecret string
}

func NewAuthHandler(q *query.Queries, secret string) *AuthHandler {
	return &AuthHandler{queries: q, jwtSecret: secret}
}

type RegisterReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	hashedPassword, err := password.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.queries.CreateUser(r.Context(), query.CreateUserParams{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Username:     req.Username,
		AvatarUrl:    pgtype.Text{Valid: false}, 
	})
	if err != nil {
		http.Error(w, "Failed to create user. Email or username might already exist.", http.StatusConflict)
		return
	}


	tokenStr, err := token.GenerateAccessToken(user.ID, h.jwtSecret, time.Hour*24*30)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": tokenStr,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user, err := h.queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if err := password.CheckPassword(req.Password, user.PasswordHash); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}


	tokenStr, err := token.GenerateAccessToken(user.ID, h.jwtSecret, time.Hour*24*30)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": tokenStr,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}