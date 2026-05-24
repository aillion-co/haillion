package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"aillion/services/notification/internal/sse"
)

type NotificationServer struct {
	hub sse.Hub
}

type Notification struct {
	UserID  string         `json:"user_id"`
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

func NewServer(hub sse.Hub) *http.ServeMux {
	server := &NotificationServer{hub: hub}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /stream", server.handleStream)
	mux.HandleFunc("POST /notify", server.handleNotify)

	return mux
}

func (s *NotificationServer) handleStream(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "missing query parameter 'user_id'", http.StatusBadRequest)
		return
	}

	// Verify if the client connection supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Set headers required for Server-Sent Events (SSE)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	// Create subscription channel
	messageChan := make(chan []byte, 10)
	s.hub.Subscribe(userID, messageChan)
	defer func() {
		s.hub.Unsubscribe(userID, messageChan)
		close(messageChan)
	}()

	// Keepalive ticker to prevent connection timeouts (15 seconds)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-messageChan:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", string(msg))
			flusher.Flush()

		case <-ticker.C:
			// Send heartbeat/ping comment to keep connection alive
			_, _ = fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

func (s *NotificationServer) handleNotify(w http.ResponseWriter, r *http.Request) {
	var req Notification
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.Type == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	msgBytes, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "failed to marshal payload", http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast(req.UserID, msgBytes)

	w.WriteHeader(http.StatusOK)
}
