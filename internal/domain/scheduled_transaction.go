package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ScheduledTransaction represents a scheduled or recurring transaction
type ScheduledTransaction struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	TransactionType string     `json:"transaction_type" db:"transaction_type"`
	Amount          float64    `json:"amount" db:"amount"`
	Currency        string     `json:"currency" db:"currency"`
	Description     string     `json:"description,omitempty" db:"description"`
	ToUserID        *uuid.UUID `json:"to_user_id,omitempty" db:"to_user_id"`

	// Scheduling
	ScheduleType      string     `json:"schedule_type" db:"schedule_type"`
	ExecuteAt         time.Time  `json:"execute_at" db:"execute_at"`
	RecurrencePattern *string    `json:"recurrence_pattern,omitempty" db:"recurrence_pattern"`
	RecurrenceEndDate *time.Time `json:"recurrence_end_date,omitempty" db:"recurrence_end_date"`
	MaxOccurrences    *int       `json:"max_occurrences,omitempty" db:"max_occurrences"`
	CurrentOccurrence int        `json:"current_occurrence" db:"current_occurrence"`

	// Status
	Status   string `json:"status" db:"status"`
	IsActive bool   `json:"is_active" db:"is_active"`

	// Audit
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	LastExecutedAt  *time.Time `json:"last_executed_at,omitempty" db:"last_executed_at"`
	NextExecutionAt *time.Time `json:"next_execution_at,omitempty" db:"next_execution_at"`
}

// ScheduledTransactionResponse represents scheduled transaction for API responses
type ScheduledTransactionResponse struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	TransactionType string     `json:"transaction_type"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	Description     string     `json:"description,omitempty"`
	ToUserID        *uuid.UUID `json:"to_user_id,omitempty"`

	ScheduleType      string     `json:"schedule_type"`
	ExecuteAt         time.Time  `json:"execute_at"`
	RecurrencePattern *string    `json:"recurrence_pattern,omitempty"`
	RecurrenceEndDate *time.Time `json:"recurrence_end_date,omitempty"`
	MaxOccurrences    *int       `json:"max_occurrences,omitempty"`
	CurrentOccurrence int        `json:"current_occurrence"`

	Status   string `json:"status"`
	IsActive bool   `json:"is_active"`

	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LastExecutedAt  *time.Time `json:"last_executed_at,omitempty"`
	NextExecutionAt *time.Time `json:"next_execution_at,omitempty"`
}

// ToResponse converts ScheduledTransaction to response
func (st *ScheduledTransaction) ToResponse() ScheduledTransactionResponse {
	return ScheduledTransactionResponse{
		ID:                st.ID,
		UserID:            st.UserID,
		TransactionType:   st.TransactionType,
		Amount:            st.Amount,
		Currency:          st.Currency,
		Description:       st.Description,
		ToUserID:          st.ToUserID,
		ScheduleType:      st.ScheduleType,
		ExecuteAt:         st.ExecuteAt,
		RecurrencePattern: st.RecurrencePattern,
		RecurrenceEndDate: st.RecurrenceEndDate,
		MaxOccurrences:    st.MaxOccurrences,
		CurrentOccurrence: st.CurrentOccurrence,
		Status:            st.Status,
		IsActive:          st.IsActive,
		CreatedAt:         st.CreatedAt,
		UpdatedAt:         st.UpdatedAt,
		LastExecutedAt:    st.LastExecutedAt,
		NextExecutionAt:   st.NextExecutionAt,
	}
}

// ScheduledTransactionRequest represents request to create scheduled transaction
type ScheduledTransactionRequest struct {
	TransactionType string     `json:"transaction_type"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	Description     string     `json:"description,omitempty"`
	ToUserID        *uuid.UUID `json:"to_user_id,omitempty"`

	ScheduleType      string     `json:"schedule_type"`
	ExecuteAt         time.Time  `json:"execute_at"`
	RecurrencePattern *string    `json:"recurrence_pattern,omitempty"`
	RecurrenceEndDate *time.Time `json:"recurrence_end_date,omitempty"`
	MaxOccurrences    *int       `json:"max_occurrences,omitempty"`
}

// Validate validates the scheduled transaction request
func (r *ScheduledTransactionRequest) Validate() error {
	// Validate transaction type
	if r.TransactionType != "credit" && r.TransactionType != "debit" && r.TransactionType != "transfer" {
		return fmt.Errorf("invalid transaction_type: must be 'credit', 'debit', or 'transfer'")
	}

	// Validate amount
	if err := validateTransactionAmount(r.Amount); err != nil {
		return err
	}

	// Validate currency
	if !IsValidCurrency(r.Currency) {
		return fmt.Errorf("unsupported currency: %s", r.Currency)
	}

	// Validate schedule type (accept both "once" and "one-time" for better UX)
	if r.ScheduleType != "once" && r.ScheduleType != "one-time" && r.ScheduleType != "recurring" {
		return fmt.Errorf("invalid schedule_type: must be 'once', 'one-time', or 'recurring'")
	}

	// Validate execute time
	if r.ExecuteAt.Before(time.Now()) {
		return fmt.Errorf("execute_at must be in the future")
	}

	// Validate transfer-specific fields
	if r.TransactionType == "transfer" {
		if r.ToUserID == nil {
			return fmt.Errorf("to_user_id is required for transfer transactions")
		}
	} else {
		if r.ToUserID != nil {
			return fmt.Errorf("to_user_id should not be provided for %s transactions", r.TransactionType)
		}
	}

	// Validate recurring options
	if r.ScheduleType == "recurring" {
		if r.RecurrencePattern == nil {
			return fmt.Errorf("recurrence_pattern is required for recurring transactions")
		}
		validPatterns := []string{"daily", "weekly", "monthly", "yearly"}
		found := false
		for _, pattern := range validPatterns {
			if *r.RecurrencePattern == pattern {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid recurrence_pattern: must be 'daily', 'weekly', 'monthly', or 'yearly'")
		}

		if r.RecurrenceEndDate != nil && r.RecurrenceEndDate.Before(r.ExecuteAt) {
			return fmt.Errorf("recurrence_end_date must be after execute_at")
		}

		if r.MaxOccurrences != nil && *r.MaxOccurrences <= 0 {
			return fmt.Errorf("max_occurrences must be greater than 0")
		}
	} else {
		// For one-time, these should not be set
		if r.RecurrencePattern != nil {
			return fmt.Errorf("recurrence_pattern should not be set for one-time transactions")
		}
		if r.RecurrenceEndDate != nil {
			return fmt.Errorf("recurrence_end_date should not be set for one-time transactions")
		}
		if r.MaxOccurrences != nil {
			return fmt.Errorf("max_occurrences should not be set for one-time transactions")
		}
	}

	return nil
}

// ScheduledTransactionExecution represents execution history
type ScheduledTransactionExecution struct {
	ID                     uuid.UUID  `json:"id" db:"id"`
	ScheduledTransactionID uuid.UUID  `json:"scheduled_transaction_id" db:"scheduled_transaction_id"`
	ExecutedAt             time.Time  `json:"executed_at" db:"executed_at"`
	Status                 string     `json:"status" db:"status"`
	TransactionID          *uuid.UUID `json:"transaction_id,omitempty" db:"transaction_id"`
	ErrorMessage           string     `json:"error_message,omitempty" db:"error_message"`
	Amount                 float64    `json:"amount" db:"amount"`
	Currency               string     `json:"currency" db:"currency"`
}

// ScheduledTransactionExecutionResponse represents execution for API responses
type ScheduledTransactionExecutionResponse struct {
	ID                     uuid.UUID  `json:"id"`
	ScheduledTransactionID uuid.UUID  `json:"scheduled_transaction_id"`
	ExecutedAt             time.Time  `json:"executed_at"`
	Status                 string     `json:"status"`
	TransactionID          *uuid.UUID `json:"transaction_id,omitempty"`
	ErrorMessage           string     `json:"error_message,omitempty"`
	Amount                 float64    `json:"amount"`
	Currency               string     `json:"currency"`
}

// ToResponse converts execution to response
func (e *ScheduledTransactionExecution) ToResponse() ScheduledTransactionExecutionResponse {
	return ScheduledTransactionExecutionResponse{
		ID:                     e.ID,
		ScheduledTransactionID: e.ScheduledTransactionID,
		ExecutedAt:             e.ExecutedAt,
		Status:                 e.Status,
		TransactionID:          e.TransactionID,
		ErrorMessage:           e.ErrorMessage,
		Amount:                 e.Amount,
		Currency:               e.Currency,
	}
}

// ScheduledTransactionFilter represents filters for scheduled transaction queries
type ScheduledTransactionFilter struct {
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Status      *string    `json:"status,omitempty"`
	Type        *string    `json:"type,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
	ExecuteFrom *time.Time `json:"execute_from,omitempty"`
	ExecuteTo   *time.Time `json:"execute_to,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
}

// ScheduledTransactionUpdateRequest represents request to update scheduled transaction
type ScheduledTransactionUpdateRequest struct {
	Description       *string    `json:"description,omitempty"`
	Status            *string    `json:"status,omitempty"`
	IsActive          *bool      `json:"is_active,omitempty"`
	ExecuteAt         *time.Time `json:"execute_at,omitempty"`
	RecurrenceEndDate *time.Time `json:"recurrence_end_date,omitempty"`
	MaxOccurrences    *int       `json:"max_occurrences,omitempty"`
}

// Validate validates the update request
func (r *ScheduledTransactionUpdateRequest) Validate() error {
	if r.Status != nil {
		validStatuses := []string{"active", "paused", "cancelled"}
		valid := false
		for _, status := range validStatuses {
			if *r.Status == status {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid status: must be 'active', 'paused', or 'cancelled'")
		}
	}

	if r.ExecuteAt != nil && r.ExecuteAt.Before(time.Now()) {
		return fmt.Errorf("execute_at must be in the future")
	}

	if r.RecurrenceEndDate != nil && r.ExecuteAt != nil && r.RecurrenceEndDate.Before(*r.ExecuteAt) {
		return fmt.Errorf("recurrence_end_date must be after execute_at")
	}

	if r.MaxOccurrences != nil && *r.MaxOccurrences <= 0 {
		return fmt.Errorf("max_occurrences must be greater than 0")
	}

	return nil
}

// CalculateNextExecution calculates the next execution time for recurring transactions
func (st *ScheduledTransaction) CalculateNextExecution() *time.Time {
	if st.ScheduleType != "recurring" || st.RecurrencePattern == nil {
		return nil
	}

	var nextTime time.Time
	baseTime := st.ExecuteAt

	// If this is not the first occurrence, use last execution time as base
	if st.LastExecutedAt != nil {
		baseTime = *st.LastExecutedAt
	}

	switch *st.RecurrencePattern {
	case "daily":
		nextTime = baseTime.AddDate(0, 0, 1)
	case "weekly":
		nextTime = baseTime.AddDate(0, 0, 7)
	case "monthly":
		nextTime = baseTime.AddDate(0, 1, 0)
	case "yearly":
		nextTime = baseTime.AddDate(1, 0, 0)
	default:
		return nil
	}

	// Check if we've reached the end conditions
	if st.RecurrenceEndDate != nil && nextTime.After(*st.RecurrenceEndDate) {
		return nil
	}

	if st.MaxOccurrences != nil && st.CurrentOccurrence+1 >= *st.MaxOccurrences {
		return nil
	}

	return &nextTime
}

// ShouldExecute checks if the scheduled transaction should be executed
func (st *ScheduledTransaction) ShouldExecute() bool {
	if !st.IsActive || st.Status != "active" {
		return false
	}

	now := time.Now()
	return st.ExecuteAt.Before(now) || st.ExecuteAt.Equal(now)
}

// MarkExecuted updates the scheduled transaction after successful execution
func (st *ScheduledTransaction) MarkExecuted(executionTime time.Time) {
	st.LastExecutedAt = &executionTime
	st.CurrentOccurrence++
	st.UpdatedAt = executionTime

	if st.ScheduleType == "recurring" {
		nextExecution := st.CalculateNextExecution()
		if nextExecution != nil {
			st.NextExecutionAt = nextExecution
			st.ExecuteAt = *nextExecution
		} else {
			// No more executions
			st.Status = "completed"
			st.IsActive = false
		}
	} else {
		// One-time transaction completed
		st.Status = "completed"
		st.IsActive = false
	}
}

// MarkFailed updates the scheduled transaction after failed execution
func (st *ScheduledTransaction) MarkFailed() {
	st.UpdatedAt = time.Now()

	// For recurring transactions, we might want to retry later
	// For now, we'll pause the transaction
	if st.ScheduleType == "recurring" {
		st.Status = "paused"
	} else {
		st.Status = "cancelled"
		st.IsActive = false
	}
}
