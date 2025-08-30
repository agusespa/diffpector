package evaluation

import (
	"fmt"

	"github.com/agusespa/diffpector/internal/types"
)

func PrintSummary(r *types.EvaluationRun) {
	fmt.Printf("\n--- Summary for %s (%s) ---\n", r.Model, r.PromptVariant)
	fmt.Printf("  Average Score: %.2f\n", r.AverageScore)
	fmt.Printf("  Success Rate:  %.2f%%\n", r.SuccessRate)
	fmt.Printf("  Total Duration:  %.2fs\n", r.TotalDuration.Seconds())
	fmt.Println()
}

func PrintEvaluationSummary(r *types.EvaluationResult) {
	fmt.Printf("\n--- Evaluation Summary for %s (%s) ---\n", r.Model, r.PromptVariant)
	fmt.Printf("  Runs: %d\n", r.TotalRuns)
	fmt.Printf("  Average Score: %.2f (±%.3f)\n", r.AggregatedStats.AverageScore, r.AggregatedStats.ScoreStdDev)
	fmt.Printf("  Score Range: %.2f - %.2f\n", r.AggregatedStats.MinScore, r.AggregatedStats.MaxScore)
	fmt.Printf("  Success Rate: %.2f%% (±%.3f%%)\n", r.AggregatedStats.AverageSuccessRate, r.AggregatedStats.SuccessRateStdDev)
	fmt.Printf("  Average Duration: %.2fs (±%.3fs)\n", r.AggregatedStats.AverageDuration, r.AggregatedStats.DurationStdDev)
	fmt.Printf("  Total Duration: %.2fs\n", r.TotalDuration.Seconds())

	if r.TotalRuns > 1 {
		fmt.Printf("\n  Test Case Performance:\n")
		for _, stats := range r.TestCaseStats {
			// Add quality indicator
			var qualityIndicator string
			if stats.QualityScore >= 0.8 {
				qualityIndicator = "🟢 GOOD"
			} else if stats.QualityScore >= 0.5 {
				qualityIndicator = "🟡 MIXED"
			} else if stats.AverageScore < 0.3 && stats.ConsistencyScore > 0.8 {
				qualityIndicator = "🔴 CONSISTENTLY BAD"
			} else {
				qualityIndicator = "🔴 POOR"
			}
			
			fmt.Printf("    %s: %.2f avg (±%.3f) - %.1f%% consistent - Quality: %.2f %s\n",
				stats.TestCaseName, stats.AverageScore, stats.ScoreStdDev, 
				stats.ConsistencyScore*100, stats.QualityScore, qualityIndicator)
		}
	}
	fmt.Println()
}

func PrintTestResult(result *types.TestCaseResult, err error) {
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		fmt.Printf("  DONE (%.2fs, score: %.2f)\n", result.ExecutionTime.Seconds(), result.Score)
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
