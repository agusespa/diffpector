package tools

import "fmt"

type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Execute(args map[string]any) (any, error)
}

type ToolName string

const (
	ToolNameGitDiff       ToolName = "git_diff"
	ToolNameReadFile      ToolName = "read_file"
	ToolNameSymbolContext ToolName = "symbol_context"
	ToolNameWriteFile     ToolName = "write_file"
	ToolNameGitGrep       ToolName = "git_grep"
	ToolNameHumanLoop     ToolName = "human_loop"
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
		panic(fmt.Sprintf("Requested tool '%s' not found in ToolRegistry", name))
	}
	return tool
}

func (r *ToolRegistry) GetAll() map[ToolName]Tool {
	return r.tools
}

func (r *ToolRegistry) GetAllAsList() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}
