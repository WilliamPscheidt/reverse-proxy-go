package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/admin"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/analytics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/balancer"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/cache"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/health"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/metrics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/ratelimit"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/resilience"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/routing"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/security"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/stream"
	"github.com/redis/go-redis/v9"
)

var (
	version = "1.0.0"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "Path to YAML configuration file")
	modeFlag := flag.String("mode", "", "Proxy mode: l7, l4, or both (overrides config)")
	algoFlag := flag.String("algo", "", "Balancing algorithm: round_robin, least_conn, ip_hash (overrides config)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ProxyGO v%s - High-Concurrency Reverse Proxy & Load Balancer\n", version)
		os.Exit(0)
	}

	// 1. Load Configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("[Main] Failed to load configuration: %v", err)
	}

	if *modeFlag != "" {
		cfg.Mode = *modeFlag
	}
	if *algoFlag != "" {
		cfg.L7.Algorithm = *algoFlag
		cfg.L4.Algorithm = *algoFlag
	}

	log.Printf("[ProxyGO] Starting ProxyGO v%s (Mode: %s)", version, cfg.Mode)

	// 2. Setup Prometheus Metrics Exporter
	var m *metrics.Metrics
	var metricsServer *http.Server

	if cfg.Metrics.Enabled {
		m = metrics.NewMetrics()
		mux := http.NewServeMux()
		mux.Handle(cfg.Metrics.Path, m.Handler())
		metricsServer = &http.Server{
			Addr:    cfg.Metrics.Listen,
			Handler: mux,
		}

		go func() {
			log.Printf("[Metrics] Exposing Prometheus metrics at http://%s%s", cfg.Metrics.Listen, cfg.Metrics.Path)
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("[Metrics] Server error: %v", err)
			}
		}()
	}

	// 3. Setup Distributed/In-Memory Rate Limiter
	var rateLimiter ratelimit.Limiter
	if cfg.RateLimit.Enabled {
		rateLimiter = ratelimit.New(cfg.RateLimit)
		log.Printf("[RateLimit] Enabled (%s): %.0f tokens/s (burst: %d) per client IP",
			rateLimiter.Strategy(), cfg.RateLimit.Rate, cfg.RateLimit.Burst)
	}

	var healthPools []*proxy.BackendPool

	// 4. Setup L7 HTTP Reverse Proxy
	var l7Server *proxy.L7Server
	var l7Pool *proxy.BackendPool
	var canaryPool *proxy.BackendPool

	if (cfg.Mode == "l7" || cfg.Mode == "both") && cfg.L7.Enabled {
		l7Pool = proxy.NewBackendPool()
		for _, b := range cfg.L7.Backends {
			u := proxy.ParseURL(b.URL)
			if u == nil {
				log.Fatalf("[L7] Invalid backend URL: %s", b.URL)
			}
			backend := proxy.NewBackend(u, u.Host, b.Weight)
			l7Pool.AddBackend(backend)
			if m != nil {
				m.SetBackendHealth(u.Host, true)
			}
		}

		bal, err := balancer.New(cfg.L7.Algorithm)
		if err != nil {
			log.Fatalf("[L7] Balancer initialization failed: %v", err)
		}

		l7Server = proxy.NewL7Server(cfg.L7, cfg.Timeouts, l7Pool, bal, rateLimiter, m)
		healthPools = append(healthPools, l7Pool)

		if len(cfg.L7.CanaryBackends) > 0 {
			canaryPool = proxy.NewBackendPool()
			for _, b := range cfg.L7.CanaryBackends {
				u := proxy.ParseURL(b.URL)
				if u != nil {
					backend := proxy.NewBackend(u, u.Host, b.Weight)
					canaryPool.AddBackend(backend)
					if m != nil {
						m.SetBackendHealth(u.Host, true)
					}
				}
			}
			l7Server.SetCanaryPool(canaryPool)
			healthPools = append(healthPools, canaryPool)
			log.Printf("[L7] Configured %d dedicated Canary upstream backend(s)", len(cfg.L7.CanaryBackends))
		}

		go func() {
			if err := l7Server.Start(); err != nil {
				log.Fatalf("[L7] Server crashed: %v", err)
			}
		}()
	}

	// 5. Setup L4 TCP Load Balancer
	var l4Server *proxy.L4Server
	var l4Pool *proxy.BackendPool

	if (cfg.Mode == "l4" || cfg.Mode == "both") && cfg.L4.Enabled {
		l4Pool = proxy.NewBackendPool()
		for _, b := range cfg.L4.Backends {
			backend := proxy.NewBackend(nil, b.Addr, b.Weight)
			l4Pool.AddBackend(backend)
			if m != nil {
				m.SetBackendHealth(b.Addr, true)
			}
		}

		bal, err := balancer.New(cfg.L4.Algorithm)
		if err != nil {
			log.Fatalf("[L4] Balancer initialization failed: %v", err)
		}

		l4Server = proxy.NewL4Server(cfg.L4, cfg.Timeouts, l4Pool, bal, rateLimiter, m)
		healthPools = append(healthPools, l4Pool)

		go func() {
			if err := l4Server.Start(); err != nil {
				log.Fatalf("[L4] Server crashed: %v", err)
			}
		}()
	}

	// 6. Setup Active Background Health Check
	var healthChecker *health.Checker
	if cfg.HealthCheck.Enabled && len(healthPools) > 0 {
		healthChecker = health.NewChecker(cfg.HealthCheck, healthPools...)
		if m != nil {
			healthChecker.SetStatusListener(func(b *proxy.Backend, healthy bool) {
				id := b.Addr
				if b.URL != nil {
					id = b.URL.Host
				}
				m.SetBackendHealth(id, healthy)
			})
		}

		ctxHealth, cancelHealth := context.WithCancel(context.Background())
		defer cancelHealth()
		healthChecker.Start(ctxHealth)
		log.Printf("[HealthCheck] Active background checker started (interval: %v, timeout: %v)",
			cfg.HealthCheck.Interval, cfg.HealthCheck.Timeout)
	}

	// 7. Setup Security WAF, Blacklist, Cache, Resilience, Router & Live Stream
	var adminServer *admin.Server
	var redisClient *redis.Client
	if cfg.RateLimit.Strategy == "redis" && cfg.RateLimit.Redis.Addr != "" {
		if rl, ok := rateLimiter.(*ratelimit.RedisLimiter); ok {
			redisClient = rl.Client()
		}
	}
	blacklist := security.NewBlacklist(redisClient)

	analyticsCollector := analytics.NewAnalytics()
	analyticsCollector.SetBlacklistChecker(blacklist.IsBlocked)

	// WAF (Web Application Firewall)
	wafCfg := security.WAFConfig{
		Enabled:            cfg.WAF.Enabled,
		BlockSQLi:          cfg.WAF.BlockSQLi,
		BlockXSS:           cfg.WAF.BlockXSS,
		BlockPathTraversal: cfg.WAF.BlockPathTraversal,
		BlockBadBots:       cfg.WAF.BlockBadBots,
		BlockedCountries:   cfg.WAF.BlockedCountries,
		AllowedCountries:   cfg.WAF.AllowedCountries,
		BlockedCIDRs:       cfg.WAF.BlockedCIDRs,
	}
	wafEngine := security.NewWAF(wafCfg)

	// Edge Response Cache
	cacheCfg := cache.CacheConfig{
		Enabled:    cfg.Cache.Enabled,
		DefaultTTL: cfg.Cache.DefaultTTL,
		MaxEntries: cfg.Cache.MaxEntries,
	}
	cacheEngine := cache.NewCache(cacheCfg, redisClient)

	// Live Request Log Stream Broker
	streamBroker := stream.NewStreamBroker(200)

	// Circuit Breakers & Retries
	circuitRegistry := resilience.NewCircuitRegistry(resilience.CircuitConfig{
		FailureThreshold: cfg.Resilience.CircuitBreaker.FailureThreshold,
		SuccessThreshold: cfg.Resilience.CircuitBreaker.SuccessThreshold,
		Cooldown:         cfg.Resilience.CircuitBreaker.Cooldown,
	})
	retryPolicy := resilience.NewRetryPolicy(resilience.RetryConfig{
		Enabled:        cfg.Resilience.Retry.Enabled,
		MaxRetries:     cfg.Resilience.Retry.MaxRetries,
		InitialBackoff: cfg.Resilience.Retry.InitialBackoff,
		MaxBackoff:     cfg.Resilience.Retry.MaxBackoff,
	})

	// L7 Router & Canary Traffic Splitting
	targetCanary := canaryPool
	if targetCanary == nil {
		targetCanary = l7Pool
	}
	routerEngine := routing.NewRouter(l7Pool)
	if len(cfg.Routes) > 0 {
		for _, r := range cfg.Routes {
			routePrimary := any(l7Pool)
			routeCanary := any(targetCanary)

			// If route specifies dedicated backends, instantiate dedicated BackendPool
			if len(r.Backends) > 0 {
				routePool := proxy.NewBackendPool()
				for _, b := range r.Backends {
					u := proxy.ParseURL(b.URL)
					if u != nil {
						backend := proxy.NewBackend(u, u.Host, b.Weight)
						routePool.AddBackend(backend)
						if m != nil {
							m.SetBackendHealth(u.Host, true)
						}
					}
				}
				if l7Server != nil {
					l7Server.InitPoolReverseProxies(routePool)
				}
				healthPools = append(healthPools, routePool)
				routePrimary = routePool
				routeCanary = routePool
				log.Printf("[L7 Router] Route '%s' (host: '%s', prefix: '%s') with %d dedicated backend(s)",
					r.ID, r.Host, r.PathPrefix, len(r.Backends))
			}

			routerEngine.AddRoute(routing.RouteConfig{
				ID:            r.ID,
				Host:          r.Host,
				PathPrefix:    r.PathPrefix,
				RewritePrefix: r.RewritePrefix,
				StripPrefix:   r.StripPrefix,
				Methods:       r.Methods,
				CanaryWeight:  r.CanaryWeight,
				PreserveHost:  r.PreserveHost,
			}, routePrimary, routeCanary)
		}
	} else if l7Pool != nil {
		routerEngine.AddRoute(routing.RouteConfig{
			ID:           "default",
			Host:         "*",
			PathPrefix:   "/",
			CanaryWeight: 10,
		}, l7Pool, targetCanary)
	}

	if l7Server != nil {
		l7Server.SetBlacklist(blacklist)
		l7Server.SetAnalytics(analyticsCollector)
		l7Server.SetWAF(wafEngine)
		l7Server.SetCache(cacheEngine)
		l7Server.SetRouter(routerEngine)
		l7Server.SetResilience(circuitRegistry, retryPolicy)
		l7Server.SetStreamBroker(streamBroker)
	}
	if l4Server != nil {
		l4Server.SetBlacklist(blacklist)
	}

	if cfg.Admin.Enabled {
		adminServer = admin.NewServer(cfg, l7Server, l4Server, rateLimiter, m, analyticsCollector, blacklist)
		adminServer.SetWAF(wafEngine)
		adminServer.SetCache(cacheEngine)
		adminServer.SetRouter(routerEngine)
		adminServer.SetCircuitRegistry(circuitRegistry)
		adminServer.SetStreamBroker(streamBroker)

		go func() {
			if err := adminServer.Start(); err != nil && err != http.ErrServerClosed {
				log.Printf("[Admin] Server error: %v", err)
			}
		}()
	}

	// 8. Wait for OS Termination Signal (Graceful Shutdown)
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGTERM)
	sig := <-stopSignal

	log.Printf("[Main] Received signal %v, shutting down gracefully...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if healthChecker != nil {
		healthChecker.Stop()
	}
	if rateLimiter != nil {
		rateLimiter.Close()
	}
	if l7Server != nil {
		_ = l7Server.Shutdown(shutdownCtx)
	}
	if l4Server != nil {
		_ = l4Server.Shutdown(shutdownCtx)
	}
	if metricsServer != nil {
		_ = metricsServer.Shutdown(shutdownCtx)
	}
	if adminServer != nil {
		_ = adminServer.Shutdown(shutdownCtx)
	}

	log.Println("[Main] Server stopped cleanly. Bye!")
}
