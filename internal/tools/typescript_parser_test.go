package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTypeScriptParser(t *testing.T) {
	parser, err := NewTypeScriptParser()
	require.NoError(t, err)
	assert.NotNil(t, parser)
	assert.Equal(t, "TypeScript", parser.Language())
	assert.Equal(t, []string{".ts", ".tsx"}, parser.SupportedExtensions())
}

func TestTypeScriptParser_ShouldExcludeFile(t *testing.T) {
	parser, err := NewTypeScriptParser()
	require.NoError(t, err)

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"regular file", "src/services/userService.ts", false},
		{"test directory", "src/test/userService.test.ts", true},
		{"tests directory", "src/tests/userService.ts", true},
		{"__tests__ directory", "src/__tests__/userService.ts", true},
		{"node_modules directory", "node_modules/package/index.ts", true},
		{"dist directory", "dist/index.ts", true},
		{"build directory", "build/index.ts", true},
		{"git directory", ".git/index.ts", true},
		{"spec file", "src/userService.spec.ts", true},
		{"test file", "src/userService.test.ts", true},
		{"tsx test file", "src/Component.test.tsx", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ShouldExcludeFile(tt.filePath, "/project")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTypeScriptParser_ParseFile(t *testing.T) {
	parser, err := NewTypeScriptParser()
	require.NoError(t, err)

	tsCode := []byte(`import { User } from './models';

export class UserService {
    private users: User[] = [];
    
    constructor() {
        this.users = [];
    }
    
    public getUser(id: string): User | undefined {
        return this.users.find(u => u.id === id);
    }
    
    public addUser(user: User): void {
        this.users.push(user);
    }
}

export interface UserRepository {
    findById(id: string): User | null;
}
`)

	symbols, err := parser.ParseFile("userService.ts", tsCode)
	require.NoError(t, err)
	assert.NotEmpty(t, symbols)

	// Check that we found some key declarations
	var foundClass, foundMethod, foundInterface bool
	for _, sym := range symbols {
		if sym.Name == "UserService" && sym.Type == "class_decl" {
			foundClass = true
		}
		if sym.Name == "getUser" && sym.Type == "method_decl" {
			foundMethod = true
		}
		if sym.Name == "UserRepository" && sym.Type == "interface_decl" {
			foundInterface = true
		}
	}

	assert.True(t, foundClass, "Should find UserService class")
	assert.True(t, foundMethod, "Should find getUser method")
	assert.True(t, foundInterface, "Should find UserRepository interface")
}
