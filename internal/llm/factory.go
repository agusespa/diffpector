package llm

import (
	"fmt"
)

type ProviderType string

const (
	ProviderOllama ProviderType = "ollama"
	ProviderOpenAI ProviderType = "openai"
)

type ProviderConfig struct {
	Type    ProviderType
	Model   string
	BaseURL string
	APIKey  string
}

func NewProvider(config ProviderConfig) (Provider, error) {
	switch config.Type {
	case ProviderOllama:
		return NewOllamaProvider(config.BaseURL, config.Model), nil
	case ProviderOpenAI:
		return NewOpenAIProvider(config.BaseURL, config.Model, config.APIKey), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s (supported: 'ollama', 'openai')", config.Type)
	}
}
