package tools

import (
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

type GoParser struct {
	parser   *sitter.Parser
	language *sitter.Language
}

func NewGoParser() (*GoParser, error) {
	lang := sitter.NewLanguage(tree_sitter_go.Language())
	parser := sitter.NewParser()
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("failed to set language for parser: %w", err)
	}
	return &GoParser{
		parser:   parser,
		language: lang,
	}, nil
}

func (gp *GoParser) Parser() *sitter.Parser {
	return gp.parser
}

func (gp *GoParser) Language() string {
	return "Go"
}

func (gp *GoParser) SitterLanguage() *sitter.Language {
	return gp.language
}

func (gp *GoParser) FindSymbolUsages(filePath, content, symbolName string) ([]types.SymbolUsage, error) {
	src := []byte(content)
	tree := gp.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse Go file: tree-sitter returned nil")
	}
	defer tree.Close()

	var usages []types.SymbolUsage

	// Query for identifier nodes that could be symbol usages
	queryText := `
	(call_expression
		function: (identifier) @call)
	(call_expression
		function: (selector_expression
			field: (field_identifier) @method_call))
	(identifier) @identifier
	`

	q, err := sitter.NewQuery(gp.language, queryText)
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

				if gp.isSymbolUsage(&node, src, symbolName) {
					context := gp.extractUsageContext(src, &node, symbolName)
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
func (gp *GoParser) isSymbolUsage(node *sitter.Node, src []byte, symbolName string) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}

	parentKind := parent.Kind()

	// Skip if this is part of a function/method/type definition
	switch parentKind {
	case "function_declaration", "method_declaration", "type_declaration", "type_spec":
		return false
	case "field_declaration", "parameter_declaration", "var_declaration", "const_declaration":
		return false
	}

	// Check if parent is a function/method declaration by walking up
	current := parent
	for current != nil {
		kind := current.Kind()
		if kind == "function_declaration" || kind == "method_declaration" {
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
func (gp *GoParser) extractUsageContext(src []byte, node *sitter.Node, symbolName string) string {
	// Find containing function and extract it entirely
	if containingFunc := gp.findContainingFunction(node); containingFunc != nil {
		return gp.formatFunctionContext(src, containingFunc, node, symbolName)
	}

	// If not in a function, extract smart line context
	return gp.extractLineContext(src, node, symbolName)
}

func (gp *GoParser) findContainingFunction(node *sitter.Node) *sitter.Node {
	current := node
	for current != nil {
		kind := current.Kind()
		if kind == "function_declaration" || kind == "method_declaration" {
			return current
		}
		current = current.Parent()
	}
	return nil
}

func (gp *GoParser) formatFunctionContext(src []byte, funcNode *sitter.Node, usageNode *sitter.Node, symbolName string) string {
	var builder strings.Builder

	// Add function signature
	signature := gp.extractFunctionSignature(src, funcNode)
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

func (gp *GoParser) extractFunctionSignature(src []byte, funcNode *sitter.Node) string {
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
	resultNode := funcNode.ChildByFieldName("result")
	result := ""
	if resultNode != nil {
		result = " " + resultNode.Utf8Text(src)
	}

	// Check if it's a method (has receiver)
	receiverNode := funcNode.ChildByFieldName("receiver")
	if receiverNode != nil {
		receiver := receiverNode.Utf8Text(src)
		return fmt.Sprintf("func %s %s%s%s", receiver, name, params, result)
	}

	return fmt.Sprintf("func %s%s%s", name, params, result)
}

func (gp *GoParser) extractLineContext(src []byte, node *sitter.Node, symbolName string) string {
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

func (gp *GoParser) GetSymbolContext(filePath, content string, symbol types.Symbol) (string, error) {
	// Parse the file to get AST
	src := []byte(content)
	tree := gp.parser.Parse(src, nil)
	if tree == nil {
		return "", fmt.Errorf("failed to parse Go file: tree-sitter returned nil")
	}
	defer tree.Close()

	// Find the symbol at the given location
	root := tree.RootNode()
	targetNode := gp.findNodeAtLocation(root, src, symbol.StartLine)
	if targetNode == nil {
		return "", fmt.Errorf("could not find symbol at line %d", symbol.StartLine)
	}

	// Use the same context extraction logic as for usages
	return gp.extractUsageContext(src, targetNode, symbol.Name), nil
}

func (gp *GoParser) findNodeAtLocation(root *sitter.Node, src []byte, targetLine int) *sitter.Node {
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

func (gp *GoParser) SupportedExtensions() []string {
	return []string{`.go`}
}

func (gp *GoParser) ParseFile(filePath, content string) ([]types.Symbol, error) {
	src := []byte(content)
	tree := gp.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse Go file: tree-sitter returned nil")
	}
	defer tree.Close()

	packageName := gp.extractPackageName(tree.RootNode(), src)

	queryText := `
	(source_file
		(function_declaration) @decl)
	(source_file
		(method_declaration) @decl)
	(source_file
		(type_declaration (type_spec) @decl))
	(source_file
		(const_declaration (const_spec) @decl))
	(source_file
		(var_declaration (var_spec) @decl))
	(source_file
		(var_declaration (var_spec_list (var_spec) @decl)))
	(source_file
		(import_declaration (import_spec) @decl))
	(source_file
		(import_declaration (import_spec_list (import_spec) @decl)))
	(source_file
		(type_declaration (type_spec (struct_type (field_declaration_list (field_declaration) @decl)))))
	(source_file
		(type_declaration (type_spec (interface_type (method_elem) @decl))))
`

	q, err := sitter.NewQuery(gp.language, queryText)
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
			nameNodes := findNameNodes(&declNode)
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

func findNameNodes(node *sitter.Node) []*sitter.Node {
	var names []*sitter.Node
	kind := node.Kind()

	// Direct identifier children for functions and methods
	if kind == "function_declaration" || kind == "method_declaration" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			names = append(names, nameNode)
		}
		return names
	}

	// Handle import specs
	if kind == "import_spec" {
		// For imports, we want the package path (string literal)
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && (child.Kind() == "interpreted_string_literal" || child.Kind() == "raw_string_literal") {
				names = append(names, child)
				return names
			}
		}
		return names
	}

	// Handle field declarations (struct fields)
	if kind == "field_declaration" {
		// Get all field names in this declaration
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && child.Kind() == "field_identifier_list" {
				// Multiple field names in one declaration
				for j := uint(0); j < child.ChildCount(); j++ {
					grandchild := child.Child(j)
					if grandchild != nil && grandchild.Kind() == "field_identifier" {
						names = append(names, grandchild)
					}
				}
				return names
			} else if child != nil && child.Kind() == "field_identifier" {
				// Single field name
				names = append(names, child)
				return names
			}
		}
		return names
	}

	// Handle method elements (interface methods)
	if kind == "method_elem" {
		// The first child should be the field_identifier (method name)
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && child.Kind() == "field_identifier" {
				names = append(names, child)
				return names
			}
		}
		return names
	}

	// For type_spec, const_spec, and var_spec
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		childKind := child.Kind()
		// Only capture the first identifier, which is the name
		if childKind == "identifier" || childKind == "type_identifier" {
			names = append(names, child)
			// Return immediately after finding the first name for these cases
			// to avoid picking up the type
			return names
		}
	}

	return nil
}

func (gp *GoParser) extractPackageName(rootNode *sitter.Node, sourceBytes []byte) string {
	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child != nil && child.Kind() == "package_clause" {
			for j := uint(0); j < child.ChildCount(); j++ {
				grandchild := child.Child(j)
				if grandchild != nil && grandchild.Kind() == "package_identifier" {
					return grandchild.Utf8Text(sourceBytes)
				}
			}
		}
	}
	return "main"
}
