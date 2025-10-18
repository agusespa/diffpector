//go:build ignore

package handler

import (
	"net/http"

	"github.com/agusespa/diffpector/evaluation/mocks/internal/database"
)

func SearchUsersHandler(db *database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		users, err := db.SearchUsers(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Process users (simplified for mock)
		_ = users
		w.WriteHeader(http.StatusOK)
	}
}
