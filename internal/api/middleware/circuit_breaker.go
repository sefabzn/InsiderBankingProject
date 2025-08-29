// Package middleware provides HTTP middleware for circuit breaker protection
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// CircuitBreakerMiddleware creates middleware that protects external service calls
func CircuitBreakerMiddleware(serviceName string, failureThreshold int32, resetTimeout time.Duration) func(http.Handler) http.Handler {
	config := utils.CircuitBreakerConfig{
		Name:             serviceName,
		FailureThreshold: failureThreshold,
		ResetTimeout:     resetTimeout,
		CallTimeout:      30 * time.Second, // Default call timeout
	}

	breaker := utils.GetCircuitBreaker(serviceName, config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if circuit breaker allows the request
			if breaker.GetState() == utils.StateOpen {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"Service temporarily unavailable","code":503,"service":"%s"}`, serviceName)))
				return
			}

			// Wrap the response writer to capture errors
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     200, // Default status
			}

			// Create a context that can be cancelled if circuit breaker opens
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()

			// Execute the request through circuit breaker
			err := breaker.Call(ctx, func(callCtx context.Context) error {
				// Create a new request with the circuit breaker context
				newReq := r.WithContext(callCtx)

				// Call the next handler
				next.ServeHTTP(wrapper, newReq)

				// Check if the response indicates a failure (5xx status codes)
				if wrapper.statusCode >= 500 {
					return fmt.Errorf("upstream service error: %d", wrapper.statusCode)
				}
				return nil
			})

			if err != nil {
				if cbErr, ok := err.(*utils.CircuitBreakerError); ok {
					// Circuit breaker is open
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"Service temporarily unavailable","code":503,"service":"%s","state":"%s"}`, serviceName, cbErr.State)))
					return
				}
				// Other error - log it and return 503
				utils.Error("circuit breaker call failed", "error", err.Error(), "service", serviceName)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"Service temporarily unavailable","code":503,"service":"%s"}`, serviceName)))
				return
			}
		})
	}
}

// responseWriterWrapper wraps http.ResponseWriter to capture status codes
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// CircuitBreakerMetricsHandler provides circuit breaker metrics endpoint
func CircuitBreakerMetricsHandler(w http.ResponseWriter, _ *http.Request) {
	metrics := utils.GetCircuitBreakerMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := `{"circuit_breakers":{`
	first := true
	for name, metric := range metrics {
		if !first {
			response += ","
		}
		response += fmt.Sprintf(`"%s":{"state":"%s","total_requests":%d,"total_failures":%d,"total_successes":%d,"current_failures":%d}`,
			name, metric.State.String(), metric.TotalRequests, metric.TotalFailures, metric.TotalSuccesses, metric.CurrentFailures)
		first = false
	}
	response += "}}"

	_, _ = w.Write([]byte(response))
}

// ExternalServiceCall performs an external service call with circuit breaker protection
func ExternalServiceCall(ctx context.Context, serviceName string, call func(context.Context) error) error {
	config := utils.CircuitBreakerConfig{
		Name:             serviceName,
		FailureThreshold: 5,                // Open after 5 failures
		ResetTimeout:     60 * time.Second, // Try to close after 60 seconds
		CallTimeout:      30 * time.Second, // Call timeout
	}

	breaker := utils.GetCircuitBreaker(serviceName, config)
	return breaker.Call(ctx, call)
}
