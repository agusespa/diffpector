package tools

import (
	"testing"
)

func TestContextGatherer_GenerateSearchPatterns(t *testing.T) {
	gatherer := NewContextGatherer()
	
	tests := []struct {
		symbol   Symbol
		expected []string
	}{
		{
			symbol: Symbol{Name: "TestFunc", Type: "function"},
			expected: []string{
				`\bTestFunc\b`,
				`TestFunc\s*\(`,
			},
		},
		{
			symbol: Symbol{Name: "GetName", Type: "method"},
			expected: []string{
				`\bGetName\b`,
				`\.GetName\s*\(`,
			},
		},
		{
			symbol: Symbol{Name: "User", Type: "type"},
			expected: []string{
				`\bUser\b`,
				`User\{`,
				`\*User`,
				`\[\]User`,
				`User\)`,
			},
		},
	}
	
	for _, test := range tests {
		patterns := gatherer.generateSearchPatterns(test.symbol)
		
		if len(patterns) < len(test.expected) {
			t.Errorf("For symbol %s, expected at least %d patterns, got %d", 
				test.symbol.Name, len(test.expected), len(patterns))
		}
		
		// Check that all expected patterns are present
		patternMap := make(map[string]bool)
		for _, pattern := range patterns {
			patternMap[pattern] = true
		}
		
		for _, expected := range test.expected {
			if !patternMap[expected] {
				t.Errorf("For symbol %s, expected pattern %s was not found", 
					test.symbol.Name, expected)
			}
		}
	}
}

func TestContextGatherer_ParseGrepLine(t *testing.T) {
	gatherer := NewContextGatherer()
	
	tests := []struct {
		line     string
		expected *Usage
	}{
		{
			line: "main.go:15:func TestFunction() {",
			expected: &Usage{
				FilePath: "main.go",
				Line:     15,
				Context:  "func TestFunction() {",
			},
		},
		{
			line: "internal/agent/agent.go:42:	user := NewUser(\"test\")",
			expected: &Usage{
				FilePath: "internal/agent/agent.go",
				Line:     42,
				Context:  "user := NewUser(\"test\")",
			},
		},
		{
			line: "invalid line format",
			expected: nil,
		},
	}
	
	for _, test := range tests {
		result := gatherer.parseGrepLine(test.line)
		
		if test.expected == nil {
			if result != nil {
				t.Errorf("Expected nil for line %s, got %+v", test.line, result)
			}
			continue
		}
		
		if result == nil {
			t.Errorf("Expected usage for line %s, got nil", test.line)
			continue
		}
		
		if result.FilePath != test.expected.FilePath ||
			result.Line != test.expected.Line ||
			result.Context != test.expected.Context {
			t.Errorf("For line %s, expected %+v, got %+v", 
				test.line, test.expected, result)
		}
	}
}

func TestContextGatherer_RemoveDuplicateUsages(t *testing.T) {
	gatherer := NewContextGatherer()
	
	usages := []Usage{
		{FilePath: "main.go", Line: 10, Context: "test"},
		{FilePath: "main.go", Line: 10, Context: "test"}, // duplicate
		{FilePath: "main.go", Line: 15, Context: "other"},
		{FilePath: "other.go", Line: 10, Context: "test"}, // different file, same line
	}
	
	unique := gatherer.removeDuplicateUsages(usages)
	
	if len(unique) != 3 {
		t.Errorf("Expected 3 unique usages, got %d", len(unique))
	}
	
	// Verify the duplicate was removed
	seen := make(map[string]int)
	for _, usage := range unique {
		key := usage.FilePath + ":" + string(rune(usage.Line))
		seen[key]++
	}
	
	for key, count := range seen {
		if count > 1 {
			t.Errorf("Found duplicate usage for %s", key)
		}
	}
}