// Package worker provides asynchronous job processing for transaction operations.
package worker

import (
	"context"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/domain"
)

// TransactionJobType defines the type of transaction job.
type TransactionJobType string

const (
	// JobTypeCredit represents credit transaction job type
	JobTypeCredit TransactionJobType = "credit"
	// JobTypeDebit represents debit transaction job type
	JobTypeDebit TransactionJobType = "debit"
	// JobTypeTransfer represents transfer transaction job type
	JobTypeTransfer TransactionJobType = "transfer"
	// JobTypeRollback represents rollback transaction job type
	JobTypeRollback TransactionJobType = "rollback"
)

// TransactionJob represents a job for asynchronous transaction processing.
type TransactionJob struct {
	ID              uuid.UUID                  `json:"id"`
	Type            TransactionJobType         `json:"type"`
	UserID          uuid.UUID                  `json:"user_id"`
	FromUserID      *uuid.UUID                 `json:"from_user_id,omitempty"`
	ToUserID        *uuid.UUID                 `json:"to_user_id,omitempty"`
	Amount          float64                    `json:"amount"`
	OriginalTxID    *uuid.UUID                 `json:"original_tx_id,omitempty"` // For rollbacks
	CreditRequest   *domain.CreditRequest      `json:"credit_request,omitempty"`
	DebitRequest    *domain.DebitRequest       `json:"debit_request,omitempty"`
	TransferRequest *domain.TransferRequest    `json:"transfer_request,omitempty"`
	ResponseChan    chan *TransactionJobResult `json:"-"` // Channel for job results
	Ctx             context.Context            `json:"-"` // Context for cancellation
}

// TransactionJobResult represents the result of a transaction job.
type TransactionJobResult struct {
	JobID       uuid.UUID                   `json:"job_id"`
	Transaction *domain.TransactionResponse `json:"transaction,omitempty"`
	Error       error                       `json:"error,omitempty"`
	Success     bool                        `json:"success"`
}

// JobQueue represents the channels for job submission and control.
type JobQueue struct {
	SubmitChan chan *TransactionJob // Channel for submitting jobs
	QuitChan   chan struct{}        // Channel for graceful shutdown
}

// NewJobQueue creates a new job queue with the specified buffer size.
func NewJobQueue(bufferSize int) *JobQueue {
	return &JobQueue{
		SubmitChan: make(chan *TransactionJob, bufferSize),
		QuitChan:   make(chan struct{}),
	}
}

// NewTransactionJob creates a new transaction job with a unique ID and response channel.
func NewTransactionJob(ctx context.Context, jobType TransactionJobType) *TransactionJob {
	return &TransactionJob{
		ID:           uuid.New(),
		Type:         jobType,
		ResponseChan: make(chan *TransactionJobResult, 1),
		Ctx:          ctx,
	}
}

// ToResult creates a job result from the current job state.
func (j *TransactionJob) ToResult(transaction *domain.TransactionResponse, err error) *TransactionJobResult {
	result := &TransactionJobResult{
		JobID:   j.ID,
		Success: err == nil,
		Error:   err,
	}

	if transaction != nil {
		result.Transaction = transaction
	}

	return result
}
