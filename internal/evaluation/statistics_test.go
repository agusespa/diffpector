package evaluation

import (
	"math"
	"testing"
	"time"

	"github.com/agusespa/diffpector/internal/types"
)

func TestNewStatisticsCalculator(t *testing.T) {
	calc := NewStatisticsCalculator()
	if calc == nil {
		t.Error("Expected non-nil statistics calculator")
	}
}

func TestCalculateRunSummary(t *testing.T) {
	calc := NewStatisticsCalculator()

	run := &types.EvaluationRun{
		Results: []types.TestCaseResult{
			{Score: 0.8, Success: true},
			{Score: 0.6, Success: true},
			{Score: 0.0, Success: false},
			{Score: 1.0, Success: true},
		},
	}

	calc.CalculateRunSummary(run)

	expectedAverage := (0.8 + 0.6 + 0.0 + 1.0) / 4.0
	if run.AverageScore != expectedAverage {
		t.Errorf("Expected average score %.2f, got %.2f", expectedAverage, run.AverageScore)
	}

	expectedSuccessRate := (3.0 / 4.0) * 100.0
	if run.SuccessRate != expectedSuccessRate {
		t.Errorf("Expected success rate %.2f, got %.2f", expectedSuccessRate, run.SuccessRate)
	}
}

func TestCalculateRunSummary_EmptyResults(t *testing.T) {
	calc := NewStatisticsCalculator()

	run := &types.EvaluationRun{
		Results: []types.TestCaseResult{},
	}

	calc.CalculateRunSummary(run)

	// Should not panic and should leave values at zero
	if run.AverageScore != 0.0 {
		t.Errorf("Expected average score 0.0 for empty results, got %.2f", run.AverageScore)
	}

	if run.SuccessRate != 0.0 {
		t.Errorf("Expected success rate 0.0 for empty results, got %.2f", run.SuccessRate)
	}
}

func TestCalculateEvaluationStats(t *testing.T) {
	calc := NewStatisticsCalculator()

	result := &types.EvaluationResult{
		IndividualRuns: []types.EvaluationRun{
			{
				AverageScore:  0.8,
				SuccessRate:   80.0,
				TotalDuration: 10 * time.Second,
				Results: []types.TestCaseResult{
					{TestCase: types.TestCase{Name: "test1"}, Score: 0.9, Success: true},
					{TestCase: types.TestCase{Name: "test2"}, Score: 0.7, Success: true},
				},
			},
			{
				AverageScore:  0.6,
				SuccessRate:   60.0,
				TotalDuration: 15 * time.Second,
				Results: []types.TestCaseResult{
					{TestCase: types.TestCase{Name: "test1"}, Score: 0.8, Success: true},
					{TestCase: types.TestCase{Name: "test2"}, Score: 0.4, Success: false},
				},
			},
		},
	}

	calc.CalculateEvaluationStats(result)

	// Check aggregated stats
	expectedAvgScore := (0.8 + 0.6) / 2.0
	if result.AggregatedStats.AverageScore != expectedAvgScore {
		t.Errorf("Expected average score %.2f, got %.2f", expectedAvgScore, result.AggregatedStats.AverageScore)
	}

	expectedAvgSuccessRate := (80.0 + 60.0) / 2.0
	if result.AggregatedStats.AverageSuccessRate != expectedAvgSuccessRate {
		t.Errorf("Expected average success rate %.2f, got %.2f", expectedAvgSuccessRate, result.AggregatedStats.AverageSuccessRate)
	}

	expectedAvgDuration := (10.0 + 15.0) / 2.0
	if result.AggregatedStats.AverageDuration != expectedAvgDuration {
		t.Errorf("Expected average duration %.2f, got %.2f", expectedAvgDuration, result.AggregatedStats.AverageDuration)
	}

	// Check min/max scores
	if result.AggregatedStats.MinScore != 0.6 {
		t.Errorf("Expected min score 0.6, got %.2f", result.AggregatedStats.MinScore)
	}

	if result.AggregatedStats.MaxScore != 0.8 {
		t.Errorf("Expected max score 0.8, got %.2f", result.AggregatedStats.MaxScore)
	}

	// Check test case stats
	if len(result.TestCaseStats) != 2 {
		t.Errorf("Expected 2 test case stats, got %d", len(result.TestCaseStats))
	}

	test1Stats := result.TestCaseStats["test1"]
	expectedTest1Avg := (0.9 + 0.8) / 2.0
	if math.Abs(test1Stats.AverageScore-expectedTest1Avg) > 0.001 {
		t.Errorf("Expected test1 average score %.2f, got %.2f", expectedTest1Avg, test1Stats.AverageScore)
	}

	if test1Stats.SuccessRate != 100.0 {
		t.Errorf("Expected test1 success rate 100.0, got %.2f", test1Stats.SuccessRate)
	}

	test2Stats := result.TestCaseStats["test2"]
	expectedTest2Avg := (0.7 + 0.4) / 2.0
	if test2Stats.AverageScore != expectedTest2Avg {
		t.Errorf("Expected test2 average score %.2f, got %.2f", expectedTest2Avg, test2Stats.AverageScore)
	}

	if test2Stats.SuccessRate != 50.0 {
		t.Errorf("Expected test2 success rate 50.0, got %.2f", test2Stats.SuccessRate)
	}
}

func TestCalculateEvaluationStats_EmptyRuns(t *testing.T) {
	calc := NewStatisticsCalculator()

	result := &types.EvaluationResult{
		IndividualRuns: []types.EvaluationRun{},
	}

	calc.CalculateEvaluationStats(result)

	// Should not panic and should leave stats at zero values
	if result.AggregatedStats.AverageScore != 0.0 {
		t.Errorf("Expected average score 0.0 for empty runs, got %.2f", result.AggregatedStats.AverageScore)
	}
}

func TestCalculateMean(t *testing.T) {
	calc := NewStatisticsCalculator()

	tests := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{1.0, 2.0, 3.0}, 2.0},
		{[]float64{0.5, 1.5}, 1.0},
		{[]float64{5.0}, 5.0},
		{[]float64{}, 0.0},
	}

	for _, test := range tests {
		result := calc.CalculateMean(test.values)
		if result != test.expected {
			t.Errorf("CalculateMean(%v) = %.2f, expected %.2f", test.values, result, test.expected)
		}
	}
}

func TestCalculateStdDev(t *testing.T) {
	calc := NewStatisticsCalculator()

	// Test with known values
	values := []float64{2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0}
	result := calc.CalculateStdDev(values)

	// Expected standard deviation is approximately 2.138
	if result < 2.0 || result > 2.2 {
		t.Errorf("CalculateStdDev(%v) = %.3f, expected around 2.138", values, result)
	}

	// Test with single value
	singleValue := []float64{5.0}
	result = calc.CalculateStdDev(singleValue)
	if result != 0.0 {
		t.Errorf("CalculateStdDev with single value should return 0.0, got %.3f", result)
	}

	// Test with empty slice
	empty := []float64{}
	result = calc.CalculateStdDev(empty)
	if result != 0.0 {
		t.Errorf("CalculateStdDev with empty slice should return 0.0, got %.3f", result)
	}
}

func TestCalculateMinMax(t *testing.T) {
	calc := NewStatisticsCalculator()

	values := []float64{3.0, 1.0, 4.0, 1.0, 5.0, 9.0, 2.0}

	min := calc.CalculateMin(values)
	if min != 1.0 {
		t.Errorf("CalculateMin(%v) = %.2f, expected 1.0", values, min)
	}

	max := calc.CalculateMax(values)
	if max != 9.0 {
		t.Errorf("CalculateMax(%v) = %.2f, expected 9.0", values, max)
	}

	// Test with empty slice
	empty := []float64{}
	min = calc.CalculateMin(empty)
	if min != 0.0 {
		t.Errorf("CalculateMin with empty slice should return 0.0, got %.2f", min)
	}

	max = calc.CalculateMax(empty)
	if max != 0.0 {
		t.Errorf("CalculateMax with empty slice should return 0.0, got %.2f", max)
	}
}

func TestCalculateConsistency(t *testing.T) {
	calc := NewStatisticsCalculator()

	// Perfect consistency (all same values)
	perfect := []float64{5.0, 5.0, 5.0, 5.0}
	result := calc.CalculateConsistency(perfect)
	if result != 1.0 {
		t.Errorf("CalculateConsistency with identical values should return 1.0, got %.3f", result)
	}

	// Some variation
	varied := []float64{4.0, 5.0, 6.0}
	result = calc.CalculateConsistency(varied)
	if result <= 0.0 || result > 1.0 {
		t.Errorf("CalculateConsistency should return value between 0 and 1, got %.3f", result)
	}

	// Single value
	single := []float64{5.0}
	result = calc.CalculateConsistency(single)
	if result != 1.0 {
		t.Errorf("CalculateConsistency with single value should return 1.0, got %.3f", result)
	}

	// Empty slice
	empty := []float64{}
	result = calc.CalculateConsistency(empty)
	if result != 1.0 {
		t.Errorf("CalculateConsistency with empty slice should return 1.0, got %.3f", result)
	}

	// Zero mean case
	zeros := []float64{0.0, 0.0, 0.0}
	result = calc.CalculateConsistency(zeros)
	if result != 1.0 {
		t.Errorf("CalculateConsistency with zero mean should return 1.0, got %.3f", result)
	}
}
