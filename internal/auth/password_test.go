package auth

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	// Test successful hashing
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal original password")
	}

	// Test empty password
	_, err = HashPassword("")
	if err == nil {
		t.Error("HashPassword should fail with empty password")
	}
}

func TestComparePassword(t *testing.T) {
	password := "testpassword123"
	wrongPassword := "wrongpassword"

	// Generate hash
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Test correct password comparison
	if !ComparePassword(hash, password) {
		t.Error("Compare should return true for correct password")
	}

	// Test wrong password comparison
	if ComparePassword(hash, wrongPassword) {
		t.Error("Compare should return false for wrong password")
	}

	// Test empty inputs
	if ComparePassword("", password) {
		t.Error("Compare should return false for empty hash")
	}

	if ComparePassword(hash, "") {
		t.Error("Compare should return false for empty password")
	}

	if ComparePassword("", "") {
		t.Error("Compare should return false for both empty")
	}
}

func TestPasswordHashUniqueness(t *testing.T) {
	password := "testpassword123"

	// Generate multiple hashes of the same password
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)

	if err1 != nil || err2 != nil {
		t.Fatalf("HashPassword failed: %v, %v", err1, err2)
	}

	// Hashes should be different due to salt
	if hash1 == hash2 {
		t.Error("Multiple hashes of same password should be different")
	}

	// But both should verify against the original password
	if !ComparePassword(hash1, password) {
		t.Error("First hash should verify against password")
	}

	if !ComparePassword(hash2, password) {
		t.Error("Second hash should verify against password")
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	testCases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "validpassword123",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "short",
			wantErr:  true,
		},
		{
			name:     "too long",
			password: strings.Repeat("a", 73),
			wantErr:  true,
		},
		{
			name:     "minimum length",
			password: "12345678",
			wantErr:  false,
		},
		{
			name:     "maximum length",
			password: strings.Repeat("a", 72),
			wantErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tc.password)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidatePasswordStrength() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestPasswordRoundTrip(t *testing.T) {
	// Test the complete round trip: password -> hash -> verify
	testPasswords := []string{
		"simplepassword",
		"ComplexP@ssw0rd!",
		"with spaces and 123",
		"unicode-тест-123",
		strings.Repeat("a", 72), // max length
	}

	for _, password := range testPasswords {
		t.Run("password_"+password[:min(10, len(password))], func(t *testing.T) {
			// Hash the password
			hash, err := HashPassword(password)
			if err != nil {
				t.Fatalf("HashPassword failed: %v", err)
			}

			// Verify correct password
			if !ComparePassword(hash, password) {
				t.Error("Password verification failed for correct password")
			}

			// Verify wrong password fails
			// For very long passwords, modify instead of append to avoid bcrypt 72-byte limit
			var wrongPassword string
			if len(password) >= 70 {
				wrongPassword = "X" + password[1:] // Change first character
			} else {
				wrongPassword = password + "wrong"
			}

			if ComparePassword(hash, wrongPassword) {
				t.Errorf("Password verification should fail for wrong password. Original: %s, Wrong: %s", password, wrongPassword)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
