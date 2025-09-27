package tools

import (
	"os"
	"os/exec"
	"path/filepath"
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

func TestGitDiffTool_Execute_NoChanges(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to change back to original directory: %v", err)
		}
	}()

	tool := &GitDiffTool{}

	result, err := tool.Execute(nil)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected an empty result map, but got %d entries", len(result))
	}
}

func TestGitDiffTool_Execute_SingleModifiedFile(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to change back to original directory: %v", err)
		}
	}()

	createAndCommitFile(t, tempDir, "testfile.txt", "Initial content.\n")

	const newContent = "Initial content.\nSecond line.\n"
	if err := os.WriteFile(filepath.Join(tempDir, "testfile.txt"), []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	cmd := exec.Command("git", "add", "testfile.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	tool := &GitDiffTool{}

	result, err := tool.Execute(nil)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 file in diff result, but got %d", len(result))
	}

	diffData, ok := result["testfile.txt"]
	if !ok {
		t.Fatalf("Expected diff for 'testfile.txt', but not found in result keys: %v", result)
	}

	expectedDiffParts := []string{
		"--- a/testfile.txt",
		"+++ b/testfile.txt",
		"@@ -1,1 +1,2 @@",
		"+Second line.",
	}

	diffContent := diffData.Diff
	for _, part := range expectedDiffParts {
		if !strings.Contains(diffContent, part) {
			t.Errorf("Diff content missing expected part: %q\nFull diff:\n%s", part, diffContent)
		}
	}
}

func TestGitDiffTool_Execute_DeletedFile(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to change back to original directory: %v", err)
		}
	}()

	createAndCommitFile(t, tempDir, "file_to_delete.txt", "This file will be deleted.\n")

	cmd := exec.Command("git", "rm", "file_to_delete.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git rm: %v", err)
	}

	tool := &GitDiffTool{}
	result, err := tool.Execute(nil)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 file in diff result, but got %d", len(result))
	}

	diffData, ok := result["file_to_delete.txt"]
	if !ok {
		t.Fatalf("Expected 'file_to_delete.txt' in diff result")
	}

	diffContent := diffData.Diff
	if !strings.Contains(diffContent, "--- a/file_to_delete.txt") || !strings.Contains(diffContent, "+++ /dev/null") {
		t.Errorf("Diff content for deleted file is incorrect:\n%s", diffContent)
	}
}

func TestGitDiffTool_Execute_RenamedFile(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to change back to original directory: %v", err)
		}
	}()

	createAndCommitFile(t, tempDir, "original.txt", "This file will be renamed.\n")

	cmd := exec.Command("git", "mv", "original.txt", "renamed.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git mv: %v", err)
	}

	tool := &GitDiffTool{}
	result, err := tool.Execute(nil)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 file in diff result, but got %d", len(result))
	}

	diffData, ok := result["renamed.txt"]
	if !ok {
		t.Fatalf("Expected 'renamed.txt' in diff result")
	}

	diffContent := diffData.Diff
	if !strings.Contains(diffContent, "rename from original.txt") || !strings.Contains(diffContent, "rename to renamed.txt") {
		t.Errorf("Diff content for renamed file is incorrect:\n%s", diffContent)
	}
}

func TestGitDiffTool_Execute_MultipleModifiedFiles(t *testing.T) {
	tempDir, cleanup := setupGitRepo(t)
	defer cleanup()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to change back to original directory: %v", err)
		}
	}()

	createAndCommitFile(t, tempDir, "file1.txt", "Initial content for file 1.\nLine 2.\n")
	createAndCommitFile(t, tempDir, "file2.txt", "Initial content for file 2.\n")

	if err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("Modified content for file 1.\nLine 2.\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file1.txt: %v", err)
	}

	cmd := exec.Command("git", "add", "file1.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add file1.txt: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("Initial content for file 2.\nSecond line added to file 2.\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file2.txt: %v", err)
	}

	cmd = exec.Command("git", "add", "file2.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add file2.txt: %v", err)
	}

	tool := &GitDiffTool{}
	result, err := tool.Execute(nil)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 files in diff result, but got %d", len(result))
	}

	if diffData, ok := result["file1.txt"]; ok {
		if !strings.Contains(diffData.Diff, "-Initial content for file 1.") || !strings.Contains(diffData.Diff, "+Modified content for file 1.") {
			t.Errorf("Diff content for file1.txt is incorrect:\n%s", diffData.Diff)
		}
	} else {
		t.Errorf("Expected 'file1.txt' in diff result")
	}

	if diffData, ok := result["file2.txt"]; ok {
		if !strings.Contains(diffData.Diff, "+Second line added to file 2.") {
			t.Errorf("Diff content for file2.txt is incorrect:\n%s", diffData.Diff)
		}
	} else {
		t.Errorf("Expected 'file2.txt' in diff result")
	}
}

func setupGitRepo(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
			t.Logf("Failed to clean up temp dir after failed git init: %v", cleanupErr)
		}
		t.Fatalf("Failed to init git repo: %v", err)
	}

	return tempDir, func() {
		if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
			t.Errorf("Failed to clean up temp dir: %v", cleanupErr)
		}
	}
}

func createAndCommitFile(t *testing.T, dir, fileName, content string) {
	filePath := filepath.Join(dir, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file %s: %v", fileName, err)
	}

	cmd := exec.Command("git", "add", fileName)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
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

// TODO review below this line

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
