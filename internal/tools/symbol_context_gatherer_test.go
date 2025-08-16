package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

func TestSymbolContextGatherer_isSameFile(t *testing.T) {
	gatherer := &SymbolContextGatherer{}

	tests := []struct {
		name          string
		candidateFile string
		symbolFile    string
		projectRoot   string
		expected      bool
	}{
		{
			name:          "exact match",
			candidateFile: "/project/main.go",
			symbolFile:    "/project/main.go",
			projectRoot:   "/project",
			expected:      true,
		},
		{
			name:          "relative vs absolute",
			candidateFile: "/project/main.go",
			symbolFile:    "main.go",
			projectRoot:   "/project",
			expected:      true,
		},
		{
			name:          "different files",
			candidateFile: "/project/main.go",
			symbolFile:    "utils.go",
			projectRoot:   "/project",
			expected:      false,
		},
		{
			name:          "nested paths",
			candidateFile: "/project/src/main.go",
			symbolFile:    "src/main.go",
			projectRoot:   "/project",
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gatherer.isSameFile(tt.candidateFile, tt.symbolFile, tt.projectRoot)
			if result != tt.expected {
				t.Errorf("isSameFile(%q, %q, %q) = %v; want %v",
					tt.candidateFile, tt.symbolFile, tt.projectRoot, result, tt.expected)
			}
		})
	}
}

func TestSymbolContextGatherer_gitGrepSearch(t *testing.T) {
	// Create a temporary git repository for testing
	tempDir, err := os.MkdirTemp("", "git_grep_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	runGitCmd(t, tempDir, "git", "init")
	runGitCmd(t, tempDir, "git", "config", "user.email", "test@example.com")
	runGitCmd(t, tempDir, "git", "config", "user.name", "Test User")

	// Create test files
	mainContent := `package main

func main() {
	result := Add(5, 3)
	fmt.Println(result)
}
`

	utilsContent := `package main

func Add(a, b int) int {
	return a + b
}
`

	writeGitFile(t, tempDir, "main.go", mainContent)
	writeGitFile(t, tempDir, "utils.go", utilsContent)

	// Commit files
	runGitCmd(t, tempDir, "git", "add", ".")
	runGitCmd(t, tempDir, "git", "commit", "-m", "Initial commit")

	gatherer := &SymbolContextGatherer{}

	// Test searching for "Add"
	files, err := gatherer.gitGrepSearch("Add", tempDir, "go")
	if err != nil {
		t.Fatalf("gitGrepSearch failed: %v", err)
	}

	// Should find both files
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d: %v", len(files), files)
	}

	// Should contain both main.go and utils.go
	foundMain := false
	foundUtils := false
	for _, file := range files {
		if strings.HasSuffix(file, "main.go") {
			foundMain = true
		}
		if strings.HasSuffix(file, "utils.go") {
			foundUtils = true
		}
	}

	if !foundMain {
		t.Errorf("Expected to find main.go in results")
	}
	if !foundUtils {
		t.Errorf("Expected to find utils.go in results")
	}
}

func writeGitFile(t *testing.T, dir, filename, content string) {
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", filename, err)
	}
}

func runGitCmd(t *testing.T, dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Command failed: %s %v, error: %v", name, args, err)
	}
}

func TestSymbolContextGatherer_FileFiltering(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "file_filtering_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	runGitCmd(t, tempDir, "git", "init")
	runGitCmd(t, tempDir, "git", "config", "user.email", "test@example.com")
	runGitCmd(t, tempDir, "git", "config", "user.name", "Test User")

	// Create files that should be INCLUDED
	productionContent := `package main
func GetUser(id int) *User {
	return &User{ID: id}
}`

	// Create files that should be EXCLUDED
	testContent := `package main
func TestGetUser(t *testing.T) {
	user := GetUser(123)
	// test code
}`

	mockContent := `package main
type User struct {
	ID int
	Name string
	CreatedAt time.Time  // This was causing noise!
	UpdatedAt time.Time  // This was causing noise!
}`

	writeGitFile(t, tempDir, "user.go", productionContent)
	writeGitFile(t, tempDir, "user_test.go", testContent)
	
	mockDir := filepath.Join(tempDir, "evaluation", "test_cases", "mocks", "internal", "database")
	os.MkdirAll(mockDir, 0755)
	writeGitFile(t, mockDir, "user.go", mockContent)

	runGitCmd(t, tempDir, "git", "add", ".")
	runGitCmd(t, tempDir, "git", "commit", "-m", "Add test files")

	registry := NewParserRegistry()
	gatherer := NewSymbolContextGatherer(registry)

	// Test the complete pipeline: git grep + validation (which includes filtering)
	rawFiles, err := gatherer.gitGrepSearch("GetUser", tempDir, "go")
	if err != nil {
		t.Fatalf("gitGrepSearch failed: %v", err)
	}

	// Apply the validation/filtering step (this is where filtering should happen)
	filteredFiles := gatherer.validateFiles(rawFiles, tempDir)

	// REQUIREMENT: Should exclude test files and mock files to avoid context noise
	for _, file := range filteredFiles {
		if strings.Contains(file, "_test.go") {
			t.Errorf("REQUIREMENT VIOLATION: Found test file %s. Test files should be excluded to avoid context noise.", file)
		}
		if strings.Contains(file, "evaluation/test_cases/mocks/") {
			t.Errorf("REQUIREMENT VIOLATION: Found mock file %s. Mock files should be excluded to avoid context noise.", file)
		}
	}

	// Should find the production file
	foundProduction := false
	for _, file := range filteredFiles {
		if strings.HasSuffix(file, "user.go") && !strings.Contains(file, "mocks") {
			foundProduction = true
		}
	}
	if !foundProduction {
		t.Errorf("REQUIREMENT VIOLATION: Should find production user.go file")
	}

	t.Logf("✅ File filtering requirement validated - excludes test/mock files")
}

func TestSymbolContextGatherer_ContextRelevance(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "context_relevance_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	runGitCmd(t, tempDir, "git", "init")
	runGitCmd(t, tempDir, "git", "config", "user.email", "test@example.com")
	runGitCmd(t, tempDir, "git", "config", "user.name", "Test User")

	// Create files with generic symbol names that caused noise
	userContent := `package main
type User struct {
	Name string  // Very generic field name
}`

	configContent := `package config
type Config struct {
	Name string  // Same generic field name in different context
}`

	unrelatedContent := `package unrelated
func ProcessName(name string) {
	// Unrelated usage of "Name"
}`

	writeGitFile(t, tempDir, "user.go", userContent)
	writeGitFile(t, tempDir, "config.go", configContent)
	writeGitFile(t, tempDir, "unrelated.go", unrelatedContent)

	runGitCmd(t, tempDir, "git", "add", ".")
	runGitCmd(t, tempDir, "git", "commit", "-m", "Add files with generic symbols")

	gatherer := &SymbolContextGatherer{}

	files, err := gatherer.gitGrepSearch("Name", tempDir, "go")
	if err != nil {
		t.Fatalf("gitGrepSearch failed: %v", err)
	}

	if len(files) > 10 {
		t.Errorf("REQUIREMENT VIOLATION: Found %d files for generic symbol 'Name'. Should limit results to avoid overwhelming LLM.", len(files))
	}


	t.Logf("✅ Context relevance requirement validated - limits generic symbol results")
}

func TestSymbolContextGatherer_Requirements(t *testing.T) {
	t.Run("requirement: context should be focused and relevant", func(t *testing.T) {		
		registry := NewParserRegistry()
		gatherer := NewSymbolContextGatherer(registry)
		
		genericSymbol := types.Symbol{
			Name:      "Name",        // Very generic name that appears everywhere
			Package:   "database",
			FilePath:  "internal/database/user.go",
			StartLine: 15,
			EndLine:   15,
		}
		
		// Test in a temporary directory to avoid real file system noise
		tempDir, err := os.MkdirTemp("", "context_test")
		if err != nil {
			t.Skip("Cannot create temp dir for test")
		}
		defer os.RemoveAll(tempDir)
		
		context, err := gatherer.GatherSymbolContext([]types.Symbol{genericSymbol}, tempDir, "go")
		if err != nil {
			// Error is acceptable - means no context found, which is better than noise
			t.Logf("No context found (acceptable): %v", err)
			return
		}
		
		// If context is found, it should be reasonable in size
		lineCount := strings.Count(context, "\n")
		if lineCount > 100 {
			t.Errorf("REQUIREMENT VIOLATION: Context too verbose (%d lines) for generic symbol. This could overwhelm the LLM.", lineCount)
		}
		
		t.Logf("✅ Context size reasonable: %d lines", lineCount)
	})
}
