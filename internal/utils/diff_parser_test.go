package utils

import (
	"maps"
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

func TestParseGitDiffForAddedLines(t *testing.T) {
	tests := []struct {
		name        string
		diffContent string
		want        map[int]bool
	}{
		{
			diffContent: `
@@ -1,0 +1,1 @@
+New line at the beginning
`,
			want: map[int]bool{1: true},
		},
		{
			name: "Multiple_Consecutive_Additions",
			diffContent: `
@@ -2,1 +2,3 @@
 Context line 2
+Added line 3
+Added line 4
 Context line 3
`,
			want: map[int]bool{3: true, 4: true},
		},
		{
			name: "Addition_After_Deletion",
			diffContent: `
@@ -1,3 +1,3 @@
 Context line 1
-Deleted line 2
+Added line 2
 Context line 3
`,
			want: map[int]bool{2: true},
		},
		{
			name: "Multiple_Hunks",
			diffContent: `
@@ -2,2 +2,3 @@
 Context line 2
-Deleted line
+Added line 3
 Context line 4
@@ -5,2 +5,4 @@
 Context line 5
+Added line 6
+Added line 7
 Context line 8
`,
			want: map[int]bool{3: true, 6: true, 7: true},
		},
		{
			name: "No_Additions",
			diffContent: `
@@ -1,3 +1,3 @@
 Context 1
-Deleted line 2
 Context 3
`,
			want: map[int]bool{},
		},
		{
			name:        "Empty_Diff",
			diffContent: ``,
			want:        map[int]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDiffChangedLines(tt.diffContent)

			if !maps.Equal(got, tt.want) {
				t.Errorf("parseGitDiffForAddedLines() got = %v, want %v", got, tt.want)
			}
		})
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
		"",                         // line 2 (empty)
		"import \"fmt\"",           // line 3
		"",                         // line 4 (empty)
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
			name: "Symbol at end of file",
			symbol: types.Symbol{
				Name:      "closing_brace",
				StartLine: 8,
				EndLine:   8,
			},
			expected: "}",
		},
		{
			name: "Symbol includes empty lines",
			symbol: types.Symbol{
				Name:      "with_empty_lines",
				StartLine: 1,
				EndLine:   4,
			},
			expected: `package main

import "fmt"
`,
		},
		{
			name: "Symbol beyond file bounds (start and end)",
			symbol: types.Symbol{
				Name:      "invalid",
				StartLine: 10,
				EndLine:   15,
			},
			expected: "",
		},
		{
			name: "Symbol starts beyond bounds but end within",
			symbol: types.Symbol{
				Name:      "invalid",
				StartLine: 20,
				EndLine:   25,
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
		{
			name: "Symbol with zero start line",
			symbol: types.Symbol{
				Name:      "zero_start",
				StartLine: 0,
				EndLine:   2,
			},
			expected: `package main
`,
		},
		{
			name: "Symbol with negative start line",
			symbol: types.Symbol{
				Name:      "negative_start",
				StartLine: -5,
				EndLine:   2,
			},
			expected: `package main
`,
		},
		{
			name: "Symbol partially beyond end (start valid, end beyond)",
			symbol: types.Symbol{
				Name:      "partial_overflow",
				StartLine: 7,
				EndLine:   15,
			},
			expected: `	fmt.Println("World")
}`,
		},
		{
			name: "Entire file extraction",
			symbol: types.Symbol{
				Name:      "entire_file",
				StartLine: 1,
				EndLine:   8,
			},
			expected: `package main

import "fmt"

func main() {
	fmt.Println("Hello")
	fmt.Println("World")
}`,
		},
		{
			name: "Symbol with start line equals file length + 1",
			symbol: types.Symbol{
				Name:      "start_equals_length_plus_one",
				StartLine: 9,
				EndLine:   10,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSymbolContent(tt.symbol, fileLines)
			if result != tt.expected {
				t.Errorf("Expected: %q, Got: %q", tt.expected, result)
			}
		})
	}
}

func TestExtractSymbolContentEmptyFile(t *testing.T) {
	emptyFileLines := []string{}

	tests := []struct {
		name     string
		symbol   types.Symbol
		expected string
	}{
		{
			name: "Symbol from empty file",
			symbol: types.Symbol{
				Name:      "from_empty",
				StartLine: 1,
				EndLine:   3,
			},
			expected: "",
		},
		{
			name: "Zero lines from empty file",
			symbol: types.Symbol{
				Name:      "zero_from_empty",
				StartLine: 0,
				EndLine:   0,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSymbolContent(tt.symbol, emptyFileLines)
			if result != tt.expected {
				t.Errorf("Expected: %q, Got: %q", tt.expected, result)
			}
		})
	}
}

func TestGetDiffContext(t *testing.T) {
	changedFileContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello")
	fmt.Println("New Line")
	fmt.Println("World")
}

func helper() {
	fmt.Println("Helper function")
	// New line added, shifting 'return'
	fmt.Println("New final line")
	return
}

type User struct {
	Name     string
	Age      int
	IsActive bool
}`

	fileContentBytes := []byte(changedFileContent)

	allSymbols := []types.Symbol{
		{
			Name:      "main",
			Package:   "main",
			FilePath:  "test.go",
			StartLine: 5,
			EndLine:   9,
		},
		{
			Name:      "helper",
			Package:   "main",
			FilePath:  "test.go",
			StartLine: 11,
			EndLine:   16,
		},
		{
			Name:      "User",
			Package:   "main",
			FilePath:  "test.go",
			StartLine: 18,
			EndLine:   22,
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
			name: "Change at Function Start Boundary",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -11,1 +11,1 @@
-func helper() {
+func helper() {
`,
			},
			expectedContext: `func helper() {
	fmt.Println("Helper function")
	// New line added, shifting 'return'
	fmt.Println("New final line")
	return
}`,
			expectedSymbols: []string{"helper"},
			wantErr:         false,
		},
		{
			name: "Change at Function End Boundary",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -16,1 +16,1 @@
-}
+}
`,
			},
			expectedContext: `func helper() {
	fmt.Println("Helper function")
	// New line added, shifting 'return'
	fmt.Println("New final line")
	return
}`,
			expectedSymbols: []string{"helper"},
			wantErr:         false,
		},
		{
			name: "Change in Middle of Function",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -6,1 +6,1 @@
-	fmt.Println("Hello")
+	fmt.Println("Hello World")
`,
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
			name: "Multiple Functions Affected",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -6,1 +6,1 @@
-	fmt.Println("Hello")
+	fmt.Println("Hello World")
@@ -12,1 +12,1 @@
-	fmt.Println("Helper function")
+	fmt.Println("Updated Helper function")
`,
			},
			expectedContext: `func main() {
	fmt.Println("Hello")
	fmt.Println("New Line")
	fmt.Println("World")
}

func helper() {
	fmt.Println("Helper function")
	// New line added, shifting 'return'
	fmt.Println("New final line")
	return
}`,
			expectedSymbols: []string{"main", "helper"},
			wantErr:         false,
		},
		{
			name: "Change in Struct Field",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -19,1 +19,1 @@
-	Name     string
+	FullName string
`,
			},
			expectedContext: `type User struct {
	Name     string
	Age      int
	IsActive bool
}`,
			expectedSymbols: []string{"User"},
			wantErr:         false,
		},
		{
			name: "Change Outside All Symbols",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -3,1 +3,1 @@
-import "fmt"
+import ("fmt"; "os")
`,
			},
			expectedContext: "",
			expectedSymbols: []string{},
			wantErr:         false,
		},
		{
			name: "Added Lines Only",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -7,0 +7,1 @@
+	fmt.Println("Added line")
`,
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
			name: "Empty Diff",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff:         "",
			},
			expectedContext: "",
			expectedSymbols: []string{},
			wantErr:         false,
		},
		{
			name: "Malformed Diff",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `invalid diff content
not a real diff`,
			},
			expectedContext: "",
			expectedSymbols: []string{},
			wantErr:         false,
		},
		{
			name: "Change at Package Declaration (Outside Symbols)",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -1,1 +1,1 @@
-package main
+package main // with comment
`,
			},
			expectedContext: "",
			expectedSymbols: []string{},
			wantErr:         false,
		},
		{
			name: "Large Hunk Affecting Multiple Symbols",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -5,10 +5,12 @@
 func main() {
 	fmt.Println("Hello")
+	fmt.Println("Added line 1")
 	fmt.Println("New Line")
 	fmt.Println("World")
 }
 
 func helper() {
+	fmt.Println("Added line 2")
 	fmt.Println("Helper function")
 	// New line added, shifting 'return'
`,
			},
			expectedContext: `func main() {
	fmt.Println("Hello")
	fmt.Println("New Line")
	fmt.Println("World")
}

func helper() {
	fmt.Println("Helper function")
	// New line added, shifting 'return'
	fmt.Println("New final line")
	return
}`,
			expectedSymbols: []string{"main", "helper"},
			wantErr:         false,
		},
		{
			name: "Change at Symbol Boundary Edge Cases",
			diffData: types.DiffData{
				AbsolutePath: "test.go",
				Diff: `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -9,1 +9,1 @@
-}
+} // end main
@@ -11,1 +11,1 @@
-func helper() {
+func helper() { // start helper
`,
			},
			expectedContext: `func main() {
	fmt.Println("Hello")
	fmt.Println("New Line")
	fmt.Println("World")
}

func helper() {
	fmt.Println("Helper function")
	// New line added, shifting 'return'
	fmt.Println("New final line")
	return
}`,
			expectedSymbols: []string{"main", "helper"},
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetDiffContext(tt.diffData, allSymbols, fileContentBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDiffContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if strings.TrimSpace(result.Context) != strings.TrimSpace(tt.expectedContext) {
				t.Errorf("Context string mismatch. Expected context:\n---\n%s\n---\n\nGot context:\n---\n%s\n---",
					tt.expectedContext, result.Context)
			}

			actualSymbolNames := make([]string, len(result.AffectedSymbols))
			for i, symbol := range result.AffectedSymbols {
				actualSymbolNames[i] = symbol.Symbol.Name
			}

			if len(actualSymbolNames) != len(tt.expectedSymbols) {
				t.Fatalf("Expected %d affected symbols, got %d. Expected: %v, Got: %v. (Check symbol boundary logic for off-by-one errors. The change is exactly on StartLine %d.)",
					len(tt.expectedSymbols), len(actualSymbolNames), tt.expectedSymbols, actualSymbolNames, allSymbols[1].StartLine)
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
