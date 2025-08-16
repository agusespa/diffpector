package tools

import (
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type TypeScriptParser struct {
	parser   *sitter.Parser
	language *sitter.Language
}

func NewTypeScriptParser() (*TypeScriptParser, error) {
	lang := sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
	parser := sitter.NewParser()
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("failed to set language for parser: %w", err)
	}
	return &TypeScriptParser{
		parser:   parser,
		language: lang,
	}, nil
}

func (tp *TypeScriptParser) Parser() *sitter.Parser {
	return tp.parser
}

func (tp *TypeScriptParser) Language() string {
	return "TypeScript"
}

func (tp *TypeScriptParser) SitterLanguage() *sitter.Language {
	return tp.language
}

func (tp *TypeScriptParser) ShouldExcludeFile(filePath, projectRoot string) bool {
	lowerPath := strings.ToLower(filePath)
	
	// Exclude test files
	if strings.Contains(lowerPath, ".test.") || strings.Contains(lowerPath, ".spec.") {
		return true
	}

	// Exclude common TypeScript/JavaScript directories and files
	tsExcludePatterns := []string{
		"node_modules/",
		"dist/",
		"build/",
		".next/",
		"coverage/",
		".git/",
		".d.ts", // Type definition files
	}

	for _, pattern := range tsExcludePatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

func (tp *TypeScriptParser) SupportedExtensions() []string {
	return []string{".ts", ".tsx"}
}

func (tp *TypeScriptParser) ParseFile(filePath, content string) ([]types.Symbol, error) {
	src := []byte(content)
	tree := tp.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse TypeScript file: tree-sitter returned nil")
	}
	defer tree.Close()

	// TypeScript doesn't have packages like Go, so we'll use the module name or file name
	moduleName := tp.extractModuleName(filePath)

	queryText := `
	(function_declaration) @decl
	(class_declaration) @decl
	(interface_declaration) @decl
	(type_alias_declaration) @decl
	(enum_declaration) @decl
	(variable_declaration (variable_declarator) @decl)
	(lexical_declaration (variable_declarator) @decl)
	(export_statement (function_declaration) @decl)
	(export_statement (class_declaration) @decl)
	(export_statement (interface_declaration) @decl)
	(export_statement (type_alias_declaration) @decl)
	(export_statement (enum_declaration) @decl)
	(export_statement (variable_declaration (variable_declarator) @decl))
	(export_statement (lexical_declaration (variable_declarator) @decl))
	(method_definition) @decl
	(public_field_definition) @decl
	(method_signature) @decl
	(property_signature) @decl
	`

	q, err := sitter.NewQuery(tp.language, queryText)
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
			nameNodes := tp.findNameNodes(&declNode)
			if len(nameNodes) == 0 {
				continue
			}

			startLine := int(declNode.StartPosition().Row) + 1
			endLine := int(declNode.EndPosition().Row) + 1

			for _, nameNode := range nameNodes {
				name := strings.TrimSpace(nameNode.Utf8Text(src))
				if name == "" {
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

func (tp *TypeScriptParser) findNameNodes(node *sitter.Node) []*sitter.Node {
	var names []*sitter.Node
	kind := node.Kind()

	// Direct identifier children for functions, classes, interfaces, types, enums
	switch kind {
	case "function_declaration", "class_declaration", "interface_declaration", 
		 "type_alias_declaration", "enum_declaration":
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			names = append(names, nameNode)
		}
		return names
	
	case "method_definition", "method_signature":
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			names = append(names, nameNode)
		}
		return names
	
	case "public_field_definition", "property_signature":
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			names = append(names, nameNode)
		}
		return names
	
	case "variable_declarator":
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			names = append(names, nameNode)
		}
		return names
	}

	return nil
}

func (tp *TypeScriptParser) extractModuleName(filePath string) string {
	// Extract module name from file path
	// For TypeScript, we'll use the file name without extension as the module name
	parts := strings.Split(filePath, "/")
	if len(parts) == 0 {
		return "unknown"
	}
	
	fileName := parts[len(parts)-1]
	// Remove extension
	if idx := strings.LastIndex(fileName, "."); idx != -1 {
		fileName = fileName[:idx]
	}
	
	return fileName
}

func (tp *TypeScriptParser) FindSymbolUsages(filePath, content, symbolName string) ([]types.SymbolUsage, error) {
	src := []byte(content)
	tree := tp.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse TypeScript file: tree-sitter returned nil")
	}
	defer tree.Close()

	var usages []types.SymbolUsage

	// Query for identifier nodes that could be symbol usages
	queryText := `
	(call_expression
		function: (identifier) @call)
	(call_expression
		function: (member_expression
			property: (property_identifier) @method_call))
	(identifier) @identifier
	(property_identifier) @property
	`

	q, err := sitter.NewQuery(tp.language, queryText)
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

				if tp.isSymbolUsage(&node, src, symbolName) {
					context := tp.extractUsageContext(src, &node, symbolName)
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
func (tp *TypeScriptParser) isSymbolUsage(node *sitter.Node, src []byte, symbolName string) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}

	parentKind := parent.Kind()

	// Skip if this is part of a function/class/interface/type definition
	switch parentKind {
	case "function_declaration", "method_definition", "class_declaration", 
		 "interface_declaration", "type_alias_declaration", "enum_declaration":
		return false
	case "variable_declarator", "formal_parameters", "required_parameter", "optional_parameter":
		return false
	case "property_signature", "method_signature":
		return false
	}

	// Check if parent is a function/method declaration by walking up
	current := parent
	for current != nil {
		kind := current.Kind()
		if kind == "function_declaration" || kind == "method_definition" {
			// Check if our node is the function name (definition)
			nameNode := current.ChildByFieldName("name")
			if nameNode != nil && nameNode.StartByte() == node.StartByte() && nameNode.EndByte() == node.EndByte() {
				return false
			}
			break
		}
		current = current.Parent()
	}

	return true
}

// extractUsageContext extracts semantic context around a symbol usage
func (tp *TypeScriptParser) extractUsageContext(src []byte, node *sitter.Node, symbolName string) string {
	// Find containing function and extract it entirely
	if containingFunc := tp.findContainingFunction(node); containingFunc != nil {
		return tp.formatFunctionContext(src, containingFunc, node, symbolName)
	}

	// If not in a function, extract smart line context
	return tp.extractLineContext(src, node, symbolName)
}

func (tp *TypeScriptParser) findContainingFunction(node *sitter.Node) *sitter.Node {
	current := node
	for current != nil {
		kind := current.Kind()
		if kind == "function_declaration" || kind == "method_definition" || 
		   kind == "arrow_function" || kind == "function_expression" {
			return current
		}
		current = current.Parent()
	}
	return nil
}

func (tp *TypeScriptParser) formatFunctionContext(src []byte, funcNode *sitter.Node, usageNode *sitter.Node, symbolName string) string {
	var builder strings.Builder

	// Add function signature
	signature := tp.extractFunctionSignature(src, funcNode)
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

func (tp *TypeScriptParser) extractFunctionSignature(src []byte, funcNode *sitter.Node) string {
	kind := funcNode.Kind()
	
	switch kind {
	case "function_declaration":
		// Extract function name
		nameNode := funcNode.ChildByFieldName("name")
		if nameNode == nil {
			return "unknown"
		}
		name := nameNode.Utf8Text(src)
		
		// Extract parameters
		paramsNode := funcNode.ChildByFieldName("parameters")
		params := ""
		if paramsNode != nil {
			params = paramsNode.Utf8Text(src)
		}
		
		// Extract return type
		returnTypeNode := funcNode.ChildByFieldName("return_type")
		returnType := ""
		if returnTypeNode != nil {
			returnType = ": " + returnTypeNode.Utf8Text(src)
		}
		
		return fmt.Sprintf("function %s%s%s", name, params, returnType)
		
	case "method_definition":
		// Extract method name
		nameNode := funcNode.ChildByFieldName("name")
		if nameNode == nil {
			return "unknown method"
		}
		name := nameNode.Utf8Text(src)
		
		// Extract parameters
		paramsNode := funcNode.ChildByFieldName("parameters")
		params := ""
		if paramsNode != nil {
			params = paramsNode.Utf8Text(src)
		}
		
		// Extract return type
		returnTypeNode := funcNode.ChildByFieldName("return_type")
		returnType := ""
		if returnTypeNode != nil {
			returnType = ": " + returnTypeNode.Utf8Text(src)
		}
		
		return fmt.Sprintf("method %s%s%s", name, params, returnType)
		
	case "arrow_function":
		// Extract parameters
		paramsNode := funcNode.ChildByFieldName("parameters")
		if paramsNode == nil {
			paramsNode = funcNode.ChildByFieldName("parameter")
		}
		params := ""
		if paramsNode != nil {
			params = paramsNode.Utf8Text(src)
		}
		
		// Extract return type
		returnTypeNode := funcNode.ChildByFieldName("return_type")
		returnType := ""
		if returnTypeNode != nil {
			returnType = ": " + returnTypeNode.Utf8Text(src)
		}
		
		return fmt.Sprintf("arrow function %s%s", params, returnType)
		
	default:
		return "function"
	}
}

func (tp *TypeScriptParser) extractLineContext(src []byte, node *sitter.Node, symbolName string) string {
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

func (tp *TypeScriptParser) GetSymbolContext(filePath, content string, symbol types.Symbol) (string, error) {
	// Parse the file to get AST
	src := []byte(content)
	tree := tp.parser.Parse(src, nil)
	if tree == nil {
		return "", fmt.Errorf("failed to parse TypeScript file: tree-sitter returned nil")
	}
	defer tree.Close()

	// Find the symbol at the given location
	root := tree.RootNode()
	targetNode := tp.findNodeAtLocation(root, src, symbol.StartLine)
	if targetNode == nil {
		return "", fmt.Errorf("could not find symbol at line %d", symbol.StartLine)
	}

	// Use the same context extraction logic as for usages
	return tp.extractUsageContext(src, targetNode, symbol.Name), nil
}

func (tp *TypeScriptParser) findNodeAtLocation(root *sitter.Node, src []byte, targetLine int) *sitter.Node {
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