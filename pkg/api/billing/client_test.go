package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBillingClient_GetSurge(t *testing.T) {
	t.Run("successful surge retrieval", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/surge", r.URL.Path)
			assert.Equal(t, "10", r.URL.Query().Get("riders"))
			assert.Equal(t, "5", r.URL.Query().Get("drivers"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"multiplier":1.5}`))
		}))
		defer server.Close()

		client := NewClient(server.URL, nil)
		multiplier, err := client.GetSurge(context.Background(), 10, 5)
		assert.NoError(t, err)
		assert.InDelta(t, 1.5, multiplier, 0.001)
	})

	t.Run("non-2xx status code error handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		client := NewClient(server.URL, nil)
		_, err := client.GetSurge(context.Background(), 10, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "surge request failed with status: 400")
	})

	t.Run("request timeout cancellation safety", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond) // Simulate slow downstream
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.URL, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		_, err := client.GetSurge(ctx, 10, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), context.DeadlineExceeded.Error())
	})
}
