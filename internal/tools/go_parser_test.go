package tools

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestParseFile_BasicDeclarationsAndUsages(t *testing.T) {
	gp, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	src := []byte(`
		package main

		import "fmt"

		const Pi = 3.14

		var globalVar int

		type MyStruct struct {
			Field int
		}

		func (m *MyStruct) Method(x int) int {
			return m.Field + x
		}

		func Foo(x int) int {
			return x + 1
		}

		func main() {
			val := Foo(42)
			s := MyStruct{Field: 10}
			fmt.Println(s.Method(val))
			_ = globalVar
		}
	`)

	tmpFile := filepath.Join(os.TempDir(), "test.go")
	if err := os.WriteFile(tmpFile, src, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	if err != nil {
		t.Fatalf("failed to create GoParser: %v", err)
	}

	symbols, err := gp.ParseFile(tmpFile, src)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(symbols) == 0 {
		t.Fatalf("expected symbols, got none")
	}

	got := map[string][]string{}
	for _, s := range symbols {
		got[s.Type] = append(got[s.Type], s.Name)
	}

	// === Declarations ===
	if !slices.Contains(got["func_decl"], "Foo") {
		t.Errorf("expected func_decl to contain Foo, got %v", got["func_decl"])
	}
	if !slices.Contains(got["func_decl"], "main") {
		t.Errorf("expected func_decl to contain main, got %v", got["func_decl"])
	}
	if !slices.Contains(got["method_decl"], "Method") {
		t.Errorf("expected method_decl to contain Method, got %v", got["method_decl"])
	}
	if !slices.Contains(got["type_decl"], "MyStruct") {
		t.Errorf("expected type_decl to contain MyStruct, got %v", got["type_decl"])
	}
	if !slices.Contains(got["const_decl"], "Pi") {
		t.Errorf("expected const_decl to contain Pi, got %v", got["const_decl"])
	}
	if !slices.Contains(got["var_decl"], "globalVar") {
		t.Errorf("expected var_decl to contain globalVar, got %v", got["var_decl"])
	}
	if !slices.Contains(got["field_decl"], "Field") {
		t.Errorf("expected field_decl to contain Field, got %v", got["field_decl"])
	}

	// === Usages ===
	if !slices.Contains(got["func_usage"], "Foo") {
		t.Errorf("expected func_usage to contain Foo, got %v", got["func_usage"])
	}
	if !slices.Contains(got["method_usage"], "Method") {
		t.Errorf("expected method_usage to contain Method, got %v", got["method_usage"])
	}
	if !slices.Contains(got["field_usage"], "Field") {
		t.Errorf("expected field_usage to contain Field, got %v", got["field_usage"])
	}
	if !slices.Contains(got["var_usage"], "globalVar") {
		t.Errorf("expected var_usage to contain globalVar, got %v", got["var_usage"])
	}
}

func TestGoParser_ShouldExcludeFile(t *testing.T) {
	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	projectRoot := "/project"

	t.Run("exclude_test_files", func(t *testing.T) {
		testCases := []struct {
			filePath string
			expected bool
			desc     string
		}{
			{"main_test.go", true, "should exclude test files with _test.go suffix"},
			{"internal/parser_test.go", true, "should exclude test files in subdirectories"},
			{"pkg/utils/helper_test.go", true, "should exclude test files in nested paths"},
			{"test_helper.go", false, "should not exclude files that start with 'test' but don't end with '_test.go'"},
			{"testing.go", false, "should not exclude files that contain 'test' but don't end with '_test.go'"},
			{"main.go", false, "should not exclude regular go files"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				result := parser.ShouldExcludeFile(tc.filePath, projectRoot)
				if result != tc.expected {
					t.Errorf("ShouldExcludeFile(%q, %q) = %v, expected %v", tc.filePath, projectRoot, result, tc.expected)
				}
			})
		}
	})

	t.Run("exclude_vendor_directory", func(t *testing.T) {
		testCases := []struct {
			filePath string
			expected bool
			desc     string
		}{
			{"vendor/github.com/pkg/errors/errors.go", true, "should exclude files in vendor directory"},
			{"vendor/golang.org/x/net/http.go", true, "should exclude files in nested vendor paths"},
			{"internal/vendor.go", false, "should not exclude files that contain 'vendor' in name but not in path"},
			{"pkg/vendor_utils.go", false, "should not exclude files with 'vendor' in filename"},
			{"src/vendor/lib.go", true, "should exclude files in vendor subdirectory"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				result := parser.ShouldExcludeFile(tc.filePath, projectRoot)
				if result != tc.expected {
					t.Errorf("ShouldExcludeFile(%q, %q) = %v, expected %v", tc.filePath, projectRoot, result, tc.expected)
				}
			})
		}
	})

	t.Run("exclude_testdata_directory", func(t *testing.T) {
		testCases := []struct {
			filePath string
			expected bool
			desc     string
		}{
			{"testdata/sample.go", true, "should exclude files in testdata directory"},
			{"internal/testdata/fixtures.go", true, "should exclude files in nested testdata paths"},
			{"pkg/parser/testdata/input.go", true, "should exclude files in testdata subdirectories"},
			{"testdata_helper.go", false, "should not exclude files with 'testdata' in filename"},
			{"internal/testdata.go", false, "should not exclude files that contain 'testdata' in name but not in path"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				result := parser.ShouldExcludeFile(tc.filePath, projectRoot)
				if result != tc.expected {
					t.Errorf("ShouldExcludeFile(%q, %q) = %v, expected %v", tc.filePath, projectRoot, result, tc.expected)
				}
			})
		}
	})

	t.Run("exclude_git_directory", func(t *testing.T) {
		testCases := []struct {
			filePath string
			expected bool
			desc     string
		}{
			{".git/config", true, "should exclude files in .git directory"},
			{".git/hooks/pre-commit", true, "should exclude files in .git subdirectories"},
			{"internal/.git/objects/abc123", true, "should exclude files in nested .git paths"},
			{"git_utils.go", false, "should not exclude files with 'git' in filename"},
			{"internal/git.go", false, "should not exclude files that contain 'git' in name but not in path"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				result := parser.ShouldExcludeFile(tc.filePath, projectRoot)
				if result != tc.expected {
					t.Errorf("ShouldExcludeFile(%q, %q) = %v, expected %v", tc.filePath, projectRoot, result, tc.expected)
				}
			})
		}
	})

	t.Run("include_regular_go_files", func(t *testing.T) {
		testCases := []struct {
			filePath string
			expected bool
			desc     string
		}{
			{"main.go", false, "should include main.go"},
			{"internal/parser.go", false, "should include go files in internal directory"},
			{"pkg/utils/helper.go", false, "should include go files in nested directories"},
			{"cmd/app/server.go", false, "should include go files in cmd directory"},
			{"api/handlers/user.go", false, "should include go files in api directory"},
			{"models/user.go", false, "should include go files in models directory"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				result := parser.ShouldExcludeFile(tc.filePath, projectRoot)
				if result != tc.expected {
					t.Errorf("ShouldExcludeFile(%q, %q) = %v, expected %v", tc.filePath, projectRoot, result, tc.expected)
				}
			})
		}
	})

	t.Run("handle_absolute_paths", func(t *testing.T) {
		testCases := []struct {
			filePath string
			expected bool
			desc     string
		}{
			{"/project/main_test.go", true, "should exclude absolute path to test file"},
			{"/project/vendor/lib.go", true, "should exclude absolute path to vendor file"},
			{"/project/testdata/sample.go", true, "should exclude absolute path to testdata file"},
			{"/project/.git/config", true, "should exclude absolute path to git file"},
			{"/project/main.go", false, "should include absolute path to regular go file"},
			{"/project/internal/parser.go", false, "should include absolute path to regular go file in subdirectory"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				result := parser.ShouldExcludeFile(tc.filePath, projectRoot)
				if result != tc.expected {
					t.Errorf("ShouldExcludeFile(%q, %q) = %v, expected %v", tc.filePath, projectRoot, result, tc.expected)
				}
			})
		}
	})

	t.Run("handle_different_project_roots", func(t *testing.T) {
		testCases := []struct {
			filePath    string
			projectRoot string
			expected    bool
			desc        string
		}{
			{"main_test.go", "/home/user/project", true, "should exclude test file regardless of project root"},
			{"vendor/lib.go", "/opt/myapp", true, "should exclude vendor file regardless of project root"},
			{"testdata/sample.go", "/var/projects/app", true, "should exclude testdata file regardless of project root"},
			{"main.go", "/different/root", false, "should include regular file regardless of project root"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				result := parser.ShouldExcludeFile(tc.filePath, tc.projectRoot)
				if result != tc.expected {
					t.Errorf("ShouldExcludeFile(%q, %q) = %v, expected %v", tc.filePath, tc.projectRoot, result, tc.expected)
				}
			})
		}
	})

	t.Run("edge_cases", func(t *testing.T) {
		testCases := []struct {
			filePath string
			expected bool
			desc     string
		}{
			{"", false, "should handle empty file path"},
			{"test.go", false, "should not exclude files that don't match exclusion patterns"},
			{"_test.go", true, "should exclude files that are just _test.go"},
			{"vendor", false, "should not exclude directory names without trailing slash when used as filename"},
			{"testdata", false, "should not exclude directory names without trailing slash when used as filename"},
			{".git", false, "should not exclude directory names without trailing slash when used as filename"},
			{"a/b/c/d/e/f/vendor/deep.go", true, "should exclude vendor files in deeply nested paths"},
			{"a/b/c/d/e/f/testdata/deep.go", true, "should exclude testdata files in deeply nested paths"},
			{"a/b/c/d/e/f/.git/deep", true, "should exclude git files in deeply nested paths"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				result := parser.ShouldExcludeFile(tc.filePath, projectRoot)
				if result != tc.expected {
					t.Errorf("ShouldExcludeFile(%q, %q) = %v, expected %v", tc.filePath, projectRoot, result, tc.expected)
				}
			})
		}
	})

	t.Run("case_sensitivity", func(t *testing.T) {
		testCases := []struct {
			filePath string
			expected bool
			desc     string
		}{
			{"VENDOR/lib.go", true, "should handle case variations in vendor directory"},
			{"Vendor/lib.go", true, "should handle case variations in vendor directory"},
			{"TESTDATA/sample.go", true, "should handle case variations in testdata directory"},
			{"TestData/sample.go", true, "should handle case variations in testdata directory"},
			{"Main_Test.go", true, "should handle case variations in test files"},
			{"MAIN_TEST.GO", true, "should handle case variations in test files"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				result := parser.ShouldExcludeFile(tc.filePath, projectRoot)
				// Note: The current implementation might be case-sensitive
				// These tests define the expected behavior - exclusions should work regardless of case
				if result != tc.expected {
					t.Errorf("ShouldExcludeFile(%q, %q) = %v, expected %v", tc.filePath, projectRoot, result, tc.expected)
				}
			})
		}
	})
}
