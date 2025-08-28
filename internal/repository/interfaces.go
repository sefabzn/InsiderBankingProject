// Package repository defines interfaces for data access.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/domain"
)

// UsersRepo defines the interface for user data operations.
type UsersRepo interface {
	// Create creates a new user.
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// GetByEmail retrieves a user by email.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// GetByUsername retrieves a user by username.
	GetByUsername(ctx context.Context, username string) (*domain.User, error)

	// Update updates an existing user.
	Update(ctx context.Context, user *domain.User) error

	// Delete deletes a user by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// ListPaginated retrieves users with pagination.
	ListPaginated(ctx context.Context, limit, offset int) ([]*domain.User, error)

	// ListAll retrieves all users without pagination (for testing purposes).
	ListAll(ctx context.Context) ([]*domain.User, error)

	// Count returns the total number of users.
	Count(ctx context.Context) (int, error)
}

// BalancesRepo defines the interface for balance data operations.
type BalancesRepo interface {
	// GetByUserID retrieves a balance by user ID.
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Balance, error)

	// Upsert creates or updates a balance.
	Upsert(ctx context.Context, balance *domain.Balance) error

	// AddAmountTx adds amount to a user's balance within a transaction.
	// This method should be used within database transactions for atomicity.
	AddAmountTx(ctx context.Context, tx interface{}, userID uuid.UUID, delta float64) error

	// GetHistorical retrieves historical balance snapshots.
	GetHistorical(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.BalanceHistoryItem, error)

	// GetAtTime retrieves balance at a specific time.
	GetAtTime(ctx context.Context, userID uuid.UUID, timestamp string) (*domain.Balance, error)
}

// TransactionsRepo defines the interface for transaction data operations.
type TransactionsRepo interface {
	// CreatePending creates a new transaction with pending status.
	CreatePending(ctx context.Context, tx *domain.Transaction) error

	// MarkCompleted marks a transaction as completed.
	MarkCompleted(ctx context.Context, id uuid.UUID) error

	// MarkFailed marks a transaction as failed.
	MarkFailed(ctx context.Context, id uuid.UUID) error

	// GetByID retrieves a transaction by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)

	// ListForUser retrieves transactions for a specific user.
	ListForUser(ctx context.Context, userID uuid.UUID, filter *domain.TransactionFilter) ([]*domain.Transaction, error)

	// List retrieves transactions with filtering.
	List(ctx context.Context, filter *domain.TransactionFilter) ([]*domain.Transaction, error)

	// Count returns the total number of transactions matching the filter.
	Count(ctx context.Context, filter *domain.TransactionFilter) (int, error)
}

// AuditRepo defines the interface for audit log operations.
type AuditRepo interface {
	// Log creates a new audit log entry.
	Log(ctx context.Context, entityType string, entityID uuid.UUID, action string, details interface{}) error

	// GetByID retrieves an audit log by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error)

	// List retrieves audit logs with filtering.
	List(ctx context.Context, filter *domain.AuditLogFilter) ([]*domain.AuditLog, error)

	// ListForEntity retrieves audit logs for a specific entity.
	ListForEntity(ctx context.Context, entityType string, entityID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error)

	// Count returns the total number of audit logs matching the filter.
	Count(ctx context.Context, filter *domain.AuditLogFilter) (int, error)
}

// EventsRepo defines the interface for event sourcing operations.
type EventsRepo interface {
	// AppendEvent appends a new event to the event store
	AppendEvent(ctx context.Context, event *domain.Event) (*domain.Event, error)

	// GetEventsByAggregate retrieves all events for a specific aggregate
	GetEventsByAggregate(ctx context.Context, aggregateType domain.AggregateType, aggregateID uuid.UUID) ([]*domain.Event, error)

	// GetEventsByType retrieves events by event type
	GetEventsByType(ctx context.Context, eventType domain.EventType, limit int, offset int) ([]*domain.Event, error)

	// GetEventsSince retrieves events since a specific time
	GetEventsSince(ctx context.Context, since time.Time, limit int) ([]*domain.Event, error)

	// GetAggregateVersion returns the current version of an aggregate
	GetAggregateVersion(ctx context.Context, aggregateType domain.AggregateType, aggregateID uuid.UUID) (int, error)

	// AppendEvents appends multiple events in a single transaction
	AppendEvents(ctx context.Context, events []*domain.Event) error

	// LoadEventEnvelope loads an event with its deserialized data
	LoadEventEnvelope(ctx context.Context, event *domain.Event, target interface{}) (*EventEnvelope, error)
}

// ScheduledTransactionsRepo defines the interface for scheduled transaction operations.
type ScheduledTransactionsRepo interface {
	// Create creates a new scheduled transaction
	Create(ctx context.Context, st *domain.ScheduledTransaction) error

	// GetByID retrieves a scheduled transaction by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ScheduledTransaction, error)

	// GetByUserID retrieves scheduled transactions for a user
	GetByUserID(ctx context.Context, userID uuid.UUID, filter *domain.ScheduledTransactionFilter) ([]*domain.ScheduledTransaction, error)

	// GetDueForExecution retrieves scheduled transactions that are due for execution
	GetDueForExecution(ctx context.Context, limit int) ([]*domain.ScheduledTransaction, error)

	// Update updates a scheduled transaction
	Update(ctx context.Context, st *domain.ScheduledTransaction) error

	// ResetStatus resets the status of a scheduled transaction (used for error recovery)
	ResetStatus(ctx context.Context, id uuid.UUID, status string) error

	// Delete deletes a scheduled transaction
	Delete(ctx context.Context, id uuid.UUID) error

	// CreateExecution creates an execution record
	CreateExecution(ctx context.Context, execution *domain.ScheduledTransactionExecution) error

	// GetExecutions retrieves execution history for a scheduled transaction
	GetExecutions(ctx context.Context, scheduledTransactionID uuid.UUID, limit int, offset int) ([]*domain.ScheduledTransactionExecution, error)

	// Count counts scheduled transactions matching the filter
	Count(ctx context.Context, userID uuid.UUID, filter *domain.ScheduledTransactionFilter) (int, error)
}

// Repositories aggregates all repository interfaces.
type Repositories struct {
	Users                 UsersRepo
	Balances              BalancesRepo
	Transactions          TransactionsRepo
	Audit                 AuditRepo
	Events                EventsRepo
	ScheduledTransactions ScheduledTransactionsRepo
}
