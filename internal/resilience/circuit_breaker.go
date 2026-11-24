package resilience

import (
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState string

const (
	StateClosed   CircuitState = "CLOSED"
	StateOpen     CircuitState = "OPEN"
	StateHalfOpen CircuitState = "HALF_OPEN"
)

// CircuitConfig configures threshold rules for circuit breaking.
type CircuitConfig struct {
	FailureThreshold int           `json:"failure_threshold" yaml:"failure_threshold"` // consecutive failures to trip
	SuccessThreshold int           `json:"success_threshold" yaml:"success_threshold"` // consecutive successes in half-open to close
	Cooldown         time.Duration `json:"cooldown" yaml:"cooldown"`                   // time to wait before half-open trial
}

// DefaultCircuitConfig returns standard production defaults.
func DefaultCircuitConfig() CircuitConfig {
	return CircuitConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Cooldown:         5 * time.Second,
	}
}

// CircuitStats holds runtime statistics of a circuit breaker.
type CircuitStats struct {
	ID                  string       `json:"id"`
	State               CircuitState `json:"state"`
	ConsecutiveFailures int          `json:"consecutive_failures"`
	TotalTrips          uint64       `json:"total_trips"`
	LastTripTime        *time.Time   `json:"last_trip_time,omitempty"`
}

// CircuitBreaker implements the tri-state circuit breaker pattern for backend protection.
type CircuitBreaker struct {
	mu                  sync.RWMutex
	id                  string
	cfg                 CircuitConfig
	state               CircuitState
	consecutiveFailures int
	consecutiveSuccess  int
	openedAt            time.Time
	totalTrips          uint64
}

// NewCircuitBreaker creates a new circuit breaker instance.
func NewCircuitBreaker(id string, cfg CircuitConfig) *CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = 5 * time.Second
	}
	return &CircuitBreaker{
		id:    id,
		cfg:   cfg,
		state: StateClosed,
	}
}

// Allow checks if a request is permitted to reach the target backend.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if cooldown has elapsed
		if now.Sub(cb.openedAt) >= cb.cfg.Cooldown {
			cb.state = StateHalfOpen
			cb.consecutiveSuccess = 0
			return true // allow single trial request
		}
		return false

	case StateHalfOpen:
		// Probing phase: allow trial requests
		return true

	default:
		return true
	}
}

// RecordSuccess registers a successful backend response.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.consecutiveSuccess++
		if cb.consecutiveSuccess >= cb.cfg.SuccessThreshold {
			cb.state = StateClosed
			cb.consecutiveFailures = 0
			cb.consecutiveSuccess = 0
		}
	case StateClosed:
		cb.consecutiveFailures = 0
	}
}

// RecordFailure registers an error, timeout, or 5xx from the backend.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		// Any failure in half-open immediately trips back to Open
		cb.state = StateOpen
		cb.openedAt = time.Now()
		cb.totalTrips++
		cb.consecutiveSuccess = 0

	case StateClosed:
		cb.consecutiveFailures++
		if cb.consecutiveFailures >= cb.cfg.FailureThreshold {
			cb.state = StateOpen
			cb.openedAt = time.Now()
			cb.totalTrips++
		}
	}
}

// State returns the current state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns a snapshot of breaker status.
func (cb *CircuitBreaker) Stats() CircuitStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var lastTrip *time.Time
	if !cb.openedAt.IsZero() {
		t := cb.openedAt
		lastTrip = &t
	}

	return CircuitStats{
		ID:                  cb.id,
		State:               cb.state,
		ConsecutiveFailures: cb.consecutiveFailures,
		TotalTrips:          cb.totalTrips,
		LastTripTime:        lastTrip,
	}
}

// CircuitRegistry manages a map of circuit breakers per backend address.
type CircuitRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	cfg      CircuitConfig
}

// NewCircuitRegistry creates a central registry for circuit breakers.
func NewCircuitRegistry(cfg CircuitConfig) *CircuitRegistry {
	return &CircuitRegistry{
		breakers: make(map[string]*CircuitBreaker),
		cfg:      cfg,
	}
}

// GetOrCreate returns or instantiates a circuit breaker for the given backend.
func (r *CircuitRegistry) GetOrCreate(backendID string) *CircuitBreaker {
	r.mu.RLock()
	cb, exists := r.breakers[backendID]
	r.mu.RUnlock()
	if exists {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if cb, exists = r.breakers[backendID]; exists {
		return cb
	}

	cb = NewCircuitBreaker(backendID, r.cfg)
	r.breakers[backendID] = cb
	return cb
}

// ListStats returns the state and statistics of all active circuit breakers.
func (r *CircuitRegistry) ListStats() []CircuitStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make([]CircuitStats, 0, len(r.breakers))
	for _, cb := range r.breakers {
		stats = append(stats, cb.Stats())
	}
	return stats
}
