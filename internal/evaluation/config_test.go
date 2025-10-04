package evaluation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

func TestLoadConfigs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-config-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configs := []types.EvaluationConfig{
		{
			Key:      "test-config",
			Provider: "ollama",
			BaseURL:  "http://localhost:11434",
			Models:   []string{"model1", "model2"},
			Prompts:  []string{"prompt1", "prompt2"},
			Runs:     3,
		},
	}

	configFile := filepath.Join(tempDir, "config.json")
	data, err := json.Marshal(configs)
	if err != nil {
		t.Fatalf("Failed to marshal configs: %v", err)
	}

	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loadedConfigs, err := LoadConfigs(configFile)
	if err != nil {
		t.Fatalf("LoadConfigs failed: %v", err)
	}

	if len(loadedConfigs) != 1 {
		t.Errorf("Expected 1 config, got %d", len(loadedConfigs))
	}

	config := loadedConfigs[0]
	if config.Key != "test-config" {
		t.Errorf("Expected key 'test-config', got '%s'", config.Key)
	}

	if config.Provider != "ollama" {
		t.Errorf("Expected provider 'ollama', got '%s'", config.Provider)
	}

	if len(config.Models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(config.Models))
	}

	if config.Runs != 3 {
		t.Errorf("Expected 3 runs, got %d", config.Runs)
	}
}

func TestLoadConfigs_FileNotFound(t *testing.T) {
	_, err := LoadConfigs("nonexistent.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("Expected 'failed to read config file' in error, got: %v", err)
	}
}

func TestLoadConfigs_InvalidJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-invalid-config-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	invalidFile := filepath.Join(tempDir, "invalid.json")
	err = os.WriteFile(invalidFile, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	_, err = LoadConfigs(invalidFile)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to parse config file") {
		t.Errorf("Expected 'failed to parse config file' in error, got: %v", err)
	}
}

func TestValidateConfig(t *testing.T) {
	// Valid config
	validConfig := types.EvaluationConfig{
		Key:      "valid",
		Provider: "ollama",
		Models:   []string{"model1"},
		Prompts:  []string{"prompt1"},
	}

	err := ValidateConfig(validConfig)
	if err != nil {
		t.Errorf("Expected valid config to pass validation, got: %v", err)
	}

	// Missing key
	missingKey := types.EvaluationConfig{
		Provider: "ollama",
		Models:   []string{"model1"},
		Prompts:  []string{"prompt1"},
	}

	err = ValidateConfig(missingKey)
	if err == nil {
		t.Error("Expected error for missing key")
	}
	if !strings.Contains(err.Error(), "missing required 'key' field") {
		t.Errorf("Expected key error, got: %v", err)
	}

	// Missing provider
	missingProvider := types.EvaluationConfig{
		Key:     "test",
		Models:  []string{"model1"},
		Prompts: []string{"prompt1"},
	}

	err = ValidateConfig(missingProvider)
	if err == nil {
		t.Error("Expected error for missing provider")
	}
	if !strings.Contains(err.Error(), "missing required 'provider' field") {
		t.Errorf("Expected provider error, got: %v", err)
	}

	// Missing models
	missingModels := types.EvaluationConfig{
		Key:      "test",
		Provider: "ollama",
		Prompts:  []string{"prompt1"},
	}

	err = ValidateConfig(missingModels)
	if err == nil {
		t.Error("Expected error for missing models")
	}
	if !strings.Contains(err.Error(), "missing required 'models' field") {
		t.Errorf("Expected models error, got: %v", err)
	}

	// Missing prompts
	missingPrompts := types.EvaluationConfig{
		Key:      "test",
		Provider: "ollama",
		Models:   []string{"model1"},
	}

	err = ValidateConfig(missingPrompts)
	if err == nil {
		t.Error("Expected error for missing prompts")
	}
	if !strings.Contains(err.Error(), "missing required 'prompts' field") {
		t.Errorf("Expected prompts error, got: %v", err)
	}
}

func TestValidateConfigs(t *testing.T) {
	// Valid configs
	validConfigs := []types.EvaluationConfig{
		{
			Key:      "config1",
			Provider: "ollama",
			Models:   []string{"model1"},
			Prompts:  []string{"prompt1"},
		},
		{
			Key:      "config2",
			Provider: "openai",
			Models:   []string{"model2"},
			Prompts:  []string{"prompt2"},
		},
	}

	err := ValidateConfigs(validConfigs)
	if err != nil {
		t.Errorf("Expected valid configs to pass validation, got: %v", err)
	}

	// Empty configs
	err = ValidateConfigs([]types.EvaluationConfig{})
	if err == nil {
		t.Error("Expected error for empty configs")
	}
	if !strings.Contains(err.Error(), "no configurations found") {
		t.Errorf("Expected empty configs error, got: %v", err)
	}

	// Duplicate keys
	duplicateConfigs := []types.EvaluationConfig{
		{
			Key:      "duplicate",
			Provider: "ollama",
			Models:   []string{"model1"},
			Prompts:  []string{"prompt1"},
		},
		{
			Key:      "duplicate",
			Provider: "openai",
			Models:   []string{"model2"},
			Prompts:  []string{"prompt2"},
		},
	}

	err = ValidateConfigs(duplicateConfigs)
	if err == nil {
		t.Error("Expected error for duplicate keys")
	}
	if !strings.Contains(err.Error(), "duplicate configuration key") {
		t.Errorf("Expected duplicate key error, got: %v", err)
	}
}
