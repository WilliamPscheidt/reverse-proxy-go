package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/metrics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/ratelimit"
)

type mockBalancer struct {
	b *Backend
}

func (m *mockBalancer) Name() string { return "mock" }
func (m *mockBalancer) NextHTTP(pool *BackendPool, req *http.Request, clientIP string) (*Backend, error) {
	return m.b, nil
}
func (m *mockBalancer) NextTCP(pool *BackendPool, clientRemoteAddr string) (*Backend, error) {
	return m.b, nil
}

func TestL7Server_EndToEnd(t *testing.T) {
	var receivedXFF, receivedHost string

	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedXFF = r.Header.Get("X-Forwarded-For")
		receivedHost = r.Header.Get("X-Forwarded-Host")
		w.Header().Set("X-Custom-Header", "proxy-test")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("backend response"))
	}))
	defer backendServer.Close()

	u, _ := url.Parse(backendServer.URL)
	backend := NewBackend(u, u.Host, 1)
	pool := NewBackendPool(backend)
	bal := &mockBalancer{b: backend}
	m := metrics.NewMetrics()

	cfg := config.L7Config{
		Enabled:   true,
		Listen:    "127.0.0.1:0", // random port
		Algorithm: "round_robin",
	}
	timeouts := config.DefaultConfig().Timeouts

	s := NewL7Server(cfg, timeouts, pool, bal, nil, m)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req.RemoteAddr = "203.0.113.195:45230"
	rec := httptest.NewRecorder()

	s.handleHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "backend response" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if receivedXFF != "203.0.113.195" {
		t.Fatalf("expected XFF 203.0.113.195, got %s", receivedXFF)
	}
	if receivedHost != "example.com" {
		t.Fatalf("expected X-Forwarded-Host example.com, got %s", receivedHost)
	}
}

func TestL7Server_RateLimit(t *testing.T) {
	u, _ := url.Parse("http://127.0.0.1:8080")
	backend := NewBackend(u, "127.0.0.1:8080", 1)
	pool := NewBackendPool(backend)
	bal := &mockBalancer{b: backend}

	rl := ratelimit.NewShardedTokenBucket(config.RateLimitConfig{
		Enabled: true,
		Rate:    1,
		Burst:   1,
	})
	defer rl.Close()

	s := NewL7Server(config.L7Config{}, config.DefaultConfig().Timeouts, pool, bal, rl, metrics.NewMetrics())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"

	// 1st request passes
	rec1 := httptest.NewRecorder()
	s.handleHTTP(rec1, req)
	// (backend will fail connect since not running, but status won't be 429)
	if rec1.Code == http.StatusTooManyRequests {
		t.Fatal("first request should not be rate limited")
	}

	// 2nd request immediate -> 429
	rec2 := httptest.NewRecorder()
	s.handleHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 Too Many Requests, got %d", rec2.Code)
	}
}

func TestL4Server_EndToEnd(t *testing.T) {
	// 1. Setup mock TCP echo backend
	backendLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on TCP backend: %v", err)
	}
	defer backendLn.Close()

	go func() {
		for {
			conn, err := backendLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c) // Echo back
			}(conn)
		}
	}()

	backendAddr := backendLn.Addr().String()
	backend := NewBackend(nil, backendAddr, 1)
	pool := NewBackendPool(backend)
	bal := &mockBalancer{b: backend}
	m := metrics.NewMetrics()

	// 2. Setup L4 Server on dynamic port
	proxyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen for L4 proxy: %v", err)
	}
	proxyAddr := proxyLn.Addr().String()
	_ = proxyLn.Close() // Release so L4 server can bind

	l4Cfg := config.L4Config{
		Enabled:   true,
		Listen:    proxyAddr,
		Algorithm: "round_robin",
	}
	timeouts := config.DefaultConfig().Timeouts
	l4 := NewL4Server(l4Cfg, timeouts, pool, bal, nil, m)

	go func() {
		_ = l4.Start()
	}()
	time.Sleep(50 * time.Millisecond)
	defer func() {
		_ = l4.Shutdown(context.Background())
	}()

	// 3. Connect client to L4 proxy
	client, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("client failed to dial proxy: %v", err)
	}
	defer client.Close()

	payload := "Hello TCP Load Balancer!"
	_, err = fmt.Fprint(client, payload)
	if err != nil {
		t.Fatalf("client write error: %v", err)
	}

	buf := make([]byte, len(payload))
	_, err = io.ReadFull(client, buf)
	if err != nil {
		t.Fatalf("client read error: %v", err)
	}

	if string(buf) != payload {
		t.Fatalf("expected echo %q, got %q", payload, string(buf))
	}
}
