// Package domain contains the core business entities and types.
package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the banking system.
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never expose password hash
	Role         string    `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	IsActive     bool      `json:"is_active" db:"is_active"`
}

// UserRole defines valid user roles.
type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// CreateUserRequest represents the data needed to create a new user.
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
}

// UpdateUserRequest represents the data that can be updated for a user.
type UpdateUserRequest struct {
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	Role     string `json:"role,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// LoginRequest represents the data needed for user login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest represents the data needed for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// UserResponse represents a user in API responses (without sensitive data).
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsActive  bool      `json:"is_active"`
}

// ToResponse converts a User to UserResponse.
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		IsActive:  u.IsActive,
	}
}

// Validate validates the user data.
func (u *User) Validate() error {
	if err := validateUsername(u.Username); err != nil {
		return fmt.Errorf("username: %w", err)
	}

	if err := validateEmail(u.Email); err != nil {
		return fmt.Errorf("email: %w", err)
	}

	if err := validateRole(u.Role); err != nil {
		return fmt.Errorf("role: %w", err)
	}

	return nil
}

// Validate validates the create user request.
func (r *CreateUserRequest) Validate() error {
	if err := validateUsername(r.Username); err != nil {
		return fmt.Errorf("username: %w", err)
	}

	if err := validateEmail(r.Email); err != nil {
		return fmt.Errorf("email: %w", err)
	}

	if err := validatePassword(r.Password); err != nil {
		return fmt.Errorf("password: %w", err)
	}

	if r.Role != "" {
		if err := validateRole(r.Role); err != nil {
			return fmt.Errorf("role: %w", err)
		}
	}

	return nil
}

// Validate validates the login request.
func (r *LoginRequest) Validate() error {
	if err := validateEmail(r.Email); err != nil {
		return fmt.Errorf("email: %w", err)
	}

	if r.Password == "" {
		return fmt.Errorf("password: password is required")
	}

	return nil
}

// Validate validates the refresh request.
func (r *RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return fmt.Errorf("refresh_token: refresh token is required")
	}

	return nil
}

// Validate validates the update user request.
func (r *UpdateUserRequest) Validate() error {
	if r.Username != "" {
		if err := validateUsername(r.Username); err != nil {
			return fmt.Errorf("username: %w", err)
		}
	}

	if r.Role != "" {
		if r.Role != "admin" && r.Role != "user" {
			return fmt.Errorf("role: must be either 'admin' or 'user'")
		}
	}

	// At least one field must be provided
	if r.Username == "" && r.Role == "" {
		return fmt.Errorf("at least one field (username or role) must be provided")
	}

	return nil
}

// validateUsername validates username format and length.
func validateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}

	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}

	if len(username) > 50 {
		return fmt.Errorf("username must be at most 50 characters")
	}

	// Username can only contain alphanumeric characters and underscores
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(username) {
		return fmt.Errorf("username can only contain letters, numbers, and underscores")
	}

	return nil
}

// validateEmail validates email format.
func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}

	if len(email) > 255 {
		return fmt.Errorf("email must be at most 255 characters")
	}

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// validatePassword validates password strength.
func validatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("password is required")
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	if len(password) > 72 { // bcrypt limit
		return fmt.Errorf("password must be at most 72 characters")
	}

	return nil
}

// validateRole validates user role.
func validateRole(role string) error {
	role = strings.ToLower(role)
	if role != string(RoleUser) && role != string(RoleAdmin) {
		return fmt.Errorf("invalid role, must be 'user' or 'admin'")
	}

	return nil
}
