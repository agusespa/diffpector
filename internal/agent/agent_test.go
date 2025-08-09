package agent

import (
	"errors"
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/internal/types"
	"github.com/agusespa/diffpector/internal/utils"
	"github.com/agusespa/diffpector/pkg/config"
)

type mockLLMProvider struct {
	model    string
	response string
	err      error
}

func (m *mockLLMProvider) Generate(prompt string) (string, error) {
	return m.response, m.err
}

func (m *mockLLMProvider) GetModel() string {
	return m.model
}

func (m *mockLLMProvider) SetModel(model string) {
	m.model = model
}

type mockWriteTool struct {
	lastFilename string
	lastContent  string
	err          error
}

func (m *mockWriteTool) Name() string {
	return string(tools.ToolNameWriteFile)
}

func (m *mockWriteTool) Description() string {
	return "Mock write file tool for testing"
}

func (m *mockWriteTool) Execute(args map[string]any) (string, error) {
	if filename, ok := args["filename"].(string); ok {
		m.lastFilename = filename
	}
	if content, ok := args["content"].(string); ok {
		m.lastContent = content
	}
	return "", m.err
}

type mockReadTool struct {
	files map[string]string
	err   error
}

func (m *mockReadTool) Name() string {
	return string(tools.ToolNameReadFile)
}

func (m *mockReadTool) Description() string {
	return "Mock read file tool for testing"
}

func (m *mockReadTool) Execute(args map[string]any) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	filename, ok := args["filename"].(string)
	if !ok {
		return "", errors.New("filename required")
	}

	if content, exists := m.files[filename]; exists {
		return content, nil
	}

	return "", errors.New("file not found")
}

func TestNewCodeReviewAgent(t *testing.T) {
	mockProvider := &mockLLMProvider{model: "test-model"}
	registry := tools.NewRegistry()
	cfg := &config.Config{}

	agent := NewCodeReviewAgent(mockProvider, registry, cfg)

	if agent.llmProvider != mockProvider {
		t.Errorf("Expected LLM provider to be set, got %v", agent.llmProvider)
	}
	if agent.toolRegistry != registry {
		t.Errorf("Expected tool registry to be set, got %v", agent.toolRegistry)
	}
	if agent.config != cfg {
		t.Errorf("Expected config to be set, got %v", agent.config)
	}
}

func TestParseStagedFiles(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []string
	}{
		{
			name:     "single file",
			output:   "main.go",
			expected: []string{"main.go"},
		},
		{
			name:     "multiple files",
			output:   "main.go\nutils.go\nconfig.go",
			expected: []string{"main.go", "utils.go", "config.go"},
		},
		{
			name:     "files with whitespace",
			output:   "  main.go  \n  utils.go  \n",
			expected: []string{"main.go", "utils.go"},
		},
		{
			name:     "empty output",
			output:   "",
			expected: []string{},
		},
		{
			name:     "only whitespace",
			output:   "   \n  \n   ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ParseStagedFiles(tt.output)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d files, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if i < len(result) && result[i] != expected {
					t.Errorf("Expected file %s, got %s", expected, result[i])
				}
			}
		})
	}
}

func TestGenerateReview(t *testing.T) {
	t.Run("approved review", func(t *testing.T) {
		mockProvider := &mockLLMProvider{
			response: "APPROVED",
		}

		registry := tools.NewRegistry()
		cfg := &config.Config{}
		agent := NewCodeReviewAgent(mockProvider, registry, cfg)

		context := &types.ReviewContext{
			Diff:         "test diff",
			ChangedFiles: []string{"test.go"},
			FileContents: make(map[string]string),
		}

		err := agent.GenerateReview(context)
		if err != nil {
			t.Errorf("Expected no error for approved review, got: %v", err)
		}
	})

	t.Run("review with issues", func(t *testing.T) {
		mockProvider := &mockLLMProvider{
			response: `[{"severity": "WARNING", "file_path": "test.go", "start_line": 1, "end_line": 1, "description": "Test issue"}]`,
		}

		registry := tools.NewRegistry()
		registry.Register(tools.ToolNameWriteFile, &mockWriteTool{})
		registry.Register(tools.ToolNameReadFile, &mockReadTool{
			files: map[string]string{
				"test.go": "package main\n",
			},
		})

		cfg := &config.Config{}
		agent := NewCodeReviewAgent(mockProvider, registry, cfg)

		context := &types.ReviewContext{
			Diff:         "test diff",
			ChangedFiles: []string{"test.go"},
			FileContents: make(map[string]string),
		}

		err := agent.GenerateReview(context)
		if err != nil {
			t.Errorf("Expected no error for review with issues, got: %v", err)
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		mockProvider := &mockLLMProvider{
			response: "invalid json",
		}

		registry := tools.NewRegistry()
		cfg := &config.Config{}
		agent := NewCodeReviewAgent(mockProvider, registry, cfg)

		context := &types.ReviewContext{
			Diff:         "test diff",
			ChangedFiles: []string{"test.go"},
			FileContents: make(map[string]string),
		}

		err := agent.GenerateReview(context)
		if err == nil || !strings.Contains(err.Error(), "format violation") {
			t.Errorf("Expected format violation error, got: %v", err)
		}
	})

	t.Run("LLM provider error", func(t *testing.T) {
		mockProvider := &mockLLMProvider{
			err: errors.New("LLM error"),
		}

		registry := tools.NewRegistry()
		cfg := &config.Config{}
		agent := NewCodeReviewAgent(mockProvider, registry, cfg)

		context := &types.ReviewContext{
			Diff:         "test diff",
			ChangedFiles: []string{"test.go"},
			FileContents: make(map[string]string),
		}

		err := agent.GenerateReview(context)
		if err == nil || !strings.Contains(err.Error(), "failed to generate code review") {
			t.Errorf("Expected LLM error, got: %v", err)
		}
	})

	t.Run("missing write_file tool", func(t *testing.T) {
		mockProvider := &mockLLMProvider{
			response: `[{"severity": "WARNING", "file_path": "test.go", "start_line": 1, "end_line": 1, "description": "Test issue"}]`,
		}

		registry := tools.NewRegistry()
		registry.Register(tools.ToolNameReadFile, &mockReadTool{})

		cfg := &config.Config{}
		agent := NewCodeReviewAgent(mockProvider, registry, cfg)

		context := &types.ReviewContext{
			Diff:         "test diff",
			ChangedFiles: []string{"test.go"},
			FileContents: make(map[string]string),
		}

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic when write_file tool is missing, but did not panic")
			}
		}()

		_ = agent.GenerateReview(context)
	})

	t.Run("missing read_file tool", func(t *testing.T) {
		mockProvider := &mockLLMProvider{
			response: `[{"severity": "WARNING", "file_path": "test.go", "start_line": 1, "end_line": 1, "description": "Test issue"}]`,
		}

		registry := tools.NewRegistry()
		registry.Register(tools.ToolNameWriteFile, &mockWriteTool{})

		cfg := &config.Config{}
		agent := NewCodeReviewAgent(mockProvider, registry, cfg)

		context := &types.ReviewContext{
			Diff:         "test diff",
			ChangedFiles: []string{"test.go"},
			FileContents: make(map[string]string),
		}

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic when read_file tool is missing, but did not panic")
			}
		}()

		_ = agent.GenerateReview(context)
	})
}
