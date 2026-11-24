package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds the registered Prometheus metrics for the proxy.
type Metrics struct {
	registry *prometheus.Registry

	httpRequestsTotal     *prometheus.CounterVec
	httpRequestDuration   *prometheus.HistogramVec
	httpRequestSummary    *prometheus.SummaryVec
	httpActiveConnections *prometheus.GaugeVec

	backendHealthStatus *prometheus.GaugeVec

	rateLimitExceededTotal prometheus.Counter

	tcpConnectionsTotal    *prometheus.CounterVec
	tcpBytesTransferred    *prometheus.CounterVec
	tcpConnectionDuration  *prometheus.HistogramVec
	tcpActiveConnections   *prometheus.GaugeVec
}

// NewMetrics initializes and registers all metrics in an isolated Prometheus registry.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	// Register standard process and Go runtime collectors
	m := &Metrics{
		registry: reg,

		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "loadbalancer",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed by the reverse proxy.",
			},
			[]string{"backend", "method", "status"},
		),

		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "loadbalancer",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request latency histogram in seconds (supports p50, p95, p99 quantiles).",
				Buckets: []float64{
					0.00025, 0.0005, 0.001, 0.002, 0.003, 0.005,
					0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0,
				},
			},
			[]string{"backend", "method", "status"},
		),

		httpRequestSummary: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:  "loadbalancer",
				Subsystem:  "http",
				Name:       "request_latency_quantiles_seconds",
				Help:       "HTTP request latency summary with calculated p50, p90, p95, p99 percentiles.",
				Objectives: map[float64]float64{0.5: 0.05, 0.90: 0.01, 0.95: 0.005, 0.99: 0.001},
				MaxAge:     time.Minute,
			},
			[]string{"backend"},
		),

		httpActiveConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "loadbalancer",
				Subsystem: "http",
				Name:      "active_connections",
				Help:      "Current active HTTP connections per backend.",
			},
			[]string{"backend"},
		),

		backendHealthStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "loadbalancer",
				Subsystem: "backend",
				Name:      "health_status",
				Help:      "Current health status of the upstream backend (1 = healthy, 0 = unhealthy).",
			},
			[]string{"backend"},
		),

		rateLimitExceededTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "loadbalancer",
				Subsystem: "ratelimit",
				Name:      "exceeded_total",
				Help:      "Total number of requests rejected by the rate limiter (HTTP 429).",
			},
		),

		tcpConnectionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "loadbalancer",
				Subsystem: "tcp",
				Name:      "connections_total",
				Help:      "Total number of TCP connections handled by the L4 proxy.",
			},
			[]string{"backend"},
		),

		tcpBytesTransferred: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "loadbalancer",
				Subsystem: "tcp",
				Name:      "bytes_transferred_total",
				Help:      "Total bytes transferred via L4 TCP proxying.",
			},
			[]string{"backend", "direction"}, // direction: "inbound" (client->backend), "outbound" (backend->client)
		),

		tcpConnectionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "loadbalancer",
				Subsystem: "tcp",
				Name:      "connection_duration_seconds",
				Help:      "Duration of TCP proxy connections in seconds.",
				Buckets:   []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0},
			},
			[]string{"backend"},
		),

		tcpActiveConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "loadbalancer",
				Subsystem: "tcp",
				Name:      "active_connections",
				Help:      "Current active TCP connections per backend.",
			},
			[]string{"backend"},
		),
	}

	reg.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestSummary,
		m.httpActiveConnections,
		m.backendHealthStatus,
		m.rateLimitExceededTotal,
		m.tcpConnectionsTotal,
		m.tcpBytesTransferred,
		m.tcpConnectionDuration,
		m.tcpActiveConnections,
	)

	return m
}

// RecordHTTPRequest records an HTTP transaction latency, method, and status.
func (m *Metrics) RecordHTTPRequest(backend, method string, statusCode int, duration time.Duration) {
	statusStr := fmt.Sprintf("%d", statusCode)
	sec := duration.Seconds()

	m.httpRequestsTotal.WithLabelValues(backend, method, statusStr).Inc()
	m.httpRequestDuration.WithLabelValues(backend, method, statusStr).Observe(sec)
	m.httpRequestSummary.WithLabelValues(backend).Observe(sec)
}

// IncHTTPActiveConn increments active HTTP connection gauge for backend.
func (m *Metrics) IncHTTPActiveConn(backend string) {
	m.httpActiveConnections.WithLabelValues(backend).Inc()
}

// DecHTTPActiveConn decrements active HTTP connection gauge for backend.
func (m *Metrics) DecHTTPActiveConn(backend string) {
	m.httpActiveConnections.WithLabelValues(backend).Dec()
}

// SetBackendHealth updates backend health gauge (1 = healthy, 0 = unhealthy).
func (m *Metrics) SetBackendHealth(backend string, healthy bool) {
	val := 0.0
	if healthy {
		val = 1.0
	}
	m.backendHealthStatus.WithLabelValues(backend).Set(val)
}

// RecordRateLimitExceeded records a rate-limiting drop (HTTP 429).
func (m *Metrics) RecordRateLimitExceeded() {
	m.rateLimitExceededTotal.Inc()
}

// RecordTCPConn records a new TCP connection handled by backend.
func (m *Metrics) RecordTCPConn(backend string) {
	m.tcpConnectionsTotal.WithLabelValues(backend).Inc()
	m.tcpActiveConnections.WithLabelValues(backend).Inc()
}

// CloseTCPConn decrements active TCP connection gauge and records duration.
func (m *Metrics) CloseTCPConn(backend string, duration time.Duration) {
	m.tcpActiveConnections.WithLabelValues(backend).Dec()
	m.tcpConnectionDuration.WithLabelValues(backend).Observe(duration.Seconds())
}

// RecordTCPBytes records transferred byte volume.
func (m *Metrics) RecordTCPBytes(backend, direction string, bytes int64) {
	m.tcpBytesTransferred.WithLabelValues(backend, direction).Add(float64(bytes))
}

// Handler returns an HTTP handler serving Prometheus metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
