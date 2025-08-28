// Package service defines interfaces for business logic services.
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/domain"
)

// AuthService defines the interface for authentication operations.
type AuthService interface {
	// Register creates a new user account.
	Register(ctx context.Context, req *domain.CreateUserRequest) (*domain.UserResponse, error)

	// Login authenticates a user and returns tokens.
	Login(ctx context.Context, email, password string) (*LoginResponse, error)

	// RefreshToken generates a new access token from a refresh token.
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)

	// ValidateToken validates an access token and returns user info.
	ValidateToken(ctx context.Context, token string) (*domain.UserResponse, error)

	// Logout invalidates a refresh token.
	Logout(ctx context.Context, refreshToken string) error
}

// UserService defines the interface for user management operations.
type UserService interface {
	// GetByID retrieves a user by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.UserResponse, error)

	// List retrieves users with pagination (admin only).
	List(ctx context.Context, limit, offset int) ([]*domain.UserResponse, error)

	// Update updates user information.
	Update(ctx context.Context, id uuid.UUID, req *domain.UpdateUserRequest) (*domain.UserResponse, error)

	// Delete deletes a user account.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetProfile returns the current user's profile.
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error)

	// UpdateProfile updates the current user's profile.
	UpdateProfile(ctx context.Context, userID uuid.UUID, req *domain.UpdateUserRequest) (*domain.UserResponse, error)
}

// BalanceService defines the interface for balance operations.
type BalanceService interface {
	// GetCurrent retrieves the current balance for a user.
	GetCurrent(ctx context.Context, userID uuid.UUID) (*domain.BalanceResponse, error)

	// GetHistorical retrieves historical balance snapshots.
	GetHistorical(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.BalanceHistoryItem, error)

	// GetAtTime retrieves balance at a specific time.
	GetAtTime(ctx context.Context, userID uuid.UUID, timestamp string) (*domain.BalanceResponse, error)

	// Initialize creates an initial balance for a new user.
	Initialize(ctx context.Context, userID uuid.UUID, initialAmount float64, currency string) error
}

// TransactionService defines the interface for transaction operations.
type TransactionService interface {
	// Credit adds money to a user's account.
	Credit(ctx context.Context, userID uuid.UUID, req *domain.CreditRequest) (*domain.TransactionResponse, error)

	// Debit removes money from a user's account.
	Debit(ctx context.Context, userID uuid.UUID, req *domain.DebitRequest) (*domain.TransactionResponse, error)

	// Transfer moves money between user accounts.
	Transfer(ctx context.Context, fromUserID uuid.UUID, req *domain.TransferRequest) (*domain.TransactionResponse, error)

	// GetByID retrieves a transaction by ID.
	GetByID(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*domain.TransactionResponse, error)

	// GetHistory retrieves transaction history for a user.
	GetHistory(ctx context.Context, userID uuid.UUID, filter *domain.TransactionFilter) ([]*domain.TransactionResponse, error)

	// ListAll retrieves all transactions (admin only).
	ListAll(ctx context.Context, filter *domain.TransactionFilter) ([]*domain.TransactionResponse, error)

	// Rollback reverses a completed transaction (if within policy window).
	Rollback(ctx context.Context, transactionID uuid.UUID, requestingUserID uuid.UUID) (*domain.TransactionResponse, error)

	// RollbackByAdmin reverses a completed transaction (admin version without permission checks).
	RollbackByAdmin(ctx context.Context, transactionID uuid.UUID) (*domain.TransactionResponse, error)

	// Sync methods for worker pool
	CreditSync(ctx context.Context, userID uuid.UUID, req *domain.CreditRequest) (*domain.TransactionResponse, error)
	DebitSync(ctx context.Context, userID uuid.UUID, req *domain.DebitRequest) (*domain.TransactionResponse, error)
	TransferSync(ctx context.Context, fromUserID uuid.UUID, req *domain.TransferRequest) (*domain.TransactionResponse, error)
	RollbackSync(ctx context.Context, transactionID uuid.UUID, requestingUserID uuid.UUID) (*domain.TransactionResponse, error)

	// SetPool sets the worker pool for async processing.
	SetPool(pool interface{})

	// SetMetricsCollector sets the metrics collector for tracking metrics.
	SetMetricsCollector(collector interface{})
}

// ScheduledTransactionService defines the interface for scheduled transaction operations.
type ScheduledTransactionService interface {
	// Create creates a new scheduled transaction.
	Create(ctx context.Context, userID uuid.UUID, req *domain.ScheduledTransactionRequest) (*domain.ScheduledTransactionResponse, error)

	// GetByID retrieves a scheduled transaction by ID.
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.ScheduledTransactionResponse, error)

	// List retrieves scheduled transactions for a user.
	List(ctx context.Context, userID uuid.UUID, filter *domain.ScheduledTransactionFilter) ([]*domain.ScheduledTransactionResponse, error)

	// Cancel cancels a scheduled transaction.
	Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	// ProcessDueTransactions processes all scheduled transactions that are due for execution.
	ProcessDueTransactions(ctx context.Context) error
}

// WorkerService defines the interface for worker operations needed by services.
type WorkerService interface {
	// SubmitTransaction submits a transaction for async processing.
	SubmitTransaction(job interface{}) error
	// GetQueueDepth returns the current queue depth.
	GetQueueDepth() int
}

// ProjectorServiceInterface defines the interface for projector services
type ProjectorServiceInterface interface {
	ProcessEventsSince(ctx context.Context, since time.Time) error
	ProcessAllEvents(ctx context.Context) error
}

// Services aggregates all service interfaces.
type Services struct {
	Auth                 AuthService
	User                 UserService
	Balance              BalanceService
	Transaction          TransactionService
	ScheduledTransaction ScheduledTransactionService
	Event                *EventService
	Projector            *ProjectorService
	Cache                CacheService
}

// LoginResponse represents the response from login operation.
type LoginResponse struct {
	User         *domain.UserResponse `json:"user"`
	AccessToken  string               `json:"access_token"`
	RefreshToken string               `json:"refresh_token"`
	ExpiresIn    int                  `json:"expires_in"`
}

// TokenResponse represents the response from token refresh operation.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}
