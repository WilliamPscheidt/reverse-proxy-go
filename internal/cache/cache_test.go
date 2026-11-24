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
