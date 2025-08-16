package tools

import (
	"fmt"
	"strings"
	"testing"
)

const sampleCode = `
package sample

import (
    "fmt"
    "strings"
)

const Pi = 3.14
const (
    E = 2.71
    Version = "1.0"
)

var GlobalVar = 42

type Person struct {
    Name string
    Age  int
}

type Greeter interface {
    Greet() string
    Say(msg string) error
}

func (p *Person) Greet() string {
    return "Hello"
}

func Add(a, b int) int {
    return a + b
}

var (
    MaxValue = 100
    MinValue = 0
)
`

func TestGoParser_ParseFile(t *testing.T) {
	parser, _ := NewGoParser()

	symbols, err := parser.ParseFile("sample.go", sampleCode)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	for _, sym := range symbols {
		t.Logf("Captured symbol: %q from line %d to %d", sym.Name, sym.StartLine, sym.EndLine)
	}

	expectedSymbols := []string{
		"\"fmt\"", "\"strings\"", "Pi", "E", "Version", "GlobalVar",
		"Person", "Name", "Age", "Greeter", "Greet", "Say", "Add",
		"MaxValue", "MinValue",
	}

	// Check that we have the expected symbols
	symbolNames := make(map[string]int)
	for _, sym := range symbols {
		symbolNames[sym.Name]++
	}

	for _, expected := range expectedSymbols {
		if symbolNames[expected] == 0 {
			t.Errorf("Expected symbol %q not found", expected)
		}
	}

	// Verify we have exactly 16 symbols (including duplicate "Greet")
	expectedCount := 16
	if len(symbols) != expectedCount {
		t.Errorf("Expected %d symbols, got %d", expectedCount, len(symbols))
	}

	// Verify "Greet" appears twice (interface method + struct method)
	if symbolNames["Greet"] != 2 {
		t.Errorf("Expected 'Greet' to appear twice, got %d times", symbolNames["Greet"])
	}
}

const usageTestCode = `
package main

import "fmt"

func Add(a, b int) int {
	return a + b
}

func Multiply(x, y int) int {
	return x * y
}

type Calculator struct {
	name string
}

func (c *Calculator) Calculate(op string, a, b int) int {
	switch op {
	case "add":
		return Add(a, b)  // Usage of Add function
	case "multiply":
		result := Multiply(a, b)  // Usage of Multiply function
		return result
	default:
		return 0
	}
}

func main() {
	calc := &Calculator{name: "MyCalc"}
	
	// Multiple usages of Add
	sum1 := Add(5, 3)
	sum2 := Add(10, 20)
	
	// Usage of Calculate method
	result := calc.Calculate("add", 1, 2)
	
	fmt.Printf("Results: %d, %d, %d\n", sum1, sum2, result)
	
	// Usage in assignment
	addFunc := Add
	_ = addFunc
}
`

func TestGoParser_FindSymbolUsages(t *testing.T) {
	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	t.Run("find_function_call_usages", func(t *testing.T) {
		usages, err := parser.FindSymbolUsages("test.go", usageTestCode, "Add")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		// Should find 4 usages of Add:
		// 1. Add(a, b) in Calculate method
		// 2. Add(5, 3) in main
		// 3. Add(10, 20) in main
		// 4. addFunc := Add in main
		expectedUsages := 4
		if len(usages) != expectedUsages {
			t.Errorf("Expected %d usages of 'Add', got %d", expectedUsages, len(usages))
			for i, usage := range usages {
				t.Logf("Usage %d at line %d: %s", i+1, usage.LineNumber, strings.TrimSpace(strings.Split(usage.Context, "\n")[2]))
			}
		}

		// Verify each usage has proper context
		for _, usage := range usages {
			if usage.SymbolName != "Add" {
				t.Errorf("Expected symbol name 'Add', got '%s'", usage.SymbolName)
			}
			if usage.FilePath != "test.go" {
				t.Errorf("Expected file path 'test.go', got '%s'", usage.FilePath)
			}
			if usage.Context == "" {
				t.Errorf("Expected non-empty context for usage at line %d", usage.LineNumber)
			}
			if !strings.Contains(usage.Context, "→") {
				t.Errorf("Expected context to contain arrow marker, got: %s", usage.Context)
			}
			// Enhanced context should include function information
			if !strings.Contains(usage.Context, "Function:") && !strings.Contains(usage.Context, "In function:") && !strings.Contains(usage.Context, "Context:") {
				t.Errorf("Expected enhanced context to include function information, got: %s", usage.Context)
			}
		}
	})

	t.Run("find_method_call_usages", func(t *testing.T) {
		usages, err := parser.FindSymbolUsages("test.go", usageTestCode, "Calculate")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		// Should find 1 usage: calc.Calculate("add", 1, 2)
		expectedUsages := 1
		if len(usages) != expectedUsages {
			t.Errorf("Expected %d usages of 'Calculate', got %d", expectedUsages, len(usages))
		}

		if len(usages) > 0 {
			usage := usages[0]
			if !strings.Contains(usage.Context, "calc.Calculate") {
				t.Errorf("Expected context to contain method call, got: %s", usage.Context)
			}
			// Should include function context information
			if !strings.Contains(usage.Context, "Function:") && !strings.Contains(usage.Context, "In function:") {
				t.Errorf("Expected enhanced context to include function information, got: %s", usage.Context)
			}
		}
	})

	t.Run("no_usages_found", func(t *testing.T) {
		usages, err := parser.FindSymbolUsages("test.go", usageTestCode, "NonExistentFunction")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		if len(usages) != 0 {
			t.Errorf("Expected 0 usages of 'NonExistentFunction', got %d", len(usages))
		}
	})

	t.Run("exclude_definitions", func(t *testing.T) {
		// Should not find the function definition itself, only usages
		usages, err := parser.FindSymbolUsages("test.go", usageTestCode, "Multiply")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		// Should find 1 usage: Multiply(a, b) in Calculate method
		expectedUsages := 1
		if len(usages) != expectedUsages {
			t.Errorf("Expected %d usages of 'Multiply', got %d", expectedUsages, len(usages))
		}

		// Verify it's not the function definition line
		if len(usages) > 0 {
			usage := usages[0]
			// The definition is at line 10: "func Multiply(x, y int) int {"
			// The usage should be at a different line
			if usage.LineNumber == 10 {
				t.Errorf("Found function definition instead of usage at line %d", usage.LineNumber)
			}
		}
	})
}

func TestGoParser_isSymbolUsage(t *testing.T) {
	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Test the isSymbolUsage method indirectly through FindSymbolUsages
	// since it's easier to test the complete workflow
	testCode := `package main

func TestFunc() {
	return
}

func main() {
	TestFunc()  // This should be detected as usage
}
`

	// Use FindSymbolUsages which internally uses isSymbolUsage
	usages, err := parser.FindSymbolUsages("test.go", testCode, "TestFunc")
	if err != nil {
		t.Fatalf("FindSymbolUsages failed: %v", err)
	}

	// Should find 1 usage (the function call), not the definition
	expectedUsages := 1
	if len(usages) != expectedUsages {
		t.Errorf("Expected %d usages of 'TestFunc', got %d", expectedUsages, len(usages))
	}

	if len(usages) > 0 {
		usage := usages[0]
		// The usage should be at line 8 (the function call), not line 3 (the definition)
		if usage.LineNumber == 3 {
			t.Error("Function definition should not be considered a usage")
		}
		if usage.LineNumber != 8 {
			t.Errorf("Expected usage at line 8, got line %d", usage.LineNumber)
		}
	}
}

func TestGoParser_extractUsageContext(t *testing.T) {
	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	t.Run("small_function_context", func(t *testing.T) {
		smallFuncCode := `package main

func Add(a, b int) int {
    return a + b
}

func main() {
    result := Add(5, 3)
    fmt.Println(result)
}`

		usages, err := parser.FindSymbolUsages("test.go", smallFuncCode, "Add")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		if len(usages) == 0 {
			t.Fatal("Expected at least one usage")
		}

		context := usages[0].Context

		// Should include function signature
		if !strings.Contains(context, "Function: func main()") {
			t.Error("Context should include function signature")
		}

		// Should include the entire small function
		if !strings.Contains(context, "result := Add(5, 3)") {
			t.Error("Context should include the usage line")
		}

		if !strings.Contains(context, "fmt.Println(result)") {
			t.Error("Context should include surrounding code")
		}

		// Should mark the usage line with arrow
		if !strings.Contains(context, "→") {
			t.Error("Context should mark usage line with arrow")
		}
	})

	t.Run("large_function_context", func(t *testing.T) {
		// Create a large function to test that we include the whole function
		var largeFuncBuilder strings.Builder
		largeFuncBuilder.WriteString("package main\n\nfunc LargeFunction() {\n")

		// Add many lines to make it large
		for i := 0; i < 60; i++ {
			if i == 30 {
				largeFuncBuilder.WriteString("    result := Add(5, 3) // Usage here\n")
			} else {
				largeFuncBuilder.WriteString(fmt.Sprintf("    // Line %d\n", i))
			}
		}
		largeFuncBuilder.WriteString("}\n\nfunc Add(a, b int) int { return a + b }")

		largeFuncCode := largeFuncBuilder.String()

		usages, err := parser.FindSymbolUsages("test.go", largeFuncCode, "Add")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		if len(usages) == 0 {
			t.Fatal("Expected at least one usage")
		}

		context := usages[0].Context

		// Should include function signature
		if !strings.Contains(context, "Function: func LargeFunction()") {
			t.Error("Context should include function signature")
		}

		// Should include the usage line
		if !strings.Contains(context, "result := Add(5, 3)") {
			t.Error("Context should include the usage line")
		}

		// Should mark the usage line with arrow
		if !strings.Contains(context, "→") {
			t.Error("Context should mark usage line with arrow")
		}

		// Should include the entire function (simplified approach)
		if !strings.Contains(context, "// Line 0") || !strings.Contains(context, "// Line 59") {
			t.Error("Context should include the entire function")
		}
	})

	t.Run("control_flow_context", func(t *testing.T) {
		controlFlowCode := `package main

func ProcessData() {
    data := loadData()
    
    if data != nil {
        result := Add(data.x, data.y)
        fmt.Println(result)
    } else {
        fmt.Println("No data")
    }
}

func Add(a, b int) int { return a + b }`

		usages, err := parser.FindSymbolUsages("test.go", controlFlowCode, "Add")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		if len(usages) == 0 {
			t.Fatal("Expected at least one usage")
		}

		context := usages[0].Context

		// Should include function signature
		if !strings.Contains(context, "Function: func ProcessData()") {
			t.Error("Context should include function signature")
		}

		// Should include the if statement context
		if !strings.Contains(context, "if data != nil") {
			t.Error("Context should include if statement")
		}

		// Should include the usage line
		if !strings.Contains(context, "result := Add(data.x, data.y)") {
			t.Error("Context should include the usage line")
		}

		// Should mark the usage line with arrow
		if !strings.Contains(context, "→") {
			t.Error("Context should mark usage line with arrow")
		}
	})
}

func TestGoParser_contextEdgeCases(t *testing.T) {
	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	t.Run("top_level_usage", func(t *testing.T) {
		topLevelCode := `package main

import "fmt"

var result = Add(5, 3)

func Add(a, b int) int { return a + b }`

		usages, err := parser.FindSymbolUsages("test.go", topLevelCode, "Add")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		if len(usages) == 0 {
			t.Fatal("Expected at least one usage")
		}

		context := usages[0].Context

		// Should handle top-level usage (not in a function)
		if !strings.Contains(context, "result = Add(5, 3)") {
			t.Error("Context should include top-level usage")
		}

		// Should mark the usage line with arrow
		if !strings.Contains(context, "→") {
			t.Error("Context should mark usage line with arrow")
		}
	})

	t.Run("method_usage", func(t *testing.T) {
		methodCode := `package main

type Calculator struct{}

func (c *Calculator) Add(a, b int) int {
    return a + b
}

func main() {
    calc := &Calculator{}
    result := calc.Add(5, 3)
    fmt.Println(result)
}`

		usages, err := parser.FindSymbolUsages("test.go", methodCode, "Add")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		if len(usages) == 0 {
			t.Fatal("Expected at least one usage")
		}

		context := usages[0].Context

		// Should include function signature
		if !strings.Contains(context, "Function: func main()") {
			t.Error("Context should include function signature")
		}

		// Should include the method call
		if !strings.Contains(context, "calc.Add(5, 3)") {
			t.Error("Context should include method call")
		}

		// Should mark the usage line with arrow
		if !strings.Contains(context, "→") {
			t.Error("Context should mark usage line with arrow")
		}
	})

	t.Run("nested_function_calls", func(t *testing.T) {
		nestedCode := `package main

func main() {
    result := Add(Multiply(2, 3), 4)
    fmt.Println(result)
}

func Add(a, b int) int { return a + b }
func Multiply(a, b int) int { return a * b }`

		// Test finding Add usage
		usages, err := parser.FindSymbolUsages("test.go", nestedCode, "Add")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		if len(usages) == 0 {
			t.Fatal("Expected at least one usage of Add")
		}

		context := usages[0].Context

		// Should include the nested call
		if !strings.Contains(context, "Add(Multiply(2, 3), 4)") {
			t.Error("Context should include nested function call")
		}

		// Test finding Multiply usage
		usages, err = parser.FindSymbolUsages("test.go", nestedCode, "Multiply")
		if err != nil {
			t.Fatalf("FindSymbolUsages failed: %v", err)
		}

		if len(usages) == 0 {
			t.Fatal("Expected at least one usage of Multiply")
		}

		context = usages[0].Context

		// Should include the nested call
		if !strings.Contains(context, "Add(Multiply(2, 3), 4)") {
			t.Error("Context should include nested function call for Multiply")
		}
	})
}

func TestGoParser_contextExtractionHelpers(t *testing.T) {
	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	t.Run("function_signature_extraction", func(t *testing.T) {
		funcCode := `package main

func Add(a, b int) int {
    return a + b
}

func (c *Calculator) Multiply(x, y float64) (float64, error) {
    return x * y, nil
}`

		// Parse the code to get AST
		src := []byte(funcCode)
		tree := parser.parser.Parse(src, nil)
		if tree == nil {
			t.Fatal("Failed to parse code")
		}
		defer tree.Close()

		// Find function nodes and test signature extraction
		root := tree.RootNode()
		for i := uint(0); i < root.ChildCount(); i++ {
			child := root.Child(i)
			if child != nil && child.Kind() == "function_declaration" {
				signature := parser.extractFunctionSignature(src, child)

				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					funcName := nameNode.Utf8Text(src)

					if funcName == "Add" {
						expected := "func Add(a, b int) int"
						if signature != expected {
							t.Errorf("Expected signature '%s', got '%s'", expected, signature)
						}
					}
				}
			} else if child != nil && child.Kind() == "method_declaration" {
				signature := parser.extractFunctionSignature(src, child)

				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					funcName := nameNode.Utf8Text(src)

					if funcName == "Multiply" {
						if !strings.Contains(signature, "func (c *Calculator) Multiply") {
							t.Errorf("Method signature should include receiver, got '%s'", signature)
						}
						if !strings.Contains(signature, "(float64, error)") {
							t.Errorf("Method signature should include return types, got '%s'", signature)
						}
					}
				}
			}
		}
	})
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
