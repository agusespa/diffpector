package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
)

type LanguageParser interface {
	// ParseFile extracts all symbols (functions, types, variables, etc.) from a file
	ParseFile(filePath string, content []byte) ([]types.Symbol, error)

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

	javaParser, err := NewJavaParser()
	if err != nil {
		panic(fmt.Errorf("failed to create Java parser: %w", err))
	}
	registry.RegisterParser(javaParser)

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

func (pr *ParserRegistry) ParseFile(filePath string, content []byte) ([]types.Symbol, error) {
	parser := pr.GetParser(filePath)
	if parser == nil {
		return []types.Symbol{}, nil
	}

	return parser.ParseFile(filePath, content)
}

func (pr *ParserRegistry) GetParser(filePath string) LanguageParser {
	ext := strings.ToLower(filepath.Ext(filePath))
	return pr.parsers[ext]
}

func (pr *ParserRegistry) IsKnownLanguage(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd":
		return false
	case ".go", ".java", ".js", ".ts", ".tsx", ".py", ".rb", ".php",
		".cs", ".cpp", ".cc", ".cxx", ".c", ".h", ".rs", ".kt",
		".scala", ".swift":
		return true
	default:
		return false
	}
}
