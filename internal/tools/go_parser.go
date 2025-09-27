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

func (gp *GoParser) SupportedExtensions() []string {
	return []string{`.go`}
}

func (gp *GoParser) ShouldExcludeFile(filePath, projectRoot string) bool {
	lowerPath := strings.ToLower(filePath)

	if strings.HasSuffix(lowerPath, "_test.go") {
		return true
	}

	goExcludePatterns := []string{
		"vendor/",
		"testdata/",
		".git/",
	}

	for _, pattern := range goExcludePatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

func (gp *GoParser) ParseFile(filePath string, content []byte) ([]types.Symbol, error) {
	tree := gp.parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse Go file")
	}
	defer tree.Close()

	packageName := gp.extractPackageName(tree.RootNode(), content)

	queryText := `
	[
	  ;; === Declarations ===
	  (function_declaration name: (identifier) @func_decl)
	  (method_declaration name: (field_identifier) @method_decl)
	  (type_spec name: (type_identifier) @type_decl)
	  (const_spec name: (identifier) @const_decl)
	  (var_spec name: (identifier) @var_decl)
	  (field_declaration name: (field_identifier) @field_decl)
	  (method_elem name: (field_identifier) @iface_method_decl)
	  (import_spec path: (interpreted_string_literal) @import_path)

	  ;; === Usages ===
	  (call_expression function: (identifier) @func_usage)
	  (call_expression function: (selector_expression field: (field_identifier) @method_usage))
	  (selector_expression field: (field_identifier) @field_usage)
	  (identifier) @var_usage
	]
	`

	q, err := sitter.NewQuery(gp.language, queryText)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, tree.RootNode(), content)

	var symbols []types.Symbol

	for {
		m := matches.Next()
		if m == nil {
			break
		}
		for _, c := range m.Captures {
			name := strings.TrimSpace(c.Node.Utf8Text(content))
			if name == "" {
				continue
			}

			startLine := int(c.Node.StartPosition().Row) + 1
			endLine := int(c.Node.EndPosition().Row) + 1
			captureName := q.CaptureNames()[c.Index]

			symbols = append(symbols, types.Symbol{
				Name:      name,
				Type:      captureName,
				Package:   packageName,
				FilePath:  filePath,
				StartLine: startLine,
				EndLine:   endLine,
			})
		}
	}

	return symbols, nil
}

// TODO review code after this line

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

func (gp *GoParser) GetSymbolContext(filePath string, symbol types.Symbol, content []byte) (string, error) {
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
