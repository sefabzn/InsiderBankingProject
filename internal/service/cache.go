package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/domain"
	"github.com/sefa-b/go-banking-sim/internal/repository"
	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// CacheService defines the interface for caching operations
type CacheService interface {
	// User cache operations
	CacheUser(ctx context.Context, user *domain.User) error
	GetCachedUser(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error)
	InvalidateUserCache(ctx context.Context, userID uuid.UUID) error

	// Balance cache operations
	CacheBalance(ctx context.Context, balance *domain.Balance) error
	GetCachedBalance(ctx context.Context, userID uuid.UUID) (*domain.BalanceResponse, error)
	InvalidateBalanceCache(ctx context.Context, userID uuid.UUID) error

	// Transaction cache operations
	CacheTransaction(ctx context.Context, transaction *domain.Transaction) error
	GetCachedTransaction(ctx context.Context, transactionID uuid.UUID) (*domain.TransactionResponse, error)
	InvalidateTransactionCache(ctx context.Context, transactionID uuid.UUID) error
	InvalidateTransactionHistoryCache(ctx context.Context, userID uuid.UUID) error

	// Session operations
	CacheSession(ctx context.Context, sessionID string, userID uuid.UUID, expiration time.Duration) error
	GetCachedSession(ctx context.Context, sessionID string) (uuid.UUID, error)
	InvalidateSession(ctx context.Context, sessionID string) error

	// Rate limiting
	CheckRateLimit(ctx context.Context, clientIP string, maxRequests int, window time.Duration) (bool, error)
	GetRateLimitCount(ctx context.Context, clientIP string) (int64, error)

	// Bulk operations
	InvalidateUserRelatedCache(ctx context.Context, userID uuid.UUID) error
	InvalidateTransactionRelatedCache(ctx context.Context, transaction *domain.Transaction) error
	CacheMultipleUsers(ctx context.Context, users []*domain.User) error
	CacheMultipleBalances(ctx context.Context, balances []*domain.Balance) error

	// Health and stats
	Health(ctx context.Context) error
	GetCacheStats(ctx context.Context) (map[string]int64, error)
}

// cacheServiceImpl provides caching functionality for the banking application
type cacheServiceImpl struct {
	redisClient *repository.RedisClient
}

// NewCacheService creates a new cache service
func NewCacheService(redisClient *repository.RedisClient) CacheService {
	return &cacheServiceImpl{
		redisClient: redisClient,
	}
}

// User cache operations
const (
	userCachePrefix    = "user:"
	userCacheTTL       = 30 * time.Minute
	balanceCachePrefix = "balance:"
	balanceCacheTTL    = 10 * time.Minute
)

// CacheUser caches user information
func (c *cacheServiceImpl) CacheUser(ctx context.Context, user *domain.User) error {
	key := userCachePrefix + user.ID.String()
	return c.redisClient.Set(ctx, key, user.ToResponse(), userCacheTTL)
}

// GetCachedUser retrieves a cached user
func (c *cacheServiceImpl) GetCachedUser(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error) {
	key := userCachePrefix + userID.String()
	var user domain.UserResponse
	err := c.redisClient.Get(ctx, key, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// InvalidateUserCache removes user from cache
func (c *cacheServiceImpl) InvalidateUserCache(ctx context.Context, userID uuid.UUID) error {
	key := userCachePrefix + userID.String()
	return c.redisClient.Del(ctx, key)
}

// CacheBalance caches balance information
func (c *cacheServiceImpl) CacheBalance(ctx context.Context, balance *domain.Balance) error {
	key := balanceCachePrefix + balance.UserID.String()
	return c.redisClient.Set(ctx, key, balance.ToResponse(), balanceCacheTTL)
}

// GetCachedBalance retrieves a cached balance
func (c *cacheServiceImpl) GetCachedBalance(ctx context.Context, userID uuid.UUID) (*domain.BalanceResponse, error) {
	key := balanceCachePrefix + userID.String()
	var balance domain.BalanceResponse
	err := c.redisClient.Get(ctx, key, &balance)
	if err != nil {
		return nil, err
	}
	return &balance, nil
}

// InvalidateBalanceCache removes balance from cache
func (c *cacheServiceImpl) InvalidateBalanceCache(ctx context.Context, userID uuid.UUID) error {
	key := balanceCachePrefix + userID.String()
	return c.redisClient.Del(ctx, key)
}

// InvalidateUserRelatedCache removes all cache entries related to a user
func (c *cacheServiceImpl) InvalidateUserRelatedCache(ctx context.Context, userID uuid.UUID) error {
	userIDStr := userID.String()

	// Invalidate user cache
	userKey := userCachePrefix + userIDStr

	// Invalidate balance cache
	balanceKey := balanceCachePrefix + userIDStr

	// Combine keys to delete
	keysToDelete := []string{userKey, balanceKey}

	if len(keysToDelete) > 0 {
		return c.redisClient.Del(ctx, keysToDelete...)
	}

	return nil
}

// InvalidateTransactionRelatedCache removes all cache entries related to a specific transaction
func (c *cacheServiceImpl) InvalidateTransactionRelatedCache(ctx context.Context, transaction *domain.Transaction) error {
	// Invalidate the transaction cache itself
	transactionKey := transactionCachePrefix + transaction.ID.String()

	// Collect all keys to invalidate
	keysToDelete := []string{transactionKey}

	// Invalidate caches for users involved in this transaction
	if transaction.FromUserID != nil {
		userIDStr := transaction.FromUserID.String()
		keysToDelete = append(keysToDelete,
			userCachePrefix+userIDStr,
			balanceCachePrefix+userIDStr,
			transactionHistoryPrefix+userIDStr,
		)
	}

	if transaction.ToUserID != nil {
		userIDStr := transaction.ToUserID.String()
		keysToDelete = append(keysToDelete,
			userCachePrefix+userIDStr,
			balanceCachePrefix+userIDStr,
			transactionHistoryPrefix+userIDStr,
		)
	}

	if len(keysToDelete) > 0 {
		return c.redisClient.Del(ctx, keysToDelete...)
	}

	return nil
}

// Transaction cache operations
const (
	transactionCachePrefix   = "transaction:"
	transactionHistoryPrefix = "transaction_history:"
	transactionCacheTTL      = 15 * time.Minute
)

// CacheTransaction caches transaction information
func (c *cacheServiceImpl) CacheTransaction(ctx context.Context, transaction *domain.Transaction) error {
	key := transactionCachePrefix + transaction.ID.String()
	return c.redisClient.Set(ctx, key, transaction.ToResponse(), transactionCacheTTL)
}

// GetCachedTransaction retrieves a cached transaction
func (c *cacheServiceImpl) GetCachedTransaction(ctx context.Context, transactionID uuid.UUID) (*domain.TransactionResponse, error) {
	key := transactionCachePrefix + transactionID.String()
	var transaction domain.TransactionResponse
	err := c.redisClient.Get(ctx, key, &transaction)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// InvalidateTransactionCache removes transaction from cache
func (c *cacheServiceImpl) InvalidateTransactionCache(ctx context.Context, transactionID uuid.UUID) error {
	key := transactionCachePrefix + transactionID.String()
	return c.redisClient.Del(ctx, key)
}

// InvalidateTransactionHistoryCache removes transaction history cache for a user
func (c *cacheServiceImpl) InvalidateTransactionHistoryCache(ctx context.Context, userID uuid.UUID) error {
	key := transactionHistoryPrefix + userID.String()
	return c.redisClient.Del(ctx, key)
}

// Session cache operations
const (
	sessionCachePrefix = "session:"
)

// CacheSession caches session information
func (c *cacheServiceImpl) CacheSession(ctx context.Context, sessionID string, userID uuid.UUID, expiration time.Duration) error {
	key := sessionCachePrefix + sessionID
	sessionData := map[string]interface{}{
		"user_id":    userID,
		"created_at": time.Now(),
	}
	return c.redisClient.Set(ctx, key, sessionData, expiration)
}

// GetCachedSession retrieves session information
func (c *cacheServiceImpl) GetCachedSession(ctx context.Context, sessionID string) (uuid.UUID, error) {
	key := sessionCachePrefix + sessionID
	var sessionData map[string]interface{}
	err := c.redisClient.Get(ctx, key, &sessionData)
	if err != nil {
		return uuid.Nil, err
	}

	userIDStr, ok := sessionData["user_id"].(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid session data")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in session")
	}

	return userID, nil
}

// InvalidateSession removes session from cache
func (c *cacheServiceImpl) InvalidateSession(ctx context.Context, sessionID string) error {
	key := sessionCachePrefix + sessionID
	return c.redisClient.Del(ctx, key)
}

// Rate limiting operations
const (
	rateLimitPrefix = "ratelimit:"
)

// CheckRateLimit checks if a client has exceeded rate limits
func (c *cacheServiceImpl) CheckRateLimit(ctx context.Context, clientIP string, maxRequests int, window time.Duration) (bool, error) {
	key := rateLimitPrefix + clientIP

	// Get current request count
	count, err := c.redisClient.Incr(ctx, key)
	if err != nil {
		return false, err
	}

	// Set expiration on first request
	if count == 1 {
		if err := c.redisClient.Expire(ctx, key, window); err != nil {
			return false, err
		}
	}

	// Check if limit exceeded
	return count <= int64(maxRequests), nil
}

// GetRateLimitCount gets current request count for a client
func (c *cacheServiceImpl) GetRateLimitCount(ctx context.Context, clientIP string) (int64, error) {
	key := rateLimitPrefix + clientIP
	count, err := c.redisClient.GetClient().Get(ctx, key).Int64()
	if err != nil {
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

// Cache warming operations
const (
	cacheWarmupPrefix = "warmup:"
	cacheWarmupTTL    = 1 * time.Hour
)

// MarkCacheWarmed marks that cache has been warmed for an entity
func (c *cacheServiceImpl) MarkCacheWarmed(ctx context.Context, entityType string, entityID string) error {
	key := cacheWarmupPrefix + entityType + ":" + entityID
	return c.redisClient.Set(ctx, key, time.Now().Format(time.RFC3339), cacheWarmupTTL)
}

// IsCacheWarmed checks if cache has been warmed for an entity
func (c *cacheServiceImpl) IsCacheWarmed(ctx context.Context, entityType string, entityID string) (bool, error) {
	key := cacheWarmupPrefix + entityType + ":" + entityID
	return c.redisClient.Exists(ctx, key)
}

// Bulk operations
// CacheMultipleUsers caches multiple users
func (c *cacheServiceImpl) CacheMultipleUsers(ctx context.Context, users []*domain.User) error {
	for _, user := range users {
		if err := c.CacheUser(ctx, user); err != nil {
			utils.Error("failed to cache user", "user_id", user.ID.String(), "error", err.Error())
		}
	}
	return nil
}

// CacheMultipleBalances caches multiple balances
func (c *cacheServiceImpl) CacheMultipleBalances(ctx context.Context, balances []*domain.Balance) error {
	for _, balance := range balances {
		if err := c.CacheBalance(ctx, balance); err != nil {
			utils.Error("failed to cache balance", "user_id", balance.UserID.String(), "error", err.Error())
		}
	}
	return nil
}

// Health check
// Health checks Redis connectivity
func (c *cacheServiceImpl) Health(ctx context.Context) error {
	return c.redisClient.Ping(ctx)
}

// Statistics
// GetCacheStats returns basic cache statistics
func (c *cacheServiceImpl) GetCacheStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	// Count users in cache
	userKeys, err := c.redisClient.Keys(ctx, userCachePrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("failed to get user keys: %w", err)
	}
	stats["cached_users"] = int64(len(userKeys))

	// Count balances in cache
	balanceKeys, err := c.redisClient.Keys(ctx, balanceCachePrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("failed to get balance keys: %w", err)
	}
	stats["cached_balances"] = int64(len(balanceKeys))

	// Count transactions in cache
	transactionKeys, err := c.redisClient.Keys(ctx, transactionCachePrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction keys: %w", err)
	}
	stats["cached_transactions"] = int64(len(transactionKeys))

	return stats, nil
}
