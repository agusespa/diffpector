package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
)

// ParseIssuesFromResponse parses LLM response into issues using the same logic as the main agent
func ParseIssuesFromResponse(review string) ([]types.Issue, error) {
	review = strings.TrimSpace(review)

	// Check for explicit approval (exact match)
	if review == "APPROVED" {
		return []types.Issue{}, nil
	}

	// Try to parse as JSON directly first
	var issues []types.Issue
	if err := json.Unmarshal([]byte(review), &issues); err == nil {
		return issues, nil
	}

	// Try to extract JSON from the response (handles code blocks and embedded JSON)
	jsonContent := extractJSON(review)
	if jsonContent != "" {
		if err := json.Unmarshal([]byte(jsonContent), &issues); err == nil {
			return issues, nil
		}
	}

	// If we get here, the model didn't follow the expected format
	// Return a parsing error that the evaluator can handle appropriately
	return nil, &FormatViolationError{
		Response: truncateString(review, 500),
		Reason:   "Model response does not match expected format (APPROVED or JSON array)",
	}
}

// FormatViolationError represents a case where the model didn't follow the expected response format
type FormatViolationError struct {
	Response string
	Reason   string
}

func (e *FormatViolationError) Error() string {
	return fmt.Sprintf("format violation: %s. Response: %s", e.Reason, e.Response)
}

// IsFormatViolation checks if an error is a format violation
func IsFormatViolation(err error) bool {
	_, ok := err.(*FormatViolationError)
	return ok
}

// extractJSON attempts to find and extract JSON content from a response
func extractJSON(response string) string {
	response = strings.TrimSpace(response)

	// Handle code blocks
	if strings.Contains(response, "```") {
		lines := strings.Split(response, "\n")
		inCodeBlock := false
		var jsonLines []string

		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inCodeBlock = !inCodeBlock
				continue
			}
			if inCodeBlock {
				jsonLines = append(jsonLines, line)
			}
		}

		if len(jsonLines) > 0 {
			response = strings.Join(jsonLines, "\n")
			response = strings.TrimSpace(response)
		}
	}

	// If it starts with [, assume it's already JSON
	if strings.HasPrefix(response, "[") {
		return response
	}

	// Look for JSON array in the response
	startIdx := strings.Index(response, "[")
	if startIdx == -1 {
		return ""
	}

	// Find the matching closing bracket
	bracketCount := 0
	endIdx := -1
	for i := startIdx; i < len(response); i++ {
		if response[i] == '[' {
			bracketCount++
		} else if response[i] == ']' {
			bracketCount--
			if bracketCount == 0 {
				endIdx = i
				break
			}
		}
	}

	if endIdx == -1 {
		return ""
	}

	return response[startIdx : endIdx+1]
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
