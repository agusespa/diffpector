package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatherEnhancedContext(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Join(wd, "../../..")

	testCases := []struct {
		name         string
		language     string
		diffFile     string
		changedFiles []string
		validate     func(t *testing.T, result types.DiffData)
	}{
		{
			name:     "Go function modification",
			language: "go",
			diffFile: filepath.Join(projectRoot, "internal/tools/integration_tests/diff/go_func_decl.diff"),
			changedFiles: []string{
				filepath.Join(projectRoot, "internal/tools/integration_tests/code_samples/go/utils/user_service.go"),
			},
			validate: func(t *testing.T, result types.DiffData) {
				// ALWAYS print debug information first
				t.Logf("=== DEBUG INFO ===")
				t.Logf("AbsolutePath: %s", result.AbsolutePath)
				t.Logf("Diff length: %d", len(result.Diff))
				t.Logf("DiffContext length: %d", len(result.DiffContext))
				t.Logf("DiffContext content: %q", result.DiffContext)
				t.Logf("Number of affected symbols: %d", len(result.AffectedSymbols))

				for i, symbolUsage := range result.AffectedSymbols {
					t.Logf("Symbol %d: Name=%s, Type=%s, Package=%s, StartLine=%d, EndLine=%d",
						i, symbolUsage.Symbol.Name, symbolUsage.Symbol.Type, symbolUsage.Symbol.Package,
						symbolUsage.Symbol.StartLine, symbolUsage.Symbol.EndLine)
					t.Logf("  Snippets: %v", symbolUsage.Snippets)
				}
				t.Logf("=== END DEBUG ===")

				// Basic structure validation
				assert.NotEmpty(t, result.AbsolutePath, "AbsolutePath should not be empty")
				assert.NotEmpty(t, result.Diff, "Diff should not be empty")
				assert.NotEmpty(t, result.DiffContext, "DiffContext should not be empty")

				// Verify the absolute path matches expected
				expectedPath := filepath.Join(projectRoot, "internal/tools/integration_tests/code_samples/go/utils/user_service.go")
				assert.Equal(t, expectedPath, result.AbsolutePath)

				// Verify diff contains expected changes
				assert.Contains(t, result.Diff, "GetUserByID", "Diff should contain GetUserByID method call")
				assert.Contains(t, result.Diff, "auditLogger", "Diff should contain auditLogger call")

				// CRITICAL ISSUE: DiffContext should contain the function signature and the changed lines
				// Based on your original test expectation, it should show the whole GetUser function
				assert.Contains(t, result.DiffContext, "func (s *UserService) GetUser",
					"DiffContext should contain the function signature - this indicates utils.GetDiffContext() is not working correctly")

				// The DiffContext should contain the actual changes from the diff
				assert.Contains(t, result.DiffContext, "GetUserByID",
					"DiffContext should contain the method call from the changed lines")

				// Check for the malformed content issue we observed
				repeatedAuditLoggerCount := strings.Count(result.DiffContext, "s.auditLogger(\"System admin accessed by ID lookup.\")")
				assert.LessOrEqual(t, repeatedAuditLoggerCount, 1,
					"DiffContext appears to contain repeated content - this indicates a bug in context extraction")

				// CRITICAL ISSUE: Should have at least one function declaration symbol
				assert.True(t, len(result.AffectedSymbols) > 0, "Should have at least one affected symbol")

				// Look for function declarations specifically
				hasFunctionDeclaration := false
				hasGetUserFunction := false
				symbolsWithSnippets := 0

				for _, symbolUsage := range result.AffectedSymbols {
					// Check if any symbol is a function declaration
					if strings.Contains(symbolUsage.Symbol.Type, "func_decl") ||
						strings.Contains(symbolUsage.Symbol.Type, "function_decl") {
						hasFunctionDeclaration = true
					}

					// Check specifically for GetUser function
					if symbolUsage.Symbol.Name == "GetUser" {
						hasGetUserFunction = true
					}

					// Check if snippets are populated
					if len(symbolUsage.Snippets) > 0 {
						symbolsWithSnippets++
					}
				}

				// These are the key issues your integration test should catch:
				assert.True(t, hasFunctionDeclaration,
					"Should have at least one function declaration symbol - parser may not be detecting function declarations correctly")

				assert.True(t, hasGetUserFunction,
					"Should have GetUser function in affected symbols - the modified function should be detected")

				assert.Greater(t, symbolsWithSnippets, 0,
					"At least some symbols should have code snippets - all snippets are currently empty")

				// Additional validation: symbols should have reasonable line numbers
				for _, symbolUsage := range result.AffectedSymbols {
					assert.Greater(t, symbolUsage.Symbol.StartLine, 0,
						"Symbol StartLine should be positive (symbol: %s)", symbolUsage.Symbol.Name)
					assert.GreaterOrEqual(t, symbolUsage.Symbol.EndLine, symbolUsage.Symbol.StartLine,
						"Symbol EndLine should be >= StartLine (symbol: %s)", symbolUsage.Symbol.Name)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read the diff file
			diff, err := os.ReadFile(tc.diffFile)
			require.NoError(t, err, "Failed to read diff file")

			// Create the initial DiffData
			diffData := types.DiffData{
				AbsolutePath: tc.changedFiles[0],
				Diff:         string(diff),
			}

			// Verify the test file exists
			_, err = os.Stat(diffData.AbsolutePath)
			require.NoError(t, err, "Test file should exist: %s", diffData.AbsolutePath)

			// Create the tool and execute
			parserRegistry := tools.NewParserRegistry()
			symbolContextTool := tools.NewSymbolContextTool(projectRoot, parserRegistry)

			args := make(map[string]any)
			args["diffData"] = diffData
			args["primaryLanguage"] = tc.language

			result, err := symbolContextTool.Execute(args)
			require.NoError(t, err, "Tool execution should not fail")

			// Run the validation function
			tc.validate(t, result)
		})
	}
}

// Helper function to truncate strings for debug output
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Helper test to verify the test files exist and are readable
func TestTestFilesExist(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Join(wd, "../../..")

	testFiles := []string{
		filepath.Join(projectRoot, "internal/tools/integration_tests/diff/go_func_decl.diff"),
		filepath.Join(projectRoot, "internal/tools/integration_tests/code_samples/go/utils/user_service.go"),
	}

	for _, file := range testFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			info, err := os.Stat(file)
			require.NoError(t, err, "File should exist: %s", file)
			assert.False(t, info.IsDir(), "Should be a file, not directory: %s", file)
			assert.True(t, info.Size() > 0, "File should not be empty: %s", file)

			// Try to read the file
			content, err := os.ReadFile(file)
			require.NoError(t, err, "Should be able to read file: %s", file)
			assert.True(t, len(content) > 0, "File content should not be empty: %s", file)
		})
	}
}

// Debug test to examine the actual content we're working with
func TestDebugFileContents(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Join(wd, "../../..")

	// Read the Go file
	goFile := filepath.Join(projectRoot, "internal/tools/integration_tests/code_samples/go/utils/user_service.go")
	goContent, err := os.ReadFile(goFile)
	require.NoError(t, err)

	t.Logf("=== GO FILE CONTENT (first 500 chars) ===")
	t.Logf("%s", truncateString(string(goContent), 500))

	// Look for GetUser function specifically
	lines := strings.Split(string(goContent), "\n")
	for i, line := range lines {
		if strings.Contains(line, "func (s *UserService) GetUser") {
			t.Logf("Found GetUser function at line %d: %s", i+1, line)
			// Show some context around it
			start := max(0, i-2)
			end := min(len(lines), i+10)
			t.Logf("Context around GetUser function:")
			for j := start; j < end; j++ {
				marker := "  "
				if j == i {
					marker = ">>>"
				}
				t.Logf("%s %d: %s", marker, j+1, lines[j])
			}
			break
		}
	}

	// Read the diff file
	diffFile := filepath.Join(projectRoot, "internal/tools/integration_tests/diff/go_func_decl.diff")
	diffContent, err := os.ReadFile(diffFile)
	require.NoError(t, err)

	t.Logf("=== DIFF FILE CONTENT ===")
	t.Logf("%s", string(diffContent))

	// Test the parser directly
	parserRegistry := tools.NewParserRegistry()
	symbols, err := parserRegistry.ParseFile(goFile, goContent)
	require.NoError(t, err)

	t.Logf("=== PARSED SYMBOLS ===")
	for i, symbol := range symbols {
		t.Logf("Symbol %d: Name=%s, Type=%s, Package=%s, StartLine=%d, EndLine=%d, FilePath=%s",
			i, symbol.Name, symbol.Type, symbol.Package, symbol.StartLine, symbol.EndLine, symbol.FilePath)
	}
}
