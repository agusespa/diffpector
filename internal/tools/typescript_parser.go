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

func (tp *TypeScriptParser) SupportedExtensions() []string {
	return []string{`.ts`, `.tsx`}
}

func (tp *TypeScriptParser) ShouldExcludeFile(filePath, projectRoot string) bool {
	lowerPath := strings.ToLower(filePath)

	tsExcludePatterns := []string{
		"test/",
		"tests/",
		"__tests__/",
		"node_modules/",
		"dist/",
		"build/",
		".git/",
		".spec.ts",
		".test.ts",
		".spec.tsx",
		".test.tsx",
	}

	for _, pattern := range tsExcludePatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

func (tp *TypeScriptParser) ParseFile(filePath string, content []byte) ([]types.Symbol, error) {
	tree := tp.parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse TypeScript file")
	}
	defer tree.Close()

	packageName := tp.extractModuleName(filePath)

	queryText := `
[
  ;; === Declarations ===
  (function_declaration) @func_decl
  (method_definition) @method_decl
  (class_declaration) @class_decl
  (interface_declaration) @interface_decl
  (type_alias_declaration) @type_decl
  (enum_declaration) @enum_decl
  (lexical_declaration) @var_decl
  (variable_declaration) @var_decl
  (import_statement) @import_decl

  ;; === Usages ===
  (call_expression function: (identifier) @func_usage)
  (call_expression function: (member_expression property: (property_identifier) @method_usage))
  (member_expression property: (property_identifier) @field_usage)
  (identifier) @var_usage
]
`

	q, err := sitter.NewQuery(tp.language, queryText)
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
				name = tp.extractDeclarationName(c.Node, content, captureName)
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

func (tp *TypeScriptParser) extractDeclarationName(node sitter.Node, content []byte, captureName string) string {
	switch captureName {
	case "func_decl", "method_decl", "class_decl", "interface_decl", "type_decl", "enum_decl":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return nameNode.Utf8Text(content)
		}
	case "var_decl":
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
			if child != nil && child.Kind() == "import_clause" {
				for j := uint(0); j < child.ChildCount(); j++ {
					grandchild := child.Child(j)
					if grandchild != nil && grandchild.Kind() == "identifier" {
						return grandchild.Utf8Text(content)
					}
				}
			}
		}
	}
	return ""
}

func (tp *TypeScriptParser) extractModuleName(filePath string) string {
	parts := strings.Split(filePath, "/")
	if len(parts) > 0 {
		fileName := parts[len(parts)-1]
		return strings.TrimSuffix(strings.TrimSuffix(fileName, ".ts"), ".tsx")
	}
	return ""
}
