package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// Validator interface for types that can validate themselves.
type Validator interface {
	Validate() error
}

// ValidationError represents a field validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResponse represents the error response format.
type ValidationResponse struct {
	Error  string            `json:"error"`
	Code   int               `json:"code"`
	Errors []ValidationError `json:"errors"`
}

// ValidateJSON creates middleware that validates JSON request bodies.
// T must implement the Validator interface.
func ValidateJSON[T Validator](next func(w http.ResponseWriter, r *http.Request, body T)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check content type
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			writeValidationError(w, []ValidationError{
				{Field: "content-type", Message: "Content-Type must be application/json"},
			})
			return
		}

		// Parse JSON body
		var body T
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields() // Reject unknown fields

		if err := decoder.Decode(&body); err != nil {
			var validationErrors []ValidationError

			// Handle different types of JSON errors
			switch {
			case strings.Contains(err.Error(), "unknown field"):
				field := extractFieldFromError(err.Error())
				validationErrors = append(validationErrors, ValidationError{
					Field:   field,
					Message: "unknown field",
				})
			case strings.Contains(err.Error(), "invalid character"):
				validationErrors = append(validationErrors, ValidationError{
					Field:   "json",
					Message: "invalid JSON format",
				})
			case strings.Contains(err.Error(), "EOF"):
				validationErrors = append(validationErrors, ValidationError{
					Field:   "body",
					Message: "request body is required",
				})
			default:
				validationErrors = append(validationErrors, ValidationError{
					Field:   "json",
					Message: "failed to parse JSON: " + err.Error(),
				})
			}

			writeValidationError(w, validationErrors)
			return
		}

		// Validate the parsed body
		if err := body.Validate(); err != nil {
			validationErrors := parseValidationError(err)
			writeValidationError(w, validationErrors)
			return
		}

		// Call the next handler with the validated body
		next(w, r, body)
	})
}

// ValidateQueryParams creates middleware that validates query parameters.
func ValidateQueryParams(validator func(*http.Request) []ValidationError) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errors := validator(r)
			if len(errors) > 0 {
				writeValidationError(w, errors)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequiredFields validates that required fields are present in JSON.
func RequiredFields(fields ...string) func(*http.Request) []ValidationError {
	return func(r *http.Request) []ValidationError {
		var errors []ValidationError
		var body map[string]interface{}

		// Parse JSON to check for required fields
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil {
			return []ValidationError{{Field: "body", Message: "invalid JSON"}}
		}

		for _, field := range fields {
			if value, exists := body[field]; !exists {
				errors = append(errors, ValidationError{
					Field:   field,
					Message: "field is required",
				})
			} else if isEmpty(value) {
				errors = append(errors, ValidationError{
					Field:   field,
					Message: "field cannot be empty",
				})
			}
		}

		return errors
	}
}

// parseValidationError converts a validation error into ValidationError slice.
func parseValidationError(err error) []ValidationError {
	var errors []ValidationError
	errorMsg := err.Error()

	// Handle different error formats
	if strings.Contains(errorMsg, ":") {
		// Format: "field: error message"
		parts := strings.SplitN(errorMsg, ":", 2)
		if len(parts) == 2 {
			field := strings.TrimSpace(parts[0])
			message := strings.TrimSpace(parts[1])
			errors = append(errors, ValidationError{
				Field:   field,
				Message: message,
			})
		} else {
			errors = append(errors, ValidationError{
				Field:   "general",
				Message: errorMsg,
			})
		}
	} else {
		// Generic error
		errors = append(errors, ValidationError{
			Field:   "general",
			Message: errorMsg,
		})
	}

	return errors
}

// extractFieldFromError extracts field name from JSON unknown field error.
func extractFieldFromError(errorMsg string) string {
	// Example: "json: unknown field \"invalidField\""
	if strings.Contains(errorMsg, "unknown field") {
		start := strings.Index(errorMsg, `"`)
		if start != -1 {
			end := strings.Index(errorMsg[start+1:], `"`)
			if end != -1 {
				return errorMsg[start+1 : start+1+end]
			}
		}
	}
	return "unknown"
}

// isEmpty checks if a value is considered empty.
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return strings.TrimSpace(v.String()) == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// writeValidationError writes a 422 Unprocessable Entity response with validation errors.
func writeValidationError(w http.ResponseWriter, errors []ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)

	response := ValidationResponse{
		Error:  "validation failed",
		Code:   422,
		Errors: errors,
	}

	json.NewEncoder(w).Encode(response)
}

// ValidateContentType creates middleware that validates request content type.
func ValidateContentType(expectedType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			if !strings.Contains(contentType, expectedType) {
				writeValidationError(w, []ValidationError{
					{
						Field:   "content-type",
						Message: fmt.Sprintf("Content-Type must be %s", expectedType),
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
