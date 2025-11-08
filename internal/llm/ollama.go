package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatWithToolsRequest struct {
	Model    string         `json:"model"`
	Messages []Message      `json:"messages"`
	Tools    []Tool         `json:"tools,omitempty"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
}

type ollamaToolCallResponse struct {
	Message struct {
		Role      string           `json:"role"`
		Content   string           `json:"content"`
		ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
	} `json:"message"`
	Done bool `json:"done"`
}

type ollamaToolCall struct {
	Function struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"function"`
}

func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

func (p *OllamaProvider) GetModel() string {
	return p.model
}

func (p *OllamaProvider) Generate(prompt string) (string, error) {
	reqBody := ollamaRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := p.client.Post(p.baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("Error closing response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		defer func() {
			closeErr := resp.Body.Close()
			if closeErr != nil {
				err = fmt.Errorf("failed to cleanly close response body: %w", closeErr)
			}
		}()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("ollama request failed with status: %d, could not read body: %w", resp.StatusCode, err)
		}

		errorBody := string(bodyBytes)

		return "", fmt.Errorf("ollama request failed with status: %d. Details: %s", resp.StatusCode, errorBody)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return ollamaResp.Response, nil
}

func (p *OllamaProvider) ChatWithTools(messages []Message, tools []Tool) (*ChatResponse, error) {
	tuningOptions := map[string]any{
		"num_ctx":        16384,
		"temperature":    0.2,
		"repeat_penalty": 1.1, // Discourage repetitive phrasing
		"top_k":          40,  // Focus on high-probability tokens
	}

	reqBody := ollamaChatWithToolsRequest{
		Model:    p.model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
		Options:  tuningOptions,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := p.client.Post(p.baseURL+"/api/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("Error closing response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("ollama request failed with status: %d, could not read body: %w", resp.StatusCode, err)
		}

		errorBody := string(bodyBytes)
		return nil, fmt.Errorf("ollama request failed with status: %d. Details: %s", resp.StatusCode, errorBody)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp ollamaToolCallResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	chatResp := &ChatResponse{
		Content: ollamaResp.Message.Content,
	}

	// First, check if Ollama returned tool calls in the proper field
	for _, tc := range ollamaResp.Message.ToolCalls {
		chatResp.ToolCalls = append(chatResp.ToolCalls, ToolCall{
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	// Fallback: If no tool calls but content looks like a tool call JSON, parse it
	if len(chatResp.ToolCalls) == 0 && ollamaResp.Message.Content != "" {
		content := ollamaResp.Message.Content

		content = strings.TrimSpace(content)
		if after, ok := strings.CutPrefix(content, "```json"); ok {
			content = after
			content = strings.TrimSuffix(content, "```")
			content = strings.TrimSpace(content)
		} else if after, ok := strings.CutPrefix(content, "```"); ok {
			content = after
			content = strings.TrimSuffix(content, "```")
			content = strings.TrimSpace(content)
		}

		// Only try to parse as tool call if it's an object (not an array) and has required fields
		if strings.HasPrefix(content, "{") && !strings.HasPrefix(content, "[") {
			var toolCallContent struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			if err := json.Unmarshal([]byte(content), &toolCallContent); err == nil && toolCallContent.Name != "" {
				chatResp.ToolCalls = append(chatResp.ToolCalls, ToolCall{
					Name:      toolCallContent.Name,
					Arguments: toolCallContent.Arguments,
				})
				chatResp.Content = ""
			}
		}
	}

	return chatResp, nil
}
