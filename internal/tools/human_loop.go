package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type HumanLoopTool struct{}

func (t *HumanLoopTool) Name() string {
	return string(ToolNameHumanLoop)
}

func (t *HumanLoopTool) Description() string {
	return "Ask the developer for clarification, intention, or additional context when the code changes are ambiguous or require domain knowledge to properly assess. Use this ONLY when necessary - not for style preferences or general best practices."
}

func (t *HumanLoopTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "The specific question to ask the developer",
			},
		},
		"required": []string{"question"},
	}
}

func (t *HumanLoopTool) Execute(args map[string]any) (any, error) {
	question, ok := args["question"].(string)
	if !ok || question == "" {
		return nil, fmt.Errorf("question parameter is required and must be a string")
	}

	fmt.Printf("\nAgent Question: %s\n", question)
	fmt.Print("Your response: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read user input: %w", err)
	}

	return strings.TrimSpace(response), nil
}
