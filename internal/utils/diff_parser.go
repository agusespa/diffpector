package utils

import (
	"fmt"
	"strings"
)

type LineRange struct {
	Start int
	Count int
}

// GetDiffContext extracts the changed line ranges for better context analysis
func GetDiffContext(diff string) map[string][]LineRange {
	context := make(map[string][]LineRange)
	lines := strings.Split(diff, "\n")

	var currentFile string

	for _, line := range lines {
		if strings.HasPrefix(line, "+++") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentFile = strings.TrimPrefix(parts[1], "b/")
			}
		} else if strings.HasPrefix(line, "@@") && currentFile != "" {
			if lineRange := ParseHunkHeader(line); lineRange != nil {
				context[currentFile] = append(context[currentFile], *lineRange)
			}
		}
	}

	return context
}

func ParseHunkHeader(header string) *LineRange {
	// Example: @@ -1,4 +1,6 @@
	fields := strings.Fields(header)
	if len(fields) < 3 {
		return nil
	}

	newRange := fields[2]
	if !strings.HasPrefix(newRange, "+") {
		return nil
	}

	newRange = strings.TrimPrefix(newRange, "+")
	parts := strings.Split(newRange, ",")

	start := 0
	count := 1

	if len(parts) > 0 {
		if _, err := fmt.Sscanf(parts[0], "%d", &start); err != nil {
			return nil
		}
	}
	if len(parts) > 1 {
		if _, err := fmt.Sscanf(parts[1], "%d", &count); err != nil {
			count = 1
		}
	}

	return &LineRange{Start: start, Count: count}
}
