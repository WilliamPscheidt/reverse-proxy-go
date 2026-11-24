package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/metrics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/ratelimit"
)

// BalancerInterface abstracts load balancing algorithms for L7.
type BalancerInterface interface {
	Name() string
	NextHTTP(pool *BackendPool, req *http.Request, clientIP string) (*Backend, error)
}

// L7Server represents the HTTP reverse proxy and load balancer server.
type L7Server struct {
	config      config.L7Config
	pool        *BackendPool
	balancer    BalancerInterface
	rateLimiter *ratelimit.ShardedTokenBucket
	metrics     *metrics.Metrics
	transport   *http.Transport
	httpServer  *http.Server
}

// NewL7Server creates and initializes an L7 HTTP Reverse Proxy server.
func NewL7Server(
	cfg config.L7Config,
	timeouts config.TimeoutsConfig,
	pool *BackendPool,
	bal BalancerInterface,
	rl *ratelimit.ShardedTokenBucket,
	m *metrics.Metrics,
) *L7Server {
	// High performance HTTP transport optimized for tens of thousands of requests per second
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeouts.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          timeouts.MaxIdleConns,
		MaxIdleConnsPerHost:   timeouts.MaxIdleConnsHost,
		IdleConnTimeout:       timeouts.IdleTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: timeouts.ReadTimeout,
	}

	// Initialize ReverseProxy on each backend with shared transport
	for _, b := range pool.GetBackends() {
		if b.URL != nil {
			targetURL := b.URL
			rp := &httputil.ReverseProxy{
				Rewrite: func(pr *httputil.ProxyRequest) {
					pr.SetURL(targetURL)
					pr.SetXForwarded()
				},
				Transport: transport,
				ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
					log.Printf("[L7] Backend %s error: %v", targetURL.Host, err)
					http.Error(w, fmt.Sprintf("Bad Gateway: upstream %s unreachable", targetURL.Host), http.StatusBadGateway)
				},
			}
			b.ReverseProxy = rp
		}
	}

	s := &L7Server{
		config:      cfg,
		pool:        pool,
		balancer:    bal,
		rateLimiter: rl,
		metrics:     m,
		transport:   transport,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHTTP)

	s.httpServer = &http.Server{
		Addr:         cfg.Listen,
		Handler:      mux,
		ReadTimeout:  timeouts.ReadTimeout,
		WriteTimeout: timeouts.WriteTimeout,
		IdleTimeout:  timeouts.IdleTimeout,
	}

	return s
}

// Start begins listening and serving HTTP traffic.
func (s *L7Server) Start() error {
	log.Printf("[L7] HTTP Reverse Proxy listening on %s (algorithm: %s)", s.config.Listen, s.balancer.Name())
	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("l7 http server error: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the L7 HTTP server.
func (s *L7Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// statusWriter wraps http.ResponseWriter to intercept response status code and size.
type statusWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}

func (s *L7Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	clientIP := getClientIP(r)

	// 1. Rate Limiting Check
	if s.rateLimiter != nil && !s.rateLimiter.Allow(clientIP) {
		if s.metrics != nil {
			s.metrics.RecordRateLimitExceeded()
			s.metrics.RecordHTTPRequest("ratelimit", r.Method, http.StatusTooManyRequests, time.Since(start))
		}
		w.Header().Set("Retry-After", "1")
		http.Error(w, "Too Many Requests - Rate Limit Exceeded", http.StatusTooManyRequests)
		return
	}

	// 2. Load Balancing Selection
	backend, err := s.balancer.NextHTTP(s.pool, r, clientIP)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordHTTPRequest("none", r.Method, http.StatusServiceUnavailable, time.Since(start))
		}
		http.Error(w, "Service Unavailable - No healthy backends", http.StatusServiceUnavailable)
		return
	}

	// Track active connections
	backend.IncConns()
	backendHost := backend.Addr
	if backend.URL != nil {
		backendHost = backend.URL.Host
	}

	if s.metrics != nil {
		s.metrics.IncHTTPActiveConn(backendHost)
	}

	defer func() {
		backend.DecConns()
		if s.metrics != nil {
			s.metrics.DecHTTPActiveConn(backendHost)
		}
	}()

	// 3. Wrap ResponseWriter to record status and latency
	sw := &statusWriter{ResponseWriter: w}

	if backend.ReverseProxy != nil {
		backend.ReverseProxy.ServeHTTP(sw, r)
	} else if backend.URL != nil {
		// Fallback proxy creation if dynamic
		targetURL := backend.URL
		rp := &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(targetURL)
				pr.SetXForwarded()
			},
			Transport: s.transport,
		}
		rp.ServeHTTP(sw, r)
	} else {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		sw.statusCode = http.StatusBadGateway
	}

	// 5. Record Prometheus Metrics
	if s.metrics != nil {
		code := sw.statusCode
		if code == 0 {
			code = http.StatusOK
		}
		duration := time.Since(start)
		s.metrics.RecordHTTPRequest(backendHost, r.Method, code, duration)
	}
}

// setProxyHeaders sets or appends standard proxy forwarding headers.
func setProxyHeaders(req *http.Request, clientIP string) {
	if prior := req.Header.Get("X-Forwarded-For"); prior != "" {
		req.Header.Set("X-Forwarded-For", prior+", "+clientIP)
	} else {
		req.Header.Set("X-Forwarded-For", clientIP)
	}

	if req.Header.Get("X-Real-IP") == "" {
		req.Header.Set("X-Real-IP", clientIP)
	}

	if req.Header.Get("X-Forwarded-Proto") == "" {
		if req.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}
	}

	if req.Header.Get("X-Forwarded-Host") == "" {
		req.Header.Set("X-Forwarded-Host", req.Host)
	}
}

// getClientIP extracts IP from X-Forwarded-For, X-Real-IP, or RemoteAddr.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// Helper to convert string address to URL
func ParseURL(raw string) *url.URL {
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	return u
}
