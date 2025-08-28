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

// EventRepository handles event sourcing operations
type EventRepository struct {
	pool *pgxpool.Pool
}

// NewEventRepository creates a new event repository
func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

// AppendEvent appends a new event to the event store
func (r *EventRepository) AppendEvent(ctx context.Context, event *domain.Event) (*domain.Event, error) {
	// Get the current version for this aggregate
	currentVersion, err := r.getCurrentVersion(ctx, event.AggregateType, event.AggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	event.Version = currentVersion + 1
	event.CreatedAt = time.Now()

	query := `
		INSERT INTO events (id, aggregate_type, aggregate_id, event_type, event_data, event_metadata, created_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	var eventID uuid.UUID
	var createdAt time.Time

	err = r.pool.QueryRow(ctx, query,
		event.ID,
		event.AggregateType,
		event.AggregateID,
		event.EventType,
		event.EventData,
		event.EventMetadata,
		event.CreatedAt,
		event.Version,
	).Scan(&eventID, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to append event: %w", err)
	}

	event.ID = eventID
	event.CreatedAt = createdAt

	return event, nil
}

// GetEventsByAggregate retrieves all events for a specific aggregate
func (r *EventRepository) GetEventsByAggregate(ctx context.Context, aggregateType domain.AggregateType, aggregateID uuid.UUID) ([]*domain.Event, error) {
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, event_data, event_metadata, created_at, version
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC
	`

	rows, err := r.pool.Query(ctx, query, string(aggregateType), aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by aggregate: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		var event domain.Event
		var eventMetadata []byte

		err := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&event.EventData,
			&eventMetadata,
			&event.CreatedAt,
			&event.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if len(eventMetadata) > 0 {
			event.EventMetadata = eventMetadata
		}

		events = append(events, &event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// GetEventsByType retrieves events by event type
func (r *EventRepository) GetEventsByType(ctx context.Context, eventType domain.EventType, limit int, offset int) ([]*domain.Event, error) {
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, event_data, event_metadata, created_at, version
		FROM events
		WHERE event_type = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, string(eventType), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by type: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		var event domain.Event
		var eventMetadata []byte

		err := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&event.EventData,
			&eventMetadata,
			&event.CreatedAt,
			&event.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if len(eventMetadata) > 0 {
			event.EventMetadata = eventMetadata
		}

		events = append(events, &event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// GetEventsSince retrieves events since a specific time
func (r *EventRepository) GetEventsSince(ctx context.Context, since time.Time, limit int) ([]*domain.Event, error) {
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, event_data, event_metadata, created_at, version
		FROM events
		WHERE created_at > $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get events since: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		var event domain.Event
		var eventMetadata []byte

		err := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&event.EventData,
			&eventMetadata,
			&event.CreatedAt,
			&event.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if len(eventMetadata) > 0 {
			event.EventMetadata = eventMetadata
		}

		events = append(events, &event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// GetAggregateVersion returns the current version of an aggregate
func (r *EventRepository) GetAggregateVersion(ctx context.Context, aggregateType domain.AggregateType, aggregateID uuid.UUID) (int, error) {
	return r.getCurrentVersion(ctx, string(aggregateType), aggregateID)
}

// getCurrentVersion gets the current version for an aggregate (internal method)
func (r *EventRepository) getCurrentVersion(ctx context.Context, aggregateType string, aggregateID uuid.UUID) (int, error) {
	query := `
		SELECT COALESCE(MAX(version), 0)
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
	`

	var version int
	err := r.pool.QueryRow(ctx, query, aggregateType, aggregateID).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return version, nil
}

// EventEnvelope represents an event with its deserialized data
type EventEnvelope struct {
	Event    *domain.Event
	Data     interface{}
	Metadata *domain.EventMetadata
}

// LoadEventEnvelope loads an event with its deserialized data
func (r *EventRepository) LoadEventEnvelope(ctx context.Context, event *domain.Event, target interface{}) (*EventEnvelope, error) {
	err := event.UnmarshalData(target)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	metadata, err := event.UnmarshalMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event metadata: %w", err)
	}

	return &EventEnvelope{
		Event:    event,
		Data:     target,
		Metadata: metadata,
	}, nil
}

// AppendEvents appends multiple events in a single transaction
func (r *EventRepository) AppendEvents(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, event := range events {
		currentVersion, err := r.getCurrentVersionTx(ctx, tx, event.AggregateType, event.AggregateID)
		if err != nil {
			return fmt.Errorf("failed to get current version: %w", err)
		}

		event.Version = currentVersion + 1
		event.CreatedAt = time.Now()

		query := `
			INSERT INTO events (id, aggregate_type, aggregate_id, event_type, event_data, event_metadata, created_at, version)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		_, err = tx.Exec(ctx, query,
			event.ID,
			event.AggregateType,
			event.AggregateID,
			event.EventType,
			event.EventData,
			event.EventMetadata,
			event.CreatedAt,
			event.Version,
		)
		if err != nil {
			return fmt.Errorf("failed to append event %s: %w", event.EventType, err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// getCurrentVersionTx gets the current version for an aggregate within a transaction
func (r *EventRepository) getCurrentVersionTx(ctx context.Context, tx pgx.Tx, aggregateType string, aggregateID uuid.UUID) (int, error) {
	query := `
		SELECT COALESCE(MAX(version), 0)
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
	`

	var version int
	err := tx.QueryRow(ctx, query, aggregateType, aggregateID).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return version, nil
}
