package balancer

import (
	"net/http"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
)

// LeastConnections routes traffic to the healthy backend with the fewest active connections.
type LeastConnections struct{}

// NewLeastConnections creates a new LeastConnections balancer.
func NewLeastConnections() *LeastConnections {
	return &LeastConnections{}
}

// Name returns the algorithm name.
func (l *LeastConnections) Name() string {
	return "least_conn"
}

// NextHTTP selects the backend with the minimum active connections.
func (l *LeastConnections) NextHTTP(pool *proxy.BackendPool, req *http.Request, clientIP string) (*proxy.Backend, error) {
	return l.selectLeast(pool)
}

// NextTCP selects the backend with the minimum active connections.
func (l *LeastConnections) NextTCP(pool *proxy.BackendPool, clientRemoteAddr string) (*proxy.Backend, error) {
	return l.selectLeast(pool)
}

func (l *LeastConnections) selectLeast(pool *proxy.BackendPool) (*proxy.Backend, error) {
	healthy := pool.GetHealthyBackends()
	if len(healthy) == 0 {
		return nil, ErrNoHealthyBackends
	}

	var best *proxy.Backend
	var minConns int64 = -1

	for _, b := range healthy {
		active := b.GetActiveConns()
		if best == nil || active < minConns {
			best = b
			minConns = active
			if minConns == 0 {
				// Optimization: cannot beat 0 active connections
				break
			}
		}
	}

	return best, nil
}
