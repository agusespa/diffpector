package llm

import (
	"fmt"
)

type ProviderType string

const (
	ProviderOllama ProviderType = "ollama"
)

type ProviderConfig struct {
	Type     ProviderType
	Model    string
	BaseURL  string
	APIKey   string
}

func NewProvider(config ProviderConfig) (Provider, error) {
	switch config.Type {
	case ProviderOllama:
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		model := config.Model
		if model == "" {
			model = "codellama"
		}
		return NewOllamaProvider(baseURL, model), nil

	default:
		return nil, fmt.Errorf("unsupported provider type: %s (only 'ollama' is supported)", config.Type)
	}
}