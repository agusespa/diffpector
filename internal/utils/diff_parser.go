package utils

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
)

func GetDiffContext(diffData types.DiffData, allSymbols []types.Symbol) (types.ContextResult, error) {
	fileContent, err := os.ReadFile(diffData.AbsolutePath)
	if err != nil {
		return types.ContextResult{}, fmt.Errorf("failed to read file: %w", err)
	}

	changedLines := parseGitDiffForAddedLines(diffData.Diff)
	if len(changedLines) == 0 {
		return types.ContextResult{}, nil
	}

	changedLinesSet := make(map[int]bool)
	for _, line := range changedLines {
		changedLinesSet[line] = true
	}

	fileLines := strings.Split(string(fileContent), "\n")

	var contextBlocks []string
	var affectedSymbols []types.SymbolUsage

	for _, symbol := range allSymbols {
		if containsChangedLines(symbol, changedLinesSet) {
			affectedSymbols = append(affectedSymbols, types.SymbolUsage{Symbol: symbol})
			content := extractSymbolContent(symbol, fileLines)
			if content != "" {
				contextBlocks = append(contextBlocks, content)
			}
		}
	}

	return types.ContextResult{
		Context:         strings.Join(contextBlocks, "\n\n"),
		AffectedSymbols: affectedSymbols,
	}, nil
}

func parseGitDiffForAddedLines(diffContent string) []int {
	var addedLines []int
	lines := strings.Split(diffContent, "\n")

	hunkRegex := regexp.MustCompile(`^@@\s+-\d+(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s+@@`)

	var currentLine int
	inHunk := false

	for _, line := range lines {
		if matches := hunkRegex.FindStringSubmatch(line); len(matches) > 1 {
			startLine, _ := strconv.Atoi(matches[1])
			currentLine = startLine
			inHunk = true
			continue
		}

		if !inHunk {
			continue
		}

		if strings.HasPrefix(line, "+") {
			addedLines = append(addedLines, currentLine)
			currentLine++
		} else if strings.HasPrefix(line, " ") {
			currentLine++
		}
	}

	return addedLines
}

func containsChangedLines(symbol types.Symbol, changedLines map[int]bool) bool {
	for line := symbol.StartLine; line <= symbol.EndLine; line++ {
		if changedLines[line] {
			return true
		}
	}
	return false
}

func extractSymbolContent(symbol types.Symbol, fileLines []string) string {
	startLine := symbol.StartLine - 1
	endLine := symbol.EndLine

	if startLine < 0 {
		startLine = 0
	}
	if endLine > len(fileLines) {
		endLine = len(fileLines)
	}
	if startLine >= endLine {
		return ""
	}

	symbolLines := fileLines[startLine:endLine]
	return strings.Join(symbolLines, "\n")
}
