package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
)

type LanguageParser interface {
	// ParseFile extracts all symbols (functions, types, variables, etc.) from a file
	ParseFile(filePath, content string) ([]types.Symbol, error)

	// FindSymbolUsages finds all usages of a specific symbol within a file
	FindSymbolUsages(filePath, content, symbolName string) ([]types.SymbolUsage, error)

	// GetSymbolContext extracts contextual information around a symbol definition
	GetSymbolContext(filePath, content string, symbol types.Symbol) (string, error)

	// SupportedExtensions returns the file extensions this parser can handle
	SupportedExtensions() []string

	// Language returns the human-readable name of the language this parser handles
	Language() string

	// ShouldExcludeFile determines if a file should be excluded from symbol context gathering
	// This allows language-specific filtering (e.g., Go excludes *_test.go, JS excludes node_modules)
	ShouldExcludeFile(filePath, projectRoot string) bool
}

type ParserRegistry struct {
	parsers map[string]LanguageParser
}

func NewParserRegistry() *ParserRegistry {
	registry := &ParserRegistry{
		parsers: make(map[string]LanguageParser),
	}

	goParser, err := NewGoParser()
	if err != nil {
		panic(fmt.Errorf("failed to create Go parser: %w", err))
	}
	registry.RegisterParser(goParser)

	tsParser, err := NewTypeScriptParser()
	if err != nil {
		panic(fmt.Errorf("failed to create TypeScript parser: %w", err))
	}
	registry.RegisterParser(tsParser)

	return registry
}

func (pr *ParserRegistry) RegisterParser(parser LanguageParser) {
	for _, ext := range parser.SupportedExtensions() {
		pr.parsers[ext] = parser
	}
}

func (pr *ParserRegistry) GetParser(filePath string) LanguageParser {
	ext := strings.ToLower(filepath.Ext(filePath))
	return pr.parsers[ext]
}

func (pr *ParserRegistry) ParseFile(filePath, content string) ([]types.Symbol, error) {
	parser := pr.GetParser(filePath)
	if parser == nil {
		return []types.Symbol{}, nil
	}
	return parser.ParseFile(filePath, content)
}

func (pr *ParserRegistry) ParseChangedFiles(fileContents map[string]string) ([]types.Symbol, error) {
	var allSymbols []types.Symbol

	for filePath, content := range fileContents {
		symbols, err := pr.ParseFile(filePath, content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", filePath, err)
		}
		allSymbols = append(allSymbols, symbols...)
	}

	return allSymbols, nil
}

func (pr *ParserRegistry) IsKnownLanguage(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".go", ".java", ".js", ".ts", ".tsx", ".py", ".rb", ".php",
		".cs", ".cpp", ".cc", ".cxx", ".c", ".rs", ".kt",
		".scala", ".swift":
		return true
	default:
		return false
	}
}
