package balancer

import (
	"net"
	"net/http"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
	"github.com/cespare/xxhash/v2"
)

// IPHash routes traffic deterministically based on client IP hash to provide session stickiness.
type IPHash struct{}

// NewIPHash creates a new IPHash balancer.
func NewIPHash() *IPHash {
	return &IPHash{}
}

// Name returns the algorithm name.
func (i *IPHash) Name() string {
	return "ip_hash"
}

// NextHTTP selects backend based on hashed client IP.
func (i *IPHash) NextHTTP(pool *proxy.BackendPool, req *http.Request, clientIP string) (*proxy.Backend, error) {
	ip := extractCleanIP(clientIP)
	return i.selectByHash(pool, ip)
}

// NextTCP selects backend based on hashed client IP or remote address.
func (i *IPHash) NextTCP(pool *proxy.BackendPool, clientRemoteAddr string) (*proxy.Backend, error) {
	ip := extractCleanIP(clientRemoteAddr)
	return i.selectByHash(pool, ip)
}

func (i *IPHash) selectByHash(pool *proxy.BackendPool, ip string) (*proxy.Backend, error) {
	healthy := pool.GetHealthyBackends()
	if len(healthy) == 0 {
		return nil, ErrNoHealthyBackends
	}

	hash := xxhash.Sum64String(ip)
	idx := hash % uint64(len(healthy))
	return healthy[idx], nil
}

// extractCleanIP strips port and spaces if present.
func extractCleanIP(addr string) string {
	if addr == "" {
		return "127.0.0.1"
	}
	host, _, err := net.SplitHostPort(addr)
	if err == nil && host != "" {
		return host
	}
	return addr
}
