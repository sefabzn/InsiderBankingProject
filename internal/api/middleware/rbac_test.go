package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/auth"
	"github.com/sefa-b/go-banking-sim/internal/domain"
)

func TestRequireRole(t *testing.T) {
	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Create middleware that requires admin role
	adminOnlyHandler := RequireRole(string(domain.RoleAdmin))(testHandler)

	tests := []struct {
		name           string
		userRole       string
		hasUser        bool
		expectedStatus int
	}{
		{
			name:           "admin user should access admin route",
			userRole:       string(domain.RoleAdmin),
			hasUser:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "regular user should not access admin route",
			userRole:       string(domain.RoleUser),
			hasUser:        true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no user in context should be forbidden",
			userRole:       "",
			hasUser:        false,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/admin", nil)

			if tt.hasUser {
				// Add user to context
				claims := &auth.Claims{
					UserID:   uuid.New(),
					Username: "testuser",
					Email:    "test@example.com",
					Role:     tt.userRole,
				}
				ctx := context.WithValue(req.Context(), UserContextKey, claims)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			adminOnlyHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("admin area"))
	})

	// Wrap with RequireAdmin middleware
	adminHandler := RequireAdmin(testHandler)

	t.Run("admin user should access admin route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)

		// Add admin user to context
		claims := &auth.Claims{
			UserID:   uuid.New(),
			Username: "admin",
			Email:    "admin@example.com",
			Role:     string(domain.RoleAdmin),
		}
		ctx := context.WithValue(req.Context(), UserContextKey, claims)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		adminHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	t.Run("non-admin user should not access admin route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)

		// Add regular user to context
		claims := &auth.Claims{
			UserID:   uuid.New(),
			Username: "user",
			Email:    "user@example.com",
			Role:     string(domain.RoleUser),
		}
		ctx := context.WithValue(req.Context(), UserContextKey, claims)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		adminHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rr.Code)
		}

		// Check response message
		body := rr.Body.String()
		if !contains(body, "insufficient permissions") {
			t.Errorf("Expected 'insufficient permissions' in response: %s", body)
		}
	})
}

func TestRequireUserRole(t *testing.T) {
	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user area"))
	})

	// Wrap with RequireUserRole middleware
	userHandler := RequireUserRole(testHandler)

	tests := []struct {
		name           string
		userRole       string
		expectedStatus int
	}{
		{
			name:           "admin should access user route",
			userRole:       string(domain.RoleAdmin),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user should access user route",
			userRole:       string(domain.RoleUser),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid role should not access user route",
			userRole:       "invalid_role",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/user", nil)

			claims := &auth.Claims{
				UserID:   uuid.New(),
				Username: "testuser",
				Email:    "test@example.com",
				Role:     tt.userRole,
			}
			ctx := context.WithValue(req.Context(), UserContextKey, claims)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			userHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestRequireOwnershipOrAdmin(t *testing.T) {
	// Mock function to extract user ID from request path
	getUserIDFromRequest := func(r *http.Request) string {
		// In real implementation, this would extract from URL params
		return r.Header.Get("X-Target-User-ID")
	}

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("resource accessed"))
	})

	// Wrap with ownership middleware
	ownershipHandler := RequireOwnershipOrAdmin(getUserIDFromRequest)(testHandler)

	userID := uuid.New()
	otherUserID := uuid.New()

	tests := []struct {
		name           string
		userRole       string
		currentUserID  uuid.UUID
		targetUserID   string
		expectedStatus int
	}{
		{
			name:           "admin can access any resource",
			userRole:       string(domain.RoleAdmin),
			currentUserID:  userID,
			targetUserID:   otherUserID.String(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user can access own resource",
			userRole:       string(domain.RoleUser),
			currentUserID:  userID,
			targetUserID:   userID.String(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user cannot access other's resource",
			userRole:       string(domain.RoleUser),
			currentUserID:  userID,
			targetUserID:   otherUserID.String(),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "missing target user ID should be forbidden",
			userRole:       string(domain.RoleUser),
			currentUserID:  userID,
			targetUserID:   "",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/resource", nil)
			req.Header.Set("X-Target-User-ID", tt.targetUserID)

			claims := &auth.Claims{
				UserID:   tt.currentUserID,
				Username: "testuser",
				Email:    "test@example.com",
				Role:     tt.userRole,
			}
			ctx := context.WithValue(req.Context(), UserContextKey, claims)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			ownershipHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestRoleHelperFunctions(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name    string
		role    string
		hasRole map[string]bool
		isAdmin bool
		isUser  bool
	}{
		{
			name: "admin user",
			role: string(domain.RoleAdmin),
			hasRole: map[string]bool{
				string(domain.RoleAdmin): true,
				string(domain.RoleUser):  false,
			},
			isAdmin: true,
			isUser:  true,
		},
		{
			name: "regular user",
			role: string(domain.RoleUser),
			hasRole: map[string]bool{
				string(domain.RoleAdmin): false,
				string(domain.RoleUser):  true,
			},
			isAdmin: false,
			isUser:  true,
		},
		{
			name: "invalid role",
			role: "invalid_role",
			hasRole: map[string]bool{
				string(domain.RoleAdmin): false,
				string(domain.RoleUser):  false,
			},
			isAdmin: false,
			isUser:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)

			claims := &auth.Claims{
				UserID:   userID,
				Username: "testuser",
				Email:    "test@example.com",
				Role:     tt.role,
			}
			ctx := context.WithValue(req.Context(), UserContextKey, claims)
			req = req.WithContext(ctx)

			// Test HasRole
			for role, expected := range tt.hasRole {
				if HasRole(req, role) != expected {
					t.Errorf("HasRole(%s) = %v, expected %v", role, HasRole(req, role), expected)
				}
			}

			// Test IsAdmin
			if IsAdmin(req) != tt.isAdmin {
				t.Errorf("IsAdmin() = %v, expected %v", IsAdmin(req), tt.isAdmin)
			}

			// Test IsUser
			if IsUser(req) != tt.isUser {
				t.Errorf("IsUser() = %v, expected %v", IsUser(req), tt.isUser)
			}

			// Test GetCurrentUserID
			currentUserID, ok := GetCurrentUserID(req)
			if !ok {
				t.Error("GetCurrentUserID() should return true for authenticated user")
			}
			if currentUserID != userID.String() {
				t.Errorf("GetCurrentUserID() = %s, expected %s", currentUserID, userID.String())
			}
		})
	}

	t.Run("no user in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		// Test helper functions with no user context
		if HasRole(req, string(domain.RoleAdmin)) {
			t.Error("HasRole should return false with no user context")
		}
		if IsAdmin(req) {
			t.Error("IsAdmin should return false with no user context")
		}
		if IsUser(req) {
			t.Error("IsUser should return false with no user context")
		}

		_, ok := GetCurrentUserID(req)
		if ok {
			t.Error("GetCurrentUserID should return false with no user context")
		}
	})
}
