package agent

import (
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/tools"
)

type mockTool struct {
	name     string
	response string
	err      error
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return "Mock tool for testing"
}

func (m *mockTool) Execute(params map[string]any) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

type mockLLMProvider struct {
	response string
	err      error
	model    string
}

func (m *mockLLMProvider) Generate(prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockLLMProvider) GetModel() string {
	if m.model == "" {
		return "mock-model"
	}
	return m.model
}

func (m *mockLLMProvider) SetModel(model string) {
	m.model = model
}

func TestValidateAndDetectLanguage(t *testing.T) {
	parserRegistry := tools.NewParserRegistry()
	agent := &CodeReviewAgent{
		parserRegistry: parserRegistry,
	}

	tests := []struct {
		name          string
		files         []string
		expectedLang  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "single go file",
			files:        []string{"main.go"},
			expectedLang: "go",
			expectError:  false,
		},
		{
			name:         "config files only",
			files:        []string{"package.json", "Dockerfile"},
			expectedLang: "",
			expectError:  false,
		},
		{
			name:         "mixed go and config",
			files:        []string{"main.go", "package.json"},
			expectedLang: "go",
			expectError:  false,
		},
		{
			name:         "html and css files",
			files:        []string{"index.html", "styles.css"},
			expectedLang: "",
			expectError:  false,
		},
		{
			name:         "mixed go with html and css",
			files:        []string{"main.go", "index.html", "styles.css"},
			expectedLang: "go",
			expectError:  false,
		},
		{
			name:         "python files",
			files:        []string{"script.py"},
			expectedLang: "python",
			expectError:  false,
		},
		{
			name:         "script files only",
			files:        []string{"deploy.sh", "setup.bash", "build.ps1"},
			expectedLang: "",
			expectError:  false,
		},
		{
			name:         "mixed go with script files",
			files:        []string{"main.go", "deploy.sh", "Dockerfile"},
			expectedLang: "go",
			expectError:  false,
		},
		{
			name:          "unsupported programming language",
			files:         []string{"script.rb"},
			expectedLang:  "",
			expectError:   true,
			errorContains: "unsupported language file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang, err := agent.ValidateAndDetectLanguage(tt.files)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if lang != tt.expectedLang {
					t.Errorf("Expected language '%s', got '%s'", tt.expectedLang, lang)
				}
			}
		})
	}
}
