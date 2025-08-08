package evaluation

import (
	"fmt"
	"strings"
	"text/template"
)

var PromptVariants = map[string]PromptVariant{
	"default": {
		Name:        "default",
		Description: "Current production prompt",
		Template:    defaultPromptTemplate,
	},
	"detailed": {
		Name:        "detailed",
		Description: "More detailed instructions with examples",
		Template:    detailedPromptTemplate,
	},
}

func GetPromptVariant(name string) (PromptVariant, error) {
	variant, exists := PromptVariants[name]
	if !exists {
		return PromptVariant{}, fmt.Errorf("prompt variant '%s' not found", name)
	}
	return variant, nil
}

func ListPromptVariants() []string {
	var names []string
	for name := range PromptVariants {
		names = append(names, name)
	}
	return names
}

func LoadPromptTemplates() (*template.Template, error) {
	tmpl := template.New("prompts")

	for name, variant := range PromptVariants {
		_, err := tmpl.New(name).Parse(variant.Template)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
		}
	}

	return tmpl, nil
}

func BuildPromptWithTemplate(variantName string, data interface{}) (string, error) {
	templates, err := LoadPromptTemplates()
	if err != nil {
		return "", fmt.Errorf("failed to load templates: %w", err)
	}

	var result strings.Builder
	err = templates.ExecuteTemplate(&result, variantName, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", variantName, err)
	}

	return result.String(), nil
}

const defaultPromptTemplate = `You are a code reviewer. Analyze the staged changes and related code context to identify potential issues.

=== STAGED CHANGES ===
{{.Diff}}

=== FILE CONTENTS ===
{{range $file, $content := .FileContents}}File: {{$file}} (CHANGED)
{{$content}}

{{end}}

{{if .SymbolAnalysis}}=== CONTEXT AND SYMBOL ANALYSIS ===
{{.SymbolAnalysis}}

{{end}}

INSTRUCTIONS:
1. Analyze the provided code changes, file contents, and symbol analysis to identify potential issues.
2. Pay special attention to the symbol analysis which shows what symbols were changed and where they are used throughout the codebase.
3. Consider the impact of changes on all the usage locations shown in the symbol analysis.
4. Provide your final review in one of two formats:
   - If no issues are found, respond with exactly: "APPROVED"
   - If issues are found, respond with a raw JSON array (no markdown formatting, no code blocks):
     [
       {
         "severity": "<severity_level>",
         "file_path": "<path_to_file>",
         "start_line": <start_line_number>,
         "end_line": <end_line_number>,
         "description": "<description_of_the_issue>"
       }
     ]
   - The "severity" should be one of: "CRITICAL", "WARNING", "MINOR".
   - The "description" should be a clear, actionable explanation of the issue.
   - Ensure line numbers are accurate based on the provided file contents.
   - IMPORTANT: Return only the raw JSON array, do not wrap it in backticks or any other formatting.

Focus on:
- Security vulnerabilities and potential risks
- Performance issues and optimization opportunities  
- Code maintainability and readability
- Potential bugs and error handling
- Breaking changes that might affect symbol usages shown in the analysis`

const detailedPromptTemplate = `You are an expert code reviewer. Your task is to thoroughly analyze the provided code changes and identify potential issues that could impact code quality, security, performance, or maintainability.

=== CODE CHANGES TO REVIEW ===
{{.Diff}}

=== COMPLETE FILE CONTENTS ===
{{range $file, $content := .FileContents}}File: {{$file}} (MODIFIED)
{{$content}}

{{end}}

{{if .SymbolAnalysis}}=== SYMBOL USAGE ANALYSIS ===
The following analysis shows how symbols in the changed code are used throughout the codebase:
{{.SymbolAnalysis}}

{{end}}

=== REVIEW INSTRUCTIONS ===

1. ANALYSIS SCOPE:
   - Examine each changed line in context of the entire file
   - Consider the impact on symbol usages shown in the analysis
   - Look for patterns that might indicate broader issues
   - Evaluate the change's effect on the overall system architecture

2. ISSUE CATEGORIES TO IDENTIFY:
   
   CRITICAL Issues (immediate action required):
   - Security vulnerabilities (SQL injection, XSS, authentication bypass, etc.)
   - Memory safety issues (buffer overflows, use-after-free, etc.)
   - Logic errors that could cause data corruption or system crashes
   - Breaking API changes that affect existing symbol usages
   
   WARNING Issues (should be addressed):
   - Performance problems (inefficient algorithms, memory leaks, etc.)
   - Error handling gaps (missing error checks, improper exception handling)
   - Code maintainability issues (complex logic, poor naming, etc.)
   - Potential race conditions or concurrency issues
   
   MINOR Issues (nice to fix):
   - Code style inconsistencies
   - Minor optimization opportunities
   - Documentation gaps
   - Unused variables or imports

3. OUTPUT FORMAT:
   - If NO issues are found, respond with exactly: "APPROVED"
   - If issues ARE found, respond with a JSON array in this exact format:

   [
     {
       "severity": "CRITICAL",
       "file_path": "path/to/file.go",
       "start_line": 42,
       "end_line": 45,
       "description": "SQL injection vulnerability: user input is directly concatenated into query string without sanitization"
     },
     {
       "severity": "WARNING", 
       "file_path": "path/to/file.go",
       "start_line": 67,
       "end_line": 67,
       "description": "Missing error handling: function call result is not checked for errors"
     }
   ]

4. QUALITY REQUIREMENTS:
   - Provide specific, actionable descriptions
   - Include accurate line numbers based on the file contents
   - Explain WHY each issue is problematic
   - Suggest how to fix the issue when possible
   - Return ONLY the JSON array, no markdown formatting or code blocks

5. SPECIAL ATTENTION:
   - Pay extra attention to how changes affect the symbol usages shown in the analysis
   - Consider backward compatibility for public APIs
   - Look for subtle bugs that might not be immediately obvious
   - Evaluate the change in the context of the broader codebase`
