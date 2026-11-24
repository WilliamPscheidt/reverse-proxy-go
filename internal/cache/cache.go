package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheConfig holds configuration options for HTTP edge caching.
type CacheConfig struct {
	Enabled    bool          `json:"enabled" yaml:"enabled"`
	DefaultTTL time.Duration `json:"default_ttl" yaml:"default_ttl"` // e.g. 60s
	MaxEntries int           `json:"max_entries" yaml:"max_entries"` // max items in memory (e.g. 10000)
}

// DefaultCacheConfig returns safe production defaults.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:    true,
		DefaultTTL: 60 * time.Second,
		MaxEntries: 10000,
	}
}

// CachedResponse stores the intercepted response payload and headers.
type CachedResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	ETag       string              `json:"etag"`
	ExpiresAt  time.Time           `json:"expires_at"`
}

// CacheStats provides live telemetry for the dashboard.
type CacheStats struct {
	Enabled       bool    `json:"enabled"`
	Hits          uint64  `json:"hits"`
	Misses        uint64  `json:"misses"`
	HitRatio      float64 `json:"hit_ratio"`
	CachedEntries int     `json:"cached_entries"`
	PurgeCount    uint64  `json:"purge_count"`
}

// Cache manages in-memory and distributed Redis response caching.
type Cache struct {
	mu         sync.RWMutex
	cfg        CacheConfig
	items      map[string]*CachedResponse
	rdb        *redis.Client
	hits       uint64
	misses     uint64
	purgeCount uint64
}

// NewCache creates an initialized edge cache.
func NewCache(cfg CacheConfig, rdb *redis.Client) *Cache {
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 60 * time.Second
	}
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 10000
	}
	c := &Cache{
		cfg:   cfg,
		items: make(map[string]*CachedResponse),
		rdb:   rdb,
	}

	// Background routine to evict expired items
	go c.startCleaner(10 * time.Second)

	return c
}

func (c *Cache) startCleaner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.ExpiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// CacheKey generates a normalized cache key from the request.
func (c *Cache) CacheKey(r *http.Request) string {
	return r.Method + ":" + r.Host + ":" + r.URL.RequestURI()
}

// IsCacheableRequest checks if the incoming request method and headers allow caching.
func (c *Cache) IsCacheableRequest(r *http.Request) bool {
	if !c.cfg.Enabled {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	cc := r.Header.Get("Cache-Control")
	if strings.Contains(cc, "no-store") {
		return false
	}
	return true
}

// Get retrieves a valid, non-expired cached response.
func (c *Cache) Get(r *http.Request) (*CachedResponse, bool) {
	if !c.IsCacheableRequest(r) {
		return nil, false
	}

	key := c.CacheKey(r)
	now := time.Now()

	// 1. Check local memory
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()

	if found {
		if now.Before(item.ExpiresAt) {
			atomic.AddUint64(&c.hits, 1)
			return item, true
		}
		// Expired
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
	}

	// 2. Check Redis if configured
	if c.rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		val, err := c.rdb.Get(ctx, "proxy:cache:"+key).Bytes()
		if err == nil {
			var cached CachedResponse
			if json.Unmarshal(val, &cached) == nil && now.Before(cached.ExpiresAt) {
				// Populate local memory
				c.mu.Lock()
				c.items[key] = &cached
				c.mu.Unlock()

				atomic.AddUint64(&c.hits, 1)
				return &cached, true
			}
		}
	}

	atomic.AddUint64(&c.misses, 1)
	return nil, false
}

// Put caches a response if status code and Cache-Control headers permit.
func (c *Cache) Put(r *http.Request, statusCode int, headers http.Header, body []byte) {
	if !c.IsCacheableRequest(r) || statusCode != http.StatusOK {
		return
	}

	resCC := headers.Get("Cache-Control")
	if strings.Contains(resCC, "no-store") || strings.Contains(resCC, "private") {
		return
	}

	ttl := c.cfg.DefaultTTL
	if strings.Contains(resCC, "max-age=") {
		parts := strings.Split(resCC, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "max-age=") {
				secStr := strings.TrimPrefix(p, "max-age=")
				if secs, err := strconv.Atoi(secStr); err == nil && secs > 0 {
					ttl = time.Duration(secs) * time.Second
				}
			}
		}
	}

	// Generate ETag (SHA-256 of response body)
	h := sha256.Sum256(body)
	etag := `W/"` + hex.EncodeToString(h[:8]) + `"`

	headerMap := make(map[string][]string, len(headers))
	for k, v := range headers {
		// Don't copy hop-by-hop headers
		if strings.EqualFold(k, "Connection") || strings.EqualFold(k, "Transfer-Encoding") {
			continue
		}
		headerMap[k] = v
	}

	cached := &CachedResponse{
		StatusCode: statusCode,
		Headers:    headerMap,
		Body:       body,
		ETag:       etag,
		ExpiresAt:  time.Now().Add(ttl),
	}

	key := c.CacheKey(r)

	c.mu.Lock()
	if len(c.items) < c.cfg.MaxEntries {
		c.items[key] = cached
	}
	c.mu.Unlock()

	// Persist in Redis asynchronously
	if c.rdb != nil {
		go func(k string, data *CachedResponse, expiration time.Duration) {
			b, err := json.Marshal(data)
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				defer cancel()
				_ = c.rdb.Set(ctx, "proxy:cache:"+k, b, expiration).Err()
			}
		}(key, cached, ttl)
	}
}

// ServeCached writes a cached response to the ResponseWriter, handling 304 Not Modified.
func (c *Cache) ServeCached(w http.ResponseWriter, r *http.Request, item *CachedResponse) {
	// Check If-None-Match
	if match := r.Header.Get("If-None-Match"); match != "" && (match == item.ETag || match == "*") {
		w.Header().Set("ETag", item.ETag)
		w.Header().Set("X-ProxyGO-Cache", "HIT")
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Copy headers
	for k, vv := range item.Headers {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.Header().Set("ETag", item.ETag)
	w.Header().Set("X-ProxyGO-Cache", "HIT")
	w.WriteHeader(item.StatusCode)
	_, _ = w.Write(item.Body)
}

// Purge removes cached items matching a pattern or exact key. Passing "*" clears all.
func (c *Cache) Purge(pattern string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	pattern = strings.TrimSpace(pattern)

	if pattern == "" || pattern == "*" {
		count = len(c.items)
		c.items = make(map[string]*CachedResponse)
	} else {
		for k := range c.items {
			if strings.Contains(k, pattern) {
				delete(c.items, k)
				count++
			}
		}
	}

	atomic.AddUint64(&c.purgeCount, uint64(count))

	// Invalidate in Redis
	if c.rdb != nil {
		go func(pat string) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if pat == "" || pat == "*" {
				keys, _ := c.rdb.Keys(ctx, "proxy:cache:*").Result()
				if len(keys) > 0 {
					_ = c.rdb.Del(ctx, keys...).Err()
				}
			} else {
				keys, _ := c.rdb.Keys(ctx, "proxy:cache:*"+pat+"*").Result()
				if len(keys) > 0 {
					_ = c.rdb.Del(ctx, keys...).Err()
				}
			}
		}(pattern)
	}

	return count
}

// Stats returns current cache statistics.
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	cachedCount := len(c.items)
	c.mu.RUnlock()

	hits := atomic.LoadUint64(&c.hits)
	misses := atomic.LoadUint64(&c.misses)
	total := hits + misses
	ratio := 0.0
	if total > 0 {
		ratio = float64(hits) / float64(total) * 100.0
	}

	return CacheStats{
		Enabled:       c.cfg.Enabled,
		Hits:          hits,
		Misses:        misses,
		HitRatio:      ratio,
		CachedEntries: cachedCount,
		PurgeCount:    atomic.LoadUint64(&c.purgeCount),
	}
}

// ResponseRecorder intercepts and records HTTP response bytes for caching.
type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode int
	Body       bytes.Buffer
}

func (rec *ResponseRecorder) WriteHeader(code int) {
	rec.StatusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *ResponseRecorder) Write(b []byte) (int, error) {
	if rec.StatusCode == 0 {
		rec.StatusCode = http.StatusOK
	}
	rec.Body.Write(b)
	return rec.ResponseWriter.Write(b)
}
