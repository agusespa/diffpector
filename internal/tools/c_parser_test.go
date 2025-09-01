package tools

import (
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

func TestCParser_ParseFile(t *testing.T) {
	parser, err := NewCParser()
	if err != nil {
		t.Fatalf("Failed to create C parser: %v", err)
	}

	tests := []struct {
		name     string
		content  string
		expected []types.Symbol
	}{
		{
			name: "simple function",
			content: `#include <stdio.h>

int add(int a, int b) {
    return a + b;
}

void print_hello() {
    printf("Hello, World!\n");
}`,
			expected: []types.Symbol{
				{Name: "add", Package: "test.c", StartLine: 3, EndLine: 5},
				{Name: "print_hello", Package: "test.c", StartLine: 7, EndLine: 9},
			},
		},
		{
			name: "struct and typedef",
			content: `typedef struct {
    int x;
    int y;
} Point;

struct Rectangle {
    Point top_left;
    Point bottom_right;
};

typedef int MyInt;`,
			expected: []types.Symbol{
				{Name: "Point", Package: "test.c", StartLine: 1, EndLine: 4},
				{Name: "Rectangle", Package: "test.c", StartLine: 6, EndLine: 9},
				{Name: "MyInt", Package: "test.c", StartLine: 11, EndLine: 11},
			},
		},
		{
			name: "function declarations",
			content: `int calculate(int x, int y);
void process_data(char* data);

int calculate(int x, int y) {
    return x * y;
}`,
			expected: []types.Symbol{
				{Name: "calculate", Package: "test.c", StartLine: 1, EndLine: 1},
				{Name: "process_data", Package: "test.c", StartLine: 2, EndLine: 2},
				{Name: "calculate", Package: "test.c", StartLine: 4, EndLine: 6},
			},
		},
		{
			name: "macros and defines",
			content: `#define MAX_SIZE 100
#define MIN(a, b) ((a) < (b) ? (a) : (b))

int buffer[MAX_SIZE];`,
			expected: []types.Symbol{
				{Name: "MAX_SIZE", Package: "test.c", StartLine: 1, EndLine: 1},
				{Name: "MIN", Package: "test.c", StartLine: 2, EndLine: 2},
				{Name: "buffer", Package: "test.c", StartLine: 4, EndLine: 4},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbols, err := parser.ParseFile("test.c", tt.content)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			if len(symbols) != len(tt.expected) {
				t.Errorf("Expected %d symbols, got %d", len(tt.expected), len(symbols))
				for i, s := range symbols {
					t.Logf("Symbol %d: %+v", i, s)
				}
				return
			}

			for i, expected := range tt.expected {
				if i >= len(symbols) {
					t.Errorf("Missing symbol %d: %+v", i, expected)
					continue
				}
				
				symbol := symbols[i]
				if symbol.Name != expected.Name {
					t.Errorf("Symbol %d name: expected %s, got %s", i, expected.Name, symbol.Name)
				}
				if symbol.StartLine != expected.StartLine {
					t.Errorf("Symbol %d start line: expected %d, got %d", i, expected.StartLine, symbol.StartLine)
				}
				if symbol.EndLine != expected.EndLine {
					t.Errorf("Symbol %d end line: expected %d, got %d", i, expected.EndLine, symbol.EndLine)
				}
			}
		})
	}
}

func TestCParser_FindSymbolUsages(t *testing.T) {
	parser, err := NewCParser()
	if err != nil {
		t.Fatalf("Failed to create C parser: %v", err)
	}

	content := `#include <stdio.h>

int add(int a, int b) {
    return a + b;
}

int main() {
    int result = add(5, 3);
    printf("Result: %d\n", result);
    return 0;
}`

	usages, err := parser.FindSymbolUsages("test.c", content, "add")
	if err != nil {
		t.Fatalf("FindSymbolUsages failed: %v", err)
	}

	if len(usages) == 0 {
		t.Error("Expected to find usages of 'add', but found none")
		return
	}

	// Should find the usage in main function
	found := false
	for _, usage := range usages {
		if usage.LineNumber == 8 && usage.SymbolName == "add" {
			found = true
			if usage.Context == "" {
				t.Error("Expected context to be non-empty")
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find usage of 'add' on line 8")
		for i, usage := range usages {
			t.Logf("Usage %d: %+v", i, usage)
		}
	}
}

func TestCParser_SupportedExtensions(t *testing.T) {
	parser, err := NewCParser()
	if err != nil {
		t.Fatalf("Failed to create C parser: %v", err)
	}

	extensions := parser.SupportedExtensions()
	expected := []string{".c", ".h"}

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

func TestCParser_ShouldExcludeFile(t *testing.T) {
	parser, err := NewCParser()
	if err != nil {
		t.Fatalf("Failed to create C parser: %v", err)
	}

	tests := []struct {
		filePath string
		expected bool
	}{
		{"src/main.c", false},
		{"include/header.h", false},
		{"test/test_main.c", true},
		{"src/utils_test.c", true},
		{"build/main.o", true},
		{"dist/lib.so", true},
		{".git/config", true},
		{"CMakeFiles/main.c", true},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := parser.ShouldExcludeFile(tt.filePath, "/project")
			if result != tt.expected {
				t.Errorf("ShouldExcludeFile(%s) = %v, expected %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestCParser_Language(t *testing.T) {
	parser, err := NewCParser()
	if err != nil {
		t.Fatalf("Failed to create C parser: %v", err)
	}

	if parser.Language() != "C" {
		t.Errorf("Expected language 'C', got '%s'", parser.Language())
	}
}