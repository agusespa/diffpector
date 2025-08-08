package llm

type Provider interface {
	Generate(prompt string) (string, error)
	GetModel() string
	SetModel(model string)
}

type Config struct {
	Model       string
	Temperature float64
	MaxTokens   int
}

var SupportedProviders = []string{"ollama"}
