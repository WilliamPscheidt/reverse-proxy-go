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
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/config"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/health"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/metrics"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/ratelimit"
	"github.com/WilliamPscheidt/reverse-proxy-go/internal/security"
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
		fmt.Printf("Reverse Proxy & Load Balancer v%s\n", version)
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

	log.Printf("[Main] Starting Reverse Proxy & Load Balancer v%s (Mode: %s)", version, cfg.Mode)

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

	// 7. Setup Security Blacklist, Analytics & Web Administration Dashboard
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

	if l7Server != nil {
		l7Server.SetBlacklist(blacklist)
		l7Server.SetAnalytics(analyticsCollector)
	}
	if l4Server != nil {
		l4Server.SetBlacklist(blacklist)
	}

	if cfg.Admin.Enabled {
		adminServer = admin.NewServer(cfg, l7Server, l4Server, rateLimiter, m, analyticsCollector, blacklist)
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
