package tools

import (
	"reflect"
	"testing"

	"github.com/agusespa/diffpector/internal/utils"
)

func TestFilterAffectedSymbols(t *testing.T) {
	testCases := []struct {
		name        string
		symbols     []Symbol
		diffContext map[string][]utils.LineRange
		want        []Symbol
	}{
		{
			name: "Basic case with multiple changes",
			symbols: []Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
				{Name: "bar", FilePath: "file.go", StartLine: 15, EndLine: 20},
				{Name: "baz", FilePath: "file.go", StartLine: 25, EndLine: 30},
			},
			diffContext: map[string][]utils.LineRange{
				"file.go": {
					{Start: 8, Count: 2},  // Change inside "foo"
					{Start: 18, Count: 1}, // Change inside "bar"
				},
			},
			want: []Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
				{Name: "bar", FilePath: "file.go", StartLine: 15, EndLine: 20},
			},
		},
		{
			name: "Change on start and end lines",
			symbols: []Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
				{Name: "bar", FilePath: "file.go", StartLine: 11, EndLine: 15},
			},
			diffContext: map[string][]utils.LineRange{
				"file.go": {
					{Start: 5, Count: 1},  // Change on the start line of "foo"
					{Start: 15, Count: 1}, // Change on the end line of "bar"
				},
			},
			want: []Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
				{Name: "bar", FilePath: "file.go", StartLine: 11, EndLine: 15},
			},
		},
		{
			name: "No affected symbols",
			symbols: []Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
			},
			diffContext: map[string][]utils.LineRange{
				"file.go": {
					{Start: 12, Count: 2}, // Change outside any symbol range
				},
			},
			want: []Symbol{},
		},
		{
			name: "Multiple symbols on the same line",
			symbols: []Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 5},
				{Name: "bar", FilePath: "file.go", StartLine: 5, EndLine: 5},
			},
			diffContext: map[string][]utils.LineRange{
				"file.go": {
					{Start: 5, Count: 1}, // Change on the single line where both symbols reside
				},
			},
			want: []Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 5},
				{Name: "bar", FilePath: "file.go", StartLine: 5, EndLine: 5},
			},
		},
		{
			name: "Change in an unrelated file",
			symbols: []Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
			},
			diffContext: map[string][]utils.LineRange{
				"other_file.go": {
					{Start: 1, Count: 1},
				},
			},
			want: []Symbol{},
		},
		// New test case to verify the fix for duplicate symbols.
		{
			name: "Single symbol with multiple overlapping changes",
			symbols: []Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
			},
			diffContext: map[string][]utils.LineRange{
				"file.go": {
					{Start: 6, Count: 1}, // First change inside "foo"
					{Start: 8, Count: 1}, // Second change inside "foo"
				},
			},
			want: []Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterAffectedSymbols(tc.symbols, tc.diffContext)

			if len(got) == 0 && len(tc.want) == 0 {
				return
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
