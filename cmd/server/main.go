package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abhinavvv-chauhan/chat-app/internal/config"
	"github.com/abhinavvv-chauhan/chat-app/internal/repository" 
	"github.com/abhinavvv-chauhan/chat-app/internal/repository/query"
	"github.com/abhinavvv-chauhan/chat-app/internal/server"
	"github.com/abhinavvv-chauhan/chat-app/internal/ws"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	dbPool, err := repository.NewDBPool(cfg.DBUrl)
	if err != nil {
		slog.Error("Database pool initialization failed", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	queries := query.New(dbPool) 

	wsHub := ws.NewHub()
	go wsHub.Run()

	router := server.NewServer()
	
	server.SetupRoutes(router, queries, cfg.JWTSecret, wsHub)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		slog.Info("Starting server", "port", cfg.ServerPort, "env", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit 

	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exiting gracefully")
}