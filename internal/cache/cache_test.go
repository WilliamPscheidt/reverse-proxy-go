package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCache_PutAndGet(t *testing.T) {
	cfg := DefaultCacheConfig()
	c := NewCache(cfg, nil)

	req, _ := http.NewRequest("GET", "http://example.com/api/data?q=1", nil)

	// Initially miss
	_, hit := c.Get(req)
	if hit {
		t.Fatal("Expected cache miss initially")
	}

	headers := http.Header{"Content-Type": []string{"application/json"}}
	body := []byte(`{"message":"hello world"}`)
	c.Put(req, http.StatusOK, headers, body)

	// Now hit
	cached, hit := c.Get(req)
	if !hit {
		t.Fatal("Expected cache hit after Put")
	}
	if string(cached.Body) != string(body) {
		t.Fatalf("Expected cached body %s, got %s", body, cached.Body)
	}
	if cached.ETag == "" {
		t.Fatal("Expected ETag to be generated")
	}

	stats := c.Stats()
	if stats.Hits != 1 || stats.Misses != 1 {
		t.Fatalf("Expected 1 hit and 1 miss, got hits=%d, misses=%d", stats.Hits, stats.Misses)
	}
}

func TestCache_ServeCached_304(t *testing.T) {
	c := NewCache(DefaultCacheConfig(), nil)

	req, _ := http.NewRequest("GET", "http://example.com/asset.js", nil)
	c.Put(req, http.StatusOK, http.Header{}, []byte("console.log(1);"))

	cached, _ := c.Get(req)

	// Request with matching If-None-Match
	req304, _ := http.NewRequest("GET", "http://example.com/asset.js", nil)
	req304.Header.Set("If-None-Match", cached.ETag)

	rec := httptest.NewRecorder()
	c.ServeCached(rec, req304, cached)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("Expected 304 Not Modified, got %d", rec.Code)
	}
	if rec.Header().Get("X-ProxyGO-Cache") != "HIT" {
		t.Fatal("Expected X-ProxyGO-Cache: HIT")
	}
}

func TestCache_Purge(t *testing.T) {
	c := NewCache(DefaultCacheConfig(), nil)

	req1, _ := http.NewRequest("GET", "http://example.com/api/v1/users", nil)
	req2, _ := http.NewRequest("GET", "http://example.com/api/v1/posts", nil)

	c.Put(req1, http.StatusOK, http.Header{}, []byte("users"))
	c.Put(req2, http.StatusOK, http.Header{}, []byte("posts"))

	if c.Stats().CachedEntries != 2 {
		t.Fatalf("Expected 2 cached entries, got %d", c.Stats().CachedEntries)
	}

	// Purge posts
	purged := c.Purge("posts")
	if purged != 1 || c.Stats().CachedEntries != 1 {
		t.Fatalf("Expected 1 purged, got %d (remaining: %d)", purged, c.Stats().CachedEntries)
	}

	// Purge all
	c.Purge("*")
	if c.Stats().CachedEntries != 0 {
		t.Fatalf("Expected 0 entries after purge all, got %d", c.Stats().CachedEntries)
	}
}

func TestCache_Expiration(t *testing.T) {
	cfg := CacheConfig{
		Enabled:    true,
		DefaultTTL: 20 * time.Millisecond,
		MaxEntries: 100,
	}
	c := NewCache(cfg, nil)

	req, _ := http.NewRequest("GET", "http://example.com/quick", nil)
	c.Put(req, http.StatusOK, http.Header{}, []byte("fast"))

	_, hit := c.Get(req)
	if !hit {
		t.Fatal("Expected hit before expiration")
	}

	time.Sleep(30 * time.Millisecond)

	_, hit = c.Get(req)
	if hit {
		t.Fatal("Expected miss after expiration")
	}
}

func TestCache_StripSensitiveHeaders(t *testing.T) {
	c := NewCache(DefaultCacheConfig(), nil)

	req, _ := http.NewRequest("GET", "http://example.com/api/profile", nil)

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Set-Cookie":    []string{"session_id=SECRET123; HttpOnly; Secure"},
		"Authorization": []string{"Bearer token123"},
	}
	c.Put(req, http.StatusOK, headers, []byte(`{"user":"alice"}`))

	cached, hit := c.Get(req)
	if !hit {
		t.Fatal("Expected cache hit")
	}

	if _, exists := cached.Headers["Set-Cookie"]; exists {
		t.Fatal("CRITICAL SECURITY VULNERABILITY: Set-Cookie MUST NOT be cached!")
	}
	if _, exists := cached.Headers["Authorization"]; exists {
		t.Fatal("CRITICAL SECURITY VULNERABILITY: Authorization MUST NOT be cached!")
	}
	if cached.Headers["Content-Type"][0] != "application/json" {
		t.Fatal("Expected Content-Type to be preserved")
	}
}

func TestCache_CookieUserIsolation(t *testing.T) {
	c := NewCache(DefaultCacheConfig(), nil)

	// User request with Cookie
	req, _ := http.NewRequest("GET", "http://example.com/dashboard", nil)
	req.Header.Set("Cookie", "session=alice_secret")

	// Backend returns 200 OK without Cache-Control: public
	headers := http.Header{"Content-Type": []string{"text/html"}}
	c.Put(req, http.StatusOK, headers, []byte("<h1>Alice Dashboard</h1>"))

	// Must NOT be cached
	_, hit := c.Get(req)
	if hit {
		t.Fatal("Security violation: authenticated request with Cookie was cached!")
	}
}

func TestCache_LRUEviction(t *testing.T) {
	cfg := CacheConfig{
		Enabled:    true,
		DefaultTTL: time.Hour,
		MaxEntries: 2,
	}
	c := NewCache(cfg, nil)

	r1, _ := http.NewRequest("GET", "http://example.com/1", nil)
	r2, _ := http.NewRequest("GET", "http://example.com/2", nil)
	r3, _ := http.NewRequest("GET", "http://example.com/3", nil)

	c.Put(r1, http.StatusOK, http.Header{}, []byte("1"))
	time.Sleep(5 * time.Millisecond)
	c.Put(r2, http.StatusOK, http.Header{}, []byte("2"))
	time.Sleep(5 * time.Millisecond)

	// Touch r1 so r2 becomes least recently accessed
	_, _ = c.Get(r1)
	time.Sleep(5 * time.Millisecond)

	// Insert r3: should evict r2
	c.Put(r3, http.StatusOK, http.Header{}, []byte("3"))

	_, hit1 := c.Get(r1)
	_, hit2 := c.Get(r2)
	_, hit3 := c.Get(r3)

	if !hit1 {
		t.Fatal("Expected r1 to remain in cache")
	}
	if hit2 {
		t.Fatal("Expected r2 to be evicted via LRU")
	}
	if !hit3 {
		t.Fatal("Expected r3 to be in cache")
	}
}
