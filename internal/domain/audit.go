package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	EntityType string          `json:"entity_type" db:"entity_type"`
	EntityID   uuid.UUID       `json:"entity_id" db:"entity_id"`
	Action     string          `json:"action" db:"action"`
	Details    json.RawMessage `json:"details,omitempty" db:"details"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}

// EntityType defines valid entity types for audit logs.
type EntityType string

const (
	// EntityUser represents user entity type for audit logs
	EntityUser EntityType = "user"
	// EntityTransaction represents transaction entity type for audit logs
	EntityTransaction EntityType = "transaction"
	// EntityBalance represents balance entity type for audit logs
	EntityBalance EntityType = "balance"
)

// AuditAction defines common audit actions.
type AuditAction string

const (
	// ActionCreated represents create action for audit logs
	ActionCreated AuditAction = "created"
	// ActionUpdated represents update action for audit logs
	ActionUpdated AuditAction = "updated"
	// ActionDeleted represents delete action for audit logs
	ActionDeleted AuditAction = "deleted"
	// ActionCompleted represents completed action for audit logs
	ActionCompleted AuditAction = "completed"
	// ActionFailed represents failed action for audit logs
	ActionFailed AuditAction = "failed"
	// ActionRolledBack represents rolled back action for audit logs
	ActionRolledBack AuditAction = "rolled_back"
)

// CreateAuditLogRequest represents the data needed to create an audit log.
type CreateAuditLogRequest struct {
	EntityType string      `json:"entity_type"`
	EntityID   uuid.UUID   `json:"entity_id"`
	Action     string      `json:"action"`
	Details    interface{} `json:"details,omitempty"`
}

// AuditLogResponse represents an audit log in API responses.
type AuditLogResponse struct {
	ID         uuid.UUID       `json:"id"`
	EntityType string          `json:"entity_type"`
	EntityID   uuid.UUID       `json:"entity_id"`
	Action     string          `json:"action"`
	Details    json.RawMessage `json:"details,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// ToResponse converts an AuditLog to AuditLogResponse.
func (a *AuditLog) ToResponse() AuditLogResponse {
	return AuditLogResponse{
		ID:         a.ID,
		EntityType: a.EntityType,
		EntityID:   a.EntityID,
		Action:     a.Action,
		Details:    a.Details,
		CreatedAt:  a.CreatedAt,
	}
}

// AuditLogFilter represents filters for audit log queries.
type AuditLogFilter struct {
	EntityType *EntityType `json:"entity_type,omitempty"`
	EntityID   *uuid.UUID  `json:"entity_id,omitempty"`
	Action     *string     `json:"action,omitempty"`
	Since      *time.Time  `json:"since,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
}
