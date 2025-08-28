// Package utils provides utility functions including metrics collection.
package utils

import (
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics
var (
	transactionsProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "banking_transactions_processed_total",
		Help: "Total number of transactions processed",
	})

	transactionQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "banking_transaction_queue_depth",
		Help: "Current depth of the transaction processing queue",
	})

	// activeGoroutines is used by Prometheus for monitoring active goroutines
	//nolint:unused // Used by Prometheus metrics collection
	activeGoroutines = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "banking_goroutines_active",
		Help: "Number of active goroutines",
	}, func() float64 {
		return float64(runtime.NumGoroutine())
	})

	// uptimeSeconds is used by Prometheus for monitoring application uptime
	//nolint:unused // Used by Prometheus metrics collection
	uptimeSeconds = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "banking_uptime_seconds",
		Help: "Application uptime in seconds",
	}, func() float64 {
		// This will be set by the collector
		return 0
	})

	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "banking_http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "endpoint", "status_code"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "banking_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "endpoint"})
)

// MetricsCollector collects basic application metrics.
type MetricsCollector struct {
	startTime             time.Time
	transactionsProcessed int64
	queueDepth            int64
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

// IncrementTransactionsProcessed increments the transaction counter.
func (m *MetricsCollector) IncrementTransactionsProcessed() {
	atomic.AddInt64(&m.transactionsProcessed, 1)
	transactionsProcessedTotal.Inc()
}

// SetQueueDepth sets the current queue depth.
func (m *MetricsCollector) SetQueueDepth(depth int) {
	atomic.StoreInt64(&m.queueDepth, int64(depth))
	transactionQueueDepth.Set(float64(depth))
}

// RecordHTTPRequest records an HTTP request metric.
func (m *MetricsCollector) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	httpRequestsTotal.WithLabelValues(method, endpoint, strconv.Itoa(statusCode)).Inc()
	httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// GetMetrics returns the current metrics as a JSON-serializable struct.
func (m *MetricsCollector) GetMetrics() *Metrics {
	return &Metrics{
		Uptime:                time.Since(m.startTime).String(),
		UptimeSeconds:         int64(time.Since(m.startTime).Seconds()),
		Goroutines:            runtime.NumGoroutine(),
		QueueDepth:            atomic.LoadInt64(&m.queueDepth),
		TransactionsProcessed: atomic.LoadInt64(&m.transactionsProcessed),
	}
}

// Metrics represents the application metrics.
type Metrics struct {
	Uptime                string `json:"uptime"`
	UptimeSeconds         int64  `json:"uptime_seconds"`
	Goroutines            int    `json:"goroutines"`
	QueueDepth            int64  `json:"queue_depth"`
	TransactionsProcessed int64  `json:"transactions_processed"`
}
