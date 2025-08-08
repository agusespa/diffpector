package tools

import (
	"fmt"
	"os/exec"
)

type GitDiffTool struct{}

func (t *GitDiffTool) Name() string {
	return "git_diff"
}

func (t *GitDiffTool) Description() string {
	return "Get the diff for staged changes (git diff --staged)"
}

func (t *GitDiffTool) Execute(args map[string]any) (string, error) {
	cmd := exec.Command("git", "diff", "--staged")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}

	return string(output), nil
}

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
