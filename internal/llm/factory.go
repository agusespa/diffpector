package llm

import (
	"fmt"
)

type ProviderType string

const (
	ProviderOllama ProviderType = "ollama"
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
	default:
		return nil, fmt.Errorf("unsupported provider type: %s (only 'ollama' is supported)", config.Type)
	}
}
