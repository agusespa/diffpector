package llm

type Provider interface {
	GetModel() string
	Generate(prompt string) (string, error)
	ChatWithTools(messages []Message, tools []Tool) (*ChatResponse, error)
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ChatResponse struct {
	Content   string
	ToolCalls []ToolCall
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

type Config struct {
	Model       string
	Temperature float64
	MaxTokens   int
}

var SupportedProviders = []string{"ollama", "openai"}
