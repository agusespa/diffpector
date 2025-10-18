//go:build ignore

package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/agusespa/diffpector/evaluation/mocks/internal/database"
)

type FileHandler struct {
	db        *database.Database
	uploadDir string
}

func NewFileHandler(db *database.Database, uploadDir string) *FileHandler {
	return &FileHandler{
		db:        db,
		uploadDir: uploadDir,
	}
}

func (h *FileHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename == "" {
		http.Error(w, "File parameter required", http.StatusBadRequest)
		return
	}

	// Validate filename format
	if err := h.validateFilename(filename); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(h.uploadDir, filename)
	
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, fullPath)
}

func (h *FileHandler) validateFilename(filename string) error {
	// Check for empty filename
	if strings.TrimSpace(filename) == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	
	// Check for invalid characters
	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range invalidChars {
		if strings.Contains(filename, char) {
			return fmt.Errorf("filename contains invalid character: %s", char)
		}
	}
	
	return nil
}

func (h *FileHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Clean filename to prevent path traversal in uploads
	filename := filepath.Clean(header.Filename)
	if strings.Contains(filename, "..") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Save file logic would go here
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File uploaded successfully"))
}