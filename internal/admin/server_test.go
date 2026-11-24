package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/balancer"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/metrics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
)

func setupTestAdminServer() (*Server, *proxy.L7Server, *proxy.L4Server) {
	cfg := config.DefaultConfig()
	m := metrics.NewMetrics()

	// L7
	u1, _ := url.Parse("http://127.0.0.1:8081")
	b1 := proxy.NewBackend(u1, "127.0.0.1:8081", 1)
	pool7 := proxy.NewBackendPool(b1)
	bal7, _ := balancer.New("round_robin")
	l7 := proxy.NewL7Server(cfg.L7, cfg.Timeouts, pool7, bal7, nil, m)

	// L4
	b2 := proxy.NewBackend(nil, "127.0.0.1:9081", 1)
	pool4 := proxy.NewBackendPool(b2)
	bal4, _ := balancer.New("round_robin")
	l4 := proxy.NewL4Server(cfg.L4, cfg.Timeouts, pool4, bal4, nil, m)

	srv := NewServer(cfg, l7, l4, nil, m)
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
	if !strings.Contains(body, "Reverse Proxy & Load Balancer") {
		t.Fatal("dashboard missing title")
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
