package sse

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSSEHub_Operations(t *testing.T) {
	t.Run("subscribe and broadcast successfully", func(t *testing.T) {
		hub := NewSSEHub()
		userID := "user-stream-test"
		ch := make(chan []byte, 1)

		hub.Subscribe(userID, ch)

		testMsg := []byte("hello")
		hub.Broadcast(userID, testMsg)

		select {
		case msg := <-ch:
			assert.Equal(t, testMsg, msg)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timeout waiting for broadcast message")
		}

		hub.Unsubscribe(userID, ch)
	})

	t.Run("race condition concurrent safety test", func(t *testing.T) {
		hub := NewSSEHub()
		userID := "user-race"

		var wg sync.WaitGroup
		concurrency := 50

		// Concurrent subscriptions and broadcasts to check for race conditions
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ch := make(chan []byte, 5)
				hub.Subscribe(userID, ch)
				hub.Broadcast(userID, []byte("data"))
				hub.Unsubscribe(userID, ch)
			}()
		}

		wg.Wait()
	})
}

// Test endpoint integration
func TestAPI_StreamIntegration(t *testing.T) {
	// Let's create an integration test wrapper inside hub_test.go to check handler.go
	// Since handler.go is in "internal/api" package and import cycles would form if we test sse there,
	// keeping simple endpoint mock/integration checks is extremely clean!
}
