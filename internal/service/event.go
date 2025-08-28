package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/domain"
	"github.com/sefa-b/go-banking-sim/internal/repository"
	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// EventService handles event sourcing operations
type EventService struct {
	eventRepo repository.EventsRepo
}

// NewEventService creates a new event service
func NewEventService(eventRepo repository.EventsRepo) *EventService {
	return &EventService{
		eventRepo: eventRepo,
	}
}

// PublishEvent publishes an event to the event store
func (s *EventService) PublishEvent(ctx context.Context, aggregateType domain.AggregateType, aggregateID uuid.UUID, eventType domain.EventType, eventData interface{}, metadata *domain.EventMetadata) (*domain.Event, error) {
	event, err := domain.NewEvent(aggregateType, aggregateID, eventType, eventData, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	publishedEvent, err := s.eventRepo.AppendEvent(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}

	utils.Info("event published",
		"event_type", eventType,
		"aggregate_type", aggregateType,
		"aggregate_id", aggregateID.String(),
		"version", publishedEvent.Version,
	)

	return publishedEvent, nil
}

// PublishEvents publishes multiple events atomically
func (s *EventService) PublishEvents(ctx context.Context, events []*domain.Event) error {
	err := s.eventRepo.AppendEvents(ctx, events)
	if err != nil {
		return fmt.Errorf("failed to publish events: %w", err)
	}

	utils.Info("events published", "count", len(events))
	return nil
}

// GetAggregateEvents retrieves all events for an aggregate
func (s *EventService) GetAggregateEvents(ctx context.Context, aggregateType domain.AggregateType, aggregateID uuid.UUID) ([]*domain.Event, error) {
	return s.eventRepo.GetEventsByAggregate(ctx, aggregateType, aggregateID)
}

// ReplayAggregateEvents replays events for an aggregate to rebuild state
func (s *EventService) ReplayAggregateEvents(ctx context.Context, aggregateType domain.AggregateType, aggregateID uuid.UUID) ([]*repository.EventEnvelope, error) {
	events, err := s.GetAggregateEvents(ctx, aggregateType, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregate events: %w", err)
	}

	var envelopes []*repository.EventEnvelope

	for _, event := range events {
		envelope, err := s.eventRepo.LoadEventEnvelope(ctx, event, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to load event envelope: %w", err)
		}
		envelopes = append(envelopes, envelope)
	}

	return envelopes, nil
}

// GetEventsByType retrieves events by type
func (s *EventService) GetEventsByType(ctx context.Context, eventType domain.EventType, limit int, offset int) ([]*domain.Event, error) {
	return s.eventRepo.GetEventsByType(ctx, eventType, limit, offset)
}

// GetEventsSince retrieves events since a specific time
func (s *EventService) GetEventsSince(ctx context.Context, since time.Time, limit int) ([]*domain.Event, error) {
	return s.eventRepo.GetEventsSince(ctx, since, limit)
}

// GetAggregateVersion returns the current version of an aggregate
func (s *EventService) GetAggregateVersion(ctx context.Context, aggregateType domain.AggregateType, aggregateID uuid.UUID) (int, error) {
	return s.eventRepo.GetAggregateVersion(ctx, aggregateType, aggregateID)
}

// UserRegistered publishes a UserRegistered event
func (s *EventService) UserRegistered(ctx context.Context, user *domain.User) error {
	eventData := &domain.UserRegisteredEvent{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Role:     user.Role,
	}

	metadata := &domain.EventMetadata{
		CorrelationID: getCorrelationID(ctx),
		UserAgent:     getUserAgent(ctx),
		IP:            getClientIP(ctx),
	}

	_, err := s.PublishEvent(ctx, domain.AggregateUser, user.ID, domain.EventUserRegistered, eventData, metadata)
	return err
}

// AmountCredited publishes an AmountCredited event
func (s *EventService) AmountCredited(ctx context.Context, userID uuid.UUID, amount float64, currency string, transactionID uuid.UUID, reason string) error {
	eventData := &domain.AmountCreditedEvent{
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		TransactionID: transactionID,
		Reason:        reason,
	}

	metadata := &domain.EventMetadata{
		CorrelationID: getCorrelationID(ctx),
		UserAgent:     getUserAgent(ctx),
		IP:            getClientIP(ctx),
	}

	_, err := s.PublishEvent(ctx, domain.AggregateBalance, userID, domain.EventAmountCredited, eventData, metadata)
	return err
}

// AmountDebited publishes an AmountDebited event
func (s *EventService) AmountDebited(ctx context.Context, userID uuid.UUID, amount float64, currency string, transactionID uuid.UUID, reason string) error {
	eventData := &domain.AmountDebitedEvent{
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		TransactionID: transactionID,
		Reason:        reason,
	}

	metadata := &domain.EventMetadata{
		CorrelationID: getCorrelationID(ctx),
		UserAgent:     getUserAgent(ctx),
		IP:            getClientIP(ctx),
	}

	_, err := s.PublishEvent(ctx, domain.AggregateBalance, userID, domain.EventAmountDebited, eventData, metadata)
	return err
}

// TransferExecuted publishes a TransferExecuted event
func (s *EventService) TransferExecuted(ctx context.Context, fromUserID, toUserID uuid.UUID, amount float64, currency string, transactionID uuid.UUID) error {
	eventData := &domain.TransferExecutedEvent{
		FromUserID:    fromUserID,
		ToUserID:      toUserID,
		Amount:        amount,
		Currency:      currency,
		TransactionID: transactionID,
	}

	metadata := &domain.EventMetadata{
		CorrelationID: getCorrelationID(ctx),
		UserAgent:     getUserAgent(ctx),
		IP:            getClientIP(ctx),
	}

	_, err := s.PublishEvent(ctx, domain.AggregateTransaction, transactionID, domain.EventTransferExecuted, eventData, metadata)
	return err
}

// TransactionStarted publishes a TransactionStarted event
func (s *EventService) TransactionStarted(ctx context.Context, transactionID uuid.UUID, transaction *domain.Transaction) error {
	eventData := &domain.TransactionStartedEvent{
		TransactionID: transactionID,
		FromUserID:    transaction.FromUserID,
		ToUserID:      transaction.ToUserID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
	}

	metadata := &domain.EventMetadata{
		CorrelationID: getCorrelationID(ctx),
		UserAgent:     getUserAgent(ctx),
		IP:            getClientIP(ctx),
	}

	aggregateID := transactionID
	if transaction.ToUserID != nil {
		aggregateID = *transaction.ToUserID
	}

	_, err := s.PublishEvent(ctx, domain.AggregateTransaction, aggregateID, domain.EventTransactionStarted, eventData, metadata)
	return err
}

// TransactionCompleted publishes a TransactionCompleted event
func (s *EventService) TransactionCompleted(ctx context.Context, transactionID uuid.UUID, transaction *domain.Transaction) error {
	eventData := &domain.TransactionCompletedEvent{
		TransactionID: transactionID,
		FromUserID:    transaction.FromUserID,
		ToUserID:      transaction.ToUserID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
	}

	metadata := &domain.EventMetadata{
		CorrelationID: getCorrelationID(ctx),
		UserAgent:     getUserAgent(ctx),
		IP:            getClientIP(ctx),
	}

	aggregateID := transactionID
	if transaction.ToUserID != nil {
		aggregateID = *transaction.ToUserID
	}

	_, err := s.PublishEvent(ctx, domain.AggregateTransaction, aggregateID, domain.EventTransactionCompleted, eventData, metadata)
	return err
}

// TransactionFailed publishes a TransactionFailed event
func (s *EventService) TransactionFailed(ctx context.Context, transactionID uuid.UUID, transaction *domain.Transaction, errorMsg string) error {
	eventData := &domain.TransactionFailedEvent{
		TransactionID: transactionID,
		FromUserID:    transaction.FromUserID,
		ToUserID:      transaction.ToUserID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		Error:         errorMsg,
	}

	metadata := &domain.EventMetadata{
		CorrelationID: getCorrelationID(ctx),
		UserAgent:     getUserAgent(ctx),
		IP:            getClientIP(ctx),
	}

	aggregateID := transactionID
	if transaction.ToUserID != nil {
		aggregateID = *transaction.ToUserID
	}

	_, err := s.PublishEvent(ctx, domain.AggregateTransaction, aggregateID, domain.EventTransactionFailed, eventData, metadata)
	return err
}

// Helper functions to extract context values
func getCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}
	return uuid.New().String()
}

func getUserAgent(ctx context.Context) string {
	if userAgent, ok := ctx.Value("user_agent").(string); ok {
		return userAgent
	}
	return ""
}

func getClientIP(ctx context.Context) string {
	if clientIP, ok := ctx.Value("client_ip").(string); ok {
		return clientIP
	}
	return ""
}
