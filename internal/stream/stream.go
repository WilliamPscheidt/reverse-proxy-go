package stream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RequestEvent represents a real-time event of a request processed by ProxyGO.
type RequestEvent struct {
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	ClientIP   string    `json:"client_ip"`
	Country    string    `json:"country"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	DurationMs float64   `json:"duration_ms"`
	Upstream   string    `json:"upstream"`
	CacheHit   bool      `json:"cache_hit"`
	WAFBlocked bool      `json:"waf_blocked"`
	WAFRule    string    `json:"waf_rule,omitempty"`
}

// StreamBroker manages real-time SSE client connections and event broadcasting.
type StreamBroker struct {
	mu          sync.RWMutex
	clients     map[chan RequestEvent]struct{}
	history     []RequestEvent
	historyCap  int
	historyIdx  int
	historyFull bool
}

// NewStreamBroker creates an event broker with a ring buffer for recent history.
func NewStreamBroker(historySize int) *StreamBroker {
	if historySize <= 0 {
		historySize = 100
	}
	return &StreamBroker{
		clients:    make(map[chan RequestEvent]struct{}),
		history:    make([]RequestEvent, historySize),
		historyCap: historySize,
	}
}

// Publish broadcasts an event to all connected clients and stores it in history.
// Guaranteed non-blocking for proxy traffic.
func (b *StreamBroker) Publish(evt RequestEvent) {
	b.mu.Lock()
	// Store in ring buffer
	b.history[b.historyIdx] = evt
	b.historyIdx++
	if b.historyIdx >= b.historyCap {
		b.historyIdx = 0
		b.historyFull = true
	}

	// Send to connected clients
	for ch := range b.clients {
		select {
		case ch <- evt:
		default:
			// Client buffer full; skip to keep proxy lock-free
		}
	}
	b.mu.Unlock()
}

// GetRecentHistory returns the most recent events in chronological order.
func (b *StreamBroker) GetRecentHistory() []RequestEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.historyFull {
		res := make([]RequestEvent, b.historyIdx)
		copy(res, b.history[:b.historyIdx])
		return res
	}

	res := make([]RequestEvent, b.historyCap)
	copy(res, b.history[b.historyIdx:])
	copy(res[b.historyCap-b.historyIdx:], b.history[:b.historyIdx])
	return res
}

// ServeSSE handles Server-Sent Events HTTP connections.
func (b *StreamBroker) ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if origin := r.Header.Get("Origin"); origin != "" {
		if strings.Contains(origin, r.Host) || strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
	}

	// Subscribe client
	ch := make(chan RequestEvent, 64)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.clients, ch)
		close(ch)
		b.mu.Unlock()
	}()

	// Flush recent history first so the stream starts populated
	history := b.GetRecentHistory()
	for _, evt := range history {
		if data, err := json.Marshal(evt); err == nil {
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
	}
	flusher.Flush()

	// Stream live events
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err == nil {
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}
