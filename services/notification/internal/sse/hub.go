package sse

import (
	"sync"
)

type Hub interface {
	Subscribe(userID string, ch chan []byte)
	Unsubscribe(userID string, ch chan []byte)
	Broadcast(userID string, msg []byte)
}

type SSEHub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan []byte]bool
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		subscribers: make(map[string]map[chan []byte]bool),
	}
}

func (h *SSEHub) Subscribe(userID string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.subscribers[userID]; !exists {
		h.subscribers[userID] = make(map[chan []byte]bool)
	}
	h.subscribers[userID][ch] = true
}

func (h *SSEHub) Unsubscribe(userID string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if userSubs, exists := h.subscribers[userID]; exists {
		delete(userSubs, ch)
		if len(userSubs) == 0 {
			delete(h.subscribers, userID)
		}
	}
}

func (h *SSEHub) Broadcast(userID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if userSubs, exists := h.subscribers[userID]; exists {
		for ch := range userSubs {
			// Non-blocking write to channel to avoid slowing down the broadcaster
			// if a single client is slow or has disconnected
			select {
			case ch <- msg:
			default:
				// Buffer full or slow client, safe to drop or handle
			}
		}
	}
}

// Ensure SSEHub implements Hub
var _ Hub = (*SSEHub)(nil)
