//go:build ignore

package handler

import (
	"fmt"
	"html"
	"net/http"
	"strconv"

	"github.com/agusespa/diffpector/evaluation/mocks/internal/database"
)

type ProfileHandler struct {
	db *database.Database
}

func NewProfileHandler(db *database.Database) *ProfileHandler {
	return &ProfileHandler{db: db}
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	bio := r.FormValue("bio")
	
	user := &database.User{
		ID:   userID,
		Name: r.FormValue("name"),
		Bio:  bio,
	}

	err = h.db.UpdateUserProfile(user)
	if err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<h1>Profile Updated</h1><p>Bio: %s</p>", bio)
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUserProfile(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<h1>%s's Profile</h1><p>Bio: %s</p>", user.Name, user.Bio)
}