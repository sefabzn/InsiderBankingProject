package middleware

import (
	"net/http"

	"github.com/sefa-b/go-banking-sim/internal/domain"
)

// RequireRole creates middleware that requires a specific role.
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context (should be set by auth middleware)
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				writeForbidden(w, "authentication required")
				return
			}

			// Check if user has required role
			if claims.Role != requiredRole {
				writeForbidden(w, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin creates middleware that requires admin role.
func RequireAdmin(next http.Handler) http.Handler {
	return RequireRole(string(domain.RoleAdmin))(next)
}

// RequireUserRole creates middleware that requires user role (or higher).
func RequireUserRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		claims, ok := GetUserFromContext(r.Context())
		if !ok {
			writeForbidden(w, "authentication required")
			return
		}

		// Allow both user and admin roles
		if claims.Role != string(domain.RoleUser) && claims.Role != string(domain.RoleAdmin) {
			writeForbidden(w, "insufficient permissions")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireOwnershipOrAdmin creates middleware that requires the user to be either:
// 1. The owner of the resource (based on userID parameter), or
// 2. An admin
func RequireOwnershipOrAdmin(getUserIDFromRequest func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				writeForbidden(w, "authentication required")
				return
			}

			// Admins can access anything
			if claims.Role == string(domain.RoleAdmin) {
				next.ServeHTTP(w, r)
				return
			}

			// For non-admins, check ownership
			requestedUserID := getUserIDFromRequest(r)
			if requestedUserID == "" {
				writeForbidden(w, "invalid resource identifier")
				return
			}

			if claims.UserID.String() != requestedUserID {
				writeForbidden(w, "can only access your own resources")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// HasRole checks if the current user has a specific role.
func HasRole(r *http.Request, role string) bool {
	claims, ok := GetUserFromContext(r.Context())
	if !ok {
		return false
	}
	return claims.Role == role
}

// IsAdmin checks if the current user is an admin.
func IsAdmin(r *http.Request) bool {
	return HasRole(r, string(domain.RoleAdmin))
}

// IsUser checks if the current user has user role (or admin).
func IsUser(r *http.Request) bool {
	claims, ok := GetUserFromContext(r.Context())
	if !ok {
		return false
	}
	return claims.Role == string(domain.RoleUser) || claims.Role == string(domain.RoleAdmin)
}

// GetCurrentUserID returns the current user's ID from the context.
func GetCurrentUserID(r *http.Request) (string, bool) {
	claims, ok := GetUserFromContext(r.Context())
	if !ok {
		return "", false
	}
	return claims.UserID.String(), true
}

// writeForbidden writes a 403 Forbidden response.
func writeForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	response := `{"error":"` + message + `","code":403}`
	_, _ = w.Write([]byte(response))
}
