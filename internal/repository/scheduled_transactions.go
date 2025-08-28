package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sefa-b/go-banking-sim/internal/domain"
)

// ScheduledTransactionRepository handles scheduled transaction operations
type ScheduledTransactionRepository struct {
	pool *pgxpool.Pool
}

// NewScheduledTransactionRepository creates a new scheduled transaction repository
func NewScheduledTransactionRepository(pool *pgxpool.Pool) *ScheduledTransactionRepository {
	return &ScheduledTransactionRepository{pool: pool}
}

// Create creates a new scheduled transaction
func (r *ScheduledTransactionRepository) Create(ctx context.Context, st *domain.ScheduledTransaction) error {
	query := `
		INSERT INTO scheduled_transactions (
			id, user_id, transaction_type, amount, currency, description, to_user_id,
			schedule_type, execute_at, recurrence_pattern, recurrence_end_date,
			max_occurrences, current_occurrence, status, is_active, created_at, updated_at,
			next_execution_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
	`

	nextExecution := st.CalculateNextExecution()

	_, err := r.pool.Exec(ctx, query,
		st.ID,
		st.UserID,
		st.TransactionType,
		st.Amount,
		st.Currency,
		st.Description,
		st.ToUserID,
		st.ScheduleType,
		st.ExecuteAt,
		st.RecurrencePattern,
		st.RecurrenceEndDate,
		st.MaxOccurrences,
		st.CurrentOccurrence,
		st.Status,
		st.IsActive,
		st.CreatedAt,
		st.UpdatedAt,
		nextExecution,
	)

	if err != nil {
		return fmt.Errorf("failed to create scheduled transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a scheduled transaction by ID
func (r *ScheduledTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ScheduledTransaction, error) {
	query := `
		SELECT id, user_id, transaction_type, amount, currency, description, to_user_id,
			   schedule_type, execute_at, recurrence_pattern, recurrence_end_date,
			   max_occurrences, current_occurrence, status, is_active, created_at,
			   updated_at, last_executed_at, next_execution_at
		FROM scheduled_transactions
		WHERE id = $1
	`

	var st domain.ScheduledTransaction
	var description, status string
	var recurrencePattern *string
	var toUserID *uuid.UUID
	var recurrenceEndDate, lastExecutedAt, nextExecutionAt *time.Time
	var maxOccurrences *int
	var isActive bool
	var createdAt, updatedAt, executeAt time.Time

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&st.ID,
		&st.UserID,
		&st.TransactionType,
		&st.Amount,
		&st.Currency,
		&description,
		&toUserID,
		&st.ScheduleType,
		&executeAt,
		&recurrencePattern,
		&recurrenceEndDate,
		&maxOccurrences,
		&st.CurrentOccurrence,
		&status,
		&isActive,
		&createdAt,
		&updatedAt,
		&lastExecutedAt,
		&nextExecutionAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("scheduled transaction not found")
		}
		return nil, fmt.Errorf("failed to get scheduled transaction: %w", err)
	}

	st.Description = description
	st.ToUserID = toUserID
	st.RecurrencePattern = recurrencePattern
	st.RecurrenceEndDate = recurrenceEndDate
	st.MaxOccurrences = maxOccurrences
	st.Status = status
	st.IsActive = isActive
	st.CreatedAt = createdAt
	st.UpdatedAt = updatedAt
	st.LastExecutedAt = lastExecutedAt
	st.NextExecutionAt = nextExecutionAt
	st.ExecuteAt = executeAt

	return &st, nil
}

// GetByUserID retrieves scheduled transactions for a user
func (r *ScheduledTransactionRepository) GetByUserID(ctx context.Context, userID uuid.UUID, filter *domain.ScheduledTransactionFilter) ([]*domain.ScheduledTransaction, error) {
	query := `
		SELECT id, user_id, transaction_type, amount, currency, description, to_user_id,
			   schedule_type, execute_at, recurrence_pattern, recurrence_end_date,
			   max_occurrences, current_occurrence, status, is_active, created_at,
			   updated_at, last_executed_at, next_execution_at
		FROM scheduled_transactions
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argCount := 1

	if filter != nil {
		if filter.Status != nil {
			argCount++
			query += fmt.Sprintf(" AND status = $%d", argCount)
			args = append(args, *filter.Status)
		}

		if filter.Type != nil {
			argCount++
			query += fmt.Sprintf(" AND transaction_type = $%d", argCount)
			args = append(args, *filter.Type)
		}

		if filter.IsActive != nil {
			argCount++
			query += fmt.Sprintf(" AND is_active = $%d", argCount)
			args = append(args, *filter.IsActive)
		}

		if filter.ExecuteFrom != nil {
			argCount++
			query += fmt.Sprintf(" AND execute_at >= $%d", argCount)
			args = append(args, *filter.ExecuteFrom)
		}

		if filter.ExecuteTo != nil {
			argCount++
			query += fmt.Sprintf(" AND execute_at <= $%d", argCount)
			args = append(args, *filter.ExecuteTo)
		}
	}

	query += " ORDER BY execute_at ASC"

	if filter != nil && filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	if filter != nil && filter.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.ScheduledTransaction
	for rows.Next() {
		var st domain.ScheduledTransaction
		var description, status string
		var recurrencePattern *string
		var toUserID *uuid.UUID
		var recurrenceEndDate, lastExecutedAt, nextExecutionAt *time.Time
		var maxOccurrences *int
		var isActive bool
		var createdAt, updatedAt, executeAt time.Time

		err := rows.Scan(
			&st.ID,
			&st.UserID,
			&st.TransactionType,
			&st.Amount,
			&st.Currency,
			&description,
			&toUserID,
			&st.ScheduleType,
			&executeAt,
			&recurrencePattern,
			&recurrenceEndDate,
			&maxOccurrences,
			&st.CurrentOccurrence,
			&status,
			&isActive,
			&createdAt,
			&updatedAt,
			&lastExecutedAt,
			&nextExecutionAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduled transaction: %w", err)
		}

		st.Description = description
		st.ToUserID = toUserID
		st.RecurrencePattern = recurrencePattern
		st.RecurrenceEndDate = recurrenceEndDate
		st.MaxOccurrences = maxOccurrences
		st.Status = status
		st.IsActive = isActive
		st.CreatedAt = createdAt
		st.UpdatedAt = updatedAt
		st.LastExecutedAt = lastExecutedAt
		st.NextExecutionAt = nextExecutionAt
		st.ExecuteAt = executeAt

		transactions = append(transactions, &st)
	}

	return transactions, nil
}

// GetDueForExecution retrieves scheduled transactions that are due for execution
func (r *ScheduledTransactionRepository) GetDueForExecution(ctx context.Context, limit int) ([]*domain.ScheduledTransaction, error) {
	// Get due transactions with time buffer to prevent immediate re-processing
	query := `
		SELECT id, user_id, transaction_type, amount, currency, description, to_user_id,
			   schedule_type, execute_at, recurrence_pattern, recurrence_end_date,
			   max_occurrences, current_occurrence, status, is_active, created_at,
			   updated_at, last_executed_at, next_execution_at
		FROM scheduled_transactions
		WHERE is_active = true
		  AND status = 'active'
		  AND execute_at <= NOW()
		  AND (schedule_type = 'recurring' OR last_executed_at IS NULL)
		  AND (updated_at IS NULL OR updated_at < NOW() - INTERVAL '1 seconds')
		ORDER BY execute_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	// Debug: Check different conditions
	totalQuery := `SELECT COUNT(*) FROM scheduled_transactions`
	var totalCount int
	r.pool.QueryRow(ctx, totalQuery).Scan(&totalCount)

	activeQuery := `SELECT COUNT(*) FROM scheduled_transactions WHERE status = 'active'`
	var activeCount int
	r.pool.QueryRow(ctx, activeQuery).Scan(&activeCount)

	dueQuery := `SELECT COUNT(*) FROM scheduled_transactions WHERE execute_at <= NOW()`
	var dueCount int
	r.pool.QueryRow(ctx, dueQuery).Scan(&dueCount)

	bufferQuery := `SELECT COUNT(*) FROM scheduled_transactions WHERE updated_at < NOW() - INTERVAL '1 seconds'`
	var bufferCount int
	r.pool.QueryRow(ctx, bufferQuery).Scan(&bufferCount)

	fmt.Printf("DEBUG: Total transactions: %d, Active: %d, Due: %d, After buffer: %d\n",
		totalCount, activeCount, dueCount, bufferCount)

	// Show recent transactions
	recentQuery := `
		SELECT id, status, is_active, execute_at, updated_at, last_executed_at
		FROM scheduled_transactions
		ORDER BY updated_at DESC LIMIT 3
	`
	rows, _ := r.pool.Query(ctx, recentQuery)
	for rows.Next() {
		var id string
		var status string
		var isActive bool
		var executeAt, updatedAt time.Time
		var lastExecutedAt *time.Time
		rows.Scan(&id, &status, &isActive, &executeAt, &updatedAt, &lastExecutedAt)
		fmt.Printf("Recent transaction: ID=%s, Status=%s, Active=%t, ExecuteAt=%v, UpdatedAt=%v\n",
			id[:8]+"...", status, isActive, executeAt, updatedAt)
	}
	rows.Close()

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get due transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.ScheduledTransaction
	for rows.Next() {
		var st domain.ScheduledTransaction
		var description, status string
		var recurrencePattern *string
		var toUserID *uuid.UUID
		var recurrenceEndDate, lastExecutedAt, nextExecutionAt *time.Time
		var maxOccurrences *int
		var isActive bool
		var createdAt, updatedAt, executeAt time.Time

		err := rows.Scan(
			&st.ID,
			&st.UserID,
			&st.TransactionType,
			&st.Amount,
			&st.Currency,
			&description,
			&toUserID,
			&st.ScheduleType,
			&executeAt,
			&recurrencePattern,
			&recurrenceEndDate,
			&maxOccurrences,
			&st.CurrentOccurrence,
			&status,
			&isActive,
			&createdAt,
			&updatedAt,
			&lastExecutedAt,
			&nextExecutionAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan due transaction: %w", err)
		}

		st.Description = description
		st.ToUserID = toUserID
		st.RecurrencePattern = recurrencePattern
		st.RecurrenceEndDate = recurrenceEndDate
		st.MaxOccurrences = maxOccurrences
		st.Status = status
		st.IsActive = isActive
		st.CreatedAt = createdAt
		st.UpdatedAt = updatedAt
		st.LastExecutedAt = lastExecutedAt
		st.NextExecutionAt = nextExecutionAt
		st.ExecuteAt = executeAt

		transactions = append(transactions, &st)
	}

	return transactions, nil
}

// Update updates a scheduled transaction
func (r *ScheduledTransactionRepository) Update(ctx context.Context, st *domain.ScheduledTransaction) error {
	query := `
		UPDATE scheduled_transactions
		SET description = $1, status = $2, is_active = $3, execute_at = $4,
			recurrence_end_date = $5, max_occurrences = $6, updated_at = $7,
			next_execution_at = $8, last_executed_at = $9, current_occurrence = $10
		WHERE id = $11
	`

	nextExecution := st.CalculateNextExecution()

	_, err := r.pool.Exec(ctx, query,
		st.Description,
		st.Status,
		st.IsActive,
		st.ExecuteAt,
		st.RecurrenceEndDate,
		st.MaxOccurrences,
		time.Now(),
		nextExecution,
		st.LastExecutedAt,
		st.CurrentOccurrence,
		st.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update scheduled transaction: %w", err)
	}

	return nil
}

// ResetStatus resets the status of a scheduled transaction (used for error recovery)
func (r *ScheduledTransactionRepository) ResetStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE scheduled_transactions
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to reset scheduled transaction status: %w", err)
	}

	return nil
}

// Delete deletes a scheduled transaction
func (r *ScheduledTransactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM scheduled_transactions WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete scheduled transaction: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("scheduled transaction not found")
	}

	return nil
}

// CreateExecution creates an execution record
func (r *ScheduledTransactionRepository) CreateExecution(ctx context.Context, execution *domain.ScheduledTransactionExecution) error {
	query := `
		INSERT INTO scheduled_transaction_executions (
			id, scheduled_transaction_id, executed_at, status, transaction_id,
			error_message, amount, currency
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		execution.ID,
		execution.ScheduledTransactionID,
		execution.ExecutedAt,
		execution.Status,
		execution.TransactionID,
		execution.ErrorMessage,
		execution.Amount,
		execution.Currency,
	)

	if err != nil {
		return fmt.Errorf("failed to create execution record: %w", err)
	}

	return nil
}

// GetExecutions retrieves execution history for a scheduled transaction
func (r *ScheduledTransactionRepository) GetExecutions(ctx context.Context, scheduledTransactionID uuid.UUID, limit int, offset int) ([]*domain.ScheduledTransactionExecution, error) {
	query := `
		SELECT id, scheduled_transaction_id, executed_at, status, transaction_id,
			   error_message, amount, currency
		FROM scheduled_transaction_executions
		WHERE scheduled_transaction_id = $1
		ORDER BY executed_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, scheduledTransactionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get executions: %w", err)
	}
	defer rows.Close()

	var executions []*domain.ScheduledTransactionExecution
	for rows.Next() {
		var execution domain.ScheduledTransactionExecution
		var transactionID *uuid.UUID
		var errorMessage string

		err := rows.Scan(
			&execution.ID,
			&execution.ScheduledTransactionID,
			&execution.ExecutedAt,
			&execution.Status,
			&transactionID,
			&errorMessage,
			&execution.Amount,
			&execution.Currency,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		execution.TransactionID = transactionID
		execution.ErrorMessage = errorMessage

		executions = append(executions, &execution)
	}

	return executions, nil
}

// Count counts scheduled transactions matching the filter
func (r *ScheduledTransactionRepository) Count(ctx context.Context, userID uuid.UUID, filter *domain.ScheduledTransactionFilter) (int, error) {
	query := `SELECT COUNT(*) FROM scheduled_transactions WHERE user_id = $1`
	args := []interface{}{userID}
	argCount := 1

	if filter != nil {
		if filter.Status != nil {
			argCount++
			query += fmt.Sprintf(" AND status = $%d", argCount)
			args = append(args, *filter.Status)
		}

		if filter.Type != nil {
			argCount++
			query += fmt.Sprintf(" AND transaction_type = $%d", argCount)
			args = append(args, *filter.Type)
		}

		if filter.IsActive != nil {
			argCount++
			query += fmt.Sprintf(" AND is_active = $%d", argCount)
			args = append(args, *filter.IsActive)
		}
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count scheduled transactions: %w", err)
	}

	return count, nil
}
