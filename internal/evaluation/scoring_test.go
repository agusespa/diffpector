package evaluation

import (
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name     string
		expected types.ExpectedResults
		actual   []types.Issue
		wantMin  float64
		wantMax  float64
	}{
		{
			name: "perfect match - should find issues and does",
			expected: types.ExpectedResults{
				ShouldFindIssues: true,
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "test issue"},
			},
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name: "perfect match - should not find issues and doesn't",
			expected: types.ExpectedResults{
				ShouldFindIssues: false,
			},
			actual:  []types.Issue{},
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name: "mismatch - should find issues but doesn't",
			expected: types.ExpectedResults{
				ShouldFindIssues: true,
			},
			actual:  []types.Issue{},
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name: "mismatch - should not find issues but does",
			expected: types.ExpectedResults{
				ShouldFindIssues: false,
			},
			actual: []types.Issue{
				{Severity: "MINOR", FilePath: "test.go", Description: "test issue"},
			},
			wantMin: 0.5, // Partial credit due to new scoring system (1.0 - 0.1 penalty for MINOR false positive)
			wantMax: 1.0,
		},
		{
			name: "complex expectations - all match",
			expected: types.ExpectedResults{
				ShouldFindIssues: true,
				MinIssues:        2,
				MaxIssues:        4,
				ExpectedSeverity: []string{"error", "warning"},
				ExpectedFiles:    []string{"test.go", "main.go"},
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "error in test"},
				{Severity: "warning", FilePath: "main.go", Description: "warning in main"},
				{Severity: "error", FilePath: "test.go", Description: "another error"},
			},
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name: "no expectations - should get full score",
			expected: types.ExpectedResults{
				ShouldFindIssues: true,
			},
			actual: []types.Issue{
				{Severity: "info", FilePath: "any.go", Description: "any issue"},
			},
			wantMin: 1.0,
			wantMax: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateScore(tt.expected, tt.actual)
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("CalculateScore() = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestCalculateIssueFoundScore(t *testing.T) {
	tests := []struct {
		name     string
		expected types.ExpectedResults
		actual   []types.Issue
		want     float64
	}{
		{
			name: "should find issues and does",
			expected: types.ExpectedResults{
				ShouldFindIssues: true,
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "test issue"},
			},
			want: 1.0,
		},
		{
			name: "should not find issues and doesn't",
			expected: types.ExpectedResults{
				ShouldFindIssues: false,
			},
			actual: []types.Issue{},
			want:   1.0,
		},
		{
			name: "should find issues but doesn't",
			expected: types.ExpectedResults{
				ShouldFindIssues: true,
			},
			actual: []types.Issue{},
			want:   0.0,
		},
		{
			name: "should not find issues but does",
			expected: types.ExpectedResults{
				ShouldFindIssues: false,
			},
			actual: []types.Issue{
				{Severity: "MINOR", FilePath: "test.go", Description: "test issue"},
			},
			want: 0.9, // 1.0 - 0.1 penalty for MINOR false positive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateIssueFoundScore(tt.expected, tt.actual)
			if score != tt.want {
				t.Errorf("calculateIssueFoundScore() = %v, want %v", score, tt.want)
			}
		})
	}
}

func TestCalculateIssueCountScore(t *testing.T) {
	tests := []struct {
		name     string
		expected types.ExpectedResults
		actual   []types.Issue
		want     float64
	}{
		{
			name: "no count expectations - not applicable",
			expected: types.ExpectedResults{
				MinIssues: 0,
				MaxIssues: 0,
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "test issue"},
			},
			want: -1.0,
		},
		{
			name: "within min/max range",
			expected: types.ExpectedResults{
				MinIssues: 2,
				MaxIssues: 4,
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "issue 1"},
				{Severity: "warning", FilePath: "test.go", Description: "issue 2"},
				{Severity: "info", FilePath: "test.go", Description: "issue 3"},
			},
			want: 1.0,
		},
		{
			name: "below minimum",
			expected: types.ExpectedResults{
				MinIssues: 3,
				MaxIssues: 5,
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "issue 1"},
			},
			want: 0.2333333333333333, // (1/3) * 0.7 = partial credit for being under minimum
		},
		{
			name: "above maximum",
			expected: types.ExpectedResults{
				MinIssues: 1,
				MaxIssues: 2,
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "issue 1"},
				{Severity: "warning", FilePath: "test.go", Description: "issue 2"},
				{Severity: "info", FilePath: "test.go", Description: "issue 3"},
			},
			want: 0.7, // 1.0 - (1 excess / 1 range) * 0.3 = 0.7
		},
		{
			name: "only min specified - meets minimum",
			expected: types.ExpectedResults{
				MinIssues: 2,
				MaxIssues: 0,
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "issue 1"},
				{Severity: "warning", FilePath: "test.go", Description: "issue 2"},
				{Severity: "info", FilePath: "test.go", Description: "issue 3"},
			},
			want: 1.0,
		},
		{
			name: "only max specified - within maximum",
			expected: types.ExpectedResults{
				MinIssues: 0,
				MaxIssues: 3,
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "issue 1"},
				{Severity: "warning", FilePath: "test.go", Description: "issue 2"},
			},
			want: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateIssueCountScore(tt.expected, tt.actual)
			if score != tt.want {
				t.Errorf("calculateIssueCountScore() = %v, want %v", score, tt.want)
			}
		})
	}
}

func TestCalculateSeverityMatchScore(t *testing.T) {
	tests := []struct {
		name     string
		expected types.ExpectedResults
		actual   []types.Issue
		want     float64
	}{
		{
			name: "no severity expectations - not applicable",
			expected: types.ExpectedResults{
				ExpectedSeverity: []string{},
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "test issue"},
			},
			want: -1.0,
		},
		{
			name: "all expected severities found",
			expected: types.ExpectedResults{
				ExpectedSeverity: []string{"error", "warning"},
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "error issue"},
				{Severity: "warning", FilePath: "test.go", Description: "warning issue"},
				{Severity: "info", FilePath: "test.go", Description: "info issue"},
			},
			want: 1.0,
		},
		{
			name: "partial severity match",
			expected: types.ExpectedResults{
				ExpectedSeverity: []string{"error", "warning", "critical"},
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "error issue"},
				{Severity: "info", FilePath: "test.go", Description: "info issue"},
			},
			want: 1.0 / 3.0, // Only 1 out of 3 expected severities found
		},
		{
			name: "no expected severities found",
			expected: types.ExpectedResults{
				ExpectedSeverity: []string{"error", "warning"},
			},
			actual: []types.Issue{
				{Severity: "info", FilePath: "test.go", Description: "info issue"},
			},
			want: 0.0,
		},
		{
			name: "duplicate severities in actual issues",
			expected: types.ExpectedResults{
				ExpectedSeverity: []string{"error"},
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test1.go", Description: "error 1"},
				{Severity: "error", FilePath: "test2.go", Description: "error 2"},
			},
			want: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateSeverityMatchScore(tt.expected, tt.actual)
			if score != tt.want {
				t.Errorf("calculateSeverityMatchScore() = %v, want %v", score, tt.want)
			}
		})
	}
}

func TestCalculateFileMatchScore(t *testing.T) {
	tests := []struct {
		name     string
		expected types.ExpectedResults
		actual   []types.Issue
		want     float64
	}{
		{
			name: "no file expectations - not applicable",
			expected: types.ExpectedResults{
				ExpectedFiles: []string{},
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "test issue"},
			},
			want: -1.0,
		},
		{
			name: "all expected files have issues",
			expected: types.ExpectedResults{
				ExpectedFiles: []string{"test.go", "main.go"},
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "test issue"},
				{Severity: "warning", FilePath: "main.go", Description: "main issue"},
				{Severity: "info", FilePath: "other.go", Description: "other issue"},
			},
			want: 1.0,
		},
		{
			name: "partial file match",
			expected: types.ExpectedResults{
				ExpectedFiles: []string{"test.go", "main.go", "util.go"},
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "test issue"},
				{Severity: "info", FilePath: "other.go", Description: "other issue"},
			},
			want: 1.0 / 3.0, // Only 1 out of 3 expected files have issues
		},
		{
			name: "no expected files have issues",
			expected: types.ExpectedResults{
				ExpectedFiles: []string{"test.go", "main.go"},
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "other.go", Description: "other issue"},
			},
			want: 0.0,
		},
		{
			name: "multiple issues in same expected file",
			expected: types.ExpectedResults{
				ExpectedFiles: []string{"test.go"},
			},
			actual: []types.Issue{
				{Severity: "error", FilePath: "test.go", Description: "error 1"},
				{Severity: "warning", FilePath: "test.go", Description: "error 2"},
			},
			want: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateFileMatchScore(tt.expected, tt.actual)
			if score != tt.want {
				t.Errorf("calculateFileMatchScore() = %v, want %v", score, tt.want)
			}
		})
	}
}
