package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/abhinavvv-chauhan/chat-app/internal/server/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spf13/viper"
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

	apiKey := viper.GetString("GROQ_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GROQ_API_KEY")
	}
	if apiKey == "" {
		http.Error(w, "GROQ_API_KEY is not configured in .env or environment variables", http.StatusInternalServerError)
		return
	}

	systemPrompt := "You are an AI assistant in a chat app. The user has been offline and missed the provided chat transcript. Summarize the conversation into three JSON fields: 'threads' (array of strings, summarizing topics), 'action_items' (array of objects with 'assignee' (string) and 'task' (string)), and 'decisions' (array of strings of finalized conclusions). Respond ONLY with valid JSON matching this schema."

	reqBody := map[string]interface{}{
		"model": "llama-3.3-70b-versatile",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": transcript},
		},
		"response_format": map[string]string{"type": "json_object"},
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to call Groq API: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		http.Error(w, "Groq API error: "+string(bodyBytes), http.StatusInternalServerError)
		return
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		http.Error(w, "Failed to decode Groq response", http.StatusInternalServerError)
		return
	}

	if len(result.Choices) == 0 {
		http.Error(w, "Empty response from Groq", http.StatusInternalServerError)
		return
	}

	txt := result.Choices[0].Message.Content

	cacheMutex.Lock()
	summaryCache[cacheKey] = cachedSummary{
		Summary:   txt,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	cacheMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(txt))
}
