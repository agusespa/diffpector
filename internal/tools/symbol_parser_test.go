package tools

import (
	"testing"
)

func TestSymbolParser_ParseGoFile(t *testing.T) {
	parser := NewSymbolParser()
	
	content := `package main

import "fmt"

type User struct {
	Name string
	Age  int
}

const MaxUsers = 100

var globalVar string

func NewUser(name string) *User {
	return &User{Name: name}
}

func (u *User) GetName() string {
	return u.Name
}

func main() {
	user := NewUser("John")
	fmt.Println(user.GetName())
}
`
	
	symbols := parser.ParseFile("test.go", content)
	
	// Verify we found the expected symbols
	expectedSymbols := map[string]string{
		"User":     "type",
		"MaxUsers": "constant",
		"NewUser":  "function",
		"GetName":  "method",
		"main":     "function",
	}
	
	if len(symbols) < len(expectedSymbols) {
		t.Errorf("Expected at least %d symbols, got %d", len(expectedSymbols), len(symbols))
	}
	
	foundSymbols := make(map[string]string)
	for _, symbol := range symbols {
		foundSymbols[symbol.Name] = symbol.Type
	}
	
	for name, expectedType := range expectedSymbols {
		if actualType, found := foundSymbols[name]; !found {
			t.Errorf("Expected to find symbol %s, but it was not found", name)
		} else if actualType != expectedType {
			t.Errorf("Expected symbol %s to be of type %s, got %s", name, expectedType, actualType)
		}
	}
}

func TestSymbolParser_ParseChangedFiles(t *testing.T) {
	parser := NewSymbolParser()
	
	fileContents := map[string]string{
		"test.go": `package main

func NewFunction() {
	// New function added
}

func main() {
	fmt.Println("Hello")
}`,
	}
	
	symbols := parser.ParseChangedFiles(fileContents)
	
	// Should find both functions
	foundNew := false
	foundMain := false
	for _, symbol := range symbols {
		if symbol.Name == "NewFunction" && symbol.Type == "function" {
			foundNew = true
		}
		if symbol.Name == "main" && symbol.Type == "function" {
			foundMain = true
		}
	}
	
	if !foundNew {
		t.Error("Expected to find NewFunction, but it was not found")
	}
	if !foundMain {
		t.Error("Expected to find main function, but it was not found")
	}
}

func TestSymbolParser_ParseNonGoFile(t *testing.T) {
	parser := NewSymbolParser()
	
	// Test with a non-Go file - should return empty symbols
	content := `function testFunction() {
	console.log("test");
}
`
	
	symbols := parser.ParseFile("test.js", content)
	
	// Should return empty symbols for unsupported file types
	if len(symbols) != 0 {
		t.Errorf("Expected 0 symbols for unsupported file type, got %d", len(symbols))
	}
}