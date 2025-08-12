package tools

import (
	"strings"
	"testing"
)

func TestGitDiffTool_Name(t *testing.T) {
	tool := &GitDiffTool{}
	if tool.Name() != "git_diff" {
		t.Errorf("Expected name 'git_diff', got %s", tool.Name())
	}
}

func TestGitDiffTool_Description(t *testing.T) {
	tool := &GitDiffTool{}
	desc := tool.Description()
	if !strings.Contains(strings.ToLower(desc), "diff") {
		t.Errorf("Expected description to contain 'diff', got: %s", desc)
	}
}

func TestGitStagedFilesTool_Name(t *testing.T) {
	tool := &GitStagedFilesTool{}
	if tool.Name() != "git_staged_files" {
		t.Errorf("Expected name 'git_staged_files', got %s", tool.Name())
	}
}

func TestGitStagedFilesTool_Description(t *testing.T) {
	tool := &GitStagedFilesTool{}
	desc := tool.Description()
	if !strings.Contains(strings.ToLower(desc), "staged files") {
		t.Errorf("Expected description to contain 'staged files', got: %s", desc)
	}
}

func TestGitDiffTool_Execute(t *testing.T) {
	tool := &GitDiffTool{}
	// This test should succeed even with no staged changes (returns empty string)
	result, err := tool.Execute(map[string]any{})
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	// Empty result is valid when no files are staged
	_ = result
}

func TestGitStagedFilesTool_Execute(t *testing.T) {
	tool := &GitStagedFilesTool{}
	// This test expects no staged files in a clean test environment.
	// The tool should return an empty string (no error) when no files are staged.
	result, err := tool.Execute(map[string]any{})
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	// Empty result is valid when no files are staged
	_ = result
}

func TestGitGrepTool_Name(t *testing.T) {
	tool := &GitGrepTool{}
	if tool.Name() != "git_grep" {
		t.Errorf("Expected name 'git_grep', got %s", tool.Name())
	}
}

func TestGitGrepTool_Description(t *testing.T) {
	tool := &GitGrepTool{}
	desc := tool.Description()
	if !strings.Contains(strings.ToLower(desc), "search") {
		t.Errorf("Expected description to contain 'search', got: %s", desc)
	}
}

func TestGitGrepTool_Execute_MissingPattern(t *testing.T) {
	tool := &GitGrepTool{}
	_, err := tool.Execute(map[string]any{})

	if err == nil {
		t.Error("Expected error for missing pattern")
	}
	if !strings.Contains(err.Error(), "pattern parameter required") {
		t.Errorf("Expected pattern error, got: %v", err)
	}
}

func TestGitGrepTool_Execute_WithPattern(t *testing.T) {
	tool := &GitGrepTool{}

	// This test assumes it's run within a Go project where "package" is common.
	result, err := tool.Execute(map[string]any{"pattern": "package"})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !strings.Contains(result, "Search results for 'package'") && !strings.Contains(result, "No matches found for pattern: package") {
		t.Errorf("Expected search results or no matches format, got: %s", result)
	}
}
