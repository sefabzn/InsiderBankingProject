package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sefa-b/go-banking-sim/internal/domain"
)

// auditRepo implements the AuditRepo interface.
type auditRepo struct {
	db *pgxpool.Pool
}

// NewAuditRepo creates a new audit repository.
func NewAuditRepo(db *pgxpool.Pool) AuditRepo {
	return &auditRepo{db: db}
}

// Log creates a new audit log entry.
func (r *auditRepo) Log(ctx context.Context, entityType string, entityID uuid.UUID, action string, details interface{}) error {
	query := `
		INSERT INTO audit_logs (id, entity_type, entity_id, action, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	id := uuid.New()
	createdAt := time.Now()

	// Convert details to JSONB
	var detailsJSON []byte
	var err error

	if details != nil {
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return fmt.Errorf("failed to marshal audit details: %w", err)
		}
	}

	_, err = r.db.Exec(ctx, query, id, entityType, entityID, action, detailsJSON, createdAt)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetByID retrieves an audit log by ID.
func (r *auditRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	query := `
		SELECT id, entity_type, entity_id, action, details, created_at
		FROM audit_logs
		WHERE id = $1`

	var auditLog domain.AuditLog
	err := r.db.QueryRow(ctx, query, id).Scan(
		&auditLog.ID,
		&auditLog.EntityType,
		&auditLog.EntityID,
		&auditLog.Action,
		&auditLog.Details,
		&auditLog.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("audit log not found")
		}
		return nil, fmt.Errorf("failed to get audit log by ID: %w", err)
	}

	return &auditLog, nil
}

// List retrieves audit logs with filtering.
func (r *auditRepo) List(ctx context.Context, filter *domain.AuditLogFilter) ([]*domain.AuditLog, error) {
	baseQuery := `
		SELECT id, entity_type, entity_id, action, details, created_at
		FROM audit_logs
		WHERE 1=1`

	args := []interface{}{}
	conditions := []string{}
	argIndex := 1

	// Apply filters
	if filter != nil {
		if filter.EntityType != nil {
			conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argIndex))
			args = append(args, string(*filter.EntityType))
			argIndex++
		}

		if filter.EntityID != nil {
			conditions = append(conditions, fmt.Sprintf("entity_id = $%d", argIndex))
			args = append(args, *filter.EntityID)
			argIndex++
		}

		if filter.Action != nil {
			conditions = append(conditions, fmt.Sprintf("action = $%d", argIndex))
			args = append(args, *filter.Action)
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

	return r.executeAuditQuery(ctx, query, args...)
}

// ListForEntity retrieves audit logs for a specific entity.
func (r *auditRepo) ListForEntity(ctx context.Context, entityType string, entityID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error) {
	query := `
		SELECT id, entity_type, entity_id, action, details, created_at
		FROM audit_logs
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	return r.executeAuditQuery(ctx, query, entityType, entityID, limit, offset)
}

// Count returns the total number of audit logs matching the filter.
func (r *auditRepo) Count(ctx context.Context, filter *domain.AuditLogFilter) (int, error) {
	baseQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`

	args := []interface{}{}
	conditions := []string{}
	argIndex := 1

	// Apply filters (same logic as List but for counting)
	if filter != nil {
		if filter.EntityType != nil {
			conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argIndex))
			args = append(args, string(*filter.EntityType))
			argIndex++
		}

		if filter.EntityID != nil {
			conditions = append(conditions, fmt.Sprintf("entity_id = $%d", argIndex))
			args = append(args, *filter.EntityID)
			argIndex++
		}

		if filter.Action != nil {
			conditions = append(conditions, fmt.Sprintf("action = $%d", argIndex))
			args = append(args, *filter.Action)
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
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	return count, nil
}

// executeAuditQuery executes an audit query and returns results.
func (r *auditRepo) executeAuditQuery(ctx context.Context, query string, args ...interface{}) ([]*domain.AuditLog, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute audit query: %w", err)
	}
	defer rows.Close()

	var auditLogs []*domain.AuditLog
	for rows.Next() {
		var auditLog domain.AuditLog
		err := rows.Scan(
			&auditLog.ID,
			&auditLog.EntityType,
			&auditLog.EntityID,
			&auditLog.Action,
			&auditLog.Details,
			&auditLog.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		auditLogs = append(auditLogs, &auditLog)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate audit logs: %w", err)
	}

	return auditLogs, nil
}
