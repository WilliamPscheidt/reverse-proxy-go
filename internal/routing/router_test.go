package routing

import (
	"net/http"
	"net/url"
	"testing"
)

func TestRouter_PathMatching(t *testing.T) {
	defaultPool := "default-pool"
	apiPool := "api-pool"
	authPool := "auth-pool"

	router := NewRouter(defaultPool)
	router.AddRoute(RouteConfig{
		ID:         "api-route",
		PathPrefix: "/api",
	}, apiPool, nil)

	router.AddRoute(RouteConfig{
		ID:         "auth-route",
		PathPrefix: "/api/auth",
	}, authPool, nil)

	// Test exact specific prefix /api/auth
	req1, _ := http.NewRequest("GET", "http://example.com/api/auth/login", nil)
	route, pool, isCanary, matched := router.Match(req1)
	if !matched || route.Config.ID != "auth-route" || pool != authPool || isCanary {
		t.Fatalf("Expected match to auth-route, got id=%v", route.Config.ID)
	}

	// Test general prefix /api
	req2, _ := http.NewRequest("GET", "http://example.com/api/users", nil)
	route, pool, _, matched = router.Match(req2)
	if !matched || route.Config.ID != "api-route" || pool != apiPool {
		t.Fatalf("Expected match to api-route, got id=%v", route.Config.ID)
	}

	// Test fallback to default
	req3, _ := http.NewRequest("GET", "http://example.com/index.html", nil)
	_, pool, _, matched = router.Match(req3)
	if matched || pool != defaultPool {
		t.Fatalf("Expected fallback to defaultPool, got matched=%v", matched)
	}
}

func TestRouter_HostMatching(t *testing.T) {
	defaultPool := "default-pool"
	apiPool := "api-pool"

	router := NewRouter(defaultPool)
	router.AddRoute(RouteConfig{
		ID:         "vhost-api",
		Host:       "api.mycloud.io",
		PathPrefix: "/",
	}, apiPool, nil)

	// Request with matching Host
	req1, _ := http.NewRequest("GET", "http://api.mycloud.io/status", nil)
	route, pool, _, matched := router.Match(req1)
	if !matched || route.Config.ID != "vhost-api" || pool != apiPool {
		t.Fatal("Expected match to vhost-api")
	}

	// Request with different Host
	req2, _ := http.NewRequest("GET", "http://web.mycloud.io/status", nil)
	_, pool, _, matched = router.Match(req2)
	if matched || pool != defaultPool {
		t.Fatal("Expected fallback to defaultPool for non-matching host")
	}
}

func TestRouter_CanaryTrafficSplit(t *testing.T) {
	primaryPool := "primary-pool"
	canaryPool := "canary-pool"

	router := NewRouter(primaryPool)
	router.AddRoute(RouteConfig{
		ID:           "canary-test",
		PathPrefix:   "/feature",
		CanaryWeight: 50, // 50% canary
	}, primaryPool, canaryPool)

	canaryCount := 0
	total := 1000

	for i := 0; i < total; i++ {
		req := &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/feature/checkout"},
			Host:   "example.com",
		}
		_, pool, isCanary, matched := router.Match(req)
		if !matched {
			t.Fatal("Expected match")
		}
		if isCanary && pool == canaryPool {
			canaryCount++
		}
	}

	// Expect roughly 50% (+/- 10%)
	ratio := float64(canaryCount) / float64(total)
	if ratio < 0.35 || ratio > 0.65 {
		t.Fatalf("Expected canary ratio near 0.50, got %.2f (%d/%d)", ratio, canaryCount, total)
	}
}
