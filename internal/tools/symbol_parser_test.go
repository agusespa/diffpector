package tools

import (
	"testing"
)

func TestParserRegistry_IsKnownLanguage(t *testing.T) {
	registry := NewParserRegistry()
	
	testCases := []struct {
		filePath string
		expected bool
	}{
		// Programming languages with parsers
		{"main.go", true},
		{"script.py", true},
		{"component.ts", true},
		{"Main.java", true},
		
		// Non-programming language files
		{"index.html", false},
		{"page.htm", false},
		{"styles.css", false},
		{"main.scss", false},
		{"theme.sass", false},
		{"config.less", false},
		{"config.txt", false},
		{"README.md", false},
		{"Dockerfile", false},
		{"package.json", false},
	}
	
	for _, tc := range testCases {
		result := registry.IsKnownLanguage(tc.filePath)
		if result != tc.expected {
			t.Errorf("IsKnownLanguage(%s) = %v, expected %v", tc.filePath, result, tc.expected)
		}
	}
}

func TestParserRegistry_GetParser(t *testing.T) {
	registry := NewParserRegistry()
	
	// Programming languages should have parsers
	programmingFiles := []string{"main.go", "script.py", "component.ts", "Main.java"}
	for _, file := range programmingFiles {
		parser := registry.GetParser(file)
		if parser == nil {
			t.Errorf("Expected parser for %s, got nil", file)
		}
	}
	
	// Web files should NOT have parsers
	webFiles := []string{"index.html", "styles.css", "main.scss"}
	for _, file := range webFiles {
		parser := registry.GetParser(file)
		if parser != nil {
			t.Errorf("Expected no parser for %s, got %v", file, parser.Language())
		}
	}
	
	// Unknown files should not have parsers
	unknownFiles := []string{"config.txt", "README.md", "Dockerfile"}
	for _, file := range unknownFiles {
		parser := registry.GetParser(file)
		if parser != nil {
			t.Errorf("Expected no parser for %s, got %v", file, parser.Language())
		}
	}
}