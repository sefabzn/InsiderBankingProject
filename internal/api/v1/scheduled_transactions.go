// Package v1 provides scheduled transaction endpoints.
package v1

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/api/middleware"
	"github.com/sefa-b/go-banking-sim/internal/domain"
)

// handleScheduleTransaction handles creating a new scheduled transaction.
func (r *Router) handleScheduleTransaction(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)

	finalHandler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Use validation middleware to parse and validate the request
		handler := middleware.ValidateJSON(func(w http.ResponseWriter, req *http.Request, body *domain.ScheduledTransactionRequest) {
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

			scheduledTx, err := r.services.ScheduledTransaction.Create(req.Context(), userID, body)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"` + err.Error() + `","code":400}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)

			response := `{"id":"` + scheduledTx.ID.String() +
				`","user_id":"` + scheduledTx.UserID.String() +
				`","transaction_type":"` + scheduledTx.TransactionType +
				`","amount":` + fmt.Sprintf("%.2f", scheduledTx.Amount) +
				`,"currency":"` + scheduledTx.Currency +
				`","description":"` + scheduledTx.Description +
				`","schedule_type":"` + scheduledTx.ScheduleType +
				`","execute_at":"` + scheduledTx.ExecuteAt.Format(time.RFC3339) +
				`","status":"` + scheduledTx.Status +
				`","is_active":` + strconv.FormatBool(scheduledTx.IsActive) + `,` +
				`"created_at":"` + scheduledTx.CreatedAt.Format(time.RFC3339) + `"}`

			w.Write([]byte(response))
		})

		handler.ServeHTTP(w, req)
	}))

	finalHandler.ServeHTTP(w, req)
}

// handleGetScheduledTransactions handles listing user's scheduled transactions.
func (r *Router) handleGetScheduledTransactions(w http.ResponseWriter, req *http.Request) {
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
		limitStr := req.URL.Query().Get("limit")
		offsetStr := req.URL.Query().Get("offset")
		status := req.URL.Query().Get("status")
		isActiveStr := req.URL.Query().Get("is_active")

		limit := 10 // Default
		offset := 0

		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}

		if offsetStr != "" {
			if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		// Build filter
		filter := &domain.ScheduledTransactionFilter{
			Limit:  limit,
			Offset: offset,
		}

		if status != "" {
			filter.Status = &status
		}

		if isActiveStr != "" {
			if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
				filter.IsActive = &isActive
			}
		}

		scheduledTxs, err := r.services.ScheduledTransaction.List(req.Context(), userID, filter)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to list scheduled transactions","code":500}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := `{"scheduled_transactions":[`
		for i, st := range scheduledTxs {
			if i > 0 {
				response += ","
			}
			response += `{"id":"` + st.ID.String() +
				`","transaction_type":"` + st.TransactionType +
				`","amount":` + fmt.Sprintf("%.2f", st.Amount) +
				`,"currency":"` + st.Currency +
				`","description":"` + st.Description +
				`","schedule_type":"` + st.ScheduleType +
				`","execute_at":"` + st.ExecuteAt.Format(time.RFC3339) +
				`","status":"` + st.Status +
				`","is_active":` + strconv.FormatBool(st.IsActive) + `,` +
				`"created_at":"` + st.CreatedAt.Format(time.RFC3339) + `"}`
		}
		response += `],"limit":` + strconv.Itoa(limit) + `,"offset":` + strconv.Itoa(offset) + `}`

		w.Write([]byte(response))
	}))

	finalHandler.ServeHTTP(w, req)
}

// handleGetScheduledTransaction handles getting a specific scheduled transaction.
func (r *Router) handleGetScheduledTransaction(w http.ResponseWriter, req *http.Request) {
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

		// Extract transaction ID from URL path
		txIDStr := req.PathValue("id")
		if txIDStr == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Scheduled transaction ID is required","code":400}`))
			return
		}

		txID, err := uuid.Parse(txIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Invalid scheduled transaction ID format","code":400}`))
			return
		}

		scheduledTx, err := r.services.ScheduledTransaction.GetByID(req.Context(), txID, userID)
		if err != nil {
			if err.Error() == "access denied: not owner of scheduled transaction" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"Access denied","code":403}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"Scheduled transaction not found","code":404}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := `{"id":"` + scheduledTx.ID.String() +
			`","user_id":"` + scheduledTx.UserID.String() +
			`","transaction_type":"` + scheduledTx.TransactionType +
			`","amount":` + fmt.Sprintf("%.2f", scheduledTx.Amount) +
			`,"currency":"` + scheduledTx.Currency +
			`","description":"` + scheduledTx.Description +
			`","schedule_type":"` + scheduledTx.ScheduleType +
			`","execute_at":"` + scheduledTx.ExecuteAt.Format(time.RFC3339) +
			`","status":"` + scheduledTx.Status +
			`","is_active":` + strconv.FormatBool(scheduledTx.IsActive) + `,` +
			`"created_at":"` + scheduledTx.CreatedAt.Format(time.RFC3339) + `"}`

		w.Write([]byte(response))
	}))

	finalHandler.ServeHTTP(w, req)
}

// handleCancelScheduledTransaction handles canceling a scheduled transaction.
func (r *Router) handleCancelScheduledTransaction(w http.ResponseWriter, req *http.Request) {
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

		// Extract transaction ID from URL path
		txIDStr := req.PathValue("id")
		if txIDStr == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Scheduled transaction ID is required","code":400}`))
			return
		}

		txID, err := uuid.Parse(txIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Invalid scheduled transaction ID format","code":400}`))
			return
		}

		err = r.services.ScheduledTransaction.Cancel(req.Context(), txID, userID)
		if err != nil {
			if err.Error() == "access denied: not owner of scheduled transaction" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"Access denied","code":403}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"Scheduled transaction not found","code":404}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Scheduled transaction cancelled successfully"}`))
	}))

	finalHandler.ServeHTTP(w, req)
}
