// Package service provides business logic for scheduled transactions.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/domain"
	"github.com/sefa-b/go-banking-sim/internal/repository"
)

// ScheduledTransactionServiceImpl implements ScheduledTransactionService.
type ScheduledTransactionServiceImpl struct {
	repos          *repository.Repositories
	transactionSvc TransactionService
}

// NewScheduledTransactionService creates a new scheduled transaction service.
func NewScheduledTransactionService(repos *repository.Repositories, transactionSvc TransactionService) ScheduledTransactionService {
	return &ScheduledTransactionServiceImpl{
		repos:          repos,
		transactionSvc: transactionSvc,
	}
}

// Create creates a new scheduled transaction.
func (s *ScheduledTransactionServiceImpl) Create(ctx context.Context, userID uuid.UUID, req *domain.ScheduledTransactionRequest) (*domain.ScheduledTransactionResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Normalize schedule type for internal consistency
	if req.ScheduleType == "once" {
		req.ScheduleType = "one-time"
	}

	// Validate transaction type and related fields
	switch req.TransactionType {
	case "credit":
		if req.ToUserID != nil {
			return nil, fmt.Errorf("credit transactions cannot have to_user_id")
		}
	case "debit":
		if req.ToUserID != nil {
			return nil, fmt.Errorf("debit transactions cannot have to_user_id")
		}
	case "transfer":
		if req.ToUserID == nil {
			return nil, fmt.Errorf("transfer transactions require to_user_id")
		}
		if *req.ToUserID == userID {
			return nil, fmt.Errorf("cannot transfer to self")
		}
	default:
		return nil, fmt.Errorf("invalid transaction type: %s", req.TransactionType)
	}

	// Create scheduled transaction
	st := &domain.ScheduledTransaction{
		ID:                uuid.New(),
		UserID:            userID,
		TransactionType:   req.TransactionType,
		Amount:            req.Amount,
		Currency:          req.Currency,
		Description:       req.Description,
		ToUserID:          req.ToUserID,
		ScheduleType:      req.ScheduleType,
		ExecuteAt:         req.ExecuteAt,
		CurrentOccurrence: 0,
		Status:            "active",
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Handle optional pointer fields
	st.RecurrencePattern = req.RecurrencePattern
	st.RecurrenceEndDate = req.RecurrenceEndDate
	st.MaxOccurrences = req.MaxOccurrences

	// Calculate next execution time
	st.NextExecutionAt = st.CalculateNextExecution()

	// Save to database
	if err := s.repos.ScheduledTransactions.Create(ctx, st); err != nil {
		return nil, fmt.Errorf("failed to create scheduled transaction: %w", err)
	}

	// Convert to response
	response := st.ToResponse()
	return &response, nil
}

// GetByID retrieves a scheduled transaction by ID.
func (s *ScheduledTransactionServiceImpl) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.ScheduledTransactionResponse, error) {
	st, err := s.repos.ScheduledTransactions.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled transaction: %w", err)
	}

	// Check ownership
	if st.UserID != userID {
		return nil, fmt.Errorf("access denied: not owner of scheduled transaction")
	}

	response := st.ToResponse()
	return &response, nil
}

// List retrieves scheduled transactions for a user.
func (s *ScheduledTransactionServiceImpl) List(ctx context.Context, userID uuid.UUID, filter *domain.ScheduledTransactionFilter) ([]*domain.ScheduledTransactionResponse, error) {
	transactions, err := s.repos.ScheduledTransactions.GetByUserID(ctx, userID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled transactions: %w", err)
	}

	var responses []*domain.ScheduledTransactionResponse
	for _, st := range transactions {
		response := st.ToResponse()
		responses = append(responses, &response)
	}

	return responses, nil
}

// Cancel cancels a scheduled transaction.
func (s *ScheduledTransactionServiceImpl) Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// Get the transaction first to verify ownership
	st, err := s.repos.ScheduledTransactions.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get scheduled transaction: %w", err)
	}

	// Check ownership
	if st.UserID != userID {
		return fmt.Errorf("access denied: not owner of scheduled transaction")
	}

	// Update status
	st.Status = "cancelled"
	st.IsActive = false
	st.UpdatedAt = time.Now()

	if err := s.repos.ScheduledTransactions.Update(ctx, st); err != nil {
		return fmt.Errorf("failed to cancel scheduled transaction: %w", err)
	}

	return nil
}

// ProcessDueTransactions processes all scheduled transactions that are due for execution.
func (s *ScheduledTransactionServiceImpl) ProcessDueTransactions(ctx context.Context) error {
	fmt.Printf("ProcessDueTransactions: Starting to check for due transactions\n")

	// Get all due transactions
	dueTransactions, err := s.repos.ScheduledTransactions.GetDueForExecution(ctx, 100) // Process up to 100 at a time
	if err != nil {
		fmt.Printf("ProcessDueTransactions: Error getting due transactions: %v\n", err)
		return fmt.Errorf("failed to get due transactions: %w", err)
	}

	fmt.Printf("ProcessDueTransactions: Found %d due transactions to process\n", len(dueTransactions))

	for _, st := range dueTransactions {
		fmt.Printf("ProcessDueTransactions: Processing transaction %s (type: %s, status: %s, active: %t)\n",
			st.ID, st.TransactionType, st.Status, st.IsActive)
		if err := s.processScheduledTransaction(ctx, st); err != nil {
			// Log error but continue processing other transactions
			fmt.Printf("Failed to process scheduled transaction %s: %v\n", st.ID, err)
		}
	}

	fmt.Printf("ProcessDueTransactions: Completed processing %d transactions\n", len(dueTransactions))
	return nil
}

// processScheduledTransaction executes a single scheduled transaction.
func (s *ScheduledTransactionServiceImpl) processScheduledTransaction(ctx context.Context, st *domain.ScheduledTransaction) error {
	// Skip if already completed
	if !st.IsActive && st.Status == "completed" {
		return nil // Already completed, skip silently
	}

	// Log processing for debugging
	fmt.Printf("Processing scheduled transaction: ID=%s, Status=%s, IsActive=%t, LastExecutedAt=%v\n",
		st.ID, st.Status, st.IsActive, st.LastExecutedAt)

	var transactionResponse *domain.TransactionResponse
	var err error

	// Execute based on transaction type
	switch st.TransactionType {
	case "credit":
		creditReq := &domain.CreditRequest{
			Amount:   st.Amount,
			Currency: st.Currency,
		}
		transactionResponse, err = s.transactionSvc.CreditSync(ctx, st.UserID, creditReq)

	case "debit":
		debitReq := &domain.DebitRequest{
			Amount:   st.Amount,
			Currency: st.Currency,
		}
		transactionResponse, err = s.transactionSvc.DebitSync(ctx, st.UserID, debitReq)

	case "transfer":
		if st.ToUserID == nil {
			return fmt.Errorf("transfer transaction missing to_user_id")
		}
		transferReq := &domain.TransferRequest{
			ToUserID: *st.ToUserID,
			Amount:   st.Amount,
			Currency: st.Currency,
		}
		transactionResponse, err = s.transactionSvc.TransferSync(ctx, st.UserID, transferReq)

	default:
		return fmt.Errorf("unknown transaction type: %s", st.TransactionType)
	}

	if err != nil {
		// Create execution record with failure
		execution := &domain.ScheduledTransactionExecution{
			ID:                     uuid.New(),
			ScheduledTransactionID: st.ID,
			ExecutedAt:             time.Now(),
			Status:                 "failed",
			ErrorMessage:           err.Error(),
			Amount:                 st.Amount,
			Currency:               st.Currency,
		}
		if err := s.repos.ScheduledTransactions.CreateExecution(ctx, execution); err != nil {
			return fmt.Errorf("failed to create execution record: %w", err)
		}
		return fmt.Errorf("transaction execution failed: %w", err)
	}

	// Create execution record with success
	execution := &domain.ScheduledTransactionExecution{
		ID:                     uuid.New(),
		ScheduledTransactionID: st.ID,
		ExecutedAt:             time.Now(),
		Status:                 "success",
		TransactionID:          &transactionResponse.ID,
		Amount:                 st.Amount,
		Currency:               st.Currency,
	}
	if err := s.repos.ScheduledTransactions.CreateExecution(ctx, execution); err != nil {
		return fmt.Errorf("failed to create execution record: %w", err)
	}

	// Update scheduled transaction
	st.LastExecutedAt = &execution.ExecutedAt
	st.CurrentOccurrence++

	// Check if we should deactivate based on recurrence rules
	if st.MaxOccurrences != nil && st.CurrentOccurrence >= *st.MaxOccurrences {
		st.IsActive = false
		st.Status = "completed"
	} else if st.RecurrenceEndDate != nil && time.Now().After(*st.RecurrenceEndDate) {
		st.IsActive = false
		st.Status = "completed"
	} else if st.ScheduleType == "once" || st.ScheduleType == "one-time" {
		// One-time transactions should be deactivated after execution
		st.IsActive = false
		st.Status = "completed"
	} else {
		// For recurring transactions, reset status to active for next execution
		st.Status = "active"
		st.NextExecutionAt = st.CalculateNextExecution()
	}

	st.UpdatedAt = time.Now()

	if err := s.repos.ScheduledTransactions.Update(ctx, st); err != nil {
		return fmt.Errorf("failed to update scheduled transaction: %w", err)
	}

	return nil
}
