package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
)

func NotifyUserIfReportNotIgnored(gitignorePath string) error {
	const reportFilename = "diffpector_report.md"

	if _, err := os.Stat(reportFilename); os.IsNotExist(err) {
		return nil
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("'%s' exists but is not in a .gitignore file", reportFilename)
		}
		return fmt.Errorf("could not read .gitignore file: %w", err)
	}

	if !strings.Contains(string(content), reportFilename) {
		return fmt.Errorf("'%s' exists but is not in your .gitignore file. Please consider adding it to avoid including it in the context of future analyses", reportFilename)
	}

	return nil
}

func DetectLanguageFromFilePath(filePath string) string {
	parts := strings.Split(filePath, ".")
	if len(parts) < 2 {
		return "" // No extension, use plain text
	}

	ext := strings.ToLower(parts[len(parts)-1])

	languageMap := map[string]string{
		"go":         "go",
		"js":         "javascript",
		"ts":         "typescript",
		"jsx":        "jsx",
		"tsx":        "tsx",
		"py":         "python",
		"java":       "java",
		"c":          "c",
		"cpp":        "cpp",
		"cc":         "cpp",
		"cxx":        "cpp",
		"h":          "c",
		"hpp":        "cpp",
		"cs":         "csharp",
		"php":        "php",
		"rb":         "ruby",
		"rs":         "rust",
		"swift":      "swift",
		"kt":         "kotlin",
		"scala":      "scala",
		"sh":         "bash",
		"bash":       "bash",
		"zsh":        "bash",
		"fish":       "bash",
		"ps1":        "powershell",
		"sql":        "sql",
		"html":       "html",
		"css":        "css",
		"scss":       "scss",
		"sass":       "sass",
		"less":       "less",
		"xml":        "xml",
		"json":       "json",
		"yaml":       "yaml",
		"yml":        "yaml",
		"toml":       "toml",
		"ini":        "ini",
		"conf":       "ini",
		"config":     "ini",
		"md":         "markdown",
		"dockerfile": "dockerfile",
		"makefile":   "makefile",
		"mk":         "makefile",
	}

	if language, exists := languageMap[ext]; exists {
		return language
	}

	return ""
}

func ParseStagedFiles(output string) []string {
	var files []string
	lines := strings.SplitSeq(strings.TrimSpace(output), "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files
}

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
