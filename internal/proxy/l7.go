package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/analytics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/cache"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/metrics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/ratelimit"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/resilience"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/routing"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/security"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/stream"
)

// BalancerInterface abstracts load balancing algorithms for L7.
type BalancerInterface interface {
	Name() string
	NextHTTP(pool *BackendPool, req *http.Request, clientIP string) (*Backend, error)
}

type l7ContextKey string

const (
	preserveHostKey l7ContextKey = "proxygo_preserve_host"
)

// L7Server represents the HTTP reverse proxy and load balancer server.
type L7Server struct {
	mu              sync.RWMutex
	config          config.L7Config
	pool            *BackendPool
	canaryPool      *BackendPool
	balancer        BalancerInterface
	rateLimiter     ratelimit.Limiter
	metrics         *metrics.Metrics
	analytics       *analytics.Analytics
	blacklist       *security.Blacklist
	waf             *security.WAF
	router          *routing.Router
	circuitRegistry *resilience.CircuitRegistry
	retryPolicy     *resilience.RetryPolicy
	cache           *cache.Cache
	streamBroker    *stream.StreamBroker
	transport       *http.Transport
	httpServer      *http.Server
}

// NewL7Server creates and initializes an L7 HTTP Reverse Proxy server.
func NewL7Server(
	cfg config.L7Config,
	timeouts config.TimeoutsConfig,
	pool *BackendPool,
	bal BalancerInterface,
	rl ratelimit.Limiter,
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
					preserve, _ := pr.In.Context().Value(preserveHostKey).(bool)
					if !preserve {
						pr.Out.Host = targetURL.Host
					}
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

// Handler returns the root http.Handler configured for the L7 server.
func (s *L7Server) Handler() http.Handler {
	return s.httpServer.Handler
}

// GetPool returns the backend pool for L7.
func (s *L7Server) GetPool() *BackendPool {
	return s.pool
}

// SetCanaryPool configures a dedicated upstream pool for canary traffic.
func (s *L7Server) SetCanaryPool(cp *BackendPool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.canaryPool = cp
	s.InitPoolReverseProxies(cp)
}

// GetCanaryPool returns the canary upstream pool.
func (s *L7Server) GetCanaryPool() *BackendPool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.canaryPool
}

// InitPoolReverseProxies initializes reverse proxy instances for backends in a pool.
func (s *L7Server) InitPoolReverseProxies(p *BackendPool) {
	if p == nil {
		return
	}
	for _, b := range p.GetBackends() {
		if b.URL != nil && b.ReverseProxy == nil {
			targetURL := b.URL
			rp := &httputil.ReverseProxy{
				Rewrite: func(pr *httputil.ProxyRequest) {
					pr.SetURL(targetURL)
					pr.SetXForwarded()
					preserve, _ := pr.In.Context().Value(preserveHostKey).(bool)
					if !preserve {
						pr.Out.Host = targetURL.Host
					}
				},
				Transport: s.transport,
				ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
					log.Printf("[L7] Backend %s error: %v", targetURL.Host, err)
					http.Error(w, fmt.Sprintf("Bad Gateway: upstream %s unreachable", targetURL.Host), http.StatusBadGateway)
				},
			}
			b.ReverseProxy = rp
		}
	}
}

// GetBalancer returns the active load balancer.
func (s *L7Server) GetBalancer() BalancerInterface {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.balancer
}

// SetAnalytics registers the analytics collector.
func (s *L7Server) SetAnalytics(a *analytics.Analytics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.analytics = a
}

// GetAnalytics returns the registered analytics collector.
func (s *L7Server) GetAnalytics() *analytics.Analytics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.analytics
}

// SetBlacklist registers the security blacklist.
func (s *L7Server) SetBlacklist(bl *security.Blacklist) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blacklist = bl
}

// GetBlacklist returns the registered security blacklist.
func (s *L7Server) GetBlacklist() *security.Blacklist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.blacklist
}

// SetWAF registers the Web Application Firewall.
func (s *L7Server) SetWAF(w *security.WAF) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waf = w
}

// GetWAF returns the WAF engine.
func (s *L7Server) GetWAF() *security.WAF {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.waf
}

// SetRouter registers the L7 router and canary manager.
func (s *L7Server) SetRouter(r *routing.Router) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.router = r
}

// GetRouter returns the L7 router.
func (s *L7Server) GetRouter() *routing.Router {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.router
}

// SetResilience registers the circuit breaker registry and retry policy.
func (s *L7Server) SetResilience(cr *resilience.CircuitRegistry, rp *resilience.RetryPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.circuitRegistry = cr
	s.retryPolicy = rp
}

// GetCircuitRegistry returns the circuit breaker registry.
func (s *L7Server) GetCircuitRegistry() *resilience.CircuitRegistry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.circuitRegistry
}

// SetCache registers the HTTP edge response cache.
func (s *L7Server) SetCache(c *cache.Cache) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = c
}

// GetCache returns the edge cache.
func (s *L7Server) GetCache() *cache.Cache {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache
}

// SetStreamBroker registers the live request log stream broker.
func (s *L7Server) SetStreamBroker(sb *stream.StreamBroker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streamBroker = sb
}

// GetStreamBroker returns the stream broker.
func (s *L7Server) GetStreamBroker() *stream.StreamBroker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.streamBroker
}

// SetBalancer dynamically updates the load balancing algorithm.
func (s *L7Server) SetBalancer(bal BalancerInterface) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balancer = bal
}

// AddBackendTarget dynamically creates and adds a new upstream backend to L7.
func (s *L7Server) AddBackendTarget(rawURL string, weight int) (*Backend, error) {
	u := ParseURL(rawURL)
	if u == nil {
		return nil, fmt.Errorf("invalid URL: %s", rawURL)
	}
	backend := NewBackend(u, u.Host, weight)
	targetURL := u
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(targetURL)
			pr.SetXForwarded()
			preserve, _ := pr.In.Context().Value(preserveHostKey).(bool)
			if !preserve {
				pr.Out.Host = targetURL.Host
			}
		},
		Transport: s.transport,
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			log.Printf("[L7] Backend %s error: %v", targetURL.Host, err)
			http.Error(w, fmt.Sprintf("Bad Gateway: upstream %s unreachable", targetURL.Host), http.StatusBadGateway)
		},
	}
	backend.ReverseProxy = rp
	s.pool.AddBackend(backend)
	if s.metrics != nil {
		s.metrics.SetBackendHealth(u.Host, true)
	}
	return backend, nil
}

// RemoveBackendTarget removes a backend from the pool by address or URL.
func (s *L7Server) RemoveBackendTarget(addr string) bool {
	return s.pool.RemoveBackend(addr)
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

// bufferedRecorder intercepts headers and body in memory before committing to w.
type bufferedRecorder struct {
	header http.Header
	code   int
	body   bytes.Buffer
}

func newBufferedRecorder() *bufferedRecorder {
	return &bufferedRecorder{header: make(http.Header)}
}

func (b *bufferedRecorder) Header() http.Header {
	return b.header
}

func (b *bufferedRecorder) WriteHeader(code int) {
	b.code = code
}

func (b *bufferedRecorder) Write(p []byte) (int, error) {
	if b.code == 0 {
		b.code = http.StatusOK
	}
	return b.body.Write(p)
}

func (b *bufferedRecorder) FlushTo(w http.ResponseWriter) {
	for k, vv := range b.header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	code := b.code
	if code == 0 {
		code = http.StatusOK
	}
	w.WriteHeader(code)
	_, _ = w.Write(b.body.Bytes())
}

func (s *L7Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	clientIP := getClientIP(r)
	country := "US"
	if s.analytics != nil {
		country = s.analytics.ResolveCountry(clientIP)
	}

	reqID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	w.Header().Set("X-Request-ID", reqID)

	s.mu.RLock()
	wafEngine := s.waf
	blEngine := s.blacklist
	rlEngine := s.rateLimiter
	routerEngine := s.router
	circuitReg := s.circuitRegistry
	retryPol := s.retryPolicy
	cacheEngine := s.cache
	streamBrk := s.streamBroker
	s.mu.RUnlock()

	// 0. Security WAF Inspection
	if wafEngine != nil {
		if blocked, rule, reason := wafEngine.Inspect(r, clientIP, country); blocked {
			duration := time.Since(start)
			if s.analytics != nil {
				s.analytics.RecordRequest(clientIP, http.StatusForbidden)
			}
			if s.metrics != nil {
				s.metrics.RecordHTTPRequest("waf", r.Method, http.StatusForbidden, duration)
			}
			if streamBrk != nil {
				streamBrk.Publish(stream.RequestEvent{
					ID:         reqID,
					Timestamp:  start,
					ClientIP:   clientIP,
					Country:    country,
					Method:     r.Method,
					Path:       r.URL.Path,
					StatusCode: http.StatusForbidden,
					DurationMs: float64(duration.Microseconds()) / 1000.0,
					Upstream:   "WAF_BLOCK",
					WAFBlocked: true,
					WAFRule:    rule,
				})
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":    "Forbidden",
				"waf_rule": rule,
				"message":  reason,
			})
			return
		}
	}

	// 1. Security Blacklist Check
	if blEngine != nil && blEngine.IsBlocked(clientIP) {
		duration := time.Since(start)
		if s.analytics != nil {
			s.analytics.RecordRequest(clientIP, http.StatusForbidden)
		}
		if s.metrics != nil {
			s.metrics.RecordHTTPRequest("blacklist", r.Method, http.StatusForbidden, duration)
		}
		if streamBrk != nil {
			streamBrk.Publish(stream.RequestEvent{
				ID:         reqID,
				Timestamp:  start,
				ClientIP:   clientIP,
				Country:    country,
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: http.StatusForbidden,
				DurationMs: float64(duration.Microseconds()) / 1000.0,
				Upstream:   "BLACKLIST",
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"Forbidden","message":"Access Denied: Your IP address has been blacklisted by administrator"}`))
		return
	}

	// 2. Rate Limiting Check
	if rlEngine != nil && !rlEngine.Allow(clientIP) {
		duration := time.Since(start)
		if s.analytics != nil {
			s.analytics.RecordRequest(clientIP, http.StatusTooManyRequests)
		}
		if s.metrics != nil {
			s.metrics.RecordRateLimitExceeded()
			s.metrics.RecordHTTPRequest("ratelimit", r.Method, http.StatusTooManyRequests, duration)
		}
		if streamBrk != nil {
			streamBrk.Publish(stream.RequestEvent{
				ID:         reqID,
				Timestamp:  start,
				ClientIP:   clientIP,
				Country:    country,
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: http.StatusTooManyRequests,
				DurationMs: float64(duration.Microseconds()) / 1000.0,
				Upstream:   "RATELIMIT",
			})
		}
		w.Header().Set("Retry-After", "1")
		http.Error(w, "Too Many Requests - Rate Limit Exceeded", http.StatusTooManyRequests)
		return
	}

	// 3. L7 Routing, Virtual Host Matching, Canary & Path Rewriting
	targetPool := s.pool
	isCanary := false
	if routerEngine != nil {
		route, pool, canary, _ := routerEngine.Match(r)
		if route != nil {
			route.ApplyRewrite(r)
			if route.Config.PreserveHost {
				r = r.WithContext(context.WithValue(r.Context(), preserveHostKey, true))
			}
		}
		if bp, ok := pool.(*BackendPool); ok && bp != nil {
			targetPool = bp
			isCanary = canary
		}
	}

	// 4. Edge Cache Check (GET/HEAD requests on canonical rewritten path)
	if cacheEngine != nil {
		if cached, hit := cacheEngine.Get(r); hit {
			duration := time.Since(start)
			cacheEngine.ServeCached(w, r, cached)
			if s.analytics != nil {
				s.analytics.RecordRequest(clientIP, cached.StatusCode)
			}
			if s.metrics != nil {
				s.metrics.RecordHTTPRequest("cache", r.Method, cached.StatusCode, duration)
			}
			if streamBrk != nil {
				streamBrk.Publish(stream.RequestEvent{
					ID:         reqID,
					Timestamp:  start,
					ClientIP:   clientIP,
					Country:    country,
					Method:     r.Method,
					Path:       r.URL.Path,
					StatusCode: cached.StatusCode,
					DurationMs: float64(duration.Microseconds()) / 1000.0,
					Upstream:   "CACHE_HIT",
					CacheHit:   true,
				})
			}
			return
		}
	}

	// 5. Load Balancing & Resilience (Circuit Breaker & Automatic Retries)
	bal := s.GetBalancer()
	maxAttempts := 1
	if retryPol != nil {
		maxAttempts = 1 + retryPol.MaxRetries()
	}

	// Buffer request body in memory so subsequent retry attempts don't send an exhausted empty body (EOF)
	var bodyBytes []byte
	if maxAttempts > 1 && r.Body != nil && (r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPost || r.Method == http.MethodPatch) {
		bodyBytes, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		r.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	var lastRecorder *bufferedRecorder
	var finalBackend *Backend
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Reset request body stream for this attempt if a body was present
		if len(bodyBytes) > 0 {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		backend, err := bal.NextHTTP(targetPool, r, clientIP)
		if err != nil {
			lastErr = err
			break
		}

		backendHost := backend.Addr
		if backend.URL != nil {
			backendHost = backend.URL.Host
		}

		// Check Circuit Breaker for this backend
		if circuitReg != nil {
			cb := circuitReg.GetOrCreate(backendHost)
			if !cb.Allow() {
				// Circuit is open for this backend; retry on next backend if possible
				if attempt < maxAttempts-1 {
					continue
				}
				duration := time.Since(start)
				if s.metrics != nil {
					s.metrics.RecordHTTPRequest(backendHost, r.Method, http.StatusServiceUnavailable, duration)
				}
				if streamBrk != nil {
					streamBrk.Publish(stream.RequestEvent{
						ID:         reqID,
						Timestamp:  start,
						ClientIP:   clientIP,
						Country:    country,
						Method:     r.Method,
						Path:       r.URL.Path,
						StatusCode: http.StatusServiceUnavailable,
						DurationMs: float64(duration.Microseconds()) / 1000.0,
						Upstream:   backendHost + " [CIRCUIT_OPEN]",
					})
				}
				http.Error(w, "Service Unavailable: Circuit Breaker Open", http.StatusServiceUnavailable)
				return
			}
		}

		finalBackend = backend
		backend.IncConns()
		if s.metrics != nil {
			s.metrics.IncHTTPActiveConn(backendHost)
		}

		// Buffer backend response so retry can occur if 5xx occurs
		bufRec := newBufferedRecorder()

		if backend.ReverseProxy != nil {
			backend.ReverseProxy.ServeHTTP(bufRec, r)
		} else if backend.URL != nil {
			rp := &httputil.ReverseProxy{
				Rewrite: func(pr *httputil.ProxyRequest) {
					pr.SetURL(backend.URL)
					pr.SetXForwarded()
					preserve, _ := pr.In.Context().Value(preserveHostKey).(bool)
					if !preserve {
						pr.Out.Host = backend.URL.Host
					}
				},
				Transport: s.transport,
			}
			rp.ServeHTTP(bufRec, r)
		}

		backend.DecConns()
		if s.metrics != nil {
			s.metrics.DecHTTPActiveConn(backendHost)
		}

		lastRecorder = bufRec
		code := bufRec.code
		if code == 0 {
			code = http.StatusOK
		}

		// Update circuit breaker
		if circuitReg != nil {
			cb := circuitReg.GetOrCreate(backendHost)
			if code >= 500 {
				cb.RecordFailure()
			} else {
				cb.RecordSuccess()
			}
		}

		// Check retryability
		if retryPol != nil && retryPol.IsRetryable(r, code, nil) && attempt < maxAttempts-1 {
			time.Sleep(retryPol.BackoffDuration(attempt))
			continue
		}

		// Success or non-retryable response
		break
	}

	duration := time.Since(start)

	if lastRecorder != nil {
		code := lastRecorder.code
		if code == 0 {
			code = http.StatusOK
		}

		// Store in Edge Cache if eligible
		if cacheEngine != nil && code == http.StatusOK {
			cacheEngine.Put(r, code, lastRecorder.header, lastRecorder.body.Bytes())
		}

		// Commit to client
		lastRecorder.FlushTo(w)

		backendHost := "unknown"
		if finalBackend != nil {
			backendHost = finalBackend.Addr
			if finalBackend.URL != nil {
				backendHost = finalBackend.URL.Host
			}
			if isCanary {
				backendHost += " [CANARY]"
			}
		}

		if s.analytics != nil {
			s.analytics.RecordRequest(clientIP, code)
		}
		if s.metrics != nil {
			s.metrics.RecordHTTPRequest(backendHost, r.Method, code, duration)
		}
		if streamBrk != nil {
			streamBrk.Publish(stream.RequestEvent{
				ID:         reqID,
				Timestamp:  start,
				ClientIP:   clientIP,
				Country:    country,
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: code,
				DurationMs: float64(duration.Microseconds()) / 1000.0,
				Upstream:   backendHost,
			})
		}
		return
	}

	// If all attempts failed or pool exhausted
	code := http.StatusServiceUnavailable
	if s.metrics != nil {
		s.metrics.RecordHTTPRequest("none", r.Method, code, duration)
	}
	if streamBrk != nil {
		streamBrk.Publish(stream.RequestEvent{
			ID:         reqID,
			Timestamp:  start,
			ClientIP:   clientIP,
			Country:    country,
			Method:     r.Method,
			Path:       r.URL.Path,
			StatusCode: code,
			DurationMs: float64(duration.Microseconds()) / 1000.0,
			Upstream:   "FAILED",
		})
	}
	errMsg := "Service Unavailable - No healthy backends"
	if lastErr != nil {
		errMsg = fmt.Sprintf("Service Unavailable: %v", lastErr)
	}
	http.Error(w, errMsg, http.StatusServiceUnavailable)
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
