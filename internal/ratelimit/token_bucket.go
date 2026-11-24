package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/cespare/xxhash/v2"
)

const (
	defaultShardCount = 256
	clientTTL         = 5 * time.Minute
)

// clientBucket tracks token state for a specific client IP.
type clientBucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

// shard holds a subset of client buckets protected by an independent mutex.
type shard struct {
	mu      sync.Mutex
	buckets map[string]*clientBucket
}

// ShardedTokenBucket implements an ultra-high performance in-memory rate limiter
// designed to eliminate lock contention at 100k+ req/s.
type ShardedTokenBucket struct {
	rate       float64
	burst      float64
	numShards  uint64
	shards     []*shard
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

// NewShardedTokenBucket initializes a token bucket rate limiter with sharded state.
func NewShardedTokenBucket(cfg config.RateLimitConfig) *ShardedTokenBucket {
	numShards := uint64(defaultShardCount)
	shards := make([]*shard, numShards)
	for i := range shards {
		shards[i] = &shard{
			buckets: make(map[string]*clientBucket, 64),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	tb := &ShardedTokenBucket{
		rate:       cfg.Rate,
		burst:      float64(cfg.Burst),
		numShards:  numShards,
		shards:     shards,
		cancelFunc: cancel,
	}

	cleanupInterval := cfg.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = 1 * time.Minute
	}

	tb.wg.Add(1)
	go tb.janitor(ctx, cleanupInterval)

	return tb
}

// getShard returns the shard responsible for the given IP address.
func (tb *ShardedTokenBucket) getShard(ip string) *shard {
	hash := xxhash.Sum64String(ip)
	return tb.shards[hash%tb.numShards]
}

// Allow checks if 1 token is available for the given IP. Returns true if allowed.
func (tb *ShardedTokenBucket) Allow(ip string) bool {
	return tb.AllowN(ip, 1.0)
}

// AllowN checks if n tokens are available for the given IP. Returns true if allowed.
func (tb *ShardedTokenBucket) AllowN(ip string, n float64) bool {
	s := tb.getShard(ip)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	b, exists := s.buckets[ip]
	if !exists {
		// New client: starts with full capacity minus requested tokens
		if tb.burst < n {
			return false
		}
		s.buckets[ip] = &clientBucket{
			tokens:     tb.burst - n,
			lastRefill: now,
			lastSeen:   now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens = math.Min(tb.burst, b.tokens+elapsed*tb.rate)
		b.lastRefill = now
	}
	b.lastSeen = now

	if b.tokens >= n {
		b.tokens -= n
		return true
	}

	return false
}

// janitor periodically purges stale client entries to prevent memory leaks.
func (tb *ShardedTokenBucket) janitor(ctx context.Context, interval time.Duration) {
	defer tb.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tb.purgeStale(time.Now())
		}
	}
}

func (tb *ShardedTokenBucket) purgeStale(now time.Time) {
	for _, s := range tb.shards {
		s.mu.Lock()
		for ip, b := range s.buckets {
			if now.Sub(b.lastSeen) > clientTTL {
				delete(s.buckets, ip)
			}
		}
		s.mu.Unlock()
	}
}

// Close gracefully stops the cleanup routine.
func (tb *ShardedTokenBucket) Close() {
	if tb.cancelFunc != nil {
		tb.cancelFunc()
		tb.wg.Wait()
	}
}
