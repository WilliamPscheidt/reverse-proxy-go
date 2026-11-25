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

func TestRouter_SubdomainAndWildcardHost(t *testing.T) {
	defaultPool := "default-pool"
	subPool := "sub-pool"
	wildPool := "wild-pool"

	router := NewRouter(defaultPool)
	// Route 1: Exact subdomain
	router.AddRoute(RouteConfig{
		ID:         "exact-sub",
		Host:       "subdominio.blabla.com",
		PathPrefix: "/",
	}, subPool, nil)

	// Route 2: Wildcard subdomain
	router.AddRoute(RouteConfig{
		ID:         "wild-sub",
		Host:       "*.blabla.com",
		PathPrefix: "/",
	}, wildPool, nil)

	// 1. Exact subdomain match
	req1, _ := http.NewRequest("GET", "http://subdominio.blabla.com/status", nil)
	route1, pool1, _, matched1 := router.Match(req1)
	if !matched1 || route1.Config.ID != "exact-sub" || pool1 != subPool {
		t.Fatalf("Expected exact-sub match, got id=%v pool=%v", route1.Config.ID, pool1)
	}

	// 2. Wildcard subdomain match (e.g. app.blabla.com)
	req2, _ := http.NewRequest("GET", "http://app.blabla.com/status", nil)
	route2, pool2, _, matched2 := router.Match(req2)
	if !matched2 || route2.Config.ID != "wild-sub" || pool2 != wildPool {
		t.Fatalf("Expected wild-sub match, got id=%v pool=%v", route2.Config.ID, pool2)
	}

	// 3. Request with port (e.g. subdominio.blabla.com:8080)
	req3, _ := http.NewRequest("GET", "http://subdominio.blabla.com:8080/dashboard", nil)
	route3, pool3, _, matched3 := router.Match(req3)
	if !matched3 || route3.Config.ID != "exact-sub" || pool3 != subPool {
		t.Fatalf("Expected exact-sub match with port, got id=%v pool=%v", route3.Config.ID, pool3)
	}

	// 4. Different domain falls back to default
	req4, _ := http.NewRequest("GET", "http://otherdomain.com/status", nil)
	_, pool4, _, matched4 := router.Match(req4)
	if matched4 || pool4 != defaultPool {
		t.Fatalf("Expected fallback to defaultPool, got matched=%v pool=%v", matched4, pool4)
	}
}

func TestRouter_PathRewriteAndStripPrefix(t *testing.T) {
	defaultPool := "default-pool"
	rewritePool := "rewrite-pool"
	stripPool := "strip-pool"

	router := NewRouter(defaultPool)

	// Route with rewrite: /teste -> /sitenovo
	router.AddRoute(RouteConfig{
		ID:            "rewrite-route",
		Host:          "*",
		PathPrefix:    "/teste",
		RewritePrefix: "/sitenovo",
	}, rewritePool, nil)

	// Route with strip prefix: /api/v1 -> /
	router.AddRoute(RouteConfig{
		ID:          "strip-route",
		Host:        "*",
		PathPrefix:  "/api/v1",
		StripPrefix: true,
	}, stripPool, nil)

	// 1. Test /teste exact rewrite
	req1, _ := http.NewRequest("GET", "http://localhost:8080/teste", nil)
	route1, pool1, _, matched1 := router.Match(req1)
	if !matched1 || pool1 != rewritePool {
		t.Fatalf("Expected rewrite-route match, got matched=%v pool=%v", matched1, pool1)
	}
	route1.ApplyRewrite(req1)
	if req1.URL.Path != "/sitenovo" {
		t.Fatalf("Expected path /sitenovo, got %s", req1.URL.Path)
	}

	// 2. Test /teste/users/profile rewrite
	req2, _ := http.NewRequest("GET", "http://localhost:8080/teste/users/profile", nil)
	route2, _, _, _ := router.Match(req2)
	route2.ApplyRewrite(req2)
	if req2.URL.Path != "/sitenovo/users/profile" {
		t.Fatalf("Expected path /sitenovo/users/profile, got %s", req2.URL.Path)
	}

	// 3. Test /api/v1/auth/login strip prefix
	req3, _ := http.NewRequest("GET", "http://localhost:8080/api/v1/auth/login", nil)
	route3, pool3, _, matched3 := router.Match(req3)
	if !matched3 || pool3 != stripPool {
		t.Fatalf("Expected strip-route match, got matched=%v pool=%v", matched3, pool3)
	}
	route3.ApplyRewrite(req3)
	if req3.URL.Path != "/auth/login" {
		t.Fatalf("Expected path /auth/login, got %s", req3.URL.Path)
	}
}

func TestRouter_PathPrefixBoundary(t *testing.T) {
	router := NewRouter("default-pool")
	router.AddRoute(RouteConfig{
		ID:         "teste-route",
		PathPrefix: "/teste",
	}, "test-pool", nil)

	// /teste matches
	req1, _ := http.NewRequest("GET", "http://localhost:8080/teste", nil)
	_, _, _, matched1 := router.Match(req1)
	if !matched1 {
		t.Fatal("Expected /teste to match /teste route")
	}

	// /teste/123 matches
	req2, _ := http.NewRequest("GET", "http://localhost:8080/teste/123", nil)
	_, _, _, matched2 := router.Match(req2)
	if !matched2 {
		t.Fatal("Expected /teste/123 to match /teste route")
	}

	// /testeXYZ MUST NOT match /teste route!
	req3, _ := http.NewRequest("GET", "http://localhost:8080/testeXYZ", nil)
	_, _, _, matched3 := router.Match(req3)
	if matched3 {
		t.Fatal("Boundary vulnerability: /testeXYZ incorrectly matched /teste route!")
	}
}
