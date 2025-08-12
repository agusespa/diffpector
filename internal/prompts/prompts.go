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
		Description: "Initial basic prompt",
		Template:    defaultPromptTemplate,
	},
	"optimized": {
		Name:        "optimized",
		Description: "Optimized prompt with strong refactor detection",
		Template:    optimizedPromptTemplate,
	},
	"optimized-v2": {
		Name:        "optimized-v2",
		Description: "Simplified optimized prompt with strong refactor detection",
		Template:    optimizedV2PromptTemplate,
	},
}

const DEFAULT_PROMPT = "optimized"

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

const optimizedPromptTemplate = `You are an expert code reviewer analyzing code changes for real issues.

=== CODE CHANGES TO REVIEW ===
{{.Diff}}

{{if .SymbolAnalysis}}=== REFERENCE CONTEXT (DO NOT REVIEW) ===
{{.SymbolAnalysis}}

{{end}}=== CLEAN REFACTORING CHECK ===
BEFORE flagging issues, ask: "Does this change produce the SAME OUTPUT for the SAME INPUT?"
If YES → This is clean refactoring → RESPOND WITH "APPROVED"

Examples of clean refactoring (ALWAYS APPROVE):
- temp = f(x); return g(temp) → return g(f(x))
- if x == "" { return true } return false → return x == ""
- Eliminating intermediate variables when result is identical
- Comment improvements without changing code behavior

=== REAL ISSUES TO DETECT ===

CRITICAL:
- Security vulnerabilities (SQL injection, exposed credentials)
- Memory safety issues (buffer overflows, resource leaks)
- Logic errors causing crashes or data corruption
- Removed error handling for operations that can fail
- Concurrency issues (removed synchronization, race conditions)

WARNING:
- Performance problems (N+1 queries, batch → individual calls in loops)
- Missing error handling for common failure cases
- Resource management issues (unclosed connections, files)
- Breaking API changes

MINOR:
- Significant readability problems (very poor naming)

IGNORE:
- Style changes that don't affect functionality
- Code simplification maintaining same behavior
- Functionally equivalent patterns
- Minor optimizations or preferences

=== RESPONSE FORMAT ===

FORMAT 1 - No issues:
APPROVED

FORMAT 2 - Issues found:
[
  {
    "severity": "CRITICAL",
    "file_path": "path/to/file.go",
    "start_line": 10,
    "end_line": 12,
    "description": "Removed SQL parameter binding, creating SQL injection vulnerability"
  }
]

RULES:
- Severity must be: "CRITICAL", "WARNING", or "MINOR"
- No explanatory text or commentary
- No markdown formatting
- If same input produces same output, it's refactoring - APPROVE it`

const optimizedV2PromptTemplate = `You are an expert code reviewer analyzing code changes for real issues.

=== CODE CHANGES TO REVIEW ===
{{.Diff}}

{{if .SymbolAnalysis}}
=== REFERENCE CONTEXT (DO NOT REVIEW) ===
{{.SymbolAnalysis}}
{{end}}

=== ISSUES TO DETECT ===

CLEAN REFACTORING CHECK: BEFORE flagging issues, ask "Does this change produce the SAME OUTPUT for the SAME INPUT?" If YES → This is clean refactoring → RESPOND WITH "APPROVED"

CRITICAL:
- Security vulnerabilities (SQL injection, exposed credentials)
- Memory safety issues (buffer overflows, resource leaks)
- Logic errors causing crashes or data corruption
- Lacking error handling for operations that can fail
- Concurrency issues (lacking synchronization, race conditions)

WARNING:
- Performance problems (N+1 queries, batch → individual calls in loops)
- Missing error handling for common failure cases
- Resource management issues (unclosed connections, files)
- Breaking API changes

MINOR:
- Significant readability problems (very poor naming)

IGNORE:
- Style changes that don't affect functionality
- Code simplification maintaining same behavior
- Functionally equivalent patterns
- Minor optimizations or preferences

=== RESPONSE FORMAT ===

FORMAT 1 - No issues:
APPROVED

FORMAT 2 - Issues found: must be valid json (array of objects):
[
  {
    "severity": "CRITICAL",
    "file_path": "path/to/file.go",
    "start_line": 10,
    "end_line": 12,
    "description": "Removed SQL parameter binding, creating SQL injection vulnerability"
  }
]

RULES:
- Severity must be: "CRITICAL", "WARNING", or "MINOR"
- No explanatory text or commentary
- No markdown formatting`
