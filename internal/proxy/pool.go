package proxy

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

// Backend represents an upstream server target (L4 or L7).
type Backend struct {
	URL                   *url.URL
	Addr                  string
	Weight                int
	Alive                 atomic.Bool
	ActiveConns           atomic.Int64
	TotalRequests         atomic.Int64
	ConsecutiveFailures   atomic.Int32
	ConsecutiveSuccesses atomic.Int32
	ReverseProxy          *httputil.ReverseProxy
}

// NewBackend creates a new backend instance initialized as alive.
func NewBackend(u *url.URL, addr string, weight int) *Backend {
	if weight <= 0 {
		weight = 1
	}
	b := &Backend{
		URL:    u,
		Addr:   addr,
		Weight: weight,
	}
	b.Alive.Store(true)
	return b
}

// IsAlive returns whether the backend is currently considered healthy.
func (b *Backend) IsAlive() bool {
	return b.Alive.Load()
}

// SetAlive updates the alive status of the backend.
func (b *Backend) SetAlive(alive bool) {
	b.Alive.Store(alive)
}

// IncConns atomically increments active connections.
func (b *Backend) IncConns() int64 {
	b.TotalRequests.Add(1)
	return b.ActiveConns.Add(1)
}

// DecConns atomically decrements active connections.
func (b *Backend) DecConns() int64 {
	return b.ActiveConns.Add(-1)
}

// GetActiveConns returns the current number of active connections.
func (b *Backend) GetActiveConns() int64 {
	return b.ActiveConns.Load()
}

// BackendPool manages a collection of backends.
type BackendPool struct {
	backends []*Backend
	mu       sync.RWMutex
}

// NewBackendPool initializes an empty backend pool.
func NewBackendPool(backends ...*Backend) *BackendPool {
	return &BackendPool{
		backends: backends,
	}
}

// AddBackend adds a backend to the pool.
func (p *BackendPool) AddBackend(b *Backend) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.backends = append(p.backends, b)
}

// GetBackends returns all backends.
func (p *BackendPool) GetBackends() []*Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]*Backend, len(p.backends))
	copy(result, p.backends)
	return result
}

// GetHealthyBackends returns only backends marked as alive.
func (p *BackendPool) GetHealthyBackends() []*Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var healthy []*Backend
	for _, b := range p.backends {
		if b.IsAlive() {
			healthy = append(healthy, b)
		}
	}
	return healthy
}

// BackendCount returns total number of configured backends.
func (p *BackendPool) BackendCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.backends)
}

// HealthyCount returns total number of healthy backends.
func (p *BackendPool) HealthyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	count := 0
	for _, b := range p.backends {
		if b.IsAlive() {
			count++
		}
	}
	return count
}

// FindByAddr looks up a backend by its address string.
func (p *BackendPool) FindByAddr(addr string) *Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, b := range p.backends {
		if b.Addr == addr || (b.URL != nil && b.URL.Host == addr) || (b.URL != nil && b.URL.String() == addr) {
			return b
		}
	}
	return nil
}

// RemoveBackend removes a backend matching addr from the pool. Returns true if removed.
func (p *BackendPool) RemoveBackend(addr string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, b := range p.backends {
		if b.Addr == addr || (b.URL != nil && b.URL.Host == addr) || (b.URL != nil && b.URL.String() == addr) {
			p.backends = append(p.backends[:i], p.backends[i+1:]...)
			return true
		}
	}
	return false
}
