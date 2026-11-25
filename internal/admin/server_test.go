package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/analytics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/balancer"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/cache"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/metrics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/resilience"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/routing"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/security"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/stream"
)

func setupTestAdminServer() (*Server, *proxy.L7Server, *proxy.L4Server) {
	cfg := config.DefaultConfig()
	m := metrics.NewMetrics()
	a := analytics.NewAnalytics()
	bl := security.NewBlacklist(nil)
	a.SetBlacklistChecker(bl.IsBlocked)

	// L7
	u1, _ := url.Parse("http://127.0.0.1:8081")
	b1 := proxy.NewBackend(u1, "127.0.0.1:8081", 1)
	pool7 := proxy.NewBackendPool(b1)
	bal7, _ := balancer.New("round_robin")
	l7 := proxy.NewL7Server(cfg.L7, cfg.Timeouts, pool7, bal7, nil, m)
	l7.SetAnalytics(a)
	l7.SetBlacklist(bl)

	waf := security.NewWAF(security.DefaultWAFConfig())
	c := cache.NewCache(cache.DefaultCacheConfig(), nil)
	r := routing.NewRouter(pool7)
	cr := resilience.NewCircuitRegistry(resilience.DefaultCircuitConfig())
	rp := resilience.NewRetryPolicy(resilience.DefaultRetryConfig())
	sb := stream.NewStreamBroker(10)

	l7.SetWAF(waf)
	l7.SetCache(c)
	l7.SetRouter(r)
	l7.SetResilience(cr, rp)
	l7.SetStreamBroker(sb)

	// L4
	b2 := proxy.NewBackend(nil, "127.0.0.1:9081", 1)
	pool4 := proxy.NewBackendPool(b2)
	bal4, _ := balancer.New("round_robin")
	l4 := proxy.NewL4Server(cfg.L4, cfg.Timeouts, pool4, bal4, nil, m)
	l4.SetBlacklist(bl)

	srv := NewServer(cfg, l7, l4, nil, m, a, bl)
	return srv, l7, l4
}

func TestAdmin_DashboardHTML(t *testing.T) {
	srv, _, _ := setupTestAdminServer()

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ProxyGO") {
		t.Fatal("dashboard missing ProxyGO title")
	}
}

func TestAdmin_APIStatus(t *testing.T) {
	srv, _, _ := setupTestAdminServer()

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if resp.L7 == nil || resp.L7.TotalNodes != 1 {
		t.Fatalf("expected 1 L7 node, got %+v", resp.L7)
	}
}

func TestAdmin_BackendCRUDAndToggle(t *testing.T) {
	srv, l7, _ := setupTestAdminServer()

	// 1. Add Backend
	payload := map[string]any{
		"type":   "l7",
		"target": "http://127.0.0.1:8089",
		"weight": 2,
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/backends", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	if l7.GetPool().BackendCount() != 2 {
		t.Fatalf("expected 2 backends in L7 pool, got %d", l7.GetPool().BackendCount())
	}

	// 2. Toggle Backend Status
	togglePayload := map[string]any{
		"type":   "l7",
		"target": "http://127.0.0.1:8089",
		"alive":  false,
	}
	tData, _ := json.Marshal(togglePayload)
	tReq := httptest.NewRequest(http.MethodPost, "/api/backends/toggle", bytes.NewReader(tData))
	tRec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(tRec, tReq)

	if tRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", tRec.Code)
	}

	b := l7.GetPool().FindByAddr("http://127.0.0.1:8089")
	if b == nil || b.IsAlive() {
		t.Fatal("expected backend to be marked unalive")
	}

	// 3. Delete Backend
	delPayload := map[string]any{
		"type":   "l7",
		"target": "http://127.0.0.1:8089",
	}
	dData, _ := json.Marshal(delPayload)
	dReq := httptest.NewRequest(http.MethodDelete, "/api/backends", bytes.NewReader(dData))
	dRec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(dRec, dReq)

	if dRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", dRec.Code)
	}
	if l7.GetPool().BackendCount() != 1 {
		t.Fatalf("expected 1 backend after deletion, got %d", l7.GetPool().BackendCount())
	}
}

func TestAdmin_AlgorithmHotSwap(t *testing.T) {
	srv, l7, _ := setupTestAdminServer()

	payload := map[string]string{
		"type":      "l7",
		"algorithm": "least_conn",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/algorithm", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	if l7.GetBalancer().Name() != "least_conn" {
		t.Fatalf("expected least_conn, got %s", l7.GetBalancer().Name())
	}
}

func TestAdmin_Analytics(t *testing.T) {
	srv, l7, _ := setupTestAdminServer()

	// Simulate requests
	l7.GetAnalytics().RecordRequest("127.0.0.1", 200)
	l7.GetAnalytics().RecordRequest("203.0.113.1", 429)

	req := httptest.NewRequest(http.MethodGet, "/api/analytics", nil)
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var snap analytics.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if snap.LastMinuteRequests != 2 {
		t.Fatalf("expected 2 requests in last minute, got %d", snap.LastMinuteRequests)
	}
	if snap.Total429 != 1 {
		t.Fatalf("expected 1 429 block, got %d", snap.Total429)
	}
}

func TestAdmin_Settings(t *testing.T) {
	srv, _, _ := setupTestAdminServer()

	payload := map[string]any{
		"mode": "l7",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/settings", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestAdmin_Blacklist(t *testing.T) {
	srv, _, _ := setupTestAdminServer()

	// 1. Block IP
	payload := map[string]string{
		"ip":     "198.51.100.22",
		"reason": "Test block",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/blacklist", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rec.Code)
	}

	// 2. List blacklist
	req = httptest.NewRequest(http.MethodGet, "/api/blacklist", nil)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var list []security.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if len(list) != 1 || list[0].IP != "198.51.100.22" {
		t.Fatalf("expected blacklisted IP in list, got %+v", list)
	}

	// 3. Unblock IP
	unblockPayload := map[string]string{"ip": "198.51.100.22"}
	uData, _ := json.Marshal(unblockPayload)
	req = httptest.NewRequest(http.MethodDelete, "/api/blacklist", bytes.NewReader(uData))
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestAdmin_WAF_Cache_Routes_APIs(t *testing.T) {
	srv, _, _ := setupTestAdminServer()

	// 1. Test WAF GET
	req := httptest.NewRequest(http.MethodGet, "/api/waf", nil)
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/waf, got %d", rec.Code)
	}

	// 2. Test Cache Stats and Purge
	req = httptest.NewRequest(http.MethodGet, "/api/cache", nil)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/cache, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/cache/purge?pattern=*", nil)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/cache/purge, got %d", rec.Code)
	}

	// 3. Test Routes API
	req = httptest.NewRequest(http.MethodGet, "/api/routes", nil)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/routes, got %d", rec.Code)
	}

	// 4. Test Circuit Breakers API
	req = httptest.NewRequest(http.MethodGet, "/api/circuit-breakers", nil)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/circuit-breakers, got %d", rec.Code)
	}

	// 5. Test Stream History API
	req = httptest.NewRequest(http.MethodGet, "/api/stream/history", nil)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/stream/history, got %d", rec.Code)
	}
}

func TestAdmin_AuthAndSessionFlow(t *testing.T) {
	srv, _, _ := setupTestAdminServer()
	srv.cfg.Admin.AuthEnabled = true
	srv.cfg.Admin.Username = "admin"
	srv.cfg.Admin.Password = "admin123"

	// 1. Unauthenticated request to / should redirect to /login
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 Found, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("expected redirect to /login, got %s", loc)
	}

	// 2. Unauthenticated request to /api/status should return 401
	req = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for /api/status, got %d", rec.Code)
	}

	// 3. GET /login should return 200 OK with login HTML
	req = httptest.NewRequest(http.MethodGet, "/login", nil)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /login, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ProxyGO Enterprise Console") {
		t.Fatal("login page missing expected title")
	}

	// 4. Invalid POST to /api/login should fail
	invalidBody := `{"username":"admin","password":"wrongpassword"}`
	req = httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(invalidBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for bad login, got %d", rec.Code)
	}

	// 5. Valid POST to /api/login should succeed and set proxygo_session cookie
	validBody := `{"username":"admin","password":"admin123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(validBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid login, got %d: %s", rec.Code, rec.Body.String())
	}

	var sessionCookie *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "proxygo_session" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("expected proxygo_session cookie to be set")
	}

	// 6. Access / with session cookie -> 200 OK
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for authenticated dashboard request, got %d", rec.Code)
	}

	// 7. Check /api/me with session cookie
	req = httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/me, got %d", rec.Code)
	}
	var meResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &meResp)
	if meResp["username"] != "admin" {
		t.Fatalf("expected username admin in /api/me, got %v", meResp["username"])
	}

	// 8. Logout via /api/logout
	req = httptest.NewRequest(http.MethodPost, "/api/logout", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/logout, got %d", rec.Code)
	}

	// 9. After logout, session is invalid -> GET / redirects to /login
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 Found after logout, got %d", rec.Code)
	}
}
