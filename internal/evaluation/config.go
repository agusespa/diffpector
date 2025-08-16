package evaluation

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/agusespa/diffpector/internal/types"
)

// LoadConfigs loads evaluation configurations from a JSON file
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

// ValidateConfig validates a single evaluation configuration
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

// ValidateConfigs validates multiple evaluation configurations
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

// FilterByKey filters configurations by key, returns all if key is empty
func FilterByKey(configs []types.EvaluationConfig, key string) []types.EvaluationConfig {
	if key == "" {
		return configs
	}

	var filtered []types.EvaluationConfig
	for _, config := range configs {
		if config.Key == key {
			filtered = append(filtered, config)
		}
	}
	return filtered
}

// GetConfigByKey returns a configuration by its key
func GetConfigByKey(configs []types.EvaluationConfig, key string) (types.EvaluationConfig, error) {
	for _, config := range configs {
		if config.Key == key {
			return config, nil
		}
	}
	return types.EvaluationConfig{}, fmt.Errorf("configuration with key '%s' not found", key)
}

// GetDefaultRuns returns the number of runs for a config, defaulting to 1 if not specified or invalid
func GetDefaultRuns(config types.EvaluationConfig) int {
	if config.Runs <= 0 {
		return 1
	}
	return config.Runs
}

// ListConfigKeys returns a list of all configuration keys
func ListConfigKeys(configs []types.EvaluationConfig) []string {
	keys := make([]string, len(configs))
	for i, config := range configs {
		keys[i] = config.Key
	}
	return keys
}

// LoadSuite loads an evaluation suite from a JSON file
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
