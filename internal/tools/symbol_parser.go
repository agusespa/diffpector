package tools

import (
	"fmt"
)

type Symbol struct {
	Name      string
	Package   string
	FilePath  string
	StartLine int
	EndLine   int
}

type SymbolParser struct {
	parserRegistry *ParserRegistry
}

func NewSymbolParser(registry *ParserRegistry) *SymbolParser {
	return &SymbolParser{
		parserRegistry: registry,
	}
}

func (sp *SymbolParser) ParseChangedFiles(fileContents map[string]string) ([]Symbol, error) {
	var allSymbols []Symbol

	for filePath, content := range fileContents {
		symbols, err := sp.ParseFile(filePath, content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", filePath, err)
		}
		allSymbols = append(allSymbols, symbols...)
	}

	return allSymbols, nil
}

func (sp *SymbolParser) ParseFile(filePath, content string) ([]Symbol, error) {
	return sp.parserRegistry.ParseFile(filePath, content)
}
