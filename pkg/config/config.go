package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	LLM LLMConfig `json:"llm"`
}

type LLMConfig struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
}

func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider: "ollama",
			Model:    "qwen2.5-coder:14b",
			BaseURL:  "http://localhost:11434",
		},
	}
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("WARNING: Failed to read config file '%s': %v. Using default configuration.\n", filename, err)
		return DefaultConfig(), nil
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %w", filename, err)
	}

	fmt.Printf("INFO: Successfully loaded configuration from '%s'.\n", filename)
	return &config, nil
}
