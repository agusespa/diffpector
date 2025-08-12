package tools

import (
	"testing"
)

const sampleCode = `
package sample

import (
    "fmt"
    "strings"
)

const Pi = 3.14
const (
    E = 2.71
    Version = "1.0"
)

var GlobalVar = 42

type Person struct {
    Name string
    Age  int
}

type Greeter interface {
    Greet() string
    Say(msg string) error
}

func (p *Person) Greet() string {
    return "Hello"
}

func Add(a, b int) int {
    return a + b
}

var (
    MaxValue = 100
    MinValue = 0
)
`

func TestGoParser_ParseFile(t *testing.T) {
	parser, _ := NewGoParser()

	symbols, err := parser.ParseFile("sample.go", sampleCode)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	for _, sym := range symbols {
		t.Logf("Captured symbol: %q from line %d to %d", sym.Name, sym.StartLine, sym.EndLine)
	}

	expectedSymbols := []string{
		"\"fmt\"", "\"strings\"", "Pi", "E", "Version", "GlobalVar", 
		"Person", "Name", "Age", "Greeter", "Greet", "Say", "Add", 
		"MaxValue", "MinValue",
	}

	// Check that we have the expected symbols
	symbolNames := make(map[string]int)
	for _, sym := range symbols {
		symbolNames[sym.Name]++
	}

	for _, expected := range expectedSymbols {
		if symbolNames[expected] == 0 {
			t.Errorf("Expected symbol %q not found", expected)
		}
	}

	// Verify we have exactly 16 symbols (including duplicate "Greet")
	expectedCount := 16
	if len(symbols) != expectedCount {
		t.Errorf("Expected %d symbols, got %d", expectedCount, len(symbols))
	}

	// Verify "Greet" appears twice (interface method + struct method)
	if symbolNames["Greet"] != 2 {
		t.Errorf("Expected 'Greet' to appear twice, got %d times", symbolNames["Greet"])
	}
}
