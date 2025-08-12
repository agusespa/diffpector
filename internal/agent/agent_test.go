package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/pkg/config"
)

type mockTool struct {
	name     string
	response string
	err      error
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return "Mock tool for testing"
}

func (m *mockTool) Execute(params map[string]any) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

type mockLLMProvider struct {
	response string
	err      error
	model    string
}

func (m *mockLLMProvider) Generate(prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockLLMProvider) GetModel() string {
	if m.model == "" {
		return "mock-model"
	}
	return m.model
}

func (m *mockLLMProvider) SetModel(model string) {
	m.model = model
}

func TestValidateAndDetectLanguage(t *testing.T) {
	parserRegistry := tools.NewParserRegistry()
	agent := &CodeReviewAgent{
		parserRegistry: parserRegistry,
	}

	tests := []struct {
		name          string
		files         []string
		expectedLang  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "single go file",
			files:        []string{"main.go"},
			expectedLang: "go",
			expectError:  false,
		},
		{
			name:         "config files only",
			files:        []string{"package.json", "Dockerfile"},
			expectedLang: "",
			expectError:  false,
		},
		{
			name:         "mixed go and config",
			files:        []string{"main.go", "package.json"},
			expectedLang: "go",
			expectError:  false,
		},
		{
			name:          "unsupported language",
			files:         []string{"script.py"},
			expectedLang:  "",
			expectError:   true,
			errorContains: "unsupported language file",
		},
		{
			name:          "unsupported_java_file",
			files:         []string{"main.go", "script.java"},
			expectedLang:  "",
			expectError:   true,
			errorContains: "unsupported language file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang, err := agent.validateAndDetectLanguage(tt.files)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if lang != tt.expectedLang {
					t.Errorf("Expected language '%s', got '%s'", tt.expectedLang, lang)
				}
			}
		})
	}
}

func TestGatherEnhancedContext(t *testing.T) {
	// Setup
	registry := tools.NewRegistry()
	parserRegistry := tools.NewParserRegistry()
	
	// Mock tools
	registry.Register(tools.ToolNameReadFile, &mockTool{
		name:     "read_file",
		response: "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}",
	})
	
	registry.Register(tools.ToolNameSymbolContext, &mockTool{
		name:     "symbol_context",
		response: "Symbol analysis: Found function 'main' in package 'main'",
	})

	agent := &CodeReviewAgent{
		toolRegistry:   registry,
		parserRegistry: parserRegistry,
	}

	// Test data
	diff := `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
 
-func main() {}
+func main() {
+	println("hello")
+}`

	changedFiles := []string{"main.go"}
	primaryLanguage := "go"

	// Execute
	context, err := agent.GatherEnhancedContext(diff, changedFiles, primaryLanguage)

	// Verify
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if context == nil {
		t.Fatal("Expected context to be non-nil")
	}

	if context.Diff != diff {
		t.Errorf("Expected diff to match input")
	}

	if len(context.ChangedFiles) != 1 || context.ChangedFiles[0] != "main.go" {
		t.Errorf("Expected changed files to contain main.go")
	}

	if len(context.FileContents) != 1 {
		t.Errorf("Expected file contents to have 1 entry, got %d", len(context.FileContents))
	}

	if context.FileContents["main.go"] == "" {
		t.Errorf("Expected file contents for main.go to be populated")
	}

	if context.SymbolAnalysis == "" {
		t.Errorf("Expected symbol analysis to be populated")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr || 
			 containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestToolRegistryIntegration(t *testing.T) {
	// Test that all required tools can be registered and retrieved
	registry := tools.NewRegistry()
	
	// Register mock implementations of required tools
	requiredTools := map[tools.ToolName]string{
		tools.ToolNameGitStagedFiles: "file1.go\nfile2.go",
		tools.ToolNameGitDiff:        "diff --git a/file1.go b/file1.go\n+added line",
		tools.ToolNameReadFile:       "package main\nfunc main() {}",
		tools.ToolNameSymbolContext:  "Found 1 function: main",
		tools.ToolNameWriteFile:      "File written successfully",
	}

	for toolName, response := range requiredTools {
		registry.Register(toolName, &mockTool{
			name:     string(toolName),
			response: response,
		})
	}

	// Verify all tools are accessible
	for toolName := range requiredTools {
		tool := registry.Get(toolName)
		if tool == nil {
			t.Errorf("Tool %s not found in registry", toolName)
		}
		
		// Test execution
		result, err := tool.Execute(map[string]any{})
		if err != nil {
			t.Errorf("Tool %s execution failed: %v", toolName, err)
		}
		if result == "" {
			t.Errorf("Tool %s returned empty result", toolName)
		}
	}
}

func TestEndToEndContextGathering(t *testing.T) {
	// Setup complete agent with all dependencies
	registry := tools.NewRegistry()
	parserRegistry := tools.NewParserRegistry()
	cfg := &config.Config{}
	
	// Register all required tools with realistic mock data
	registry.Register(tools.ToolNameGitStagedFiles, &mockTool{
		name:     "git_staged_files",
		response: "main.go\nutils.go",
	})
	
	registry.Register(tools.ToolNameGitDiff, &mockTool{
		name: "git_diff",
		response: `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main
 
 func main() {
-	println("old")
+	println("new")
+	doSomething()
 }`,
	})
	
	registry.Register(tools.ToolNameReadFile, &mockTool{
		name: "read_file",
		response: `package main

func main() {
	println("new")
	doSomething()
}

func doSomething() {
	// implementation
}`,
	})
	
	registry.Register(tools.ToolNameSymbolContext, &mockTool{
		name:     "symbol_context",
		response: "Symbol analysis: Found functions 'main', 'doSomething' in package 'main'",
	})

	// Create agent
	mockProvider := &mockLLMProvider{
		response: `[{"severity": "minor", "file_path": "main.go", "start_line": 4, "end_line": 4, "description": "Consider adding error handling"}]`,
	}
	
	agent := NewCodeReviewAgent(mockProvider, registry, cfg, parserRegistry)

	// Test the main execution flow components
	t.Run("validate_and_detect_language", func(t *testing.T) {
		files := []string{"main.go", "utils.go"}
		lang, err := agent.validateAndDetectLanguage(files)
		if err != nil {
			t.Fatalf("Language validation failed: %v", err)
		}
		if lang != "go" {
			t.Errorf("Expected language 'go', got '%s'", lang)
		}
	})

	t.Run("gather_enhanced_context", func(t *testing.T) {
		diff := "mock diff content"
		files := []string{"main.go"}
		primaryLang := "go"
		
		context, err := agent.GatherEnhancedContext(diff, files, primaryLang)
		if err != nil {
			t.Fatalf("Context gathering failed: %v", err)
		}
		
		// Verify context structure
		if context.Diff != diff {
			t.Error("Diff not preserved in context")
		}
		if len(context.ChangedFiles) != 1 {
			t.Error("Changed files not preserved in context")
		}
		if len(context.FileContents) != 1 {
			t.Error("File contents not populated")
		}
		if context.SymbolAnalysis == "" {
			t.Error("Symbol analysis not populated")
		}
	})

	t.Run("read_file_contents", func(t *testing.T) {
		files := []string{"main.go", "utils.go"}
		contents, err := agent.readFileContents(files)
		if err != nil {
			t.Fatalf("Reading file contents failed: %v", err)
		}
		
		if len(contents) != 2 {
			t.Errorf("Expected 2 file contents, got %d", len(contents))
		}
		
		for _, file := range files {
			if contents[file] == "" {
				t.Errorf("File content for %s is empty", file)
			}
		}
	})
}

func TestErrorHandling(t *testing.T) {
	registry := tools.NewRegistry()
	parserRegistry := tools.NewParserRegistry()
	cfg := &config.Config{}
	
	// Register tools that will fail
	registry.Register(tools.ToolNameReadFile, &mockTool{
		name: "read_file",
		err:  fmt.Errorf("file not found"),
	})
	
	mockProvider := &mockLLMProvider{}
	agent := NewCodeReviewAgent(mockProvider, registry, cfg, parserRegistry)

	t.Run("read_file_error", func(t *testing.T) {
		files := []string{"nonexistent.go"}
		_, err := agent.readFileContents(files)
		if err == nil {
			t.Error("Expected error when reading non-existent file")
		}
	})

	t.Run("symbol_context_error", func(t *testing.T) {
		// Register failing symbol context tool
		registry.Register(tools.ToolNameSymbolContext, &mockTool{
			name: "symbol_context",
			err:  fmt.Errorf("symbol analysis failed"),
		})
		
		// Register working read file tool
		registry.Register(tools.ToolNameReadFile, &mockTool{
			name:     "read_file",
			response: "package main",
		})

		diff := "mock diff"
		files := []string{"main.go"}
		primaryLang := "go"
		
		_, err := agent.GatherEnhancedContext(diff, files, primaryLang)
		if err == nil {
			t.Error("Expected error when symbol analysis fails")
		}
		if !containsString(err.Error(), "symbol analysis failed") {
			t.Errorf("Expected error message to contain 'symbol analysis failed', got: %s", err.Error())
		}
	})
}

func TestWithRealFileOperations(t *testing.T) {
	// Skip if not in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository, skipping test with git operations")
	}

	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	testContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup registry with real tools
	registry := tools.NewRegistry()
	parserRegistry := tools.NewParserRegistry()
	
	// Register real tools
	registry.Register(tools.ToolNameReadFile, &tools.ReadFileTool{})
	registry.Register(tools.ToolNameSymbolContext, tools.NewSymbolContextTool(tempDir, parserRegistry))

	// Mock provider for this test
	mockProvider := &mockLLMProvider{
		response: "Mock review response",
	}

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(mockProvider, registry, cfg, parserRegistry)

	// Test reading file contents
	t.Run("read_file_contents_real", func(t *testing.T) {
		files := []string{testFile}
		contents, err := agent.readFileContents(files)
		if err != nil {
			t.Fatalf("Failed to read file contents: %v", err)
		}

		if len(contents) != 1 {
			t.Errorf("Expected 1 file content, got %d", len(contents))
		}

		content := contents[testFile]
		if content == "" {
			t.Error("File content is empty")
		}

		// Verify content contains expected elements
		if !containsString(content, "package main") {
			t.Error("Content should contain 'package main'")
		}
		if !containsString(content, "func main()") {
			t.Error("Content should contain 'func main()'")
		}
	})

	// Test symbol analysis with real parser
	t.Run("symbol_analysis_real", func(t *testing.T) {
		// Create mock diff that affects the test file
		mockDiff := `diff --git a/test.go b/test.go
index 1234567..abcdefg 100644
--- a/test.go
+++ b/test.go
@@ -4,6 +4,7 @@ import "fmt"
 
 func main() {
 	fmt.Println("Hello, World!")
+	greet("Test")
 }
 
 func greet(name string) string:`

		fileContents := map[string]string{
			testFile: testContent,
		}

		symbolTool := registry.Get(tools.ToolNameSymbolContext)
		result, err := symbolTool.Execute(map[string]any{
			"diff":             mockDiff,
			"file_contents":    fileContents,
			"primary_language": "go",
		})

		if err != nil {
			t.Fatalf("Symbol analysis failed: %v", err)
		}

		if result == "" {
			t.Error("Symbol analysis returned empty result")
		}

		// The result should contain information about the symbols
		t.Logf("Symbol analysis result: %s", result)
	})

	// Test language validation with real parser registry
	t.Run("validate_language_real", func(t *testing.T) {
		files := []string{testFile}
		lang, err := agent.validateAndDetectLanguage(files)
		if err != nil {
			t.Fatalf("Language validation failed: %v", err)
		}

		if lang != "go" {
			t.Errorf("Expected language 'go', got '%s'", lang)
		}
	})
}

func TestContextGatheringWithRealComponents(t *testing.T) {
	// This test verifies that the context gathering works with real components
	// but uses mock data to avoid git dependencies
	
	// Create a temporary test directory
	tempDir := t.TempDir()
	
	registry := tools.NewRegistry()
	parserRegistry := tools.NewParserRegistry()
	
	// Register real tools
	registry.Register(tools.ToolNameReadFile, &tools.ReadFileTool{})
	registry.Register(tools.ToolNameSymbolContext, tools.NewSymbolContextTool(tempDir, parserRegistry))

	mockProvider := &mockLLMProvider{
		response: "Mock review response",
	}

	cfg := &config.Config{}
	agent := NewCodeReviewAgent(mockProvider, registry, cfg, parserRegistry)
	testFile := filepath.Join(tempDir, "example.go")
	testContent := `package example

import "fmt"

type User struct {
	Name string
	Age  int
}

func (u *User) Greet() string {
	return fmt.Sprintf("Hello, I'm %s", u.Name)
}

func NewUser(name string, age int) *User {
	return &User{Name: name, Age: age}
}
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock diff that shows changes to the file
	mockDiff := `diff --git a/example.go b/example.go
index 1234567..abcdefg 100644
--- a/example.go
+++ b/example.go
@@ -8,7 +8,8 @@ type User struct {
 }
 
 func (u *User) Greet() string {
-	return fmt.Sprintf("Hello, I'm %s", u.Name)
+	greeting := fmt.Sprintf("Hello, I'm %s", u.Name)
+	return greeting
 }
 
 func NewUser(name string, age int) *User {`

	changedFiles := []string{testFile}
	primaryLanguage := "go"

	// Test the complete context gathering flow
	context, err := agent.GatherEnhancedContext(mockDiff, changedFiles, primaryLanguage)
	if err != nil {
		t.Fatalf("Context gathering failed: %v", err)
	}

	// Verify all context components are populated
	if context.Diff != mockDiff {
		t.Error("Diff not preserved in context")
	}

	if len(context.ChangedFiles) != 1 || context.ChangedFiles[0] != testFile {
		t.Error("Changed files not preserved correctly")
	}

	if len(context.FileContents) != 1 {
		t.Error("File contents not populated")
	}

	fileContent := context.FileContents[testFile]
	if fileContent == "" {
		t.Error("File content is empty")
	}

	// Verify the file content contains expected elements
	if !containsString(fileContent, "type User struct") {
		t.Error("File content should contain struct definition")
	}

	if !containsString(fileContent, "func (u *User) Greet()") {
		t.Error("File content should contain method definition")
	}

	if context.SymbolAnalysis == "" {
		t.Error("Symbol analysis not populated")
	}

	// Log the results for manual inspection
	t.Logf("Context gathered successfully:")
	t.Logf("- Changed files: %v", context.ChangedFiles)
	t.Logf("- File contents length: %d", len(context.FileContents[testFile]))
	t.Logf("- Symbol analysis: %s", context.SymbolAnalysis)
}