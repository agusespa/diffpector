package evaluation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/agusespa/diffpector/internal/types"
)

// Scorer interface for different scoring strategies
type Scorer interface {
	Score(expected types.ExpectedResults, actual []types.Issue) float64
}

// SimpleScorer implements basic scoring logic
type SimpleScorer struct{}

func NewSimpleScorer() *SimpleScorer {
	return &SimpleScorer{}
}

func (s *SimpleScorer) Score(expected types.ExpectedResults, actual []types.Issue) float64 {
	metrics := []ScoringMetric{
		&IssueFoundMetric{},
		&IssueCountMetric{},
		&SeverityMatchMetric{},
		&FileMatchMetric{},
	}

	var totalScore, applicableMetrics float64
	for _, metric := range metrics {
		score := metric.Calculate(expected, actual)
		if score >= 0 { // -1 means not applicable
			totalScore += score
			applicableMetrics++
		}
	}

	if applicableMetrics == 0 {
		return 1.0 // No expectations, full score
	}

	return totalScore / applicableMetrics
}

// ScoringMetric interface for individual scoring components
type ScoringMetric interface {
	Calculate(expected types.ExpectedResults, actual []types.Issue) float64
}

// IssueFoundMetric checks if issues were found when expected
type IssueFoundMetric struct{}

func (m *IssueFoundMetric) Calculate(expected types.ExpectedResults, actual []types.Issue) float64 {
	hasIssues := len(actual) > 0
	if expected.ShouldFindIssues == hasIssues {
		return 1.0
	}
	return 0.0
}

// IssueCountMetric checks if the number of issues is within expected range
type IssueCountMetric struct{}

func (m *IssueCountMetric) Calculate(expected types.ExpectedResults, actual []types.Issue) float64 {
	if expected.MinIssues == 0 && expected.MaxIssues == 0 {
		return -1.0 // Not applicable
	}

	count := len(actual)
	minOk := expected.MinIssues == 0 || count >= expected.MinIssues
	maxOk := expected.MaxIssues == 0 || count <= expected.MaxIssues

	if minOk && maxOk {
		return 1.0
	}
	return 0.0
}

// SeverityMatchMetric checks if expected severities are present
type SeverityMatchMetric struct{}

func (m *SeverityMatchMetric) Calculate(expected types.ExpectedResults, actual []types.Issue) float64 {
	if len(expected.ExpectedSeverity) == 0 {
		return -1.0 // Not applicable
	}

	actualSeverities := make(map[string]bool)
	for _, issue := range actual {
		actualSeverities[issue.Severity] = true
	}

	matchCount := 0
	for _, expectedSev := range expected.ExpectedSeverity {
		if actualSeverities[expectedSev] {
			matchCount++
		}
	}

	return float64(matchCount) / float64(len(expected.ExpectedSeverity))
}

// FileMatchMetric checks if expected files have issues
type FileMatchMetric struct{}

func (m *FileMatchMetric) Calculate(expected types.ExpectedResults, actual []types.Issue) float64 {
	if len(expected.ExpectedFiles) == 0 {
		return -1.0 // Not applicable
	}

	actualFiles := make(map[string]bool)
	for _, issue := range actual {
		actualFiles[issue.FilePath] = true
	}

	matchCount := 0
	for _, expectedFile := range expected.ExpectedFiles {
		if actualFiles[expectedFile] {
			matchCount++
		}
	}

	return float64(matchCount) / float64(len(expected.ExpectedFiles))
}

// RunComparison holds comparison results between multiple evaluation runs
type RunComparison struct {
	Runs          []types.EvaluationRun
	BestByScore   *types.EvaluationRun
	Fastest       *types.EvaluationRun
	MostSuccesful *types.EvaluationRun
}

// CompareResults loads and compares all evaluation results in a directory
func CompareResults(resultsDir string) error {
	runs, err := LoadEvaluationRuns(resultsDir)
	if err != nil {
		return fmt.Errorf("failed to load evaluation runs: %w", err)
	}

	if len(runs) == 0 {
		fmt.Println("No evaluation results found in", resultsDir)
		return nil
	}

	fmt.Printf("Found %d evaluation runs\n", len(runs))

	comparison := CompareRuns(runs)
	comparison.PrintComparison()

	return nil
}

// LoadEvaluationRuns loads all evaluation run results from a directory
func LoadEvaluationRuns(dir string) ([]types.EvaluationRun, error) {
	var runs []types.EvaluationRun

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob for json files in %s: %w", dir, err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read result file %s: %w", file, err)
		}

		var run types.EvaluationRun
		if err := json.Unmarshal(data, &run); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result file %s: %w", file, err)
		}
		runs = append(runs, run)
	}

	return runs, nil
}

// CompareRuns analyzes multiple evaluation runs and identifies the best performers
func CompareRuns(runs []types.EvaluationRun) *RunComparison {
	if len(runs) == 0 {
		return &RunComparison{}
	}

	// Sort runs by average score (descending)
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].AverageScore > runs[j].AverageScore
	})

	comparison := &RunComparison{
		Runs:        runs,
		BestByScore: &runs[0],
	}

	// Find the fastest run
	fastest := runs[0]
	for _, run := range runs {
		if run.TotalDuration < fastest.TotalDuration {
			fastest = run
		}
	}
	comparison.Fastest = &fastest

	// Find the most successful run
	mostSuccesful := runs[0]
	for _, run := range runs {
		if run.SuccessRate > mostSuccesful.SuccessRate {
			mostSuccesful = run
		}
	}
	comparison.MostSuccesful = &mostSuccesful

	return comparison
}

// PrintComparison prints a formatted comparison of evaluation runs
func (rc *RunComparison) PrintComparison() {
	fmt.Println("\n--- Evaluation Comparison ---")

	if len(rc.Runs) == 0 {
		fmt.Println("No runs to compare.")
		return
	}

	fmt.Println("\n🏆 Best Overall (by Score):")
	fmt.Printf("  Model: %s, Prompt: %s, Score: %.2f\n", rc.BestByScore.Model, rc.BestByScore.PromptVariant, rc.BestByScore.AverageScore)

	fmt.Println("\n🚀 Fastest Execution:")
	fmt.Printf("  Model: %s, Prompt: %s, Time: %.2fs\n", rc.Fastest.Model, rc.Fastest.PromptVariant, rc.Fastest.TotalDuration.Seconds())

	fmt.Println("\n✅ Highest Success Rate:")
	fmt.Printf("  Model: %s, Prompt: %s, Rate: %.2f%%\n", rc.MostSuccesful.Model, rc.MostSuccesful.PromptVariant, rc.MostSuccesful.SuccessRate)

	fmt.Println("\n--- Full Ranking (by Score) ---")
	for i, run := range rc.Runs {
		fmt.Printf("  %d. Model: %s, Prompt: %s, Score: %.2f, Success: %.2f%%, Time: %.2fs\n",
			i+1,
			run.Model,
			run.PromptVariant,
			run.AverageScore,
			run.SuccessRate,
			run.TotalDuration.Seconds(),
		)
	}
	fmt.Println()
}