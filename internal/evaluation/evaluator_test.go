package evaluation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/llm"
	"github.com/agusespa/diffpector/internal/types"
)

type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) GetModel() string {
	return "mock-model"
}

func (m *mockProvider) Generate(prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockProvider) ChatWithTools(messages []llm.Message, tools []llm.Tool) (*llm.ChatResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.ChatResponse{
		Content:   m.response,
		ToolCalls: nil,
	}, nil
}

func TestNewEvaluator(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-eval-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	testSuite := &types.EvaluationSuite{
		TestCases: []types.TestCase{
			{
				Name:        "Test Case 1",
				Description: "A simple test case",
				DiffFile:    "",
				Expected:    types.ExpectedResults{ShouldFindIssues: false},
			},
		},
		BaseDir: tempDir,
	}

	suiteFilePath := filepath.Join(tempDir, "test_suite.json")
	suiteData, _ := json.Marshal(testSuite)
	err = os.WriteFile(suiteFilePath, suiteData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test suite file: %v", err)
	}

	evaluator, err := NewEvaluator(suiteFilePath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	if evaluator.suite == nil {
		t.Error("Expected suite to be loaded")
	}

	if evaluator.resultsDir == "" {
		t.Error("Expected results directory to be set")
	}
}

func TestNewEvaluator_InvalidSuiteFile(t *testing.T) {
	_, err := NewEvaluator("nonexistent.json", "/tmp")
	if err == nil {
		t.Error("Expected error for nonexistent suite file")
	}
}

func TestRunSingleTest_ApprovedResponse(t *testing.T) {
	tempDir, mockFiles := setupTestEnvironment(t)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	evaluator, testCase := createTestEvaluator(t, tempDir, mockFiles)
	provider := &mockProvider{response: "APPROVED"}

	result, err := evaluator.runSingleTest(testCase, provider, "test-model", "default")
	if err != nil {
		t.Fatalf("runSingleTest() failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if len(result.Issues) != 0 {
		t.Errorf("Expected 0 issues, got %d", len(result.Issues))
	}

	if result.Score != 1.0 {
		t.Errorf("Expected score 1.0 for approved response, got %f", result.Score)
	}
}

func TestRunSingleTest_JSONResponse(t *testing.T) {
	tempDir, mockFiles := setupTestEnvironment(t)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	evaluator, testCase := createTestEvaluator(t, tempDir, mockFiles)

	jsonResponse := `[
		{
			"severity": "CRITICAL",
			"file_path": "test.go",
			"start_line": 10,
			"end_line": 12,
			"description": "SQL injection vulnerability"
		}
	]`

	provider := &mockProvider{response: jsonResponse}

	result, err := evaluator.runSingleTest(testCase, provider, "test-model", "default")
	if err != nil {
		t.Fatalf("runSingleTest() failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if len(result.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != "CRITICAL" {
		t.Errorf("Expected severity CRITICAL, got %s", issue.Severity)
	}

	if issue.Description != "SQL injection vulnerability" {
		t.Errorf("Expected specific description, got %s", issue.Description)
	}
}

func TestRunSingleTest_MalformedResponse(t *testing.T) {
	tempDir, mockFiles := setupTestEnvironment(t)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	evaluator, testCase := createTestEvaluator(t, tempDir, mockFiles)

	// Response that starts with text but contains JSON
	malformedResponse := `The code looks good overall, but I found some issues:
	[
		{
			"severity": "WARNING",
			"file_path": "test.go",
			"start_line": 5,
			"end_line": 5,
			"description": "Missing error handling"
		}
	]`

	provider := &mockProvider{response: malformedResponse}

	result, err := evaluator.runSingleTest(testCase, provider, "test-model", "default")
	if err != nil {
		t.Fatalf("runSingleTest() failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if len(result.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(result.Issues))
	}
}

func TestRunSingleTest_FormatViolation(t *testing.T) {
	tempDir, mockFiles := setupTestEnvironment(t)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	evaluator, testCase := createTestEvaluator(t, tempDir, mockFiles)

	// Response that violates the expected format
	formatViolationResponse := "As a code reviewer, I will focus on reviewing only the code changes shown in the diff above. Here are my findings:"

	provider := &mockProvider{response: formatViolationResponse}

	result, err := evaluator.runSingleTest(testCase, provider, "test-model", "default")
	if err != nil {
		t.Fatalf("runSingleTest() failed: %v", err)
	}

	// Format violations should be marked as failures
	if result.Success {
		t.Error("Expected success to be false for format violation")
	}

	if len(result.Issues) != 0 {
		t.Errorf("Expected 0 issues for format violation (couldn't parse), got %d", len(result.Issues))
	}

	// Format violations should get zero score
	if result.Score != 0.0 {
		t.Errorf("Expected score 0.0 for format violation, got %f", result.Score)
	}

	// Should have an error message about format violation
	if len(result.Errors) == 0 {
		t.Error("Expected error message for format violation")
	} else if !strings.Contains(result.Errors[0], "Format violation") {
		t.Errorf("Expected format violation error, got: %s", result.Errors[0])
	}
}

func TestRunSingleTest_ProviderError(t *testing.T) {
	tempDir, mockFiles := setupTestEnvironment(t)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	evaluator, testCase := createTestEvaluator(t, tempDir, mockFiles)
	provider := &mockProvider{err: errors.New("provider error")}

	_, err := evaluator.runSingleTest(testCase, provider, "test-model", "default")
	if err == nil {
		t.Error("Expected error when provider fails")
	}

	if !strings.Contains(err.Error(), "agent review failed") {
		t.Errorf("Expected 'agent review failed' in error, got: %v", err)
	}
}

// Helper functions for test setup

func setupTestEnvironment(t *testing.T) (string, string) {
	tempDir, err := os.MkdirTemp("", "test-eval-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create mock files directory
	mockFiles := filepath.Join(tempDir, "mock_files")
	err = os.MkdirAll(mockFiles, 0755)
	if err != nil {
		t.Fatalf("Failed to create mock files dir: %v", err)
	}

	// Create a test file
	testFileContent := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n"
	err = os.WriteFile(filepath.Join(mockFiles, "test.go"), []byte(testFileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	return tempDir, mockFiles
}

func createTestEvaluator(t *testing.T, tempDir, mockFiles string) (*Evaluator, types.TestCase) {
	// Create a test diff file with full path to the mock file
	testFilePath := filepath.Join(mockFiles, "test.go")
	diffContent := fmt.Sprintf(`--- a/%s
+++ b/%s
@@ -1,5 +1,5 @@
 package main
 
 func main() {
-	println("Hello, World!")
+	fmt.Println("Hello, World!")
 }
`, testFilePath, testFilePath)
	diffFilePath := filepath.Join(tempDir, "test.diff")
	err := os.WriteFile(diffFilePath, []byte(diffContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create diff file: %v", err)
	}

	testSuite := &types.EvaluationSuite{
		TestCases: []types.TestCase{
			{
				Name:        "Test Case 1",
				Description: "A simple test case",
				DiffFile:    "test.diff",
				Expected: types.ExpectedResults{
					ShouldFindIssues: false,
					MinIssues:        0,
					MaxIssues:        5,
				},
			},
		},
		BaseDir:      tempDir,
		MockFilesDir: mockFiles,
	}

	suiteFilePath := filepath.Join(tempDir, "test_suite.json")
	suiteData, _ := json.Marshal(testSuite)
	err = os.WriteFile(suiteFilePath, suiteData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test suite file: %v", err)
	}

	evaluator, err := NewEvaluator(suiteFilePath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	return evaluator, testSuite.TestCases[0]
}
