package utils

import (
	"strings"
)

func ParseStagedFiles(commandOutput string) []string {
	if commandOutput == "" {
		return []string{}
	}

	var files []string
	filePaths := strings.SplitSeq(strings.TrimSpace(commandOutput), "\n")

	for file := range filePaths {
		file = strings.TrimSpace(file)
		if file != "" {
			files = append(files, file)
		}
	}

	return files
}
