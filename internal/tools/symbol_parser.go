package tools

import (
	"fmt"
	"strings"
)

// Symbol represents a code symbol (function, type, variable, etc.)
type Symbol struct {
	Name     string
	Type     string // "function", "type", "variable", "method", "constant"
	Package  string
	FilePath string
	Line     int
}

// SymbolParser parses code to extract symbols using language-specific parsers
type SymbolParser struct {
	parserRegistry *ParserRegistry
}

func NewSymbolParser() *SymbolParser {
	return &SymbolParser{
		parserRegistry: NewParserRegistry(),
	}
}

// ParseChangedFiles parses symbols from a list of files with their contents
func (sp *SymbolParser) ParseChangedFiles(fileContents map[string]string) []Symbol {
	var allSymbols []Symbol
	
	for filePath, content := range fileContents {
		symbols := sp.ParseFile(filePath, content)
		allSymbols = append(allSymbols, symbols...)
	}
	
	return allSymbols
}

// ParseFile parses a file and extracts all symbols using the appropriate language parser
func (sp *SymbolParser) ParseFile(filePath, content string) []Symbol {
	return sp.parserRegistry.ParseFile(filePath, content)
}

// RegisterParser allows registering additional language parsers
func (sp *SymbolParser) RegisterParser(parser LanguageParser) {
	sp.parserRegistry.RegisterParser(parser)
}

// GetSupportedLanguages returns the list of supported languages
func (sp *SymbolParser) GetSupportedLanguages() []string {
	return sp.parserRegistry.GetSupportedLanguages()
}



// ExtractChangedFilesFromDiff extracts the list of changed files from a git diff
func (sp *SymbolParser) ExtractChangedFilesFromDiff(diff string) []string {
	var changedFiles []string
	lines := strings.Split(diff, "\n")
	
	for _, line := range lines {
		// Look for the +++ b/filename pattern which indicates a changed file
		if strings.HasPrefix(line, "+++") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				filePath := strings.TrimPrefix(parts[1], "b/")
				// Skip /dev/null (deleted files)
				if filePath != "/dev/null" {
					changedFiles = append(changedFiles, filePath)
				}
			}
		}
	}
	
	return changedFiles
}

// GetDiffContext extracts the changed line ranges for better context analysis
func (sp *SymbolParser) GetDiffContext(diff string) map[string][]LineRange {
	context := make(map[string][]LineRange)
	lines := strings.Split(diff, "\n")
	
	var currentFile string
	
	for _, line := range lines {
		if strings.HasPrefix(line, "+++") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentFile = strings.TrimPrefix(parts[1], "b/")
			}
		} else if strings.HasPrefix(line, "@@") && currentFile != "" {
			// Parse hunk header: @@ -oldStart,oldCount +newStart,newCount @@
			// Extract the new file line range
			if lineRange := sp.parseHunkHeader(line); lineRange != nil {
				context[currentFile] = append(context[currentFile], *lineRange)
			}
		}
	}
	
	return context
}

// LineRange represents a range of lines that were changed
type LineRange struct {
	Start int
	Count int
}

// parseHunkHeader parses a git diff hunk header to extract line range
func (sp *SymbolParser) parseHunkHeader(header string) *LineRange {
	// Example: @@ -1,4 +1,6 @@
	parts := strings.Fields(header)
	if len(parts) < 3 {
		return nil
	}
	
	// Parse the +newStart,newCount part
	newPart := parts[2] // +1,6
	if !strings.HasPrefix(newPart, "+") {
		return nil
	}
	
	newPart = strings.TrimPrefix(newPart, "+")
	rangeParts := strings.Split(newPart, ",")
	
	var start, count int
	if len(rangeParts) >= 1 {
		if _, err := fmt.Sscanf(rangeParts[0], "%d", &start); err != nil {
			return nil
		}
	}
	if len(rangeParts) >= 2 {
		if _, err := fmt.Sscanf(rangeParts[1], "%d", &count); err != nil {
			count = 1 // Default to 1 if parsing fails
		}
	} else {
		count = 1
	}
	
	return &LineRange{Start: start, Count: count}
}

