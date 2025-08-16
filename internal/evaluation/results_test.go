package evaluation

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agusespa/diffpector/internal/types"
)

func TestNewResultsManager(t *testing.T) {
	rm := NewResultsManager("/test/dir")

	if rm.resultsDir != "/test/dir" {
		t.Errorf("Expected resultsDir to be '/test/dir', got '%s'", rm.resultsDir)
	}
}

func TestSaveEvaluationResults_SingleRun(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-eval-results-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rm := NewResultsManager(tempDir)

	result := &types.EvaluationResult{
		Model:         "test-model",
		PromptVariant: "test-prompt",
		TotalRuns:     1,
		StartTime:     time.Unix(1234567890, 0),
		IndividualRuns: []types.EvaluationRun{
			{Model: "test-model", PromptVariant: "test-prompt"},
		},
	}

	err = rm.SaveEvaluationResults(result)
	if err != nil {
		t.Fatalf("SaveEvaluationResults failed: %v", err)
	}

	// Check filename for single run
	expectedFilename := "eval_test-model_test-prompt_1234567890.json"
	expectedPath := filepath.Join(tempDir, expectedFilename)

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to be created", expectedPath)
	}
}

func TestSaveEvaluationResults_MultipleRuns(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-eval-results-multi-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rm := NewResultsManager(tempDir)

	result := &types.EvaluationResult{
		Model:         "test-model",
		PromptVariant: "test-prompt",
		TotalRuns:     3,
		StartTime:     time.Unix(1234567890, 0),
		IndividualRuns: []types.EvaluationRun{
			{Model: "test-model", PromptVariant: "test-prompt"},
			{Model: "test-model", PromptVariant: "test-prompt"},
			{Model: "test-model", PromptVariant: "test-prompt"},
		},
	}

	err = rm.SaveEvaluationResults(result)
	if err != nil {
		t.Fatalf("SaveEvaluationResults failed: %v", err)
	}

	// Check filename for multiple runs
	expectedFilename := "eval_test-model_test-prompt_3runs_1234567890.json"
	expectedPath := filepath.Join(tempDir, expectedFilename)

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to be created", expectedPath)
	}
}
