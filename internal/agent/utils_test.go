package agent

import (
	"os"
	"testing"
)

func TestNotifyUserIfReportNotIgnored(t *testing.T) {
	const reportFilename = "diffpector_report.md"
	const gitignoreFilename = ".test.gitignore"

	tests := []struct {
		name              string
		reportFileExists  bool
		gitignoreExists   bool
		gitignoreContains string
		expectedError     string
	}{
		{
			name:             "report does not exist",
			reportFileExists: false,
			expectedError:    "",
		},
		{
			name:             "report exists, gitignore does not",
			reportFileExists: true,
			gitignoreExists:  false,
			expectedError:    "'diffpector_report.md' exists but is not in a .gitignore file",
		},
		{
			name:              "report exists, gitignore contains it",
			reportFileExists:  true,
			gitignoreExists:   true,
			gitignoreContains: reportFilename,
			expectedError:     "",
		},
		{
			name:              "report exists, gitignore does not contain it",
			reportFileExists:  true,
			gitignoreExists:   true,
			gitignoreContains: "other_file.txt",
			expectedError:     "'diffpector_report.md' exists but is not in your .gitignore file. Please consider adding it to avoid including it in the context of future analyses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Remove(reportFilename)
			_ = os.Remove(gitignoreFilename)

			if tt.reportFileExists {
				_, err := os.Create(reportFilename)
				if err != nil {
					t.Fatalf("Failed to create report file: %v", err)
				}
			}

			if tt.gitignoreExists {
				err := os.WriteFile(gitignoreFilename, []byte(tt.gitignoreContains), 0644)
				if err != nil {
					t.Fatalf("Failed to create gitignore file: %v", err)
				}
			}

			err := NotifyUserIfReportNotIgnored(gitignoreFilename)

			var gotError string
			if err != nil {
				gotError = err.Error()
			}

			if gotError != tt.expectedError {
				t.Errorf("Expected error '%s', but got '%s'", tt.expectedError, gotError)
			}

			_ = os.Remove(reportFilename)
			_ = os.Remove(gitignoreFilename)
		})
	}
}
