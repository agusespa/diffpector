package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewOllamaProvider(t *testing.T) {
	baseURL := "http://localhost:11434"
	model := "test-model"

	provider := NewOllamaProvider(baseURL, model)

	if provider.baseURL != baseURL {
		t.Errorf("Expected baseURL %s, got %s", baseURL, provider.baseURL)
	}
	if provider.model != model {
		t.Errorf("Expected model %s, got %s", model, provider.model)
	}
	if provider.client == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestOllamaProvider_GetModel(t *testing.T) {
	model := "test-model"
	provider := NewOllamaProvider("http://localhost:11434", model)

	if provider.GetModel() != model {
		t.Errorf("Expected model %s, got %s", model, provider.GetModel())
	}
}

func TestOllamaProvider_SetModel(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:11434", "initial-model")
	newModel := "new-model"

	provider.SetModel(newModel)

	if provider.GetModel() != newModel {
		t.Errorf("Expected model %s, got %s", newModel, provider.GetModel())
	}
}

func TestOllamaProvider_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/generate" {
			t.Errorf("Expected path /api/generate, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var req ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if req.Model != "test-model" {
			t.Errorf("Expected model test-model, got %s", req.Model)
		}
		if req.Prompt != "test prompt" {
			t.Errorf("Expected prompt 'test prompt', got %s", req.Prompt)
		}
		if req.Stream != false {
			t.Errorf("Expected stream false, got %t", req.Stream)
		}

		response := ollamaResponse{
			Response: "test response",
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "test-model")
	result, err := provider.Generate("test prompt")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "test response" {
		t.Errorf("Expected response 'test response', got %s", result)
	}
}

func TestOllamaProvider_Generate_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "test-model")
	_, err := provider.Generate("test prompt")

	if err == nil {
		t.Error("Expected error for HTTP 500 response")
	}
	if !contains(err.Error(), "ollama request failed with status: 500") {
		t.Errorf("Expected status error, got: %v", err)
	}
}

func TestOllamaProvider_Generate_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "test-model")
	_, err := provider.Generate("test prompt")

	if err == nil {
		t.Error("Expected error for invalid JSON response")
	}
	if !contains(err.Error(), "failed to unmarshal response") {
		t.Errorf("Expected unmarshal error, got: %v", err)
	}
}

func TestOllamaProvider_Generate_NetworkError(t *testing.T) {
	provider := NewOllamaProvider("http://invalid-url-that-does-not-exist:12345", "test-model")
	
	// Set a short timeout for the test to avoid waiting too long
	provider.client.Timeout = 100 * time.Millisecond
	
	_, err := provider.Generate("test prompt")

	if err == nil {
		t.Error("Expected error for network failure")
	}
	if !contains(err.Error(), "failed to make request") {
		t.Errorf("Expected network error, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
