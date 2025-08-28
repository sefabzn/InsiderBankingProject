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

// balancesRepo implements the BalancesRepo interface.
type balancesRepo struct {
	db *pgxpool.Pool
}

// NewBalancesRepo creates a new balances repository.
func NewBalancesRepo(db *pgxpool.Pool) BalancesRepo {
	return &balancesRepo{db: db}
}

// GetByUserID retrieves a balance by user ID.
func (r *balancesRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Balance, error) {
	query := `
		SELECT user_id, amount, currency, last_updated_at
		FROM balances
		WHERE user_id = $1`

	var balance domain.Balance
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&balance.UserID,
		&balance.Amount,
		&balance.Currency,
		&balance.LastUpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("balance not found for user")
		}
		return nil, fmt.Errorf("failed to get balance by user ID: %w", err)
	}

	return &balance, nil
}

// Upsert creates or updates a balance.
func (r *balancesRepo) Upsert(ctx context.Context, balance *domain.Balance) error {
	query := `
		INSERT INTO balances (user_id, amount, currency, last_updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id)
		DO UPDATE SET
			amount = EXCLUDED.amount,
			currency = EXCLUDED.currency,
			last_updated_at = EXCLUDED.last_updated_at`

	balance.LastUpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, query, balance.UserID, balance.Amount, balance.Currency, balance.LastUpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert balance: %w", err)
	}

	return nil
}

// AddAmountTx adds amount to a user's balance within a transaction.
// This method should be used within database transactions for atomicity.
func (r *balancesRepo) AddAmountTx(ctx context.Context, tx interface{}, userID uuid.UUID, delta float64) error {
	// Type assert the transaction
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type")
	}

	// Use SELECT FOR UPDATE to prevent concurrent modifications
	query := `
		UPDATE balances 
		SET amount = amount + $2, last_updated_at = $3
		WHERE user_id = $1
		RETURNING amount`

	now := time.Now()
	var newAmount float64
	err := pgxTx.QueryRow(ctx, query, userID, delta, now).Scan(&newAmount)

	if err != nil {
		if err == pgx.ErrNoRows {
			// If balance doesn't exist, create it with the delta amount
			insertQuery := `
				INSERT INTO balances (user_id, amount, currency, last_updated_at)
				VALUES ($1, $2, $3, $4)`

			_, insertErr := pgxTx.Exec(ctx, insertQuery, userID, delta, "USD", now)
			if insertErr != nil {
				return fmt.Errorf("failed to create balance: %w", insertErr)
			}
			return nil
		}
		return fmt.Errorf("failed to add amount to balance: %w", err)
	}

	// Check for negative balance (business rule)
	if newAmount < 0 {
		return fmt.Errorf("insufficient funds: balance would be negative (%.2f)", newAmount)
	}

	return nil
}

// GetHistorical retrieves historical balance snapshots.
// Note: This is a simplified implementation. In a real system, you might have a separate table for balance history.
func (r *balancesRepo) GetHistorical(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.BalanceHistoryItem, error) {
	// For MVP, we'll simulate historical data by looking at transaction history
	// In a production system, you'd maintain a separate balance_history table
	query := `
		WITH user_transactions AS (
			-- Get standalone credits (not part of transfers)
			SELECT t.created_at, t.amount, 'credit' as type, 'credit' as reason
			FROM transactions t
			WHERE t.to_user_id = $1
				AND t.from_user_id IS NULL
				AND t.type = 'credit'
				AND t.status = 'success'

			UNION ALL

			-- Get standalone debits (not part of transfers) - exclude debits that occur within 1 second of a transfer
			SELECT t.created_at, -t.amount, 'debit' as type, 'debit' as reason
			FROM transactions t
			WHERE t.from_user_id = $1
				AND t.to_user_id IS NULL
				AND t.type = 'debit'
				AND t.status = 'success'
				AND NOT EXISTS (
					SELECT 1 FROM transactions t2
					WHERE t2.type = 'transfer'
						AND (t2.from_user_id = $1 OR t2.to_user_id = $1)
						AND t2.status = 'success'
						AND ABS(EXTRACT(EPOCH FROM (t.created_at - t2.created_at))) < 1
				)

			UNION ALL

			-- Get transfers (these are the main records)
			SELECT t.created_at,
				   CASE WHEN t.to_user_id = $1 THEN t.amount ELSE -t.amount END,
				   'transfer' as type,
				   CASE WHEN t.to_user_id = $1 THEN 'transfer_received' ELSE 'transfer_sent' END as reason
			FROM transactions t
			WHERE (t.from_user_id = $1 OR t.to_user_id = $1)
				AND t.type = 'transfer'
				AND t.status = 'success'
		)
		SELECT
			created_at as timestamp,
			SUM(amount) OVER (ORDER BY created_at) as running_balance,
			reason
		FROM user_transactions
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical balances: %w", err)
	}
	defer rows.Close()

	var history []*domain.BalanceHistoryItem
	for rows.Next() {
		var item domain.BalanceHistoryItem
		err := rows.Scan(
			&item.Timestamp,
			&item.Amount,
			&item.Reason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance history item: %w", err)
		}
		item.UserID = userID
		history = append(history, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate balance history: %w", err)
	}

	return history, nil
}

// GetAtTime retrieves balance at a specific time.
// This is a simplified implementation for MVP.
func (r *balancesRepo) GetAtTime(ctx context.Context, userID uuid.UUID, timestamp string) (*domain.Balance, error) {
	// Parse timestamp
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format: %w", err)
	}
	//add 1 second to ensure the filter balance at specified time
	t = t.Add(1 * time.Second)
	// Calculate balance at specific time by summing transactions up to that point
	query := `
		SELECT
			$1::uuid as user_id,
			COALESCE(SUM(
				CASE
					WHEN t.type = 'credit' AND t.to_user_id = $1 THEN t.amount
					WHEN t.type = 'debit' AND t.from_user_id = $1 THEN -t.amount
					WHEN t.type = 'transfer' AND t.to_user_id = $1 THEN t.amount
					WHEN t.type = 'transfer' AND t.from_user_id = $1 THEN -t.amount
					ELSE 0
				END
			), 0) as amount,
			'USD'::text as currency,
			$2::timestamptz as last_updated_at
		FROM transactions t
		WHERE (t.from_user_id = $1 OR t.to_user_id = $1)
			AND t.status = 'success'
			AND t.created_at <= $2::timestamptz`

	var balance domain.Balance
	err = r.db.QueryRow(ctx, query, userID, t).Scan(
		&balance.UserID,
		&balance.Amount,
		&balance.Currency,
		&balance.LastUpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get balance at time: %w", err)
	}

	return &balance, nil
}
