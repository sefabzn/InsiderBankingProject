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

// ProjectorService handles event projection to rebuild read models from events
type ProjectorService struct {
	eventRepo       repository.EventsRepo
	userRepo        repository.UsersRepo
	balanceRepo     repository.BalancesRepo
	transactionRepo repository.TransactionsRepo
}

// NewProjectorService creates a new projector service
func NewProjectorService(
	eventRepo repository.EventsRepo,
	userRepo repository.UsersRepo,
	balanceRepo repository.BalancesRepo,
	transactionRepo repository.TransactionsRepo,
) *ProjectorService {
	return &ProjectorService{
		eventRepo:       eventRepo,
		userRepo:        userRepo,
		balanceRepo:     balanceRepo,
		transactionRepo: transactionRepo,
	}
}

// ProjectAll rebuilds all read models from events
func (p *ProjectorService) ProjectAll(ctx context.Context) error {
	utils.Info("starting full projection of all aggregates")

	// Project users
	if err := p.projectUsers(ctx); err != nil {
		return fmt.Errorf("failed to project users: %w", err)
	}

	// Project balances
	if err := p.projectBalances(ctx); err != nil {
		return fmt.Errorf("failed to project balances: %w", err)
	}

	// Project transactions
	if err := p.projectTransactions(ctx); err != nil {
		return fmt.Errorf("failed to project transactions: %w", err)
	}

	utils.Info("completed full projection")
	return nil
}

// ProjectUser rebuilds a specific user's state from events
func (p *ProjectorService) ProjectUser(ctx context.Context, userID uuid.UUID) error {
	utils.Info("projecting user", "user_id", userID.String())

	events, err := p.eventRepo.GetEventsByAggregate(ctx, domain.AggregateUser, userID)
	if err != nil {
		return fmt.Errorf("failed to get user events: %w", err)
	}

	return p.projectUserEvents(ctx, userID, events)
}

// ProjectBalance rebuilds a specific balance from events
func (p *ProjectorService) ProjectBalance(ctx context.Context, userID uuid.UUID) error {
	utils.Info("projecting balance", "user_id", userID.String())

	events, err := p.eventRepo.GetEventsByAggregate(ctx, domain.AggregateBalance, userID)
	if err != nil {
		return fmt.Errorf("failed to get balance events: %w", err)
	}

	return p.projectBalanceEvents(ctx, userID, events)
}

// projectUsers rebuilds all user read models
func (p *ProjectorService) projectUsers(ctx context.Context) error {
	// Get all user registered events
	events, err := p.eventRepo.GetEventsByType(ctx, domain.EventUserRegistered, 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to get user registered events: %w", err)
	}

	for _, event := range events {
		var eventData domain.UserRegisteredEvent
		if err := event.UnmarshalData(&eventData); err != nil {
			utils.Error("failed to unmarshal user registered event", "error", err.Error())
			continue
		}

		// Create or update user in read model
		user := &domain.User{
			ID:           eventData.UserID,
			Username:     eventData.Username,
			Email:        eventData.Email,
			PasswordHash: eventData.PasswordHash,
			Role:         eventData.Role,
			IsActive:     true,
			CreatedAt:    event.CreatedAt,
			UpdatedAt:    event.CreatedAt,
		}

		if err := p.userRepo.Create(ctx, user); err != nil {
			// If user already exists, update instead
			if err := p.userRepo.Update(ctx, user); err != nil {
				utils.Error("failed to create/update user in projection", "error", err.Error())
			}
		}
	}

	return nil
}

// projectBalances rebuilds all balance read models
func (p *ProjectorService) projectBalances(ctx context.Context) error {
	// Get all balance-related events
	eventTypes := []domain.EventType{
		domain.EventBalanceInitialized,
		domain.EventAmountCredited,
		domain.EventAmountDebited,
	}

	for _, eventType := range eventTypes {
		events, err := p.eventRepo.GetEventsByType(ctx, eventType, 1000, 0)
		if err != nil {
			return fmt.Errorf("failed to get %s events: %w", eventType, err)
		}

		for _, event := range events {
			if err := p.projectBalanceEvent(ctx, event); err != nil {
				utils.Error("failed to project balance event", "error", err.Error(), "event_type", eventType)
			}
		}
	}

	return nil
}

// projectTransactions rebuilds all transaction read models
func (p *ProjectorService) projectTransactions(ctx context.Context) error {
	// Get all transaction-related events
	eventTypes := []domain.EventType{
		domain.EventTransactionStarted,
		domain.EventTransactionCompleted,
		domain.EventTransactionFailed,
		domain.EventTransferExecuted,
	}

	for _, eventType := range eventTypes {
		events, err := p.eventRepo.GetEventsByType(ctx, eventType, 1000, 0)
		if err != nil {
			return fmt.Errorf("failed to get %s events: %w", eventType, err)
		}

		for _, event := range events {
			if err := p.projectTransactionEvent(ctx, event); err != nil {
				utils.Error("failed to project transaction event", "error", err.Error(), "event_type", eventType)
			}
		}
	}

	return nil
}

// projectUserEvents applies events for a specific user
func (p *ProjectorService) projectUserEvents(ctx context.Context, userID uuid.UUID, events []*domain.Event) error {
	for _, event := range events {
		switch event.EventType {
		case string(domain.EventUserRegistered):
			var eventData domain.UserRegisteredEvent
			if err := event.UnmarshalData(&eventData); err != nil {
				return err
			}

			user := &domain.User{
				ID:           eventData.UserID,
				Username:     eventData.Username,
				Email:        eventData.Email,
				PasswordHash: eventData.PasswordHash,
				Role:         eventData.Role,
				IsActive:     true,
				CreatedAt:    event.CreatedAt,
				UpdatedAt:    event.CreatedAt,
			}

			p.userRepo.Create(ctx, user) // Ignore error if user exists

		case string(domain.EventUserUpdated):
			var eventData domain.UserUpdatedEvent
			if err := event.UnmarshalData(&eventData); err != nil {
				return err
			}

			// Get current user and apply updates
			user, err := p.userRepo.GetByID(ctx, userID)
			if err != nil {
				return err
			}

			// Apply new data from event
			if newUsername, ok := eventData.NewData["username"].(string); ok {
				user.Username = newUsername
			}
			if newEmail, ok := eventData.NewData["email"].(string); ok {
				user.Email = newEmail
			}
			if newRole, ok := eventData.NewData["role"].(string); ok {
				user.Role = newRole
			}
			user.UpdatedAt = event.CreatedAt

			p.userRepo.Update(ctx, user)
		}
	}
	return nil
}

// projectBalanceEvents applies events for a specific balance
func (p *ProjectorService) projectBalanceEvents(ctx context.Context, userID uuid.UUID, events []*domain.Event) error {
	// Get current balance first (like projectBalanceEvent does)
	currentBalance, err := p.balanceRepo.GetByUserID(ctx, userID)
	if err != nil {
		// If no balance exists, start from zero
		currentBalance = &domain.Balance{
			UserID:   userID,
			Amount:   0.0,
			Currency: "USD", // Default currency
		}
	}

	// Apply events on top of current balance
	for _, event := range events {
		switch event.EventType {
		case string(domain.EventBalanceInitialized):
			var eventData domain.BalanceInitializedEvent
			if err := event.UnmarshalData(&eventData); err != nil {
				return err
			}
			// Only set initial amount if balance is zero (prevent double initialization)
			if currentBalance.Amount == 0.0 {
				currentBalance.Amount = eventData.Amount
				currentBalance.Currency = eventData.Currency
			}

		case string(domain.EventAmountCredited):
			var eventData domain.AmountCreditedEvent
			if err := event.UnmarshalData(&eventData); err != nil {
				return err
			}
			currentBalance.Amount += eventData.Amount

		case string(domain.EventAmountDebited):
			var eventData domain.AmountDebitedEvent
			if err := event.UnmarshalData(&eventData); err != nil {
				return err
			}
			currentBalance.Amount -= eventData.Amount
		}
		currentBalance.LastUpdatedAt = event.CreatedAt
	}

	// Update balance in read model
	return p.balanceRepo.Upsert(ctx, currentBalance)
}

// projectBalanceEvent applies a single balance event
func (p *ProjectorService) projectBalanceEvent(ctx context.Context, event *domain.Event) error {
	switch event.EventType {
	case string(domain.EventBalanceInitialized):
		var eventData domain.BalanceInitializedEvent
		if err := event.UnmarshalData(&eventData); err != nil {
			return err
		}

		balance := &domain.Balance{
			UserID:        eventData.UserID,
			Amount:        eventData.Amount,
			Currency:      eventData.Currency,
			LastUpdatedAt: event.CreatedAt,
		}
		return p.balanceRepo.Upsert(ctx, balance)

	case string(domain.EventAmountCredited):
		var eventData domain.AmountCreditedEvent
		if err := event.UnmarshalData(&eventData); err != nil {
			return err
		}

		// Get current balance and add amount
		balance, err := p.balanceRepo.GetByUserID(ctx, eventData.UserID)
		if err != nil {
			// Create new balance if not exists
			balance = &domain.Balance{
				UserID:        eventData.UserID,
				Amount:        eventData.Amount,
				LastUpdatedAt: event.CreatedAt,
			}
		} else {
			balance.Amount += eventData.Amount
			balance.LastUpdatedAt = event.CreatedAt
		}
		return p.balanceRepo.Upsert(ctx, balance)

	case string(domain.EventAmountDebited):
		var eventData domain.AmountDebitedEvent
		if err := event.UnmarshalData(&eventData); err != nil {
			return err
		}

		// Get current balance and subtract amount
		balance, err := p.balanceRepo.GetByUserID(ctx, eventData.UserID)
		if err != nil {
			return err
		}
		balance.Amount -= eventData.Amount
		balance.LastUpdatedAt = event.CreatedAt
		return p.balanceRepo.Upsert(ctx, balance)
	}

	return nil
}

// projectTransactionEvent applies a single transaction event
func (p *ProjectorService) projectTransactionEvent(ctx context.Context, event *domain.Event) error {
	switch event.EventType {
	case string(domain.EventTransactionStarted):
		var eventData domain.TransactionStartedEvent
		if err := event.UnmarshalData(&eventData); err != nil {
			return err
		}

		transaction := &domain.Transaction{
			ID:         eventData.TransactionID,
			FromUserID: eventData.FromUserID,
			ToUserID:   eventData.ToUserID,
			Amount:     eventData.Amount,
			Type:       eventData.Type,
			Status:     string(domain.StatusPending),
			CreatedAt:  event.CreatedAt,
		}
		return p.transactionRepo.CreatePending(ctx, transaction)

	case string(domain.EventTransactionCompleted):
		var eventData domain.TransactionCompletedEvent
		if err := event.UnmarshalData(&eventData); err != nil {
			return err
		}
		return p.transactionRepo.MarkCompleted(ctx, eventData.TransactionID)

	case string(domain.EventTransactionFailed):
		var eventData domain.TransactionFailedEvent
		if err := event.UnmarshalData(&eventData); err != nil {
			return err
		}
		return p.transactionRepo.MarkFailed(ctx, eventData.TransactionID)

	case string(domain.EventTransferExecuted):
		var eventData domain.TransferExecutedEvent
		if err := event.UnmarshalData(&eventData); err != nil {
			return err
		}

		transaction := &domain.Transaction{
			ID:         eventData.TransactionID,
			FromUserID: &eventData.FromUserID,
			ToUserID:   &eventData.ToUserID,
			Amount:     eventData.Amount,
			Type:       string(domain.TypeTransfer),
			Status:     string(domain.StatusSuccess),
			CreatedAt:  event.CreatedAt,
		}
		return p.transactionRepo.CreatePending(ctx, transaction)
	}

	return nil
}

// RebuildReadModels completely rebuilds all read models from scratch
func (p *ProjectorService) RebuildReadModels(ctx context.Context) error {
	utils.Info("starting complete read model rebuild")

	// Clear existing data (optional - depends on requirements)
	// This would require additional methods in repositories to truncate tables

	// Project all aggregates
	return p.ProjectAll(ctx)
}

// ProcessAllEvents processes all events from the beginning
func (p *ProjectorService) ProcessAllEvents(ctx context.Context) error {
	utils.Info("processing all events from the beginning")

	// Get all events from the beginning
	since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) // Very early date to get all events
	return p.ProcessEventsSince(ctx, since)
}

// ProcessEventsSince processes events since a specific time
func (p *ProjectorService) ProcessEventsSince(ctx context.Context, since time.Time) error {
	utils.Info("processing events since", "since", since.Format(time.RFC3339))

	events, err := p.eventRepo.GetEventsSince(ctx, since, 1000)
	if err != nil {
		return fmt.Errorf("failed to get events since %v: %w", since, err)
	}

	for _, event := range events {
		switch domain.AggregateType(event.AggregateType) {
		case domain.AggregateUser:
			var eventData domain.UserRegisteredEvent
			if err := event.UnmarshalData(&eventData); err != nil {
				utils.Error("failed to process user event", "error", err.Error())
				continue
			}
			if err := p.projectUserEvents(ctx, event.AggregateID, []*domain.Event{event}); err != nil {
				utils.Error("failed to project user event", "error", err.Error())
			}

		case domain.AggregateBalance:
			if err := p.projectBalanceEvent(ctx, event); err != nil {
				utils.Error("failed to project balance event", "error", err.Error())
			}

		case domain.AggregateTransaction:
			if err := p.projectTransactionEvent(ctx, event); err != nil {
				utils.Error("failed to project transaction event", "error", err.Error())
			}
		}
	}

	utils.Info("completed processing events", "count", len(events))
	return nil
}
