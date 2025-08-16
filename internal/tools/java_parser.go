package tools

import (
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

type JavaParser struct {
	parser   *sitter.Parser
	language *sitter.Language
}

func NewJavaParser() (*JavaParser, error) {
	lang := sitter.NewLanguage(tree_sitter_java.Language())
	parser := sitter.NewParser()
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("failed to set language for parser: %w", err)
	}
	return &JavaParser{
		parser:   parser,
		language: lang,
	}, nil
}

func (jp *JavaParser) Parser() *sitter.Parser {
	return jp.parser
}

func (jp *JavaParser) Language() string {
	return "Java"
}

func (jp *JavaParser) SitterLanguage() *sitter.Language {
	return jp.language
}

func (jp *JavaParser) ShouldExcludeFile(filePath, projectRoot string) bool {
	lowerPath := strings.ToLower(filePath)
	
	// Exclude test files
	if strings.Contains(lowerPath, "test.java") || strings.Contains(lowerPath, "tests.java") {
		return true
	}
	
	// Exclude test directories
	if strings.Contains(lowerPath, "/test/") || strings.Contains(lowerPath, "/tests/") {
		return true
	}

	// Exclude common Java directories and files
	javaExcludePatterns := []string{
		"target/",        // Maven build directory
		"build/",         // Gradle build directory
		".gradle/",       // Gradle cache
		"bin/",           // Eclipse build directory
		"out/",           // IntelliJ build directory
		".git/",
		".class",         // Compiled class files
		"generated/",     // Generated source files
	}

	for _, pattern := range javaExcludePatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

func (jp *JavaParser) SupportedExtensions() []string {
	return []string{".java"}
}

func (jp *JavaParser) ParseFile(filePath, content string) ([]types.Symbol, error) {
	src := []byte(content)
	tree := jp.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse Java file: tree-sitter returned nil")
	}
	defer tree.Close()

	// Extract package name
	packageName := jp.extractPackageName(tree.RootNode(), src)

	queryText := `
	(class_declaration) @decl
	(interface_declaration) @decl
	(enum_declaration) @decl
	(annotation_type_declaration) @decl
	(method_declaration) @decl
	(constructor_declaration) @decl
	(field_declaration (variable_declarator) @decl)
	(constant_declaration (variable_declarator) @decl)
	(enum_constant) @decl
	(annotation_type_element_declaration) @decl
	`

	q, err := sitter.NewQuery(jp.language, queryText)
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
			nameNodes := jp.findNameNodes(&declNode)
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
					Package:   packageName,
					FilePath:  filePath,
					StartLine: startLine,
					EndLine:   endLine,
				})
			}
		}
	}

	return symbols, nil
}

func (jp *JavaParser) findNameNodes(node *sitter.Node) []*sitter.Node {
	var names []*sitter.Node
	kind := node.Kind()

	// Direct identifier children for classes, interfaces, enums, methods
	switch kind {
	case "class_declaration", "interface_declaration", "enum_declaration", 
		 "annotation_type_declaration", "method_declaration", "constructor_declaration",
		 "annotation_type_element_declaration":
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
	
	case "enum_constant":
		// For enum constants, the name is usually the first child
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && child.Kind() == "identifier" {
				names = append(names, child)
				return names
			}
		}
		return names
	}

	return nil
}

func (jp *JavaParser) extractPackageName(rootNode *sitter.Node, sourceBytes []byte) string {
	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child != nil && child.Kind() == "package_declaration" {
			for j := uint(0); j < child.ChildCount(); j++ {
				grandchild := child.Child(j)
				if grandchild != nil && (grandchild.Kind() == "scoped_identifier" || grandchild.Kind() == "identifier") {
					return grandchild.Utf8Text(sourceBytes)
				}
			}
		}
	}
	return "default"
}

func (jp *JavaParser) FindSymbolUsages(filePath, content, symbolName string) ([]types.SymbolUsage, error) {
	src := []byte(content)
	tree := jp.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse Java file: tree-sitter returned nil")
	}
	defer tree.Close()

	var usages []types.SymbolUsage

	// Query for identifier nodes that could be symbol usages
	queryText := `
	(method_invocation
		name: (identifier) @call)
	(method_invocation
		object: (identifier) @object_call)
	(field_access
		field: (identifier) @field_access)
	(identifier) @identifier
	`

	q, err := sitter.NewQuery(jp.language, queryText)
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

				if jp.isSymbolUsage(&node, src, symbolName) {
					context := jp.extractUsageContext(src, &node, symbolName)
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
func (jp *JavaParser) isSymbolUsage(node *sitter.Node, src []byte, symbolName string) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}

	parentKind := parent.Kind()

	// Skip if this is part of a class/interface/method/field definition
	switch parentKind {
	case "class_declaration", "interface_declaration", "enum_declaration",
		 "method_declaration", "constructor_declaration", "annotation_type_declaration":
		return false
	case "field_declaration", "constant_declaration", "formal_parameter", 
		 "catch_formal_parameter", "enhanced_for_statement":
		return false
	case "variable_declarator":
		return false
	}

	// Check if parent is a method/constructor declaration by walking up
	current := parent
	for current != nil {
		kind := current.Kind()
		if kind == "method_declaration" || kind == "constructor_declaration" {
			// Check if our node is the method name (definition)
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
func (jp *JavaParser) extractUsageContext(src []byte, node *sitter.Node, symbolName string) string {
	// Find containing method and extract it entirely
	if containingMethod := jp.findContainingMethod(node); containingMethod != nil {
		return jp.formatMethodContext(src, containingMethod, node, symbolName)
	}

	// If not in a method, extract smart line context
	return jp.extractLineContext(src, node, symbolName)
}

func (jp *JavaParser) findContainingMethod(node *sitter.Node) *sitter.Node {
	current := node
	for current != nil {
		kind := current.Kind()
		if kind == "method_declaration" || kind == "constructor_declaration" {
			return current
		}
		current = current.Parent()
	}
	return nil
}

func (jp *JavaParser) formatMethodContext(src []byte, methodNode *sitter.Node, usageNode *sitter.Node, symbolName string) string {
	var builder strings.Builder

	// Add method signature
	signature := jp.extractMethodSignature(src, methodNode)
	builder.WriteString(fmt.Sprintf("Method: %s\n", signature))
	builder.WriteString("Context:\n")

	// Add the entire method with usage highlighted
	methodText := methodNode.Utf8Text(src)
	lines := strings.Split(methodText, "\n")

	usageLine := int(usageNode.StartPosition().Row - methodNode.StartPosition().Row)

	for i, line := range lines {
		prefix := "  "
		if i == usageLine {
			prefix = "→ " // Highlight the usage line
		}
		builder.WriteString(fmt.Sprintf("%s%s\n", prefix, line))
	}

	return builder.String()
}

func (jp *JavaParser) extractMethodSignature(src []byte, methodNode *sitter.Node) string {
	kind := methodNode.Kind()
	
	switch kind {
	case "method_declaration":
		// Extract method name
		nameNode := methodNode.ChildByFieldName("name")
		if nameNode == nil {
			return "unknown"
		}
		name := nameNode.Utf8Text(src)
		
		// Extract parameters
		paramsNode := methodNode.ChildByFieldName("parameters")
		params := ""
		if paramsNode != nil {
			params = paramsNode.Utf8Text(src)
		}
		
		// Extract return type
		typeNode := methodNode.ChildByFieldName("type")
		returnType := ""
		if typeNode != nil {
			returnType = typeNode.Utf8Text(src) + " "
		}
		
		// Extract modifiers
		modifiers := jp.extractModifiers(src, methodNode)
		
		return fmt.Sprintf("%s%s%s%s", modifiers, returnType, name, params)
		
	case "constructor_declaration":
		// Extract constructor name
		nameNode := methodNode.ChildByFieldName("name")
		if nameNode == nil {
			return "unknown constructor"
		}
		name := nameNode.Utf8Text(src)
		
		// Extract parameters
		paramsNode := methodNode.ChildByFieldName("parameters")
		params := ""
		if paramsNode != nil {
			params = paramsNode.Utf8Text(src)
		}
		
		// Extract modifiers
		modifiers := jp.extractModifiers(src, methodNode)
		
		return fmt.Sprintf("%s%s%s", modifiers, name, params)
		
	default:
		return "method"
	}
}

func (jp *JavaParser) extractModifiers(src []byte, node *sitter.Node) string {
	var modifiers []string
	
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "modifiers" {
			for j := uint(0); j < child.ChildCount(); j++ {
				modifier := child.Child(j)
				if modifier != nil {
					modifiers = append(modifiers, modifier.Utf8Text(src))
				}
			}
			break
		}
	}
	
	if len(modifiers) > 0 {
		return strings.Join(modifiers, " ") + " "
	}
	return ""
}

func (jp *JavaParser) extractLineContext(src []byte, node *sitter.Node, symbolName string) string {
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

func (jp *JavaParser) GetSymbolContext(filePath, content string, symbol types.Symbol) (string, error) {
	// Parse the file to get AST
	src := []byte(content)
	tree := jp.parser.Parse(src, nil)
	if tree == nil {
		return "", fmt.Errorf("failed to parse Java file: tree-sitter returned nil")
	}
	defer tree.Close()

	// Find the symbol at the given location
	root := tree.RootNode()
	targetNode := jp.findNodeAtLocation(root, src, symbol.StartLine)
	if targetNode == nil {
		return "", fmt.Errorf("could not find symbol at line %d", symbol.StartLine)
	}

	// Use the same context extraction logic as for usages
	return jp.extractUsageContext(src, targetNode, symbol.Name), nil
}

func (jp *JavaParser) findNodeAtLocation(root *sitter.Node, src []byte, targetLine int) *sitter.Node {
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