//go:build ignore

package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"internal/tools/integration_tests/code_samples/go/utils"
)

// UserService defines the necessary methods from the core service.
type UserService interface {
	GetUser(ctx context.Context, id string) (*utils.User, error)
}

// UserHandler handles HTTP requests related to users.
type UserHandler struct {
	service UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc UserService) *UserHandler {
	return &UserHandler{service: svc}
}

// HandleGetUser handles GET /users/{id} requests.
func (h *UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user, err := h.service.GetUser(ctx, userID)

	if err != nil {
		log.Printf("Handler failed to fulfill request for user %s: %v", userID, err)
		http.Error(w, "Internal server error retrieving user details", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
