package agent

import (
	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/pkg/config"
	"errors"
	"strings"
	"testing"
)

type mockReadFileTool struct {
	files map[string]string
	err   error
}

func (m *mockReadFileTool) Name() string {
	return "read_file"
}

func (m *mockReadFileTool) Description() string {
	return "Mock read file tool for testing"
}

func (m *mockReadFileTool) Execute(args map[string]any) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	filename, ok := args["filename"].(string)
	if !ok {
		return "", errors.New("filename required")
	}

	if content, exists := m.files[filename]; exists {
		return content, nil
	}

	return "", errors.New("file not found")
}

type mockSymbolContextTool struct {
	response string
	err      error
	lastArgs map[string]any
}

func (m *mockSymbolContextTool) Name() string {
	return "symbol_context"
}

func (m *mockSymbolContextTool) Description() string {
	return "Mock symbol context tool for testing"
}

func (m *mockSymbolContextTool) Execute(args map[string]any) (string, error) {
	m.lastArgs = args
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestGatherEnhancedContext_Success(t *testing.T) {
	mockReadTool := &mockReadFileTool{
		files: map[string]string{
			"main.go": `package main

func main() {
	fmt.Println("Hello, World!")
}`,
			"utils.go": `package main

func Helper() string {
	return "helper"
}`,
		},
	}

	mockSymbolTool := &mockSymbolContextTool{
		response: "=== CONTEXT AND SYMBOL ANALYSIS ===\nSymbol: main (function)\nLocation: main.go:3\nFound 0 usage(s)\n",
	}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)
	registry.Register("symbol_context", mockSymbolTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
 
+import "fmt"
 func main() {`

	changedFiles := []string{"main.go", "utils.go"}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if context == nil {
		t.Fatal("Expected context to be non-nil")
	}

	// Check basic structure
	if context.Diff != diff {
		t.Errorf("Expected diff to be preserved, got: %s", context.Diff)
	}

	if len(context.ChangedFiles) != 2 {
		t.Errorf("Expected 2 changed files, got: %d", len(context.ChangedFiles))
	}

	// Check file contents were read
	if len(context.FileContents) != 2 {
		t.Errorf("Expected 2 file contents, got: %d", len(context.FileContents))
	}

	expectedContent := `package main

func main() {
	fmt.Println("Hello, World!")
}`
	if context.FileContents["main.go"] != expectedContent {
		t.Errorf("Expected main.go content to match, got: %s", context.FileContents["main.go"])
	}

	// Check symbol analysis was performed
	if context.SymbolAnalysis == "" {
		t.Error("Expected symbol analysis to be populated")
	}

	// Check that symbol context tool was called with correct parameters
	if mockSymbolTool.lastArgs == nil {
		t.Fatal("Expected symbol context tool to be called")
	}

	if mockSymbolTool.lastArgs["diff"] != diff {
		t.Error("Expected diff to be passed to symbol context tool")
	}

	changedFilesArg, ok := mockSymbolTool.lastArgs["changed_files"].([]string)
	if !ok {
		t.Fatal("Expected changed_files to be passed as []string")
	}

	if len(changedFilesArg) != 2 {
		t.Errorf("Expected 2 changed files in symbol context args, got: %d", len(changedFilesArg))
	}

	fileContentsArg, ok := mockSymbolTool.lastArgs["file_contents"].(map[string]string)
	if !ok {
		t.Fatal("Expected file_contents to be passed as map[string]string")
	}

	if len(fileContentsArg) != 2 {
		t.Errorf("Expected 2 file contents in symbol context args, got: %d", len(fileContentsArg))
	}
}

func TestGatherEnhancedContext_MissingReadTool(t *testing.T) {
	registry := tools.NewRegistry()

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := "test diff"
	changedFiles := []string{"main.go"}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	if err == nil {
		t.Fatal("Expected error for missing read_file tool")
	}

	if context != nil {
		t.Error("Expected context to be nil on error")
	}

	expectedError := "read_file tool not available"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: '%s'", expectedError, err.Error())
	}
}

func TestGatherEnhancedContext_ReadFileError(t *testing.T) {
	mockReadTool := &mockReadFileTool{
		err: errors.New("permission denied"),
	}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := "test diff"
	changedFiles := []string{"main.go", "utils.go"}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	if err != nil {
		t.Fatalf("Expected no error even with read failures, got: %v", err)
	}

	if context == nil {
		t.Fatal("Expected context to be non-nil")
	}

	if len(context.FileContents) != 0 {
		t.Errorf("Expected empty file contents due to read errors, got: %d", len(context.FileContents))
	}
}

func TestGatherEnhancedContext_NoSymbolContextTool(t *testing.T) {
	mockReadTool := &mockReadFileTool{
		files: map[string]string{
			"main.go": "package main\nfunc main() {}",
		},
	}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := "test diff"
	changedFiles := []string{"main.go"}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	// Should not fail, but symbol analysis should be empty
	if err != nil {
		t.Fatalf("Expected no error without symbol context tool, got: %v", err)
	}

	if context == nil {
		t.Fatal("Expected context to be non-nil")
	}

	// File contents should be populated
	if len(context.FileContents) != 1 {
		t.Errorf("Expected 1 file content, got: %d", len(context.FileContents))
	}

	// Symbol analysis should be empty
	if context.SymbolAnalysis != "" {
		t.Errorf("Expected empty symbol analysis without tool, got: %s", context.SymbolAnalysis)
	}
}

func TestGatherEnhancedContext_SymbolContextToolError(t *testing.T) {
	mockReadTool := &mockReadFileTool{
		files: map[string]string{
			"main.go": "package main\nfunc main() {}",
		},
	}

	mockSymbolTool := &mockSymbolContextTool{
		err: errors.New("symbol analysis failed"),
	}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)
	registry.Register("symbol_context", mockSymbolTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := "test diff"
	changedFiles := []string{"main.go"}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	// Should not fail completely, but symbol analysis should be empty
	if err != nil {
		t.Fatalf("Expected no error with symbol context failure, got: %v", err)
	}

	if context == nil {
		t.Fatal("Expected context to be non-nil")
	}

	// File contents should be populated
	if len(context.FileContents) != 1 {
		t.Errorf("Expected 1 file content, got: %d", len(context.FileContents))
	}

	// Symbol analysis should be empty due to error
	if context.SymbolAnalysis != "" {
		t.Errorf("Expected empty symbol analysis due to error, got: %s", context.SymbolAnalysis)
	}
}

func TestGatherEnhancedContext_EmptyChangedFiles(t *testing.T) {
	mockReadTool := &mockReadFileTool{
		files: map[string]string{},
	}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := "test diff"
	changedFiles := []string{}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	// Should not fail
	if err != nil {
		t.Fatalf("Expected no error with empty changed files, got: %v", err)
	}

	if context == nil {
		t.Fatal("Expected context to be non-nil")
	}

	// Should have empty file contents
	if len(context.FileContents) != 0 {
		t.Errorf("Expected empty file contents, got: %d", len(context.FileContents))
	}

	// Should preserve diff and changed files
	if context.Diff != diff {
		t.Errorf("Expected diff to be preserved, got: %s", context.Diff)
	}

	if len(context.ChangedFiles) != 0 {
		t.Errorf("Expected empty changed files, got: %d", len(context.ChangedFiles))
	}
}

func TestGatherEnhancedContext_ContextStructure(t *testing.T) {
	mockReadTool := &mockReadFileTool{
		files: map[string]string{
			"main.go": "package main",
		},
	}

	mockSymbolTool := &mockSymbolContextTool{
		response: "symbol analysis result",
	}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)
	registry.Register("symbol_context", mockSymbolTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := "test diff content"
	changedFiles := []string{"main.go"}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if context.Diff != diff {
		t.Errorf("Expected diff '%s', got '%s'", diff, context.Diff)
	}

	if len(context.ChangedFiles) != 1 || context.ChangedFiles[0] != "main.go" {
		t.Errorf("Expected changed files ['main.go'], got %v", context.ChangedFiles)
	}

	if context.FileContents == nil {
		t.Error("Expected FileContents to be initialized")
	}

	if context.SymbolAnalysis != "symbol analysis result" {
		t.Errorf("Expected symbol analysis 'symbol analysis result', got '%s'", context.SymbolAnalysis)
	}
}

func TestGatherEnhancedContext_SpecialCharactersInFiles(t *testing.T) {
	mockReadTool := &mockReadFileTool{
		files: map[string]string{
			"unicode.go": "package main\n// 测试 unicode characters\nfunc 测试() string {\n\treturn \"🚀 rocket\"\n}",
			"special.go": "package main\n// Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?\nfunc main() {}",
		},
	}

	mockSymbolTool := &mockSymbolContextTool{response: "special char analysis"}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)
	registry.Register("symbol_context", mockSymbolTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := "special char diff"
	changedFiles := []string{"unicode.go", "special.go"}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	if err != nil {
		t.Fatalf("Expected no error with special characters, got: %v", err)
	}

	if len(context.FileContents) != 2 {
		t.Errorf("Expected 2 file contents, got: %d", len(context.FileContents))
	}

	expectedUnicode := "package main\n// 测试 unicode characters\nfunc 测试() string {\n\treturn \"🚀 rocket\"\n}"
	if context.FileContents["unicode.go"] != expectedUnicode {
		t.Errorf("Expected unicode content to be preserved, got: %s", context.FileContents["unicode.go"])
	}
}

func TestGatherEnhancedContext_EmptyFiles(t *testing.T) {
	mockReadTool := &mockReadFileTool{
		files: map[string]string{
			"empty.go":      "",
			"whitespace.go": "   \n\t\n   ",
			"normal.go":     "package main",
		},
	}

	mockSymbolTool := &mockSymbolContextTool{response: "empty file analysis"}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)
	registry.Register("symbol_context", mockSymbolTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := "empty file diff"
	changedFiles := []string{"empty.go", "whitespace.go", "normal.go"}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	if err != nil {
		t.Fatalf("Expected no error with empty files, got: %v", err)
	}

	if len(context.FileContents) != 3 {
		t.Errorf("Expected 3 file contents, got: %d", len(context.FileContents))
	}

	// Check that empty content is preserved
	if context.FileContents["empty.go"] != "" {
		t.Errorf("Expected empty file to remain empty, got: '%s'", context.FileContents["empty.go"])
	}

	if context.FileContents["whitespace.go"] != "   \n\t\n   " {
		t.Errorf("Expected whitespace to be preserved, got: '%s'", context.FileContents["whitespace.go"])
	}
}

func TestGatherEnhancedContext_SymbolContextToolParameters(t *testing.T) {
	mockReadTool := &mockReadFileTool{
		files: map[string]string{
			"test.go": "package main\nfunc test() {}",
		},
	}

	mockSymbolTool := &mockSymbolContextTool{response: "parameter test"}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)
	registry.Register("symbol_context", mockSymbolTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := "test diff content"
	changedFiles := []string{"test.go"}

	_, err := agent.GatherEnhancedContext(diff, changedFiles)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if mockSymbolTool.lastArgs == nil {
		t.Fatal("Expected symbol context tool to be called")
	}

	if diffArg, ok := mockSymbolTool.lastArgs["diff"].(string); !ok || diffArg != diff {
		t.Errorf("Expected diff parameter '%s', got '%v'", diff, mockSymbolTool.lastArgs["diff"])
	}

	changedFilesArg, ok := mockSymbolTool.lastArgs["changed_files"].([]string)
	if !ok {
		t.Fatal("Expected changed_files parameter to be []string")
	}
	if len(changedFilesArg) != 1 || changedFilesArg[0] != "test.go" {
		t.Errorf("Expected changed_files ['test.go'], got %v", changedFilesArg)
	}

	fileContentsArg, ok := mockSymbolTool.lastArgs["file_contents"].(map[string]string)
	if !ok {
		t.Fatal("Expected file_contents parameter to be map[string]string")
	}
	if len(fileContentsArg) != 1 {
		t.Errorf("Expected 1 file content, got %d", len(fileContentsArg))
	}
	if fileContentsArg["test.go"] != "package main\nfunc test() {}" {
		t.Errorf("Expected file content to match, got: %s", fileContentsArg["test.go"])
	}
}

func TestGatherEnhancedContext_SymbolUsageIntegration(t *testing.T) {
	// This test simulates a realistic scenario:
	// 1. A function `ProcessUser` in user.go has changes inside its body (not signature)
	// 2. The function is used in multiple other files
	// 3. The symbol analysis should detect the function and find its usages
	// 4. The context should include information about where the function is used

	// Mock file contents representing a realistic e-commerce user management system
	mockReadTool := &mockReadFileTool{
		files: map[string]string{
			"internal/user/user.go": `package user

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type User struct {
	ID        int       ` + "`json:\"id\"`" + `
	Username  string    ` + "`json:\"username\"`" + `
	Email     string    ` + "`json:\"email\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	IsActive  bool      ` + "`json:\"is_active\"`" + `
}

// ProcessUser validates and processes a user for business operations
// This function has internal changes but the signature remains the same
func ProcessUser(user *User) error {
	// NEW: Enhanced validation logic added (this is what changed)
	if user == nil {
		return errors.New("user cannot be nil")
	}
	
	// NEW: Username validation
	if strings.TrimSpace(user.Username) == "" {
		return errors.New("username is required")
	}
	
	// NEW: Email format validation
	if !strings.Contains(user.Email, "@") || len(user.Email) < 5 {
		return errors.New("invalid email format")
	}
	
	// NEW: Business rule - inactive users cannot be processed
	if !user.IsActive {
		return errors.New("inactive users cannot be processed")
	}
	
	// Existing logic (unchanged)
	fmt.Printf("Processing user: %s (%s)\n", user.Username, user.Email)
	return nil
}

// CreateUser creates a new user instance
func CreateUser(username, email string) *User {
	return &User{
		Username:  username,
		Email:     email,
		CreatedAt: time.Now(),
		IsActive:  true,
	}
}

// ValidateUserData performs basic data validation
func ValidateUserData(username, email string) error {
	if username == "" {
		return errors.New("username required")
	}
	if email == "" {
		return errors.New("email required")
	}
	return nil
}`,

			"internal/handlers/user_handler.go": `package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	
	"myapp/internal/user"
)

type UserHandler struct {
	userService *user.Service
}

func NewUserHandler(service *user.Service) *UserHandler {
	return &UserHandler{userService: service}
}

// CreateUserHandler handles POST /users
func (h *UserHandler) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string ` + "`json:\"username\"`" + `
		Email    string ` + "`json:\"email\"`" + `
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	newUser := user.CreateUser(req.Username, req.Email)
	
	// USAGE: ProcessUser is called here for validation
	if err := user.ProcessUser(newUser); err != nil {
		http.Error(w, fmt.Sprintf("User validation failed: %v", err), http.StatusBadRequest)
		return
	}
	
	// Save user logic would go here...
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newUser)
}

// UpdateUserHandler handles PUT /users/{id}
func (h *UserHandler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	
	// Get existing user logic...
	existingUser := &user.User{ID: userID, Username: "existing", Email: "existing@test.com", IsActive: true}
	
	// USAGE: ProcessUser is called here before update
	if err := user.ProcessUser(existingUser); err != nil {
		http.Error(w, fmt.Sprintf("User processing failed: %v", err), http.StatusBadRequest)
		return
	}
	
	w.WriteHeader(http.StatusOK)
}`,

			"internal/services/user_service.go": `package services

import (
	"fmt"
	"log"
	
	"myapp/internal/user"
	"myapp/internal/database"
)

type UserService struct {
	db     *database.DB
	logger *log.Logger
}

func NewUserService(db *database.DB, logger *log.Logger) *UserService {
	return &UserService{db: db, logger: logger}
}

// RegisterUser registers a new user in the system
func (s *UserService) RegisterUser(username, email string) (*user.User, error) {
	// Validate input data
	if err := user.ValidateUserData(username, email); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	newUser := user.CreateUser(username, email)
	
	// USAGE: ProcessUser is called here for business validation
	if err := user.ProcessUser(newUser); err != nil {
		s.logger.Printf("User processing failed for %s: %v", username, err)
		return nil, fmt.Errorf("user processing failed: %w", err)
	}
	
	// Database save logic...
	s.logger.Printf("Successfully registered user: %s", username)
	return newUser, nil
}

// BulkProcessUsers processes multiple users at once
func (s *UserService) BulkProcessUsers(users []*user.User) error {
	for i, u := range users {
		// USAGE: ProcessUser is called for each user in bulk operation
		if err := user.ProcessUser(u); err != nil {
			return fmt.Errorf("bulk processing failed at index %d: %w", i, err)
		}
	}
	
	s.logger.Printf("Successfully processed %d users in bulk", len(users))
	return nil
}

// ActivateUser activates a user account
func (s *UserService) ActivateUser(userID int) error {
	// Get user from database...
	existingUser := &user.User{ID: userID, Username: "test", Email: "test@example.com", IsActive: false}
	
	// Activate the user
	existingUser.IsActive = true
	
	// USAGE: ProcessUser is called after activation
	if err := user.ProcessUser(existingUser); err != nil {
		return fmt.Errorf("failed to process activated user: %w", err)
	}
	
	return nil
}`,

			"cmd/main.go": `package main

import (
	"fmt"
	"log"
	"os"
	
	"myapp/internal/user"
	"myapp/internal/services"
)

func main() {
	logger := log.New(os.Stdout, "APP: ", log.LstdFlags)
	
	// Create some test users
	adminUser := user.CreateUser("admin", "admin@company.com")
	regularUser := user.CreateUser("john_doe", "john@example.com")
	
	// USAGE: ProcessUser is called in main for initial validation
	if err := user.ProcessUser(adminUser); err != nil {
		logger.Fatalf("Failed to process admin user: %v", err)
	}
	
	// USAGE: ProcessUser is called for regular user too
	if err := user.ProcessUser(regularUser); err != nil {
		logger.Fatalf("Failed to process regular user: %v", err)
	}
	
	fmt.Println("All users processed successfully!")
	
	// Initialize service
	userService := services.NewUserService(nil, logger)
	
	// Test bulk processing
	testUsers := []*user.User{adminUser, regularUser}
	if err := userService.BulkProcessUsers(testUsers); err != nil {
		logger.Fatalf("Bulk processing failed: %v", err)
	}
}`,

			"internal/user/user_test.go": `package user

import (
	"testing"
	"time"
)

func TestProcessUser_ValidUser(t *testing.T) {
	user := CreateUser("testuser", "test@example.com")
	
	// USAGE: ProcessUser is called in test
	err := ProcessUser(user)
	if err != nil {
		t.Errorf("Expected no error for valid user, got: %v", err)
	}
}

func TestProcessUser_NilUser(t *testing.T) {
	// USAGE: ProcessUser is called with nil
	err := ProcessUser(nil)
	if err == nil {
		t.Error("Expected error for nil user")
	}
	
	expectedMsg := "user cannot be nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestProcessUser_EmptyUsername(t *testing.T) {
	user := &User{
		Username:  "",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		IsActive:  true,
	}
	
	// USAGE: ProcessUser is called with invalid username
	err := ProcessUser(user)
	if err == nil {
		t.Error("Expected error for empty username")
	}
}

func TestProcessUser_InvalidEmail(t *testing.T) {
	user := &User{
		Username:  "testuser",
		Email:     "invalid-email",
		CreatedAt: time.Now(),
		IsActive:  true,
	}
	
	// USAGE: ProcessUser is called with invalid email
	err := ProcessUser(user)
	if err == nil {
		t.Error("Expected error for invalid email")
	}
}

func TestProcessUser_InactiveUser(t *testing.T) {
	user := &User{
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		IsActive:  false, // This should cause ProcessUser to fail
	}
	
	// USAGE: ProcessUser is called with inactive user
	err := ProcessUser(user)
	if err == nil {
		t.Error("Expected error for inactive user")
	}
}

func TestCreateUser(t *testing.T) {
	username := "newuser"
	email := "newuser@example.com"
	
	// USAGE: CreateUser is called
	user := CreateUser(username, email)
	
	if user.Username != username {
		t.Errorf("Expected username %s, got %s", username, user.Username)
	}
	if user.Email != email {
		t.Errorf("Expected email %s, got %s", email, user.Email)
	}
	if !user.IsActive {
		t.Error("Expected new user to be active")
	}
}`,
		},
	}

	mockSymbolTool := &mockSymbolContextTool{
		response: `=== SYMBOL ANALYSIS AND CONTEXT ===

Symbol: ProcessUser (function)
Location: internal/user/user.go:18
Found 8 usage(s):

In internal/handlers/user_handler.go:

Usage at line 32 (in function: CreateUserHandler):
 24:   }
 25:   
 26:   if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
 27:   	http.Error(w, "Invalid JSON", http.StatusBadRequest)
 28:   	return
 29:   }
 30:   
 31:   newUser := user.CreateUser(req.Username, req.Email)
→ 32:   if err := user.ProcessUser(newUser); err != nil {
 33:   	http.Error(w, fmt.Sprintf("User validation failed: %v", err), http.StatusBadRequest)
 34:   	return
 35:   }
 36:   
 37:   // Save user logic would go here...
 38:   w.Header().Set("Content-Type", "application/json")
 39:   json.NewEncoder(w).Encode(newUser)
 40:   }

----------------------------------------

Usage at line 48 (in function: UpdateUserHandler):
 40:   func (h *UserHandler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
 41:   	userID, _ := strconv.Atoi(r.URL.Query().Get("id"))
 42:   	
 43:   	// Get existing user logic...
 44:   	existingUser := &user.User{ID: userID, Username: "existing", Email: "existing@test.com", IsActive: true}
 45:   	
→ 48:   	if err := user.ProcessUser(existingUser); err != nil {
 49:   		http.Error(w, fmt.Sprintf("User processing failed: %v", err), http.StatusBadRequest)
 50:   		return
 51:   	}
 52:   	
 53:   	w.WriteHeader(http.StatusOK)
 54:   }


In internal/services/user_service.go:

Usage at line 28 (in function: RegisterUser):
 20:   	if err := user.ValidateUserData(username, email); err != nil {
 21:   		return nil, fmt.Errorf("validation failed: %w", err)
 22:   	}
 23:   	
 24:   	newUser := user.CreateUser(username, email)
 25:   	
→ 28:   	if err := user.ProcessUser(newUser); err != nil {
 29:   		s.logger.Printf("User processing failed for %s: %v", username, err)
 30:   		return nil, fmt.Errorf("user processing failed: %w", err)
 31:   	}
 32:   	
 33:   	// Database save logic...
 34:   	s.logger.Printf("Successfully registered user: %s", username)
 35:   	return newUser, nil
 36:   }

----------------------------------------

Usage at line 40 (in function: BulkProcessUsers):
 32:   func (s *UserService) BulkProcessUsers(users []*user.User) error {
 33:   	for i, u := range users {
→ 40:   		if err := user.ProcessUser(u); err != nil {
 41:   			return fmt.Errorf("bulk processing failed at index %d: %w", i, err)
 42:   		}
 43:   	}
 44:   	
 45:   	s.logger.Printf("Successfully processed %d users in bulk", len(users))
 46:   	return nil
 47:   }

Related files: internal/handlers/user_handler.go, internal/services/user_service.go, cmd/main.go, internal/user/user_test.go

================================================================================

Symbol: CreateUser (function)
Location: internal/user/user.go:44
Found 4 usage(s):

In internal/handlers/user_handler.go:

Usage at line 30 (in function: CreateUserHandler):
 26:   	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
 27:   		http.Error(w, "Invalid JSON", http.StatusBadRequest)
 28:   		return
 29:   	}
 30:   	
→ 31:   	newUser := user.CreateUser(req.Username, req.Email)
 32:   	
 33:   	// USAGE: ProcessUser is called here for validation
 34:   	if err := user.ProcessUser(newUser); err != nil {

Related files: internal/handlers/user_handler.go, internal/services/user_service.go, cmd/main.go, internal/user/user_test.go`,
	}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)
	registry.Register("symbol_context", mockSymbolTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := `diff --git a/internal/user/user.go b/internal/user/user.go
index a1b2c3d..e4f5g6h 100644
--- a/internal/user/user.go
+++ b/internal/user/user.go
@@ -18,8 +18,20 @@ type User struct {
 // ProcessUser validates and processes a user for business operations
 // This function has internal changes but the signature remains the same
 func ProcessUser(user *User) error {
-	// Basic validation
+	// NEW: Enhanced validation logic added (this is what changed)
 	if user == nil {
-		return errors.New("user is required")
+		return errors.New("user cannot be nil")
 	}
+	
+	// NEW: Username validation
+	if strings.TrimSpace(user.Username) == "" {
+		return errors.New("username is required")
+	}
+	
+	// NEW: Email format validation
+	if !strings.Contains(user.Email, "@") || len(user.Email) < 5 {
+		return errors.New("invalid email format")
+	}
+	
+	// NEW: Business rule - inactive users cannot be processed
+	if !user.IsActive {
+		return errors.New("inactive users cannot be processed")
+	}
 	
 	// Existing logic (unchanged)
-	fmt.Printf("Processing user: %s\n", user.Email)
+	fmt.Printf("Processing user: %s (%s)\n", user.Username, user.Email)
 	return nil
 }`

	changedFiles := []string{"internal/user/user.go"}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if context == nil {
		t.Fatal("Expected context to be non-nil")
	}

	if context.Diff != diff {
		t.Error("Expected diff to be preserved")
	}

	if len(context.ChangedFiles) != 1 || context.ChangedFiles[0] != "internal/user/user.go" {
		t.Errorf("Expected changed files ['internal/user/user.go'], got %v", context.ChangedFiles)
	}

	if len(context.FileContents) != 1 {
		t.Errorf("Expected 1 file content (internal/user/user.go), got %d", len(context.FileContents))
	}

	expectedUserGoContent := mockReadTool.files["internal/user/user.go"]
	if context.FileContents["internal/user/user.go"] != expectedUserGoContent {
		t.Error("Expected internal/user/user.go content to match mock file")
	}

	if context.SymbolAnalysis == "" {
		t.Fatal("Expected symbol analysis to be populated")
	}

	// Check that symbol analysis contains information about ProcessUser
	if !strings.Contains(context.SymbolAnalysis, "ProcessUser") {
		t.Error("Expected symbol analysis to contain ProcessUser function")
	}

	// Check that symbol analysis contains usage information
	if !strings.Contains(context.SymbolAnalysis, "Found 8 usage(s)") {
		t.Error("Expected symbol analysis to show ProcessUser usages")
	}

	// Check that symbol analysis mentions the files where ProcessUser is used
	expectedUsageFiles := []string{
		"internal/handlers/user_handler.go",
		"internal/services/user_service.go",
		"cmd/main.go",
		"internal/user/user_test.go",
	}
	for _, file := range expectedUsageFiles {
		if !strings.Contains(context.SymbolAnalysis, file) {
			t.Errorf("Expected symbol analysis to mention usage in %s", file)
		}
	}

	// Check that symbol analysis contains specific realistic usage lines with enhanced context
	expectedUsages := []string{
		"Usage at line 32 (in function: CreateUserHandler)",
		"→ 32:   if err := user.ProcessUser(newUser); err != nil {",
		"Usage at line 28 (in function: RegisterUser)",
		"→ 28:   	if err := user.ProcessUser(newUser); err != nil {",
		"Usage at line 40 (in function: BulkProcessUsers)",
		"→ 40:   		if err := user.ProcessUser(u); err != nil {",
	}
	for _, usage := range expectedUsages {
		if !strings.Contains(context.SymbolAnalysis, usage) {
			t.Errorf("Expected symbol analysis to contain usage: %s", usage)
		}
	}

	// Check that enhanced context includes surrounding lines
	expectedContextLines := []string{
		"http.Error(w, fmt.Sprintf(\"User validation failed: %v\", err), http.StatusBadRequest)",
		"s.logger.Printf(\"User processing failed for %s: %v\", username, err)",
		"return fmt.Errorf(\"bulk processing failed at index %d: %w\", i, err)",
	}
	for _, contextLine := range expectedContextLines {
		if !strings.Contains(context.SymbolAnalysis, contextLine) {
			t.Errorf("Expected symbol analysis to contain context line: %s", contextLine)
		}
	}

	// Verify that the symbol context tool was called with correct parameters
	if mockSymbolTool.lastArgs == nil {
		t.Fatal("Expected symbol context tool to be called")
	}

	// Check diff parameter
	if mockSymbolTool.lastArgs["diff"] != diff {
		t.Error("Expected correct diff to be passed to symbol context tool")
	}

	// Check changed_files parameter
	changedFilesArg, ok := mockSymbolTool.lastArgs["changed_files"].([]string)
	if !ok || len(changedFilesArg) != 1 || changedFilesArg[0] != "internal/user/user.go" {
		t.Errorf("Expected changed_files ['internal/user/user.go'], got %v", mockSymbolTool.lastArgs["changed_files"])
	}

	// Check file_contents parameter
	fileContentsArg, ok := mockSymbolTool.lastArgs["file_contents"].(map[string]string)
	if !ok {
		t.Fatal("Expected file_contents to be passed as map[string]string")
	}

	if len(fileContentsArg) != 1 {
		t.Errorf("Expected 1 file content in args, got %d", len(fileContentsArg))
	}

	if fileContentsArg["internal/user/user.go"] != expectedUserGoContent {
		t.Error("Expected internal/user/user.go content to be passed correctly to symbol context tool")
	}

	// Verify that CreateUser function is also detected (it's in the same file)
	if !strings.Contains(context.SymbolAnalysis, "CreateUser") {
		t.Error("Expected symbol analysis to also contain CreateUser function")
	}

	if !strings.Contains(context.SymbolAnalysis, "Found 4 usage(s)") {
		t.Error("Expected symbol analysis to show CreateUser usages")
	}

	// Verify enhanced context format is used
	if !strings.Contains(context.SymbolAnalysis, "Usage at line") {
		t.Error("Expected symbol analysis to use enhanced context format with 'Usage at line'")
	}

	// Verify function context is included
	if !strings.Contains(context.SymbolAnalysis, "(in function:") {
		t.Error("Expected symbol analysis to include function context")
	}

	// Verify arrow indicators for actual usage lines
	if !strings.Contains(context.SymbolAnalysis, "→") {
		t.Error("Expected symbol analysis to include arrow indicators for usage lines")
	}

	// Verify realistic cross-file references are detected
	if !strings.Contains(context.SymbolAnalysis, "user.ProcessUser") {
		t.Error("Expected symbol analysis to show qualified function calls (user.ProcessUser)")
	}
}

func TestGatherEnhancedContext_MultiFileSymbolUsage(t *testing.T) {

	mockReadTool := &mockReadFileTool{
		files: map[string]string{
			"auth.go": `package main

import "errors"

func ValidateUser(username, password string) error {
	if username == "" {
		return errors.New("username required")
	}
	
	// This calls a function in database.go
	user, err := GetUserFromDB(username)
	if err != nil {
		return err
	}
	
	return CheckPassword(user, password)
}`,

			"database.go": `package main

import "fmt"

type User struct {
	Username string
	Password string
}

// This function is called from auth.go
func GetUserFromDB(username string) (*User, error) {
	// Simulated database lookup with new caching logic
	fmt.Printf("Looking up user: %s\n", username)
	return &User{Username: username, Password: "hashed"}, nil
}

func CheckPassword(user *User, password string) error {
	// Password checking logic
	if user.Password != "hashed" {
		return fmt.Errorf("invalid password")
	}
	return nil
}`,
		},
	}

	mockSymbolTool := &mockSymbolContextTool{
		response: `=== CONTEXT AND SYMBOL ANALYSIS ===

Symbol: ValidateUser (function)
Location: auth.go:5
Found 0 usage(s)
Related files: 

---

Symbol: GetUserFromDB (function)
Location: database.go:12
Found 1 usage(s):
  In auth.go:
    Line 11: 	user, err := GetUserFromDB(username)
Related files: auth.go

---

Symbol: CheckPassword (function)
Location: database.go:18
Found 1 usage(s):
  In auth.go:
    Line 15: 	return CheckPassword(user, password)
Related files: auth.go`,
	}

	registry := tools.NewRegistry()
	registry.Register("read_file", mockReadTool)
	registry.Register("symbol_context", mockSymbolTool)

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(&mockLLMProvider{}, registry, cfg)

	diff := `diff --git a/auth.go b/auth.go
index 1234567..abcdefg 100644
--- a/auth.go
+++ b/auth.go
@@ -10,6 +10,7 @@ func ValidateUser(username, password string) error {
 	// This calls a function in database.go
 	user, err := GetUserFromDB(username)
 	if err != nil {
+		fmt.Printf("Database error: %v\n", err)
 		return err
 	}
 
diff --git a/database.go b/database.go
index 2345678..bcdefgh 100644
--- a/database.go
+++ b/database.go
@@ -12,6 +12,7 @@ type User struct {
 // This function is called from auth.go
 func GetUserFromDB(username string) (*User, error) {
 	// Simulated database lookup with new caching logic
+	// TODO: Add caching here
 	fmt.Printf("Looking up user: %s\n", username)
 	return &User{Username: username, Password: "hashed"}, nil
 }`

	changedFiles := []string{"auth.go", "database.go"}

	context, err := agent.GatherEnhancedContext(diff, changedFiles)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(context.FileContents) != 2 {
		t.Errorf("Expected 2 file contents, got %d", len(context.FileContents))
	}

	// Verify symbol analysis shows cross-references
	if !strings.Contains(context.SymbolAnalysis, "GetUserFromDB") {
		t.Error("Expected symbol analysis to contain GetUserFromDB")
	}

	if !strings.Contains(context.SymbolAnalysis, "CheckPassword") {
		t.Error("Expected symbol analysis to contain CheckPassword")
	}

	// Verify cross-file usage is detected
	if !strings.Contains(context.SymbolAnalysis, "In auth.go:") {
		t.Error("Expected symbol analysis to show usage in auth.go")
	}

	// Verify the tool received both files
	fileContentsArg := mockSymbolTool.lastArgs["file_contents"].(map[string]string)
	if len(fileContentsArg) != 2 {
		t.Errorf("Expected 2 files passed to symbol tool, got %d", len(fileContentsArg))
	}

	if _, hasAuth := fileContentsArg["auth.go"]; !hasAuth {
		t.Error("Expected auth.go to be passed to symbol tool")
	}

	if _, hasDB := fileContentsArg["database.go"]; !hasDB {
		t.Error("Expected database.go to be passed to symbol tool")
	}
}
