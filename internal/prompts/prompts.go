package prompts

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/agusespa/diffpector/internal/types"
)

var PromptVariants = map[string]types.PromptVariant{
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

func GetPromptVariant(name string) (types.PromptVariant, error) {
	variant, exists := PromptVariants[name]
	if !exists {
		return types.PromptVariant{}, fmt.Errorf("prompt variant '%s' not found", name)
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

func BuildPromptWithTemplate(variantName string, data any) (string, error) {
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

const defaultPromptTemplate = `You are a code reviewer. Your task is to review ONLY the code changes shown in the diff below.

=== CODE CHANGES TO REVIEW ===
{{.Diff}}

=== REFERENCE MATERIALS (DO NOT REVIEW) ===
The following sections provide context to help you understand the changes above.
DO NOT report any issues in the reference materials - they are for context only.

{{if .SymbolAnalysis}}--- Symbol Usage Context ---
{{.SymbolAnalysis}}

{{end}}=== END OF REFERENCE MATERIALS ===

CRITICAL REVIEW INSTRUCTIONS:
1. REVIEW ONLY THE "CODE CHANGES TO REVIEW" SECTION ABOVE
2. Look for lines that start with + or - in the diff - these are the ONLY lines you should analyze
3. The "REFERENCE MATERIALS" section is provided to help you understand context - DO NOT report issues in reference materials
4. If you see potential issues in the reference materials, IGNORE them unless they appear in the actual diff changes
5. Focus on REAL issues, not stylistic preferences or functionally equivalent refactoring
6. Clean refactoring that maintains the same behavior should not be flagged as issues

RESPONSE FORMAT - CRITICAL:
You MUST respond in one of these two formats ONLY:

FORMAT 1 - No issues found:
APPROVED

FORMAT 2 - Issues found:
[
  {
    "severity": "CRITICAL",
    "file_path": "path/to/file.go",
    "start_line": 10,
    "end_line": 12,
    "description": "Clear description of the issue"
  }
]

IMPORTANT RULES:
- DO NOT include any explanatory text, reasoning, or commentary
- DO NOT wrap JSON in code blocks or markdown formatting
- DO NOT say "Here are my findings" or similar phrases
- DO NOT explain your reasoning - just provide the result
- The "severity" must be exactly one of: "CRITICAL", "WARNING", "MINOR"
- If you find no issues, respond with exactly "APPROVED" and nothing else
- If you find issues, respond with only the raw JSON array and nothing else

Focus on:
- Security vulnerabilities and potential risks
- Performance issues and optimization opportunities  
- Code maintainability and readability
- Potential bugs and error handling
- Breaking changes that might affect symbol usages shown in the analysis`

const detailedPromptTemplate = `You are an expert code reviewer. Your task is to thoroughly analyze ONLY the code changes shown in the diff and identify potential issues that could impact code quality, security, performance, or maintainability.

=== CODE CHANGES TO REVIEW ===
{{.Diff}}

=== REFERENCE MATERIALS (DO NOT REVIEW) ===
The following sections provide context to help you understand the impact of the changes above.
You must NOT report any issues found in the reference materials.

{{if .SymbolAnalysis}}--- Symbol Usage Analysis (Reference Only) ---
The following shows how symbols in the changed code are used throughout the codebase:
{{.SymbolAnalysis}}

{{end}}=== END OF REFERENCE MATERIALS ===

=== REVIEW INSTRUCTIONS ===

1. ANALYSIS SCOPE - CRITICAL:
   - ONLY examine lines marked with + or - in the "CODE CHANGES TO REVIEW" section
   - COMPLETELY IGNORE all code in the "REFERENCE MATERIALS" section
   - The reference materials are provided only to help you understand the context and impact of changes
   - If you see a potential issue in the reference materials, DO NOT report it unless it appears in the actual diff
   - Focus on actual bugs, security issues, and performance problems - not stylistic changes or equivalent refactoring

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

3. RESPONSE FORMAT - CRITICAL:
You MUST respond in one of these two formats ONLY:

FORMAT 1 - No issues found:
APPROVED

FORMAT 2 - Issues found:
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

IMPORTANT RULES:
- DO NOT include any explanatory text, reasoning, or commentary
- DO NOT wrap JSON in code blocks or markdown formatting  
- DO NOT say "Here are my findings" or similar phrases
- DO NOT explain your reasoning - just provide the result
- The "severity" must be exactly one of: "CRITICAL", "WARNING", "MINOR"
- If you find no issues, respond with exactly "APPROVED" and nothing else
- If you find issues, respond with only the raw JSON array and nothing else
- Provide specific, actionable descriptions
- Include accurate line numbers based on the file contents

4. FINAL REMINDER - CRITICAL:
   - Your job is to review CHANGES, not entire files
   - Only report issues in lines that have + or - prefixes in the diff
   - Reference materials are for understanding context - NOT for finding issues
   - RESPOND WITH ONLY "APPROVED" OR JSON ARRAY - NO OTHER TEXT`
