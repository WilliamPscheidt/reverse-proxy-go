package balancer

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
)

// ErrNoHealthyBackends is returned when all backends in the pool are unhealthy.
var ErrNoHealthyBackends = errors.New("no healthy backends available in pool")

// Balancer defines the interface for load balancing algorithms across L4 and L7.
type Balancer interface {
	Name() string
	NextHTTP(pool *proxy.BackendPool, req *http.Request, clientIP string) (*proxy.Backend, error)
	NextTCP(pool *proxy.BackendPool, clientRemoteAddr string) (*proxy.Backend, error)
}

// New creates a Balancer based on the algorithm name.
// Supported algorithms: "round_robin", "least_conn", "ip_hash".
func New(algorithm string) (Balancer, error) {
	switch strings.ToLower(strings.TrimSpace(algorithm)) {
	case "round_robin", "roundrobin", "rr":
		return NewRoundRobin(), nil
	case "least_conn", "leastconn", "lc":
		return NewLeastConnections(), nil
	case "ip_hash", "iphash", "hash":
		return NewIPHash(), nil
	default:
		return nil, fmt.Errorf("unknown balancing algorithm: %q (supported: round_robin, least_conn, ip_hash)", algorithm)
	}
}
