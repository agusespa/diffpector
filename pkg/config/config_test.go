package config

import (
	"os"
	"testing"
)

var defaultConfig = DefaultConfig()

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		configJSON  string
		expectError bool
		expected    *Config
	}{
		{
			name:     "minimal config",
			filename: "valid_config.json",
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
			name:        "no filename provided",
			filename:    "",
			configJSON:  "",
			expectError: false,
			expected:    defaultConfig,
		},
		{
			name:        "file not found returns default",
			filename:    "nonexistent.json", // LoadConfig will try to read this file
			configJSON:  "",
			expectError: false, // The *new* behavior is no error
			expected:    defaultConfig,
		},
		{
			name:        "invalid json",
			filename:    "invalid.json",
			configJSON:  `{"invalid": json}`,
			expectError: true,
			expected:    nil,
		},
		{
			name:        "empty config",
			filename:    "empty_config.json",
			configJSON:  `{}`,
			expectError: false,
			expected: &Config{
				LLM: LLMConfig{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config *Config
			var err error
			var filenameToLoad string

			if tt.filename == "" || tt.filename == "nonexistent.json" {
				filenameToLoad = tt.filename
			} else {
				tmpFile, createErr := os.CreateTemp("", tt.filename)
				if createErr != nil {
					t.Fatalf("Failed to create temp file: %v", createErr)
				}
				defer func() {
					if rErr := os.Remove(tmpFile.Name()); rErr != nil {
						t.Errorf("Failed to remove temp file %s: %v", tmpFile.Name(), rErr)
					}
				}()

				if _, writeErr := tmpFile.WriteString(tt.configJSON); writeErr != nil {
					t.Fatalf("Failed to write config: %v", writeErr)
				}
				if cErr := tmpFile.Close(); cErr != nil {
					t.Fatalf("Failed to close temp file %s: %v", tmpFile.Name(), cErr)
				}

				filenameToLoad = tmpFile.Name()
			}

			config, err = LoadConfig(filenameToLoad)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expected != nil {
				if config.LLM.Provider != tt.expected.LLM.Provider {
					t.Errorf("LLM Provider mismatch: Expected %q, Got %q", tt.expected.LLM.Provider, config.LLM.Provider)
				}
				if config.LLM.Model != tt.expected.LLM.Model {
					t.Errorf("LLM Model mismatch: Expected %q, Got %q", tt.expected.LLM.Model, config.LLM.Model)
				}
				if config.LLM.BaseURL != tt.expected.LLM.BaseURL {
					t.Errorf("LLM BaseURL mismatch: Expected %q, Got %q", tt.expected.LLM.BaseURL, config.LLM.BaseURL)
				}
			}
		})
	}
}
