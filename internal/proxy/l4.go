package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/metrics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/ratelimit"
)

// BalancerTCPInterface abstracts load balancing algorithms for L4.
type BalancerTCPInterface interface {
	Name() string
	NextTCP(pool *BackendPool, clientRemoteAddr string) (*Backend, error)
}

// L4Server represents a high-throughput L4 TCP reverse proxy and load balancer.
type L4Server struct {
	mu          sync.RWMutex
	config      config.L4Config
	timeouts    config.TimeoutsConfig
	pool        *BackendPool
	balancer    BalancerTCPInterface
	rateLimiter *ratelimit.ShardedTokenBucket
	metrics     *metrics.Metrics
	bufferPool  *BufferPool
	listener    net.Listener
	closed      atomic.Bool
	wg          sync.WaitGroup
}

// NewL4Server creates and initializes an L4 TCP load balancer.
func NewL4Server(
	cfg config.L4Config,
	timeouts config.TimeoutsConfig,
	pool *BackendPool,
	bal BalancerTCPInterface,
	rl *ratelimit.ShardedTokenBucket,
	m *metrics.Metrics,
) *L4Server {
	return &L4Server{
		config:      cfg,
		timeouts:    timeouts,
		pool:        pool,
		balancer:    bal,
		rateLimiter: rl,
		metrics:     m,
		bufferPool:  NewBufferPool(DefaultBufferSize),
	}
}

// Start begins listening and serving TCP traffic.
func (s *L4Server) Start() error {
	ln, err := net.Listen("tcp", s.config.Listen)
	if err != nil {
		return fmt.Errorf("l4 listen on %s failed: %w", s.config.Listen, err)
	}
	s.listener = ln
	log.Printf("[L4] TCP Load Balancer listening on %s (algorithm: %s)", s.config.Listen, s.balancer.Name())

	for {
		clientConn, err := ln.Accept()
		if err != nil {
			if s.closed.Load() {
				return nil
			}
			log.Printf("[L4] Accept error: %v", err)
			continue
		}

		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			s.handleTCPConn(c)
		}(clientConn)
	}
}

// Shutdown gracefully stops the L4 TCP server.
func (s *L4Server) Shutdown(ctx context.Context) error {
	s.closed.Store(true)
	if s.listener != nil {
		_ = s.listener.Close()
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetPool returns the backend pool for L4.
func (s *L4Server) GetPool() *BackendPool {
	return s.pool
}

// GetBalancer returns the active load balancer for TCP.
func (s *L4Server) GetBalancer() BalancerTCPInterface {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.balancer
}

// SetBalancer dynamically updates the load balancing algorithm for TCP.
func (s *L4Server) SetBalancer(bal BalancerTCPInterface) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balancer = bal
}

// AddBackendTarget dynamically adds a new backend to L4.
func (s *L4Server) AddBackendTarget(addr string, weight int) *Backend {
	backend := NewBackend(nil, addr, weight)
	s.pool.AddBackend(backend)
	if s.metrics != nil {
		s.metrics.SetBackendHealth(addr, true)
	}
	return backend
}

// RemoveBackendTarget removes a backend from the L4 pool.
func (s *L4Server) RemoveBackendTarget(addr string) bool {
	return s.pool.RemoveBackend(addr)
}

func (s *L4Server) handleTCPConn(clientConn net.Conn) {
	defer clientConn.Close()
	start := time.Now()

	clientAddr := clientConn.RemoteAddr().String()
	clientIP, _, err := net.SplitHostPort(clientAddr)
	if err != nil {
		clientIP = clientAddr
	}

	// 1. Rate Limiting Check
	if s.rateLimiter != nil && !s.rateLimiter.Allow(clientIP) {
		if s.metrics != nil {
			s.metrics.RecordRateLimitExceeded()
		}
		return
	}

	// 2. Select backend
	bal := s.GetBalancer()
	backend, err := bal.NextTCP(s.pool, clientAddr)
	if err != nil {
		log.Printf("[L4] No healthy backends available for client %s", clientAddr)
		return
	}

	backend.IncConns()
	backendAddr := backend.Addr
	if backend.URL != nil && backendAddr == "" {
		backendAddr = backend.URL.Host
	}

	if s.metrics != nil {
		s.metrics.RecordTCPConn(backendAddr)
	}

	defer func() {
		backend.DecConns()
		if s.metrics != nil {
			s.metrics.CloseTCPConn(backendAddr, time.Since(start))
		}
	}()

	// 3. Dial upstream TCP target
	backendConn, err := net.DialTimeout("tcp", backendAddr, s.timeouts.DialTimeout)
	if err != nil {
		log.Printf("[L4] Failed to dial backend %s: %v", backendAddr, err)
		return
	}
	defer backendConn.Close()

	// 4. Bi-directional zero-allocation streaming
	var pipeWg sync.WaitGroup
	pipeWg.Add(2)

	// client -> backend
	go func() {
		defer pipeWg.Done()
		buf := s.bufferPool.Get()
		defer s.bufferPool.Put(buf)

		n, _ := io.CopyBuffer(backendConn, clientConn, *buf)
		if s.metrics != nil {
			s.metrics.RecordTCPBytes(backendAddr, "inbound", n)
		}
		if tcpConn, ok := backendConn.(*net.TCPConn); ok {
			_ = tcpConn.CloseWrite()
		}
	}()

	// backend -> client
	go func() {
		defer pipeWg.Done()
		buf := s.bufferPool.Get()
		defer s.bufferPool.Put(buf)

		n, _ := io.CopyBuffer(clientConn, backendConn, *buf)
		if s.metrics != nil {
			s.metrics.RecordTCPBytes(backendAddr, "outbound", n)
		}
		if tcpConn, ok := clientConn.(*net.TCPConn); ok {
			_ = tcpConn.CloseWrite()
		}
	}()

	pipeWg.Wait()
}
