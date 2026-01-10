package evaluation

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/agusespa/diffpector/internal/types"
)

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	mean := calculateMean(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func CalculateRunSummary(r *types.EvaluationRun) {
	if len(r.Results) == 0 {
		return
	}

	var totalScore float64
	var successfulTests int
	for _, result := range r.Results {
		totalScore += result.Score
		if result.Success {
			successfulTests++
		}
	}

	r.AverageScore = totalScore / float64(len(r.Results))
	r.SuccessRate = (float64(successfulTests) / float64(len(r.Results))) * 100
}

func CalculateEvaluationStats(result *types.EvaluationResult) {
	if len(result.IndividualRuns) == 0 {
		return
	}

	var scores, successRates, durations []float64
	for _, run := range result.IndividualRuns {
		scores = append(scores, run.AverageScore)
		successRates = append(successRates, run.SuccessRate)
		durations = append(durations, run.TotalDuration.Seconds())
	}

	result.AggregatedStats = types.EvaluationStats{
		AverageScore:       calculateMean(scores),
		ScoreStdDev:        calculateStdDev(scores),
		AverageSuccessRate: calculateMean(successRates),
		SuccessRateStdDev:  calculateStdDev(successRates),
		AverageDuration:    calculateMean(durations),
		DurationStdDev:     calculateStdDev(durations),
	}

	testCaseResults := make(map[string][]float64)
	for _, run := range result.IndividualRuns {
		for _, testResult := range run.Results {
			name := testResult.TestCase.Name
			testCaseResults[name] = append(testCaseResults[name], testResult.Score)
		}
	}

	result.TestCaseStats = make(map[string]types.TestCaseStats)
	for testName, scores := range testCaseResults {
		result.TestCaseStats[testName] = types.TestCaseStats{
			TestCaseName: testName,
			AverageScore: calculateMean(scores),
			ScoreStdDev:  calculateStdDev(scores),
		}
	}
}

func PrintRunHeader(modelName, promptVariant string, numRuns int) {
	if numRuns == 1 {
		fmt.Printf("Running evaluation for model: %s, prompt: %s\n", modelName, promptVariant)
	} else {
		fmt.Printf("Running %d evaluations for model: %s, prompt: %s\n", numRuns, modelName, promptVariant)
	}
}

func PrintMultiRunProgress(runNum, totalRuns int) {
	fmt.Printf("\n--- Run %d/%d ---\n", runNum, totalRuns)
}

func PrintTestResult(result *types.TestCaseResult, err error) {
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		fmt.Printf("  DONE (%.2fs, score: %.2f)\n", result.ExecutionTime.Seconds(), result.Score)
	}
}

func PrintSummary(r *types.EvaluationRun) {
	fmt.Printf("\n=== Evaluation Summary for %s (%s) ===\n", r.Model, r.PromptVariant)
	fmt.Printf("Average Score: %.2f\n", r.AverageScore)
	fmt.Printf("Success Rate:  %.2f%%\n", r.SuccessRate)
	fmt.Printf("Total Duration:  %.2fs\n", r.TotalDuration.Seconds())
	fmt.Println()
}

func PrintEvaluationSummary(r *types.EvaluationResult) {
	fmt.Printf("\n=== Evaluation Summary for %s (%s) ===\n", r.Model, r.PromptVariant)
	fmt.Printf("Runs: %d\n", r.TotalRuns)
	fmt.Printf("Average Score: %.2f (±%.2f)\n", r.AggregatedStats.AverageScore, r.AggregatedStats.ScoreStdDev)
	fmt.Printf("Success Rate: %.2f%% (±%.2f%%)\n", r.AggregatedStats.AverageSuccessRate, r.AggregatedStats.SuccessRateStdDev)
	fmt.Printf("Average Duration: %.2fs (±%.2fs)\n", r.AggregatedStats.AverageDuration, r.AggregatedStats.DurationStdDev)
	fmt.Printf("Total Duration: %.2fs\n", r.TotalDuration.Seconds())

	if r.TotalRuns > 1 && len(r.TestCaseStats) > 0 {
		fmt.Printf("\nTest Case Performance:\n")
		for _, stats := range r.TestCaseStats {
			fmt.Printf("  %s: %.2f (±%.2f)\n", stats.TestCaseName, stats.AverageScore, stats.ScoreStdDev)
		}
	}
	fmt.Println()
}

func SaveEvaluationResults(resultsDir string, result *types.EvaluationResult) error {
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	var filename string
	if result.TotalRuns == 1 {
		filename = fmt.Sprintf("eval_%s_%s_%d.json",
			result.Model, result.PromptVariant, result.StartTime.Unix())
	} else {
		filename = fmt.Sprintf("eval_%s_%s_%druns_%d.json",
			result.Model, result.PromptVariant, result.TotalRuns, result.StartTime.Unix())
	}
	filepath := filepath.Join(resultsDir, filename)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}

	fmt.Printf("Results saved to: %s\n", filepath)
	return nil
}

type ComparisonResult struct {
	Model         string
	PromptVariant string
	Runs          int
	AvgScore      float64
	StdDev        float64
	SuccessRate   float64
	AvgDuration   float64
}

func CompareResults(resultsDir string) error {
	runs, err := loadEvaluationRuns(resultsDir)
	if err != nil {
		return err
	}

	if len(runs) == 0 {
		fmt.Println("No evaluation results found")
		return nil
	}

	modelGroups := make(map[string][]types.EvaluationRun)
	for _, run := range runs {
		modelGroups[run.Model] = append(modelGroups[run.Model], run)
	}

	fmt.Printf("\n=== Model Comparison ===\n")
	fmt.Printf("Found %d evaluation runs\n\n", len(runs))

	for model, modelRuns := range modelGroups {
		results := aggregateRuns(modelRuns)
		printComparison(model, "Model", results)
	}

	return nil
}

func ComparePrompts(resultsDir string) error {
	runs, err := loadEvaluationRuns(resultsDir)
	if err != nil {
		return err
	}

	if len(runs) == 0 {
		fmt.Println("No evaluation results found")
		return nil
	}

	modelGroups := make(map[string][]types.EvaluationRun)
	for _, run := range runs {
		modelGroups[run.Model] = append(modelGroups[run.Model], run)
	}

	fmt.Printf("\n=== Prompt Comparison ===\n")

	for model, modelRuns := range modelGroups {
		promptGroups := make(map[string][]types.EvaluationRun)
		for _, run := range modelRuns {
			promptGroups[run.PromptVariant] = append(promptGroups[run.PromptVariant], run)
		}

		var results []ComparisonResult
		for _, promptRuns := range promptGroups {
			results = append(results, aggregateRuns(promptRuns)...)
		}

		printComparison(model, "Prompt", results)
	}

	return nil
}

func aggregateRuns(runs []types.EvaluationRun) []ComparisonResult {
	if len(runs) == 0 {
		return nil
	}

	groups := make(map[string][]types.EvaluationRun)
	for _, run := range runs {
		key := run.Model + "|" + run.PromptVariant
		groups[key] = append(groups[key], run)
	}

	var results []ComparisonResult
	for _, groupRuns := range groups {
		var scores, successRates, durations []float64
		for _, run := range groupRuns {
			scores = append(scores, run.AverageScore)
			successRates = append(successRates, run.SuccessRate)
			durations = append(durations, run.TotalDuration.Seconds())
		}

		results = append(results, ComparisonResult{
			Model:         groupRuns[0].Model,
			PromptVariant: groupRuns[0].PromptVariant,
			Runs:          len(groupRuns),
			AvgScore:      calculateMean(scores),
			StdDev:        calculateStdDev(scores),
			SuccessRate:   calculateMean(successRates),
			AvgDuration:   calculateMean(durations),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].AvgScore > results[j].AvgScore
	})

	return results
}

func printComparison(groupName, groupType string, results []ComparisonResult) {
	if len(results) == 0 {
		return
	}

	fmt.Printf("\n%s: %s\n", groupType, groupName)
	fmt.Println("Rank | Variant | Score | Success | Duration | Runs")
	fmt.Println("-----|---------|-------|---------|----------|-----")

	for i, r := range results {
		variant := r.PromptVariant
		if groupType == "Model" {
			variant = r.Model
		}

		stdDevStr := ""
		if r.Runs > 1 {
			stdDevStr = fmt.Sprintf(" (±%.2f)", r.StdDev)
		}

		fmt.Printf("%4d | %-15s | %.2f%s | %.1f%% | %.2fs | %d\n",
			i+1, variant, r.AvgScore, stdDevStr, r.SuccessRate, r.AvgDuration, r.Runs)
	}
}

func loadEvaluationRuns(dir string) ([]types.EvaluationRun, error) {
	var runs []types.EvaluationRun

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to find result files: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		var evalResult types.EvaluationResult
		if err := json.Unmarshal(data, &evalResult); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}

		if len(evalResult.IndividualRuns) == 0 {
			return nil, fmt.Errorf("no runs found in %s", file)
		}

		runs = append(runs, evalResult.IndividualRuns...)
	}

	return runs, nil
}

func CalculateScore(expected types.ExpectedResults, actual []types.Issue) float64 {
	// 1. False Positive Check
	if !expected.ShouldFindIssues {
		if len(actual) > 0 {
			return 0.0 // Failed: Found issues when none expected
		}
		return 1.0 // Success: Found none as expected
	}

	// 2. False Negative Check
	if len(actual) == 0 {
		return 0.0 // Failed: Found none when expected
	}

	score := 1.0

	// 3. Severity Check (The most important metric)
	// If the model found issues, but they are all "Minor" when we expect "Critical",
	// it missed the point.
	maxFoundSev := getMaxSeverity(actual)
	maxExpectedSev := getMaxSeverityFromStrings(expected.ExpectedSeverity)

	if maxFoundSev < maxExpectedSev {
		// Penalty: 50% reduction if severity isn't high enough
		// e.g. Found WARNING (20), Expected CRITICAL (40).
		score -= 0.5
	}

	// 4. Count Check (Secondary metric)
	if expected.MinIssues > 0 && len(actual) < expected.MinIssues {
		score -= 0.2
	}
	if expected.MaxIssues > 0 && len(actual) > expected.MaxIssues {
		score -= 0.1 // Less penalty for finding too many (could be noise)
	}

	if score < 0 {
		return 0.0
	}
	return score
}

// Helpers for Severity Logic
func getSeverityLevel(s string) int {
	switch s {
	case "CRITICAL":
		return 40
	case "HIGH":
		return 30
	case "WARNING", "MEDIUM":
		return 20
	case "MINOR", "LOW":
		return 10
	default:
		return 0
	}
}

func getMaxSeverity(issues []types.Issue) int {
	max := 0
	for _, issue := range issues {
		lvl := getSeverityLevel(issue.Severity)
		if lvl > max {
			max = lvl
		}
	}
	return max
}

func getMaxSeverityFromStrings(severities []string) int {
	max := 0
	for _, s := range severities {
		lvl := getSeverityLevel(s)
		if lvl > max {
			max = lvl
		}
	}
	return max
}
