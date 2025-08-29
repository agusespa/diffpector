package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
)

// ParseIssuesFromResponse parses LLM response into issues
func ParseIssuesFromResponse(review string) ([]types.Issue, error) {
	review = strings.TrimSpace(review)

	// 1. Check for approval responses (flexible matching)
	if isApprovalResponse(review) {
		return []types.Issue{}, nil
	}

	// 2. Try direct JSON parsing
	if issues, err := tryDirectJSONParse(review); err == nil {
		return issues, nil
	}

	// 3. Try to extract and parse JSON from mixed content
	if issues, err := tryExtractAndParseJSON(review); err == nil {
		return issues, nil
	}

	// 4. Try to repair incomplete JSON
	if issues, err := tryRepairIncompleteJSON(review); err == nil {
		return issues, nil
	}

	// 5. Try to extract individual issue objects (even if array is malformed)
	if issues, err := tryExtractIndividualIssues(review); err == nil && len(issues) > 0 {
		return issues, nil
	}

	// 6. If all else fails, return format violation
	return nil, &FormatViolationError{
		Response: truncateString(review, 500),
		Reason:   "Could not parse response as APPROVED or valid JSON array",
	}
}

type FormatViolationError struct {
	Response string
	Reason   string
}

func (e *FormatViolationError) Error() string {
	return fmt.Sprintf("format violation: %s. Response: %s", e.Reason, e.Response)
}

func IsFormatViolation(err error) bool {
	_, ok := err.(*FormatViolationError)
	return ok
}

func isApprovalResponse(response string) bool {
	response = strings.ToUpper(strings.TrimSpace(response))
	
	approvalPatterns := []string{
		"APPROVED",
		"NO ISSUES FOUND",
		"LOOKS GOOD",
		"LGTM",
		"NO PROBLEMS",
		"CLEAN REFACTORING",
	}
	
	for _, pattern := range approvalPatterns {
		if response == pattern {
			return true
		}
	}
	
	if len(response) < 50 && (strings.Contains(response, "NO ISSUE") || 
		strings.Contains(response, "GOOD") || 
		strings.Contains(response, "CLEAN")) {
		return true
	}
	
	return false
}

func tryDirectJSONParse(response string) ([]types.Issue, error) {
	var issues []types.Issue
	err := json.Unmarshal([]byte(response), &issues)
	return issues, err
}

func tryExtractAndParseJSON(response string) ([]types.Issue, error) {
	jsonContent := extractJSON(response)
	if jsonContent == "" {
		return nil, fmt.Errorf("no JSON found")
	}
	
	var issues []types.Issue
	err := json.Unmarshal([]byte(jsonContent), &issues)
	return issues, err
}

func tryRepairIncompleteJSON(response string) ([]types.Issue, error) {
	lastBrace := strings.LastIndex(response, "}")
	if lastBrace == -1 {
		return nil, fmt.Errorf("no closing brace found")
	}
	
	truncated := response[:lastBrace+1]
	
	// If it starts with [ but doesn't end with ], try adding the closing bracket
	if strings.HasPrefix(strings.TrimSpace(truncated), "[") && 
	   !strings.HasSuffix(strings.TrimSpace(truncated), "]") {
		repaired := strings.TrimSpace(truncated) + "\n]"
		
		var issues []types.Issue
		if err := json.Unmarshal([]byte(repaired), &issues); err == nil {
			return issues, nil
		}
	}
	
	return nil, fmt.Errorf("could not repair JSON")
}

// tryExtractIndividualIssues extracts individual issue objects even if array structure is broken
func tryExtractIndividualIssues(response string) ([]types.Issue, error) {
	// Use regex to find individual issue objects
	issuePattern := `\{[^{}]*"severity"[^{}]*"file_path"[^{}]*"start_line"[^{}]*"end_line"[^{}]*"description"[^{}]*\}`
	re := regexp.MustCompile(issuePattern)
	matches := re.FindAllString(response, -1)
	
	if len(matches) == 0 {
		return nil, fmt.Errorf("no issue objects found")
	}
	
	var issues []types.Issue
	for _, match := range matches {
		var issue types.Issue
		if err := json.Unmarshal([]byte(match), &issue); err == nil {
			issues = append(issues, issue)
		}
	}
	
	if len(issues) == 0 {
		return nil, fmt.Errorf("no valid issues parsed")
	}
	
	return issues, nil
}

func extractJSON(response string) string {
	response = strings.TrimSpace(response)

	if strings.Contains(response, "```") {
		if extracted := extractFromCodeBlock(response); extracted != "" {
			return extracted
		}
	}

	startIdx := strings.Index(response, "[")
	if startIdx == -1 {
		return ""
	}

	bracketCount := 0
	inString := false
	escaped := false
	endIdx := -1

	for i := startIdx; i < len(response); i++ {
		char := response[i]
		
		if escaped {
			escaped = false
			continue
		}
		
		if char == '\\' {
			escaped = true
			continue
		}
		
		if char == '"' {
			inString = !inString
			continue
		}
		
		if !inString {
			if char == '[' || char == '{' {
				bracketCount++
			} else if char == ']' || char == '}' {
				bracketCount--
				if bracketCount == 0 && char == ']' {
					endIdx = i
					break
				}
			}
		}
	}

	if endIdx == -1 {
		return ""
	}

	return response[startIdx : endIdx+1]
}

func extractFromCodeBlock(response string) string {
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
		return strings.TrimSpace(strings.Join(jsonLines, "\n"))
	}
	return ""
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
