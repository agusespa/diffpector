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
  ;; === Declarations (capture entire nodes, not just names) ===
  (function_declaration) @func_decl
  (method_declaration) @method_decl
  (type_spec) @type_decl
  (const_spec) @const_decl
  (var_spec) @var_decl
  (field_declaration) @field_decl
  (method_elem) @iface_method_decl
  (import_spec) @import_decl

  ;; === Usages (these can stay as they are) ===
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
			captureName := q.CaptureNames()[c.Index]
			startLine := int(c.Node.StartPosition().Row) + 1
			endLine := int(c.Node.EndPosition().Row) + 1

			var name string

			// For declarations, extract the name from the node structure
			if strings.HasSuffix(captureName, "_decl") {
				name = extractDeclarationName(c.Node, content, captureName)
			} else {
				// For usages, the captured node is already the name
				name = strings.TrimSpace(c.Node.Utf8Text(content))
			}

			if name == "" {
				continue
			}

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

func extractDeclarationName(node sitter.Node, content []byte, captureName string) string {
	switch captureName {
	case "func_decl":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return nameNode.Utf8Text(content)
		}
	case "method_decl":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return nameNode.Utf8Text(content)
		}
	case "type_decl":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return nameNode.Utf8Text(content)
		}
	case "const_decl", "var_decl":
		// These might have multiple names, just get the first one
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && (child.Kind() == "identifier" || child.Kind() == "type_identifier") {
				return child.Utf8Text(content)
			}
		}
	case "field_decl":
		nameNodes := findNameNodes(node)
		if len(nameNodes) > 0 {
			return nameNodes[0].Utf8Text(content)
		}
	case "import_decl":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && (child.Kind() == "interpreted_string_literal" || child.Kind() == "raw_string_literal") {
				return child.Utf8Text(content)
			}
		}
	}
	return ""
}

func findNameNodes(node sitter.Node) []*sitter.Node {
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
