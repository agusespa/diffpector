package utils

import (
	"os"
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

func TestGetDiffContext(t *testing.T) {
	testFileContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello")
	fmt.Println("New Line")
	fmt.Println("World")
}

func helper() {
	fmt.Println("Helper function")
	fmt.Println("Added line")
	return
}

type User struct {
	Name  string
	Age   int
	Email string
}`

	tmpFile, err := os.CreateTemp("", "test*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.WriteString(testFileContent); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	allSymbols := []types.Symbol{
		{
			Name:      "main",
			Package:   "main",
			FilePath:  tmpFile.Name(),
			StartLine: 5,
			EndLine:   9,
		},
		{
			Name:      "helper",
			Package:   "main",
			FilePath:  tmpFile.Name(),
			StartLine: 11,
			EndLine:   15,
		},
		{
			Name:      "User",
			Package:   "main",
			FilePath:  tmpFile.Name(),
			StartLine: 17,
			EndLine:   21,
		},
	}

	tests := []struct {
		name            string
		diffData        types.DiffData
		expectedContext string
		expectedSymbols []string
		wantErr         bool
	}{
		{
			name: "Change in main function",
			diffData: types.DiffData{
				AbsolutePath: tmpFile.Name(),
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -5,6 +5,6 @@ import "fmt"
 func main() {
 	fmt.Println("Hello")
-	fmt.Println("Old Line")
+	fmt.Println("New Line")
 	fmt.Println("World")
 }`,
			},
			expectedContext: `func main() {
	fmt.Println("Hello")
	fmt.Println("New Line")
	fmt.Println("World")
}`,
			expectedSymbols: []string{"main"},
			wantErr:         false,
		},
		{
			name: "Change in helper function",
			diffData: types.DiffData{
				AbsolutePath: tmpFile.Name(),
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -12,6 +12,6 @@ func main() {
 func helper() {
 	fmt.Println("Helper function")
-	fmt.Println("Old line")
+	fmt.Println("Added line")
 	return
 }`,
			},
			expectedContext: `func helper() {
	fmt.Println("Helper function")
	fmt.Println("Added line")
	return
}`,
			expectedSymbols: []string{"helper"},
			wantErr:         false,
		},
		{
			name: "Multiple symbols changed",
			diffData: types.DiffData{
				AbsolutePath: tmpFile.Name(),
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -5,6 +5,6 @@ import "fmt"
 func main() {
 	fmt.Println("Hello")
-	fmt.Println("Old Line")
+	fmt.Println("New Line")
 	fmt.Println("World")
 }
@@ -18,5 +18,5 @@ type User struct {
 	Name  string
 	Age   int
-	// Old field
+	Email string
 }`,
			},
			expectedContext: `func main() {
	fmt.Println("Hello")
	fmt.Println("New Line")
	fmt.Println("World")
}

type User struct {
	Name  string
	Age   int
	Email string
}`,
			expectedSymbols: []string{"main", "User"},
			wantErr:         false,
		},
		{
			name: "No changes for this file",
			diffData: types.DiffData{
				AbsolutePath: tmpFile.Name(),
				Diff:         ``,
			},
			expectedContext: "",
			expectedSymbols: []string{},
			wantErr:         false,
		},
		{
			name: "File doesn't exist",
			diffData: types.DiffData{
				AbsolutePath: "/nonexistent/file.go",
				Diff:         "some diff",
			},
			expectedContext: "",
			expectedSymbols: []string{},
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetDiffContext(tt.diffData, allSymbols)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDiffContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if strings.TrimSpace(result.Context) != strings.TrimSpace(tt.expectedContext) {
				t.Errorf("Expected context:\n---\n%s\n---\n\nGot context:\n---\n%s\n---",
					tt.expectedContext, result.Context)
			}

			actualSymbolNames := make([]string, len(result.AffectedSymbols))
			for i, symbol := range result.AffectedSymbols {
				actualSymbolNames[i] = symbol.Symbol.Name
			}

			if len(actualSymbolNames) != len(tt.expectedSymbols) {
				t.Errorf("Expected %d affected symbols, got %d. Expected: %v, Got: %v",
					len(tt.expectedSymbols), len(actualSymbolNames), tt.expectedSymbols, actualSymbolNames)
				return
			}

			expectedSet := make(map[string]bool)
			for _, name := range tt.expectedSymbols {
				expectedSet[name] = true
			}

			for _, actualName := range actualSymbolNames {
				if !expectedSet[actualName] {
					t.Errorf("Unexpected symbol in results: %s", actualName)
				}
			}
		})
	}
}

func TestGetDiffContextSymbolDetails(t *testing.T) {
	testFileContent := `package main

func test() {
	println("test")
}`

	tmpFile, err := os.CreateTemp("", "test*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.WriteString(testFileContent); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	symbols := []types.Symbol{
		{
			Name:      "test",
			Package:   "main",
			FilePath:  tmpFile.Name(),
			StartLine: 3,
			EndLine:   5,
		},
	}

	diffData := types.DiffData{
		AbsolutePath: tmpFile.Name(),
		Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -2,4 +2,4 @@ package main
 
 func test() {
-	println("old")
+	println("test")
 }`,
	}

	result, err := GetDiffContext(diffData, symbols)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify we got the expected symbol back with all its details
	if len(result.AffectedSymbols) != 1 {
		t.Fatalf("Expected 1 affected symbol, got %d", len(result.AffectedSymbols))
	}

	symbol := result.AffectedSymbols[0]
	if symbol.Symbol.Name != "test" {
		t.Errorf("Expected symbol name 'test', got '%s'", symbol.Symbol.Name)
	}
	if symbol.Symbol.Package != "main" {
		t.Errorf("Expected package 'main', got '%s'", symbol.Symbol.Package)
	}
	if symbol.Symbol.StartLine != 3 {
		t.Errorf("Expected start line 3, got %d", symbol.Symbol.StartLine)
	}
	if symbol.Symbol.EndLine != 5 {
		t.Errorf("Expected end line 5, got %d", symbol.Symbol.EndLine)
	}
}

func TestContainsChangedLines(t *testing.T) {
	symbol := types.Symbol{
		Name:      "testFunc",
		StartLine: 5,
		EndLine:   10,
	}

	tests := []struct {
		name         string
		changedLines map[int]bool
		expected     bool
	}{
		{
			name:         "Contains changed line",
			changedLines: map[int]bool{7: true},
			expected:     true,
		},
		{
			name:         "Contains multiple changed lines",
			changedLines: map[int]bool{5: true, 8: true},
			expected:     true,
		},
		{
			name:         "No changed lines in symbol",
			changedLines: map[int]bool{2: true, 15: true},
			expected:     false,
		},
		{
			name:         "Changed line at start boundary",
			changedLines: map[int]bool{5: true},
			expected:     true,
		},
		{
			name:         "Changed line at end boundary",
			changedLines: map[int]bool{10: true},
			expected:     true,
		},
		{
			name:         "Empty changed lines",
			changedLines: map[int]bool{},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsChangedLines(symbol, tt.changedLines)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractSymbolContent(t *testing.T) {
	fileLines := []string{
		"package main",             // line 1
		"",                         // line 2
		"import \"fmt\"",           // line 3
		"",                         // line 4
		"func main() {",            // line 5
		"\tfmt.Println(\"Hello\")", // line 6
		"\tfmt.Println(\"World\")", // line 7
		"}",                        // line 8
	}

	tests := []struct {
		name     string
		symbol   types.Symbol
		expected string
	}{
		{
			name: "Normal symbol extraction",
			symbol: types.Symbol{
				Name:      "main",
				StartLine: 5,
				EndLine:   8,
			},
			expected: `func main() {
	fmt.Println("Hello")
	fmt.Println("World")
}`,
		},
		{
			name: "Single line symbol",
			symbol: types.Symbol{
				Name:      "import",
				StartLine: 3,
				EndLine:   3,
			},
			expected: `import "fmt"`,
		},
		{
			name: "Symbol at start of file",
			symbol: types.Symbol{
				Name:      "package",
				StartLine: 1,
				EndLine:   1,
			},
			expected: "package main",
		},
		{
			name: "Symbol beyond file bounds",
			symbol: types.Symbol{
				Name:      "invalid",
				StartLine: 10,
				EndLine:   15,
			},
			expected: "",
		},
		{
			name: "Symbol with start > end",
			symbol: types.Symbol{
				Name:      "invalid",
				StartLine: 8,
				EndLine:   5,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSymbolContent(tt.symbol, fileLines)
			if result != tt.expected {
				t.Errorf("Expected:\n%s\n\nGot:\n%s", tt.expected, result)
			}
		})
	}
}
