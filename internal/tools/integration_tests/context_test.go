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
	projectRoot := wd

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
			diffFile: filepath.Join(projectRoot, "diff/go_func_decl.diff"),
			changedFiles: []string{
				filepath.Join(projectRoot, "code_samples/go/utils/user_service.go"),
			},
			validate: func(t *testing.T, result types.DiffData) {
				assert.NotEmpty(t, result.AbsolutePath, "AbsolutePath should not be empty")
				assert.NotEmpty(t, result.Diff, "Diff should not be empty")
				assert.NotEmpty(t, result.DiffContext, "DiffContext should not be empty")

				expectedPath := filepath.Join(projectRoot, "code_samples/go/utils/user_service.go")
				assert.Equal(t, expectedPath, result.AbsolutePath)

				assert.Contains(t, result.Diff, "GetUserByID", "Diff should contain GetUserByID method call")
				assert.Contains(t, result.Diff, "auditLogger", "Diff should contain auditLogger call")

				assert.Contains(t, result.DiffContext, "func (s *UserService) GetUser",
					"DiffContext should contain the function signature")
				assert.Contains(t, result.DiffContext, "GetUserByID",
					"DiffContext should contain the method call from the changed lines")

				assert.True(t, len(result.AffectedSymbols) > 0, "Should have at least one affected symbol")
				assert.Equal(t, "GetUser", result.AffectedSymbols[0].Symbol.Name, "The first affected symbol name should be 'GetUser'")

				getUserSymbol := result.AffectedSymbols[0]
				assert.NotEmpty(t, getUserSymbol.Snippets, "GetUser symbol should have snippets")
				assert.Contains(t, getUserSymbol.Snippets, "=== Symbol: GetUser (Package: utils) ===",
					"Snippet should contain symbol header")

				assert.True(t,
					strings.Contains(getUserSymbol.Snippets, "Definition in"), "Snippet should contain definition information")

				assert.True(t, strings.Contains(getUserSymbol.Snippets, "Usage in"), "Snippet should contain usage information")

				assert.Contains(t, getUserSymbol.Snippets, "GetUser(ctx context", "Snippet should contain the GetUser function code")

			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parserRegistry := tools.NewParserRegistry()
			symbolContextTool := tools.NewSymbolContextTool(projectRoot, parserRegistry)

			diff, err := os.ReadFile(tc.diffFile)
			require.NoError(t, err, "Failed to read diff file")

			diffData := types.DiffData{
				AbsolutePath: tc.changedFiles[0],
				Diff:         string(diff),
			}

			args := make(map[string]any)
			args["diffData"] = diffData
			args["primaryLanguage"] = tc.language

			result, err := symbolContextTool.Execute(args)
			require.NoError(t, err, "Tool execution should not fail")

			resultData, ok := result.(types.DiffData)
			require.True(t, ok, "Tool result should be of type types.DiffData")

			tc.validate(t, resultData)
		})
	}
}
