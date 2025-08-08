package tools

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

// GoParser implements LanguageParser for Go files
type GoParser struct {
	fileSet *token.FileSet
}

func NewGoParser() *GoParser {
	return &GoParser{
		fileSet: token.NewFileSet(),
	}
}

func (gp *GoParser) Language() string {
	return "Go"
}

func (gp *GoParser) SupportedExtensions() []string {
	return []string{".go"}
}

func (gp *GoParser) ParseFile(filePath, content string) []Symbol {
	return gp.parseGoFileContent(filePath, content)
}

func (gp *GoParser) parseGoFileContent(filePath, content string) []Symbol {
	var symbols []Symbol
	
	// Parse the Go source code
	file, err := parser.ParseFile(gp.fileSet, filePath, content, parser.ParseComments)
	if err != nil {
		// If parsing fails, try to extract symbols using regex as fallback
		return gp.extractSymbolsWithRegex(filePath, content)
	}
	
	packageName := file.Name.Name
	
	// Walk the AST to find symbols
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		
		switch node := n.(type) {
		case *ast.FuncDecl:
			pos := gp.fileSet.Position(node.Pos())
			symbolType := "function"
			if node.Recv != nil {
				symbolType = "method"
			}
			symbols = append(symbols, Symbol{
				Name:     node.Name.Name,
				Type:     symbolType,
				Package:  packageName,
				FilePath: filePath,
				Line:     pos.Line,
			})
			
		case *ast.TypeSpec:
			pos := gp.fileSet.Position(node.Pos())
			symbols = append(symbols, Symbol{
				Name:     node.Name.Name,
				Type:     "type",
				Package:  packageName,
				FilePath: filePath,
				Line:     pos.Line,
			})
			
		case *ast.GenDecl:
			if node.Tok == token.CONST {
				for _, spec := range node.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						pos := gp.fileSet.Position(spec.Pos())
						for _, name := range valueSpec.Names {
							symbols = append(symbols, Symbol{
								Name:     name.Name,
								Type:     "constant",
								Package:  packageName,
								FilePath: filePath,
								Line:     pos.Line,
							})
						}
					}
				}
			} else if node.Tok == token.VAR {
				for _, spec := range node.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						pos := gp.fileSet.Position(spec.Pos())
						for _, name := range valueSpec.Names {
							symbols = append(symbols, Symbol{
								Name:     name.Name,
								Type:     "variable",
								Package:  packageName,
								FilePath: filePath,
								Line:     pos.Line,
							})
						}
					}
				}
			}
		}
		return true
	})
	
	return symbols
}

// Fallback regex-based symbol extraction for when AST parsing fails
func (gp *GoParser) extractSymbolsWithRegex(filePath, content string) []Symbol {
	var symbols []Symbol
	lines := strings.Split(content, "\n")
	
	// Regex patterns for Go symbols
	funcRegex := regexp.MustCompile(`^func\s+(\w+)\s*\(`)
	methodRegex := regexp.MustCompile(`^func\s+\([^)]+\)\s+(\w+)\s*\(`)
	typeRegex := regexp.MustCompile(`^type\s+(\w+)\s+`)
	varRegex := regexp.MustCompile(`^var\s+(\w+)\s+`)
	constRegex := regexp.MustCompile(`^const\s+(\w+)\s+`)
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		if matches := funcRegex.FindStringSubmatch(line); len(matches) > 1 {
			symbols = append(symbols, Symbol{
				Name:     matches[1],
				Type:     "function",
				FilePath: filePath,
				Line:     i + 1,
			})
		} else if matches := methodRegex.FindStringSubmatch(line); len(matches) > 1 {
			symbols = append(symbols, Symbol{
				Name:     matches[1],
				Type:     "method",
				FilePath: filePath,
				Line:     i + 1,
			})
		} else if matches := typeRegex.FindStringSubmatch(line); len(matches) > 1 {
			symbols = append(symbols, Symbol{
				Name:     matches[1],
				Type:     "type",
				FilePath: filePath,
				Line:     i + 1,
			})
		} else if matches := varRegex.FindStringSubmatch(line); len(matches) > 1 {
			symbols = append(symbols, Symbol{
				Name:     matches[1],
				Type:     "variable",
				FilePath: filePath,
				Line:     i + 1,
			})
		} else if matches := constRegex.FindStringSubmatch(line); len(matches) > 1 {
			symbols = append(symbols, Symbol{
				Name:     matches[1],
				Type:     "constant",
				FilePath: filePath,
				Line:     i + 1,
			})
		}
	}
	
	return symbols
}