package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"aillion/services/notification/internal/sse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_NotificationStreamFlow(t *testing.T) {
	hub := sse.NewSSEHub()
	mux := NewServer(hub)

	server := httptest.NewServer(mux)
	defer server.Close()

	userID := "rider-777"

	// Create SSE Client Connection
	clientCtx, clientCancel := context.WithCancel(context.Background())
	defer clientCancel()

	req, err := http.NewRequestWithContext(clientCtx, "GET", server.URL+"/stream?user_id="+userID, nil)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Trigger broadcast notification via POST /notify
	payload := Notification{
		UserID:  userID,
		Type:    "trip_accepted",
		Payload: map[string]any{"trip_id": "trip-abc"},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	notifyReq, err := http.NewRequest("POST", server.URL+"/notify", bytes.NewReader(body))
	require.NoError(t, err)

	notifyResp, err := client.Do(notifyReq)
	require.NoError(t, err)
	defer func() { _ = notifyResp.Body.Close() }()

	assert.Equal(t, http.StatusOK, notifyResp.StatusCode)

	// Read stream response on client side
	reader := io.LimitReader(resp.Body, 1024)
	buf := make([]byte, 512)
	n, err := reader.Read(buf)
	require.NoError(t, err)

	streamOutput := string(buf[:n])
	assert.Contains(t, streamOutput, "data:")
	assert.Contains(t, streamOutput, "trip_accepted")
	assert.Contains(t, streamOutput, "trip-abc")
}

func TestHandler_StreamMissingParam(t *testing.T) {
	hub := sse.NewSSEHub()
	mux := NewServer(hub)

	req := httptest.NewRequest("GET", "/stream", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing query parameter")
}

func TestHandler_NotifyInvalidFields(t *testing.T) {
	hub := sse.NewSSEHub()
	mux := NewServer(hub)

	// Missing fields
	body, err := json.Marshal(Notification{Type: "trip"})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/notify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing required fields")
}
