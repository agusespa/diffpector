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
	parserRegistry *ParserRegistry
}

func NewSymbolContextGatherer(registry *ParserRegistry) *SymbolContextGatherer {
	return &SymbolContextGatherer{
		parserRegistry: registry,
	}
}

func (g *SymbolContextGatherer) GatherSymbolContext(affectedSymbols []types.SymbolUsage, projectRoot, primaryLanguage string) error {
	if len(affectedSymbols) == 0 {
		return nil
	}

	// Track processed symbols to avoid duplication in secondary pass
	processedRefs := make(map[string]bool)

	for i := range affectedSymbols {
		var contextBuilder strings.Builder

		// Primary Pass: Gather context for the affected symbol and extract references
		primaryContext, refs, err := g.gatherContextWithRefs(affectedSymbols[i].Symbol, projectRoot, primaryLanguage)
		if err != nil {
			continue
		}

		if primaryContext != "" {
			contextBuilder.WriteString(fmt.Sprintf(">>>>> Symbol: %s (Package: %s)\n",
				affectedSymbols[i].Symbol.Name,
				affectedSymbols[i].Symbol.Package))
			contextBuilder.WriteString(primaryContext)
		}

		// Secondary Pass: Gather definitions for referenced symbols
		for _, refName := range refs {
			// Skip if already processed or is the symbol itself
			if processedRefs[refName] || refName == affectedSymbols[i].Symbol.Name {
				continue
			}
			processedRefs[refName] = true

			// Create a symbol with just the name
			refSym := types.Symbol{Name: refName}

			// Find definitions only (no recursive ref extraction)
			secondaryContext, err := g.gatherDefinitionsOnly(refSym, projectRoot, primaryLanguage)
			if err != nil {
				continue
			}

			if secondaryContext != "" {
				contextBuilder.WriteString(fmt.Sprintf(">>>> Referenced Symbol: %s\n", refName))
				contextBuilder.WriteString(secondaryContext)
			}
		}

		affectedSymbols[i].Snippets = contextBuilder.String()
	}

	return nil
}

// gatherContextWithRefs finds definitions and usages of a symbol,
// and extracts references to other symbols used within the definition.
func (g *SymbolContextGatherer) gatherContextWithRefs(symbol types.Symbol, projectRoot, primaryLanguage string) (string, []string, error) {
	candidateFiles, err := g.findCandidateFiles(symbol, projectRoot, primaryLanguage)
	if err != nil {
		return "", nil, fmt.Errorf("failed to find candidate files for symbol %s: %w", symbol.Name, err)
	}
	if len(candidateFiles) == 0 {
		return "", nil, nil
	}

	var contextBuilder strings.Builder
	var references []string
	refMap := make(map[string]bool)
	seen := make(map[string]bool)

	for _, filePath := range candidateFiles {
		content, err := os.ReadFile(filepath.Join(filePath))
		if err != nil {
			continue
		}
		symbols, err := g.parserRegistry.ParseFile(filePath, content)
		if err != nil {
			continue
		}

		for _, s := range symbols {
			if s.Name != symbol.Name {
				continue
			}

			snippet := extractSnippet(content, s.StartLine, s.EndLine)

			if strings.HasSuffix(s.Type, "_decl") {
				key := fmt.Sprintf("decl:%s:%d-%d", filePath, s.StartLine, s.EndLine)
				if !seen[key] {
					seen[key] = true
					contextBuilder.WriteString(fmt.Sprintf(">>>>>> Definition in %s (lines %d-%d):\n", filePath, s.StartLine, s.EndLine))
					contextBuilder.WriteString(snippet)
					contextBuilder.WriteString("\n")

					// Extract references: scan for usages INSIDE this declaration
					for _, inner := range symbols {
						if inner.StartLine >= s.StartLine && inner.EndLine <= s.EndLine {
							if isUsageType(inner.Type) && inner.Name != symbol.Name {
								if !refMap[inner.Name] {
									refMap[inner.Name] = true
									references = append(references, inner.Name)
								}
							}
						}
					}
				}
			}

			if isUsageType(s.Type) {
				key := fmt.Sprintf("usage:%s:%d-%d", filePath, s.StartLine, s.EndLine)
				if !seen[key] {
					seen[key] = true
					contextBuilder.WriteString(fmt.Sprintf(">>>>>> Usage in %s (line %d):\n", filePath, s.StartLine))
					contextBuilder.WriteString(snippet)
					contextBuilder.WriteString("\n")
				}
			}
		}
	}

	return contextBuilder.String(), references, nil
}

// gatherDefinitionsOnly finds only definitions of a symbol (no usages, no recursive refs).
func (g *SymbolContextGatherer) gatherDefinitionsOnly(symbol types.Symbol, projectRoot, primaryLanguage string) (string, error) {
	candidateFiles, err := g.findCandidateFiles(symbol, projectRoot, primaryLanguage)
	if err != nil {
		return "", fmt.Errorf("failed to find candidate files for symbol %s: %w", symbol.Name, err)
	}
	if len(candidateFiles) == 0 {
		return "", nil
	}

	var contextBuilder strings.Builder
	seen := make(map[string]bool)

	for _, filePath := range candidateFiles {
		content, err := os.ReadFile(filepath.Join(filePath))
		if err != nil {
			continue
		}
		symbols, err := g.parserRegistry.ParseFile(filePath, content)
		if err != nil {
			continue
		}

		for _, s := range symbols {
			if s.Name != symbol.Name {
				continue
			}

			if strings.HasSuffix(s.Type, "_decl") {
				key := fmt.Sprintf("decl:%s:%d-%d", filePath, s.StartLine, s.EndLine)
				if !seen[key] {
					seen[key] = true
					snippet := extractSnippet(content, s.StartLine, s.EndLine)
					contextBuilder.WriteString(fmt.Sprintf(">>>>>> Definition in %s (lines %d-%d):\n", filePath, s.StartLine, s.EndLine))
					contextBuilder.WriteString(snippet)
					contextBuilder.WriteString("\n")
				}
			}
		}
	}

	return contextBuilder.String(), nil
}

func isUsageType(symbolType string) bool {
	return strings.HasSuffix(symbolType, "_usage") || strings.Contains(symbolType, "jsx_") || symbolType == "type_usage"
}

func (g *SymbolContextGatherer) findCandidateFiles(symbol types.Symbol, projectRoot, primaryLanguage string) ([]string, error) {
	files, err := g.gitGrepSearch(symbol.Name, projectRoot, primaryLanguage)
	if err != nil {
		return nil, err
	}

	return g.validateFiles(files, projectRoot), nil
}

func (g *SymbolContextGatherer) gitGrepSearch(pattern, projectRoot, primaryLanguage string) ([]string, error) {
	includePatterns := g.getIncludePatterns(primaryLanguage)

	args := []string{"grep", "-l", pattern}
	if len(includePatterns) > 0 {
		args = append(args, "--")
		args = append(args, includePatterns...)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	if err != nil {
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

		parser := g.parserRegistry.GetParser(relPath)
		if parser != nil && parser.ShouldExcludeFile(relPath, projectRoot) {
			continue
		}

		absPath := file
		if !filepath.IsAbs(file) {
			absPath = filepath.Join(projectRoot, file)
		}

		validFiles = append(validFiles, absPath)
	}

	return validFiles
}

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

func extractSnippet(content []byte, start, end int) string {
	lines := strings.Split(string(content), "\n")

	// include 2 lines before and after the symbol for context
	lo := max(0, start-3)
	hi := min(len(lines), end+2)

	return strings.Join(lines[lo:hi], "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
