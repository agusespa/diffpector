package utils

import (
	"strings"
)

func DetectLanguageFromFilePath(filePath string) string {
	parts := strings.Split(filePath, ".")
	if len(parts) < 2 {
		return "" // No extension, use plain text
	}

	ext := strings.ToLower(parts[len(parts)-1])

	languageMap := map[string]string{
		"go":    "go",
		"js":    "javascript",
		"ts":    "typescript",
		"jsx":   "jsx",
		"tsx":   "tsx",
		"py":    "python",
		"java":  "java",
		"c":     "c",
		"cpp":   "cpp",
		"cc":    "cpp",
		"cxx":   "cpp",
		"h":     "c",
		"hpp":   "cpp",
		"cs":    "csharp",
		"php":   "php",
		"rb":    "ruby",
		"rs":    "rust",
		"swift": "swift",
		"kt":    "kotlin",
		"scala": "scala", "sh": "bash",
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
