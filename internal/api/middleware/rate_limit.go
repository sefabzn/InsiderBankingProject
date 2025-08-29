// Package middleware provides HTTP middleware functions.
package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sefa-b/go-banking-sim/internal/service"
)

// RateLimitMiddleware creates middleware that enforces rate limits using Redis
func RateLimitMiddleware(cacheService service.CacheService, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			clientIP := getClientIP(r)

			// Check rate limit
			if cacheService != nil {
				allowed, err := cacheService.CheckRateLimit(r.Context(), clientIP, maxRequests, window)
				if err != nil {
					// Log error but allow request to proceed
					// In production, you might want to handle this differently
					next.ServeHTTP(w, r)
					return
				}

				if !allowed {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
					w.Header().Set("X-RateLimit-Window", window.String())
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte(`{"error":"Rate limit exceeded","code":429,"retry_after":"` + window.String() + `"}`))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (common with proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if strings.Contains(ip, ":") {
		ip, _, _ = strings.Cut(ip, ":")
	}
	return ip
}
