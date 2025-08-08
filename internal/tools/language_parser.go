package tools

import (
	"path/filepath"
	"strings"
)

// LanguageParser defines the interface for language-specific parsers
type LanguageParser interface {
	// ParseFile parses a file and extracts symbols
	ParseFile(filePath, content string) []Symbol
	// SupportedExtensions returns the file extensions this parser supports
	SupportedExtensions() []string
	// Language returns the name of the language this parser handles
	Language() string
}

// ParserRegistry manages language-specific parsers
type ParserRegistry struct {
	parsers map[string]LanguageParser // extension -> parser
}

func NewParserRegistry() *ParserRegistry {
	registry := &ParserRegistry{
		parsers: make(map[string]LanguageParser),
	}
	
	// Register Go parser by default
	goParser := NewGoParser()
	registry.RegisterParser(goParser)
	
	return registry
}

// RegisterParser registers a language parser for its supported extensions
func (pr *ParserRegistry) RegisterParser(parser LanguageParser) {
	for _, ext := range parser.SupportedExtensions() {
		pr.parsers[ext] = parser
	}
}

// GetParser returns the appropriate parser for a file
func (pr *ParserRegistry) GetParser(filePath string) LanguageParser {
	ext := strings.ToLower(filepath.Ext(filePath))
	return pr.parsers[ext]
}

// ParseFile parses a file using the appropriate language parser
func (pr *ParserRegistry) ParseFile(filePath, content string) []Symbol {
	parser := pr.GetParser(filePath)
	if parser == nil {
		return []Symbol{} // No parser available for this file type
	}
	return parser.ParseFile(filePath, content)
}

// GetSupportedLanguages returns a list of supported languages
func (pr *ParserRegistry) GetSupportedLanguages() []string {
	languages := make(map[string]bool)
	var result []string
	
	for _, parser := range pr.parsers {
		if !languages[parser.Language()] {
			languages[parser.Language()] = true
			result = append(result, parser.Language())
		}
	}
	
	return result
}