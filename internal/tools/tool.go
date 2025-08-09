package tools

import "fmt"

type Tool interface {
	Execute(args map[string]any) (string, error)
	Description() string
	Name() string
}

type ToolName string

const (
	ToolNameGitStagedFiles ToolName = "git_staged_files"
	ToolNameGitDiff        ToolName = "git_diff"
	ToolNameReadFile       ToolName = "read_file"
	ToolNameSymbolContext  ToolName = "symbol_context"
	ToolNameWriteFile      ToolName = "write_file"
	ToolNameGitGrep        ToolName = "git_grep"
	ToolNameAppendFile     ToolName = "append_file"
)

type Registry struct {
	tools map[ToolName]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[ToolName]Tool),
	}
}

func (r *Registry) Register(name ToolName, tool Tool) {
	r.tools[name] = tool
}

func (r *Registry) Get(name ToolName) Tool {
	tool, exists := r.tools[name]
	if !exists {
		panic(fmt.Sprintf("BUG: Requested tool '%s' not found in ToolRegistry", name))
	}
	return tool
}

func (r *Registry) List() map[ToolName]Tool {
	return r.tools
}
