package server

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/abhinavvv-chauhan/chat-app/internal/handler"
	"github.com/abhinavvv-chauhan/chat-app/internal/repository/query"
	"github.com/abhinavvv-chauhan/chat-app/internal/server/middleware"
	"github.com/abhinavvv-chauhan/chat-app/internal/ws"
	"github.com/go-chi/chi/v5"
)

func SetupRoutes(r *chi.Mux, queries *query.Queries, jwtSecret string, hub *ws.Hub) {

	authHandler := handler.NewAuthHandler(queries, jwtSecret)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Get("/ws", ws.ServeWS(hub, jwtSecret, queries))

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthGuard(jwtSecret))
			
			r.Post("/upload", handler.UploadFile)
			
			workspaceHandler := handler.NewWorkspaceHandler(queries)
			r.Post("/workspaces", workspaceHandler.CreateWorkspaceWithGeneralChannel)
			r.Get("/workspaces", workspaceHandler.GetMyWorkspaces) 
			r.Get("/workspaces/{workspaceID}/channels", workspaceHandler.GetWorkspaceChannels) 
			r.Post("/workspaces/{workspaceID}/channels", workspaceHandler.CreateChannel)
			r.Post("/workspaces/{workspaceID}/join", workspaceHandler.JoinWorkspace)
			r.Get("/discover", workspaceHandler.GetDiscoverWorkspaces)

			messageHandler := handler.NewMessageHandler(queries)
			r.Get("/channels/{channelID}/messages", messageHandler.GetMessages)

			pollHandler := handler.NewPollHandler(queries)
			r.Post("/polls", pollHandler.CreatePoll)
			r.Get("/polls/{pollID}", pollHandler.GetPoll)
			r.Post("/polls/{pollID}/vote", pollHandler.Vote)

			r.Get("/me", func(w http.ResponseWriter, req *http.Request) {
				userID := req.Context().Value(middleware.UserIDKey)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status": "success", "message": "VIP Access Granted", "user_id": "` + userID.(string) + `"}`))
			})
		})
	})

	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "uploads"))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(filesDir)))

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "web/index.html")
	})
}