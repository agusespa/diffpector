package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
