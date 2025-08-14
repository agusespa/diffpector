package tools

import (
	"fmt"
	"path/filepath"

	"github.com/agusespa/diffpector/internal/types"
	"github.com/agusespa/diffpector/internal/utils"
)

type SymbolContextTool struct {
	registry    *ParserRegistry
	gatherer    *SymbolContextGatherer
	projectRoot string
}

func NewSymbolContextTool(projectRoot string, registry *ParserRegistry) *SymbolContextTool {
	return &SymbolContextTool{
		registry:    registry,
		gatherer:    NewSymbolContextGatherer(registry),
		projectRoot: projectRoot,
	}
}

func (t *SymbolContextTool) Name() string {
	return "symbol_context"
}

func (t *SymbolContextTool) Description() string {
	return "Analyze code changes to find symbols and gather related context"
}

func (t *SymbolContextTool) Execute(args map[string]any) (string, error) {
	fileContents, hasContents := args["file_contents"].(map[string]string)
	if !hasContents {
		return "", fmt.Errorf("file_contents parameter required for symbol analysis")
	}

	diff, hasDiff := args["diff"].(string)
	if !hasDiff {
		return "", fmt.Errorf("diff parameter required for symbol analysis")
	}

	primaryLanguage, hasLang := args["primary_language"].(string)
	if !hasLang {
		primaryLanguage = ""
	}

	allSymbols, err := t.registry.ParseChangedFiles(fileContents)
	if err != nil {
		return "", fmt.Errorf("failed to parse changed files: %w", err)
	}

	diffContext := utils.GetDiffContext(diff)

	// Normalize diff context paths to absolute paths to match fileContents keys
	normalizedDiffContext := make(map[string][]utils.LineRange)
	for filePath, ranges := range diffContext {
		absPath := t.normalizeFilePath(filePath)
		normalizedDiffContext[absPath] = ranges
	}

	affectedSymbols := FilterAffectedSymbols(allSymbols, normalizedDiffContext)

	symbolContext, err := t.gatherer.GatherSymbolContext(affectedSymbols, t.projectRoot, primaryLanguage)
	if err != nil {
		return "", fmt.Errorf("failed to gather symbol context: %w", err)
	}

	if symbolContext == "" {
		if primaryLanguage == "" {
			return "Configuration-only changes detected. No symbol analysis performed.", nil
		}
		return "No additional context found for affected symbols.", nil
	}

	return symbolContext, nil
}

func FilterAffectedSymbols(symbols []types.Symbol, diffContext map[string][]utils.LineRange) []types.Symbol {
	var affected []types.Symbol

	seen := make(map[string]bool)

	for _, s := range symbols {
		key := s.FilePath + ":" + s.Name

		if seen[key] {
			continue
		}

		ranges, ok := diffContext[s.FilePath]
		if !ok {
			continue
		}

		for _, r := range ranges {
			diffStart := r.Start
			diffEnd := r.Start + r.Count - 1

			if s.EndLine >= diffStart && s.StartLine <= diffEnd {
				affected = append(affected, s)
				seen[key] = true
				break
			}
		}
	}

	return affected
}

func (t *SymbolContextTool) normalizeFilePath(filePath string) string {
	if filepath.IsAbs(filePath) {
		return filepath.Clean(filePath)
	}
	return filepath.Clean(filepath.Join(t.projectRoot, filePath))
}
