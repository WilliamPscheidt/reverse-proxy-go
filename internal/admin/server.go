package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/analytics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/balancer"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/metrics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/ratelimit"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/security"
)

// Server handles administrative web dashboard and REST API endpoints.
type Server struct {
	cfg         *config.Config
	l7Server    *proxy.L7Server
	l4Server    *proxy.L4Server
	rateLimiter ratelimit.Limiter
	metrics     *metrics.Metrics
	analytics   *analytics.Analytics
	blacklist   *security.Blacklist
	httpServer  *http.Server
	startTime   time.Time
}

// NewServer creates a new Admin Server instance.
func NewServer(
	cfg *config.Config,
	l7 *proxy.L7Server,
	l4 *proxy.L4Server,
	rl ratelimit.Limiter,
	m *metrics.Metrics,
	a *analytics.Analytics,
	bl *security.Blacklist,
) *Server {
	s := &Server{
		cfg:         cfg,
		l7Server:    l7,
		l4Server:    l4,
		rateLimiter: rl,
		metrics:     m,
		analytics:   a,
		blacklist:   bl,
		startTime:   time.Now(),
	}

	mux := http.NewServeMux()

	// Web UI
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/admin", s.handleDashboard)

	// REST API
	mux.HandleFunc("/api/status", s.handleAPIStatus)
	mux.HandleFunc("/api/analytics", s.handleAPIAnalytics)
	mux.HandleFunc("/api/analytics/reset", s.handleAPIAnalyticsReset)
	mux.HandleFunc("/api/blacklist", s.handleAPIBlacklist)
	mux.HandleFunc("/api/settings", s.handleAPISettings)
	mux.HandleFunc("/api/config", s.handleAPIConfig)
	mux.HandleFunc("/api/backends", s.handleAPIBackends)
	mux.HandleFunc("/api/backends/toggle", s.handleAPIToggleBackend)
	mux.HandleFunc("/api/algorithm", s.handleAPISetAlgorithm)
	mux.HandleFunc("/api/ratelimit", s.handleAPISetRateLimit)

	s.httpServer = &http.Server{
		Addr:    cfg.Admin.Listen,
		Handler: mux,
	}

	return s
}

// Start begins listening on the admin port.
func (s *Server) Start() error {
	log.Printf("[Admin] Web Dashboard & API available at http://%s (or http://localhost%s)", s.cfg.Admin.Listen, s.cfg.Admin.Listen)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the admin server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// StatusResponse is returned by /api/status.
type StatusResponse struct {
	Uptime         string                   `json:"uptime"`
	Mode           string                   `json:"mode"`
	BlacklistCount int                      `json:"blacklist_count"`
	RateLimit      RateLimitInfo            `json:"rate_limit"`
	HealthCheck    config.HealthCheckConfig `json:"health_check"`
	L7             *LayerStatus             `json:"l7,omitempty"`
	L4             *LayerStatus             `json:"l4,omitempty"`
}

type RateLimitInfo struct {
	Enabled   bool    `json:"enabled"`
	Strategy  string  `json:"strategy"`
	Healthy   bool    `json:"healthy"`
	Rate      float64 `json:"rate"`
	Burst     int     `json:"burst"`
	RedisAddr string  `json:"redis_addr,omitempty"`
}

type LayerStatus struct {
	Enabled      bool   `json:"enabled"`
	Listen       string `json:"listen"`
	Algorithm    string `json:"algorithm"`
	TotalNodes   int    `json:"total_nodes"`
	HealthyNodes int    `json:"healthy_nodes"`
	ActiveConns  int64  `json:"active_conns"`
	TotalReqs    int64  `json:"total_requests"`
}

type BackendInfo struct {
	Type          string `json:"type"` // "l7" or "l4"
	Target        string `json:"target"`
	Weight        int    `json:"weight"`
	Alive         bool   `json:"alive"`
	ActiveConns   int64  `json:"active_conns"`
	TotalRequests int64  `json:"total_requests"`
	Failures      int32  `json:"consecutive_failures"`
	Successes     int32  `json:"consecutive_successes"`
}

func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	uptime := time.Since(s.startTime).Truncate(time.Second).String()

	strategy := "memory"
	healthy := true
	if s.rateLimiter != nil {
		strategy = s.rateLimiter.Strategy()
		healthy = s.rateLimiter.IsHealthy()
	}

	blCount := 0
	if s.blacklist != nil {
		blCount = s.blacklist.Count()
	}

	resp := StatusResponse{
		Uptime:         uptime,
		Mode:           s.cfg.Mode,
		BlacklistCount: blCount,
		HealthCheck:    s.cfg.HealthCheck,
		RateLimit: RateLimitInfo{
			Enabled:   s.cfg.RateLimit.Enabled,
			Strategy:  strategy,
			Healthy:   healthy,
			Rate:      s.cfg.RateLimit.Rate,
			Burst:     s.cfg.RateLimit.Burst,
			RedisAddr: s.cfg.RateLimit.Redis.Addr,
		},
	}

	if s.l7Server != nil {
		pool := s.l7Server.GetPool()
		var active int64
		var total int64
		for _, b := range pool.GetBackends() {
			active += b.GetActiveConns()
			total += b.TotalRequests.Load()
		}

		resp.L7 = &LayerStatus{
			Enabled:      s.cfg.L7.Enabled,
			Listen:       s.cfg.L7.Listen,
			Algorithm:    s.l7Server.GetBalancer().Name(),
			TotalNodes:   pool.BackendCount(),
			HealthyNodes: pool.HealthyCount(),
			ActiveConns:  active,
			TotalReqs:    total,
		}
	}

	if s.l4Server != nil {
		pool := s.l4Server.GetPool()
		var active int64
		var total int64
		for _, b := range pool.GetBackends() {
			active += b.GetActiveConns()
			total += b.TotalRequests.Load()
		}

		resp.L4 = &LayerStatus{
			Enabled:      s.cfg.L4.Enabled,
			Listen:       s.cfg.L4.Listen,
			Algorithm:    s.l4Server.GetBalancer().Name(),
			TotalNodes:   pool.BackendCount(),
			HealthyNodes: pool.HealthyCount(),
			ActiveConns:  active,
			TotalReqs:    total,
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleAPIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(s.cfg)
}

func (s *Server) handleAPIBackends(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var list []BackendInfo

		if s.l7Server != nil {
			for _, b := range s.l7Server.GetPool().GetBackends() {
				target := b.Addr
				if b.URL != nil {
					target = b.URL.String()
				}
				list = append(list, BackendInfo{
					Type:          "l7",
					Target:        target,
					Weight:        b.Weight,
					Alive:         b.IsAlive(),
					ActiveConns:   b.GetActiveConns(),
					TotalRequests: b.TotalRequests.Load(),
					Failures:      b.ConsecutiveFailures.Load(),
					Successes:     b.ConsecutiveSuccesses.Load(),
				})
			}
		}

		if s.l4Server != nil {
			for _, b := range s.l4Server.GetPool().GetBackends() {
				target := b.Addr
				list = append(list, BackendInfo{
					Type:          "l4",
					Target:        target,
					Weight:        b.Weight,
					Alive:         b.IsAlive(),
					ActiveConns:   b.GetActiveConns(),
					TotalRequests: b.TotalRequests.Load(),
					Failures:      b.ConsecutiveFailures.Load(),
					Successes:     b.ConsecutiveSuccesses.Load(),
				})
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(list)

	case http.MethodPost:
		// Add backend
		var req struct {
			Type   string `json:"type"`   // "l7" or "l4"
			Target string `json:"target"` // URL or host:port
			Weight int    `json:"weight"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
			return
		}

		if req.Weight <= 0 {
			req.Weight = 1
		}

		req.Type = strings.ToLower(strings.TrimSpace(req.Type))
		if req.Type == "l7" && s.l7Server != nil {
			b, err := s.l7Server.AddBackendTarget(req.Target, req.Weight)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "created", "backend": b.URL.String()})
			return
		} else if req.Type == "l4" && s.l4Server != nil {
			b := s.l4Server.AddBackendTarget(req.Target, req.Weight)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "created", "backend": b.Addr})
			return
		} else {
			http.Error(w, "unsupported type or layer disabled", http.StatusBadRequest)
			return
		}

	case http.MethodDelete:
		// Remove backend
		var req struct {
			Type   string `json:"type"`
			Target string `json:"target"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
			return
		}

		req.Type = strings.ToLower(strings.TrimSpace(req.Type))
		var ok bool
		if req.Type == "l7" && s.l7Server != nil {
			ok = s.l7Server.RemoveBackendTarget(req.Target)
		} else if req.Type == "l4" && s.l4Server != nil {
			ok = s.l4Server.RemoveBackendTarget(req.Target)
		}

		if !ok {
			http.Error(w, "backend not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "removed", "target": req.Target})

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAPIToggleBackend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type   string `json:"type"`
		Target string `json:"target"`
		Alive  bool   `json:"alive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	var backend *proxy.Backend
	if req.Type == "l7" && s.l7Server != nil {
		backend = s.l7Server.GetPool().FindByAddr(req.Target)
	} else if req.Type == "l4" && s.l4Server != nil {
		backend = s.l4Server.GetPool().FindByAddr(req.Target)
	}

	if backend == nil {
		http.Error(w, "backend not found in pool", http.StatusNotFound)
		return
	}

	backend.SetAlive(req.Alive)
	if s.metrics != nil {
		id := backend.Addr
		if backend.URL != nil {
			id = backend.URL.Host
		}
		s.metrics.SetBackendHealth(id, req.Alive)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "updated",
		"target": req.Target,
		"alive":  req.Alive,
	})
}

func (s *Server) handleAPISetAlgorithm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type      string `json:"type"`      // "l7", "l4" or "both"
		Algorithm string `json:"algorithm"` // "round_robin", "least_conn", "ip_hash"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	bal, err := balancer.New(req.Algorithm)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid algorithm: %v", err), http.StatusBadRequest)
		return
	}

	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if (req.Type == "l7" || req.Type == "both") && s.l7Server != nil {
		s.l7Server.SetBalancer(bal)
	}
	if (req.Type == "l4" || req.Type == "both") && s.l4Server != nil {
		s.l4Server.SetBalancer(bal)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "algorithm_updated",
		"algorithm": bal.Name(),
	})
}

func (s *Server) handleAPISetRateLimit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Enabled *bool   `json:"enabled"`
		Rate    float64 `json:"rate"`
		Burst   int     `json:"burst"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	if req.Enabled != nil {
		s.cfg.RateLimit.Enabled = *req.Enabled
	}
	if req.Rate > 0 && req.Burst > 0 && s.rateLimiter != nil {
		s.cfg.RateLimit.Rate = req.Rate
		s.cfg.RateLimit.Burst = req.Burst
		s.rateLimiter.SetRateAndBurst(req.Rate, req.Burst)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ratelimit_updated",
		"enabled": s.cfg.RateLimit.Enabled,
		"rate":    s.cfg.RateLimit.Rate,
		"burst":   s.cfg.RateLimit.Burst,
	})
}

func (s *Server) handleAPIAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if s.analytics != nil {
		_ = json.NewEncoder(w).Encode(s.analytics.GetSnapshot())
	} else {
		_ = json.NewEncoder(w).Encode(analytics.Snapshot{})
	}
}

func (s *Server) handleAPIAnalyticsReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.analytics != nil {
		s.analytics.Reset()
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "analytics_reset"})
}

func (s *Server) handleAPIBlacklist(w http.ResponseWriter, r *http.Request) {
	if s.blacklist == nil {
		http.Error(w, "blacklist service not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	switch r.Method {
	case http.MethodGet:
		entries := s.blacklist.List()
		_ = json.NewEncoder(w).Encode(entries)

	case http.MethodPost:
		var req struct {
			IP     string `json:"ip"`
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.IP) == "" {
			http.Error(w, "ip address cannot be empty", http.StatusBadRequest)
			return
		}

		entry := s.blacklist.Block(req.IP, req.Reason)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "blocked",
			"entry":  entry,
		})

	case http.MethodDelete:
		var req struct {
			IP string `json:"ip"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.IP) == "" {
			http.Error(w, "ip address cannot be empty", http.StatusBadRequest)
			return
		}

		ok := s.blacklist.Unblock(req.IP)
		if !ok {
			http.Error(w, "ip not found in blacklist", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "unblocked",
			"ip":     req.IP,
		})

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAPISettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(s.cfg)
	case http.MethodPost:
		var req struct {
			Mode        *string                   `json:"mode"`
			HealthCheck *config.HealthCheckConfig `json:"health_check"`
			Timeouts    *config.TimeoutsConfig    `json:"timeouts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
			return
		}
		if req.Mode != nil && *req.Mode != "" {
			s.cfg.Mode = *req.Mode
		}
		if req.HealthCheck != nil {
			s.cfg.HealthCheck = *req.HealthCheck
		}
		if req.Timeouts != nil {
			s.cfg.Timeouts = *req.Timeouts
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "settings_updated", "config": s.cfg})
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
