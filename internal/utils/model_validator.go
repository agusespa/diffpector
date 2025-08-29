package utils

import (
	"fmt"
	"slices"
)

var ProblematicModels = []string{
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
	if slices.Contains(ProblematicModels, model) {
		return fmt.Errorf("model '%s' has known issues and cannot be used", model)
	}
	return nil
}

func WarnIfUnapproved(model string) {
	if !slices.Contains(ApprovedModels, model) {
		fmt.Printf("Warning: Model '%s' is not tested. You may experience unexpected results.\n\n", model)
	}
}