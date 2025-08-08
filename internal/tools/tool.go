package tools

type Tool interface {
	Execute(args map[string]any) (string, error)
	Description() string
	Name() string
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(name string, tool Tool) {
	r.tools[name] = tool
}

func (r *Registry) Get(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

func (r *Registry) List() map[string]Tool {
	return r.tools
}

func (r *Registry) GetDescriptions() map[string]string {
	descriptions := make(map[string]string)
	for name, tool := range r.tools {
		descriptions[name] = tool.Description()
	}
	return descriptions
}
