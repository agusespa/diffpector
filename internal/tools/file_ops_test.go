package tools

import (
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

	// Create a temporary file for testing
	tmpfile, err := os.CreateTemp("", "test_write_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Test writing to the file
	content := "hello world"
	args := map[string]any{
		"filename": tmpfile.Name(),
		"content":  content,
	}

	result, err := tool.Execute(args)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !strings.Contains(result, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify file content
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

	// Create a temporary file with content
	tmpfile, err := os.CreateTemp("", "test_read_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := "hello from read test"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	// Test reading from the file
	args := map[string]any{"filename": tmpfile.Name()}
	result, err := tool.Execute(args)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !strings.Contains(result, content) {
		t.Errorf("Expected result to contain file content, got: %s", result)
	}
}

func TestAppendFileTool_Name(t *testing.T) {
	tool := &AppendFileTool{}
	if tool.Name() != "append_file" {
		t.Errorf("Expected name 'append_file', got %s", tool.Name())
	}
}

func TestAppendFileTool_Description(t *testing.T) {
	tool := &AppendFileTool{}
	desc := tool.Description()
	if !strings.Contains(strings.ToLower(desc), "append") {
		t.Errorf("Expected description to contain 'append', got: %s", desc)
	}
}

func TestAppendFileTool_Execute(t *testing.T) {
	tool := &AppendFileTool{}

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test_append_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// First write
	initialContent := "initial content"
	err = os.WriteFile(tmpfile.Name(), []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// Append to the file
	appendedContent := "appended content"
	args := map[string]any{
		"filename": tmpfile.Name(),
		"content":  appendedContent,
	}

	result, err := tool.Execute(args)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !strings.Contains(result, "Successfully appended") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify file content
	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read back file: %v", err)
	}

	expectedContent := initialContent + "\n" + appendedContent
	if string(data) != expectedContent {
		t.Errorf("Expected file content '%s', got '%s'", expectedContent, string(data))
	}
}
