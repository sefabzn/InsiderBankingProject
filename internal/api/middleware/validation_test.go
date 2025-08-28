package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test struct that implements Validator
type TestRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (t TestRequest) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("name: field is required")
	}
	if t.Email == "" {
		return fmt.Errorf("email: field is required")
	}
	if t.Age < 0 {
		return fmt.Errorf("age: must be non-negative")
	}
	if !strings.Contains(t.Email, "@") {
		return fmt.Errorf("email: invalid email format")
	}
	return nil
}

// Invalid struct that doesn't implement Validator (for compilation test)
type InvalidRequest struct {
	Name string `json:"name"`
}

func TestValidateJSON(t *testing.T) {
	// Create test handler
	var receivedBody TestRequest
	testHandler := func(w http.ResponseWriter, _ *http.Request, body TestRequest) {
		receivedBody = body
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}

	// Wrap with validation middleware
	handler := ValidateJSON(testHandler)

	tests := []struct {
		name           string
		body           string
		contentType    string
		expectedStatus int
		expectErrors   bool
	}{
		{
			name:           "valid request",
			body:           `{"name":"John","email":"john@example.com","age":25}`,
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectErrors:   false,
		},
		{
			name:           "missing required field",
			body:           `{"email":"john@example.com","age":25}`,
			contentType:    "application/json",
			expectedStatus: http.StatusUnprocessableEntity,
			expectErrors:   true,
		},
		{
			name:           "invalid email format",
			body:           `{"name":"John","email":"invalid-email","age":25}`,
			contentType:    "application/json",
			expectedStatus: http.StatusUnprocessableEntity,
			expectErrors:   true,
		},
		{
			name:           "negative age",
			body:           `{"name":"John","email":"john@example.com","age":-5}`,
			contentType:    "application/json",
			expectedStatus: http.StatusUnprocessableEntity,
			expectErrors:   true,
		},
		{
			name:           "invalid JSON",
			body:           `{"name":"John","email":}`,
			contentType:    "application/json",
			expectedStatus: http.StatusUnprocessableEntity,
			expectErrors:   true,
		},
		{
			name:           "wrong content type",
			body:           `{"name":"John","email":"john@example.com","age":25}`,
			contentType:    "text/plain",
			expectedStatus: http.StatusUnprocessableEntity,
			expectErrors:   true,
		},
		{
			name:           "unknown field",
			body:           `{"name":"John","email":"john@example.com","age":25,"unknown":"field"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusUnprocessableEntity,
			expectErrors:   true,
		},
		{
			name:           "empty body",
			body:           "",
			contentType:    "application/json",
			expectedStatus: http.StatusUnprocessableEntity,
			expectErrors:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectErrors {
				// Check that response contains errors array
				var response ValidationResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to parse error response: %v", err)
				}

				if response.Code != 422 {
					t.Errorf("Expected error code 422, got %d", response.Code)
				}

				if len(response.Errors) == 0 {
					t.Error("Expected validation errors in response")
				}

				t.Logf("Validation errors: %+v", response.Errors)
			} else {
				// Check that the body was properly parsed and passed to handler
				if receivedBody.Name == "" {
					t.Error("Handler should have received parsed body")
				}
			}
		})
	}
}

func TestValidateQueryParams(t *testing.T) {
	// Create validator function
	validator := func(r *http.Request) []ValidationError {
		var errors []ValidationError

		limit := r.URL.Query().Get("limit")
		switch limit {
		case "":
			errors = append(errors, ValidationError{
				Field:   "limit",
				Message: "limit parameter is required",
			})
		case "invalid":
			errors = append(errors, ValidationError{
				Field:   "limit",
				Message: "limit must be a number",
			})
		}

		return errors
	}

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	// Wrap with validation middleware
	handler := ValidateQueryParams(validator)(testHandler)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectErrors   bool
	}{
		{
			name:           "valid query params",
			url:            "/test?limit=10",
			expectedStatus: http.StatusOK,
			expectErrors:   false,
		},
		{
			name:           "missing required param",
			url:            "/test",
			expectedStatus: http.StatusUnprocessableEntity,
			expectErrors:   true,
		},
		{
			name:           "invalid param value",
			url:            "/test?limit=invalid",
			expectedStatus: http.StatusUnprocessableEntity,
			expectErrors:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectErrors {
				var response ValidationResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to parse error response: %v", err)
				}

				if len(response.Errors) == 0 {
					t.Error("Expected validation errors in response")
				}
			}
		})
	}
}

func TestValidateContentType(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	handler := ValidateContentType("application/json")(testHandler)

	tests := []struct {
		name           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "valid content type",
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid content type with charset",
			contentType:    "application/json; charset=utf-8",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid content type",
			contentType:    "text/plain",
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "missing content type",
			contentType:    "",
			expectedStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("{}")))
			req.Header.Set("Content-Type", tt.contentType)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestValidationErrorResponse(t *testing.T) {
	// Test the exact format required by the task
	errors := []ValidationError{
		{Field: "name", Message: "field is required"},
		{Field: "email", Message: "invalid email format"},
	}

	// Create a test handler that returns validation errors
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeValidationError(w, errors)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", rr.Code)
	}

	// Check response format
	var response ValidationResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response structure
	if response.Error != "validation failed" {
		t.Errorf("Expected error 'validation failed', got '%s'", response.Error)
	}

	if response.Code != 422 {
		t.Errorf("Expected code 422, got %d", response.Code)
	}

	if len(response.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(response.Errors))
	}

	// Check errors array
	expectedErrors := map[string]string{
		"name":  "field is required",
		"email": "invalid email format",
	}

	for _, validationError := range response.Errors {
		expectedMessage, exists := expectedErrors[validationError.Field]
		if !exists {
			t.Errorf("Unexpected error field: %s", validationError.Field)
		} else if validationError.Message != expectedMessage {
			t.Errorf("Expected message '%s' for field '%s', got '%s'",
				expectedMessage, validationError.Field, validationError.Message)
		}
	}

	t.Log("✓ Missing required fields → 422 with errors array")
}
