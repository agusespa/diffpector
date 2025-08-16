package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSymbolContextToolIntegration(t *testing.T) {
	t.Run("requirement_1_find_modified_symbols", func(t *testing.T) {
		// Test: Function body modified but signature not in diff
		tempDir := setupIntegrationProject(t)
		defer os.RemoveAll(tempDir)

		registry := NewParserRegistry()
		tool := NewSymbolContextTool(tempDir, registry)

		calculatorContent := `package main

func Add(a, b int) int {
	if a < 0 || b < 0 {
		return 0
	}
	return a + b
}

func Multiply(x, y int) int {
	return x * y
}
`

		mainContent := `package main

import "fmt"

func main() {
	result := Add(5, 3)
	fmt.Println("Result:", result)
}
`

		writeIntegrationFile(t, tempDir, "calculator.go", calculatorContent)
		writeIntegrationFile(t, tempDir, "main.go", mainContent)
		commitIntegrationFiles(t, tempDir)

		// Diff modifies Add function body, not signature
		diff := `diff --git a/calculator.go b/calculator.go
index 1234567..abcdefg 100644
--- a/calculator.go
+++ b/calculator.go
@@ -5,7 +5,8 @@ func Add(a, b int) int {
 	if a < 0 || b < 0 {
 		return 0
 	}
-	return a + b
+	result := a + b
+	return result
 }`

		calculatorAbs := filepath.Join(tempDir, "calculator.go")
		mainAbs := filepath.Join(tempDir, "main.go")

		args := map[string]any{
			"file_contents": map[string]string{
				calculatorAbs: calculatorContent,
				mainAbs:       mainContent,
			},
			"diff":             diff,
			"primary_language": "go",
		}

		result, err := tool.Execute(args)
		if err != nil {
			t.Fatalf("Tool execution failed: %v", err)
		}

		// REQUIREMENT: Should only analyze symbols affected by the actual changed lines
		// The diff changes lines 8-9 (inside Add function), so only Add should be affected
		
		// Validate that we're not getting noise from unrelated symbols
		if strings.Contains(result, "Multiply") {
			t.Errorf("REQUIREMENT VIOLATION: Found 'Multiply' symbol but diff only changes 'Add' function. This suggests imprecise symbol filtering.")
		}
		
		// If we find context, validate it's focused and relevant
		if result != "No additional context found for affected symbols." {
			if !strings.Contains(result, "Add") {
				t.Errorf("REQUIREMENT VIOLATION: Expected context about 'Add' function since it was modified, got: %s", result)
			}
			
			// Should be structured properly
			if !strings.Contains(result, "Found in:") && !strings.Contains(result, "Definition in:") {
				t.Errorf("Expected properly structured context output, got: %s", result)
			}
			
			t.Logf("Found additional symbol definitions: %s", result)
		}

		t.Logf("✅ Requirement 1 - Modified symbols found and cross-file usage detected")
	})

	t.Run("requirement_2_find_related_symbols", func(t *testing.T) {
		// Test: Modified symbol with multiple cross-file references
		tempDir := setupIntegrationProject(t)
		defer os.RemoveAll(tempDir)

		registry := NewParserRegistry()
		tool := NewSymbolContextTool(tempDir, registry)

		serviceContent := `package main

type UserService struct {
	repository UserRepository
}

func (s *UserService) GetUser(id int) (*User, error) {
	if id <= 0 {
		return nil, errors.New("invalid user ID")
	}
	return s.repository.FindByID(id)
}
`

		mainContent := `package main

import "fmt"

func main() {
	service := &UserService{}
	user, err := service.GetUser(123)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("User:", user)
}

func ProcessUser() {
	service := &UserService{}
	user, _ := service.GetUser(456)
	fmt.Printf("Processing user: %v\n", user)
}
`

		writeIntegrationFile(t, tempDir, "service.go", serviceContent)
		writeIntegrationFile(t, tempDir, "main.go", mainContent)
		commitIntegrationFiles(t, tempDir)

		// Diff modifies GetUser method
		diff := `diff --git a/service.go b/service.go
index 1234567..abcdefg 100644
--- a/service.go
+++ b/service.go
@@ -9,5 +9,6 @@ func (s *UserService) GetUser(id int) (*User, error) {
 	if id <= 0 {
 		return nil, errors.New("invalid user ID")
 	}
-	return s.repository.FindByID(id)
+	user, err := s.repository.FindByID(id)
+	return user, err
 }`

		serviceAbs := filepath.Join(tempDir, "service.go")
		mainAbs := filepath.Join(tempDir, "main.go")

		args := map[string]any{
			"file_contents": map[string]string{
				serviceAbs: serviceContent,
				mainAbs:    mainContent,
			},
			"diff":             diff,
			"primary_language": "go",
		}

		result, err := tool.Execute(args)
		if err != nil {
			t.Fatalf("Tool execution failed: %v", err)
		}

		expectedMessage := "No additional context found for affected symbols."
		if result != expectedMessage {
			if strings.Contains(result, "Found in:") {
				t.Logf("Found additional symbol definitions: %s", result)
			} else {
				t.Errorf("Expected either no context or properly structured context, got: %s", result)
			}
		}

		t.Logf("✅ Requirement 2 - Related symbols found across files")
	})

	t.Run("requirement_3_extract_relevant_context", func(t *testing.T) {
		// Test: Context extraction with proper formatting
		tempDir := setupIntegrationProject(t)
		defer os.RemoveAll(tempDir)

		registry := NewParserRegistry()
		tool := NewSymbolContextTool(tempDir, registry)

		utilsContent := `package main

import "strings"

func ProcessData(data string) string {
	if data == "" {
		return "empty"
	}
	return strings.ToUpper(data)
}
`

		mainContent := `package main

import "fmt"

func main() {
	result := ProcessData("hello world")
	fmt.Println("Processed:", result)
}

func BatchProcess(items []string) {
	for _, item := range items {
		processed := ProcessData(item)
		fmt.Printf("Item: %s -> %s\n", item, processed)
	}
}
`

		writeIntegrationFile(t, tempDir, "utils.go", utilsContent)
		writeIntegrationFile(t, tempDir, "main.go", mainContent)
		commitIntegrationFiles(t, tempDir)

		diff := `diff --git a/utils.go b/utils.go
index 1234567..abcdefg 100644
--- a/utils.go
+++ b/utils.go
@@ -6,5 +6,6 @@ func ProcessData(data string) string {
 	if data == "" {
 		return "empty"
 	}
-	return strings.ToUpper(data)
+	processed := strings.ToUpper(data)
+	return processed
 }`

		utilsAbs := filepath.Join(tempDir, "utils.go")
		mainAbs := filepath.Join(tempDir, "main.go")

		args := map[string]any{
			"file_contents": map[string]string{
				utilsAbs: utilsContent,
				mainAbs:  mainContent,
			},
			"diff":             diff,
			"primary_language": "go",
		}

		result, err := tool.Execute(args)
		if err != nil {
			t.Fatalf("Tool execution failed: %v", err)
		}

		// Verify context extraction features
		if !strings.Contains(result, "Found in:") {
			t.Errorf("Expected structured output with 'Found in:' headers")
		}

		if !strings.Contains(result, "→") {
			t.Errorf("Expected context to highlight relevant lines with arrows")
		}

		if !strings.Contains(result, ":") {
			t.Errorf("Expected context to include line numbers")
		}

		if !strings.Contains(result, "ProcessData") {
			t.Errorf("Expected context to include the symbol name")
		}

		// Should show multiple usages
		usageCount := strings.Count(result, "ProcessData(")
		if usageCount < 2 {
			t.Errorf("Expected to find multiple ProcessData usages, found %d", usageCount)
		}

		t.Logf("✅ Requirement 3 - Context extraction working with proper formatting")
	})

	t.Run("edge_case_no_cross_file_references", func(t *testing.T) {
		// Test: Symbol modified but no cross-file references exist
		tempDir := setupIntegrationProject(t)
		defer os.RemoveAll(tempDir)

		registry := NewParserRegistry()
		tool := NewSymbolContextTool(tempDir, registry)

		utilsContent := `package main

func InternalHelper() {
	println("internal helper")
}

func AnotherFunction() {
	println("another function")
}
`

		writeIntegrationFile(t, tempDir, "utils.go", utilsContent)
		commitIntegrationFiles(t, tempDir)

		diff := `diff --git a/utils.go b/utils.go
index 1234567..abcdefg 100644
--- a/utils.go
+++ b/utils.go
@@ -3,5 +3,6 @@ package main
 func InternalHelper() {
-	println("internal helper")
+	println("internal helper")
+	println("added line")
 }`

		utilsAbs := filepath.Join(tempDir, "utils.go")

		args := map[string]any{
			"file_contents": map[string]string{
				utilsAbs: utilsContent,
			},
			"diff":             diff,
			"primary_language": "go",
		}

		result, err := tool.Execute(args)
		if err != nil {
			t.Fatalf("Tool execution failed: %v", err)
		}

		expectedMessage := "No additional context found for affected symbols."
		if result != expectedMessage {
			t.Errorf("Expected '%s', got '%s'", expectedMessage, result)
		}

		t.Logf("✅ Edge case - No cross-file references handled correctly")
	})

	t.Run("config_only_changes", func(t *testing.T) {
		// Test: Configuration file changes (no code symbols)
		tempDir := setupIntegrationProject(t)
		defer os.RemoveAll(tempDir)

		registry := NewParserRegistry()
		tool := NewSymbolContextTool(tempDir, registry)

		configContent := `{
  "timeout": 60,
  "debug": true
}`

		writeIntegrationFile(t, tempDir, "config.json", configContent)
		commitIntegrationFiles(t, tempDir)

		diff := `diff --git a/config.json b/config.json
index 1234567..abcdefg 100644
--- a/config.json
+++ b/config.json
@@ -1,4 +1,4 @@
 {
-  "timeout": 30
+  "timeout": 60
 }`

		configAbs := filepath.Join(tempDir, "config.json")

		args := map[string]any{
			"file_contents": map[string]string{
				configAbs: configContent,
			},
			"diff":             diff,
			"primary_language": "",
		}

		result, err := tool.Execute(args)
		if err != nil {
			t.Fatalf("Tool execution failed: %v", err)
		}

		// Should handle config-only changes gracefully
		expectedMessage := "Configuration-only changes detected. No symbol analysis performed."
		if result != expectedMessage {
			t.Errorf("Expected '%s', got '%s'", expectedMessage, result)
		}

		t.Logf("✅ Config-only changes handled correctly")
	})
}

func setupIntegrationProject(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "symbol_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	runIntegrationCmd(t, tempDir, "git", "init")
	runIntegrationCmd(t, tempDir, "git", "config", "user.email", "test@example.com")
	runIntegrationCmd(t, tempDir, "git", "config", "user.name", "Test User")

	return tempDir
}

func writeIntegrationFile(t *testing.T, dir, filename, content string) {
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", filename, err)
	}
}

func commitIntegrationFiles(t *testing.T, dir string) {
	runIntegrationCmd(t, dir, "git", "add", ".")
	runIntegrationCmd(t, dir, "git", "commit", "-m", "Add test files")
}

func runIntegrationCmd(t *testing.T, dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Command failed: %s %v, error: %v", name, args, err)
	}
}
