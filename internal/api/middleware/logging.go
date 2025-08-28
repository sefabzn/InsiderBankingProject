// Package middleware provides HTTP middleware functions.
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/utils"
	"go.opentelemetry.io/otel/trace"
)

// MetricsMiddleware creates middleware that records HTTP request metrics.
func MetricsMiddleware(metricsCollector *utils.MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Record metrics (skip /healthz and /metrics endpoints to avoid recursion)
			if r.URL.Path != "/healthz" && r.URL.Path != "/metrics" {
				metricsCollector.RecordHTTPRequest(r.Method, r.URL.Path, rw.statusCode, duration)
			}
		})
	}
}

// LoggingMiddleware creates middleware that logs HTTP requests with structured logging.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate request ID
		requestID := uuid.New().String()

		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Add request ID to request context for downstream use
		ctx := r.Context()
		ctx = context.WithValue(ctx, "request_id", requestID)
		r = r.WithContext(ctx)

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Calculate duration
		duration := time.Since(start)

		// Log the request
		utils.Info("http_request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"user_agent", r.Header.Get("User-Agent"),
			"remote_addr", r.RemoteAddr,
			"status", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"duration", duration.String(),
		)
	})
}

// TracingMiddleware adds distributed tracing support with trace ID in headers.
func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get current span context
			ctx := r.Context()
			span := trace.SpanFromContext(ctx)

			// Add trace ID to response headers for debugging
			if span != nil {
				traceID := span.SpanContext().TraceID().String()
				if traceID != "" {
					w.Header().Set("X-Trace-ID", traceID)
				}
			}

			// Call next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
