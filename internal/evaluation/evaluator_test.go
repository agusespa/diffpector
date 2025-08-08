package evaluation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

type mockProvider struct{}

func (m *mockProvider) GetModel() string {
	return "mock-model"
}

func (m *mockProvider) Generate(prompt string) (string, error) {
	return "APPROVED", nil
}

func (m *mockProvider) SetModel(model string) {
	// TODO Mock implementation
}

func TestRunEvaluation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-eval-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testSuite := &types.EvaluationSuite{
		TestCases: []types.TestCase{
			{
				Name:        "Test Case 1",
				Description: "A simple test case",
				DiffFile:    "", // No diff file for simplicity
				Expected:    types.ExpectedResults{ShouldFindIssues: false},
			},
		},
		BaseDir: tempDir,
	}

	suiteFilePath := filepath.Join(tempDir, "test_suite.json")
	suiteFile, err := os.Create(suiteFilePath)
	if err != nil {
		t.Fatalf("Failed to create test suite file: %v", err)
	}
	if err := json.NewEncoder(suiteFile).Encode(testSuite); err != nil {
		t.Fatalf("Failed to write to test suite file: %v", err)
	}
	suiteFile.Close()

	evaluator, err := NewEvaluator(suiteFilePath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	provider := &mockProvider{}

	_, err = evaluator.runSingleTest(testSuite.TestCases[0], provider, "default")
	if err != nil {
		t.Errorf("runSingleTest() failed: %v", err)
	}
}
