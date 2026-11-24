package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config defines the complete configuration for the reverse proxy and load balancer.
type Config struct {
	Mode        string            `yaml:"mode"` // "l7", "l4", or "both"
	Admin       AdminConfig       `yaml:"admin"`
	L7          L7Config          `yaml:"l7"`
	L4          L4Config          `yaml:"l4"`
	Metrics     MetricsConfig     `yaml:"metrics"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
	RateLimit   RateLimitConfig   `yaml:"rate_limit"`
	Timeouts    TimeoutsConfig    `yaml:"timeouts"`
	WAF         WAFConfig         `yaml:"waf"`
	Cache       CacheConfig       `yaml:"cache"`
	Resilience  ResilienceConfig  `yaml:"resilience"`
	Routes      []RouteConfig     `yaml:"routes"`
}

// AdminConfig holds web administration dashboard settings.
type AdminConfig struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"` // e.g., ":8090"
}

// BackendConfig holds backend target configuration.
type BackendConfig struct {
	URL    string `yaml:"url"`    // For L7 (e.g., "http://127.0.0.1:8081")
	Addr   string `yaml:"addr"`   // For L4 (e.g., "127.0.0.1:9081")
	Weight int    `yaml:"weight"` // Relative weight for weighted algorithms
}

// L7Config holds HTTP reverse proxy settings.
type L7Config struct {
	Enabled        bool            `yaml:"enabled"`
	Listen         string          `yaml:"listen"`    // e.g., ":8080"
	Algorithm      string          `yaml:"algorithm"` // "round_robin", "least_conn", "ip_hash"
	Backends       []BackendConfig `yaml:"backends"`
	CanaryBackends []BackendConfig `yaml:"canary_backends"`
}

// L4Config holds TCP load balancer settings.
type L4Config struct {
	Enabled   bool            `yaml:"enabled"`
	Listen    string          `yaml:"listen"`    // e.g., ":9000"
	Algorithm string          `yaml:"algorithm"` // "round_robin", "least_conn", "ip_hash"
	Backends  []BackendConfig `yaml:"backends"`
}

// MetricsConfig holds Prometheus metrics exporter settings.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"` // e.g., ":2112"
	Path    string `yaml:"path"`   // e.g., "/metrics"
}

// HealthCheckConfig holds active background health check settings.
type HealthCheckConfig struct {
	Enabled            bool          `yaml:"enabled"`
	Interval           time.Duration `yaml:"interval"`             // e.g., 5s
	Timeout            time.Duration `yaml:"timeout"`              // e.g., 2s
	Path               string        `yaml:"path"`                 // HTTP health check path, e.g. "/health"
	ExpectedStatus     int           `yaml:"expected_status"`      // e.g. 200
	UnhealthyThreshold int           `yaml:"unhealthy_threshold"`  // consecutive failures before unhealthy
	HealthyThreshold   int           `yaml:"healthy_threshold"`    // consecutive successes before healthy
}

// RedisConfig holds connection options for distributed Redis rate limiting.
type RedisConfig struct {
	Addr     string        `yaml:"addr"`      // e.g., "127.0.0.1:6379"
	Password string        `yaml:"password"`  // optional
	DB       int           `yaml:"db"`        // default 0
	Timeout  time.Duration `yaml:"timeout"`   // e.g., 200ms
	PoolSize int           `yaml:"pool_size"` // e.g., 100
}

// RateLimitConfig holds token bucket rate limiter configuration.
type RateLimitConfig struct {
	Enabled         bool          `yaml:"enabled"`
	Strategy        string        `yaml:"strategy"`         // "redis" or "memory"
	Rate            float64       `yaml:"rate"`             // tokens replenished per second per client
	Burst           int           `yaml:"burst"`            // max token capacity per client
	CleanupInterval time.Duration `yaml:"cleanup_interval"` // interval to purge stale client buckets
	Redis           RedisConfig   `yaml:"redis"`
}

// TimeoutsConfig specifies network and connection timeouts.
type TimeoutsConfig struct {
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	DialTimeout       time.Duration `yaml:"dial_timeout"`
	MaxIdleConns      int           `yaml:"max_idle_conns"`
	MaxIdleConnsHost  int           `yaml:"max_idle_conns_per_host"`
}

// WAFConfig holds rules and toggles for Web Application Firewall inspection.
type WAFConfig struct {
	Enabled            bool     `yaml:"enabled"`
	BlockSQLi          bool     `yaml:"block_sqli"`
	BlockXSS           bool     `yaml:"block_xss"`
	BlockPathTraversal bool     `yaml:"block_path_traversal"`
	BlockBadBots       bool     `yaml:"block_bad_bots"`
	BlockedCountries   []string `yaml:"blocked_countries"`
	AllowedCountries   []string `yaml:"allowed_countries"`
	BlockedCIDRs       []string `yaml:"blocked_cidrs"`
}

// CacheConfig holds edge caching configuration.
type CacheConfig struct {
	Enabled    bool          `yaml:"enabled"`
	DefaultTTL time.Duration `yaml:"default_ttl"`
	MaxEntries int           `yaml:"max_entries"`
}

// CircuitBreakerConfig defines threshold rules for circuit breaking.
type CircuitBreakerConfig struct {
	FailureThreshold int           `yaml:"failure_threshold"`
	SuccessThreshold int           `yaml:"success_threshold"`
	Cooldown         time.Duration `yaml:"cooldown"`
}

// RetryConfig specifies retry behavior on upstream failures.
type RetryConfig struct {
	Enabled        bool          `yaml:"enabled"`
	MaxRetries     int           `yaml:"max_retries"`
	InitialBackoff time.Duration `yaml:"initial_backoff"`
	MaxBackoff     time.Duration `yaml:"max_backoff"`
}

// ResilienceConfig bundles circuit breaking and retry policies.
type ResilienceConfig struct {
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	Retry          RetryConfig          `yaml:"retry"`
}

// RouteConfig defines L7 routing and canary splitting.
type RouteConfig struct {
	ID           string   `yaml:"id"`
	Host         string   `yaml:"host"`
	PathPrefix   string   `yaml:"path_prefix"`
	Methods      []string `yaml:"methods"`
	CanaryWeight int      `yaml:"canary_weight"`
	StripPrefix  bool     `yaml:"strip_prefix"`
}

// DefaultConfig returns safe, high-performance defaults.
func DefaultConfig() *Config {
	return &Config{
		Mode: "both",
		Admin: AdminConfig{
			Enabled: true,
			Listen:  ":8090",
		},
		L7: L7Config{
			Enabled:   true,
			Listen:    ":8080",
			Algorithm: "round_robin",
			Backends: []BackendConfig{
				{URL: "http://127.0.0.1:8081", Weight: 1},
				{URL: "http://127.0.0.1:8082", Weight: 1},
				{URL: "http://127.0.0.1:8083", Weight: 1},
			},
			CanaryBackends: []BackendConfig{
				{URL: "http://127.0.0.1:8084", Weight: 1},
			},
		},
		L4: L4Config{
			Enabled:   true,
			Listen:    ":9000",
			Algorithm: "round_robin",
			Backends: []BackendConfig{
				{Addr: "127.0.0.1:9081", Weight: 1},
				{Addr: "127.0.0.1:9082", Weight: 1},
			},
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Listen:  ":2112",
			Path:    "/metrics",
		},
		HealthCheck: HealthCheckConfig{
			Enabled:            true,
			Interval:           3 * time.Second,
			Timeout:            1 * time.Second,
			Path:               "/health",
			ExpectedStatus:     200,
			UnhealthyThreshold: 3,
			HealthyThreshold:   2,
		},
		RateLimit: RateLimitConfig{
			Enabled:         true,
			Strategy:        "redis",
			Rate:            100000, // 100k req/s for ultra-high concurrency tests
			Burst:           200000,
			CleanupInterval: 1 * time.Minute,
			Redis: RedisConfig{
				Addr:     "127.0.0.1:6379",
				Password: "",
				DB:       0,
				Timeout:  150 * time.Millisecond,
				PoolSize: 100,
			},
		},
		Timeouts: TimeoutsConfig{
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     30 * time.Second,
			IdleTimeout:      120 * time.Second,
			DialTimeout:      3 * time.Second,
			MaxIdleConns:     10000,
			MaxIdleConnsHost: 2000,
		},
		WAF: WAFConfig{
			Enabled:            true,
			BlockSQLi:          true,
			BlockXSS:           true,
			BlockPathTraversal: true,
			BlockBadBots:       true,
			BlockedCountries:   []string{},
			AllowedCountries:   []string{},
			BlockedCIDRs:       []string{},
		},
		Cache: CacheConfig{
			Enabled:    true,
			DefaultTTL: 60 * time.Second,
			MaxEntries: 10000,
		},
		Resilience: ResilienceConfig{
			CircuitBreaker: CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 2,
				Cooldown:         5 * time.Second,
			},
			Retry: RetryConfig{
				Enabled:        true,
				MaxRetries:     2,
				InitialBackoff: 25 * time.Millisecond,
				MaxBackoff:     150 * time.Millisecond,
			},
		},
		Routes: []RouteConfig{},
	}
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if file not present
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML config: %w", err)
	}

	return cfg, nil
}
