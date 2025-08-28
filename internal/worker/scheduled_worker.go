// Package worker provides background workers for processing scheduled transactions.
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// ScheduledTransactionProcessor defines the interface for processing scheduled transactions.
type ScheduledTransactionProcessor interface {
	ProcessDueTransactions(ctx context.Context) error
}

// ScheduledWorker processes scheduled transactions that are due for execution.
type ScheduledWorker struct {
	scheduledSvc ScheduledTransactionProcessor
	ticker       *time.Ticker
	stopChan     chan struct{}
	running      bool
}

// NewScheduledWorker creates a new scheduled transaction worker.
func NewScheduledWorker(scheduledSvc ScheduledTransactionProcessor) *ScheduledWorker {
	return &ScheduledWorker{
		scheduledSvc: scheduledSvc,
		stopChan:     make(chan struct{}),
		running:      false,
	}
}

// Start begins the scheduled worker processing loop.
func (w *ScheduledWorker) Start(interval time.Duration) {
	if w.running {
		utils.Warn("scheduled worker is already running")
		return
	}

	w.running = true
	w.ticker = time.NewTicker(interval)

	utils.Info("starting scheduled transaction worker", slog.String("interval", interval.String()))

	go w.processLoop()
}

// Stop gracefully stops the scheduled worker.
func (w *ScheduledWorker) Stop(ctx context.Context) error {
	if !w.running {
		return nil
	}

	utils.Info("stopping scheduled transaction worker")

	// Signal stop
	close(w.stopChan)

	// Stop ticker
	if w.ticker != nil {
		w.ticker.Stop()
	}

	// Wait for graceful shutdown or context timeout
	done := make(chan struct{})
	go func() {
		// Wait for the processing loop to finish
		for w.running {
			time.Sleep(100 * time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
		utils.Info("scheduled transaction worker stopped gracefully")
		return nil
	case <-ctx.Done():
		utils.Warn("scheduled transaction worker stop timed out")
		return ctx.Err()
	}
}

// processLoop runs the main processing loop for scheduled transactions.
func (w *ScheduledWorker) processLoop() {
	defer func() {
		w.running = false
	}()

	for {
		select {
		case <-w.ticker.C:
			w.processDueTransactions()
		case <-w.stopChan:
			return
		}
	}
}

// processDueTransactions processes all scheduled transactions that are due.
func (w *ScheduledWorker) processDueTransactions() {
	ctx := context.Background()

	utils.Info("checking for due scheduled transactions")

	err := w.scheduledSvc.ProcessDueTransactions(ctx)
	if err != nil {
		utils.Error("failed to process due transactions", slog.String("error", err.Error()))
		return
	}

	utils.Info("completed processing due scheduled transactions")
}
