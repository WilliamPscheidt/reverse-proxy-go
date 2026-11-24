package resilience

import (
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// RetryConfig specifies retry behavior on upstream failures.
type RetryConfig struct {
	Enabled       bool          `json:"enabled" yaml:"enabled"`
	MaxRetries    int           `json:"max_retries" yaml:"max_retries"`       // e.g. 2
	InitialBackoff time.Duration `json:"initial_backoff" yaml:"initial_backoff"` // e.g. 25ms
	MaxBackoff    time.Duration `json:"max_backoff" yaml:"max_backoff"`         // e.g. 200ms
}

// DefaultRetryConfig returns safe defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Enabled:        true,
		MaxRetries:     2,
		InitialBackoff: 20 * time.Millisecond,
		MaxBackoff:     150 * time.Millisecond,
	}
}

// RetryPolicy executes retries with exponential backoff and jitter.
type RetryPolicy struct {
	cfg RetryConfig
}

// NewRetryPolicy initializes a retry manager.
func NewRetryPolicy(cfg RetryConfig) *RetryPolicy {
	return &RetryPolicy{cfg: cfg}
}

// IsRetryable determines if a request can safely be retried on another backend.
func (p *RetryPolicy) IsRetryable(req *http.Request, statusCode int, err error) bool {
	if !p.cfg.Enabled || p.cfg.MaxRetries <= 0 {
		return false
	}

	// Network / dial / socket error is always retryable
	if err != nil {
		return true
	}

	// Only retry idempotent HTTP methods to prevent duplicate mutation on POST/PATCH
	method := strings.ToUpper(req.Method)
	switch method {
	case "GET", "HEAD", "OPTIONS", "PUT", "DELETE":
		// Safe to retry
	default:
		return false
	}

	// Upstream failure status codes: 502 (Bad Gateway), 503 (Service Unavailable), 504 (Gateway Timeout)
	return statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

// BackoffDuration calculates exponential backoff with full jitter for attempt n (0-indexed).
func (p *RetryPolicy) BackoffDuration(attempt int) time.Duration {
	if attempt <= 0 {
		return p.cfg.InitialBackoff
	}

	backoff := p.cfg.InitialBackoff * time.Duration(1<<attempt)
	if backoff > p.cfg.MaxBackoff {
		backoff = p.cfg.MaxBackoff
	}

	// Full jitter between 0 and backoff
	jitter := time.Duration(rand.Int63n(int64(backoff) + 1))
	return jitter
}

// MaxRetries returns the configured maximum retry attempts.
func (p *RetryPolicy) MaxRetries() int {
	if !p.cfg.Enabled {
		return 0
	}
	return p.cfg.MaxRetries
}
