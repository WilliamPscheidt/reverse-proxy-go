package routing

import (
	"math/rand"
	"net/http"
	"path"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// RouteConfig holds metadata and configuration for a route rule.
type RouteConfig struct {
	ID            string   `json:"id" yaml:"id"`
	Host          string   `json:"host" yaml:"host"`                   // e.g. "sub.example.com", "*.example.com", or "*"
	PathPrefix    string   `json:"path_prefix" yaml:"path_prefix"`     // e.g. "/api", "/teste"
	RewritePrefix string   `json:"rewrite_prefix" yaml:"rewrite_prefix"` // e.g. "/sitenovo"
	StripPrefix   bool     `json:"strip_prefix" yaml:"strip_prefix"`
	Methods       []string `json:"methods" yaml:"methods"`             // e.g. ["GET", "POST"] or empty for all
	CanaryWeight  int      `json:"canary_weight" yaml:"canary_weight"` // 0 to 100 percentage
	PreserveHost  bool     `json:"preserve_host" yaml:"preserve_host"` // true = keep client Host header, false = rewrite to upstream host
	Backends      []string `json:"backends,omitempty" yaml:"backends,omitempty"` // Target backend addresses
}

// Route represents a runtime routing entry with its associated target pools.
type Route struct {
	Config      RouteConfig
	PrimaryPool any
	CanaryPool  any
	totalHits   uint64
	canaryHits  uint64
}

// RouteInfo provides runtime statistics for dashboard display.
type RouteInfo struct {
	Config     RouteConfig `json:"config"`
	TotalHits  uint64      `json:"total_hits"`
	CanaryHits uint64      `json:"canary_hits"`
}

// Router handles L7 path, host, and header-based routing with traffic splitting.
type Router struct {
	mu          sync.RWMutex
	routes      []*Route
	defaultPool any
	rng         *rand.Rand
	rngMu       sync.Mutex
}

// NewRouter creates an initialized Router instance with a fallback default pool.
func NewRouter(defaultPool any) *Router {
	return &Router{
		routes:      make([]*Route, 0),
		defaultPool: defaultPool,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// AddRoute registers a new route rule, keeping routes sorted by longest prefix first.
func (r *Router) AddRoute(cfg RouteConfig, primary any, canary any) *Route {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cfg.PathPrefix == "" {
		cfg.PathPrefix = "/"
	}
	if cfg.Host == "" {
		cfg.Host = "*"
	}
	if cfg.CanaryWeight < 0 {
		cfg.CanaryWeight = 0
	} else if cfg.CanaryWeight > 100 {
		cfg.CanaryWeight = 100
	}

	route := &Route{
		Config:      cfg,
		PrimaryPool: primary,
		CanaryPool:  canary,
	}

	// Remove existing route with the same ID if exists
	newRoutes := make([]*Route, 0, len(r.routes)+1)
	for _, existing := range r.routes {
		if existing.Config.ID != cfg.ID {
			newRoutes = append(newRoutes, existing)
		}
	}
	newRoutes = append(newRoutes, route)

	// Sort routes so more specific routes match before general ones:
	// 1. Non-wildcard host routes come before wildcard host ("*") routes
	// 2. Exact host routes come before wildcard subdomain ("*.domain") routes
	// 3. Longer PathPrefix routes come before shorter ones (/api/v2 > /api > /)
	sort.Slice(newRoutes, func(i, j int) bool {
		iWild := newRoutes[i].Config.Host == "*" || newRoutes[i].Config.Host == ""
		jWild := newRoutes[j].Config.Host == "*" || newRoutes[j].Config.Host == ""
		if iWild != jWild {
			return !iWild
		}

		iSub := strings.HasPrefix(newRoutes[i].Config.Host, "*.")
		jSub := strings.HasPrefix(newRoutes[j].Config.Host, "*.")
		if iSub != jSub {
			return !iSub
		}

		return len(newRoutes[i].Config.PathPrefix) > len(newRoutes[j].Config.PathPrefix)
	})

	r.routes = newRoutes
	return route
}

// matchHost tests if request host matches the configured route host pattern.
// Supports:
// - "*" or empty: matches any host
// - exact match: "api.example.com"
// - wildcard subdomain: "*.example.com" (matches "sub.example.com", "sub.sub.example.com", etc.)
// - glob pattern: "site-*.com"
func matchHost(pattern, host string) bool {
	if pattern == "*" || pattern == "" {
		return true
	}
	pattern = strings.ToLower(pattern)
	host = strings.ToLower(host)

	// Exact match
	if pattern == host {
		return true
	}

	// Wildcard subdomain: *.domain.com
	if strings.HasPrefix(pattern, "*.") {
		domain := pattern[2:]
		if strings.HasSuffix(host, "."+domain) || host == domain {
			return true
		}
	}

	// Glob pattern match
	matched, err := path.Match(pattern, host)
	if err == nil && matched {
		return true
	}

	return false
}

// ApplyRewrite applies PathPrefix stripping or RewritePrefix replacement to req.URL.
func (rt *Route) ApplyRewrite(req *http.Request) {
	if rt == nil || req == nil || req.URL == nil {
		return
	}

	origPath := req.URL.Path
	if origPath == "" {
		origPath = "/"
	}

	if rt.Config.RewritePrefix != "" {
		prefix := rt.Config.PathPrefix
		if prefix == "" {
			prefix = "/"
		}
		if prefix == "/" {
			req.URL.Path = strings.TrimSuffix(rt.Config.RewritePrefix, "/") + origPath
		} else if strings.HasPrefix(origPath, prefix) {
			suffix := strings.TrimPrefix(origPath, prefix)
			var newPath string
			if suffix == "" || suffix == "/" {
				newPath = rt.Config.RewritePrefix
				if suffix == "/" && !strings.HasSuffix(newPath, "/") {
					newPath += "/"
				}
			} else {
				newPath = strings.TrimSuffix(rt.Config.RewritePrefix, "/") + "/" + strings.TrimPrefix(suffix, "/")
			}
			req.URL.Path = newPath
		}
	} else if rt.Config.StripPrefix {
		prefix := rt.Config.PathPrefix
		if prefix != "" && prefix != "/" && strings.HasPrefix(origPath, prefix) {
			stripped := strings.TrimPrefix(origPath, prefix)
			if stripped == "" || !strings.HasPrefix(stripped, "/") {
				stripped = "/" + stripped
			}
			req.URL.Path = stripped
		}
	}
	req.URL.RawPath = req.URL.EscapedPath()
}

// RemoveRoute deletes a route rule by ID.
func (r *Router) RemoveRoute(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	newRoutes := make([]*Route, 0, len(r.routes))
	for _, rt := range r.routes {
		if rt.Config.ID == id {
			found = true
			continue
		}
		newRoutes = append(newRoutes, rt)
	}
	if found {
		r.routes = newRoutes
	}
	return found
}

// SetCanaryWeight dynamically adjusts the traffic percentage to the canary pool.
func (r *Router) SetCanaryWeight(id string, weight int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if weight < 0 {
		weight = 0
	} else if weight > 100 {
		weight = 100
	}

	for _, rt := range r.routes {
		if rt.Config.ID == id {
			rt.Config.CanaryWeight = weight
			return true
		}
	}
	return false
}

// matchPathPrefix checks if reqPath matches route prefix, respecting segment boundaries.
// e.g. /teste matches /teste and /teste/foo, but NOT /testeXYZ.
func matchPathPrefix(reqPath, prefix string) bool {
	if prefix == "" || prefix == "/" {
		return true
	}
	cleanPrefix := strings.TrimSuffix(prefix, "/")
	if reqPath == cleanPrefix {
		return true
	}
	return strings.HasPrefix(reqPath, cleanPrefix+"/")
}

// Match inspects the request and selects the appropriate route and backend pool.
// Returns the matched Route, selected BackendPool, whether canary was chosen, and whether a custom route matched.
func (r *Router) Match(req *http.Request) (route *Route, pool any, isCanary bool, matched bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	host := req.Host
	if colon := strings.IndexByte(host, ':'); colon != -1 {
		host = host[:colon] // strip port
	}
	path := req.URL.Path

	for _, rt := range r.routes {
		// 1. Host match (exact, wildcard subdomain *.blabla.com, or glob)
		if !matchHost(rt.Config.Host, host) {
			continue
		}

		// 2. Path prefix match (ensuring path boundary: /teste matches /teste or /teste/..., but NOT /testeXYZ)
		if !matchPathPrefix(path, rt.Config.PathPrefix) {
			continue
		}

		// 3. Method match
		if len(rt.Config.Methods) > 0 {
			methodMatches := false
			for _, m := range rt.Config.Methods {
				if strings.EqualFold(m, req.Method) {
					methodMatches = true
					break
				}
			}
			if !methodMatches {
				continue
			}
		}

		// Matched this route!
		atomic.AddUint64(&rt.totalHits, 1)

		// 4. Traffic Splitting / Canary selection
		if rt.CanaryPool != nil && rt.Config.CanaryWeight > 0 {
			r.rngMu.Lock()
			roll := r.rng.Intn(100)
			r.rngMu.Unlock()

			if roll < rt.Config.CanaryWeight {
				atomic.AddUint64(&rt.canaryHits, 1)
				return rt, rt.CanaryPool, true, true
			}
		}

		return rt, rt.PrimaryPool, false, true
	}

	// Fallback to default pool
	return nil, r.defaultPool, false, false
}

// ListRoutes returns an overview of all active routes and their traffic statistics.
func (r *Router) ListRoutes() []RouteInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info := make([]RouteInfo, 0, len(r.routes))
	for _, rt := range r.routes {
		info = append(info, RouteInfo{
			Config:     rt.Config,
			TotalHits:  atomic.LoadUint64(&rt.totalHits),
			CanaryHits: atomic.LoadUint64(&rt.canaryHits),
		})
	}
	return info
}
