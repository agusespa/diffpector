package utils

import (
	"fmt"
	"slices"
)

var UnsuitableModels = []string{
	"codellama:13b",
	"codestral",
	"qwen3:14b",
}

var ApprovedModels = []string{
	"qwen2.5-coder:14b",
	"qwen2.5-coder:7b",
	"codegemma:7b",
	"llama3.1:8b",
	"gpt-oss:20b",
	"codestral:22b",
}

func ValidateModel(model string) error {
	// Empty model is allowed for llama.cpp (model is loaded at server startup)
	if model == "" {
		return nil
	}

	if slices.Contains(UnsuitableModels, model) {
		return fmt.Errorf("model '%s' is unsupported and cannot be used", model)
	}

	if !slices.Contains(ApprovedModels, model) {
		fmt.Printf("Warning: Model '%s' is not tested - you may experience unexpected results.\n\n", model)
	}

	return nil
}
