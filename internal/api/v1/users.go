package v1

import (
	"net/http"

	"strconv"

	"github.com/google/uuid"
	"github.com/sefa-b/go-banking-sim/internal/api/middleware"
	"github.com/sefa-b/go-banking-sim/internal/domain"
)

// handleGetUser handles getting a specific user by ID (admin only).
func (r *Router) handleGetUser(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)
	adminMiddleware := middleware.RequireAdmin

	finalHandler := authMiddleware(adminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Extract user ID from URL path
		userIDStr := req.PathValue("id")
		if userIDStr == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"User ID is required","code":400}`))
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Invalid user ID format","code":400}`))
			return
		}

		user, err := r.services.User.GetByID(req.Context(), userID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"User not found","code":404}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := `{"id":"` + user.ID.String() +
			`","username":"` + user.Username +
			`","email":"` + user.Email +
			`","role":"` + user.Role +
			`","created_at":"` + user.CreatedAt.Format("2006-01-02T15:04:05Z07:00") +
			`","updated_at":"` + user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00") + `"}`

		w.Write([]byte(response))
	})))

	finalHandler.ServeHTTP(w, req)
}

// handleUpdateUser handles updating a user (admin only).
func (r *Router) handleUpdateUser(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)
	adminMiddleware := middleware.RequireAdmin

	finalHandler := authMiddleware(adminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Extract user ID from URL path
		userIDStr := req.PathValue("id")
		if userIDStr == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"User ID is required","code":400}`))
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Invalid user ID format","code":400}`))
			return
		}

		// Parse and validate request body
		handler := middleware.ValidateJSON(func(w http.ResponseWriter, req *http.Request, body *domain.UpdateUserRequest) {
			user, err := r.services.User.Update(req.Context(), userID, body)
			if err != nil {
				if err.Error() == "failed to get user: user not found" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(`{"error":"User not found","code":404}`))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Failed to update user","code":400}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			response := `{"id":"` + user.ID.String() +
				`","username":"` + user.Username +
				`","email":"` + user.Email +
				`","role":"` + user.Role +
				`","created_at":"` + user.CreatedAt.Format("2006-01-02T15:04:05Z07:00") +
				`","updated_at":"` + user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00") +
				`","is_active":` + strconv.FormatBool(user.IsActive) + `}`

			w.Write([]byte(response))
		})

		handler.ServeHTTP(w, req)
	})))

	finalHandler.ServeHTTP(w, req)
}

// handleDeleteUser handles deleting a user (admin only).
func (r *Router) handleDeleteUser(w http.ResponseWriter, req *http.Request) {
	authMiddleware := middleware.AuthMiddleware(r.jwtManager)
	adminMiddleware := middleware.RequireAdmin

	finalHandler := authMiddleware(adminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Extract user ID from URL path
		userIDStr := req.PathValue("id")
		if userIDStr == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"User ID is required","code":400}`))
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Invalid user ID format","code":400}`))
			return
		}

		err = r.services.User.Delete(req.Context(), userID)
		if err != nil {
			switch err.Error() {
			case "failed to get user: user not found", "failed to delete user: user not found", "user not found: user not found or already inactive":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":"User not found","code":404}`))
				return
			case "user cannot be deleted: associated transactions exist":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict) // 409 Conflict
				w.Write([]byte(`{"error":"User cannot be deleted: associated transactions exist","code":409}`))
				return
			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"Failed to delete user : ` + err.Error() + `","code":500}`))
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"User is deleted"}`))
	})))

	finalHandler.ServeHTTP(w, req)
}
