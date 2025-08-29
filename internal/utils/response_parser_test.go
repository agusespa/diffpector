package utils

import (
	"testing"
)

// Test Core Requirement: Parser should recognize approval responses
func TestParseIssuesFromResponse_ApprovalResponses(t *testing.T) {
	approvalCases := []string{
		"APPROVED",           // Exact format from prompt
		"No issues found",    // Natural language approval
		"Looks good",         // Common approval phrase
		"LGTM",              // Developer shorthand
		"Clean refactoring", // Specific approval type
	}

	for _, response := range approvalCases {
		t.Run(response, func(t *testing.T) {
			issues, err := ParseIssuesFromResponse(response)
			
			if err != nil {
				t.Errorf("Should accept approval response, got error: %v", err)
			}
			if len(issues) != 0 {
				t.Errorf("Approval should return 0 issues, got %d", len(issues))
			}
		})
	}
}

// Test Core Requirement: Parser should reject ambiguous text
func TestParseIssuesFromResponse_AmbiguousText(t *testing.T) {
	ambiguousCases := []string{
		"The code has some issues",      // Mentions issues but no format
		"I found several problems",      // Mentions problems but no format  
		"There are critical vulnerabilities", // Mentions severity but no format
		"Here are my findings:",         // Incomplete response
	}

	for _, response := range ambiguousCases {
		t.Run(response, func(t *testing.T) {
			_, err := ParseIssuesFromResponse(response)
			
			if err == nil {
				t.Error("Should reject ambiguous text without proper format")
			}
			if !IsFormatViolation(err) {
				t.Errorf("Should return format violation error, got: %v", err)
			}
		})
	}
}

// Test Core Requirement: Parser should handle well-formed JSON
func TestParseIssuesFromResponse_ValidJSON(t *testing.T) {
	response := `[{"severity": "CRITICAL", "file_path": "test.go", "start_line": 10, "end_line": 12, "description": "SQL injection vulnerability"}]`

	issues, err := ParseIssuesFromResponse(response)
	
	if err != nil {
		t.Fatalf("Should parse valid JSON, got error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("Should find 1 issue, got %d", len(issues))
	}
	
	issue := issues[0]
	if issue.Severity != "CRITICAL" {
		t.Errorf("Should preserve severity, got %s", issue.Severity)
	}
	if issue.FilePath != "test.go" {
		t.Errorf("Should preserve file path, got %s", issue.FilePath)
	}
}

// Test Core Requirement: Parser should extract JSON from mixed content
func TestParseIssuesFromResponse_JSONInText(t *testing.T) {
	testCases := []struct {
		name     string
		response string
	}{
		{
			name: "JSON with surrounding text",
			response: `I found these issues:
[{"severity": "WARNING", "file_path": "main.go", "start_line": 5, "end_line": 5, "description": "Missing error handling"}]
Please fix them.`,
		},
		{
			name: "JSON in code block",
			response: "```json\n" + `[{"severity": "MINOR", "file_path": "utils.go", "start_line": 15, "end_line": 15, "description": "Unused variable"}]` + "\n```",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			issues, err := ParseIssuesFromResponse(tc.response)
			
			if err != nil {
				t.Fatalf("Should extract JSON from mixed content, got error: %v", err)
			}
			if len(issues) != 1 {
				t.Fatalf("Should find 1 issue, got %d", len(issues))
			}
		})
	}
}

// Test Core Requirement: Parser should recover from common JSON issues
func TestParseIssuesFromResponse_RecoverableIssues(t *testing.T) {
	testCases := []struct {
		name           string
		response       string
		expectedIssues int
		reason         string
	}{
		{
			name: "Incomplete JSON (missing closing bracket)",
			response: `[
  {"severity": "MINOR", "file_path": "test.go", "start_line": 10, "end_line": 10, "description": "Issue 1"},
  {"severity": "WARNING", "file_path": "test.go", "start_line": 20, "end_line": 20, "description": "Issue 2"}`,
			expectedIssues: 2,
			reason:         "Should repair missing closing bracket",
		},
		{
			name: "Individual objects without array",
			response: `Found these issues:
{"severity": "CRITICAL", "file_path": "auth.go", "start_line": 15, "end_line": 15, "description": "SQL injection"}
Also found:
{"severity": "WARNING", "file_path": "main.go", "start_line": 22, "end_line": 22, "description": "Missing error handling"}`,
			expectedIssues: 2,
			reason:         "Should extract individual JSON objects",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			issues, err := ParseIssuesFromResponse(tc.response)
			
			if err != nil {
				t.Fatalf("%s, got error: %v", tc.reason, err)
			}
			if len(issues) != tc.expectedIssues {
				t.Errorf("%s, expected %d issues, got %d", tc.reason, tc.expectedIssues, len(issues))
			}
		})
	}
}

// Test Core Requirement: Parser should fail gracefully on truly invalid input
func TestParseIssuesFromResponse_UnrecoverableIssues(t *testing.T) {
	invalidCases := []string{
		"This text mentions issues but has no valid JSON or approval",
		"[{invalid json structure without proper fields}]",
		"I found problems but won't tell you what they are",
	}

	for _, response := range invalidCases {
		t.Run(response[:20]+"...", func(t *testing.T) {
			_, err := ParseIssuesFromResponse(response)
			
			if err == nil {
				t.Error("Should fail on truly invalid input")
			}
			if !IsFormatViolation(err) {
				t.Errorf("Should return format violation, got: %v", err)
			}
		})
	}
}

// Test Core Requirement: Parser should handle multiple issues correctly
func TestParseIssuesFromResponse_MultipleIssues(t *testing.T) {
	response := `[
		{"severity": "CRITICAL", "file_path": "auth.go", "start_line": 10, "end_line": 10, "description": "SQL injection"},
		{"severity": "WARNING", "file_path": "main.go", "start_line": 20, "end_line": 20, "description": "Missing error handling"},
		{"severity": "MINOR", "file_path": "utils.go", "start_line": 30, "end_line": 30, "description": "Unused variable"}
	]`

	issues, err := ParseIssuesFromResponse(response)
	
	if err != nil {
		t.Fatalf("Should parse multiple issues, got error: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("Should find 3 issues, got %d", len(issues))
	}
	
	// Verify all severities are preserved
	severities := []string{issues[0].Severity, issues[1].Severity, issues[2].Severity}
	expected := []string{"CRITICAL", "WARNING", "MINOR"}
	
	for i, expectedSeverity := range expected {
		if severities[i] != expectedSeverity {
			t.Errorf("Issue %d should have severity %s, got %s", i, expectedSeverity, severities[i])
		}
	}
}

// Test Edge Case: Empty JSON array should be treated as no issues (like approval)
func TestParseIssuesFromResponse_EmptyArray(t *testing.T) {
	response := "[]"
	
	issues, err := ParseIssuesFromResponse(response)
	
	if err != nil {
		t.Fatalf("Should accept empty JSON array, got error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("Empty array should return 0 issues, got %d", len(issues))
	}
}
