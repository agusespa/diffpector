//go:build ignore

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/agusespa/diffpector/evaluation/mocks/internal/auth"
	"github.com/agusespa/diffpector/evaluation/mocks/internal/database"
)

type AdminHandler struct {
	db   *database.Database
	auth *auth.AuthService
}

func NewAdminHandler(db *database.Database, authService *auth.AuthService) *AdminHandler {
	return &AdminHandler{
		db:   db,
		auth: authService,
	}
}

// DeleteUser allows admins to delete user accounts
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current user ID from session/token (simplified)
	currentUserID, err := strconv.Atoi(r.Header.Get("X-User-ID"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get target user ID to delete
	targetUserID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil {
		http.Error(w, "Invalid target user ID", http.StatusBadRequest)
		return
	}

	// Check if current user has admin permissions
	// This is the vulnerable part - it assumes CheckUserPermission returns an error for unauthorized users
	permission, err := h.auth.CheckUserPermission(currentUserID, "users")
	if err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// The vulnerability: this code assumes that if no error occurred, the user has admin access
	// But CheckUserPermission might return PermissionRead (1) or PermissionWrite (2) without error
	// Only PermissionAdmin (3) should be allowed for user deletion

	// Delete the user
	err = h.db.DeleteUser(targetUserID)
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "user deleted"})
}

// ListUsers allows users to view user list based on their permission level
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	currentUserID, err := strconv.Atoi(r.Header.Get("X-User-ID"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Proper permission check using ValidateAccess
	err = h.auth.ValidateAccess(currentUserID, "users", auth.PermissionRead)
	if err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	users, err := h.db.GetAllUsers()
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}