package tools

import (
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

type PythonParser struct {
	parser   *sitter.Parser
	language *sitter.Language
}

func NewPythonParser() (*PythonParser, error) {
	lang := sitter.NewLanguage(tree_sitter_python.Language())
	parser := sitter.NewParser()
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("failed to set language for parser: %w", err)
	}
	return &PythonParser{
		parser:   parser,
		language: lang,
	}, nil
}

func (pp *PythonParser) Parser() *sitter.Parser {
	return pp.parser
}

func (pp *PythonParser) Language() string {
	return "Python"
}

func (pp *PythonParser) SitterLanguage() *sitter.Language {
	return pp.language
}

func (pp *PythonParser) ShouldExcludeFile(filePath, projectRoot string) bool {
	lowerPath := strings.ToLower(filePath)
	
	// Exclude test files
	if strings.Contains(lowerPath, "test_") || strings.Contains(lowerPath, "_test.py") {
		return true
	}
	
	// Exclude test directories
	if strings.Contains(lowerPath, "/test/") || strings.Contains(lowerPath, "/tests/") {
		return true
	}

	// Exclude common Python directories and files
	pythonExcludePatterns := []string{
		"__pycache__/",   // Python bytecode cache
		".pytest_cache/", // Pytest cache
		"venv/",          // Virtual environment
		"env/",           // Virtual environment
		".venv/",         // Virtual environment
		"site-packages/", // Installed packages
		"build/",         // Build directory
		"dist/",          // Distribution directory
		".git/",
		".pyc",           // Compiled Python files
		".pyo",           // Optimized Python files
		".pyd",           // Python extension modules
		"migrations/",    // Django migrations
		"__init__.py",    // Often just imports, less useful for context
	}

	for _, pattern := range pythonExcludePatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

func (pp *PythonParser) SupportedExtensions() []string {
	return []string{".py", ".pyw"}
}

func (pp *PythonParser) ParseFile(filePath, content string) ([]types.Symbol, error) {
	src := []byte(content)
	tree := pp.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse Python file: tree-sitter returned nil")
	}
	defer tree.Close()

	// Python doesn't have packages like Go/Java, use module name from file path
	moduleName := pp.extractModuleName(filePath)

	queryText := `
	(function_definition) @decl
	(class_definition) @decl
	(assignment
		left: (identifier) @decl)
	(assignment
		left: (pattern_list (identifier) @decl))
	`

	q, err := sitter.NewQuery(pp.language, queryText)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	var symbols []types.Symbol

	matches := qc.Matches(q, tree.RootNode(), src)

	for {
		m := matches.Next()
		if m == nil {
			break
		}
		for _, c := range m.Captures {
			declNode := c.Node
			nameNodes := pp.findNameNodes(&declNode)
			if len(nameNodes) == 0 {
				continue
			}

			startLine := int(declNode.StartPosition().Row) + 1
			endLine := int(declNode.EndPosition().Row) + 1

			for _, nameNode := range nameNodes {
				name := strings.TrimSpace(nameNode.Utf8Text(src))
				if name == "" || pp.shouldSkipSymbol(name) {
					continue
				}

				symbols = append(symbols, types.Symbol{
					Name:      name,
					Package:   moduleName,
					FilePath:  filePath,
					StartLine: startLine,
					EndLine:   endLine,
				})
			}
		}
	}

	return symbols, nil
}

func (pp *PythonParser) shouldSkipSymbol(name string) bool {
	// Skip common Python built-ins and private symbols
	if strings.HasPrefix(name, "_") && !strings.HasPrefix(name, "__") {
		return true
	}
	
	// Skip very common imports that are less useful for context
	commonImports := map[string]bool{
		"os": true, "sys": true, "json": true, "re": true,
		"time": true, "datetime": true, "typing": true,
	}
	
	return commonImports[name]
}

func (pp *PythonParser) findNameNodes(node *sitter.Node) []*sitter.Node {
	var names []*sitter.Node
	kind := node.Kind()

	switch kind {
	case "function_definition", "class_definition":
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			names = append(names, nameNode)
		}
		return names
	
	case "identifier":
		// Direct identifier node
		names = append(names, node)
		return names
		
	case "dotted_name":
		// For imports like "package.module", we want the full dotted name
		names = append(names, node)
		return names
	}

	return nil
}

func (pp *PythonParser) extractModuleName(filePath string) string {
	// Convert file path to Python module name
	// e.g., "src/utils/helper.py" -> "src.utils.helper"
	modulePath := strings.TrimSuffix(filePath, ".py")
	modulePath = strings.TrimSuffix(modulePath, ".pyw")
	return strings.ReplaceAll(modulePath, "/", ".")
}

func (pp *PythonParser) FindSymbolUsages(filePath, content, symbolName string) ([]types.SymbolUsage, error) {
	src := []byte(content)
	tree := pp.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse Python file: tree-sitter returned nil")
	}
	defer tree.Close()

	var usages []types.SymbolUsage

	// Query for identifier nodes that could be symbol usages
	queryText := `
	(call
		function: (identifier) @call)
	(call
		function: (attribute
			object: (identifier) @object_call))
	(attribute
		attribute: (identifier) @attr_access)
	(identifier) @identifier
	`

	q, err := sitter.NewQuery(pp.language, queryText)
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

				if pp.isSymbolUsage(&node, src, symbolName) {
					context := pp.extractUsageContext(src, &node, symbolName)
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
func (pp *PythonParser) isSymbolUsage(node *sitter.Node, src []byte, symbolName string) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}

	parentKind := parent.Kind()

	// Skip if this is part of a function/class/variable definition
	switch parentKind {
	case "function_definition", "class_definition":
		return false
	case "assignment":
		// Check if this is the left side of assignment (definition)
		leftNode := parent.ChildByFieldName("left")
		if leftNode != nil && pp.nodeContains(leftNode, node) {
			return false
		}
	case "parameters", "default_parameter", "typed_parameter":
		return false
	case "for_statement":
		// Check if this is the loop variable
		leftNode := parent.ChildByFieldName("left")
		if leftNode != nil && pp.nodeContains(leftNode, node) {
			return false
		}
	}

	return true
}

func (pp *PythonParser) nodeContains(parent *sitter.Node, child *sitter.Node) bool {
	return parent.StartByte() <= child.StartByte() && child.EndByte() <= parent.EndByte()
}

// extractUsageContext extracts semantic context around a symbol usage
func (pp *PythonParser) extractUsageContext(src []byte, node *sitter.Node, symbolName string) string {
	// Find containing function/class and extract it entirely
	if containingFunc := pp.findContainingFunction(node); containingFunc != nil {
		return pp.formatFunctionContext(src, containingFunc, node, symbolName)
	}

	// If not in a function, extract smart line context
	return pp.extractLineContext(src, node, symbolName)
}

func (pp *PythonParser) findContainingFunction(node *sitter.Node) *sitter.Node {
	current := node
	for current != nil {
		kind := current.Kind()
		if kind == "function_definition" || kind == "class_definition" {
			return current
		}
		current = current.Parent()
	}
	return nil
}

func (pp *PythonParser) formatFunctionContext(src []byte, funcNode *sitter.Node, usageNode *sitter.Node, symbolName string) string {
	var builder strings.Builder

	// Add function/class signature
	signature := pp.extractSignature(src, funcNode)
	kind := "Function"
	if funcNode.Kind() == "class_definition" {
		kind = "Class"
	}
	builder.WriteString(fmt.Sprintf("%s: %s\n", kind, signature))
	builder.WriteString("Context:\n")

	// Add the entire function/class with usage highlighted
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

func (pp *PythonParser) extractSignature(src []byte, node *sitter.Node) string {
	kind := node.Kind()
	
	switch kind {
	case "function_definition":
		// Extract function name
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return "unknown"
		}
		name := nameNode.Utf8Text(src)
		
		// Extract parameters
		paramsNode := node.ChildByFieldName("parameters")
		params := ""
		if paramsNode != nil {
			params = paramsNode.Utf8Text(src)
		}
		
		// Extract return type annotation if present
		returnType := ""
		if returnNode := node.ChildByFieldName("return_type"); returnNode != nil {
			returnType = " -> " + returnNode.Utf8Text(src)
		}
		
		return fmt.Sprintf("def %s%s%s", name, params, returnType)
		
	case "class_definition":
		// Extract class name
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return "unknown class"
		}
		name := nameNode.Utf8Text(src)
		
		// Extract superclasses if present
		superclasses := ""
		if superNode := node.ChildByFieldName("superclasses"); superNode != nil {
			superclasses = superNode.Utf8Text(src)
		}
		
		return fmt.Sprintf("class %s%s", name, superclasses)
		
	default:
		return "definition"
	}
}

func (pp *PythonParser) extractLineContext(src []byte, node *sitter.Node, symbolName string) string {
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

func (pp *PythonParser) GetSymbolContext(filePath, content string, symbol types.Symbol) (string, error) {
	// Parse the file to get AST
	src := []byte(content)
	tree := pp.parser.Parse(src, nil)
	if tree == nil {
		return "", fmt.Errorf("failed to parse Python file: tree-sitter returned nil")
	}
	defer tree.Close()

	// Find the symbol at the given location
	root := tree.RootNode()
	targetNode := pp.findNodeAtLocation(root, src, symbol.StartLine)
	if targetNode == nil {
		return "", fmt.Errorf("could not find symbol at line %d", symbol.StartLine)
	}

	// Use the same context extraction logic as for usages
	return pp.extractUsageContext(src, targetNode, symbol.Name), nil
}

func (pp *PythonParser) findNodeAtLocation(root *sitter.Node, src []byte, targetLine int) *sitter.Node {
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