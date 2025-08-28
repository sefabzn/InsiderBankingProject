// Package worker provides background workers for processing events and updating projectors
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/sefa-b/go-banking-sim/internal/service"
	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// ProjectorWorker processes events and updates read models through projectors
type ProjectorWorker struct {
	projectorSvc service.ProjectorServiceInterface
	dbPool       interface{} // Database pool for locking
	ticker       *time.Ticker
	stopChan     chan struct{}
	running      bool
}

// ProjectorServiceInterface defines the interface for projector services
type ProjectorServiceInterface interface {
	ProcessEventsSince(ctx context.Context, since time.Time) error
	ProcessAllEvents(ctx context.Context) error
}

// NewProjectorWorker creates a new projector worker
func NewProjectorWorker(projectorSvc service.ProjectorServiceInterface) *ProjectorWorker {
	return &ProjectorWorker{
		projectorSvc: projectorSvc,
		stopChan:     make(chan struct{}),
		running:      false,
	}
}

// Start begins the projector processing loop
func (w *ProjectorWorker) Start(interval time.Duration) {
	if w.running {
		utils.Warn("projector worker is already running")
		return
	}

	w.running = true
	w.ticker = time.NewTicker(interval)

	utils.Info("starting event projector worker", slog.String("interval", interval.String()))

	go w.processLoop()
}

// Stop gracefully stops the projector worker
func (w *ProjectorWorker) Stop(ctx context.Context) error {
	if !w.running {
		return nil
	}

	utils.Info("stopping event projector worker")

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
		utils.Info("event projector worker stopped gracefully")
		return nil
	case <-ctx.Done():
		utils.Warn("event projector worker stop timed out")
		return ctx.Err()
	}
}

// processLoop runs the main processing loop for event projection
func (w *ProjectorWorker) processLoop() {
	defer func() {
		w.running = false
	}()

	// Only the first instance should process existing events on startup
	// Use a simple database lock to coordinate between instances
	if w.tryAcquireLock("projector_startup_lock") {
		utils.Info("acquired startup lock, processing all existing events")
		if err := w.projectorSvc.ProcessAllEvents(context.Background()); err != nil {
			utils.Error("failed to process existing events", slog.String("error", err.Error()))
		}
		w.releaseLock("projector_startup_lock")
	} else {
		utils.Info("another instance is processing startup events, skipping")
	}

	for {
		select {
		case <-w.ticker.C:
			w.processNewEventsWithLock()
		case <-w.stopChan:
			return
		}
	}
}

// processNewEvents processes new events since the last run
func (w *ProjectorWorker) processNewEvents() {
	ctx := context.Background()

	// Process events from the last 5 minutes to catch any missed events
	since := time.Now().Add(-5 * time.Minute)

	utils.Info("processing new events", slog.String("since", since.Format(time.RFC3339)))

	err := w.projectorSvc.ProcessEventsSince(ctx, since)
	if err != nil {
		utils.Error("failed to process new events", slog.String("error", err.Error()))
		return
	}

	utils.Info("completed processing new events")
}

// processNewEventsWithLock processes new events with locking to prevent race conditions
func (w *ProjectorWorker) processNewEventsWithLock() {
	lockKey := "projector_processing_lock"

	if !w.tryAcquireLock(lockKey) {
		utils.Info("another instance is processing events, skipping this cycle")
		return
	}

	defer w.releaseLock(lockKey)

	ctx := context.Background()

	// Process events from the last 5 minutes to catch any missed events
	since := time.Now().Add(-5 * time.Minute)

	utils.Info("processing new events with lock", slog.String("since", since.Format(time.RFC3339)))

	err := w.projectorSvc.ProcessEventsSince(ctx, since)
	if err != nil {
		utils.Error("failed to process new events", slog.String("error", err.Error()))
		return
	}

	utils.Info("completed processing new events with lock")
}

// tryAcquireLock attempts to acquire a database lock
func (w *ProjectorWorker) tryAcquireLock(lockKey string) bool {
	// For now, implement a simple instance-based locking
	// In production, you'd use Redis, database locks, or leader election

	// Simple approach: only allow instance 1 to process events
	// This is a temporary solution for the MVP
	return true // Allow all for now, will implement proper locking later

	utils.Info("lock acquisition attempted", slog.String("lock_key", lockKey))
	return true
}

// releaseLock releases a database lock
func (w *ProjectorWorker) releaseLock(lockKey string) {
	utils.Info("lock released", slog.String("lock_key", lockKey))
	// TODO: Implement proper lock release
}
