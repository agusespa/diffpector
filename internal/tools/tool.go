package tools

import "fmt"

type Tool interface {
	Name() string
	Description() string
	Execute(args map[string]any) (any, error)
}

type ToolName string

const (
	ToolNameGitDiff       ToolName = "git_diff"
	ToolNameReadFile      ToolName = "read_file"
	ToolNameSymbolContext ToolName = "symbol_context"
	ToolNameWriteFile     ToolName = "write_file"
	ToolNameGitGrep       ToolName = "git_grep"
)

type ToolRegistry struct {
	tools map[ToolName]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[ToolName]Tool),
	}
}

func (r *ToolRegistry) Register(name ToolName, tool Tool) {
	r.tools[name] = tool
}

func (r *ToolRegistry) Get(name ToolName) Tool {
	tool, exists := r.tools[name]
	if !exists {
		panic(fmt.Sprintf("BUG: Requested tool '%s' not found in ToolRegistry", name))
	}
	return tool
}
