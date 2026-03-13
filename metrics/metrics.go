package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "newapi_http_requests_total",
			Help: "Total number of HTTP requests processed by the server.",
		},
		[]string{"route_tag", "method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "newapi_http_request_duration_seconds",
			Help:    "HTTP request latency distributions.",
			Buckets: []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 30},
		},
		[]string{"route_tag", "method", "path"},
	)
	relayUpstreamRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "newapi_relay_upstream_requests_total",
			Help: "Total number of upstream requests by channel.",
		},
		[]string{"channel_id", "channel_type", "status"},
	)
	relayUpstreamLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "newapi_relay_upstream_latency_seconds",
			Help:    "Upstream request latency distributions.",
			Buckets: []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 30},
		},
		[]string{"channel_id", "channel_type"},
	)
	relayRetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "newapi_relay_retries_total",
			Help: "Total number of relay retries by channel.",
		},
		[]string{"channel_id", "channel_type"},
	)
	rateLimitHitsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "newapi_rate_limit_hits_total",
			Help: "Total number of rate limit hits.",
		},
		[]string{"scope", "mark"},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		relayUpstreamRequestsTotal,
		relayUpstreamLatency,
		relayRetriesTotal,
		rateLimitHitsTotal,
	)
}

func ObserveHTTPRequest(routeTag, method, path string, status int, duration time.Duration) {
	if routeTag == "" {
		routeTag = "web"
	}
	if path == "" {
		path = "unknown"
	}
	statusStr := strconv.Itoa(status)
	httpRequestsTotal.WithLabelValues(routeTag, method, path, statusStr).Inc()
	httpRequestDuration.WithLabelValues(routeTag, method, path).Observe(duration.Seconds())
}

func ObserveUpstreamRequest(channelId, channelType int, status int, duration time.Duration) {
	idStr := strconv.Itoa(channelId)
	typeStr := strconv.Itoa(channelType)
	statusStr := strconv.Itoa(status)
	relayUpstreamRequestsTotal.WithLabelValues(idStr, typeStr, statusStr).Inc()
	relayUpstreamLatency.WithLabelValues(idStr, typeStr).Observe(duration.Seconds())
}

func IncRelayRetry(channelId, channelType int) {
	idStr := strconv.Itoa(channelId)
	typeStr := strconv.Itoa(channelType)
	relayRetriesTotal.WithLabelValues(idStr, typeStr).Inc()
}

func IncRateLimitHit(scope, mark string) {
	if scope == "" {
		scope = "unknown"
	}
	if mark == "" {
		mark = "unknown"
	}
	rateLimitHitsTotal.WithLabelValues(scope, mark).Inc()
}
