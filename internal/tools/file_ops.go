package tools

import (
	"fmt"
	"os"
)

type WriteFileTool struct{}

func (t *WriteFileTool) Name() string {
	return string(ToolNameWriteFile)
}

func (t *WriteFileTool) Description() string {
	return "Write content to a specified file"
}

func (t *WriteFileTool) Execute(args map[string]any) (any, error) {
	filename, ok := args["filename"].(string)
	if !ok {
		return "", fmt.Errorf("filename parameter required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content parameter required")
	}

	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), filename), nil
}

type ReadFileTool struct{}

func (t *ReadFileTool) Name() string {
	return string(ToolNameReadFile)
}

func (t *ReadFileTool) Description() string {
	return "Read content from a specified file"
}

func (t *ReadFileTool) Execute(args map[string]any) (any, error) {
	filename, ok := args["filename"].(string)
	if !ok {
		return "", fmt.Errorf("filename parameter required")
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

type AppendFileTool struct{}

func (t *AppendFileTool) Name() string {
	return string(ToolNameAppendFile)
}

func (t *AppendFileTool) Description() string {
	return "Append content to an existing file"
}

func (t *AppendFileTool) Execute(args map[string]any) (any, error) {
	filename, ok := args["filename"].(string)
	if !ok {
		return "", fmt.Errorf("filename parameter required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content parameter required")
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open file for appending: %w", err)
	}
	defer file.Close()

	if stat, err := file.Stat(); err == nil && stat.Size() > 0 {
		content = "\n" + content
	}

	if _, err := file.WriteString(content); err != nil {
		return "", fmt.Errorf("failed to append to file: %w", err)
	}

	return fmt.Sprintf("Successfully appended %d bytes to %s", len(content), filename), nil
}
