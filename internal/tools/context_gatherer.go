package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// ContextGatherer finds related code context for symbols
type ContextGatherer struct {
	gitGrepTool *GitGrepTool
	readTool    *ReadFileTool
}

func NewContextGatherer() *ContextGatherer {
	return &ContextGatherer{
		gitGrepTool: &GitGrepTool{},
		readTool:    &ReadFileTool{},
	}
}

// ContextResult represents found context for a symbol
type ContextResult struct {
	Symbol       Symbol
	Usages       []Usage
	RelatedFiles []string
}

// Usage represents a usage of a symbol with enhanced context
type Usage struct {
	FilePath         string
	Line             int
	Context          string   // The line containing the usage
	SurroundingLines []string // Lines before and after the usage (with line numbers)
	ContainingFunc   string   // Name of the function containing this usage
}

// GatherContextForSymbols finds usages and related context for the given symbols
func (cg *ContextGatherer) GatherContextForSymbols(symbols []Symbol) []ContextResult {
	var results []ContextResult

	for _, symbol := range symbols {
		result := ContextResult{
			Symbol: symbol,
		}

		// Find usages of this symbol
		usages := cg.findSymbolUsages(symbol)
		result.Usages = usages

		// Extract unique file paths from usages
		fileMap := make(map[string]bool)
		for _, usage := range usages {
			if usage.FilePath != symbol.FilePath {
				fileMap[usage.FilePath] = true
			}
		}

		for filePath := range fileMap {
			result.RelatedFiles = append(result.RelatedFiles, filePath)
		}

		results = append(results, result)
	}

	return results
}

// findSymbolUsages uses git grep to find all usages of a symbol with enhanced context
func (cg *ContextGatherer) findSymbolUsages(symbol Symbol) []Usage {
	var usages []Usage

	// Search for the symbol name
	searchPatterns := cg.generateSearchPatterns(symbol)

	for _, pattern := range searchPatterns {
		foundUsages := cg.searchPatternWithContext(pattern, symbol.Name)
		usages = append(usages, foundUsages...)
	}

	// Remove duplicates
	usages = cg.removeDuplicateUsages(usages)

	// Enhance each usage with surrounding context
	for i := range usages {
		cg.enhanceUsageContext(&usages[i])
	}

	return usages
}

// generateSearchPatterns creates search patterns based on symbol type
func (cg *ContextGatherer) generateSearchPatterns(symbol Symbol) []string {
	patterns := []string{
		// Basic symbol name
		fmt.Sprintf(`\b%s\b`, symbol.Name),
	}

	switch symbol.Type {
	case "function":
		// Function calls
		patterns = append(patterns, fmt.Sprintf(`%s\s*\(`, symbol.Name))
	case "method":
		// Method calls
		patterns = append(patterns, fmt.Sprintf(`\.%s\s*\(`, symbol.Name))
	case "type":
		// Type usage in declarations, type assertions, etc.
		patterns = append(patterns,
			fmt.Sprintf(`%s\{`, symbol.Name),   // struct literals
			fmt.Sprintf(`\*%s`, symbol.Name),   // pointer types
			fmt.Sprintf(`\[\]%s`, symbol.Name), // slice types
			fmt.Sprintf(`%s\)`, symbol.Name),   // type assertions
		)
	case "variable", "constant":
		// Variable/constant usage
		patterns = append(patterns, fmt.Sprintf(`\b%s\b`, symbol.Name))
	}

	return patterns
}

// parseGrepLine parses a git grep output line into a Usage
func (cg *ContextGatherer) parseGrepLine(line string) *Usage {
	// Git grep output format: filename:line_number:content
	parts := strings.SplitN(line, ":", 3)
	if len(parts) < 3 {
		return nil
	}

	filePath := parts[0]
	lineNum := 0
	if _, err := fmt.Sscanf(parts[1], "%d", &lineNum); err != nil {
		return nil
	}

	context := strings.TrimSpace(parts[2])

	return &Usage{
		FilePath: filePath,
		Line:     lineNum,
		Context:  context,
	}
}

// removeDuplicateUsages removes duplicate usage entries
func (cg *ContextGatherer) removeDuplicateUsages(usages []Usage) []Usage {
	seen := make(map[string]bool)
	var unique []Usage

	for _, usage := range usages {
		key := fmt.Sprintf("%s:%d", usage.FilePath, usage.Line)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, usage)
		}
	}

	return unique
}

// searchPatternWithContext executes git grep for a specific pattern
func (cg *ContextGatherer) searchPatternWithContext(pattern, symbolName string) []Usage {
	var usages []Usage

	// Use simple git grep first to find matches
	cmd := exec.Command("git", "grep", "-n", "-E", pattern)
	output, err := cmd.Output()
	if err != nil {
		// No matches found or error - return empty
		return usages
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		usage := cg.parseGrepLine(line)
		if usage != nil {
			usages = append(usages, *usage)
		}
	}

	return usages
}

// enhanceUsageContext adds surrounding context to a usage
func (cg *ContextGatherer) enhanceUsageContext(usage *Usage) {
	// Read the file content to get more context
	args := map[string]any{"filename": usage.FilePath}
	fileContent, err := cg.readTool.Execute(args)
	if err != nil {
		return
	}

	lines := strings.Split(fileContent, "\n")
	if usage.Line <= 0 || usage.Line > len(lines) {
		return
	}

	// Extract surrounding lines (8 lines before and after for good context)
	contextSize := 8
	startLine := max(0, usage.Line-contextSize-1)
	endLine := min(len(lines), usage.Line+contextSize)

	var surroundingLines []string
	for i := startLine; i < endLine; i++ {
		prefix := "  "
		if i == usage.Line-1 { // The actual usage line (0-indexed)
			prefix = "→ "
		}
		surroundingLines = append(surroundingLines, fmt.Sprintf("%3d: %s%s", i+1, prefix, lines[i]))
	}
	usage.SurroundingLines = surroundingLines

	// Find the containing function - this is useful structural information
	usage.ContainingFunc = cg.findContainingFunction(lines, usage.Line-1)
}

// findContainingFunction finds the function that contains the given line
func (cg *ContextGatherer) findContainingFunction(lines []string, lineIndex int) string {
	// Search backwards for function declaration
	for i := lineIndex; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// Look for function declarations
		if strings.HasPrefix(line, "func ") {
			// Extract function name
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				funcName := parts[1]
				// Remove receiver if it's a method
				if strings.Contains(funcName, ")") {
					parenIndex := strings.Index(funcName, ")")
					if parenIndex+1 < len(funcName) {
						funcName = funcName[parenIndex+1:]
					}
				}
				// Remove parameters
				if parenIndex := strings.Index(funcName, "("); parenIndex != -1 {
					funcName = funcName[:parenIndex]
				}
				return strings.TrimSpace(funcName)
			}
		}
	}

	return "unknown"
}

// Helper functions
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

// GetRelatedFileContent reads the content of related files
func (cg *ContextGatherer) GetRelatedFileContent(filePaths []string) map[string]string {
	content := make(map[string]string)

	for _, filePath := range filePaths {
		args := map[string]any{"filename": filePath}
		fileContent, err := cg.readTool.Execute(args)
		if err == nil {
			content[filePath] = fileContent
		}
	}

	return content
}
