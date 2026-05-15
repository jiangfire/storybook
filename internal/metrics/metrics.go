// Package metrics owns the Prometheus collectors used across the service.
// A package-scoped registry keeps our exposition independent of the global
// default registry, making tests deterministic and preventing accidental
// double-registration.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Registry is the single registry exposed by /metrics.
var Registry = prometheus.NewRegistry()

var (
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds keyed by method, gin route and response status.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)

	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests handled keyed by method, gin route and response status.",
		},
		[]string{"method", "route", "status"},
	)

	AICallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_calls_total",
			Help: "Total AI calls keyed by operation, model and outcome (success|error|retry).",
		},
		[]string{"operation", "model", "status"},
	)

	AICallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_call_duration_seconds",
			Help:    "AI call latency in seconds keyed by operation and model.",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 20, 30, 60},
		},
		[]string{"operation", "model"},
	)

	AITokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_tokens_total",
			Help: "Total tokens consumed keyed by operation, model and kind (prompt|completion).",
		},
		[]string{"operation", "model", "kind"},
	)

	WSConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ws_connections",
			Help: "Current number of active WebSocket connections.",
		},
	)

	CacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Cache hits keyed by cache name.",
		},
		[]string{"cache"},
	)

	CacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Cache misses keyed by cache name.",
		},
		[]string{"cache"},
	)

	RateLimitRejections = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_rejections_total",
			Help: "Requests rejected by a rate limiter keyed by limiter name.",
		},
		[]string{"limiter"},
	)
)

func init() {
	Registry.MustRegister(
		HTTPRequestDuration,
		HTTPRequestsTotal,
		AICallsTotal,
		AICallDuration,
		AITokensTotal,
		WSConnections,
		CacheHits,
		CacheMisses,
		RateLimitRejections,
	)
}
