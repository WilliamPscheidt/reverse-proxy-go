package health

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
)

// StatusListener is an optional callback when a backend health state changes.
type StatusListener func(backend *proxy.Backend, healthy bool)

// Checker performs periodic active health checks against backend pools.
type Checker struct {
	config     config.HealthCheckConfig
	pools      []*proxy.BackendPool
	httpClient *http.Client
	listener   StatusListener
	mu         sync.Mutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewChecker creates a new Checker with an optimized HTTP client.
func NewChecker(cfg config.HealthCheckConfig, pools ...*proxy.BackendPool) *Checker {
	if cfg.Interval <= 0 {
		cfg.Interval = 3 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 1 * time.Second
	}
	if cfg.Path == "" {
		cfg.Path = "/health"
	}
	if cfg.ExpectedStatus == 0 {
		cfg.ExpectedStatus = http.StatusOK
	}
	if cfg.UnhealthyThreshold <= 0 {
		cfg.UnhealthyThreshold = 3
	}
	if cfg.HealthyThreshold <= 0 {
		cfg.HealthyThreshold = 2
	}

	return &Checker{
		config: cfg,
		pools:  pools,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				DisableKeepAlives: true, // Don't hold idle connections for health checks
			},
		},
		stopCh: make(chan struct{}),
	}
}

// SetStatusListener registers a callback invoked when a backend transitions healthy <-> unhealthy.
func (c *Checker) SetStatusListener(listener StatusListener) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listener = listener
}

// Start launches background health check workers.
func (c *Checker) Start(ctx context.Context) {
	c.wg.Add(1)
	go c.runLoop(ctx)
}

// Stop gracefully terminates the health checker.
func (c *Checker) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

func (c *Checker) runLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.Interval)
	defer ticker.Stop()

	// Perform initial check immediately
	c.checkAll()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.checkAll()
		}
	}
}

// checkAll iterates through all backends across registered pools concurrently.
func (c *Checker) checkAll() {
	var backends []*proxy.Backend
	for _, pool := range c.pools {
		backends = append(backends, pool.GetBackends()...)
	}

	var wg sync.WaitGroup
	for _, b := range backends {
		wg.Add(1)
		go func(backend *proxy.Backend) {
			defer wg.Done()
			c.checkBackend(backend)
		}(b)
	}
	wg.Wait()
}

// checkBackend checks a single backend and updates its status.
func (c *Checker) checkBackend(b *proxy.Backend) {
	var healthy bool

	if b.URL != nil {
		// HTTP health check
		url := fmt.Sprintf("%s%s", b.URL.String(), c.config.Path)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err == nil {
			req.Header.Set("User-Agent", "Go-ReverseProxy-HealthCheck/1.0")
			resp, err := c.httpClient.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				healthy = (resp.StatusCode == c.config.ExpectedStatus)
			}
		}
	} else if b.Addr != "" {
		// TCP dial health check
		conn, err := net.DialTimeout("tcp", b.Addr, c.config.Timeout)
		if err == nil {
			_ = conn.Close()
			healthy = true
		}
	}

	c.updateBackendStatus(b, healthy)
}

func (c *Checker) updateBackendStatus(b *proxy.Backend, healthy bool) {
	currentlyAlive := b.IsAlive()

	if healthy {
		b.ConsecutiveFailures.Store(0)
		successes := b.ConsecutiveSuccesses.Add(1)

		if !currentlyAlive && int(successes) >= c.config.HealthyThreshold {
			b.SetAlive(true)
			log.Printf("[HealthCheck] Backend %s recovered (healthy)", getBackendIdentifier(b))
			c.notifyListener(b, true)
		}
	} else {
		b.ConsecutiveSuccesses.Store(0)
		failures := b.ConsecutiveFailures.Add(1)

		if currentlyAlive && int(failures) >= c.config.UnhealthyThreshold {
			b.SetAlive(false)
			log.Printf("[HealthCheck] Backend %s failed %d checks, marked UNHEALTHY", getBackendIdentifier(b), failures)
			c.notifyListener(b, false)
		}
	}
}

func (c *Checker) notifyListener(b *proxy.Backend, healthy bool) {
	c.mu.Lock()
	listener := c.listener
	c.mu.Unlock()

	if listener != nil {
		listener(b, healthy)
	}
}

func getBackendIdentifier(b *proxy.Backend) string {
	if b.URL != nil {
		return b.URL.String()
	}
	return b.Addr
}
