package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Event represents an event in the event sourcing system
type Event struct {
	ID            uuid.UUID `json:"id" db:"id"`
	AggregateType string    `json:"aggregate_type" db:"aggregate_type"`
	AggregateID   uuid.UUID `json:"aggregate_id" db:"aggregate_id"`
	EventType     string    `json:"event_type" db:"event_type"`
	EventData     []byte    `json:"event_data" db:"event_data"`
	EventMetadata []byte    `json:"event_metadata,omitempty" db:"event_metadata"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	Version       int       `json:"version" db:"version"`
}

// EventEnvelope wraps an event with metadata for serialization
type EventEnvelope struct {
	Event     interface{} `json:"event"`
	Metadata  interface{} `json:"metadata,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// AggregateType defines valid aggregate types
type AggregateType string

const (
	AggregateUser        AggregateType = "user"
	AggregateBalance     AggregateType = "balance"
	AggregateTransaction AggregateType = "transaction"
)

// Event Types
type EventType string

const (
	// User Events
	EventUserRegistered EventType = "UserRegistered"
	EventUserUpdated    EventType = "UserUpdated"
	EventUserDeleted    EventType = "UserDeleted"

	// Balance Events
	EventBalanceInitialized EventType = "BalanceInitialized"
	EventAmountCredited     EventType = "AmountCredited"
	EventAmountDebited      EventType = "AmountDebited"

	// Transaction Events
	EventTransactionStarted    EventType = "TransactionStarted"
	EventTransactionCompleted  EventType = "TransactionCompleted"
	EventTransactionFailed     EventType = "TransactionFailed"
	EventTransactionRolledBack EventType = "TransactionRolledBack"
	EventTransferExecuted      EventType = "TransferExecuted"
)

// UserRegisteredEvent represents a user registration event
type UserRegisteredEvent struct {
	UserID       uuid.UUID `json:"user_id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"`
}

// UserUpdatedEvent represents a user update event
type UserUpdatedEvent struct {
	UserID  uuid.UUID              `json:"user_id"`
	OldData map[string]interface{} `json:"old_data"`
	NewData map[string]interface{} `json:"new_data"`
}

// BalanceInitializedEvent represents balance initialization
type BalanceInitializedEvent struct {
	UserID   uuid.UUID `json:"user_id"`
	Amount   float64   `json:"amount"`
	Currency string    `json:"currency"`
}

// AmountCreditedEvent represents a credit transaction
type AmountCreditedEvent struct {
	UserID        uuid.UUID `json:"user_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	TransactionID uuid.UUID `json:"transaction_id"`
	Reason        string    `json:"reason"`
}

// AmountDebitedEvent represents a debit transaction
type AmountDebitedEvent struct {
	UserID        uuid.UUID `json:"user_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	TransactionID uuid.UUID `json:"transaction_id"`
	Reason        string    `json:"reason"`
}

// TransferExecutedEvent represents a transfer transaction
type TransferExecutedEvent struct {
	FromUserID    uuid.UUID `json:"from_user_id"`
	ToUserID      uuid.UUID `json:"to_user_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	TransactionID uuid.UUID `json:"transaction_id"`
}

// TransactionStartedEvent represents transaction initiation
type TransactionStartedEvent struct {
	TransactionID uuid.UUID  `json:"transaction_id"`
	UserID        uuid.UUID  `json:"user_id,omitempty"`
	FromUserID    *uuid.UUID `json:"from_user_id,omitempty"`
	ToUserID      *uuid.UUID `json:"to_user_id,omitempty"`
	Amount        float64    `json:"amount"`
	Type          string     `json:"type"`
}

// TransactionCompletedEvent represents transaction completion
type TransactionCompletedEvent struct {
	TransactionID uuid.UUID  `json:"transaction_id"`
	UserID        uuid.UUID  `json:"user_id,omitempty"`
	FromUserID    *uuid.UUID `json:"from_user_id,omitempty"`
	ToUserID      *uuid.UUID `json:"to_user_id,omitempty"`
	Amount        float64    `json:"amount"`
	Type          string     `json:"type"`
}

// TransactionFailedEvent represents transaction failure
type TransactionFailedEvent struct {
	TransactionID uuid.UUID  `json:"transaction_id"`
	UserID        uuid.UUID  `json:"user_id,omitempty"`
	FromUserID    *uuid.UUID `json:"from_user_id,omitempty"`
	ToUserID      *uuid.UUID `json:"to_user_id,omitempty"`
	Amount        float64    `json:"amount"`
	Type          string     `json:"type"`
	Error         string     `json:"error"`
}

// EventMetadata represents optional event metadata
type EventMetadata struct {
	CorrelationID string                 `json:"correlation_id,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	IP            string                 `json:"ip,omitempty"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
}

// NewEvent creates a new event
func NewEvent(aggregateType AggregateType, aggregateID uuid.UUID, eventType EventType, eventData interface{}, metadata *EventMetadata) (*Event, error) {
	eventDataBytes, err := json.Marshal(eventData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	var metadataBytes []byte
	if metadata != nil {
		metadataBytes, err = json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal event metadata: %w", err)
		}
	}

	return &Event{
		ID:            uuid.New(),
		AggregateType: string(aggregateType),
		AggregateID:   aggregateID,
		EventType:     string(eventType),
		EventData:     eventDataBytes,
		EventMetadata: metadataBytes,
		CreatedAt:     time.Now(),
		Version:       1, // Will be set by repository based on current version
	}, nil
}

// UnmarshalData deserializes the event data into the provided interface
func (e *Event) UnmarshalData(target interface{}) error {
	return json.Unmarshal(e.EventData, target)
}

// UnmarshalMetadata deserializes the event metadata
func (e *Event) UnmarshalMetadata() (*EventMetadata, error) {
	if len(e.EventMetadata) == 0 {
		return nil, nil
	}

	var metadata EventMetadata
	err := json.Unmarshal(e.EventMetadata, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event metadata: %w", err)
	}

	return &metadata, nil
}
