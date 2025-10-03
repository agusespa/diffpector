package utils

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
)

func GetDiffContext(diffData types.DiffData, allSymbols []types.Symbol, fileContent []byte) (types.ContextResult, error) {
	fileLines := strings.Split(string(fileContent), "\n")

	changedLinesSet := getDiffChangedLines(diffData.Diff)
	if len(changedLinesSet) == 0 {
		return types.ContextResult{}, nil
	}

	var contextBlocks []string
	var affectedSymbols []types.SymbolUsage

	for _, symbol := range allSymbols {
		if containsChangedLines(symbol, changedLinesSet) && isDeclaration(symbol.Type) {
			affectedSymbols = append(affectedSymbols, types.SymbolUsage{Symbol: symbol})
			content := extractSymbolContent(symbol, fileLines)
			contextBlocks = append(contextBlocks, content)
		}
	}

	// fmt.Println("DIFF CONTEXT at diff_parser.go: ", contextBlocks)

	return types.ContextResult{
		Context:         strings.Join(contextBlocks, "\n\n"),
		AffectedSymbols: affectedSymbols,
	}, nil
}

func getDiffChangedLines(diffContent string) map[int]bool {
	addedLines := make(map[int]bool)
	lines := strings.Split(diffContent, "\n")

	hunkRegex := regexp.MustCompile(`^@@\s+-\d+(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s+@@`)

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		matches := hunkRegex.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		startLine, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		newFileLineNum := startLine
		for i++; i < len(lines); i++ {
			hunkLine := lines[i]

			if strings.HasPrefix(hunkLine, "@@") {
				i--
				break
			}

			switch {
			case strings.HasPrefix(hunkLine, "+"):
				addedLines[newFileLineNum] = true
				newFileLineNum++
			case strings.HasPrefix(hunkLine, " "):
				newFileLineNum++
			}
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

func isDeclaration(symbolType string) bool {
	declarationTypes := []string{
		// Go declarations
		"func_decl",
		"method_decl",
		"type_decl",
		"const_decl",
		"var_decl",
		"field_decl",
		"iface_method_decl",
		"import_decl",
	}

	return slices.Contains(declarationTypes, symbolType)
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
