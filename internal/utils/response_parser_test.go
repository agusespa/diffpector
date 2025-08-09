package utils

import (
	"testing"
)

func TestParseIssuesFromResponse_Approved(t *testing.T) {
	// Only exact "APPROVED" should be accepted as approval
	response := "APPROVED"

	issues, err := ParseIssuesFromResponse(response)
	if err != nil {
		t.Errorf("Expected no error for response %q, got: %v", response, err)
	}
	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for response %q, got %d", response, len(issues))
	}
}

func TestParseIssuesFromResponse_NonStandardApproval(t *testing.T) {
	// These should now be treated as format violations
	nonStandardResponses := []string{
		"The code looks good",
		"No issues found",
		"Everything looks good",
		"LGTM - looks good to me",
	}

	for _, response := range nonStandardResponses {
		_, err := ParseIssuesFromResponse(response)
		if err == nil {
			t.Errorf("Expected format violation error for response %q", response)
		}
		if !IsFormatViolation(err) {
			t.Errorf("Expected format violation error for response %q, got: %v", response, err)
		}
	}
}

func TestParseIssuesFromResponse_ValidJSON(t *testing.T) {
	jsonResponse := `[
		{
			"severity": "CRITICAL",
			"file_path": "test.go",
			"start_line": 10,
			"end_line": 12,
			"description": "SQL injection vulnerability"
		}
	]`

	issues, err := ParseIssuesFromResponse(jsonResponse)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Severity != "CRITICAL" {
		t.Errorf("Expected severity CRITICAL, got %s", issue.Severity)
	}
	if issue.Description != "SQL injection vulnerability" {
		t.Errorf("Expected specific description, got %s", issue.Description)
	}
}

func TestParseIssuesFromResponse_JSONWithText(t *testing.T) {
	response := `As a code reviewer, I found the following issues:

[
	{
		"severity": "WARNING",
		"file_path": "main.go",
		"start_line": 5,
		"end_line": 5,
		"description": "Missing error handling"
	}
]

Please address these concerns.`

	issues, err := ParseIssuesFromResponse(response)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Severity != "WARNING" {
		t.Errorf("Expected severity WARNING, got %s", issue.Severity)
	}
}

func TestParseIssuesFromResponse_CodeBlock(t *testing.T) {
	response := `Here are the issues I found:

` + "```json" + `
[
	{
		"severity": "MINOR",
		"file_path": "utils.go",
		"start_line": 15,
		"end_line": 15,
		"description": "Unused variable"
	}
]
` + "```" + `

That's all I found.`

	issues, err := ParseIssuesFromResponse(response)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Severity != "MINOR" {
		t.Errorf("Expected severity MINOR, got %s", issue.Severity)
	}
}

func TestParseIssuesFromResponse_FormatViolation(t *testing.T) {
	response := "As a code reviewer, I will focus on reviewing only the code changes shown in the diff above. Here are my findings:"

	_, err := ParseIssuesFromResponse(response)
	if err == nil {
		t.Error("Expected format violation error")
	}

	if !IsFormatViolation(err) {
		t.Errorf("Expected format violation error, got: %v", err)
	}
}

func TestParseIssuesFromResponse_InvalidJSON(t *testing.T) {
	response := "This is not JSON and contains no approval phrases and mentions critical issues but provides no valid format."

	_, err := ParseIssuesFromResponse(response)
	if err == nil {
		t.Error("Expected error for invalid response format")
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `[{"severity": "CRITICAL"}]`,
			expected: `[{"severity": "CRITICAL"}]`,
		},
		{
			input:    `Text before [{"severity": "WARNING"}] text after`,
			expected: `[{"severity": "WARNING"}]`,
		},
		{
			input:    `No JSON here`,
			expected: ``,
		},
		{
			input:    "Code block:\n```\n" + `[{"severity": "MINOR"}]` + "\n```\nEnd",
			expected: `[{"severity": "MINOR"}]`,
		},
	}

	for _, test := range tests {
		result := extractJSON(test.input)
		if result != test.expected {
			t.Errorf("extractJSON(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
