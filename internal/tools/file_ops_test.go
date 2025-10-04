package tools

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestWriteFileTool_Name(t *testing.T) {
	tool := &WriteFileTool{}
	if tool.Name() != "write_file" {
		t.Errorf("Expected name 'write_file', got %s", tool.Name())
	}
}

func TestWriteFileTool_Description(t *testing.T) {
	tool := &WriteFileTool{}
	desc := tool.Description()
	if !strings.Contains(strings.ToLower(desc), "write") {
		t.Errorf("Expected description to contain 'write', got: %s", desc)
	}
}

func TestWriteFileTool_Execute(t *testing.T) {
	tool := &WriteFileTool{}

	tmpfile, err := os.CreateTemp("", "test_write_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := "hello world"
	args := map[string]any{
		"filename": tmpfile.Name(),
		"content":  content,
	}

	result, err := tool.Execute(args)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("Expected result to be a string, but got %T", result)
	}

	if !strings.Contains(resultStr, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", resultStr)
	}

	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read back file: %v", err)
	}
	if string(data) != content {
		t.Errorf("Expected file content '%s', got '%s'", content, string(data))
	}
}

func TestReadFileTool_Name(t *testing.T) {
	tool := &ReadFileTool{}
	if tool.Name() != "read_file" {
		t.Errorf("Expected name 'read_file', got %s", tool.Name())
	}
}

func TestReadFileTool_Description(t *testing.T) {
	tool := &ReadFileTool{}
	desc := tool.Description()
	if !strings.Contains(strings.ToLower(desc), "read") {
		t.Errorf("Expected description to contain 'read', got: %s", desc)
	}
}

func TestReadFileTool_Execute(t *testing.T) {
	tool := &ReadFileTool{}

	tmpfile, err := os.CreateTemp("", "test_read_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if rErr := os.Remove(tmpfile.Name()); rErr != nil {
			fmt.Printf("Error removing temporary file %s: %v", tmpfile.Name(), rErr)
		}
	}()

	content := "hello from read test"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file %s: %v", tmpfile.Name(), err)
	}

	args := map[string]any{"filename": tmpfile.Name()}
	result, err := tool.Execute(args)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("Expected result to be a string, but got %T", result)
	}

	if !strings.Contains(resultStr, content) {
		t.Errorf("Expected result to contain file content, got: %s", resultStr)
	}
}
