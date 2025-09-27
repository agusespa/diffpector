package tools

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
	"github.com/sourcegraph/go-diff/diff"
)

type GitDiffTool struct{}

func (t *GitDiffTool) Name() ToolName {
	return ToolNameGitDiff
}

func (t *GitDiffTool) Description() string {
	return "Get list of diff organized by file"
}

func (t *GitDiffTool) Execute(args map[string]any) (map[string]types.DiffData, error) {
	rootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	repoRootBytes, err := rootCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo root: %w", err)
	}
	repoRoot := strings.TrimSpace(string(repoRootBytes))

	cmd := exec.Command("git", "diff", "--staged")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff: %w", err)
	}

	fileDiffs, err := diff.ParseMultiFileDiff(out)
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	result := make(map[string]types.DiffData)

	for _, fd := range fileDiffs {
		name := fd.NewName
		if name == "/dev/null" {
			name = fd.OrigName
		}
		name = stripGitPrefix(name)

		absPath := filepath.Join(repoRoot, name)

		diffContentBytes, err := diff.PrintFileDiff(fd)
		if err != nil {
			return nil, fmt.Errorf("failed to print diff for file %s: %w", name, err)
		}

		diffData := types.DiffData{
			AbsolutePath: absPath,
			Diff:         string(diffContentBytes),
		}
		result[name] = diffData
	}

	return result, nil
}

func stripGitPrefix(path string) string {
	if strings.HasPrefix(path, "a/") || strings.HasPrefix(path, "b/") {
		return path[2:]
	}
	return path
}

// TODO reveiw below this line

type GitStagedFilesTool struct{}

func (t *GitStagedFilesTool) Name() string {
	return "git_staged_files"
}

func (t *GitStagedFilesTool) Description() string {
	return "Get list of staged files (git diff --staged --name-only)"
}

func (t *GitStagedFilesTool) Execute(args map[string]any) (string, error) {
	cmd := exec.Command("git", "diff", "--staged", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get staged files: %w", err)
	}

	return string(output), nil
}

type GitGrepTool struct{}

func (t *GitGrepTool) Name() string {
	return "git_grep"
}

func (t *GitGrepTool) Description() string {
	return "Search for patterns in tracked files using git grep"
}

func (t *GitGrepTool) Execute(args map[string]any) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern parameter required")
	}

	cmd := exec.Command("git", "grep", "-n", "--", pattern)
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return fmt.Sprintf("No matches found for pattern: %s", pattern), nil
		}
		return "", fmt.Errorf("failed to search pattern: %w", err)
	}

	result := string(output)
	if len(result) > 1000 {
		result = result[:1000] + "\n... (truncated - too many matches)"
	}

	return fmt.Sprintf("Search results for '%s':\n%s", pattern, result), nil
}
