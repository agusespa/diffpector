package tools

import (
	"fmt"
	"strings"

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

func (gp *GoParser) SupportedExtensions() []string {
	return []string{`.go`}
}

func (gp *GoParser) ParseFile(filePath, content string) ([]Symbol, error) {
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
	var symbols []Symbol

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

				symbols = append(symbols, Symbol{
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
