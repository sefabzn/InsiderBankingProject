// Package middleware provides HTTP middleware functions.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/utils"
	"go.opentelemetry.io/otel/attribute"
)

// contextKey represents a context key type
type contextKey string

// requestIDKey is the context key for request ID
const requestIDKey contextKey = "request_id"

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
		ctx = context.WithValue(ctx, requestIDKey, requestID)
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

// TracingMiddleware adds distributed tracing support with span creation and trace ID in headers.
func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a new span for this request
			ctx := r.Context()
			tracer := utils.GetTracer(serviceName)
			ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
			defer span.End()

			// Add request information to the span
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.user_agent", r.Header.Get("User-Agent")),
				attribute.String("http.remote_addr", r.RemoteAddr),
			)

			// Add trace ID to response headers for debugging
			traceID := span.SpanContext().TraceID().String()
			if traceID != "" {
				w.Header().Set("X-Trace-ID", traceID)
			}

			// Call next handler with the span context
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
