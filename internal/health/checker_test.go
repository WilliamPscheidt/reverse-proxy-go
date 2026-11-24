package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
)

func TestHealthChecker_Transitions(t *testing.T) {
	var serverFail atomic.Bool
	serverFail.Store(false)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serverFail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	backend := proxy.NewBackend(u, server.Listener.Addr().String(), 1)
	pool := proxy.NewBackendPool(backend)

	cfg := config.HealthCheckConfig{
		Enabled:            true,
		Interval:           50 * time.Millisecond,
		Timeout:            50 * time.Millisecond,
		Path:               "/",
		ExpectedStatus:     200,
		UnhealthyThreshold: 2,
		HealthyThreshold:   2,
	}

	checker := NewChecker(cfg, pool)

	var transitions atomic.Int32
	checker.SetStatusListener(func(b *proxy.Backend, healthy bool) {
		transitions.Add(1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	checker.Start(ctx)
	defer checker.Stop()

	// Initial check: should remain healthy
	time.Sleep(120 * time.Millisecond)
	if !backend.IsAlive() {
		t.Fatal("backend should be alive initially")
	}

	// Trigger server failures
	serverFail.Store(true)

	// Wait for UnhealthyThreshold checks (2 * 50ms = 100ms + buffer)
	time.Sleep(200 * time.Millisecond)
	if backend.IsAlive() {
		t.Fatal("backend should be marked unhealthy after consecutive failures")
	}

	// Recover server
	serverFail.Store(false)

	// Wait for HealthyThreshold checks
	time.Sleep(200 * time.Millisecond)
	if !backend.IsAlive() {
		t.Fatal("backend should be restored to healthy after consecutive successes")
	}

	if transitions.Load() < 2 {
		t.Errorf("expected at least 2 state transitions, got %d", transitions.Load())
	}
}
