package tools

import (
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_c "github.com/tree-sitter/tree-sitter-c/bindings/go"
)

type CParser struct {
	parser   *sitter.Parser
	language *sitter.Language
}

func NewCParser() (*CParser, error) {
	lang := sitter.NewLanguage(tree_sitter_c.Language())
	parser := sitter.NewParser()
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("failed to set language for parser: %w", err)
	}
	return &CParser{
		parser:   parser,
		language: lang,
	}, nil
}

func (cp *CParser) Parser() *sitter.Parser {
	return cp.parser
}

func (cp *CParser) Language() string {
	return "C"
}

func (cp *CParser) SitterLanguage() *sitter.Language {
	return cp.language
}

func (cp *CParser) ShouldExcludeFile(filePath, projectRoot string) bool {
	lowerPath := strings.ToLower(filePath)
	
	// Exclude test files
	if strings.Contains(lowerPath, "test_") || strings.Contains(lowerPath, "_test.c") {
		return true
	}
	
	// Exclude test directories
	if strings.Contains(lowerPath, "/test/") || strings.Contains(lowerPath, "/tests/") {
		return true
	}

	// Exclude common C directories and files
	cExcludePatterns := []string{
		"build/",         // Build directory
		"dist/",          // Distribution directory
		"obj/",           // Object files directory
		".o",             // Object files
		".so",            // Shared libraries
		".a",             // Static libraries
		".dylib",         // macOS dynamic libraries
		".dll",           // Windows dynamic libraries
		"cmakefiles/",    // CMake build files
		".cmake/",        // CMake cache
		".git/",          // Git directory
	}

	for _, pattern := range cExcludePatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

func (cp *CParser) SupportedExtensions() []string {
	return []string{".c", ".h"}
}

func (cp *CParser) ParseFile(filePath, content string) ([]types.Symbol, error) {
	src := []byte(content)
	tree := cp.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse C file: tree-sitter returned nil")
	}
	defer tree.Close()

	// C doesn't have packages like Go/Java, use file name as module
	moduleName := cp.extractModuleName(filePath)

	queryText := `
	(function_definition) @decl
	(declaration) @decl
	(struct_specifier) @decl
	(union_specifier) @decl
	(enum_specifier) @decl
	(type_definition) @decl
	(preproc_def) @decl
	(preproc_function_def) @decl
	`

	q, err := sitter.NewQuery(cp.language, queryText)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	var symbols []types.Symbol

	matches := qc.Matches(q, tree.RootNode(), src)
	
	// Use a map to deduplicate symbols by name and line
	symbolMap := make(map[string]types.Symbol)

	for {
		m := matches.Next()
		if m == nil {
			break
		}
		for _, c := range m.Captures {
			declNode := c.Node
			nameNodes := cp.findNameNodes(&declNode)
			if len(nameNodes) == 0 {
				continue
			}

			startLine := int(declNode.StartPosition().Row) + 1
			endLine := int(declNode.EndPosition().Row) + 1
			
			// For single-line declarations, make sure end line equals start line
			if declNode.Kind() == "declaration" || declNode.Kind() == "preproc_def" || declNode.Kind() == "preproc_function_def" {
				// Preprocessor directives often include the newline, so force single line
				endLine = startLine
			}

			for _, nameNode := range nameNodes {
				name := strings.TrimSpace(nameNode.Utf8Text(src))
				if name == "" || cp.shouldSkipSymbol(name) {
					continue
				}

				// Create a unique key for deduplication
				key := fmt.Sprintf("%s:%d", name, startLine)
				
				// Only add if we haven't seen this symbol at this line before
				if _, exists := symbolMap[key]; !exists {
					symbolMap[key] = types.Symbol{
						Name:      name,
						Package:   moduleName,
						FilePath:  filePath,
						StartLine: startLine,
						EndLine:   endLine,
					}
				}
			}
		}
	}
	
	// Convert map to slice and sort by line number for consistent ordering
	for _, symbol := range symbolMap {
		symbols = append(symbols, symbol)
	}
	
	// Sort symbols by start line for consistent test results
	for i := 0; i < len(symbols)-1; i++ {
		for j := i + 1; j < len(symbols); j++ {
			if symbols[i].StartLine > symbols[j].StartLine {
				symbols[i], symbols[j] = symbols[j], symbols[i]
			}
		}
	}

	return symbols, nil
}

func (cp *CParser) shouldSkipSymbol(name string) bool {
	// Skip common C built-ins and system symbols
	commonBuiltins := map[string]bool{
		"printf": true, "scanf": true, "malloc": true, "free": true,
		"strlen": true, "strcpy": true, "strcmp": true, "strcat": true,
		"memcpy": true, "memset": true, "sizeof": true,
	}
	
	return commonBuiltins[name]
}

func (cp *CParser) findNameNodes(node *sitter.Node) []*sitter.Node {
	var names []*sitter.Node
	kind := node.Kind()

	switch kind {
	case "function_definition":
		// Look for function_declarator child
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && child.Kind() == "function_declarator" {
				if nameNode := cp.extractNameFromDeclarator(child); nameNode != nil {
					names = append(names, nameNode)
				}
			}
		}
		return names
		
	case "declaration":
		// Handle function declarations and variable declarations
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child == nil {
				continue
			}
			
			if child.Kind() == "function_declarator" {
				if nameNode := cp.extractNameFromDeclarator(child); nameNode != nil {
					names = append(names, nameNode)
				}
			} else if child.Kind() == "init_declarator" {
				// Variable declaration with initializer
				for j := uint(0); j < child.ChildCount(); j++ {
					grandchild := child.Child(j)
					if grandchild != nil && grandchild.Kind() == "identifier" {
						names = append(names, grandchild)
						break // Only take the first identifier (the variable name)
					}
				}
			} else if child.Kind() == "array_declarator" {
				// Array declaration
				for j := uint(0); j < child.ChildCount(); j++ {
					grandchild := child.Child(j)
					if grandchild != nil && grandchild.Kind() == "identifier" {
						names = append(names, grandchild)
						break // Only take the first identifier (the variable name)
					}
				}
			} else if child.Kind() == "identifier" {
				// Simple variable declaration
				names = append(names, child)
			}
		}
		return names
		
	case "struct_specifier", "union_specifier", "enum_specifier":
		// Look for type_identifier child (the name)
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && child.Kind() == "type_identifier" {
				names = append(names, child)
				break
			}
		}
		return names
		
	case "type_definition":
		// Look for the last type_identifier child (the typedef name)
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && child.Kind() == "type_identifier" {
				// Take the last type_identifier as it's the typedef name
				names = []*sitter.Node{child}
			}
		}
		return names
		
	case "preproc_def", "preproc_function_def":
		// Look for identifier child (the macro name)
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && child.Kind() == "identifier" {
				names = append(names, child)
				break
			}
		}
		return names
		
	case "identifier", "type_identifier":
		// Direct identifier node
		names = append(names, node)
		return names
	}

	return nil
}

func (cp *CParser) extractNameFromDeclarator(declarator *sitter.Node) *sitter.Node {
	// Function declarator structure: function_declarator -> identifier
	for i := uint(0); i < declarator.ChildCount(); i++ {
		child := declarator.Child(i)
		if child != nil && child.Kind() == "identifier" {
			return child
		}
	}
	return nil
}

func (cp *CParser) extractModuleName(filePath string) string {
	// Convert file path to module name
	// e.g., "src/utils/helper.c" -> "src.utils.helper"
	modulePath := strings.TrimSuffix(filePath, ".c")
	modulePath = strings.TrimSuffix(modulePath, ".h")
	return strings.ReplaceAll(modulePath, "/", ".")
}

func (cp *CParser) FindSymbolUsages(filePath, content, symbolName string) ([]types.SymbolUsage, error) {
	src := []byte(content)
	tree := cp.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse C file: tree-sitter returned nil")
	}
	defer tree.Close()

	var usages []types.SymbolUsage

	// Query for identifier nodes that could be symbol usages
	queryText := `
	(call_expression
		function: (identifier) @call)
	(call_expression
		function: (field_expression
			field: (field_identifier) @method_call))
	(identifier) @identifier
	`

	q, err := sitter.NewQuery(cp.language, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to create query: %w", err)
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, tree.RootNode(), src)

	processedLines := make(map[int]bool) // Avoid duplicate context for same line

	for {
		m := matches.Next()
		if m == nil {
			break
		}

		for _, c := range m.Captures {
			node := c.Node
			nodeText := strings.TrimSpace(node.Utf8Text(src))

			if nodeText == symbolName {
				lineNum := int(node.StartPosition().Row) + 1

				if processedLines[lineNum] {
					continue
				}

				if cp.isSymbolUsage(&node, src, symbolName) {
					context := cp.extractUsageContext(src, &node, symbolName)
					if context != "" {
						usages = append(usages, types.SymbolUsage{
							SymbolName: symbolName,
							FilePath:   filePath,
							LineNumber: lineNum,
							Context:    context,
						})
						processedLines[lineNum] = true
					}
				}
			}
		}
	}

	return usages, nil
}

// isSymbolUsage determines if an AST node represents a symbol usage (not definition)
func (cp *CParser) isSymbolUsage(node *sitter.Node, src []byte, symbolName string) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}

	parentKind := parent.Kind()

	// Skip if this is part of a function/struct/variable definition
	switch parentKind {
	case "function_definition", "function_declarator":
		return false
	case "declaration":
		// Check if this is part of a variable declaration
		return false
	case "struct_specifier", "union_specifier", "enum_specifier":
		return false
	case "typedef_declaration":
		return false
	case "preproc_def":
		return false
	}

	return true
}

// extractUsageContext extracts semantic context around a symbol usage
func (cp *CParser) extractUsageContext(src []byte, node *sitter.Node, symbolName string) string {
	// Find containing function and extract it entirely
	if containingFunc := cp.findContainingFunction(node); containingFunc != nil {
		return cp.formatFunctionContext(src, containingFunc, node, symbolName)
	}

	// If not in a function, extract smart line context
	return cp.extractLineContext(src, node, symbolName)
}

func (cp *CParser) findContainingFunction(node *sitter.Node) *sitter.Node {
	current := node
	for current != nil {
		kind := current.Kind()
		if kind == "function_definition" {
			return current
		}
		current = current.Parent()
	}
	return nil
}

func (cp *CParser) formatFunctionContext(src []byte, funcNode *sitter.Node, usageNode *sitter.Node, symbolName string) string {
	var builder strings.Builder

	// Add function signature
	signature := cp.extractFunctionSignature(src, funcNode)
	builder.WriteString(fmt.Sprintf("Function: %s\n", signature))
	builder.WriteString("Context:\n")

	// Add the entire function with usage highlighted
	funcText := funcNode.Utf8Text(src)
	lines := strings.Split(funcText, "\n")

	usageLine := int(usageNode.StartPosition().Row - funcNode.StartPosition().Row)

	for i, line := range lines {
		prefix := "  "
		if i == usageLine {
			prefix = "→ " // Highlight the usage line
		}
		builder.WriteString(fmt.Sprintf("%s%s\n", prefix, line))
	}

	return builder.String()
}

func (cp *CParser) extractFunctionSignature(src []byte, funcNode *sitter.Node) string {
	// Extract function declarator
	for i := uint(0); i < funcNode.ChildCount(); i++ {
		child := funcNode.Child(i)
		if child != nil && child.Kind() == "function_declarator" {
			// Get the function name
			nameNode := cp.extractNameFromDeclarator(child)
			if nameNode == nil {
				return "unknown"
			}
			
			name := nameNode.Utf8Text(src)
			
			// Get parameters
			params := ""
			for j := uint(0); j < child.ChildCount(); j++ {
				paramChild := child.Child(j)
				if paramChild != nil && paramChild.Kind() == "parameter_list" {
					params = paramChild.Utf8Text(src)
					break
				}
			}
			
			// Get return type (look for type specifier in parent)
			returnType := "void"
			for j := uint(0); j < funcNode.ChildCount(); j++ {
				typeChild := funcNode.Child(j)
				if typeChild != nil && (typeChild.Kind() == "primitive_type" || typeChild.Kind() == "type_identifier") {
					returnType = typeChild.Utf8Text(src)
					break
				}
			}
			
			return fmt.Sprintf("%s %s%s", returnType, name, params)
		}
	}
	
	return "unknown function"
}

func (cp *CParser) extractLineContext(src []byte, node *sitter.Node, symbolName string) string {
	// Simple line-based context with a few lines before and after
	lines := strings.Split(string(src), "\n")
	usageLine := int(node.StartPosition().Row)

	// Simple context: 3 lines before and after
	contextSize := 3
	start := usageLine - contextSize
	if start < 0 {
		start = 0
	}

	end := usageLine + contextSize
	if end >= len(lines) {
		end = len(lines) - 1
	}

	var builder strings.Builder
	builder.WriteString("Context:\n")

	for i := start; i <= end; i++ {
		prefix := "  "
		if i == usageLine {
			prefix = "→ "
		}
		builder.WriteString(fmt.Sprintf("%s%d: %s\n", prefix, i+1, lines[i]))
	}

	return builder.String()
}

func (cp *CParser) GetSymbolContext(filePath, content string, symbol types.Symbol) (string, error) {
	// Parse the file to get AST
	src := []byte(content)
	tree := cp.parser.Parse(src, nil)
	if tree == nil {
		return "", fmt.Errorf("failed to parse C file: tree-sitter returned nil")
	}
	defer tree.Close()

	// Find the symbol at the given location
	root := tree.RootNode()
	targetNode := cp.findNodeAtLocation(root, src, symbol.StartLine)
	if targetNode == nil {
		return "", fmt.Errorf("could not find symbol at line %d", symbol.StartLine)
	}

	// Use the same context extraction logic as for usages
	return cp.extractUsageContext(src, targetNode, symbol.Name), nil
}

func (cp *CParser) findNodeAtLocation(root *sitter.Node, src []byte, targetLine int) *sitter.Node {
	// Simple approach: find any node that contains the target line
	var findNode func(*sitter.Node) *sitter.Node
	findNode = func(node *sitter.Node) *sitter.Node {
		startLine := int(node.StartPosition().Row) + 1
		endLine := int(node.EndPosition().Row) + 1

		// If this node contains the target line
		if startLine <= targetLine && targetLine <= endLine {
			// Check children first (more specific)
			for i := uint(0); i < node.ChildCount(); i++ {
				child := node.Child(i)
				if child != nil {
					if result := findNode(child); result != nil {
						return result
					}
				}
			}
			// If no child contains it, return this node
			return node
		}
		return nil
	}

	return findNode(root)
}