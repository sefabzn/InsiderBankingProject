// Package main is the entry point for the Go Banking Simulation server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sefa-b/go-banking-sim/internal/api/middleware"
	v1 "github.com/sefa-b/go-banking-sim/internal/api/v1"
	"github.com/sefa-b/go-banking-sim/internal/auth"
	"github.com/sefa-b/go-banking-sim/internal/config"
	"github.com/sefa-b/go-banking-sim/internal/domain"
	"github.com/sefa-b/go-banking-sim/internal/repository"
	"github.com/sefa-b/go-banking-sim/internal/service"
	"github.com/sefa-b/go-banking-sim/internal/utils"
	"github.com/sefa-b/go-banking-sim/internal/worker"
)

// transactionServiceAdapter adapts the service.TransactionService to worker.TransactionService interface
type transactionServiceAdapter struct {
	service service.TransactionService
}

func (a *transactionServiceAdapter) CreditSync(ctx context.Context, userID string, req interface{}) (interface{}, error) {
	// Parse userID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Type assert the request
	creditReq, ok := req.(*domain.CreditRequest)
	if !ok {
		return nil, fmt.Errorf("invalid credit request type")
	}

	// Call the sync version to avoid circular dependency
	return a.service.CreditSync(ctx, uid, creditReq)
}

func (a *transactionServiceAdapter) DebitSync(ctx context.Context, userID string, req interface{}) (interface{}, error) {
	// Parse userID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Type assert the request
	debitReq, ok := req.(*domain.DebitRequest)
	if !ok {
		return nil, fmt.Errorf("invalid debit request type")
	}

	return a.service.DebitSync(ctx, uid, debitReq)
}

func (a *transactionServiceAdapter) TransferSync(ctx context.Context, fromUserID string, req interface{}) (interface{}, error) {
	// Parse fromUserID
	fromUID, err := uuid.Parse(fromUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid from user ID: %w", err)
	}

	// Type assert the request
	transferReq, ok := req.(*domain.TransferRequest)
	if !ok {
		return nil, fmt.Errorf("invalid transfer request type")
	}

	return a.service.TransferSync(ctx, fromUID, transferReq)
}

func (a *transactionServiceAdapter) RollbackSync(ctx context.Context, transactionID string, requestingUserID string) (interface{}, error) {
	// Parse transactionID
	txID, err := uuid.Parse(transactionID)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction ID: %w", err)
	}

	// Parse requestingUserID
	reqUID, err := uuid.Parse(requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid requesting user ID: %w", err)
	}

	return a.service.RollbackSync(ctx, txID, reqUID)
}

func main() {
	cfg := config.Load()

	// Initialize structured logger
	utils.InitLogger(cfg.Environment, "go-banking-sim")

	// Initialize metrics collector
	metricsCollector := utils.NewMetricsCollector()

	// Initialize distributed tracing
	ctx := context.Background()
	shutdownTracer, err := utils.InitTracer(ctx, "go-banking-sim", "1.0.0", "jaeger:14250")
	if err != nil {
		utils.Error("failed to initialize tracer", "error", err.Error())
		os.Exit(1)
	}
	defer shutdownTracer()

	// Initialize database connection (if DB_URL is provided)
	var db *repository.DB
	if cfg.DBUrl != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		db, err = repository.Connect(ctx, cfg.DBUrl)
		if err != nil {
			utils.Error("failed to connect to database", slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer db.Close()
	} else {
		utils.Warn("no database URL provided, running without database")
	}

	// Initialize Redis connection
	var redisClient *repository.RedisClient
	redisConfig := repository.RedisConfig{
		Addr:     "redis:6379", // Default Redis address in Docker
		Password: "redis_password",
		DB:       0,
	}

	redisClient, err = repository.NewRedisClient(redisConfig)
	if err != nil {
		utils.Warn("failed to connect to Redis, running without cache", slog.String("error", err.Error()))
	} else {
		defer redisClient.Close()
	}

	// Initialize repositories (if database is available)
	var repos *repository.Repositories
	if db != nil {
		repos = &repository.Repositories{
			Users:                 repository.NewUsersRepo(db.Pool),
			Balances:              repository.NewBalancesRepo(db.Pool),
			Transactions:          repository.NewTransactionsRepo(db.Pool),
			Audit:                 repository.NewAuditRepo(db.Pool),
			Events:                repository.NewEventRepository(db.Pool),
			ScheduledTransactions: repository.NewScheduledTransactionRepository(db.Pool),
		}
	}

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, "go-banking-sim")

	// Initialize services first
	var services *service.Services
	if repos != nil {
		// Create event service first as it's needed by other services
		eventSvc := service.NewEventService(repos.Events)

		// Create balance service first since transaction service depends on it
		balanceSvc := service.NewBalanceService(repos)
		transactionSvc := service.NewTransactionService(repos, balanceSvc, nil, eventSvc, db.Pool) // Worker pool will be set later

		services = &service.Services{
			Auth:                 service.NewAuthService(repos, jwtManager, eventSvc),
			User:                 service.NewUserService(repos),
			Balance:              balanceSvc,
			Transaction:          transactionSvc,
			ScheduledTransaction: service.NewScheduledTransactionService(repos, transactionSvc),
			Event:                eventSvc,
			Projector:            service.NewProjectorService(repos.Events, repos.Users, repos.Balances, repos.Transactions),
		}

		// Initialize cache service if Redis is available
		if redisClient != nil {
			cacheService := service.NewCacheService(redisClient)
			services.Cache = cacheService

			// Inject cache service into existing services
			if userSvc, ok := services.User.(*service.UserServiceImpl); ok {
				userSvc.SetCacheService(cacheService)
			}
			if balanceSvc, ok := services.Balance.(*service.BalanceServiceImpl); ok {
				balanceSvc.SetCacheService(cacheService)
			}
			if transactionSvc, ok := services.Transaction.(*service.TransactionServiceImpl); ok {
				transactionSvc.SetCacheService(cacheService)
			}
		}
	}

	// Initialize worker pool for async transaction processing
	var workerPool *worker.WorkerPool
	var jobQueue *worker.JobQueue
	if repos != nil && services != nil {
		jobQueue = worker.NewJobQueue(100) // Buffer size of 100 jobs

		// Create an adapter that implements the worker's TransactionService interface
		adapter := &transactionServiceAdapter{service: services.Transaction}
		workerPool = worker.NewWorkerPool(jobQueue, adapter)

		// Set the worker pool on the transaction service to enable job submission
		services.Transaction.SetWorkerPool(workerPool)

		// Set metrics collector on transaction service for tracking transaction counts
		services.Transaction.SetMetricsCollector(metricsCollector)

		// Set initial queue depth (will be updated by worker pool if available)
		metricsCollector.SetQueueDepth(0)
	}

	// Initialize scheduled transaction worker
	var scheduledWorker *worker.ScheduledWorker
	if services != nil && services.ScheduledTransaction != nil {
		scheduledWorker = worker.NewScheduledWorker(services.ScheduledTransaction)
	}

	// Initialize event projector worker
	var projectorWorker *worker.ProjectorWorker
	if services != nil && services.Projector != nil {
		projectorWorker = worker.NewProjectorWorker(services.Projector)
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Add health endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Add Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Add basic metrics endpoint (JSON format)
	mux.HandleFunc("/api/v1/metrics/basic", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		metrics := metricsCollector.GetMetrics()
		json.NewEncoder(w).Encode(metrics)
	})

	// Add circuit breaker metrics endpoint
	mux.HandleFunc("/api/v1/metrics/circuit-breakers", middleware.CircuitBreakerMetricsHandler)

	// Register v1 API routes
	if repos != nil && services != nil {
		apiRouter := v1.NewRouter(repos, services, jwtManager)
		apiRouter.RegisterRoutes(mux)
	} else {
		utils.Warn("skipping API routes registration due to missing database")
	}

	// Basic server setup with OpenTelemetry tracing, metrics and logging middleware
	server := &http.Server{
		Addr: cfg.GetAddr(),
		Handler: middleware.LoggingMiddleware(
			middleware.TracingMiddleware("go-banking-sim")(
				middleware.MetricsMiddleware(metricsCollector)(mux),
			),
		),
	}

	// Channel to listen for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start worker pool if available
	if workerPool != nil {
		workerPool.Start(5) // Start with 5 workers
	}

	// Start scheduled worker if available
	if scheduledWorker != nil {
		scheduledWorker.Start(30 * time.Second) // Check every 10 seconds for testing
	}

	// Start projector worker if available
	if projectorWorker != nil {
		projectorWorker.Start(60 * time.Second) // Process events every 60 seconds
	}

	// Start server in goroutine
	go func() {
		utils.Info("server starting",
			slog.String("addr", cfg.GetAddr()),
			slog.String("env", cfg.Environment),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Error("server failed to start", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-quit
	utils.Info("shutting down server")

	// Stop worker pool gracefully
	if workerPool != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := workerPool.Stop(shutdownCtx); err != nil {
			utils.Error("worker pool shutdown error", slog.String("error", err.Error()))
		}
		shutdownCancel()
	}

	// Stop scheduled worker gracefully
	if scheduledWorker != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := scheduledWorker.Stop(shutdownCtx); err != nil {
			utils.Error("scheduled worker shutdown error", slog.String("error", err.Error()))
		}
		shutdownCancel()
	}

	// Stop projector worker gracefully
	if projectorWorker != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := projectorWorker.Stop(shutdownCtx); err != nil {
			utils.Error("projector worker shutdown error", slog.String("error", err.Error()))
		}
		shutdownCancel()
	}

	// Create context with 5 second timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		utils.Error("server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	utils.Info("server stopped gracefully")
}
