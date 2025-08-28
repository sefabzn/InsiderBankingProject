package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Balance represents a user's account balance.
type Balance struct {
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	Amount        float64   `json:"amount" db:"amount"`
	Currency      string    `json:"currency" db:"currency"`
	LastUpdatedAt time.Time `json:"last_updated_at" db:"last_updated_at"`
}

// BalanceResponse represents a balance in API responses.
type BalanceResponse struct {
	UserID        uuid.UUID `json:"user_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

// ToResponse converts a Balance to BalanceResponse.
func (b *Balance) ToResponse() BalanceResponse {
	return BalanceResponse{
		UserID:        b.UserID,
		Amount:        b.Amount,
		Currency:      b.Currency,
		LastUpdatedAt: b.LastUpdatedAt,
	}
}

// BalanceHistoryItem represents a historical balance snapshot.
type BalanceHistoryItem struct {
	UserID    uuid.UUID `json:"user_id"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Timestamp time.Time `json:"timestamp"`
	Reason    string    `json:"reason"`
}

// Currency represents a supported currency code
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyGBP Currency = "GBP"
	CurrencyJPY Currency = "JPY"
	CurrencyCAD Currency = "CAD"
	CurrencyAUD Currency = "AUD"
)

// SupportedCurrencies returns all supported currency codes
func SupportedCurrencies() []Currency {
	return []Currency{
		CurrencyUSD, CurrencyEUR, CurrencyGBP,
		CurrencyJPY, CurrencyCAD, CurrencyAUD,
	}
}

// IsValidCurrency checks if a currency code is supported
func IsValidCurrency(currency string) bool {
	supported := SupportedCurrencies()
	for _, c := range supported {
		if string(c) == currency {
			return true
		}
	}
	return false
}

// Validate validates the balance data including currency
func (b *Balance) Validate() error {
	if err := validateAmount(b.Amount); err != nil {
		return err
	}
	if !IsValidCurrency(b.Currency) {
		return fmt.Errorf("unsupported currency: %s", b.Currency)
	}
	return nil
}

// validateAmount validates balance amount
func validateAmount(amount float64) error {
	if amount < 0 {
		return fmt.Errorf("amount cannot be negative")
	}
	if amount > 10000000 { // reasonable upper limit for balance
		return fmt.Errorf("amount cannot exceed 10,000,000")
	}
	return nil
}
