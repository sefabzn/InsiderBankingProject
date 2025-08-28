package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/auth"
)

func TestAuthMiddleware(t *testing.T) {
	// Setup JWT manager
	jwtManager := auth.NewJWTManager("test-secret", "test-issuer")

	// Setup test user
	userID := uuid.New()
	username := "testuser"
	email := "test@example.com"
	role := "user"

	// Generate valid token
	validToken, err := jwtManager.GenerateAccessToken(userID, username, email, role)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Create test handler that returns user info
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetUserFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("no user in context"))
			return
		}

		w.WriteHeader(http.StatusOK)
		response := fmt.Sprintf(`{"user_id":"%s","username":"%s","email":"%s","role":"%s"}`,
			claims.UserID, claims.Username, claims.Email, claims.Role)
		w.Write([]byte(response))
	})

	// Wrap with auth middleware
	authMiddleware := AuthMiddleware(jwtManager)
	protectedHandler := authMiddleware(testHandler)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectUserData bool
	}{
		{
			name:           "valid bearer token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectUserData: true,
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectUserData: false,
		},
		{
			name:           "invalid header format",
			authHeader:     "InvalidFormat " + validToken,
			expectedStatus: http.StatusUnauthorized,
			expectUserData: false,
		},
		{
			name:           "bearer without token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectUserData: false,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			expectUserData: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			protectedHandler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check response for successful authentication
			if tt.expectUserData {
				body := rr.Body.String()
				if body == "" {
					t.Error("Expected user data in response body")
				}

				// Should contain user information
				if !contains(body, username) || !contains(body, email) {
					t.Errorf("Response should contain user data: %s", body)
				}
			}
		})
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	// Setup JWT manager
	jwtManager := auth.NewJWTManager("test-secret", "test-issuer")

	// Generate valid token
	userID := uuid.New()
	validToken, err := jwtManager.GenerateAccessToken(userID, "testuser", "test@example.com", "user")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Create test handler that checks for user context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetUserFromContext(r.Context())
		if ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"authenticated":true,"user_id":"` + claims.UserID.String() + `"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"authenticated":false}`))
		}
	})

	// Wrap with optional auth middleware
	optionalAuthMiddleware := OptionalAuthMiddleware(jwtManager)
	handler := optionalAuthMiddleware(testHandler)

	tests := []struct {
		name       string
		authHeader string
		expectAuth bool
	}{
		{
			name:       "valid token should authenticate",
			authHeader: "Bearer " + validToken,
			expectAuth: true,
		},
		{
			name:       "no token should not authenticate but allow access",
			authHeader: "",
			expectAuth: false,
		},
		{
			name:       "invalid token should not authenticate but allow access",
			authHeader: "Bearer invalid.token",
			expectAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/optional", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Should always return 200
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}

			body := rr.Body.String()
			if tt.expectAuth {
				if !contains(body, `"authenticated":true`) {
					t.Errorf("Expected authenticated:true in response: %s", body)
				}
			} else {
				if !contains(body, `"authenticated":false`) {
					t.Errorf("Expected authenticated:false in response: %s", body)
				}
			}
		})
	}
}

func TestRequireUser(t *testing.T) {
	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with RequireUser middleware
	protectedHandler := RequireUser(testHandler)

	t.Run("no user in context should return 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		rr := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("user in context should allow access", func(t *testing.T) {
		// Create request with user context
		req := httptest.NewRequest("GET", "/protected", nil)

		// Add user claims to context
		claims := &auth.Claims{
			UserID:   uuid.New(),
			Username: "testuser",
			Email:    "test@example.com",
			Role:     "user",
		}
		ctx := req.Context()
		ctx = context.WithValue(ctx, UserContextKey, claims)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}

func TestGetUserFromContext(t *testing.T) {
	userID := uuid.New()
	claims := &auth.Claims{
		UserID:   userID,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}

	t.Run("context with user should return claims", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserContextKey, claims)

		retrievedClaims, ok := GetUserFromContext(ctx)
		if !ok {
			t.Error("Expected to find user in context")
		}

		if retrievedClaims.UserID != userID {
			t.Errorf("Expected UserID %v, got %v", userID, retrievedClaims.UserID)
		}
	})

	t.Run("context without user should return false", func(t *testing.T) {
		ctx := context.Background()

		_, ok := GetUserFromContext(ctx)
		if ok {
			t.Error("Expected not to find user in empty context")
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || s[0:len(substr)] == substr || contains(s[1:], substr))
}
