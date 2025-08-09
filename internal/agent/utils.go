package agent

import (
	"fmt"
	"os"
	"strings"
)

func NotifyUserIfReportNotIgnored(gitignorePath string) error {
	const reportFilename = "diffpector_report.md"

	if _, err := os.Stat(reportFilename); os.IsNotExist(err) {
		return nil
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("'%s' exists but is not in a .gitignore file", reportFilename)
		}
		return fmt.Errorf("could not read .gitignore file: %w", err)
	}

	if !strings.Contains(string(content), reportFilename) {
		return fmt.Errorf("'%s' exists but is not in your .gitignore file. Please consider adding it to avoid including it in the context of future analyses", reportFilename)
	}

	return nil
}
