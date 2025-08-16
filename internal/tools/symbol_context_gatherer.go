package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
)

type SymbolContextGatherer struct {
	registry *ParserRegistry
}

func NewSymbolContextGatherer(registry *ParserRegistry) *SymbolContextGatherer {
	return &SymbolContextGatherer{
		registry: registry,
	}
}

// GatherSymbolContext finds definitions and usages of affected symbols across the codebase
func (g *SymbolContextGatherer) GatherSymbolContext(affectedSymbols []types.Symbol, projectRoot, primaryLanguage string) (string, error) {
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
func (g *SymbolContextGatherer) findSymbolDefinitions(symbol types.Symbol, projectRoot, primaryLanguage string) (string, error) {
	candidateFiles, err := g.findCandidateFiles(symbol, projectRoot, primaryLanguage)
	if err != nil {
		return "", fmt.Errorf("failed to find candidate files for symbol %s: %w", symbol.Name, err)
	}

	if len(candidateFiles) == 0 {
		return "", nil
	}

	var contextBuilder strings.Builder

	for _, filePath := range candidateFiles {

		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		symbols, err := g.registry.ParseFile(filePath, string(content))
		if err != nil {
			continue
		}

		isOriginalFile := g.isSameFile(filePath, symbol.FilePath, projectRoot)

		// Look for symbol definitions with the same name
		for _, foundSymbol := range symbols {
			if foundSymbol.Name == symbol.Name {
				// Skip showing the exact same symbol definition from the original file
				// (to avoid showing the modified symbol's own definition)
				if isOriginalFile && foundSymbol.StartLine == symbol.StartLine && foundSymbol.EndLine == symbol.EndLine {
					continue
				}

				// Use the language parser to extract context for definitions
				parser := g.registry.GetParser(filePath)
				if parser != nil {
					context, err := parser.GetSymbolContext(filePath, string(content), foundSymbol)
					if err == nil && context != "" {
						contextBuilder.WriteString(fmt.Sprintf("Definition in: %s\n", filePath))
						contextBuilder.WriteString(context)
						contextBuilder.WriteString("\n")
					}
				}
			}
		}

		// Look for symbol usages using the language parser
		parser := g.registry.GetParser(filePath)
		if parser != nil {
			usages, err := parser.FindSymbolUsages(filePath, string(content), symbol.Name)
			if err == nil {
				for _, usage := range usages {
					contextBuilder.WriteString(fmt.Sprintf("Found in: %s\n", filePath))
					contextBuilder.WriteString(usage.Context)
					contextBuilder.WriteString("\n")
				}
			}
		}
	}

	return contextBuilder.String(), nil
}

// isSameFile checks if two file paths refer to the same file, handling relative vs absolute paths
func (g *SymbolContextGatherer) isSameFile(candidateFile, symbolFile, projectRoot string) bool {
	// Direct comparison first
	if candidateFile == symbolFile {
		return true
	}

	// Convert symbol file to absolute path if it's relative
	var symbolAbsPath string
	if filepath.IsAbs(symbolFile) {
		symbolAbsPath = symbolFile
	} else {
		symbolAbsPath = filepath.Join(projectRoot, symbolFile)
	}

	// Clean both paths to handle any . or .. components
	candidateClean := filepath.Clean(candidateFile)
	symbolClean := filepath.Clean(symbolAbsPath)

	return candidateClean == symbolClean
}

// findCandidateFiles uses git grep to find files that might contain the symbol
func (g *SymbolContextGatherer) findCandidateFiles(symbol types.Symbol, projectRoot, primaryLanguage string) ([]string, error) {
	files, err := g.gitGrepSearch(symbol.Name, projectRoot, primaryLanguage)
	if err != nil {
		return nil, err
	}

	return g.validateFiles(files, projectRoot), nil
}

func (g *SymbolContextGatherer) gitGrepSearch(pattern, projectRoot, primaryLanguage string) ([]string, error) {
	// Get language-specific include patterns
	includePatterns := g.getIncludePatterns(primaryLanguage)

	// Build git grep command with pathspec patterns
	args := []string{"grep", "-l", pattern}

	// Add pathspec patterns (git grep uses -- to separate patterns from paths)
	if len(includePatterns) > 0 {
		args = append(args, "--")
		args = append(args, includePatterns...)
	}

	// Use git grep to find files containing the pattern
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
	return files, nil
}

// validateFiles ensures files exist and converts relative paths to absolute if needed
func (g *SymbolContextGatherer) validateFiles(files []string, projectRoot string) []string {
	var validFiles []string

	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		relPath := file
		if filepath.IsAbs(file) {
			var err error
			relPath, err = filepath.Rel(projectRoot, file)
			if err != nil {
				relPath = file
			}
		}

		// Use language-specific filtering to reduce noise (pass relative path)
		parser := g.registry.GetParser(relPath)
		if parser != nil && parser.ShouldExcludeFile(relPath, projectRoot) {
			continue
		}
		
		// Fallback: exclude obvious test files even if no parser found
		if strings.HasSuffix(relPath, "_test.go") {
			continue
		}

		// Convert to absolute path for final result (needed by other parts of the system)
		absPath := file
		if !filepath.IsAbs(file) {
			absPath = filepath.Join(projectRoot, file)
		}

		// Check if file exists and is readable
		if _, err := os.Stat(absPath); err != nil {
			continue
		}

		validFiles = append(validFiles, absPath)
	}

	return validFiles
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
