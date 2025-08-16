package utils

import (
	"fmt"
	"sort"
	"strings"
)

type LineRange struct {
	Start int
	Count int
}

// GetDiffContext extracts the actual changed line ranges, not the entire hunk ranges
func GetDiffContext(diff string) map[string][]LineRange {
	if diff == "" {
		return make(map[string][]LineRange)
	}
	
	context := make(map[string][]LineRange)
	lines := strings.Split(diff, "\n")

	var currentFile string
	var currentLineNum int
	changedLines := make(map[string][]int) // Track individual changed line numbers

	for _, line := range lines {
		if strings.HasPrefix(line, "+++") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentFile = strings.TrimPrefix(parts[1], "b/")
			}
		} else if strings.HasPrefix(line, "@@") && currentFile != "" {
			// Parse hunk header to get starting line number for new file
			if hunkInfo := ParseHunkHeader(line); hunkInfo != nil {
				currentLineNum = hunkInfo.Start
			}
		} else if currentFile != "" && currentLineNum > 0 {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				// Added line - this line number is changed
				changedLines[currentFile] = append(changedLines[currentFile], currentLineNum)
				currentLineNum++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				// Deleted line - the current line position is changed
				changedLines[currentFile] = append(changedLines[currentFile], currentLineNum)
				// Don't increment currentLineNum for deleted lines
			} else if strings.HasPrefix(line, " ") {
				// Context line - increment line number
				currentLineNum++
			}
		}
	}

	// Convert individual line numbers to ranges
	for file, lineNums := range changedLines {
		if len(lineNums) == 0 {
			continue
		}

		// Remove duplicates and sort line numbers
		uniqueLines := make(map[int]bool)
		for _, line := range lineNums {
			uniqueLines[line] = true
		}
		
		sortedLines := make([]int, 0, len(uniqueLines))
		for line := range uniqueLines {
			sortedLines = append(sortedLines, line)
		}
		sort.Ints(sortedLines)

		// Group consecutive lines into ranges
		if len(sortedLines) == 0 {
			continue
		}
		
		start := sortedLines[0]
		count := 1

		for i := 1; i < len(sortedLines); i++ {
			if sortedLines[i] == sortedLines[i-1]+1 {
				// Consecutive line, extend range
				count++
			} else {
				// Gap found, save current range and start new one
				context[file] = append(context[file], LineRange{Start: start, Count: count})
				start = sortedLines[i]
				count = 1
			}
		}

		// Add final range
		context[file] = append(context[file], LineRange{Start: start, Count: count})
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
