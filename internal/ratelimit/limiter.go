package ratelimit

import "github.com/WilliamPscheidt/reverse-proxy-go/internal/config"

// Limiter defines the unified contract for both in-memory and distributed rate limiters.
type Limiter interface {
	Allow(ip string) bool
	AllowN(ip string, n float64) bool
	Close()
	Strategy() string
	IsHealthy() bool
}

// New creates a Limiter according to the configuration strategy ("redis" or "memory").
func New(cfg config.RateLimitConfig) Limiter {
	if cfg.Strategy == "redis" && cfg.Redis.Addr != "" {
		return NewRedisLimiter(cfg)
	}
	return NewShardedTokenBucket(cfg)
}
