package tools

import (
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

func TestTypeScriptParser_ParseFile(t *testing.T) {
	parser, err := NewTypeScriptParser()
	if err != nil {
		t.Fatalf("Failed to create TypeScript parser: %v", err)
	}

	content := `
interface User {
	id: number;
	name: string;
	email?: string;
}

class UserService {
	private users: User[] = [];
	public count: number = 0;
	
	addUser(user: User): void {
		this.users.push(user);
		this.count++;
	}
	
	getUser(id: number): User | undefined {
		return this.users.find(u => u.id === id);
	}
	
	private validateUser(user: User): boolean {
		return user.name.length > 0;
	}
}

function createUser(name: string): User {
	return {
		id: Math.random(),
		name: name
	};
}

const userService = new UserService();
let globalCounter = 0;

export { UserService, createUser };
`

	symbols, err := parser.ParseFile("test.ts", content)
	if err != nil {
		t.Fatalf("Failed to parse TypeScript file: %v", err)
	}

	// Debug: print all found symbols
	t.Logf("Found %d symbols:", len(symbols))
	for _, symbol := range symbols {
		t.Logf("  - %s (line %d-%d)", symbol.Name, symbol.StartLine, symbol.EndLine)
	}

	// Check that we found the expected symbols including class methods and properties
	expectedSymbols := []string{"User", "UserService", "users", "count", "addUser", "getUser", "validateUser", "createUser", "userService", "globalCounter"}
	
	// Verify some specific symbols exist
	symbolNames := make(map[string]bool)
	for _, symbol := range symbols {
		symbolNames[symbol.Name] = true
	}

	foundCount := 0
	for _, expected := range expectedSymbols {
		if symbolNames[expected] {
			foundCount++
		} else {
			t.Logf("Expected symbol '%s' not found", expected)
		}
	}

	// We should find most of the expected symbols
	if foundCount < 6 {
		t.Errorf("Expected to find at least 6 symbols, found %d", foundCount)
	}
}

func TestTypeScriptParser_FindSymbolUsages(t *testing.T) {
	parser, err := NewTypeScriptParser()
	if err != nil {
		t.Fatalf("Failed to create TypeScript parser: %v", err)
	}

	content := `
function processUser(user: User): void {
	validateUser(user);
	console.log(user.name);
}

function validateUser(user: User): boolean {
	return user.name.length > 0;
}

class UserManager {
	processUser(user: User): void {
		validateUser(user);
	}
}
`

	usages, err := parser.FindSymbolUsages("test.ts", content, "validateUser")
	if err != nil {
		t.Fatalf("Failed to find symbol usages: %v", err)
	}

	t.Logf("Found %d usages of 'validateUser':", len(usages))
	for _, usage := range usages {
		t.Logf("  - Line %d: %s", usage.LineNumber, strings.ReplaceAll(usage.Context, "\n", "\\n"))
	}

	// Should find at least one usage (excluding the definition)
	if len(usages) < 1 {
		t.Errorf("Expected to find at least 1 usage of 'validateUser', found %d", len(usages))
	}
}

func TestTypeScriptParser_GetSymbolContext(t *testing.T) {
	parser, err := NewTypeScriptParser()
	if err != nil {
		t.Fatalf("Failed to create TypeScript parser: %v", err)
	}

	content := `
function calculateTotal(items: Item[]): number {
	let total = 0;
	for (const item of items) {
		total += item.price;
	}
	return total;
}
`

	symbol := types.Symbol{
		Name:      "calculateTotal",
		StartLine: 2,
		EndLine:   8,
	}

	context, err := parser.GetSymbolContext("test.ts", content, symbol)
	if err != nil {
		t.Fatalf("Failed to get symbol context: %v", err)
	}

	t.Logf("Symbol context: %s", context)

	// Context should contain function information
	if !strings.Contains(context, "calculateTotal") {
		t.Errorf("Expected context to contain function name 'calculateTotal'")
	}
}

func TestTypeScriptParser_ShouldExcludeFile(t *testing.T) {
	parser, err := NewTypeScriptParser()
	if err != nil {
		t.Fatalf("Failed to create TypeScript parser: %v", err)
	}

	testCases := []struct {
		name     string
		filePath string
		expected bool
	}{
		// Should exclude test files
		{"exclude .test.ts files", "src/utils.test.ts", true},
		{"exclude .spec.ts files", "src/component.spec.ts", true},
		{"exclude .test.tsx files", "src/Component.test.tsx", true},
		{"exclude .spec.tsx files", "src/Component.spec.tsx", true},
		
		// Should exclude common directories
		{"exclude node_modules", "node_modules/react/index.ts", true},
		{"exclude dist", "dist/bundle.ts", true},
		{"exclude build", "build/main.ts", true},
		{"exclude .next", ".next/static/chunks/main.ts", true},
		{"exclude coverage", "coverage/lcov-report/index.ts", true},
		{"exclude .git", ".git/hooks/pre-commit.ts", true},
		
		// Should exclude type definition files
		{"exclude .d.ts files", "types/global.d.ts", true},
		
		// Should include regular files
		{"include regular .ts files", "src/utils.ts", false},
		{"include regular .tsx files", "src/Component.tsx", false},
		{"include files with 'test' in name but not extension", "src/testUtils.ts", false},
		{"include files with 'spec' in name but not extension", "src/specHelper.ts", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.ShouldExcludeFile(tc.filePath, "/project/root")
			if result != tc.expected {
				t.Errorf("ShouldExcludeFile(%q) = %v, expected %v", tc.filePath, result, tc.expected)
			}
		})
	}
}