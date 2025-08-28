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

// transactionsRepo implements the TransactionsRepo interface.
type transactionsRepo struct {
	db *pgxpool.Pool
}

// NewTransactionsRepo creates a new transactions repository.
func NewTransactionsRepo(db *pgxpool.Pool) TransactionsRepo {
	return &transactionsRepo{db: db}
}

// CreatePending creates a new transaction with pending status.
func (r *transactionsRepo) CreatePending(ctx context.Context, tx *domain.Transaction) error {
	query := `
		INSERT INTO transactions (id, from_user_id, to_user_id, amount, type, status, created_at, currency)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	tx.Status = string(domain.StatusPending)
	tx.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, query, tx.ID, tx.FromUserID, tx.ToUserID, tx.Amount, tx.Type, tx.Status, tx.CreatedAt, tx.Currency)
	if err != nil {
		return fmt.Errorf("failed to create pending transaction: %w", err)
	}

	return nil
}

// MarkCompleted marks a transaction as completed.
func (r *transactionsRepo) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	return r.updateTransactionStatus(ctx, id, string(domain.StatusPending), string(domain.StatusSuccess))
}

// MarkFailed marks a transaction as failed.
func (r *transactionsRepo) MarkFailed(ctx context.Context, id uuid.UUID) error {
	return r.updateTransactionStatus(ctx, id, string(domain.StatusPending), string(domain.StatusFailed))
}

// updateTransactionStatus updates transaction status with validation.
func (r *transactionsRepo) updateTransactionStatus(ctx context.Context, id uuid.UUID, expectedCurrentStatus, newStatus string) error {
	query := `
		UPDATE transactions
		SET status = $2
		WHERE id = $1 AND status = $3`

	result, err := r.db.Exec(ctx, query, id, newStatus, expectedCurrentStatus)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		// Check if transaction exists with different status
		var currentStatus string
		checkQuery := `SELECT status FROM transactions WHERE id = $1`
		checkErr := r.db.QueryRow(ctx, checkQuery, id).Scan(&currentStatus)

		if checkErr == pgx.ErrNoRows {
			return fmt.Errorf("transaction not found")
		} else if checkErr != nil {
			return fmt.Errorf("failed to check transaction status: %w", checkErr)
		}

		return fmt.Errorf("invalid state transition: cannot change from %s to %s", currentStatus, newStatus)
	}

	return nil
}

// GetByID retrieves a transaction by ID.
func (r *transactionsRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, type, status, created_at, currency
		FROM transactions
		WHERE id = $1`

	var tx domain.Transaction
	err := r.db.QueryRow(ctx, query, id).Scan(
		&tx.ID,
		&tx.FromUserID,
		&tx.ToUserID,
		&tx.Amount,
		&tx.Type,
		&tx.Status,
		&tx.CreatedAt,
		&tx.Currency,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction by ID: %w", err)
	}

	return &tx, nil
}

// ListForUser retrieves transactions for a specific user.
func (r *transactionsRepo) ListForUser(ctx context.Context, userID uuid.UUID, filter *domain.TransactionFilter) ([]*domain.Transaction, error) {
	baseQuery := `
		SELECT id, from_user_id, to_user_id, amount, type, status, created_at, currency
		FROM transactions
		WHERE (from_user_id = $1 OR to_user_id = $1)`

	args := []interface{}{userID}
	conditions := []string{}
	argIndex := 2

	// Apply filters
	if filter != nil {
		if filter.Type != nil {
			conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
			args = append(args, string(*filter.Type))
			argIndex++
		}

		if filter.Status != nil {
			conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
			args = append(args, string(*filter.Status))
			argIndex++
		}

		if filter.Since != nil {
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
			args = append(args, *filter.Since)
			argIndex++
		}
	}

	// Build final query
	query := baseQuery
	if len(conditions) > 0 {
		query += " AND " + conditions[0]
		for _, condition := range conditions[1:] {
			query += " AND " + condition
		}
	}

	query += " ORDER BY created_at DESC"

	// Apply pagination
	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, filter.Limit)
			argIndex++
		}

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	return r.executeTransactionQuery(ctx, query, args...)
}

// List retrieves transactions with filtering.
func (r *transactionsRepo) List(ctx context.Context, filter *domain.TransactionFilter) ([]*domain.Transaction, error) {
	baseQuery := `
		SELECT id, from_user_id, to_user_id, amount, type, status, created_at, currency
		FROM transactions
		WHERE 1=1`

	args := []interface{}{}
	conditions := []string{}
	argIndex := 1

	// Apply filters
	if filter != nil {
		if filter.UserID != nil {
			conditions = append(conditions, fmt.Sprintf("(from_user_id = $%d OR to_user_id = $%d)", argIndex, argIndex))
			args = append(args, *filter.UserID)
			argIndex++
		}

		if filter.Type != nil {
			conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
			args = append(args, string(*filter.Type))
			argIndex++
		}

		if filter.Status != nil {
			conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
			args = append(args, string(*filter.Status))
			argIndex++
		}

		if filter.Since != nil {
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
			args = append(args, *filter.Since)
			argIndex++
		}
	}

	// Build final query
	query := baseQuery
	for _, condition := range conditions {
		query += " AND " + condition
	}

	query += " ORDER BY created_at DESC"

	// Apply pagination
	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, filter.Limit)
			argIndex++
		}

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	return r.executeTransactionQuery(ctx, query, args...)
}

// Count returns the total number of transactions matching the filter.
func (r *transactionsRepo) Count(ctx context.Context, filter *domain.TransactionFilter) (int, error) {
	baseQuery := `SELECT COUNT(*) FROM transactions WHERE 1=1`

	args := []interface{}{}
	conditions := []string{}
	argIndex := 1

	// Apply filters (same logic as List but for counting)
	if filter != nil {
		if filter.UserID != nil {
			conditions = append(conditions, fmt.Sprintf("(from_user_id = $%d OR to_user_id = $%d)", argIndex, argIndex))
			args = append(args, *filter.UserID)
			argIndex++
		}

		if filter.Type != nil {
			conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
			args = append(args, string(*filter.Type))
			argIndex++
		}

		if filter.Status != nil {
			conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
			args = append(args, string(*filter.Status))
			argIndex++
		}

		if filter.Since != nil {
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
			args = append(args, *filter.Since)
			argIndex++
		}
	}

	// Build final query
	query := baseQuery
	for _, condition := range conditions {
		query += " AND " + condition
	}

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}

// executeTransactionQuery executes a transaction query and returns results.
func (r *transactionsRepo) executeTransactionQuery(ctx context.Context, query string, args ...interface{}) ([]*domain.Transaction, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute transaction query: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.FromUserID,
			&tx.ToUserID,
			&tx.Amount,
			&tx.Type,
			&tx.Status,
			&tx.CreatedAt,
			&tx.Currency,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, &tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate transactions: %w", err)
	}

	return transactions, nil
}
