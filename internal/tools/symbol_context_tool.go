package tools

import (
	"fmt"
	"strings"
)

type SymbolContextTool struct {
	parser   *SymbolParser
	gatherer *ContextGatherer
}

func NewSymbolContextTool() *SymbolContextTool {
	return &SymbolContextTool{
		parser:   NewSymbolParser(),
		gatherer: NewContextGatherer(),
	}
}

func (t *SymbolContextTool) Name() string {
	return "symbol_context"
}

func (t *SymbolContextTool) Description() string {
	return "Analyze code changes to find symbols and gather related context"
}

func (t *SymbolContextTool) Execute(args map[string]any) (string, error) {
	diff, ok := args["diff"].(string)
	if !ok {
		return "", fmt.Errorf("diff parameter required")
	}

	changedFiles, hasFiles := args["changed_files"].([]string)
	if !hasFiles {
		return "", fmt.Errorf("changed_files parameter required")
	}

	if len(changedFiles) == 0 {
		return "No changed files provided", nil
	}

	fileContents, hasContents := args["file_contents"].(map[string]string)
	if !hasContents {
		return "", fmt.Errorf("file_contents parameter required for symbol analysis")
	}

	allSymbols := t.parser.ParseChangedFiles(fileContents)

	if len(allSymbols) == 0 {
		return "No symbols found in the changed files", nil
	}

	// Get diff context to understand which symbols are likely affected
	diffContext := t.parser.GetDiffContext(diff)

	// Filter symbols that are likely affected by the changes
	affectedSymbols := t.filterAffectedSymbols(allSymbols, diffContext)

	// Gather context for affected symbols
	contextResults := t.gatherer.GatherContextForSymbols(affectedSymbols)

	// Format the results
	return t.formatResults(contextResults)
}

func (t *SymbolContextTool) formatResults(results []ContextResult) (string, error) {
	var output strings.Builder

	for i, result := range results {
		if i > 0 {
			output.WriteString("\n" + strings.Repeat("=", 80) + "\n\n")
		}

		symbol := result.Symbol
		output.WriteString(fmt.Sprintf("Symbol: %s (%s)\n", symbol.Name, symbol.Type))
		output.WriteString(fmt.Sprintf("Location: %s:%d\n", symbol.FilePath, symbol.Line))

		if len(result.Usages) > 0 {
			output.WriteString(fmt.Sprintf("Found %d usage(s):\n\n", len(result.Usages)))

			// Group usages by file for better readability
			fileUsages := make(map[string][]Usage)
			for _, usage := range result.Usages {
				fileUsages[usage.FilePath] = append(fileUsages[usage.FilePath], usage)
			}

			for filePath, usages := range fileUsages {
				output.WriteString(fmt.Sprintf("In %s:\n", filePath))

				for j, usage := range usages {
					if j > 0 {
						output.WriteString("\n" + strings.Repeat("-", 40) + "\n")
					}

					output.WriteString(fmt.Sprintf("\nUsage at line %d", usage.Line))

					// Function context
					if usage.ContainingFunc != "unknown" && usage.ContainingFunc != "" {
						output.WriteString(fmt.Sprintf(" (in function: %s)", usage.ContainingFunc))
					}
					output.WriteString(":\n")

					// Surrounding context - this is the key improvement
					if len(usage.SurroundingLines) > 0 {
						for _, line := range usage.SurroundingLines {
							output.WriteString(fmt.Sprintf("%s\n", line))
						}
					} else {
						// Fallback to simple context
						output.WriteString(fmt.Sprintf("  %d: %s\n", usage.Line, usage.Context))
					}

					output.WriteString("\n")
				}
				output.WriteString("\n")
			}
		} else {
			output.WriteString("No usages found\n")
		}

		if len(result.RelatedFiles) > 0 {
			output.WriteString(fmt.Sprintf("Related files: %s\n", strings.Join(result.RelatedFiles, ", ")))
		}
	}

	return output.String(), nil
}

func (t *SymbolContextTool) filterAffectedSymbols(allSymbols []Symbol, diffContext map[string][]LineRange) []Symbol {
	var affectedSymbols []Symbol

	for _, symbol := range allSymbols {
		if t.isSymbolAffected(symbol, diffContext) {
			affectedSymbols = append(affectedSymbols, symbol)
		}
	}

	// If no symbols seem directly affected, return all symbols from changed files
	// This ensures we don't miss anything due to imperfect heuristics
	if len(affectedSymbols) == 0 {
		return allSymbols
	}

	return affectedSymbols
}

func (t *SymbolContextTool) isSymbolAffected(symbol Symbol, diffContext map[string][]LineRange) bool {
	ranges, exists := diffContext[symbol.FilePath]
	if !exists {
		return false
	}

	// Check if the symbol's line is within or near any changed range
	for _, lineRange := range ranges {
		symbolLine := symbol.Line
		rangeStart := lineRange.Start
		rangeEnd := lineRange.Start + lineRange.Count - 1

		// Symbol is affected if:
		// 1. It's directly in the changed range
		// 2. It's within a reasonable distance (e.g., 10 lines) of the changed range
		//    This catches cases where the function signature is unchanged but body is modified
		const proximityThreshold = 10

		if (symbolLine >= rangeStart && symbolLine <= rangeEnd) ||
			(symbolLine >= rangeStart-proximityThreshold && symbolLine <= rangeEnd+proximityThreshold) {
			return true
		}
	}

	return false
}
