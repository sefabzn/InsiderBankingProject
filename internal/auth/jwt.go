package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType represents the type of JWT token.
type TokenType string

const (
	// AccessToken represents access token type
	AccessToken TokenType = "access"
	// RefreshToken represents refresh token type
	RefreshToken TokenType = "refresh"
)

// Token durations
const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
)

// Claims represents JWT claims structure.
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Type     TokenType `json:"type"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations.
type JWTManager struct {
	secretKey []byte
	issuer    string
}

// NewJWTManager creates a new JWT manager.
func NewJWTManager(secretKey, issuer string) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
		issuer:    issuer,
	}
}

// GenerateAccessToken generates an access token for a user.
func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, username, email, role string) (string, error) {
	return m.generateToken(userID, username, email, role, AccessToken, AccessTokenDuration)
}

// GenerateRefreshToken generates a refresh token for a user.
func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID, username, email, role string) (string, error) {
	return m.generateToken(userID, username, email, role, RefreshToken, RefreshTokenDuration)
}

// generateToken generates a JWT token with specified parameters.
func (m *JWTManager) generateToken(userID uuid.UUID, username, email, role string, tokenType TokenType, duration time.Duration) (string, error) {
	now := time.Now()

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Role:     role,
		Type:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
			Subject:   userID.String(),
			Audience:  []string{"go-banking-sim"},
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims.
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return claims, nil
}

// ValidateAccessToken validates an access token specifically.
func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != AccessToken {
		return nil, fmt.Errorf("token is not an access token")
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token specifically.
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != RefreshToken {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token from a valid refresh token.
func (m *JWTManager) RefreshAccessToken(refreshTokenString string) (string, error) {
	claims, err := m.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new access token with same user info
	return m.GenerateAccessToken(claims.UserID, claims.Username, claims.Email, claims.Role)
}

// GetUserFromToken extracts user information from a valid token.
func (m *JWTManager) GetUserFromToken(tokenString string) (*Claims, error) {
	return m.ValidateToken(tokenString)
}

// IsTokenExpired checks if a token is expired without validating signature.
// This is useful for determining if a token needs refresh.
func (m *JWTManager) IsTokenExpired(tokenString string) bool {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(_ *jwt.Token) (interface{}, error) {
		return m.secretKey, nil
	})

	if err != nil {
		return true
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return true
	}

	return claims.ExpiresAt.Time.Before(time.Now())
}

// TokenPair represents an access and refresh token pair.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// GenerateTokenPair generates both access and refresh tokens.
func (m *JWTManager) GenerateTokenPair(userID uuid.UUID, username, email, role string) (*TokenPair, error) {
	accessToken, err := m.GenerateAccessToken(userID, username, email, role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := m.GenerateRefreshToken(userID, username, email, role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(AccessTokenDuration.Seconds()),
	}, nil
}
