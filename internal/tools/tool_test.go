package tools

import (
	"testing"
)

// Mock tool for testing
type mockTool struct {
	name        string
	description string
	result      string
	err         error
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Execute(args map[string]any) (string, error) {
	return m.result, m.err
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Error("Expected registry to be created")
	}
	if registry.tools == nil {
		t.Error("Expected tools map to be initialized")
	}
	if len(registry.tools) != 0 {
		t.Error("Expected empty registry initially")
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
	}

	registry.Register("test_tool", tool)

	if len(registry.tools) != 1 {
		t.Errorf("Expected 1 tool in registry, got %d", len(registry.tools))
	}

	retrievedTool, exists := registry.tools["test_tool"]
	if !exists {
		t.Error("Expected tool to be registered")
	}
	if retrievedTool != tool {
		t.Error("Expected retrieved tool to match registered tool")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
	}

	// Test getting non-existent tool
	_, exists := registry.Get("nonexistent")
	if exists {
		t.Error("Expected tool to not exist")
	}

	// Register and test getting existing tool
	registry.Register("test_tool", tool)
	retrievedTool, exists := registry.Get("test_tool")
	if !exists {
		t.Error("Expected tool to exist")
	}
	if retrievedTool != tool {
		t.Error("Expected retrieved tool to match registered tool")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()
	tool1 := &mockTool{name: "tool1", description: "Tool 1"}
	tool2 := &mockTool{name: "tool2", description: "Tool 2"}

	// Test empty registry
	tools := registry.List()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(tools))
	}

	// Register tools and test
	registry.Register("tool1", tool1)
	registry.Register("tool2", tool2)

	tools = registry.List()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	if tools["tool1"] != tool1 {
		t.Error("Expected tool1 to be in list")
	}
	if tools["tool2"] != tool2 {
		t.Error("Expected tool2 to be in list")
	}
}

func TestRegistry_RegisterOverwrite(t *testing.T) {
	registry := NewRegistry()
	tool1 := &mockTool{name: "test_tool", description: "Tool 1"}
	tool2 := &mockTool{name: "test_tool", description: "Tool 2"}

	registry.Register("test_tool", tool1)
	registry.Register("test_tool", tool2) // Overwrite

	retrievedTool, exists := registry.Get("test_tool")
	if !exists {
		t.Error("Expected tool to exist")
	}
	if retrievedTool != tool2 {
		t.Error("Expected retrieved tool to be the second tool (overwritten)")
	}
	if len(registry.tools) != 1 {
		t.Errorf("Expected 1 tool in registry, got %d", len(registry.tools))
	}
}