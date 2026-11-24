package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/balancer"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/health"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/metrics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/ratelimit"
)

func TestIntegration_ActiveHealthCheck_AutomaticFailover(t *testing.T) {
	var b1Healthy atomic.Bool
	b1Healthy.Store(true)

	var b1Reqs atomic.Int64
	var b2Reqs atomic.Int64

	b1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if !b1Healthy.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		b1Reqs.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("b1"))
	}))
	defer b1.Close()

	b2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		b2Reqs.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("b2"))
	}))
	defer b2.Close()

	u1, _ := url.Parse(b1.URL)
	u2, _ := url.Parse(b2.URL)

	backend1 := proxy.NewBackend(u1, u1.Host, 1)
	backend2 := proxy.NewBackend(u2, u2.Host, 1)

	pool := proxy.NewBackendPool(backend1, backend2)
	bal, _ := balancer.New("round_robin")
	m := metrics.NewMetrics()

	timeouts := config.DefaultConfig().Timeouts
	l7Cfg := config.L7Config{
		Enabled:   true,
		Listen:    "127.0.0.1:0",
		Algorithm: "round_robin",
	}

	l7 := proxy.NewL7Server(l7Cfg, timeouts, pool, bal, nil, m)

	hcCfg := config.HealthCheckConfig{
		Enabled:            true,
		Interval:           50 * time.Millisecond,
		Timeout:            40 * time.Millisecond,
		Path:               "/health",
		ExpectedStatus:     200,
		UnhealthyThreshold: 2,
		HealthyThreshold:   2,
	}

	checker := health.NewChecker(hcCfg, pool)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	checker.Start(ctx)
	defer checker.Stop()

	// Initial check: both healthy
	time.Sleep(120 * time.Millisecond)

	// Route 4 requests: should distribute 2 to b1 and 2 to b2
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		l7.Handler().ServeHTTP(rec, req)
	}

	if b1Reqs.Load() == 0 || b2Reqs.Load() == 0 {
		t.Fatalf("expected both backends to receive requests: b1=%d, b2=%d", b1Reqs.Load(), b2Reqs.Load())
	}

	// Now fail backend 1
	b1Healthy.Store(false)

	// Wait for 2 health checks to trigger unhealthiness
	time.Sleep(150 * time.Millisecond)

	if backend1.IsAlive() {
		t.Fatal("backend1 should be marked unhealthy")
	}

	// Send 6 more requests: 100% must go to backend 2
	b2Before := b2Reqs.Load()
	b1Before := b1Reqs.Load()

	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		l7.Handler().ServeHTTP(rec, req)
		if rec.Body.String() != "b2" {
			t.Fatalf("request %d expected b2, got %s", i, rec.Body.String())
		}
	}

	if b1Reqs.Load() != b1Before {
		t.Fatalf("backend1 received unexpected traffic while dead! before=%d, after=%d", b1Before, b1Reqs.Load())
	}
	if b2Reqs.Load() != b2Before+6 {
		t.Fatalf("backend2 should have received all 6 requests")
	}

	// Restore backend 1
	b1Healthy.Store(true)

	// Wait for 2 consecutive healthy checks
	time.Sleep(160 * time.Millisecond)

	if !backend1.IsAlive() {
		t.Fatal("backend1 should be restored to healthy")
	}
}

func TestIntegration_RateLimiterAndPrometheus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	backend := proxy.NewBackend(u, u.Host, 1)
	pool := proxy.NewBackendPool(backend)
	bal, _ := balancer.New("round_robin")
	m := metrics.NewMetrics()

	rl := ratelimit.NewShardedTokenBucket(config.RateLimitConfig{
		Enabled: true,
		Rate:    2,
		Burst:   2,
	})
	defer rl.Close()

	l7 := proxy.NewL7Server(config.L7Config{}, config.DefaultConfig().Timeouts, pool, bal, rl, m)

	clientIP := "192.168.10.50:4120"

	// 2 requests pass
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = clientIP
		rec := httptest.NewRecorder()
		l7.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d should succeed, got %d", i+1, rec.Code)
		}
	}

	// 3rd request rejected with 429
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = clientIP
	rec := httptest.NewRecorder()
	l7.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", rec.Code)
	}

	// Check /metrics includes the rate limit drop
	mReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mRec := httptest.NewRecorder()
	m.Handler().ServeHTTP(mRec, mReq)

	if !strings.Contains(mRec.Body.String(), "loadbalancer_ratelimit_exceeded_total 1") {
		t.Errorf("metrics missing rate limit exceeded counter")
	}
}
