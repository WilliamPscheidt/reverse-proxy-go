package balancer

import (
	"net/http"
	"sync/atomic"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
)

// RoundRobin distributes traffic sequentially across available healthy backends.
type RoundRobin struct {
	counter atomic.Uint64
}

// NewRoundRobin creates a new RoundRobin balancer.
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

// Name returns the algorithm name.
func (r *RoundRobin) Name() string {
	return "round_robin"
}

// NextHTTP selects the next backend for HTTP requests.
func (r *RoundRobin) NextHTTP(pool *proxy.BackendPool, req *http.Request, clientIP string) (*proxy.Backend, error) {
	healthy := pool.GetHealthyBackends()
	if len(healthy) == 0 {
		return nil, ErrNoHealthyBackends
	}

	idx := r.counter.Add(1) - 1
	return healthy[idx%uint64(len(healthy))], nil
}

// NextTCP selects the next backend for TCP connections.
func (r *RoundRobin) NextTCP(pool *proxy.BackendPool, clientRemoteAddr string) (*proxy.Backend, error) {
	healthy := pool.GetHealthyBackends()
	if len(healthy) == 0 {
		return nil, ErrNoHealthyBackends
	}

	idx := r.counter.Add(1) - 1
	return healthy[idx%uint64(len(healthy))], nil
}
