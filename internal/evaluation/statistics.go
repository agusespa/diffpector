package evaluation

import (
	"math"

	"github.com/agusespa/diffpector/internal/types"
)

// StatisticsCalculator handles all statistical calculations for evaluation results
type StatisticsCalculator struct{}

func NewStatisticsCalculator() *StatisticsCalculator {
	return &StatisticsCalculator{}
}

// CalculateRunSummary calculates summary statistics for a single evaluation run
func (s *StatisticsCalculator) CalculateRunSummary(r *types.EvaluationRun) {
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

// CalculateEvaluationStats calculates aggregated statistics for multiple evaluation runs
func (s *StatisticsCalculator) CalculateEvaluationStats(result *types.EvaluationResult) {
	if len(result.IndividualRuns) == 0 {
		return
	}

	// Calculate overall statistics
	var scores, successRates, durations []float64
	for _, run := range result.IndividualRuns {
		scores = append(scores, run.AverageScore)
		successRates = append(successRates, run.SuccessRate)
		durations = append(durations, run.TotalDuration.Seconds())
	}

	result.AggregatedStats = types.EvaluationStats{
		AverageScore:       s.CalculateMean(scores),
		ScoreStdDev:        s.CalculateStdDev(scores),
		MinScore:           s.CalculateMin(scores),
		MaxScore:           s.CalculateMax(scores),
		AverageSuccessRate: s.CalculateMean(successRates),
		SuccessRateStdDev:  s.CalculateStdDev(successRates),
		AverageDuration:    s.CalculateMean(durations),
		DurationStdDev:     s.CalculateStdDev(durations),
	}

	// Calculate per-test-case statistics
	s.calculateTestCaseStats(result)
}

func (s *StatisticsCalculator) calculateTestCaseStats(result *types.EvaluationResult) {
	testCaseResults := make(map[string][]float64)
	testCaseSuccesses := make(map[string][]bool)

	for _, run := range result.IndividualRuns {
		for _, testResult := range run.Results {
			name := testResult.TestCase.Name
			testCaseResults[name] = append(testCaseResults[name], testResult.Score)
			testCaseSuccesses[name] = append(testCaseSuccesses[name], testResult.Success)
		}
	}

	result.TestCaseStats = make(map[string]types.TestCaseStats)
	for testName, scores := range testCaseResults {
		successes := testCaseSuccesses[testName]
		successCount := 0
		for _, success := range successes {
			if success {
				successCount++
			}
		}

		result.TestCaseStats[testName] = types.TestCaseStats{
			TestCaseName:     testName,
			AverageScore:     s.CalculateMean(scores),
			ScoreStdDev:      s.CalculateStdDev(scores),
			SuccessRate:      float64(successCount) / float64(len(successes)) * 100,
			ConsistencyScore: s.CalculateConsistency(scores),
		}
	}
}

// Statistical helper functions
func (s *StatisticsCalculator) CalculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (s *StatisticsCalculator) CalculateStdDev(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	mean := s.CalculateMean(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func (s *StatisticsCalculator) CalculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func (s *StatisticsCalculator) CalculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func (s *StatisticsCalculator) CalculateConsistency(values []float64) float64 {
	if len(values) <= 1 {
		return 1.0
	}
	stdDev := s.CalculateStdDev(values)
	mean := s.CalculateMean(values)
	if mean == 0 {
		return 1.0
	}
	// Consistency is inverse of coefficient of variation, capped at 1.0
	cv := stdDev / mean
	consistency := 1.0 / (1.0 + cv)
	if consistency > 1.0 {
		consistency = 1.0
	}
	return consistency
}
