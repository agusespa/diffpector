package tests

import (
	"os"
	"path/filepath"
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
				assert.NotEmpty(t, result.AbsolutePath, "AbsolutePath should not be empty")
				assert.NotEmpty(t, result.Diff, "Diff should not be empty")
				assert.NotEmpty(t, result.DiffContext, "DiffContext should not be empty")

				expectedPath := filepath.Join(projectRoot, "internal/tools/integration_tests/code_samples/go/utils/user_service.go")
				assert.Equal(t, expectedPath, result.AbsolutePath)

				assert.Contains(t, result.Diff, "GetUserByID", "Diff should contain GetUserByID method call")
				assert.Contains(t, result.Diff, "auditLogger", "Diff should contain auditLogger call")

				assert.Contains(t, result.DiffContext, "func (s *UserService) GetUser",
					"DiffContext should contain the function signature")
				assert.Contains(t, result.DiffContext, "GetUserByID",
					"DiffContext should contain the method call from the changed lines")

				assert.True(t, len(result.AffectedSymbols) > 0, "Should have at least one affected symbol")
				assert.Equal(t, "GetUser", result.AffectedSymbols[0].Symbol.Name, "The first affected symbol name should be 'GetUser'")

				// TODO test affected symbols snippets
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

			tc.validate(t, result)
		})
	}
}
