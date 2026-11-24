package ratelimit

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
)

func TestTokenBucket_BurstAndRefill(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:         true,
		Rate:            10, // 10 tokens/sec
		Burst:           5,  // burst 5
		CleanupInterval: time.Minute,
	}
	tb := NewShardedTokenBucket(cfg)
	defer tb.Close()

	ip := "192.168.1.10"

	// First 5 requests should pass (burst)
	for i := 0; i < 5; i++ {
		if !tb.Allow(ip) {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}

	// 6th request should fail immediately
	if tb.Allow(ip) {
		t.Fatal("request 6 should have been rejected (bucket empty)")
	}

	// Wait 250ms -> should refill ~2.5 tokens
	time.Sleep(250 * time.Millisecond)

	// Should allow 2 tokens now
	if !tb.Allow(ip) {
		t.Fatal("expected token to be refilled and allowed")
	}
	if !tb.Allow(ip) {
		t.Fatal("expected second refilled token to be allowed")
	}
	if tb.Allow(ip) {
		t.Fatal("expected third token to be rejected")
	}
}

func BenchmarkTokenBucket_ParallelSingleIP(b *testing.B) {
	cfg := config.RateLimitConfig{
		Enabled:         true,
		Rate:            10000000,
		Burst:           10000000,
		CleanupInterval: time.Minute,
	}
	tb := NewShardedTokenBucket(cfg)
	defer tb.Close()

	ip := "192.168.1.1"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow(ip)
		}
	})
}

func BenchmarkTokenBucket_ParallelMultiIP(b *testing.B) {
	cfg := config.RateLimitConfig{
		Enabled:         true,
		Rate:            10000000,
		Burst:           10000000,
		CleanupInterval: time.Minute,
	}
	tb := NewShardedTokenBucket(cfg)
	defer tb.Close()

	ips := make([]string, 1024)
	for i := range ips {
		ips[i] = fmt.Sprintf("10.0.%d.%d", (i/256)%256, i%256)
	}

	var counter uint64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			counter++
			idx := counter % uint64(len(ips))
			mu.Unlock()
			tb.Allow(ips[idx])
		}
	})
}
