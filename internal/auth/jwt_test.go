package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTManager(t *testing.T) {
	secretKey := "test-secret-key-for-jwt-signing"
	issuer := "go-banking-sim-test"
	manager := NewJWTManager(secretKey, issuer)

	userID := uuid.New()
	username := "testuser"
	email := "test@example.com"
	role := "user"

	t.Run("generate and validate access token", func(t *testing.T) {
		// Generate access token
		token, err := manager.GenerateAccessToken(userID, username, email, role)
		if err != nil {
			t.Fatalf("Failed to generate access token: %v", err)
		}

		if token == "" {
			t.Error("Token should not be empty")
		}

		// Validate access token
		claims, err := manager.ValidateAccessToken(token)
		if err != nil {
			t.Fatalf("Failed to validate access token: %v", err)
		}

		// Check claims
		if claims.UserID != userID {
			t.Errorf("Expected UserID %v, got %v", userID, claims.UserID)
		}
		if claims.Username != username {
			t.Errorf("Expected Username %v, got %v", username, claims.Username)
		}
		if claims.Email != email {
			t.Errorf("Expected Email %v, got %v", email, claims.Email)
		}
		if claims.Role != role {
			t.Errorf("Expected Role %v, got %v", role, claims.Role)
		}
		if claims.Type != AccessToken {
			t.Errorf("Expected Type %v, got %v", AccessToken, claims.Type)
		}
	})

	t.Run("generate and validate refresh token", func(t *testing.T) {
		// Generate refresh token
		token, err := manager.GenerateRefreshToken(userID, username, email, role)
		if err != nil {
			t.Fatalf("Failed to generate refresh token: %v", err)
		}

		// Validate refresh token
		claims, err := manager.ValidateRefreshToken(token)
		if err != nil {
			t.Fatalf("Failed to validate refresh token: %v", err)
		}

		if claims.Type != RefreshToken {
			t.Errorf("Expected Type %v, got %v", RefreshToken, claims.Type)
		}
	})

	t.Run("token type validation", func(t *testing.T) {
		accessToken, _ := manager.GenerateAccessToken(userID, username, email, role)
		refreshToken, _ := manager.GenerateRefreshToken(userID, username, email, role)

		// Access token should fail refresh token validation
		_, err := manager.ValidateRefreshToken(accessToken)
		if err == nil {
			t.Error("Access token should fail refresh token validation")
		}

		// Refresh token should fail access token validation
		_, err = manager.ValidateAccessToken(refreshToken)
		if err == nil {
			t.Error("Refresh token should fail access token validation")
		}
	})

	t.Run("token refresh", func(t *testing.T) {
		// Generate refresh token
		refreshToken, err := manager.GenerateRefreshToken(userID, username, email, role)
		if err != nil {
			t.Fatalf("Failed to generate refresh token: %v", err)
		}

		// Refresh access token
		newAccessToken, err := manager.RefreshAccessToken(refreshToken)
		if err != nil {
			t.Fatalf("Failed to refresh access token: %v", err)
		}

		// Validate new access token
		claims, err := manager.ValidateAccessToken(newAccessToken)
		if err != nil {
			t.Fatalf("Failed to validate refreshed access token: %v", err)
		}

		if claims.UserID != userID {
			t.Error("Refreshed token should have same user ID")
		}
	})

	t.Run("generate token pair", func(t *testing.T) {
		pair, err := manager.GenerateTokenPair(userID, username, email, role)
		if err != nil {
			t.Fatalf("Failed to generate token pair: %v", err)
		}

		if pair.AccessToken == "" {
			t.Error("Access token should not be empty")
		}
		if pair.RefreshToken == "" {
			t.Error("Refresh token should not be empty")
		}
		if pair.ExpiresIn != int64(AccessTokenDuration.Seconds()) {
			t.Errorf("Expected ExpiresIn %d, got %d", int64(AccessTokenDuration.Seconds()), pair.ExpiresIn)
		}

		// Validate both tokens
		_, err = manager.ValidateAccessToken(pair.AccessToken)
		if err != nil {
			t.Errorf("Access token from pair should be valid: %v", err)
		}

		_, err = manager.ValidateRefreshToken(pair.RefreshToken)
		if err != nil {
			t.Errorf("Refresh token from pair should be valid: %v", err)
		}
	})
}

func TestJWTExpiration(t *testing.T) {
	secretKey := "test-secret-key"
	issuer := "test-issuer"
	manager := NewJWTManager(secretKey, issuer)

	userID := uuid.New()
	username := "testuser"
	email := "test@example.com"
	role := "user"

	t.Run("token expiration detection", func(t *testing.T) {
		// Create a manager with very short token duration for testing
		shortManager := &JWTManager{
			secretKey: []byte(secretKey),
			issuer:    issuer,
		}

		// Generate token with 1 millisecond duration
		token, err := shortManager.generateToken(userID, username, email, role, AccessToken, 1*time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to generate short-lived token: %v", err)
		}

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		// Token should be expired
		if !manager.IsTokenExpired(token) {
			t.Error("Token should be detected as expired")
		}

		// Validation should fail
		_, err = manager.ValidateToken(token)
		if err == nil {
			t.Error("Expired token validation should fail")
		}
	})
}

func TestJWTInvalidTokens(t *testing.T) {
	secretKey := "test-secret-key"
	issuer := "test-issuer"
	manager := NewJWTManager(secretKey, issuer)

	testCases := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "empty token",
			token:       "",
			expectError: true,
		},
		{
			name:        "invalid format",
			token:       "invalid.token.format",
			expectError: true,
		},
		{
			name:        "malformed jwt",
			token:       "not.a.jwt",
			expectError: true,
		},
		{
			name:        "valid format but wrong signature",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.ValidateToken(tc.token)
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got error: %v", tc.expectError, err != nil)
			}
		})
	}
}

func TestJWTRoundTrip(t *testing.T) {
	secretKey := "test-secret-key-for-round-trip"
	issuer := "go-banking-sim"
	manager := NewJWTManager(secretKey, issuer)

	userID := uuid.New()
	username := "roundtripuser"
	email := "roundtrip@example.com"
	role := "admin"

	// Generate token
	token, err := manager.GenerateAccessToken(userID, username, email, role)
	if err != nil {
		t.Fatalf("Token generation failed: %v", err)
	}

	// Validate token (round-trip)
	claims, err := manager.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("Token validation failed: %v", err)
	}

	// Verify all fields match
	if claims.UserID != userID {
		t.Errorf("UserID mismatch: expected %v, got %v", userID, claims.UserID)
	}
	if claims.Username != username {
		t.Errorf("Username mismatch: expected %v, got %v", username, claims.Username)
	}
	if claims.Email != email {
		t.Errorf("Email mismatch: expected %v, got %v", email, claims.Email)
	}
	if claims.Role != role {
		t.Errorf("Role mismatch: expected %v, got %v", role, claims.Role)
	}
	if claims.Type != AccessToken {
		t.Errorf("Type mismatch: expected %v, got %v", AccessToken, claims.Type)
	}

	t.Log("JWT round-trip test passed - all fields preserved correctly")
}
