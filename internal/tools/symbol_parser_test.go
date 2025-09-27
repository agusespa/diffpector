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
		{"program.c", true},
		{"header.h", true},

		// Script files (should return false - treated like config files)
		{"deploy.sh", false},
		{"setup.bash", false},
		{"config.zsh", false},
		{"install.fish", false},
		{"build.ps1", false},
		{"run.bat", false},
		{"start.cmd", false},

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
		{"config.yaml", false},
		{"settings.toml", false},
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

	// TODO extend when added parsers
	// programmingFiles := []string{"main.go", "script.py", "component.ts", "Main.java", "program.c"}
	// Programming languages should have parsers
	programmingFiles := []string{"main.go"}
	for _, file := range programmingFiles {
		parser := registry.GetParser(file)
		if parser == nil {
			t.Errorf("Expected parser for %s, got nil", file)
		}
	}

	// Script files should NOT have parsers
	scriptFiles := []string{"deploy.sh", "setup.bash", "config.zsh", "build.ps1", "run.bat"}
	for _, file := range scriptFiles {
		parser := registry.GetParser(file)
		if parser != nil {
			t.Errorf("Expected no parser for %s, got %v", file, parser.Language())
		}
	}

	// Web and config files should NOT have parsers
	otherFiles := []string{"index.html", "styles.css", "main.scss", "config.txt", "README.md", "Dockerfile", "package.json"}
	for _, file := range otherFiles {
		parser := registry.GetParser(file)
		if parser != nil {
			t.Errorf("Expected no parser for %s, got %v", file, parser.Language())
		}
	}
}
