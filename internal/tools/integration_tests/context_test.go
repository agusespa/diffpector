package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestGatherEnhancedContext(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	projectRoot := filepath.Join(wd, "../../..")

	testCases := []struct {
		name           string
		language       string
		diffFile       string
		changedFiles   []string
		expectedResult *types.DiffData
	}{
		{
			name:         "Go utility",
			language:     "go",
			diffFile:     filepath.Join(projectRoot, "internal/tools/integration_tests/diff/go_utility.diff"),
			changedFiles: []string{filepath.Join(projectRoot, "internal/tools/integration_tests/code_samples/go/utils/database.go")},
			expectedResult: &types.DiffData{
				AbsolutePath: "/Users/agusespa/Code/diffpector/internal/tools/integration_tests/code_samples/go/utils/database.go",
				Diff: `--- a/internal/tools/integration_tests/code_samples/go/utils/database.go
+++ b/internal/tools/integration_tests/code_samples/go/utils/database.go
@@ -14,11 +14,3 @@
 
 	// Simulate a database query
 	user := &DBUser{ // Added a comment
-		ID:        id,
-		Name:      "John Doe",
-		Email:     "john.doe@example.com",
-	}
-
-	return user
+		ID:    id,
+		Name:  "Jane Doe",
+		Email: "jane.doe@example.com",
+	}
+	return user
 }`,
				DiffContext: "\tuser := &DBUser{ // Added a comment",
				AffectedSymbols: []types.SymbolUsage{
					{
						Symbol: types.Symbol{Name: "user", Type: "var_usage", Package: "utils", FilePath: "/Users/agusespa/Code/diffpector/internal/tools/integration_tests/code_samples/go/utils/database.go", StartLine: 21, EndLine: 21},
					},
				},
			},
		},
		{
			name:     "Go feature",
			language: "go",
			diffFile: filepath.Join(projectRoot, "internal/tools/integration_tests/diff/go_feature.diff"),
			changedFiles: []string{
				filepath.Join(projectRoot, "internal/tools/integration_tests/code_samples/go/utils/user_service.go"),
			},
			expectedResult: &types.DiffData{
				AbsolutePath: filepath.Join(projectRoot, "internal/tools/integration_tests/code_samples/go/utils/user_service.go"),
				Diff: `
--- a/internal/tools/integration_tests/code_samples/go/utils/user_service.go
+++ b/internal/tools/integration_tests/code_samples/go/utils/user_service.go
@@ -14,3 +14,8 @@
 func (s *UserService) GetUser(id string) (*User, error) {
 	return s.userRepo.GetUserByID(id)
 }
+
+// GetUserAsAdmin gets a user by their ID and returns it with an admin note.
+func (s *UserService) GetUserAsAdmin(id string) (*User, error) {
+	return s.userRepo.GetUserByID(id)
+}
`,
				DiffContext: "",
				AffectedSymbols: []types.SymbolUsage{
					{
						Symbol: types.Symbol{
							Name:      "GetUserAsAdmin",
							Type:      "func_decl",
							Package:   "utils",
							FilePath:  filepath.Join(projectRoot, "internal/tools/integration_tests/code_samples/go/utils/user_service.go"),
							StartLine: 18, // adjust to the actual line number of the function in the file
							EndLine:   21, // adjust based on where the function ends
						},
						Snippets: []string{
							"func (s *UserService) GetUserAsAdmin(id string) (*User, error) {",
							"\treturn s.userRepo.GetUserByID(id)",
							"}",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff, err := os.ReadFile(tc.diffFile)
			assert.NoError(t, err)

			diffData := types.DiffData{
				AbsolutePath: tc.changedFiles[0],
				Diff:         string(diff),
			}

			parserRegistry := tools.NewParserRegistry()
			symbolContextTool := tools.NewSymbolContextTool(projectRoot, parserRegistry)

			args := make(map[string]any)
			args["diffData"] = diffData
			args["primaryLanguage"] = tc.language

			result, err := symbolContextTool.Execute(args)
			assert.NoError(t, err)

			assert.Equal(t, *tc.expectedResult, result)
		})
	}
}
