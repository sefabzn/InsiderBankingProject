package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/auth"
	"github.com/sefa-b/go-banking-sim/internal/domain"
	"github.com/sefa-b/go-banking-sim/internal/repository"
	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// authService implements the AuthService interface.
type authService struct {
	repos      *repository.Repositories
	jwtManager *auth.JWTManager
	eventSvc   *EventService // Event service for publishing domain events
}

// NewAuthService creates a new authentication service.
func NewAuthService(repos *repository.Repositories, jwtManager *auth.JWTManager, eventSvc *EventService) AuthService {
	return &authService{
		repos:      repos,
		jwtManager: jwtManager,
		eventSvc:   eventSvc,
	}
}

// Register creates a new user account with an initial balance.
func (s *authService) Register(ctx context.Context, req *domain.CreateUserRequest) (*domain.UserResponse, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if email already exists
	existingUser, err := s.repos.Users.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("email already registered")
	}

	// Check if username already exists
	existingUser, err = s.repos.Users.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("username already taken")
	}

	// Hash the password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Set default role if not provided
	role := req.Role
	if role == "" {
		role = string(domain.RoleUser)
	}

	// Create user directly in database for immediate availability
	user := &domain.User{
		Username:     req.Username,
		Email:        strings.ToLower(req.Email),
		PasswordHash: hashedPassword,
		Role:         role,
		IsActive:     true,
	}

	if err := s.repos.Users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create initial balance directly
	balance := &domain.Balance{
		UserID:   user.ID,
		Amount:   0.00,
		Currency: "USD",
	}

	if err := s.repos.Balances.Upsert(ctx, balance); err != nil {
		return nil, fmt.Errorf("failed to create initial balance: %w", err)
	}

	// Publish domain events for event sourcing
	if s.eventSvc != nil {
		// Publish UserRegistered event
		if err := s.eventSvc.UserRegistered(ctx, user); err != nil {
			utils.Error("failed to publish UserRegistered event",
				"user_id", user.ID,
				"error", err.Error(),
			)
		}

		// Publish BalanceInitialized event
		if err := s.eventSvc.AmountCredited(ctx, user.ID, 0.00, "USD", uuid.Nil, "initial_balance"); err != nil {
			utils.Error("failed to publish BalanceInitialized event",
				"user_id", user.ID,
				"error", err.Error(),
			)
		}
	}

	// Log the registration for audit
	if s.repos.Audit != nil {
		auditDetails := map[string]interface{}{
			"user_id":  user.ID,
			"username": req.Username,
			"email":    strings.ToLower(req.Email),
			"role":     role,
		}
		if err := s.repos.Audit.Log(ctx, "user", user.ID, "register", auditDetails); err != nil {
			utils.Error("failed to log registration audit",
				"user_id", user.ID,
				"error", err.Error(),
			)
		}
	}

	// Create response from the created user
	response := user.ToResponse()

	return &response, nil
}

// Login authenticates a user and returns tokens.
func (s *authService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	// Get user by email
	user, err := s.repos.Users.GetByEmail(ctx, strings.ToLower(email))
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Verify password
	if !auth.ComparePassword(user.PasswordHash, password) {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Generate token pair
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Log the login for audit
	if s.repos.Audit != nil {
		auditDetails := map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
		}
		if err := s.repos.Audit.Log(ctx, "user", user.ID, "login", auditDetails); err != nil {
			utils.Error("failed to log login audit",
				"user_id", user.ID,
				"error", err.Error(),
			)
		}
	}

	userResponse := user.ToResponse()
	return &LoginResponse{
		User:         &userResponse,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int(tokenPair.ExpiresIn),
	}, nil
}

// RefreshToken generates a new access token from a refresh token.
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	// Generate new access token
	newAccessToken, err := s.jwtManager.RefreshAccessToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken: newAccessToken,
		ExpiresIn:   int(auth.AccessTokenDuration.Seconds()),
	}, nil
}

// ValidateToken validates an access token and returns user info.
func (s *authService) ValidateToken(ctx context.Context, token string) (*domain.UserResponse, error) {
	claims, err := s.jwtManager.ValidateAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Get fresh user data from database
	user, err := s.repos.Users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	response := user.ToResponse()
	return &response, nil
}

// Logout invalidates a refresh token.
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	// For MVP, we don't maintain a blacklist of tokens
	// In production, you would store invalidated tokens in Redis or database
	// For now, we just validate that the token is valid before "invalidating" it

	_, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return fmt.Errorf("invalid refresh token: %w", err)
	}

	// Log the logout for audit
	if s.repos.Audit != nil {
		claims, _ := s.jwtManager.ValidateRefreshToken(refreshToken)
		if claims != nil {
			auditDetails := map[string]interface{}{
				"user_id": claims.UserID,
			}
			if err := s.repos.Audit.Log(ctx, "user", claims.UserID, "logout", auditDetails); err != nil {
				utils.Error("failed to log logout audit",
					"user_id", claims.UserID,
					"error", err.Error(),
				)
			}
		}
	}

	return nil
}
