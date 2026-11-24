package stream

import (
	"testing"
	"time"
)

func TestStreamBroker_RingBuffer(t *testing.T) {
	broker := NewStreamBroker(5)

	for i := 1; i <= 7; i++ {
		broker.Publish(RequestEvent{
			ID:         string(rune('0' + i)),
			StatusCode: 200,
			Timestamp:  time.Now(),
		})
	}

	history := broker.GetRecentHistory()
	if len(history) != 5 {
		t.Fatalf("Expected 5 history items, got %d", len(history))
	}

	// Oldest in history should be '3' and newest '7'
	if history[0].ID != "3" {
		t.Fatalf("Expected oldest ID to be '3', got %s", history[0].ID)
	}
	if history[4].ID != "7" {
		t.Fatalf("Expected newest ID to be '7', got %s", history[4].ID)
	}
}
