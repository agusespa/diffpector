//go:build ignore

package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/agusespa/diffpector/evaluation/mocks/internal/auth"
	"github.com/agusespa/diffpector/evaluation/mocks/internal/database"
	"github.com/agusespa/diffpector/evaluation/mocks/internal/handler"
	_ "github.com/lib/pq"
)

func main() {
	// Initialize database connection
	db, err := sql.Open("postgres", "postgres://user:password@localhost/myapp?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize database layer
	database := database.NewDatabase(db)

	// Initialize services
	authService := auth.NewAuthService(database)

	// Initialize handlers
	profileHandler := handler.NewProfileHandler(database)
	userHandler := handler.SearchUsersHandler(database)
	fileHandler := handler.NewFileHandler(database, "uploads/")
	adminHandler := handler.NewAdminHandler(database, authService)

	// Setup routes
	http.HandleFunc("/profile/update", profileHandler.UpdateProfile)
	http.HandleFunc("/profile", profileHandler.GetProfile)
	http.HandleFunc("/users/search", userHandler)
	http.HandleFunc("/files/download", fileHandler.ServeFile)
	http.HandleFunc("/files/upload", fileHandler.UploadFile)
	http.HandleFunc("/admin/users", adminHandler.ListUsers)
	http.HandleFunc("/admin/users/delete", adminHandler.DeleteUser)

	// Start server
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}