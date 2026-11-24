package ratelimit

import (
	"testing"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
)

func TestRedisLimiter_FallbackWhenOffline(t *testing.T) {
	// Points to an unreachable port to test seamless in-memory fallback
	cfg := config.RateLimitConfig{
		Enabled:  true,
		Strategy: "redis",
		Rate:     5,
		Burst:    3,
		Redis: config.RedisConfig{
			Addr:    "127.0.0.1:65530", // offline port
			Timeout: 50 * time.Millisecond,
		},
	}

	limiter := NewRedisLimiter(cfg)
	defer limiter.Close()

	if limiter.IsHealthy() {
		t.Fatal("expected limiter to report unhealthy for unreachable redis")
	}

	ip := "203.0.113.1"

	// Fallback should allow up to burst (3)
	for i := 0; i < 3; i++ {
		if !limiter.Allow(ip) {
			t.Fatalf("request %d should have been allowed by fallback", i+1)
		}
	}

	// 4th request must be rejected
	if limiter.Allow(ip) {
		t.Fatal("4th request should have been rejected by fallback")
	}
}

func TestLimiterFactory(t *testing.T) {
	cfgMem := config.RateLimitConfig{
		Strategy: "memory",
		Rate:     10,
		Burst:    10,
	}
	limMem := New(cfgMem)
	defer limMem.Close()

	if limMem.Strategy() != "memory" {
		t.Fatalf("expected memory strategy, got %s", limMem.Strategy())
	}
}
