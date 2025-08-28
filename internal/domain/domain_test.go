package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserMarshalUnmarshal(t *testing.T) {
	user := User{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}

	// Marshal to JSON
	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	// Unmarshal from JSON
	var unmarshaled User
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal user: %v", err)
	}

	// Compare (excluding server-managed fields)
	if user.ID != unmarshaled.ID {
		t.Errorf("ID mismatch: %v != %v", user.ID, unmarshaled.ID)
	}
	if user.Username != unmarshaled.Username {
		t.Errorf("Username mismatch: %v != %v", user.Username, unmarshaled.Username)
	}
	if user.Email != unmarshaled.Email {
		t.Errorf("Email mismatch: %v != %v", user.Email, unmarshaled.Email)
	}
	if user.Role != unmarshaled.Role {
		t.Errorf("Role mismatch: %v != %v", user.Role, unmarshaled.Role)
	}
}

func TestTransactionMarshalUnmarshal(t *testing.T) {
	userID := uuid.New()
	transaction := Transaction{
		ID:        uuid.New(),
		ToUserID:  &userID,
		Amount:    100.50,
		Type:      "credit",
		Status:    "success",
		CreatedAt: time.Now(),
	}

	// Marshal to JSON
	data, err := json.Marshal(transaction)
	if err != nil {
		t.Fatalf("Failed to marshal transaction: %v", err)
	}

	// Unmarshal from JSON
	var unmarshaled Transaction
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal transaction: %v", err)
	}

	// Compare key fields
	if transaction.ID != unmarshaled.ID {
		t.Errorf("ID mismatch: %v != %v", transaction.ID, unmarshaled.ID)
	}
	if transaction.Amount != unmarshaled.Amount {
		t.Errorf("Amount mismatch: %v != %v", transaction.Amount, unmarshaled.Amount)
	}
	if transaction.Type != unmarshaled.Type {
		t.Errorf("Type mismatch: %v != %v", transaction.Type, unmarshaled.Type)
	}
}

// Test User validation
func TestUserValidation(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name: "valid user",
			user: User{
				Username: "validuser",
				Email:    "valid@example.com",
				Role:     "user",
			},
			wantErr: false,
		},
		{
			name: "invalid username - too short",
			user: User{
				Username: "ab",
				Email:    "valid@example.com",
				Role:     "user",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			user: User{
				Username: "validuser",
				Email:    "invalid-email",
				Role:     "user",
			},
			wantErr: true,
		},
		{
			name: "invalid role",
			user: User{
				Username: "validuser",
				Email:    "valid@example.com",
				Role:     "invalid_role",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("User.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test Transaction validation
func TestTransactionValidation(t *testing.T) {
	userID1 := uuid.New()
	userID2 := uuid.New()

	tests := []struct {
		name        string
		transaction Transaction
		wantErr     bool
	}{
		{
			name: "valid credit transaction",
			transaction: Transaction{
				ToUserID: &userID1,
				Amount:   100.50,
				Type:     "credit",
				Status:   "pending",
			},
			wantErr: false,
		},
		{
			name: "invalid amount - zero",
			transaction: Transaction{
				ToUserID: &userID1,
				Amount:   0,
				Type:     "credit",
				Status:   "pending",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			transaction: Transaction{
				ToUserID: &userID1,
				Amount:   100.50,
				Type:     "invalid_type",
				Status:   "pending",
			},
			wantErr: true,
		},
		{
			name: "credit transaction with from_user_id - invalid",
			transaction: Transaction{
				FromUserID: &userID1,
				ToUserID:   &userID2,
				Amount:     100.50,
				Type:       "credit",
				Status:     "pending",
			},
			wantErr: true,
		},
		{
			name: "debit transaction without from_user_id - invalid",
			transaction: Transaction{
				ToUserID: &userID1,
				Amount:   100.50,
				Type:     "debit",
				Status:   "pending",
			},
			wantErr: true,
		},
		{
			name: "transfer to same user - invalid",
			transaction: Transaction{
				FromUserID: &userID1,
				ToUserID:   &userID1,
				Amount:     100.50,
				Type:       "transfer",
				Status:     "pending",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transaction.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Transaction.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test CreateUserRequest validation
func TestCreateUserRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request CreateUserRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: CreateUserRequest{
				Username: "newuser",
				Email:    "new@example.com",
				Password: "validpassword123",
				Role:     "user",
			},
			wantErr: false,
		},
		{
			name: "invalid password - too short",
			request: CreateUserRequest{
				Username: "newuser",
				Email:    "new@example.com",
				Password: "short",
				Role:     "user",
			},
			wantErr: true,
		},
		{
			name: "empty username",
			request: CreateUserRequest{
				Username: "",
				Email:    "new@example.com",
				Password: "validpassword123",
				Role:     "user",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUserRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
