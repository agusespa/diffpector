package tools

import (
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

func TestPythonParser_ParseFile(t *testing.T) {
	parser, err := NewPythonParser()
	if err != nil {
		t.Fatalf("Failed to create Python parser: %v", err)
	}

	testCases := []struct {
		name     string
		content  string
		expected []types.Symbol
	}{
		{
			name: "function definitions",
			content: `def hello_world():
    print("Hello, World!")

def calculate_sum(a, b):
    return a + b

async def fetch_data():
    pass`,
			expected: []types.Symbol{
				{Name: "hello_world", Package: "test.py", StartLine: 1, EndLine: 2},
				{Name: "calculate_sum", Package: "test.py", StartLine: 4, EndLine: 5},
				{Name: "fetch_data", Package: "test.py", StartLine: 7, EndLine: 8},
			},
		},
		{
			name: "class definitions",
			content: `class Person:
    def __init__(self, name):
        self.name = name
    
    def greet(self):
        return f"Hello, {self.name}"

class Employee(Person):
    def __init__(self, name, employee_id):
        super().__init__(name)
        self.employee_id = employee_id`,
			expected: []types.Symbol{
				{Name: "Person", Package: "test.py", StartLine: 1, EndLine: 6},
				{Name: "__init__", Package: "test.py", StartLine: 2, EndLine: 3},
				{Name: "greet", Package: "test.py", StartLine: 5, EndLine: 6},
				{Name: "Employee", Package: "test.py", StartLine: 8, EndLine: 11},
				{Name: "__init__", Package: "test.py", StartLine: 9, EndLine: 11},
			},
		},
		{
			name: "variable assignments",
			content: `API_KEY = "secret"
DEBUG = True
config = {"host": "localhost", "port": 8080}
x, y = 10, 20`,
			expected: []types.Symbol{
				{Name: "API_KEY", Package: "test.py", StartLine: 1, EndLine: 1},
				{Name: "DEBUG", Package: "test.py", StartLine: 2, EndLine: 2},
				{Name: "config", Package: "test.py", StartLine: 3, EndLine: 3},
				{Name: "x", Package: "test.py", StartLine: 4, EndLine: 4},
				{Name: "y", Package: "test.py", StartLine: 4, EndLine: 4},
			},
		},
		{
			name: "import statements",
			content: `import os
import json
from typing import List, Dict
from datetime import datetime as dt
import requests as req`,
			expected: []types.Symbol{
				// Imports are complex to parse with tree-sitter, focusing on core symbols for now
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			symbols, err := parser.ParseFile("test.py", tc.content)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			if len(symbols) != len(tc.expected) {
				t.Errorf("Expected %d symbols, got %d", len(tc.expected), len(symbols))
				for i, s := range symbols {
					t.Logf("Symbol %d: %+v", i, s)
				}
				return
			}

			for i, expected := range tc.expected {
				if i >= len(symbols) {
					t.Errorf("Missing symbol %d: %+v", i, expected)
					continue
				}
				
				actual := symbols[i]
				if actual.Name != expected.Name {
					t.Errorf("Symbol %d name: expected %s, got %s", i, expected.Name, actual.Name)
				}
				if actual.StartLine != expected.StartLine {
					t.Errorf("Symbol %d start line: expected %d, got %d", i, expected.StartLine, actual.StartLine)
				}
			}
		})
	}
}

func TestPythonParser_FindSymbolUsages(t *testing.T) {
	parser, err := NewPythonParser()
	if err != nil {
		t.Fatalf("Failed to create Python parser: %v", err)
	}

	content := `def calculate_total(items):
    total = 0
    for item in items:
        total += item.price
    return total

def process_order(order):
    items = order.items
    total = calculate_total(items)
    return {"total": total, "items": len(items)}`

	usages, err := parser.FindSymbolUsages("test.py", content, "calculate_total")
	if err != nil {
		t.Fatalf("FindSymbolUsages failed: %v", err)
	}

	if len(usages) != 1 {
		t.Errorf("Expected 1 usage, got %d", len(usages))
		for i, usage := range usages {
			t.Logf("Usage %d: %+v", i, usage)
		}
		return
	}

	usage := usages[0]
	if usage.LineNumber != 9 {
		t.Errorf("Expected usage on line 9, got line %d", usage.LineNumber)
	}
	if usage.SymbolName != "calculate_total" {
		t.Errorf("Expected symbol name 'calculate_total', got '%s'", usage.SymbolName)
	}
}

func TestPythonParser_SupportedExtensions(t *testing.T) {
	parser, err := NewPythonParser()
	if err != nil {
		t.Fatalf("Failed to create Python parser: %v", err)
	}

	extensions := parser.SupportedExtensions()
	expected := []string{".py", ".pyw"}

	if len(extensions) != len(expected) {
		t.Errorf("Expected %d extensions, got %d", len(expected), len(extensions))
		return
	}

	for i, ext := range expected {
		if extensions[i] != ext {
			t.Errorf("Expected extension %s, got %s", ext, extensions[i])
		}
	}
}

func TestPythonParser_ShouldExcludeFile(t *testing.T) {
	parser, err := NewPythonParser()
	if err != nil {
		t.Fatalf("Failed to create Python parser: %v", err)
	}

	testCases := []struct {
		filePath string
		expected bool
	}{
		{"src/main.py", false},
		{"test_main.py", true},
		{"src/test_utils.py", true},
		{"tests/test_integration.py", true},
		{"__pycache__/main.cpython-39.pyc", true},
		{"venv/lib/python3.9/site-packages/requests.py", true},
		{"src/__init__.py", true},
		{"utils/helper.py", false},
		{"migrations/0001_initial.py", true},
	}

	for _, tc := range testCases {
		t.Run(tc.filePath, func(t *testing.T) {
			result := parser.ShouldExcludeFile(tc.filePath, "/project")
			if result != tc.expected {
				t.Errorf("ShouldExcludeFile(%s): expected %v, got %v", tc.filePath, tc.expected, result)
			}
		})
	}
}

func TestPythonParser_Language(t *testing.T) {
	parser, err := NewPythonParser()
	if err != nil {
		t.Fatalf("Failed to create Python parser: %v", err)
	}

	if parser.Language() != "Python" {
		t.Errorf("Expected language 'Python', got '%s'", parser.Language())
	}
}