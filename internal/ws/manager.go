package ws

import (
	"log/slog"
	"net/http"

	"github.com/abhinavvv-chauhan/chat-app/internal/repository/query"
	"github.com/abhinavvv-chauhan/chat-app/pkg/token"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func ServeWS(hub *Hub, jwtSecret string, queries *query.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.URL.Query().Get("token")

		channelID := r.URL.Query().Get("channelId")

		if tokenStr == "" || channelID == "" {
			http.Error(w, "Missing token or channelId", http.StatusUnauthorized)
			return
		}

		claims, err := token.ValidateToken(tokenStr, jwtSecret)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		var userUUID pgtype.UUID
		userUUID.Scan(claims.UserID)
		username, _ := queries.GetUsername(r.Context(), userUUID)

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("Failed to upgrade websocket", "error", err)
			return
		}

		slog.Info("User joined channel!", "user_id", claims.UserID, "channel_id", channelID)

		client := &Client{
			hub:       hub,
			conn:      conn,
			send:      make(chan []byte, 256),
			userID:    claims.UserID,
			username:  username,
			channelID: channelID,
			queries:   queries,
		}

		client.hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}
