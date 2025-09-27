package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSymbolContextGatherer_gitGrepSearch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "git_grep_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	runGitCmd(t, tempDir, "git", "init")

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

	runGitCmd(t, tempDir, "git", "add", ".")
	runGitCmd(t, tempDir, "git", "commit", "-m", "Initial commit")

	gatherer := &SymbolContextGatherer{}

	files, err := gatherer.gitGrepSearch("Add", tempDir, "go")
	if err != nil {
		t.Fatalf("gitGrepSearch failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d: %v", len(files), files)
	}

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

func TestSymbolContextGatherer_validateFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "file_filtering_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	runGitCmd(t, tempDir, "git", "init")
	runGitCmd(t, tempDir, "git", "config", "user.email", "test@example.com")
	runGitCmd(t, tempDir, "git", "config", "user.name", "Test User")

	// Files that should be INCLUDED
	productionContent := `package main
func GetUser(id int) *User {
	return &User{ID: id}
}`

	// Files that should be EXCLUDED
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
	if err := os.MkdirAll(mockDir, 0755); err != nil {
		t.Fatalf("Failed to create mock dir: %v", err)
	}
	writeGitFile(t, mockDir, "user.go", mockContent)

	runGitCmd(t, tempDir, "git", "add", ".")
	runGitCmd(t, tempDir, "git", "commit", "-m", "Add test files")

	registry := NewParserRegistry()
	gatherer := NewSymbolContextGatherer(registry)

	rawFiles, err := gatherer.gitGrepSearch("GetUser", tempDir, "go")
	if err != nil {
		t.Fatalf("gitGrepSearch failed: %v", err)
	}

	filteredFiles := gatherer.validateFiles(rawFiles, tempDir)

	for _, file := range filteredFiles {
		if strings.Contains(file, "_test.go") {
			t.Errorf("REQUIREMENT VIOLATION: Found test file %s. Test files should be excluded to avoid context noise.", file)
		}
		if strings.Contains(file, "evaluation/test_cases/mocks/") {
			t.Errorf("REQUIREMENT VIOLATION: Found mock file %s. Mock files should be excluded to avoid context noise.", file)
		}
	}

	foundProduction := false
	for _, file := range filteredFiles {
		if strings.HasSuffix(file, "user.go") && !strings.Contains(file, "mocks") {
			foundProduction = true
		}
	}
	if !foundProduction {
		t.Errorf("REQUIREMENT VIOLATION: Should find production user.go file")
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
