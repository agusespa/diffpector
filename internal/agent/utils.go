package agent

import (
	"strings"
	"github.com/agusespa/diffpector/internal/types"
)

// DetectLanguageFromFilePath detects the programming language based on file extension
func DetectLanguageFromFilePath(filePath string) string {
	// Extract file extension
	parts := strings.Split(filePath, ".")
	if len(parts) < 2 {
		return "" // No extension, use plain text
	}
	
	ext := strings.ToLower(parts[len(parts)-1])
	
	// Map common file extensions to markdown language identifiers
	languageMap := map[string]string{
		"go":     "go",
		"js":     "javascript",
		"ts":     "typescript",
		"jsx":    "jsx",
		"tsx":    "tsx",
		"py":     "python",
		"java":   "java",
		"c":      "c",
		"cpp":    "cpp",
		"cc":     "cpp",
		"cxx":    "cpp",
		"h":      "c",
		"hpp":    "cpp",
		"cs":     "csharp",
		"php":    "php",
		"rb":     "ruby",
		"rs":     "rust",
		"swift":  "swift",
		"kt":     "kotlin",
		"scala":  "scala",
		"sh":     "bash",
		"bash":   "bash",
		"zsh":    "bash",
		"fish":   "bash",
		"ps1":    "powershell",
		"sql":    "sql",
		"html":   "html",
		"css":    "css",
		"scss":   "scss",
		"sass":   "sass",
		"less":   "less",
		"xml":    "xml",
		"json":   "json",
		"yaml":   "yaml",
		"yml":    "yaml",
		"toml":   "toml",
		"ini":    "ini",
		"conf":   "ini",
		"config": "ini",
		"md":     "markdown",
		"dockerfile": "dockerfile",
		"makefile":   "makefile",
		"mk":     "makefile",
	}
	
	if language, exists := languageMap[ext]; exists {
		return language
	}
	
	// For unknown extensions, return empty string (plain text)
	return ""
}

// ParseStagedFiles parses the output from git staged files command
func ParseStagedFiles(output string) []string {
	var files []string
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files
}

// CountIssuesBySeverity counts issues by their severity level
func CountIssuesBySeverity(issues []types.Issue) (critical, warning, minor int) {
	for _, issue := range issues {
		switch issue.Severity {
		case "CRITICAL":
			critical++
		case "WARNING":
			warning++
		case "MINOR":
			minor++
		}
	}
	return critical, warning, minor
}
