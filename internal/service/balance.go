package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/domain"
	"github.com/sefa-b/go-banking-sim/internal/repository"
	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// BalanceServiceImpl implements the BalanceService interface.
type BalanceServiceImpl struct {
	repos *repository.Repositories
	cache CacheService // Optional cache service
}

// NewBalanceService creates a new balance service.
func NewBalanceService(repos *repository.Repositories) BalanceService {
	return &BalanceServiceImpl{
		repos: repos,
		cache: nil, // Will be set later if cache is available
	}
}

// SetCacheService sets the cache service for this balance service
func (s *BalanceServiceImpl) SetCacheService(cache CacheService) {
	s.cache = cache
}

// GetCurrent retrieves the current balance for a user.
func (s *BalanceServiceImpl) GetCurrent(ctx context.Context, userID uuid.UUID) (*domain.BalanceResponse, error) {
	// Try cache first if available
	if s.cache != nil {
		cachedBalance, err := s.cache.GetCachedBalance(ctx, userID)
		if err == nil {
			utils.Info("cache hit for balance", "user_id", userID.String())
			return cachedBalance, nil
		}
		// Cache miss or error - continue to database
		utils.Info("cache miss for balance", "user_id", userID.String())
	}

	balance, err := s.repos.Balances.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	response := balance.ToResponse()

	// Cache the result if cache is available
	if s.cache != nil {
		if err := s.cache.CacheBalance(ctx, balance); err != nil {
			utils.Error("failed to cache balance", "user_id", userID.String(), "error", err.Error())
			// Don't fail the request if caching fails
		}
	}

	return &response, nil
}

// GetHistorical retrieves historical balance snapshots.
func (s *BalanceServiceImpl) GetHistorical(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.BalanceHistoryItem, error) {
	// Call the repository to get historical balance snapshots
	history, err := s.repos.Balances.GetHistorical(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical balance from repository: %w", err)
	}

	return history, nil
}

// GetAtTime retrieves balance at a specific time.
func (s *BalanceServiceImpl) GetAtTime(ctx context.Context, userID uuid.UUID, timestamp string) (*domain.BalanceResponse, error) {
	balance, err := s.repos.Balances.GetAtTime(ctx, userID, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance at time: %w", err)
	}

	response := balance.ToResponse()
	return &response, nil
}

// Initialize creates an initial balance for a new user.
func (s *BalanceServiceImpl) Initialize(ctx context.Context, userID uuid.UUID, initialAmount float64, currency string) error {
	// Validate currency
	if !domain.IsValidCurrency(currency) {
		return fmt.Errorf("unsupported currency: %s", currency)
	}

	balance := &domain.Balance{
		UserID:   userID,
		Amount:   initialAmount,
		Currency: currency,
	}

	err := s.repos.Balances.Upsert(ctx, balance)
	if err != nil {
		return fmt.Errorf("failed to initialize balance: %w", err)
	}

	// Invalidate any cached balance for this user since we created a new one
	if s.cache != nil {
		if err := s.cache.InvalidateBalanceCache(ctx, userID); err != nil {
			utils.Error("failed to invalidate balance cache during initialization", "user_id", userID.String(), "error", err.Error())
			// Don't fail the operation if cache invalidation fails
		}
	}

	// Log the balance initialization for audit
	if s.repos.Audit != nil {
		auditDetails := map[string]interface{}{
			"user_id":  userID,
			"amount":   initialAmount,
			"currency": currency,
		}
		if err := s.repos.Audit.Log(ctx, "balance", userID, "initialize", auditDetails); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("failed to log balance initialize audit: %v\n", err)
		}
	}

	return nil
}
