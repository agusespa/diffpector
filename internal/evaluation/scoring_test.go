package evaluation

import (
	"math"
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name     string
		expected types.ExpectedResults
		actual   []types.Issue
		want     float64
	}{
		{
			name: "Perfect Match",
			expected: types.ExpectedResults{
				ShouldFindIssues: true,
				ExpectedSeverity: []string{"CRITICAL"},
				MinIssues:        1,
			},
			actual: []types.Issue{
				{Severity: "CRITICAL", FilePath: "main.go"},
			},
			want: 1.0,
		},
		{
			name: "False Positive",
			expected: types.ExpectedResults{
				ShouldFindIssues: false,
			},
			actual: []types.Issue{
				{Severity: "INFO", FilePath: "main.go"},
			},
			want: 0.0,
		},
		{
			name: "False Negative",
			expected: types.ExpectedResults{
				ShouldFindIssues: true,
			},
			actual: []types.Issue{},
			want:   0.0,
		},
		{
			name: "Severity Mismatch (Penalty)",
			expected: types.ExpectedResults{
				ShouldFindIssues: true,
				ExpectedSeverity: []string{"CRITICAL"},
			},
			actual: []types.Issue{
				{Severity: "WARNING", FilePath: "main.go"},
			},
			want: 0.5, // 1.0 - 0.5
		},
		{
			name: "Count Mismatch (Penalty)",
			expected: types.ExpectedResults{
				ShouldFindIssues: true,
				MinIssues:        5,
			},
			actual: []types.Issue{
				{Severity: "CRITICAL"}, // 1 issue
			},
			want: 0.8, // 1.0 - 0.2
		},
		{
			name: "Found Issue when None Expected (Strict)",
			expected: types.ExpectedResults{
				ShouldFindIssues: false,
			},
			actual: []types.Issue{
				{Severity: "WARNING"},
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateScore(tt.expected, tt.actual)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("CalculateScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
