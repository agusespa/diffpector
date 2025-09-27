package tools

import (
	"fmt"
	"os"

	"github.com/agusespa/diffpector/internal/types"
	"github.com/agusespa/diffpector/internal/utils"
)

type SymbolContextTool struct {
	parserRegistry *ParserRegistry
	gatherer       *SymbolContextGatherer
	projectRoot    string
}

func NewSymbolContextTool(projectRoot string, registry *ParserRegistry) *SymbolContextTool {
	return &SymbolContextTool{
		parserRegistry: registry,
		gatherer:       NewSymbolContextGatherer(registry),
		projectRoot:    projectRoot,
	}
}

func (t *SymbolContextTool) Name() string {
	return "symbol_context"
}

func (t *SymbolContextTool) Description() string {
	return "Analyze code changes to find symbols and gather related context"
}

func (t *SymbolContextTool) Execute(args map[string]any) (types.DiffData, error) {
	diffData, hasDiff := args["diffData"].(types.DiffData)
	if !hasDiff {
		return types.DiffData{}, fmt.Errorf("diffData parameter required for symbol analysis")
	}

	primaryLanguage, hasLang := args["primaryLanguage"].(string)
	if !hasLang {
		return types.DiffData{}, fmt.Errorf("primaryLanguage parameter required for symbol analysis")
	}

	content, err := os.ReadFile(diffData.AbsolutePath)
	if err != nil {
		return types.DiffData{}, fmt.Errorf("failed to read changed file: %w", err)
	}

	allSymbols, err := t.parserRegistry.ParseFile(diffData.AbsolutePath, content)
	if err != nil {
		return types.DiffData{}, fmt.Errorf("failed to parse changed files: %w", err)
	}

	diffContext, err := utils.GetDiffContext(diffData, allSymbols, content)
	if err != nil {
		return types.DiffData{}, fmt.Errorf("failed extract diff context: %w", err)
	}
	diffData.DiffContext = diffContext.Context
	diffData.AffectedSymbols = diffContext.AffectedSymbols

	err = t.gatherer.GatherSymbolContext(diffData.AffectedSymbols, t.projectRoot, primaryLanguage)
	if err != nil {
		return types.DiffData{}, fmt.Errorf("failed to gather symbol usage context: %w", err)
	}

	return diffData, nil
}
