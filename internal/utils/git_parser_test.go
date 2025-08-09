package utils

import (
	"reflect"
	"testing"
)

func TestParseStagedFiles(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name:   "empty input",
			input:  "",
			expect: []string{},
		},
		{
			name:   "single file",
			input:  "file1.go",
			expect: []string{"file1.go"},
		},
		{
			name:   "multiple files",
			input:  "file1.go\nfile2.go\nfile3.go",
			expect: []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:   "files with trailing and leading spaces",
			input:  "  file1.go  \n  file2.go\nfile3.go  ",
			expect: []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:   "input with blank lines",
			input:  "file1.go\n\nfile2.go\n\n",
			expect: []string{"file1.go", "file2.go"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseStagedFiles(tc.input)
			if !reflect.DeepEqual(got, tc.expect) {
				t.Errorf("ParseStagedFiles(%q) = %v; want %v", tc.input, got, tc.expect)
			}
		})
	}
}
