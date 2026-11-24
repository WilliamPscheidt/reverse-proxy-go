package ratelimit

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/redis/go-redis/v9"
)

// luaTokenBucketScript is an atomic Redis Lua script implementing the Token Bucket algorithm per client IP.
// KEYS[1]: ratelimit:{client_ip}
// ARGV[1]: burst capacity
// ARGV[2]: refill rate per second
// ARGV[3]: current unix timestamp in seconds (float)
// ARGV[4]: requested tokens count
const luaTokenBucketScript = `
local key = KEYS[1]
local burst = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local data = redis.call('HMGET', key, 'tokens', 'last_updated')
local tokens = tonumber(data[1])
local last_updated = tonumber(data[2])

if not tokens then
    tokens = burst
    last_updated = now
else
    local delta = math.max(0, now - last_updated)
    tokens = math.min(burst, tokens + delta * rate)
    last_updated = now
end

if tokens >= requested then
    tokens = tokens - requested
    redis.call('HMSET', key, 'tokens', tokens, 'last_updated', last_updated)
    local ttl = math.ceil(burst / rate) + 60
    redis.call('EXPIRE', key, ttl)
    return 1
else
    redis.call('HMSET', key, 'tokens', tokens, 'last_updated', last_updated)
    return 0
end
`

// RedisLimiter implements distributed per-IP rate limiting backed by Redis.
type RedisLimiter struct {
	cfgMu       sync.RWMutex
	client      *redis.Client
	script      *redis.Script
	rate        float64
	burst       float64
	timeout     time.Duration
	fallback    *ShardedTokenBucket
	healthy     atomic.Bool
	stopCh      chan struct{}
}

// NewRedisLimiter creates a new distributed rate limiter with fallback to in-memory on partition.
func NewRedisLimiter(cfg config.RateLimitConfig) *RedisLimiter {
	timeout := cfg.Redis.Timeout
	if timeout <= 0 {
		timeout = 200 * time.Millisecond
	}

	poolSize := cfg.Redis.PoolSize
	if poolSize <= 0 {
		poolSize = 100
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  timeout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		PoolSize:     poolSize,
	})

	rl := &RedisLimiter{
		client:   rdb,
		script:   redis.NewScript(luaTokenBucketScript),
		rate:     cfg.Rate,
		burst:    float64(cfg.Burst),
		timeout:  timeout,
		fallback: NewShardedTokenBucket(cfg),
		stopCh:   make(chan struct{}),
	}

	// Initial health probe
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err == nil {
		rl.healthy.Store(true)
		log.Printf("[RateLimit:Redis] Connected to Redis at %s (Distributed Rate Limiting Active)", cfg.Redis.Addr)
	} else {
		rl.healthy.Store(false)
		log.Printf("[RateLimit:Redis] Warning: Redis at %s unavailable (%v). Using local in-memory fallback.", cfg.Redis.Addr, err)
	}

	go rl.healthProbeLoop(cfg.Redis.Addr)

	return rl
}

func (r *RedisLimiter) healthProbeLoop(addr string) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
			err := r.client.Ping(ctx).Err()
			cancel()

			wasHealthy := r.healthy.Load()
			isHealthy := (err == nil)

			if isHealthy != wasHealthy {
				r.healthy.Store(isHealthy)
				if isHealthy {
					log.Printf("[RateLimit:Redis] Restored connection to Redis at %s", addr)
				} else {
					log.Printf("[RateLimit:Redis] Lost connection to Redis at %s (%v). Switching to local fallback.", addr, err)
				}
			}
		}
	}
}

// SetRateAndBurst dynamically updates the rate and burst parameters for Redis and fallback.
func (r *RedisLimiter) SetRateAndBurst(rate float64, burst int) {
	r.cfgMu.Lock()
	defer r.cfgMu.Unlock()
	r.rate = rate
	r.burst = float64(burst)
	if r.fallback != nil {
		r.fallback.SetRateAndBurst(rate, burst)
	}
}

// GetRateAndBurst returns the current rate and burst values.
func (r *RedisLimiter) GetRateAndBurst() (float64, int) {
	r.cfgMu.RLock()
	defer r.cfgMu.RUnlock()
	return r.rate, int(r.burst)
}

// Strategy returns "redis" when healthy, or "redis (fallback: memory)" when offline.
func (r *RedisLimiter) Strategy() string {
	if r.healthy.Load() {
		return "redis"
	}
	return "redis (fallback: memory)"
}

// IsHealthy returns true if the Redis connection is active.
func (r *RedisLimiter) IsHealthy() bool {
	return r.healthy.Load()
}

// Allow checks if 1 token is available for client IP.
func (r *RedisLimiter) Allow(ip string) bool {
	return r.AllowN(ip, 1.0)
}

// AllowN evaluates rate limit atomically in Redis or falls back to in-memory.
func (r *RedisLimiter) AllowN(ip string, n float64) bool {
	if !r.healthy.Load() {
		return r.fallback.AllowN(ip, n)
	}

	r.cfgMu.RLock()
	burst := r.burst
	rate := r.rate
	r.cfgMu.RUnlock()

	key := fmt.Sprintf("ratelimit:%s", ip)
	now := float64(time.Now().UnixNano()) / 1e9

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	res, err := r.script.Run(ctx, r.client, []string{key}, burst, rate, now, n).Int64()
	if err != nil {
		// Log error and transparently fallback to avoid blocking valid traffic
		r.healthy.Store(false)
		return r.fallback.AllowN(ip, n)
	}

	return res == 1
}

// Close gracefully stops the health probe and closes client connections.
func (r *RedisLimiter) Close() {
	close(r.stopCh)
	_ = r.client.Close()
	if r.fallback != nil {
		r.fallback.Close()
	}
}
