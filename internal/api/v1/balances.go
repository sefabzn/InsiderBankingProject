package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/api/middleware"
	"github.com/sefa-b/go-banking-sim/internal/domain"
)

// handleGetCurrentBalance handles getting the current user's balance.
func (r *Router) handleGetCurrentBalance(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)

	finalHandler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context
		userIDStr, ok := middleware.GetCurrentUserID(req)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"User not authenticated","code":401}`))
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Invalid user ID","code":500}`))
			return
		}

		// Get the user's current balance
		balance, err := r.services.Balance.GetCurrent(req.Context(), userID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to get balance","code":500}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Ensure currency is valid, default to USD if empty
		currency := balance.Currency
		if currency == "" {
			currency = "USD"
		}

		// Use proper JSON marshaling for better reliability
		response := map[string]interface{}{
			"user_id":         balance.UserID.String(),
			"amount":          balance.Amount,
			"currency":        currency,
			"last_updated_at": balance.LastUpdatedAt.Format(time.RFC3339),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to encode response","code":500}`))
			return
		}
	}))

	finalHandler.ServeHTTP(w, req)
}

// handleGetHistoricalBalance handles getting historical balance snapshots.
func (r *Router) handleGetHistoricalBalance(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)

	finalHandler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context
		userIDStr, ok := middleware.GetCurrentUserID(req)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"User not authenticated","code":401}`))
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Invalid user ID","code":500}`))
			return
		}

		// Parse limit query parameter
		limitStr := req.URL.Query().Get("limit")
		limit := 10 // Default limit
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}

		// Get historical balance snapshots
		history, err := r.services.Balance.GetHistorical(req.Context(), userID, limit)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to get balance history","code":500}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Build JSON response manually
		response := `{"history":[`
		for i, item := range history {
			if i > 0 {
				response += ","
			}
			response += `{"user_id":"` + item.UserID.String() +
				`","amount":` + fmt.Sprintf("%.2f", item.Amount) +
				`,"timestamp":"` + item.Timestamp.Format(time.RFC3339) +
				`","reason":"` + item.Reason + `"}`
		}
		response += `],"limit":` + strconv.Itoa(limit) + `}`

		w.Write([]byte(response))
	}))

	finalHandler.ServeHTTP(w, req)
}

// handleGetBalanceAtTime handles getting the user's balance at a specific time.
func (r *Router) handleGetBalanceAtTime(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)

	finalHandler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context
		userIDStr, ok := middleware.GetCurrentUserID(req)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"User not authenticated","code":401}`))
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Invalid user ID","code":500}`))
			return
		}

		// Parse timestamp query parameter
		timestampStr := req.URL.Query().Get("timestamp")

		if timestampStr == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Timestamp parameter is required","code":400}`))
			return
		}

		//use repository to get at time
		balance, err := r.services.Balance.GetAtTime(req.Context(), userID, timestampStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf(`{"error":"Failed to get balance at time: %s","code":500}`, err.Error())))
			return
		}
		//return the balance
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := fmt.Sprintf(`{"user_id":"%s","amount":%.2f,"timestamp":"%s","reason":"%s"}`,
			balance.UserID.String(),
			balance.Amount,
			balance.LastUpdatedAt.Format(time.RFC3339), "showing current balance at the requested time")
		w.Write([]byte(response))
	}))

	finalHandler.ServeHTTP(w, req)
}

// Helper functions for JSON parsing and UUID formatting
func parseJSONBody(req *http.Request, v interface{}) error {
	if req.Body == nil {
		return fmt.Errorf("empty request body")
	}
	defer req.Body.Close()

	decoder := json.NewDecoder(req.Body)
	return decoder.Decode(v)
}

func formatUUID(uuidPtr *uuid.UUID) string {
	if uuidPtr == nil {
		return "null"
	}
	return `"` + uuidPtr.String() + `"`
}

// handleCredit handles crediting money to a user's account.
func (r *Router) handleCredit(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)

	finalHandler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context
		userIDStr, ok := middleware.GetCurrentUserID(req)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"User not authenticated","code":401}`))
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Invalid user ID","code":500}`))
			return
		}

		// Parse request body
		var creditReq domain.CreditRequest
		if err := parseJSONBody(req, &creditReq); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Invalid JSON request body","code":400}`))
			return
		}

		// Process the credit transaction
		transaction, err := r.services.Transaction.Credit(req.Context(), userID, &creditReq)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"` + err.Error() + `","code":400}`))
			return
		}

		// Return 201 Created with transaction details
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		response := `{"id":"` + transaction.ID.String() +
			`","from_user_id":` + formatUUID(transaction.FromUserID) +
			`,"to_user_id":` + formatUUID(transaction.ToUserID) +
			`,"amount":` + fmt.Sprintf("%.2f", transaction.Amount) +
			`,"currency":"` + transaction.Currency + `","type":"` + transaction.Type +
			`","status":"` + transaction.Status +
			`","created_at":"` + transaction.CreatedAt.Format("2006-01-02T15:04:05Z07:00") + `"}`

		w.Write([]byte(response))
	}))

	finalHandler.ServeHTTP(w, req)
}

// handleDebit handles debiting money from a user's account.
func (r *Router) handleDebit(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)

	finalHandler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context
		userIDStr, ok := middleware.GetCurrentUserID(req)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"User not authenticated","code":401}`))
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Invalid user ID","code":500}`))
			return
		}

		// Parse request body
		var debitReq domain.DebitRequest
		if err := parseJSONBody(req, &debitReq); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Invalid JSON request body","code":400}`))
			return
		}

		// Process the debit transaction
		transaction, err := r.services.Transaction.Debit(req.Context(), userID, &debitReq)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"` + err.Error() + `","code":400}`))
			return
		}

		// Return 201 Created with transaction details
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		response := `{"id":"` + transaction.ID.String() +
			`","from_user_id":` + formatUUID(transaction.FromUserID) +
			`,"to_user_id":` + formatUUID(transaction.ToUserID) +
			`,"amount":` + fmt.Sprintf("%.2f", transaction.Amount) +
			`,"currency":"` + transaction.Currency + `","type":"` + transaction.Type +
			`","status":"` + transaction.Status +
			`","created_at":"` + transaction.CreatedAt.Format("2006-01-02T15:04:05Z07:00") + `"}`

		w.Write([]byte(response))
	}))

	finalHandler.ServeHTTP(w, req)
}

// handleTransfer handles transferring money between users.
func (r *Router) handleTransfer(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)

	finalHandler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context
		userIDStr, ok := middleware.GetCurrentUserID(req)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"User not authenticated","code":401}`))
			return
		}

		fromUserID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Invalid user ID","code":500}`))
			return
		}

		// Parse request body
		var transferReq domain.TransferRequest
		if err := parseJSONBody(req, &transferReq); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Invalid JSON request body","code":400}`))
			return
		}

		// Process the transfer transaction
		transaction, err := r.services.Transaction.Transfer(req.Context(), fromUserID, &transferReq)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"` + err.Error() + `","code":400}`))
			return
		}

		// Return 201 Created with transaction details
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		response := `{"id":"` + transaction.ID.String() +
			`","from_user_id":` + formatUUID(transaction.FromUserID) +
			`,"to_user_id":` + formatUUID(transaction.ToUserID) +
			`,"amount":` + fmt.Sprintf("%.2f", transaction.Amount) +
			`,"currency":"` + transaction.Currency + `","type":"` + transaction.Type +
			`","status":"` + transaction.Status +
			`","created_at":"` + transaction.CreatedAt.Format("2006-01-02T15:04:05Z07:00") + `"}`

		w.Write([]byte(response))
	}))

	finalHandler.ServeHTTP(w, req)
}

// handleGetTransaction handles retrieving a specific transaction by ID.
func (r *Router) handleGetTransaction(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)

	finalHandler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context
		userIDStr, ok := middleware.GetCurrentUserID(req)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"User not authenticated","code":401}`))
			return
		}

		requestingUserID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Invalid user ID","code":500}`))
			return
		}

		// Extract transaction ID from URL path
		path := req.URL.Path
		// Path format: /api/v1/transactions/{id}
		pathParts := strings.Split(path, "/")
		if len(pathParts) < 5 || pathParts[4] == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Transaction ID is required","code":400}`))
			return
		}

		transactionIDStr := pathParts[4]
		transactionID, err := uuid.Parse(transactionIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Invalid transaction ID format","code":400}`))
			return
		}

		// Get the transaction with authorization check
		transaction, err := r.services.Transaction.GetByID(req.Context(), transactionID, requestingUserID)
		if err != nil {
			// Check if it's an access denied error
			if err.Error() == "access denied: you don't have permission to view this transaction" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"Access denied: you don't have permission to view this transaction","code":403}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"Transaction not found","code":404}`))
			return
		}

		// Return 200 OK with transaction details
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Use proper JSON marshaling to ensure valid JSON
		jsonResponse, err := json.Marshal(transaction)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to marshal transaction response","code":500}`))
			return
		}

		w.Write(jsonResponse)
	}))

	finalHandler.ServeHTTP(w, req)
}

// handleGetTransactionHistory handles retrieving transaction history for the authenticated user.
func (r *Router) handleGetTransactionHistory(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)

	finalHandler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context
		userIDStr, ok := middleware.GetCurrentUserID(req)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"User not authenticated","code":401}`))
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Invalid user ID","code":500}`))
			return
		}

		// Parse query parameters
		filter := &domain.TransactionFilter{}

		// Parse limit parameter
		if limitStr := req.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
				filter.Limit = limit
			} else if limit <= 0 || limit > 100 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Limit must be between 1 and 100","code":400}`))
				return
			}
		}

		// Parse offset parameter
		if offsetStr := req.URL.Query().Get("offset"); offsetStr != "" {
			if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
				filter.Offset = offset
			} else if offset < 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Offset must be non-negative","code":400}`))
				return
			}
		}

		// Parse type parameter
		if typeStr := req.URL.Query().Get("type"); typeStr != "" {
			switch typeStr {
			case "credit":
				transactionType := domain.TypeCredit
				filter.Type = &transactionType
			case "debit":
				transactionType := domain.TypeDebit
				filter.Type = &transactionType
			case "transfer":
				transactionType := domain.TypeTransfer
				filter.Type = &transactionType
			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Invalid type. Must be 'credit', 'debit', or 'transfer'","code":400}`))
				return
			}
		}

		// Parse status parameter
		if statusStr := req.URL.Query().Get("status"); statusStr != "" {
			switch statusStr {
			case "pending":
				transactionStatus := domain.StatusPending
				filter.Status = &transactionStatus
			case "success":
				transactionStatus := domain.StatusSuccess
				filter.Status = &transactionStatus
			case "failed":
				transactionStatus := domain.StatusFailed
				filter.Status = &transactionStatus
			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Invalid status. Must be 'pending', 'success', or 'failed'","code":400}`))
				return
			}
		}

		// Parse since parameter (RFC3339 timestamp)
		if sinceStr := req.URL.Query().Get("since"); sinceStr != "" {
			if sinceTime, err := time.Parse(time.RFC3339, sinceStr); err == nil {
				filter.Since = &sinceTime
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Invalid since parameter. Must be RFC3339 timestamp","code":400}`))
				return
			}
		}

		// Get transaction history
		transactions, err := r.services.Transaction.GetHistory(req.Context(), userID, filter)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to get transaction history","code":500}`))
			return
		}

		// Return 200 OK with transaction history
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Build JSON response using proper marshaling
		type TransactionHistoryResponse struct {
			Transactions []*domain.TransactionResponse `json:"transactions"`
			Limit        int                           `json:"limit"`
			Offset       int                           `json:"offset"`
		}

		// Convert []*domain.TransactionResponse to []*domain.TransactionResponse for JSON marshaling
		txResponses := make([]*domain.TransactionResponse, len(transactions))
		copy(txResponses, transactions)

		responseData := TransactionHistoryResponse{
			Transactions: txResponses,
			Limit:        filter.Limit,
			Offset:       filter.Offset,
		}

		jsonResponse, err := json.Marshal(responseData)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to marshal response","code":500}`))
			return
		}

		w.Write(jsonResponse)
	}))

	finalHandler.ServeHTTP(w, req)
}

// handleRollbackTransaction handles rolling back a completed transaction.
func (r *Router) handleRollbackTransaction(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)

	finalHandler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get user ID from context
		userIDStr, ok := middleware.GetCurrentUserID(req)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"User not authenticated","code":401}`))
			return
		}

		requestingUserID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Invalid user ID","code":500}`))
			return
		}

		// Extract transaction ID from URL path
		path := req.URL.Path
		// Path format: /api/v1/transactions/{id}/rollback
		pathParts := strings.Split(path, "/")
		if len(pathParts) < 6 || pathParts[4] == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Transaction ID is required","code":400}`))
			return
		}

		transactionIDStr := pathParts[4]
		transactionID, err := uuid.Parse(transactionIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Invalid transaction ID format","code":400}`))
			return
		}

		// Check if user is admin
		isAdmin := middleware.IsAdmin(req)

		// Process the rollback transaction
		var transaction *domain.TransactionResponse

		if isAdmin {
			// Admin can rollback any transaction
			transaction, err = r.services.Transaction.RollbackByAdmin(req.Context(), transactionID)
		} else {
			// Regular user can only rollback their own transactions
			transaction, err = r.services.Transaction.Rollback(req.Context(), transactionID, requestingUserID)
		}

		if err != nil {
			// Check for specific error types
			switch {
			case err.Error() == "access denied: you don't have permission to rollback this transaction":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"Access denied: you don't have permission to rollback this transaction","code":403}`))
				return
			case err.Error() == "can only rollback completed transactions":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Can only rollback completed transactions","code":400}`))
				return
			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"` + err.Error() + `","code":400}`))
				return
			}
		}

		// Return 201 Created with rollback transaction details
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		response := `{"id":"` + transaction.ID.String() +
			`","from_user_id":` + formatUUID(transaction.FromUserID) +
			`,"to_user_id":` + formatUUID(transaction.ToUserID) +
			`,"amount":` + fmt.Sprintf("%.2f", transaction.Amount) +
			`,"currency":"` + transaction.Currency + `","type":"` + transaction.Type +
			`","status":"` + transaction.Status +
			`","created_at":"` + transaction.CreatedAt.Format("2006-01-02T15:04:05Z07:00") + `"}`

		w.Write([]byte(response))
	}))

	finalHandler.ServeHTTP(w, req)
}
