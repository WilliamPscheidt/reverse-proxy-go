package resilience

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestCircuitBreaker_Lifecycle(t *testing.T) {
	cfg := CircuitConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Cooldown:         50 * time.Millisecond,
	}

	cb := NewCircuitBreaker("backend-1", cfg)

	// Initially closed
	if cb.State() != StateClosed {
		t.Fatalf("Expected StateClosed, got %v", cb.State())
	}
	if !cb.Allow() {
		t.Fatal("Expected Allow() = true in closed state")
	}

	// 2 failures (under threshold)
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Fatalf("Expected StateClosed after 2 failures, got %v", cb.State())
	}

	// 3rd failure trips to OPEN
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatalf("Expected StateOpen after 3 failures, got %v", cb.State())
	}
	if cb.Allow() {
		t.Fatal("Expected Allow() = false in open state before cooldown")
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Next Allow() transitions to HALF_OPEN
	if !cb.Allow() {
		t.Fatal("Expected Allow() = true after cooldown in Half-Open")
	}
	if cb.State() != StateHalfOpen {
		t.Fatalf("Expected StateHalfOpen, got %v", cb.State())
	}

	// 1st success in half-open
	cb.RecordSuccess()
	if cb.State() != StateHalfOpen {
		t.Fatal("Expected StateHalfOpen after 1 success")
	}

	// 2nd success in half-open closes the circuit
	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Fatalf("Expected StateClosed after 2 successes, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cfg := CircuitConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Cooldown:         10 * time.Millisecond,
	}

	cb := NewCircuitBreaker("backend-fail", cfg)
	cb.RecordFailure() // Trips to Open
	time.Sleep(15 * time.Millisecond)
	cb.Allow() // Transitions to HalfOpen

	// Failure in HalfOpen trips immediately to Open
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatalf("Expected StateOpen on failure during half-open, got %v", cb.State())
	}
}

func TestRetryPolicy_Rules(t *testing.T) {
	cfg := DefaultRetryConfig()
	p := NewRetryPolicy(cfg)

	// GET with 502 Bad Gateway is retryable
	reqGet, _ := http.NewRequest("GET", "http://example.com/api", nil)
	if !p.IsRetryable(reqGet, http.StatusBadGateway, nil) {
		t.Fatal("Expected GET 502 to be retryable")
	}

	// GET with 200 OK is NOT retryable
	if p.IsRetryable(reqGet, http.StatusOK, nil) {
		t.Fatal("Expected GET 200 NOT to be retryable")
	}

	// POST with 502 is NOT retryable (non-idempotent)
	reqPost, _ := http.NewRequest("POST", "http://example.com/api", nil)
	if p.IsRetryable(reqPost, http.StatusBadGateway, nil) {
		t.Fatal("Expected POST 502 NOT to be retryable")
	}

	// Network error is retryable for GET
	if !p.IsRetryable(reqGet, 0, errors.New("connection reset")) {
		t.Fatal("Expected network error to be retryable")
	}
}
