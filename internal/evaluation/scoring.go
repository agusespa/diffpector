package evaluation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/agusespa/diffpector/internal/types"
)

func CalculateScore(expected types.ExpectedResults, actual []types.Issue) float64 {
	scores := []float64{
		calculateIssueFoundScore(expected, actual),
		calculateIssueCountScore(expected, actual),
		calculateSeverityMatchScore(expected, actual),
		calculateFileMatchScore(expected, actual),
	}

	var totalScore, applicableMetrics float64
	for _, score := range scores {
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

// calculateIssueFoundScore checks if issues were found when expected
func calculateIssueFoundScore(expected types.ExpectedResults, actual []types.Issue) float64 {
	hasIssues := len(actual) > 0
	if expected.ShouldFindIssues == hasIssues {
		return 1.0
	}
	return 0.0
}

// calculateIssueCountScore checks if the number of issues is within expected range
func calculateIssueCountScore(expected types.ExpectedResults, actual []types.Issue) float64 {
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

// calculateSeverityMatchScore checks if expected severities are present
func calculateSeverityMatchScore(expected types.ExpectedResults, actual []types.Issue) float64 {
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

// calculateFileMatchScore checks if expected files have issues
func calculateFileMatchScore(expected types.ExpectedResults, actual []types.Issue) float64 {
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

		// Try to unmarshal as new unified EvaluationResult format first
		var evalResult types.EvaluationResult
		if err := json.Unmarshal(data, &evalResult); err == nil && len(evalResult.IndividualRuns) > 0 {
			// New format - extract individual runs
			runs = append(runs, evalResult.IndividualRuns...)
			continue
		}

		// Fall back to old single EvaluationRun format
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

// PromptStats holds performance statistics for a prompt variant
type PromptStats struct {
	PromptVariant      string
	Runs               int
	AverageScore       float64
	ScoreStdDev        float64
	AverageSuccessRate float64
	FalsePositiveRate  float64
	TestCaseBreakdown  map[string]float64 // test case name -> average score
}

// ComparePrompts analyzes and compares different prompt variants
func ComparePrompts(resultsDir string) error {
	runs, err := LoadEvaluationRuns(resultsDir)
	if err != nil {
		return fmt.Errorf("failed to load evaluation runs: %w", err)
	}

	if len(runs) == 0 {
		fmt.Println("No evaluation results found in", resultsDir)
		return nil
	}

	// Group runs by model, then by prompt variant
	modelGroups := make(map[string][]types.EvaluationRun)
	for _, run := range runs {
		modelGroups[run.Model] = append(modelGroups[run.Model], run)
	}

	// Process each model separately
	for model, modelRuns := range modelGroups {
		comparePromptsForModel(model, modelRuns)
	}

	return nil
}

func comparePromptsForModel(model string, modelRuns []types.EvaluationRun) {

	// Group runs by prompt variant
	promptGroups := make(map[string][]types.EvaluationRun)
	for _, run := range modelRuns {
		promptGroups[run.PromptVariant] = append(promptGroups[run.PromptVariant], run)
	}

	// Calculate stats for each prompt
	var promptStats []PromptStats
	for promptName, runs := range promptGroups {
		stats := calculatePromptStats(promptName, runs)
		promptStats = append(promptStats, stats)
	}

	// Sort by average score
	sort.Slice(promptStats, func(i, j int) bool {
		return promptStats[i].AverageScore > promptStats[j].AverageScore
	})

	printPromptComparison(model, promptStats)
}

func calculatePromptStats(promptName string, runs []types.EvaluationRun) PromptStats {
	if len(runs) == 0 {
		return PromptStats{PromptVariant: promptName}
	}

	var scores, successRates []float64
	var falsePositives int
	totalCleanRefactorTests := 0
	testCaseScores := make(map[string][]float64)

	for _, run := range runs {
		scores = append(scores, run.AverageScore)
		successRates = append(successRates, run.SuccessRate)

		// Analyze individual test cases
		for _, result := range run.Results {
			testName := result.TestCase.Name
			testCaseScores[testName] = append(testCaseScores[testName], result.Score)

			// Check for false positives in clean_refactor test
			if testName == "clean_refactor" {
				totalCleanRefactorTests++
				if len(result.Issues) > 0 {
					falsePositives++
				}
			}
		}
	}

	// Calculate test case averages
	calc := NewStatisticsCalculator()
	testCaseBreakdown := make(map[string]float64)
	for testName, scores := range testCaseScores {
		testCaseBreakdown[testName] = calc.CalculateMean(scores)
	}

	falsePositiveRate := 0.0
	if totalCleanRefactorTests > 0 {
		falsePositiveRate = float64(falsePositives) / float64(totalCleanRefactorTests)
	}

	return PromptStats{
		PromptVariant:      promptName,
		Runs:               len(runs),
		AverageScore:       calc.CalculateMean(scores),
		ScoreStdDev:        calc.CalculateStdDev(scores),
		AverageSuccessRate: calc.CalculateMean(successRates),
		FalsePositiveRate:  falsePositiveRate,
		TestCaseBreakdown:  testCaseBreakdown,
	}
}

func printPromptComparison(model string, promptStats []PromptStats) {
	fmt.Printf("\n=== Prompt Comparison for Model: %s ===\n", model)

	if len(promptStats) == 0 {
		fmt.Println("No prompt results to compare.")
		return
	}

	// Overall ranking
	fmt.Println("\n🏆 Prompt Performance Ranking:")
	for i, stats := range promptStats {
		fmt.Printf("  %d. %s: %.3f (±%.3f) | Success: %.1f%% | FP Rate: %.1f%% | Runs: %d\n",
			i+1, stats.PromptVariant, stats.AverageScore, stats.ScoreStdDev,
			stats.AverageSuccessRate, stats.FalsePositiveRate*100, stats.Runs)
	}

	// Test case breakdown for top prompts
	if len(promptStats) >= 2 {
		fmt.Println("\n📊 Test Case Performance (Top 2 Prompts):")

		// Get all test case names
		testCases := make(map[string]bool)
		for _, stats := range promptStats[:2] {
			for testName := range stats.TestCaseBreakdown {
				testCases[testName] = true
			}
		}

		// Sort test case names for consistent output
		var sortedTestCases []string
		for testName := range testCases {
			sortedTestCases = append(sortedTestCases, testName)
		}
		sort.Strings(sortedTestCases)

		for _, testName := range sortedTestCases {
			fmt.Printf("  %s:\n", testName)
			for _, stats := range promptStats[:2] {
				if score, exists := stats.TestCaseBreakdown[testName]; exists {
					fmt.Printf("    %s: %.3f\n", stats.PromptVariant, score)
				}
			}
		}
	}

	// Recommendations
	if len(promptStats) >= 2 {
		best := promptStats[0]
		fmt.Printf("\n💡 Recommendation: Use '%s' prompt\n", best.PromptVariant)
		fmt.Printf("   - Highest average score: %.3f\n", best.AverageScore)
		if best.FalsePositiveRate == 0 {
			fmt.Printf("   - No false positives detected\n")
		} else {
			fmt.Printf("   - False positive rate: %.1f%%\n", best.FalsePositiveRate*100)
		}
	}

	fmt.Println()
}
