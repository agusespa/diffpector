//go:build ignore

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type User struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Profile *Profile `json:"profile"`
}

type Profile struct {
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
}

type UserService interface {
	GetByID(id int) (*User, error)
}

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUser handles user retrieval with proper error handling (before state)
// This is the SAFE implementation that checks for both errors and nil users
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "missing id parameter", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id parameter", http.StatusBadRequest)
		return
	}

	// IMPORTANT: This service call can return (nil, nil) for non-existent users
	user, err := h.userService.GetByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
