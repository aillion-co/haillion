package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("empty URL returns error", func(t *testing.T) {
		client, err := NewClient("", nil)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "base URL cannot be empty")
	})

	t.Run("valid URL returns client", func(t *testing.T) {
		client, err := NewClient("http://localhost:8080", nil)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})
}

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

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)
		multiplier, err := client.GetSurge(context.Background(), 10, 5)
		assert.NoError(t, err)
		assert.InDelta(t, 1.5, multiplier, 0.001)
	})

	t.Run("non-2xx status code error handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)
		_, err = client.GetSurge(context.Background(), 10, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "surge request failed with status: 400")
	})

	t.Run("request timeout cancellation safety", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond) // Simulate slow downstream
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		_, err = client.GetSurge(ctx, 10, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), context.DeadlineExceeded.Error())
	})
}

func TestBillingClient_EstimateFare(t *testing.T) {
	t.Run("successful fare estimate", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/fare/estimate", r.URL.Path)
			assert.Equal(t, "SW1A", r.URL.Query().Get("from"))
			assert.Equal(t, "EC1A", r.URL.Query().Get("to"))
			assert.Equal(t, "10", r.URL.Query().Get("riders"))
			assert.Equal(t, "5", r.URL.Query().Get("drivers"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"distance_miles": 1.78,
				"base_pence": 178,
				"surge_multiplier": 1.2,
				"total_pence": 250,
				"minimum_applied": true
			}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		fare, err := client.EstimateFare(context.Background(), "SW1A", "EC1A", 10, 5)
		assert.NoError(t, err)
		assert.InDelta(t, 1.78, fare.DistanceMiles, 0.001)
		assert.Equal(t, int64(178), fare.BasePence)
		assert.InDelta(t, 1.2, fare.SurgeMultiplier, 0.001)
		assert.Equal(t, int64(250), fare.TotalPence)
		assert.True(t, fare.MinimumApplied)
	})

	t.Run("non-2xx status code error handling with snippet", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`invalid postcode format error message that is hopefully detailed`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		_, err = client.EstimateFare(context.Background(), "invalid", "postcode", 0, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 400")
		assert.Contains(t, err.Error(), "invalid postcode format error message")
	})

	t.Run("request timeout cancellation safety", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		_, err = client.EstimateFare(ctx, "SW1A", "EC1A", 0, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), context.DeadlineExceeded.Error())
	})
}
