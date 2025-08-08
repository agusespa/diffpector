package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		expectError bool
		expected    *Config
	}{
		{
			name: "minimal config",
			configJSON: `{
				"llm": {
					"provider": "ollama",
					"model": "qwen2.5-coder",
					"base_url": "http://localhost:11434"
				}
			}`,
			expectError: false,
			expected: &Config{
				LLM: LLMConfig{
					Provider: "ollama",
					Model:    "qwen2.5-coder",
					BaseURL:  "http://localhost:11434",
				},
			},
		},
		{
			name:        "invalid json",
			configJSON:  `{"invalid": json}`,
			expectError: true,
		},
		{
			name:        "empty config",
			configJSON:  `{}`,
			expectError: false,
			expected: &Config{
				LLM: LLMConfig{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "config_test_*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.configJSON); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}
			tmpFile.Close()

			config, err := LoadConfig(tmpFile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expected != nil {
				if config.LLM.Provider != tt.expected.LLM.Provider {
					t.Errorf("Expected provider %s, got %s", tt.expected.LLM.Provider, config.LLM.Provider)
				}
				if config.LLM.Model != tt.expected.LLM.Model {
					t.Errorf("Expected model %s, got %s", tt.expected.LLM.Model, config.LLM.Model)
				}
				if config.LLM.BaseURL != tt.expected.LLM.BaseURL {
					t.Errorf("Expected base_url %s, got %s", tt.expected.LLM.BaseURL, config.LLM.BaseURL)
				}
			}
		})
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}
