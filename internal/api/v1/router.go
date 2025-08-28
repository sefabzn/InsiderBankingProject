// Package v1 provides version 1 of the HTTP API.
package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/sefa-b/go-banking-sim/internal/api/middleware"
	"github.com/sefa-b/go-banking-sim/internal/auth"
	"github.com/sefa-b/go-banking-sim/internal/domain"
	"github.com/sefa-b/go-banking-sim/internal/repository"
	"github.com/sefa-b/go-banking-sim/internal/service"
	"github.com/sefa-b/go-banking-sim/internal/utils"
)

// Router holds the dependencies needed for v1 API routes.
type Router struct {
	repos      *repository.Repositories
	services   *service.Services
	jwtManager *auth.JWTManager
}

// NewRouter creates a new v1 API router.
func NewRouter(repos *repository.Repositories, services *service.Services, jwtManager *auth.JWTManager) *Router {
	return &Router{
		repos:      repos,
		services:   services,
		jwtManager: jwtManager,
	}
}

// RegisterRoutes registers all v1 API routes on the provided mux.
func (r *Router) RegisterRoutes(mux *http.ServeMux) {
	// Health/ping endpoint
	mux.HandleFunc("GET /api/v1/ping", r.handlePing)

	// Test endpoint to retrieve all users (no validation)
	mux.HandleFunc("GET /api/v1/test/users", r.handleTestGetAllUsers)

	// Auth routes
	mux.HandleFunc("POST /api/v1/auth/register", r.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", r.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", r.handleRefresh)

	// User routes (admin only)
	mux.HandleFunc("GET /api/v1/users", r.handleListUsers)
	mux.HandleFunc("GET /api/v1/users/{id}", r.handleGetUser)
	mux.HandleFunc("PUT /api/v1/users/{id}", r.handleUpdateUser)
	mux.HandleFunc("DELETE /api/v1/users/{id}", r.handleDeleteUser)

	// Balance routes
	mux.HandleFunc("GET /api/v1/balances/current", r.handleGetCurrentBalance)
	mux.HandleFunc("GET /api/v1/balances/historical", r.handleGetHistoricalBalance)
	mux.HandleFunc("GET /api/v1/balances/at-time", r.handleGetBalanceAtTime)

	// Scheduled transaction routes (avoid conflict with transaction routes)
	mux.HandleFunc("POST /api/v1/scheduled-transactions", r.handleScheduleTransaction)
	mux.HandleFunc("GET /api/v1/scheduled-transactions", r.handleGetScheduledTransactions)
	mux.HandleFunc("GET /api/v1/scheduled-transactions/{id}", r.handleGetScheduledTransaction)
	mux.HandleFunc("DELETE /api/v1/scheduled-transactions/{id}", r.handleCancelScheduledTransaction)

	// Transaction routes
	mux.HandleFunc("POST /api/v1/transactions/credit", r.handleCredit)
	mux.HandleFunc("POST /api/v1/transactions/debit", r.handleDebit)
	mux.HandleFunc("POST /api/v1/transactions/transfer", r.handleTransfer)
	mux.HandleFunc("POST /api/v1/transactions/{id}/rollback", r.handleRollbackTransaction)
	mux.HandleFunc("GET /api/v1/transactions/{id}", r.handleGetTransaction)
	mux.HandleFunc("GET /api/v1/transactions/history", r.handleGetTransactionHistory)
}

// handlePing responds to ping requests for testing connectivity.
func (r *Router) handlePing(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"pong"}`))
}

// handleRegister handles user registration.
func (r *Router) handleRegister(w http.ResponseWriter, req *http.Request) {
	utils.Info("handleRegister called", "method", req.Method, "path", req.URL.Path)
	// Use validation middleware to parse and validate the request
	handler := middleware.ValidateJSON(func(w http.ResponseWriter, req *http.Request, body *domain.CreateUserRequest) {
		utils.Info("registration validation passed", "username", body.Username, "email", body.Email)
		// Call the auth service to register the user
		userResponse, err := r.services.Auth.Register(req.Context(), body)
		if err != nil {
			// Check for specific error types to return appropriate status codes
			switch {
			case err.Error() == "email already registered":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error":"Email already registered","code":409}`))
				return
			case err.Error() == "username already taken":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error":"Username already taken","code":409}`))
				return
			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Registration failed","code":400}`))
				return
			}
		}

		// Return 201 Created with user data (no tokens per requirement)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		// Convert to JSON manually for precise control
		response := `{"id":"` + userResponse.ID.String() +
			`","username":"` + userResponse.Username +
			`","email":"` + userResponse.Email +
			`","role":"` + userResponse.Role +
			`","created_at":"` + userResponse.CreatedAt.Format("2006-01-02T15:04:05Z07:00") +
			`","updated_at":"` + userResponse.UpdatedAt.Format("2006-01-02T15:04:05Z07:00") +
			`","is_active":` + strconv.FormatBool(userResponse.IsActive) + `}`

		w.Write([]byte(response))
	})

	handler.ServeHTTP(w, req)
}

// handleLogin handles user login.
func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	// Use validation middleware to parse and validate the request
	handler := middleware.ValidateJSON(func(w http.ResponseWriter, req *http.Request, body *domain.LoginRequest) {
		// Call the auth service to login the user
		loginResponse, err := r.services.Auth.Login(req.Context(), body.Email, body.Password)
		if err != nil {
			// Return 401 for authentication failures
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Invalid email or password","code":401}`))
			return
		}

		// Return 200 OK with user data and tokens
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Convert to JSON manually for precise control
		userJson := `{"id":"` + loginResponse.User.ID.String() +
			`","username":"` + loginResponse.User.Username +
			`","email":"` + loginResponse.User.Email +
			`","role":"` + loginResponse.User.Role +
			`","created_at":"` + loginResponse.User.CreatedAt.Format("2006-01-02T15:04:05Z07:00") +
			`","updated_at":"` + loginResponse.User.UpdatedAt.Format("2006-01-02T15:04:05Z07:00") +
			`","is_active":` + strconv.FormatBool(loginResponse.User.IsActive) + `}`

		response := `{"user":` + userJson +
			`,"access_token":"` + loginResponse.AccessToken +
			`","refresh_token":"` + loginResponse.RefreshToken +
			`","expires_in":` + fmt.Sprintf("%d", loginResponse.ExpiresIn) + `}`

		w.Write([]byte(response))
	})

	handler.ServeHTTP(w, req)
}

// handleListUsers handles listing users with pagination (admin only).
func (r *Router) handleListUsers(w http.ResponseWriter, req *http.Request) {
	// Apply authentication and admin authorization middleware
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)
	adminMiddleware := middleware.RequireAdmin

	// Chain middlewares: auth -> admin -> handler
	finalHandler := authMiddleware(adminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Parse query parameters
		limitStr := req.URL.Query().Get("limit")
		offsetStr := req.URL.Query().Get("offset")

		limit := 0  // Default (no limit)
		offset := 0 // Default

		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit >= 0 {
				limit = parsedLimit
			} else if parsedLimit < 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Limit must be non-negative","code":400}`))
				return
			}
		}

		if offsetStr != "" {
			if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			} else if parsedOffset < 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Offset must be non-negative","code":400}`))
				return
			}
		}

		// Call the user service to list users
		users, err := r.services.User.List(req.Context(), limit, offset)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to list users","code":500}`))
			return
		}

		// Return 200 OK with users list
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Build JSON response manually
		response := `{"users":[`
		for i, user := range users {
			if i > 0 {
				response += ","
			}
			response += `{"id":"` + user.ID.String() +
				`","username":"` + user.Username +
				`","email":"` + user.Email +
				`","role":"` + user.Role +
				`","created_at":"` + user.CreatedAt.Format("2006-01-02T15:04:05Z07:00") +
				`","updated_at":"` + user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00") +
				`","is_active":` + strconv.FormatBool(user.IsActive) + `}`
		}
		response += `],"limit":` + strconv.Itoa(limit) + `,"offset":` + strconv.Itoa(offset) + `}`

		w.Write([]byte(response))
	})))

	finalHandler.ServeHTTP(w, req)
}

func (r *Router) handleRefresh(w http.ResponseWriter, req *http.Request) {
	// Use validation middleware to parse and validate the request
	handler := middleware.ValidateJSON(func(w http.ResponseWriter, req *http.Request, body *domain.RefreshRequest) {
		// Call the auth service to refresh the token
		tokenResponse, err := r.services.Auth.RefreshToken(req.Context(), body.RefreshToken)
		if err != nil {
			// Return 401 for invalid refresh tokens
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Invalid refresh token","code":401}`))
			return
		}

		// Return 200 OK with new access token
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Convert to JSON manually for precise control
		response := `{"access_token":"` + tokenResponse.AccessToken +
			`","expires_in":` + fmt.Sprintf("%d", tokenResponse.ExpiresIn) + `}`

		w.Write([]byte(response))
	})

	handler.ServeHTTP(w, req)
}

// handleTestGetAllUsers handles retrieving all users for testing (no validation).
func (r *Router) handleTestGetAllUsers(w http.ResponseWriter, req *http.Request) {
	// Call the repository directly to get all users
	users, err := r.repos.Users.ListAll(req.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve users","code":500}`))
		return
	}

	// Return 200 OK with users list
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Build JSON response manually
	response := `{"users":[`
	for i, user := range users {
		if i > 0 {
			response += ","
		}
		response += `{"id":"` + user.ID.String() +
			`","username":"` + user.Username +
			`","email":"` + user.Email +
			`","role":"` + user.Role +
			`","created_at":"` + user.CreatedAt.Format("2006-01-02T15:04:05Z07:00") +
			`","updated_at":"` + user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00") +
			`","is_active":` + strconv.FormatBool(user.IsActive) + `}`
	}
	response += `],"total":` + fmt.Sprintf("%d", len(users)) + `}`

	w.Write([]byte(response))
}
