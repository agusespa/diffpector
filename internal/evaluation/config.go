package evaluation

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/agusespa/diffpector/internal/types"
)

func LoadConfigs(path string) ([]types.EvaluationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", path, err)
	}

	var configs []types.EvaluationConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return configs, nil
}

func ValidateConfig(config types.EvaluationConfig) error {
	if config.Key == "" {
		return fmt.Errorf("missing required 'key' field")
	}
	if config.Provider == "" {
		return fmt.Errorf("missing required 'provider' field")
	}
	if len(config.Models) == 0 {
		return fmt.Errorf("missing required 'models' field")
	}
	if len(config.Prompts) == 0 {
		return fmt.Errorf("missing required 'prompts' field")
	}
	return nil
}

func ValidateConfigs(configs []types.EvaluationConfig) error {
	if len(configs) == 0 {
		return fmt.Errorf("no configurations found")
	}

	keys := make(map[string]bool)
	for _, config := range configs {
		if err := ValidateConfig(config); err != nil {
			return err
		}

		if keys[config.Key] {
			return fmt.Errorf("duplicate configuration key: %s", config.Key)
		}
		keys[config.Key] = true
	}

	return nil
}

func LoadSuite(path string) (*types.EvaluationSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read suite file at %s: %w", path, err)
	}

	var suite types.EvaluationSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse suite file %s: %w", path, err)
	}

	return &suite, nil
}
