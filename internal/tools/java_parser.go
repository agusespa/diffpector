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

func (jp *JavaParser) SupportedExtensions() []string {
	return []string{`.java`}
}

func (jp *JavaParser) ShouldExcludeFile(filePath, projectRoot string) bool {
	lowerPath := strings.ToLower(filePath)

	javaExcludePatterns := []string{
		"test/",
		"tests/",
		"target/",
		"build/",
		".git/",
	}

	for _, pattern := range javaExcludePatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

func (jp *JavaParser) ParseFile(filePath string, content []byte) ([]types.Symbol, error) {
	tree := jp.parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse Java file")
	}
	defer tree.Close()

	packageName := jp.extractPackageName(tree.RootNode(), content)

	queryText := `
[
  ;; === Declarations ===
  (method_declaration) @method_decl
  (constructor_declaration) @constructor_decl
  (class_declaration) @class_decl
  (interface_declaration) @interface_decl
  (enum_declaration) @enum_decl
  (field_declaration) @field_decl
  (constant_declaration) @const_decl
  (import_declaration) @import_decl

  ;; === Usages ===
  (method_invocation name: (identifier) @method_usage)
  (field_access field: (identifier) @field_usage)
  (identifier) @var_usage
  (type_identifier) @type_usage
]
`

	q, err := sitter.NewQuery(jp.language, queryText)
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

			if strings.HasSuffix(captureName, "_decl") {
				name = jp.extractDeclarationName(c.Node, content, captureName)
			} else {
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

func (jp *JavaParser) extractDeclarationName(node sitter.Node, content []byte, captureName string) string {
	switch captureName {
	case "method_decl", "constructor_decl", "class_decl", "interface_decl", "enum_decl":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return nameNode.Utf8Text(content)
		}
	case "field_decl":
		declarator := node.ChildByFieldName("declarator")
		if declarator != nil {
			if nameNode := declarator.ChildByFieldName("name"); nameNode != nil {
				return nameNode.Utf8Text(content)
			}
		}
	case "const_decl":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && child.Kind() == "variable_declarator" {
				if nameNode := child.ChildByFieldName("name"); nameNode != nil {
					return nameNode.Utf8Text(content)
				}
			}
		}
	case "import_decl":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && (child.Kind() == "scoped_identifier" || child.Kind() == "identifier") {
				return child.Utf8Text(content)
			}
		}
	}
	return ""
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
	return ""
}
