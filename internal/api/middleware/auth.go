// Package middleware provides HTTP middleware functions.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/sefa-b/go-banking-sim/internal/auth"
)

// ContextKey is a type for context keys to avoid collisions.
type ContextKey string

const (
	// UserContextKey is the context key for storing user claims.
	UserContextKey ContextKey = "user"
)

// AuthMiddleware creates middleware that validates JWT tokens from Authorization header.
func AuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeUnauthorized(w, "missing authorization header")
				return
			}

			// Check Bearer prefix
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				writeUnauthorized(w, "invalid authorization header format")
				return
			}

			// Extract token
			token := strings.TrimPrefix(authHeader, bearerPrefix)
			if token == "" {
				writeUnauthorized(w, "missing token")
				return
			}

			// Validate token
			claims, err := jwtManager.ValidateAccessToken(token)
			if err != nil {
				writeUnauthorized(w, "invalid token")
				return
			}

			// Add user claims to request context
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			r = r.WithContext(ctx)

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuthMiddleware creates middleware that validates JWT tokens but doesn't require them.
// If a valid token is provided, user claims are added to context.
// If no token or invalid token, request continues without user context.
func OptionalAuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				const bearerPrefix = "Bearer "
				if strings.HasPrefix(authHeader, bearerPrefix) {
					token := strings.TrimPrefix(authHeader, bearerPrefix)
					if token != "" {
						// Try to validate token
						if claims, err := jwtManager.ValidateAccessToken(token); err == nil {
							// Add user claims to request context if valid
							ctx := context.WithValue(r.Context(), UserContextKey, claims)
							r = r.WithContext(ctx)
						}
					}
				}
			}

			// Continue to next handler regardless of token validation result
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext extracts user claims from request context.
func GetUserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*auth.Claims)
	return claims, ok
}

// RequireUser middleware ensures a user is authenticated.
// Should be used after AuthMiddleware.
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetUserFromContext(r.Context())
		if !ok {
			writeUnauthorized(w, "authentication required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeUnauthorized writes a 401 Unauthorized response.
func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	response := `{"error":"` + message + `","code":401}`
	_, _ = w.Write([]byte(response))
}
