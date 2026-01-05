package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProvider_Generate(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		mockResponse   openAIResponse
		expectedResult string
		expectError    bool
	}{
		{
			name:   "successful generation",
			prompt: "test prompt",
			mockResponse: openAIResponse{
				Choices: []struct {
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Role:    "assistant",
							Content: "test response",
						},
						FinishReason: "stop",
					},
				},
			},
			expectedResult: "test response",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/chat/completions", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(tt.mockResponse)
				require.NoError(t, err)
			}))
			defer server.Close()

			provider := NewOpenAIProvider(server.URL, "test-model", "")
			result, err := provider.Generate(tt.prompt)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestOpenAIProvider_ChatWithTools(t *testing.T) {
	tests := []struct {
		name         string
		messages     []Message
		tools        []Tool
		mockResponse openAIToolCallResponse
		expectError  bool
	}{
		{
			name: "successful chat without tool calls",
			messages: []Message{
				{Role: "user", Content: "test message"},
			},
			tools: []Tool{},
			mockResponse: openAIToolCallResponse{
				Choices: []struct {
					Message struct {
						Role      string           `json:"role"`
						Content   string           `json:"content"`
						ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Message: struct {
							Role      string           `json:"role"`
							Content   string           `json:"content"`
							ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
						}{
							Role:    "assistant",
							Content: "test response",
						},
						FinishReason: "stop",
					},
				},
			},
			expectError: false,
		},
		{
			name: "successful chat with tool calls",
			messages: []Message{
				{Role: "user", Content: "test message"},
			},
			tools: []Tool{
				{
					Type: "function",
					Function: ToolFunction{
						Name:        "test_tool",
						Description: "A test tool",
						Parameters:  map[string]any{"type": "object"},
					},
				},
			},
			mockResponse: openAIToolCallResponse{
				Choices: []struct {
					Message struct {
						Role      string           `json:"role"`
						Content   string           `json:"content"`
						ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Message: struct {
							Role      string           `json:"role"`
							Content   string           `json:"content"`
							ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
						}{
							Role:    "assistant",
							Content: "",
							ToolCalls: []openAIToolCall{
								{
									ID:   "call_123",
									Type: "function",
									Function: struct {
										Name      string `json:"name"`
										Arguments string `json:"arguments"`
									}{
										Name:      "test_tool",
										Arguments: `{"arg1": "value1"}`,
									},
								},
							},
						},
						FinishReason: "tool_calls",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/chat/completions", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(tt.mockResponse)
				require.NoError(t, err)
			}))
			defer server.Close()

			provider := NewOpenAIProvider(server.URL, "test-model", "")
			result, err := provider.ChatWithTools(tt.messages, tt.tools)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				if len(tt.mockResponse.Choices[0].Message.ToolCalls) > 0 {
					assert.NotEmpty(t, result.ToolCalls)
					assert.Equal(t, "test_tool", result.ToolCalls[0].Name)
				} else {
					assert.Equal(t, "test response", result.Content)
				}
			}
		})
	}
}

func TestOpenAIProvider_GetModel(t *testing.T) {
	provider := NewOpenAIProvider("http://localhost:8080", "test-model", "")
	assert.Equal(t, "test-model", provider.GetModel())
}
