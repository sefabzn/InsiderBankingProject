package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/domain"
	"github.com/sefa-b/go-banking-sim/internal/repository"
	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// UserServiceImpl implements the UserService interface.
type UserServiceImpl struct {
	repos *repository.Repositories
	cache CacheService // Optional cache service
}

// NewUserService creates a new user service.
func NewUserService(repos *repository.Repositories) UserService {
	return &UserServiceImpl{
		repos: repos,
		cache: nil, // Will be set later if cache is available
	}
}

// SetCacheService sets the cache service for this user service
func (s *UserServiceImpl) SetCacheService(cache CacheService) {
	s.cache = cache
}

// GetByID retrieves a user by ID.
func (s *UserServiceImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.UserResponse, error) {
	// Try cache first if available
	if s.cache != nil {
		cachedUser, err := s.cache.GetCachedUser(ctx, id)
		if err == nil {
			utils.Info("cache hit for user", "user_id", id.String())
			return cachedUser, nil
		}
		// Cache miss or error - continue to database
		utils.Info("cache miss for user", "user_id", id.String())
	}

	user, err := s.repos.Users.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	response := user.ToResponse()

	// Cache the result if cache is available
	if s.cache != nil {
		if err := s.cache.CacheUser(ctx, user); err != nil {
			utils.Error("failed to cache user", "user_id", id.String(), "error", err.Error())
			// Don't fail the request if caching fails
		}
	}

	return &response, nil
}

// List retrieves users with pagination (admin only).
func (s *UserServiceImpl) List(ctx context.Context, limit, offset int) ([]*domain.UserResponse, error) {
	// If limit is 0 or negative, it means no limit should be applied.
	// The repository layer will handle this appropriately.
	if offset < 0 {
		offset = 0
	}

	users, err := s.repos.Users.ListPaginated(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert to response format
	responses := make([]*domain.UserResponse, len(users))
	for i, user := range users {
		response := user.ToResponse()
		responses[i] = &response
	}

	return responses, nil
}

// Update updates user information.
func (s *UserServiceImpl) Update(ctx context.Context, id uuid.UUID, req *domain.UpdateUserRequest) (*domain.UserResponse, error) {
	// Get existing user
	user, err := s.repos.Users.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update fields if provided
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	// Validate updated user
	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Update in database
	if err := s.repos.Users.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Invalidate cache after successful update
	if s.cache != nil {
		if err := s.cache.InvalidateUserCache(ctx, user.ID); err != nil {
			utils.Error("failed to invalidate user cache", "user_id", user.ID.String(), "error", err.Error())
			// Don't fail the request if cache invalidation fails
		}
	}

	// Log the update for audit
	if s.repos.Audit != nil {
		auditDetails := map[string]interface{}{
			"user_id":   user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"is_active": user.IsActive,
		}
		if err := s.repos.Audit.Log(ctx, "user", user.ID, "update", auditDetails); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("failed to log user update audit: %v\n", err)
		}
	}

	response := user.ToResponse()

	// Cache the updated user
	if s.cache != nil {
		if err := s.cache.CacheUser(ctx, user); err != nil {
			utils.Error("failed to cache updated user", "user_id", user.ID.String(), "error", err.Error())
		}
	}

	return &response, nil
}

// Delete deletes a user account.
func (s *UserServiceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if user exists and get user details for audit
	user, err := s.repos.Users.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Soft delete user
	if err := s.repos.Users.Delete(ctx, id); err != nil {
		if err.Error() == "user not found or already inactive" {
			return fmt.Errorf("user not found: %w", err)
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Invalidate cache after successful deletion
	if s.cache != nil {
		if err := s.cache.InvalidateUserCache(ctx, id); err != nil {
			utils.Error("failed to invalidate user cache after deletion", "user_id", id.String(), "error", err.Error())
			// Don't fail the request if cache invalidation fails
		}
	}

	// Log the deletion for audit
	if s.repos.Audit != nil {
		auditDetails := map[string]interface{}{
			"user_id":   user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"is_active": user.IsActive,
		}
		if err := s.repos.Audit.Log(ctx, "user", user.ID, "delete", auditDetails); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("failed to log user delete audit: %v\n", err)
		}
	}

	return nil
}

// GetProfile returns the current user's profile.
func (s *UserServiceImpl) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error) {
	return s.GetByID(ctx, userID)
}

// UpdateProfile updates the current user's profile.
func (s *UserServiceImpl) UpdateProfile(ctx context.Context, userID uuid.UUID, req *domain.UpdateUserRequest) (*domain.UserResponse, error) {
	return s.Update(ctx, userID, req)
}
