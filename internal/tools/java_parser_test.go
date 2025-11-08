package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJavaParser(t *testing.T) {
	parser, err := NewJavaParser()
	require.NoError(t, err)
	assert.NotNil(t, parser)
	assert.Equal(t, "Java", parser.Language())
	assert.Equal(t, []string{".java"}, parser.SupportedExtensions())
}

func TestJavaParser_ShouldExcludeFile(t *testing.T) {
	parser, err := NewJavaParser()
	require.NoError(t, err)

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"regular file", "src/main/java/com/example/Main.java", false},
		{"test directory", "src/test/java/com/example/MainTest.java", true},
		{"target directory", "target/classes/Main.java", true},
		{"build directory", "build/Main.java", true},
		{"git directory", ".git/Main.java", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ShouldExcludeFile(tt.filePath, "/project")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJavaParser_ParseFile(t *testing.T) {
	parser, err := NewJavaParser()
	require.NoError(t, err)

	javaCode := []byte(`package com.example;

import java.util.List;

public class Calculator {
    private int value;
    
    public Calculator() {
        this.value = 0;
    }
    
    public int add(int a, int b) {
        return a + b;
    }
    
    public void setValue(int newValue) {
        this.value = newValue;
    }
}
`)

	symbols, err := parser.ParseFile("Calculator.java", javaCode)
	require.NoError(t, err)
	assert.NotEmpty(t, symbols)

	// Check that we found some key declarations
	var foundClass, foundMethod, foundField bool
	for _, sym := range symbols {
		if sym.Name == "Calculator" && sym.Type == "class_decl" {
			foundClass = true
			assert.Equal(t, "com.example", sym.Package)
		}
		if sym.Name == "add" && sym.Type == "method_decl" {
			foundMethod = true
		}
		if sym.Name == "value" && sym.Type == "field_decl" {
			foundField = true
		}
	}

	assert.True(t, foundClass, "Should find Calculator class")
	assert.True(t, foundMethod, "Should find add method")
	assert.True(t, foundField, "Should find value field")
}
