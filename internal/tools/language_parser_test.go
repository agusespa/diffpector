package tools

import (
	"testing"
)

func TestParserRegistry_RegisterAndGetParser(t *testing.T) {
	registry := NewParserRegistry()
	
	// Test that Go parser is registered by default
	goParser := registry.GetParser("test.go")
	if goParser == nil {
		t.Error("Expected Go parser to be registered by default")
	}
	
	if goParser.Language() != "Go" {
		t.Errorf("Expected Go parser language to be 'Go', got '%s'", goParser.Language())
	}
	
	// Test unsupported file type
	unknownParser := registry.GetParser("test.unknown")
	if unknownParser != nil {
		t.Error("Expected nil parser for unsupported file type")
	}
}

func TestParserRegistry_GetSupportedLanguages(t *testing.T) {
	registry := NewParserRegistry()
	
	languages := registry.GetSupportedLanguages()
	
	if len(languages) == 0 {
		t.Error("Expected at least one supported language")
	}
	
	// Should include Go by default
	found := false
	for _, lang := range languages {
		if lang == "Go" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected 'Go' to be in supported languages")
	}
}

func TestGoParser_SupportedExtensions(t *testing.T) {
	parser := NewGoParser()
	
	extensions := parser.SupportedExtensions()
	
	if len(extensions) != 1 || extensions[0] != ".go" {
		t.Errorf("Expected Go parser to support only '.go' extension, got %v", extensions)
	}
}

func TestGoParser_ParseFile(t *testing.T) {
	parser := NewGoParser()
	
	content := `package main

func TestFunction() {
	// test
}

type TestType struct {
	Field string
}

var testVar string
const testConst = "value"
`
	
	symbols := parser.ParseFile("test.go", content)
	
	expectedSymbols := []string{"TestFunction", "TestType", "testVar", "testConst"}
	
	if len(symbols) < len(expectedSymbols) {
		t.Errorf("Expected at least %d symbols, got %d", len(expectedSymbols), len(symbols))
	}
	
	foundNames := make(map[string]bool)
	for _, symbol := range symbols {
		foundNames[symbol.Name] = true
	}
	
	for _, expected := range expectedSymbols {
		if !foundNames[expected] {
			t.Errorf("Expected to find symbol %s, but it was not found", expected)
		}
	}
}