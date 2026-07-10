package handler

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/abhinavvv-chauhan/chat-app/internal/server/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/generative-ai-go/genai"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/option"
)

type cachedSummary struct {
	Summary   string
	ExpiresAt time.Time
}

var (
	summaryCache = make(map[string]cachedSummary)
	cacheMutex   sync.Mutex
)

func (h *MessageHandler) HandleCatchUp(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	if channelID == "" {
		http.Error(w, "Channel ID is required", http.StatusBadRequest)
		return
	}

	userIDStr := r.Context().Value(middleware.UserIDKey).(string)
	var userUUID pgtype.UUID
	userUUID.Scan(userIDStr)
	
	var chanUUID pgtype.UUID
	chanUUID.Scan(channelID)

	messages, err := h.queries.GetMissedMessages(r.Context(), chanUUID, userUUID)
	if err != nil {
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	if len(messages) == 0 {
		// Fallback: If there are no strictly "unread" messages, just summarize the last 50 messages
		// so the feature is always usable and testable.
		messages, err = h.queries.GetMessagesWithUser(r.Context(), chanUUID, 50, 0)
		if err != nil || len(messages) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"threads": [], "action_items": [], "decisions": []}`))
			return
		}
	}

	var transcriptBuilder strings.Builder
	for _, msg := range messages {
		transcriptBuilder.WriteString(fmt.Sprintf("[%s] @%s: %s\n", msg.CreatedAt.Time.Format("15:04"), msg.Username, msg.Content))
	}
	transcript := transcriptBuilder.String()

	cacheKey := fmt.Sprintf("%s:%s", channelID, userIDStr)
	cacheMutex.Lock()
	if c, ok := summaryCache[cacheKey]; ok && time.Now().Before(c.ExpiresAt) {
		cacheMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(c.Summary))
		return
	}
	cacheMutex.Unlock()

	apiKey := viper.GetString("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		http.Error(w, "Gemini API key is not configured in .env or environment variables", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		http.Error(w, "Failed to initialize AI client: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-3.5-flash")
	model.ResponseMIMEType = "application/json"
	
	systemPrompt := genai.Text("You are an AI assistant in a chat app. The user has been offline and missed the provided chat transcript. Summarize the conversation into three JSON fields: 'threads' (array of strings, summarizing topics), 'action_items' (array of objects with 'assignee' (string) and 'task' (string)), and 'decisions' (array of strings of finalized conclusions). Respond ONLY with valid JSON matching this schema.")
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{systemPrompt},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(transcript))
	if err != nil {
		fmt.Printf("Gemini API error: %v\n", err)
		http.Error(w, "Failed to generate summary: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		http.Error(w, "Empty response from AI", http.StatusInternalServerError)
		return
	}

	part := resp.Candidates[0].Content.Parts[0]
	if txt, ok := part.(genai.Text); ok {
		cacheMutex.Lock()
		summaryCache[cacheKey] = cachedSummary{
			Summary:   string(txt),
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		cacheMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(txt))
		return
	}

	http.Error(w, "Invalid response from AI", http.StatusInternalServerError)
}
