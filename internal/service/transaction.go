// Package service provides business logic for transaction operations.
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sefa-b/go-banking-sim/internal/domain"
	"github.com/sefa-b/go-banking-sim/internal/repository"
	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// TransactionServiceImpl implements the TransactionService interface.
type TransactionServiceImpl struct {
	repos            *repository.Repositories
	balanceService   BalanceService
	workerPool       WorkerService
	metricsCollector interface{}   // Will hold metrics collector to avoid circular imports
	cache            CacheService  // Optional cache service
	eventSvc         *EventService // Event service for publishing domain events
	dbPool           interface{}   // Database pool for transactions
}

// NewTransactionService creates a new transaction service.
func NewTransactionService(repos *repository.Repositories, balanceService BalanceService, workerPool WorkerService, eventSvc *EventService, dbPool interface{}) TransactionService {
	return &TransactionServiceImpl{
		repos:          repos,
		balanceService: balanceService,
		workerPool:     workerPool,
		cache:          nil, // Will be set later if cache is available
		eventSvc:       eventSvc,
		dbPool:         dbPool,
	}
}

// SetCacheService sets the cache service for this transaction service
func (s *TransactionServiceImpl) SetCacheService(cache CacheService) {
	s.cache = cache
}

// SetMetricsCollector sets the metrics collector for tracking transaction metrics.
func (s *TransactionServiceImpl) SetMetricsCollector(collector interface{}) {
	s.metricsCollector = collector
}

// incrementTransactionCounter increments the transaction processed counter if metrics collector is available.
func (s *TransactionServiceImpl) incrementTransactionCounter() {
	if s.metricsCollector != nil {
		// Use reflection-like approach to call IncrementTransactionsProcessed
		if mc, ok := s.metricsCollector.(interface{ IncrementTransactionsProcessed() }); ok {
			mc.IncrementTransactionsProcessed()
		}
	}
}

// SetPool sets the worker pool for async processing.
func (s *TransactionServiceImpl) SetPool(pool interface{}) {
	if wp, ok := pool.(WorkerService); ok {
		s.workerPool = wp
	}
}

// SyncTransactionService provides synchronous transaction operations for worker pool.
type SyncTransactionService interface {
	CreditSync(ctx context.Context, userID uuid.UUID, req *domain.CreditRequest) (*domain.TransactionResponse, error)
	DebitSync(ctx context.Context, userID uuid.UUID, req *domain.DebitRequest) (*domain.TransactionResponse, error)
	TransferSync(ctx context.Context, fromUserID uuid.UUID, req *domain.TransferRequest) (*domain.TransactionResponse, error)
	RollbackSync(ctx context.Context, transactionID uuid.UUID, requestingUserID uuid.UUID) (*domain.TransactionResponse, error)
}

// Ensure TransactionServiceImpl implements SyncTransactionService
var _ SyncTransactionService = (*TransactionServiceImpl)(nil)

// CreditSync processes a credit synchronously (for internal use by worker pool).
func (s *TransactionServiceImpl) CreditSync(ctx context.Context, userID uuid.UUID, req *domain.CreditRequest) (*domain.TransactionResponse, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid credit request: %w", err)
	}

	// Use the balance service to get current balance (with caching)
	currentBalanceResp, err := s.balanceService.GetCurrent(ctx, userID)
	if err != nil && !isNotFoundError(err) {
		return nil, fmt.Errorf("failed to get current balance: %w", err)
	}

	var currentBalance *domain.Balance
	if currentBalanceResp != nil {
		currentBalance = &domain.Balance{
			UserID:   currentBalanceResp.UserID,
			Amount:   currentBalanceResp.Amount,
			Currency: currentBalanceResp.Currency,
		}
	}

	// If user has no balance, create one with the transaction currency
	if currentBalance == nil {
		currentBalance = &domain.Balance{
			UserID:   userID,
			Amount:   0,
			Currency: req.Currency,
		}
	} else if currentBalance.Currency != req.Currency {
		return nil, fmt.Errorf("currency mismatch: user balance is in %s but transaction is in %s", currentBalance.Currency, req.Currency)
	}

	// Calculate new amount
	newAmount := currentBalance.Amount + req.Amount

	// Create balance update
	newBalance := &domain.Balance{
		UserID:   userID,
		Amount:   newAmount,
		Currency: req.Currency,
	}

	// Create the transaction record as pending first
	transaction := &domain.Transaction{
		FromUserID: nil,     // Credits don't have a source
		ToUserID:   &userID, // The user receiving the credit
		Amount:     req.Amount,
		Currency:   req.Currency,
		Type:       string(domain.TypeCredit),
		Status:     string(domain.StatusPending), // Start as pending
	}

	// Create the transaction in the database
	if err := s.repos.Transactions.CreatePending(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update the balance
	if err := s.repos.Balances.Upsert(ctx, newBalance); err != nil {
		// Mark transaction as failed if balance update fails
		_ = s.repos.Transactions.MarkFailed(ctx, transaction.ID)
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	// Mark transaction as completed only after successful balance update
	if err := s.repos.Transactions.MarkCompleted(ctx, transaction.ID); err != nil {
		return nil, fmt.Errorf("failed to mark transaction completed: %w", err)
	}

	// Note: Events are published at higher levels (e.g., in Transfer method)
	// to avoid double-counting when CreditSync/DebitSync are called from Transfer

	// Invalidate related caches after successful update
	if s.cache != nil {
		// Invalidate balance cache
		if err := s.cache.InvalidateBalanceCache(ctx, userID); err != nil {
			utils.Error("failed to invalidate balance cache", "user_id", userID.String(), "error", err.Error())
		}

		// Invalidate transaction history cache for the user
		if err := s.cache.InvalidateTransactionHistoryCache(ctx, userID); err != nil {
			utils.Error("failed to invalidate transaction history cache", "user_id", userID.String(), "error", err.Error())
		}

		// Cache the new transaction
		if err := s.cache.CacheTransaction(ctx, transaction); err != nil {
			utils.Error("failed to cache transaction", "transaction_id", transaction.ID.String(), "error", err.Error())
		}
	}

	// Log the audit event
	_ = s.repos.Audit.Log(ctx, "transaction", transaction.ID, "credit", map[string]interface{}{
		"user_id": userID,
		"amount":  req.Amount,
	})

	// Increment transaction counter for metrics
	s.incrementTransactionCounter()

	// Return the transaction response
	response := transaction.ToResponse()
	return &response, nil
}

// DebitSync removes money from a user's account synchronously (for internal use by worker pool).
func (s *TransactionServiceImpl) DebitSync(ctx context.Context, userID uuid.UUID, req *domain.DebitRequest) (*domain.TransactionResponse, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid debit request: %w", err)
	}

	// Check if user has sufficient balance
	balanceResp, err := s.balanceService.GetCurrent(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current balance: %w", err)
	}

	// Convert BalanceResponse to Balance for consistency
	balance := &domain.Balance{
		Amount:   balanceResp.Amount,
		Currency: balanceResp.Currency,
	}

	if balance.Currency != req.Currency {
		return nil, fmt.Errorf("currency mismatch: user balance is in %s but transaction is in %s", balance.Currency, req.Currency)
	}

	if balance.Amount < req.Amount {
		return nil, fmt.Errorf("insufficient funds: current balance %.2f %s, requested %.2f %s", balance.Amount, balance.Currency, req.Amount, req.Currency)
	}

	// Create the transaction record
	transaction := &domain.Transaction{
		FromUserID: &userID, // The user being debited
		ToUserID:   nil,     // Debits don't have a destination
		Amount:     req.Amount,
		Currency:   req.Currency,
		Type:       string(domain.TypeDebit),
		Status:     string(domain.StatusPending),
	}

	// Create the transaction in the database
	if err := s.repos.Transactions.CreatePending(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update the user's balance (negative amount for debit)
	newAmount := balance.Amount - req.Amount
	newBalance := &domain.Balance{
		UserID:   userID,
		Amount:   newAmount,
		Currency: balance.Currency,
	}

	if err := s.repos.Balances.Upsert(ctx, newBalance); err != nil {
		// Mark transaction as failed
		_ = s.repos.Transactions.MarkFailed(ctx, transaction.ID)
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	// Mark transaction as completed
	if err := s.repos.Transactions.MarkCompleted(ctx, transaction.ID); err != nil {
		return nil, fmt.Errorf("failed to mark transaction completed: %w", err)
	}

	// Note: Events are published at higher levels (e.g., in Transfer method)
	// to avoid double-counting when CreditSync/DebitSync are called from Transfer

	// Invalidate related caches after successful update
	if s.cache != nil {
		// Invalidate balance cache
		if err := s.cache.InvalidateBalanceCache(ctx, userID); err != nil {
			utils.Error("failed to invalidate balance cache", "user_id", userID.String(), "error", err.Error())
		}

		// Invalidate transaction history cache for the user
		if err := s.cache.InvalidateTransactionHistoryCache(ctx, userID); err != nil {
			utils.Error("failed to invalidate transaction history cache", "user_id", userID.String(), "error", err.Error())
		}

		// Cache the new transaction
		if err := s.cache.CacheTransaction(ctx, transaction); err != nil {
			utils.Error("failed to cache transaction", "transaction_id", transaction.ID.String(), "error", err.Error())
		}
	}

	// Log the audit event
	_ = s.repos.Audit.Log(ctx, "transaction", transaction.ID, "debit", map[string]interface{}{
		"user_id": userID,
		"amount":  req.Amount,
	})

	// Increment transaction counter for metrics
	s.incrementTransactionCounter()

	// Return the transaction response
	response := transaction.ToResponse()
	return &response, nil
}

// Debit removes money from a user's account asynchronously.
func (s *TransactionServiceImpl) Debit(ctx context.Context, userID uuid.UUID, req *domain.DebitRequest) (*domain.TransactionResponse, error) {
	// For now, always use sync processing to avoid worker pool complexity
	// TODO: Fix worker pool implementation
	return s.DebitSync(ctx, userID, req)
}

// TransferSync moves money between user accounts synchronously (for internal use by worker pool).
func (s *TransactionServiceImpl) TransferSync(ctx context.Context, fromUserID uuid.UUID, req *domain.TransferRequest) (*domain.TransactionResponse, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid transfer request: %w", err)
	}

	// Check if from user has sufficient balance
	fromBalanceResp, err := s.balanceService.GetCurrent(ctx, fromUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender balance: %w", err)
	}

	fromBalance := &domain.Balance{
		Amount:   fromBalanceResp.Amount,
		Currency: fromBalanceResp.Currency,
	}

	if fromBalance.Currency != req.Currency {
		return nil, fmt.Errorf("currency mismatch: sender balance is in %s but transaction is in %s", fromBalance.Currency, req.Currency)
	}

	if fromBalance.Amount < req.Amount {
		return nil, fmt.Errorf("insufficient funds: current balance %.2f %s, requested %.2f %s", fromBalance.Amount, fromBalance.Currency, req.Amount, req.Currency)
	}

	// Check receiver's balance and currency
	toBalanceResp, err := s.balanceService.GetCurrent(ctx, req.ToUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get receiver balance: %w", err)
	}

	toBalance := &domain.Balance{
		Currency: toBalanceResp.Currency,
	}

	if toBalance.Currency != req.Currency {
		return nil, fmt.Errorf("currency mismatch: receiver balance is in %s but transaction is in %s", toBalance.Currency, req.Currency)
	}

	// Create the transaction record
	transaction := &domain.Transaction{
		FromUserID: &fromUserID,
		ToUserID:   &req.ToUserID,
		Amount:     req.Amount,
		Currency:   req.Currency,
		Type:       string(domain.TypeTransfer),
		Status:     string(domain.StatusPending),
	}

	// Create the transaction in the database
	if err := s.repos.Transactions.CreatePending(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Use database transaction to ensure atomicity
	if s.dbPool == nil {
		_ = s.repos.Transactions.MarkFailed(ctx, transaction.ID)
		return nil, fmt.Errorf("database pool not available")
	}

	// Type assert to pgxpool.Pool
	pool, ok := s.dbPool.(*pgxpool.Pool)
	if !ok {
		_ = s.repos.Transactions.MarkFailed(ctx, transaction.ID)
		return nil, fmt.Errorf("invalid database pool type")
	}

	// Begin database transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		_ = s.repos.Transactions.MarkFailed(ctx, transaction.ID)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx) // Rollback error is typically safe to ignore
	}()

	// Debit sender (subtract amount)
	if err := s.repos.Balances.AddAmountTx(ctx, tx, fromUserID, -req.Amount); err != nil {
		_ = s.repos.Transactions.MarkFailed(ctx, transaction.ID)
		return nil, fmt.Errorf("failed to debit sender: %w", err)
	}

	// Credit receiver (add amount)
	if err := s.repos.Balances.AddAmountTx(ctx, tx, req.ToUserID, req.Amount); err != nil {
		_ = s.repos.Transactions.MarkFailed(ctx, transaction.ID)
		return nil, fmt.Errorf("failed to credit receiver: %w", err)
	}

	// Commit the database transaction
	if err := tx.Commit(ctx); err != nil {
		_ = s.repos.Transactions.MarkFailed(ctx, transaction.ID)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Mark transaction as completed
	if err := s.repos.Transactions.MarkCompleted(ctx, transaction.ID); err != nil {
		return nil, fmt.Errorf("failed to mark transaction completed: %w", err)
	}

	// Publish events for the transfer
	if s.eventSvc != nil {
		if err := s.eventSvc.TransferExecuted(ctx, fromUserID, req.ToUserID, req.Amount, req.Currency, transaction.ID); err != nil {
			utils.Error("failed to publish transfer executed event", "error", err.Error())
		}
	}

	// Invalidate related caches after successful update
	if s.cache != nil {
		// Invalidate sender's balance cache
		if err := s.cache.InvalidateBalanceCache(ctx, fromUserID); err != nil {
			utils.Error("failed to invalidate sender balance cache", "user_id", fromUserID.String(), "error", err.Error())
		}

		// Invalidate receiver's balance cache
		if err := s.cache.InvalidateBalanceCache(ctx, req.ToUserID); err != nil {
			utils.Error("failed to invalidate receiver balance cache", "user_id", req.ToUserID.String(), "error", err.Error())
		}

		// Invalidate transaction history cache for both users
		if err := s.cache.InvalidateTransactionHistoryCache(ctx, fromUserID); err != nil {
			utils.Error("failed to invalidate sender transaction history cache", "user_id", fromUserID.String(), "error", err.Error())
		}

		if err := s.cache.InvalidateTransactionHistoryCache(ctx, req.ToUserID); err != nil {
			utils.Error("failed to invalidate receiver transaction history cache", "user_id", req.ToUserID.String(), "error", err.Error())
		}

		// Cache the new transaction
		if err := s.cache.CacheTransaction(ctx, transaction); err != nil {
			utils.Error("failed to cache transaction", "transaction_id", transaction.ID.String(), "error", err.Error())
		}
	}

	// Log the audit event
	_ = s.repos.Audit.Log(ctx, "transaction", transaction.ID, "transfer", map[string]interface{}{
		"from_user_id": fromUserID,
		"to_user_id":   req.ToUserID,
		"amount":       req.Amount,
	})

	// Increment transaction counter for metrics
	s.incrementTransactionCounter()

	// Return the transaction response
	response := transaction.ToResponse()
	return &response, nil
}

// Transfer moves money between user accounts asynchronously.
func (s *TransactionServiceImpl) Transfer(ctx context.Context, fromUserID uuid.UUID, req *domain.TransferRequest) (*domain.TransactionResponse, error) {
	// For now, always use sync processing to avoid worker pool complexity
	// TODO: Fix worker pool implementation
	return s.TransferSync(ctx, fromUserID, req)
}

// GetByID retrieves a transaction by ID.
func (s *TransactionServiceImpl) GetByID(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*domain.TransactionResponse, error) {
	// Try cache first if available
	if s.cache != nil {
		cachedTransaction, err := s.cache.GetCachedTransaction(ctx, id)
		if err == nil {
			utils.Info("cache hit for transaction", "transaction_id", id.String())
			return cachedTransaction, nil
		}
		// Cache miss or error - continue to database
		utils.Info("cache miss for transaction", "transaction_id", id.String())
	}

	transaction, err := s.repos.Transactions.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Check if user has permission to view this transaction
	// Users can view transactions they're involved in, admins can view all
	canView := false
	if transaction.FromUserID != nil && *transaction.FromUserID == requestingUserID {
		canView = true
	}
	if transaction.ToUserID != nil && *transaction.ToUserID == requestingUserID {
		canView = true
	}

	// TODO: Add admin role check here when user roles are available in context

	if !canView {
		return nil, fmt.Errorf("access denied: you don't have permission to view this transaction")
	}

	response := transaction.ToResponse()

	// Cache the result if cache is available
	if s.cache != nil {
		if err := s.cache.CacheTransaction(ctx, transaction); err != nil {
			utils.Error("failed to cache transaction", "transaction_id", id.String(), "error", err.Error())
			// Don't fail the request if caching fails
		}
	}

	return &response, nil
}

// Credit adds money to a user's account asynchronously.
func (s *TransactionServiceImpl) Credit(ctx context.Context, userID uuid.UUID, req *domain.CreditRequest) (*domain.TransactionResponse, error) {
	// For now, always use sync processing to avoid worker pool complexity
	// TODO: Fix worker pool implementation
	return s.CreditSync(ctx, userID, req)
}

// GetHistory retrieves transaction history for a user.
func (s *TransactionServiceImpl) GetHistory(ctx context.Context, userID uuid.UUID, filter *domain.TransactionFilter) ([]*domain.TransactionResponse, error) {
	// Set the user ID filter to ensure user only sees their own transactions
	if filter == nil {
		filter = &domain.TransactionFilter{}
	}
	filter.UserID = &userID

	// Try cache first if available and no complex filters are used
	useCache := s.cache != nil && filter.Limit <= 50 && filter.Offset == 0 &&
		filter.Type == nil && filter.Status == nil && filter.Since == nil

	// TODO: Implement more sophisticated caching based on filter parameters

	transactions, err := s.repos.Transactions.ListForUser(ctx, userID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	// Convert to response format
	responses := make([]*domain.TransactionResponse, len(transactions))
	for i, tx := range transactions {
		response := tx.ToResponse()
		responses[i] = &response

		// Cache individual transactions if cache is available
		if s.cache != nil {
			if err := s.cache.CacheTransaction(ctx, tx); err != nil {
				utils.Error("failed to cache transaction in history", "transaction_id", tx.ID.String(), "error", err.Error())
				// Don't fail the request if caching fails
			}
		}
	}

	// Simple caching for basic queries (no filters, small result set)
	// Note: Slice caching not implemented yet - individual transactions are cached above
	_ = useCache && s.cache != nil && len(responses) <= 20 // Placeholder for future slice caching

	return responses, nil
}

// ListAll retrieves all transactions (admin only).
func (s *TransactionServiceImpl) ListAll(ctx context.Context, filter *domain.TransactionFilter) ([]*domain.TransactionResponse, error) {
	// TODO: Add admin role check here when user roles are available in context

	transactions, err := s.repos.Transactions.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	// Convert to response format
	responses := make([]*domain.TransactionResponse, len(transactions))
	for i, tx := range transactions {
		response := tx.ToResponse()
		responses[i] = &response
	}

	return responses, nil
}

// RollbackSync reverses a completed transaction synchronously (for internal use by worker pool).
func (s *TransactionServiceImpl) RollbackSync(ctx context.Context, transactionID uuid.UUID, requestingUserID uuid.UUID) (*domain.TransactionResponse, error) {
	// Get the original transaction
	originalTx, err := s.repos.Transactions.GetByID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original transaction: %w", err)
	}

	// Check if transaction is completed and can be rolled back
	if originalTx.Status != string(domain.StatusSuccess) {
		return nil, fmt.Errorf("can only rollback completed transactions")
	}

	// Check if user has permission to rollback this transaction
	// For now, only allow the user who initiated the transaction
	canRollback := false
	if originalTx.FromUserID != nil && *originalTx.FromUserID == requestingUserID {
		canRollback = true
	}
	// TODO: Add admin role check

	if !canRollback {
		return nil, fmt.Errorf("access denied: you don't have permission to rollback this transaction")
	}

	return s.rollbackTransaction(ctx, originalTx, requestingUserID)
}

// Rollback reverses a completed transaction asynchronously (if within policy window).
func (s *TransactionServiceImpl) Rollback(ctx context.Context, transactionID uuid.UUID, requestingUserID uuid.UUID) (*domain.TransactionResponse, error) {
	// For now, always use sync processing to avoid worker pool complexity
	// TODO: Fix worker pool implementation
	return s.RollbackSync(ctx, transactionID, requestingUserID)
}

// RollbackByAdmin reverses a completed transaction (admin version without permission checks).
func (s *TransactionServiceImpl) RollbackByAdmin(ctx context.Context, transactionID uuid.UUID) (*domain.TransactionResponse, error) {
	// Get the original transaction
	originalTx, err := s.repos.Transactions.GetByID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original transaction: %w", err)
	}

	// Check if transaction is completed and can be rolled back
	if originalTx.Status != string(domain.StatusSuccess) {
		return nil, fmt.Errorf("can only rollback completed transactions")
	}

	return s.rollbackTransaction(ctx, originalTx, uuid.Nil) // No specific user for admin rollbacks
}

// rollbackTransaction performs the actual rollback logic without permission checks.
func (s *TransactionServiceImpl) rollbackTransaction(ctx context.Context, originalTx *domain.Transaction, requestingUserID uuid.UUID) (*domain.TransactionResponse, error) {
	// Determine the correct rollback transaction type and user assignments
	var rollbackType string
	var fromUserID, toUserID *uuid.UUID

	switch originalTx.Type {
	case string(domain.TypeCredit):
		// Original credit: NULL -> user123
		// Rollback debit: user123 -> NULL (remove money from user)
		rollbackType = string(domain.TypeDebit)
		fromUserID = originalTx.ToUserID
		toUserID = nil
	case string(domain.TypeDebit):
		// Original debit: user123 -> NULL
		// Rollback credit: NULL -> user123 (add money to user)
		rollbackType = string(domain.TypeCredit)
		fromUserID = nil
		toUserID = originalTx.FromUserID
	case string(domain.TypeTransfer):
		// Original transfer: user123 -> user456
		// Rollback transfer: user456 -> user123 (reverse direction)
		rollbackType = string(domain.TypeTransfer)
		fromUserID = originalTx.ToUserID
		toUserID = originalTx.FromUserID
	}

	// Create a rollback transaction
	rollbackTx := &domain.Transaction{
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Amount:     originalTx.Amount,
		Currency:   originalTx.Currency,
		Type:       rollbackType,
		Status:     string(domain.StatusPending),
	}

	// Create the rollback transaction
	if err := s.repos.Transactions.CreatePending(ctx, rollbackTx); err != nil {
		return nil, fmt.Errorf("failed to create rollback transaction: %w", err)
	}

	// Execute the rollback based on rollback transaction type (not original)
	switch rollbackType {
	case string(domain.TypeCredit):
		// Rollback credit: add money to the user (rollback of debit transaction)
		if toUserID != nil {
			currentBalance, err := s.repos.Balances.GetByUserID(ctx, *toUserID)
			if err != nil && !isNotFoundError(err) {
				_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
				return nil, fmt.Errorf("failed to get balance for rollback: %w", err)
			}

			var newAmount float64
			if currentBalance != nil {
				// User has existing balance, add the rollback amount
				newAmount = currentBalance.Amount + originalTx.Amount
			} else {
				// User has no balance record, create one with the rollback amount
				newAmount = originalTx.Amount
			}

			// Ensure amount is not negative (defensive check)
			if newAmount < 0 {
				_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
				return nil, fmt.Errorf("rollback would result in negative balance: current=%.2f, rollback_amount=%.2f",
					currentBalance.Amount, originalTx.Amount)
			}

			newBalance := &domain.Balance{
				UserID:   *toUserID,
				Amount:   newAmount,
				Currency: originalTx.Currency,
			}
			if err := s.repos.Balances.Upsert(ctx, newBalance); err != nil {
				_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
				return nil, fmt.Errorf("failed to rollback credit: %w", err)
			}
		}
	case string(domain.TypeDebit):
		// Rollback debit: remove money from the user (rollback of credit transaction)
		if fromUserID != nil {
			currentBalance, err := s.repos.Balances.GetByUserID(ctx, *fromUserID)
			if err != nil && !isNotFoundError(err) {
				_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
				return nil, fmt.Errorf("failed to get balance for rollback: %w", err)
			}
			if currentBalance != nil {
				newAmount := currentBalance.Amount - originalTx.Amount

				// Ensure amount is not negative (defensive check)
				if newAmount < 0 {
					_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
					return nil, fmt.Errorf("rollback would result in negative balance: current=%.2f, rollback_amount=%.2f",
						currentBalance.Amount, originalTx.Amount)
				}

				newBalance := &domain.Balance{
					UserID:   *fromUserID,
					Amount:   newAmount,
					Currency: originalTx.Currency,
				}
				if err := s.repos.Balances.Upsert(ctx, newBalance); err != nil {
					_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
					return nil, fmt.Errorf("failed to rollback debit: %w", err)
				}
			}
		}
	case string(domain.TypeTransfer):
		// Rollback transfer: move money back from recipient to sender
		if fromUserID != nil && toUserID != nil {
			// Debit the recipient (who was the original sender)
			recipientBalance, err := s.repos.Balances.GetByUserID(ctx, *fromUserID)
			if err != nil && !isNotFoundError(err) {
				_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
				return nil, fmt.Errorf("failed to get recipient balance for rollback: %w", err)
			}
			if recipientBalance != nil {
				recipientNewAmount := recipientBalance.Amount - originalTx.Amount

				// Ensure amount is not negative (defensive check)
				if recipientNewAmount < 0 {
					_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
					return nil, fmt.Errorf("transfer rollback would result in negative balance for recipient: current=%.2f, rollback_amount=%.2f",
						recipientBalance.Amount, originalTx.Amount)
				}

				recipientNewBalance := &domain.Balance{
					UserID:   *fromUserID,
					Amount:   recipientNewAmount,
					Currency: originalTx.Currency,
				}
				if err := s.repos.Balances.Upsert(ctx, recipientNewBalance); err != nil {
					_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
					return nil, fmt.Errorf("failed to rollback transfer (debit recipient): %w", err)
				}
			}

			// Credit the sender (who was the original recipient)
			senderBalance, err := s.repos.Balances.GetByUserID(ctx, *toUserID)
			if err != nil && !isNotFoundError(err) {
				// Rollback the previous operation
				if recipientBalance != nil {
					_ = s.repos.Balances.Upsert(ctx, recipientBalance)
				}
				_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
				return nil, fmt.Errorf("failed to get sender balance for rollback: %w", err)
			}
			var senderNewAmount float64
			if senderBalance != nil {
				senderNewAmount = senderBalance.Amount + originalTx.Amount
			} else {
				senderNewAmount = originalTx.Amount
			}

			// Ensure amount is not negative (defensive check)
			if senderNewAmount < 0 {
				// Rollback the previous operation
				if recipientBalance != nil {
					_ = s.repos.Balances.Upsert(ctx, recipientBalance)
				}
				_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
				currentAmount := 0.0
				if senderBalance != nil {
					currentAmount = senderBalance.Amount
				}
				return nil, fmt.Errorf("transfer rollback would result in negative balance for sender: current=%.2f, rollback_amount=%.2f",
					currentAmount, originalTx.Amount)
			}

			senderNewBalance := &domain.Balance{
				UserID:   *toUserID,
				Amount:   senderNewAmount,
				Currency: originalTx.Currency,
			}
			if err := s.repos.Balances.Upsert(ctx, senderNewBalance); err != nil {
				// Rollback the previous operation
				if recipientBalance != nil {
					_ = s.repos.Balances.Upsert(ctx, recipientBalance)
				}
				_ = s.repos.Transactions.MarkFailed(ctx, rollbackTx.ID)
				return nil, fmt.Errorf("failed to rollback transfer (credit sender): %w", err)
			}
		}
	}

	// Mark rollback transaction as completed
	if err := s.repos.Transactions.MarkCompleted(ctx, rollbackTx.ID); err != nil {
		return nil, fmt.Errorf("failed to mark rollback completed: %w", err)
	}

	// Invalidate related caches after successful rollback
	if s.cache != nil {
		// Determine which users' caches need to be invalidated based on rollback type
		switch rollbackType {
		case string(domain.TypeCredit):
			// Rollback credit: affects the recipient's balance and transaction history
			if toUserID != nil {
				if err := s.cache.InvalidateBalanceCache(ctx, *toUserID); err != nil {
					utils.Error("failed to invalidate balance cache during rollback", "user_id", toUserID.String(), "error", err.Error())
				}
				if err := s.cache.InvalidateTransactionHistoryCache(ctx, *toUserID); err != nil {
					utils.Error("failed to invalidate transaction history cache during rollback", "user_id", toUserID.String(), "error", err.Error())
				}
			}
		case string(domain.TypeDebit):
			// Rollback debit: affects the sender's balance and transaction history
			if fromUserID != nil {
				if err := s.cache.InvalidateBalanceCache(ctx, *fromUserID); err != nil {
					utils.Error("failed to invalidate balance cache during rollback", "user_id", fromUserID.String(), "error", err.Error())
				}
				if err := s.cache.InvalidateTransactionHistoryCache(ctx, *fromUserID); err != nil {
					utils.Error("failed to invalidate transaction history cache during rollback", "user_id", fromUserID.String(), "error", err.Error())
				}
			}
		case string(domain.TypeTransfer):
			// Rollback transfer: affects both sender and receiver
			if fromUserID != nil {
				if err := s.cache.InvalidateBalanceCache(ctx, *fromUserID); err != nil {
					utils.Error("failed to invalidate sender balance cache during rollback", "user_id", fromUserID.String(), "error", err.Error())
				}
				if err := s.cache.InvalidateTransactionHistoryCache(ctx, *fromUserID); err != nil {
					utils.Error("failed to invalidate sender transaction history cache during rollback", "user_id", fromUserID.String(), "error", err.Error())
				}
			}
			if toUserID != nil {
				if err := s.cache.InvalidateBalanceCache(ctx, *toUserID); err != nil {
					utils.Error("failed to invalidate receiver balance cache during rollback", "user_id", toUserID.String(), "error", err.Error())
				}
				if err := s.cache.InvalidateTransactionHistoryCache(ctx, *toUserID); err != nil {
					utils.Error("failed to invalidate receiver transaction history cache during rollback", "user_id", toUserID.String(), "error", err.Error())
				}
			}
		}

		// Cache the rollback transaction
		if err := s.cache.CacheTransaction(ctx, rollbackTx); err != nil {
			utils.Error("failed to cache rollback transaction", "transaction_id", rollbackTx.ID.String(), "error", err.Error())
		}

		// Invalidate the original transaction cache
		if err := s.cache.InvalidateTransactionCache(ctx, originalTx.ID); err != nil {
			utils.Error("failed to invalidate original transaction cache", "transaction_id", originalTx.ID.String(), "error", err.Error())
		}
	}

	// Log the rollback audit event
	_ = s.repos.Audit.Log(ctx, "transaction", rollbackTx.ID, "rollback", map[string]interface{}{
		"original_transaction_id": originalTx.ID,
		"user_id":                 requestingUserID,
		"amount":                  originalTx.Amount,
	})

	// Increment transaction counter for metrics (rollback is also a transaction)
	s.incrementTransactionCounter()

	response := rollbackTx.ToResponse()
	return &response, nil
}

// isNotFoundError checks if an error indicates a "not found" condition.
func isNotFoundError(err error) bool {
	return err != nil && err.Error() == "balance not found for user"
}
