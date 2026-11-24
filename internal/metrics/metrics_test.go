package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetrics_ExposeAndRecord(t *testing.T) {
	m := NewMetrics()

	backend := "127.0.0.1:8081"
	m.RecordHTTPRequest(backend, "GET", 200, 2*time.Millisecond)
	m.IncHTTPActiveConn(backend)
	m.SetBackendHealth(backend, true)
	m.RecordRateLimitExceeded()
	m.RecordTCPConn(backend)
	m.RecordTCPBytes(backend, "inbound", 1024)
	m.CloseTCPConn(backend, 50*time.Millisecond)
	m.DecHTTPActiveConn(backend)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	requiredSubstrings := []string{
		"loadbalancer_http_requests_total",
		"loadbalancer_http_request_duration_seconds",
		"loadbalancer_backend_health_status",
		"loadbalancer_ratelimit_exceeded_total",
		"loadbalancer_tcp_connections_total",
	}

	for _, sub := range requiredSubstrings {
		if !strings.Contains(body, sub) {
			t.Errorf("missing metric %q in /metrics output", sub)
		}
	}
}
