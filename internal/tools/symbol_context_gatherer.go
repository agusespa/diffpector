package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type SymbolContextGatherer struct {
	parser *SymbolParser
}

func NewSymbolContextGatherer(parser *SymbolParser) *SymbolContextGatherer {
	return &SymbolContextGatherer{
		parser: parser,
	}
}

// GatherSymbolContext finds definitions and usages of affected symbols across the codebase
func (g *SymbolContextGatherer) GatherSymbolContext(affectedSymbols []Symbol, projectRoot, primaryLanguage string) (string, error) {
	if len(affectedSymbols) == 0 {
		return "", nil
	}

	var contextBuilder strings.Builder

	for _, symbol := range affectedSymbols {
		context, err := g.findSymbolDefinitions(symbol, projectRoot, primaryLanguage)
		if err != nil {
			// Log but don't fail the whole process
			continue
		}

		if context != "" {
			contextBuilder.WriteString(fmt.Sprintf("=== Symbol: %s (Package: %s) ===\n", symbol.Name, symbol.Package))
			contextBuilder.WriteString(context)
			contextBuilder.WriteString("\n")
		}
	}

	return contextBuilder.String(), nil
}

// findSymbolDefinitions uses grep to find candidate files, then AST parsing for accuracy
func (g *SymbolContextGatherer) findSymbolDefinitions(symbol Symbol, projectRoot, primaryLanguage string) (string, error) {
	candidateFiles, err := g.findCandidateFiles(symbol, projectRoot, primaryLanguage)
	if err != nil {
		return "", fmt.Errorf("failed to find candidate files for symbol %s: %w", symbol.Name, err)
	}

	if len(candidateFiles) == 0 {
		return "", nil
	}

	var contextBuilder strings.Builder

	for _, filePath := range candidateFiles {
		// Skip the original file where the symbol was found
		if filePath == symbol.FilePath {
			continue
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		// Use AST to find actual symbol definitions
		symbols, err := g.parser.ParseFile(filePath, string(content))
		if err != nil {
			continue
		}

		for _, foundSymbol := range symbols {
			if foundSymbol.Name == symbol.Name {
				// Extract context around the symbol
				context := g.extractSymbolContext(string(content), foundSymbol)
				if context != "" {
					contextBuilder.WriteString(fmt.Sprintf("Found in: %s\n", filePath))
					contextBuilder.WriteString(context)
					contextBuilder.WriteString("\n")
				}
			}
		}
	}

	return contextBuilder.String(), nil
}

// findCandidateFiles uses git grep to quickly find files that mention the symbol name
func (g *SymbolContextGatherer) findCandidateFiles(symbol Symbol, projectRoot, primaryLanguage string) ([]string, error) {
	// Get language-specific include patterns
	includePatterns := g.getIncludePatterns(primaryLanguage)

	// Build git grep command with pathspec patterns
	args := []string{"grep", "-l", symbol.Name}

	// Add pathspec patterns (git grep uses -- to separate patterns from paths)
	if len(includePatterns) > 0 {
		args = append(args, "--")
		args = append(args, includePatterns...)
	}

	// Use git grep to find files containing the symbol name
	cmd := exec.Command("git", args...)
	cmd.Dir = projectRoot // Set working directory
	output, err := cmd.Output()
	if err != nil {
		// If git grep finds no matches, it returns exit code 1, which is not an error for us
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("git grep command failed: %w", err)
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	return g.validateFiles(files, projectRoot), nil
}

// validateFiles ensures files exist and converts relative paths to absolute if needed
func (g *SymbolContextGatherer) validateFiles(files []string, projectRoot string) []string {
	var validFiles []string

	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		// Convert relative path to absolute path if needed
		if !filepath.IsAbs(file) {
			file = filepath.Join(projectRoot, file)
		}

		// Check if file exists and is readable
		if _, err := os.Stat(file); err == nil {
			validFiles = append(validFiles, file)
		}
	}

	return validFiles
}

// extractSymbolContext extracts relevant code context around a symbol
func (g *SymbolContextGatherer) extractSymbolContext(content string, symbol Symbol) string {
	lines := strings.Split(content, "\n")

	// Ensure we have valid line numbers
	if symbol.StartLine < 1 || symbol.StartLine > len(lines) {
		return ""
	}

	// Convert to 0-based indexing
	startIdx := symbol.StartLine - 1
	endIdx := symbol.EndLine - 1

	if endIdx >= len(lines) {
		endIdx = len(lines) - 1
	}

	// Add some context lines before and after
	contextBefore := 2
	contextAfter := 2

	actualStart := startIdx - contextBefore
	if actualStart < 0 {
		actualStart = 0
	}

	actualEnd := endIdx + contextAfter
	if actualEnd >= len(lines) {
		actualEnd = len(lines) - 1
	}

	var contextBuilder strings.Builder

	for i := actualStart; i <= actualEnd; i++ {
		lineNum := i + 1
		prefix := "  "

		// Mark the actual symbol lines
		if i >= startIdx && i <= endIdx {
			prefix = "→ "
		}

		contextBuilder.WriteString(fmt.Sprintf("%s%d: %s\n", prefix, lineNum, lines[i]))
	}

	return contextBuilder.String()
}

// getIncludePatterns returns file patterns to search for a given language
func (g *SymbolContextGatherer) getIncludePatterns(language string) []string {
	commonConfigs := []string{
		"*.json", "*.yaml", "*.yml", "*.toml", "*.xml",
		"*.md", "*.txt", "*.conf", "*.config", "*.ini",
		"Dockerfile", "Makefile", "*.mk",
	}

	languagePatterns := g.getLanguageSpecificPatterns(language)

	allPatterns := make([]string, len(languagePatterns))
	copy(allPatterns, languagePatterns)
	allPatterns = append(allPatterns, commonConfigs...)

	return allPatterns
}

func (g *SymbolContextGatherer) getLanguageSpecificPatterns(language string) []string {
	patterns := map[string][]string{
		"go": {
			"*.go", "go.mod", "go.sum",
		},
		"java": {
			"*.java", "*.gradle", "*.xml", "pom.xml",
			"build.gradle", "settings.gradle",
		},
		"javascript": {
			"*.js", "*.mjs", "*.cjs", "package.json",
			"webpack.config.js", "rollup.config.js",
		},
		"typescript": {
			"*.ts", "*.tsx", "*.js", "*.jsx",
			"package.json", "tsconfig.json",
		},
	}

	if langPatterns, exists := patterns[language]; exists {
		return langPatterns
	}

	return []string{} // Return empty for unknown languages
}
