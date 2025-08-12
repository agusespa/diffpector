package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

type LanguageParser interface {
	ParseFile(filePath, content string) ([]Symbol, error)
	SupportedExtensions() []string
	Language() string
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

	// javaParser := NewJavaParser()
	// registry.RegisterParser(javaParser)

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

func (pr *ParserRegistry) ParseFile(filePath, content string) ([]Symbol, error) {
	parser := pr.GetParser(filePath)
	if parser == nil {
		return []Symbol{}, nil
	}
	return parser.ParseFile(filePath, content)
}

func (pr *ParserRegistry) IsKnownLanguage(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".go", ".java", ".js", ".ts", ".py", ".rb", ".php",
		".cs", ".cpp", ".cc", ".cxx", ".c", ".rs", ".kt",
		".scala", ".swift":
		return true
	default:
		return false
	}
}
