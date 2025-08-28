// Package worker provides asynchronous job processing for transaction operations.
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sefa-b/go-banking-sim/internal/domain"
	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// TransactionService defines the interface for transaction operations needed by the worker pool.
type TransactionService interface {
	CreditSync(ctx context.Context, userID string, req interface{}) (interface{}, error)
	DebitSync(ctx context.Context, userID string, req interface{}) (interface{}, error)
	TransferSync(ctx context.Context, fromUserID string, req interface{}) (interface{}, error)
	RollbackSync(ctx context.Context, transactionID string, requestingUserID string) (interface{}, error)
}

// Pool manages a pool of workers that process transaction jobs asynchronously.
type Pool struct {
	jobQueue       *JobQueue
	transactionSvc TransactionService
	workers        []*Worker
	wg             sync.WaitGroup
	stopped        chan struct{}
	jobsProcessed  int64
	mu             sync.RWMutex
}

// Worker represents a single worker in the pool.
type Worker struct {
	id       int
	jobQueue *JobQueue
	svc      TransactionService
	stopped  chan struct{}
}

// Stats represents worker pool statistics.
type Stats struct {
	ActiveWorkers int   `json:"active_workers"`
	JobsProcessed int64 `json:"jobs_processed"`
	QueueSize     int   `json:"queue_size"`
}

// NewPool creates a new worker pool.
func NewPool(jobQueue *JobQueue, transactionSvc TransactionService) *Pool {
	return &Pool{
		jobQueue:       jobQueue,
		transactionSvc: transactionSvc,
		stopped:        make(chan struct{}),
	}
}

// Start starts the specified number of workers.
func (wp *Pool) Start(numWorkers int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	utils.Info("starting worker pool",
		slog.Int("num_workers", numWorkers),
	)

	for i := 0; i < numWorkers; i++ {
		worker := &Worker{
			id:       i + 1,
			jobQueue: wp.jobQueue,
			svc:      wp.transactionSvc,
			stopped:  make(chan struct{}),
		}

		wp.workers = append(wp.workers, worker)

		wp.wg.Add(1)
		go worker.start(&wp.wg, &wp.jobsProcessed)
	}

	utils.Info("worker pool started successfully",
		slog.Int("num_workers", len(wp.workers)),
	)
}

// Stop gracefully stops all workers.
func (wp *Pool) Stop(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	utils.Info("stopping worker pool",
		slog.Int("active_workers", len(wp.workers)),
	)

	// Close quit channel to signal workers to stop
	close(wp.jobQueue.QuitChan)

	// Wait for all workers to finish or context timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		utils.Info("worker pool stopped gracefully")
	case <-ctx.Done():
		utils.Warn("worker pool shutdown timed out")
		return ctx.Err()
	}

	// Close stopped channel to signal complete shutdown
	close(wp.stopped)
	return nil
}

// SubmitJob submits a job to the worker pool.
func (wp *Pool) SubmitJob(job *TransactionJob) {
	select {
	case wp.jobQueue.SubmitChan <- job:
		utils.Debug("job submitted successfully",
			slog.String("job_id", job.ID.String()),
			slog.String("type", string(job.Type)),
		)
	default:
		// Queue is full, return error via response channel
		result := job.ToResult(nil, fmt.Errorf("job queue is full"))
		select {
		case job.ResponseChan <- result:
		default:
			utils.Warn("could not send job result - response channel full",
				slog.String("job_id", job.ID.String()),
			)
		}
	}
}

// GetStats returns current worker pool statistics.
func (wp *Pool) GetStats() Stats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	return Stats{
		ActiveWorkers: len(wp.workers),
		JobsProcessed: atomic.LoadInt64(&wp.jobsProcessed),
		QueueSize:     len(wp.jobQueue.SubmitChan),
	}
}

// IsStopped returns whether the worker pool has been stopped.
func (wp *Pool) IsStopped() bool {
	select {
	case <-wp.stopped:
		return true
	default:
		return false
	}
}

// start begins processing jobs for a worker.
func (w *Worker) start(wg *sync.WaitGroup, jobsProcessed *int64) {
	defer wg.Done()

	utils.Info("worker started",
		slog.Int("worker_id", w.id),
	)

	for {
		select {
		case job := <-w.jobQueue.SubmitChan:
			w.processJob(job, jobsProcessed)

		case <-w.stopped:
			utils.Info("worker stopped",
				slog.Int("worker_id", w.id),
			)
			return
		}
	}
}

// processJob processes a single transaction job.
func (w *Worker) processJob(job *TransactionJob, jobsProcessed *int64) {
	startTime := time.Now()

	utils.Debug("processing job",
		slog.String("job_id", job.ID.String()),
		slog.String("type", string(job.Type)),
		slog.Int("worker_id", w.id),
	)

	var result *TransactionJobResult
	var err error

	// Process the job based on its type
	switch job.Type {
	case JobTypeCredit:
		result, err = w.processCredit(job)
	case JobTypeDebit:
		result, err = w.processDebit(job)
	case JobTypeTransfer:
		result, err = w.processTransfer(job)
	case JobTypeRollback:
		result, err = w.processRollback(job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.Type)
		result = job.ToResult(nil, err)
	}

	if err != nil {
		utils.Error("job processing failed",
			slog.String("job_id", job.ID.String()),
			slog.String("type", string(job.Type)),
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(startTime)),
		)
		result = job.ToResult(nil, err)
	} else {
		utils.Info("job processed successfully",
			slog.String("job_id", job.ID.String()),
			slog.String("type", string(job.Type)),
			slog.Duration("duration", time.Since(startTime)),
		)
	}

	// Send result back via response channel
	select {
	case job.ResponseChan <- result:
		atomic.AddInt64(jobsProcessed, 1)
	case <-time.After(5 * time.Second):
		utils.Warn("timeout sending job result",
			slog.String("job_id", job.ID.String()),
		)
	}
}

// processCredit processes a credit job.
func (w *Worker) processCredit(job *TransactionJob) (*TransactionJobResult, error) {
	if job.CreditRequest == nil {
		return job.ToResult(nil, fmt.Errorf("invalid credit job: missing credit_request")), nil
	}

	transaction, err := w.svc.CreditSync(job.Ctx, job.UserID.String(), job.CreditRequest)
	if err != nil {
		return job.ToResult(nil, err), nil
	}

	// Type assert the result
	txResponse, ok := transaction.(*domain.TransactionResponse)
	if !ok {
		return job.ToResult(nil, fmt.Errorf("invalid response type from credit operation")), nil
	}

	return job.ToResult(txResponse, nil), nil
}

// processDebit processes a debit job.
func (w *Worker) processDebit(job *TransactionJob) (*TransactionJobResult, error) {
	if job.DebitRequest == nil {
		return job.ToResult(nil, fmt.Errorf("invalid debit job: missing debit_request")), nil
	}

	transaction, err := w.svc.DebitSync(job.Ctx, job.UserID.String(), job.DebitRequest)
	if err != nil {
		return job.ToResult(nil, err), nil
	}

	// Type assert the result
	txResponse, ok := transaction.(*domain.TransactionResponse)
	if !ok {
		return job.ToResult(nil, fmt.Errorf("invalid response type from debit operation")), nil
	}

	return job.ToResult(txResponse, nil), nil
}

// processTransfer processes a transfer job.
func (w *Worker) processTransfer(job *TransactionJob) (*TransactionJobResult, error) {
	if job.FromUserID == nil || job.TransferRequest == nil {
		return job.ToResult(nil, fmt.Errorf("invalid transfer job: missing from_user_id or transfer_request")), nil
	}

	transaction, err := w.svc.TransferSync(job.Ctx, job.FromUserID.String(), job.TransferRequest)
	if err != nil {
		return job.ToResult(nil, err), nil
	}

	// Type assert the result
	txResponse, ok := transaction.(*domain.TransactionResponse)
	if !ok {
		return job.ToResult(nil, fmt.Errorf("invalid response type from transfer operation")), nil
	}

	return job.ToResult(txResponse, nil), nil
}

// processRollback processes a rollback job.
func (w *Worker) processRollback(job *TransactionJob) (*TransactionJobResult, error) {
	if job.OriginalTxID == nil {
		return job.ToResult(nil, fmt.Errorf("invalid rollback job: missing original_tx_id")), nil
	}

	transaction, err := w.svc.RollbackSync(job.Ctx, job.OriginalTxID.String(), job.UserID.String())
	if err != nil {
		return job.ToResult(nil, err), nil
	}

	// Type assert the result
	txResponse, ok := transaction.(*domain.TransactionResponse)
	if !ok {
		return job.ToResult(nil, fmt.Errorf("invalid response type from rollback operation")), nil
	}

	return job.ToResult(txResponse, nil), nil
}
