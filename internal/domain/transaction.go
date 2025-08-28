package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Transaction represents a financial transaction.
type Transaction struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	FromUserID *uuid.UUID `json:"from_user_id,omitempty" db:"from_user_id"`
	ToUserID   *uuid.UUID `json:"to_user_id,omitempty" db:"to_user_id"`
	Amount     float64    `json:"amount" db:"amount"`
	Currency   string     `json:"currency" db:"currency"`
	Type       string     `json:"type" db:"type"`
	Status     string     `json:"status" db:"status"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// TransactionType defines valid transaction types.
type TransactionType string

const (
	// TypeCredit represents credit transaction type
	TypeCredit TransactionType = "credit"
	// TypeDebit represents debit transaction type
	TypeDebit TransactionType = "debit"
	// TypeTransfer represents transfer transaction type
	TypeTransfer TransactionType = "transfer"
)

// TransactionStatus defines valid transaction statuses.
type TransactionStatus string

const (
	// StatusPending represents pending transaction status
	StatusPending TransactionStatus = "pending"
	// StatusSuccess represents success transaction status
	StatusSuccess TransactionStatus = "success"
	// StatusFailed represents failed transaction status
	StatusFailed TransactionStatus = "failed"
)

// CreateTransactionRequest represents the data needed for a transaction.
type CreateTransactionRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Type     string  `json:"type"`
}

// TransferRequest represents the data needed for a transfer transaction.
type TransferRequest struct {
	ToUserID uuid.UUID `json:"to_user_id"`
	Amount   float64   `json:"amount"`
	Currency string    `json:"currency"`
}

// CreditRequest represents the data needed for a credit transaction.
type CreditRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// DebitRequest represents the data needed for a debit transaction.
type DebitRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// TransactionResponse represents a transaction in API responses.
type TransactionResponse struct {
	ID         uuid.UUID  `json:"id"`
	FromUserID *uuid.UUID `json:"from_user_id,omitempty"`
	ToUserID   *uuid.UUID `json:"to_user_id,omitempty"`
	Amount     float64    `json:"amount"`
	Currency   string     `json:"currency"`
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ToResponse converts a Transaction to TransactionResponse.
func (t *Transaction) ToResponse() TransactionResponse {
	return TransactionResponse{
		ID:         t.ID,
		FromUserID: t.FromUserID,
		ToUserID:   t.ToUserID,
		Amount:     t.Amount,
		Currency:   t.Currency,
		Type:       t.Type,
		Status:     t.Status,
		CreatedAt:  t.CreatedAt,
	}
}

// TransactionFilter represents filters for transaction queries.
type TransactionFilter struct {
	UserID *uuid.UUID         `json:"user_id,omitempty"`
	Type   *TransactionType   `json:"type,omitempty"`
	Status *TransactionStatus `json:"status,omitempty"`
	Since  *time.Time         `json:"since,omitempty"`
	Limit  int                `json:"limit,omitempty"`
	Offset int                `json:"offset,omitempty"`
}

// validateTransactionAmount validates transaction amount.
func validateTransactionAmount(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	if amount > 1000000 { // reasonable upper limit
		return fmt.Errorf("amount cannot exceed 1,000,000")
	}

	return nil
}

// Validate validates the transaction data.
func (t *Transaction) Validate() error {
	if err := validateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	// Validate currency (default to USD if empty for backward compatibility)
	currency := t.Currency
	if currency == "" {
		currency = "USD"
	}
	if !IsValidCurrency(currency) {
		return fmt.Errorf("unsupported currency: %s", currency)
	}

	if err := validateTransactionType(t.Type); err != nil {
		return fmt.Errorf("type: %w", err)
	}

	if err := validateTransactionStatus(t.Status); err != nil {
		return fmt.Errorf("status: %w", err)
	}

	// Validate business rules based on transaction type
	if err := t.validateBusinessRules(); err != nil {
		return err
	}

	return nil
}

// Validate validates the transfer request.
func (r *TransferRequest) Validate() error {
	if err := validateTransactionAmount(r.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if !IsValidCurrency(r.Currency) {
		return fmt.Errorf("unsupported currency: %s", r.Currency)
	}

	if r.ToUserID == uuid.Nil {
		return fmt.Errorf("to_user_id is required")
	}

	return nil
}

// Validate validates the credit request.
func (r *CreditRequest) Validate() error {
	if err := validateTransactionAmount(r.Amount); err != nil {
		return err
	}

	if !IsValidCurrency(r.Currency) {
		return fmt.Errorf("unsupported currency: %s", r.Currency)
	}

	return nil
}

// Validate validates the debit request.
func (r *DebitRequest) Validate() error {
	if err := validateTransactionAmount(r.Amount); err != nil {
		return err
	}

	if !IsValidCurrency(r.Currency) {
		return fmt.Errorf("unsupported currency: %s", r.Currency)
	}

	return nil
}

// validateTransactionType validates transaction type.
func validateTransactionType(txType string) error {
	txType = strings.ToLower(txType)
	if txType != string(TypeCredit) && txType != string(TypeDebit) && txType != string(TypeTransfer) {
		return fmt.Errorf("invalid type, must be 'credit', 'debit', or 'transfer'")
	}

	return nil
}

// validateTransactionStatus validates transaction status.
func validateTransactionStatus(status string) error {
	status = strings.ToLower(status)
	if status != string(StatusPending) && status != string(StatusSuccess) && status != string(StatusFailed) {
		return fmt.Errorf("invalid status, must be 'pending', 'success', or 'failed'")
	}

	return nil
}

// validateBusinessRules validates business logic rules for transactions.
func (t *Transaction) validateBusinessRules() error {
	switch t.Type {
	case string(TypeCredit):
		if t.ToUserID == nil {
			return fmt.Errorf("credit transaction must have to_user_id")
		}
		if t.FromUserID != nil {
			return fmt.Errorf("credit transaction must not have from_user_id")
		}

	case string(TypeDebit):
		if t.FromUserID == nil {
			return fmt.Errorf("debit transaction must have from_user_id")
		}
		if t.ToUserID != nil {
			return fmt.Errorf("debit transaction must not have to_user_id")
		}

	case string(TypeTransfer):
		if t.FromUserID == nil {
			return fmt.Errorf("transfer transaction must have from_user_id")
		}
		if t.ToUserID == nil {
			return fmt.Errorf("transfer transaction must have to_user_id")
		}
		if t.FromUserID != nil && t.ToUserID != nil && *t.FromUserID == *t.ToUserID {
			return fmt.Errorf("transfer cannot be to the same user")
		}
	}

	return nil
}
