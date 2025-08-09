package utils

import (
	"strings"
)

func ParseStagedFiles(output string) []string {
	if output == "" {
		return []string{}
	}

	var files []string
	lines := strings.SplitSeq(strings.TrimSpace(output), "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files
}
