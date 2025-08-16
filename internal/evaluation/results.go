package evaluation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agusespa/diffpector/internal/types"
)

type ResultsManager struct {
	resultsDir string
}

func NewResultsManager(resultsDir string) *ResultsManager {
	return &ResultsManager{
		resultsDir: resultsDir,
	}
}

func (rm *ResultsManager) SaveEvaluationResults(result *types.EvaluationResult) error {
	if err := os.MkdirAll(rm.resultsDir, 0755); err != nil {
		return fmt.Errorf("failed to create results directory at %s: %w", rm.resultsDir, err)
	}

	var filename string
	if result.TotalRuns == 1 {
		filename = fmt.Sprintf("eval_%s_%s_%d.json",
			result.Model, result.PromptVariant, result.StartTime.Unix())
	} else {
		filename = fmt.Sprintf("eval_%s_%s_%druns_%d.json",
			result.Model, result.PromptVariant, result.TotalRuns, result.StartTime.Unix())
	}
	filepath := filepath.Join(rm.resultsDir, filename)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write results file to %s: %w", filepath, err)
	}

	fmt.Printf("Results saved to: %s\n", filepath)
	return nil
}
