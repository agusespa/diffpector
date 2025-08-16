package tools

import (
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

func TestJavaParser_ParseFile(t *testing.T) {
	parser, err := NewJavaParser()
	if err != nil {
		t.Fatalf("Failed to create Java parser: %v", err)
	}

	content := `
package com.example.service;

import java.util.List;
import java.util.ArrayList;
import java.util.Optional;

public interface UserRepository {
	Optional<User> findById(String id);
	List<User> findAll();
	void save(User user);
}

public class UserService {
	private final UserRepository repository;
	private static final String DEFAULT_ROLE = "USER";
	
	public UserService(UserRepository repository) {
		this.repository = repository;
	}
	
	public Optional<User> getUser(String id) {
		if (id == null || id.isEmpty()) {
			throw new IllegalArgumentException("ID cannot be null or empty");
		}
		return repository.findById(id);
	}
	
	public void createUser(String name, String email) {
		User user = new User(name, email, DEFAULT_ROLE);
		repository.save(user);
	}
	
	private boolean validateEmail(String email) {
		return email != null && email.contains("@");
	}
}

public enum UserRole {
	ADMIN("admin"),
	USER("user"),
	GUEST("guest");
	
	private final String value;
	
	UserRole(String value) {
		this.value = value;
	}
	
	public String getValue() {
		return value;
	}
}

@Entity
public class User {
	@Id
	private String id;
	private String name;
	private String email;
	private UserRole role;
	
	public User() {}
	
	public User(String name, String email, String role) {
		this.name = name;
		this.email = email;
		this.role = UserRole.valueOf(role);
	}
	
	// Getters and setters would be here
}
`

	symbols, err := parser.ParseFile("UserService.java", content)
	if err != nil {
		t.Fatalf("Failed to parse Java file: %v", err)
	}

	// Debug: print all found symbols
	t.Logf("Found %d symbols:", len(symbols))
	for _, symbol := range symbols {
		t.Logf("  - %s (line %d-%d)", symbol.Name, symbol.StartLine, symbol.EndLine)
	}

	// Check that we found the expected symbols
	expectedSymbols := []string{
		"UserRepository", "findById", "findAll", "save",
		"UserService", "repository", "DEFAULT_ROLE", "getUser", "createUser", "validateEmail",
		"UserRole", "ADMIN", "USER", "GUEST", "value", "getValue",
		"User", "id", "name", "email", "role",
	}
	
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
	if foundCount < 15 {
		t.Errorf("Expected to find at least 15 symbols, found %d", foundCount)
	}

	// Verify package name is extracted correctly
	for _, symbol := range symbols {
		if symbol.Package != "com.example.service" {
			t.Errorf("Expected package 'com.example.service', got '%s'", symbol.Package)
			break
		}
	}
}

func TestJavaParser_FindSymbolUsages(t *testing.T) {
	parser, err := NewJavaParser()
	if err != nil {
		t.Fatalf("Failed to create Java parser: %v", err)
	}

	content := `
package com.example;

public class OrderService {
	private UserService userService;
	
	public void processOrder(String userId, Order order) {
		User user = userService.getUser(userId);
		if (user == null) {
			throw new IllegalArgumentException("User not found");
		}
		
		validateOrder(order);
		saveOrder(order);
	}
	
	private void validateOrder(Order order) {
		if (order.getItems().isEmpty()) {
			throw new IllegalArgumentException("Order must have items");
		}
	}
	
	private void saveOrder(Order order) {
		// Save logic here
	}
	
	public void cancelOrder(String orderId) {
		Order order = findOrder(orderId);
		validateOrder(order);
		order.cancel();
	}
}
`

	usages, err := parser.FindSymbolUsages("OrderService.java", content, "validateOrder")
	if err != nil {
		t.Fatalf("Failed to find symbol usages: %v", err)
	}

	t.Logf("Found %d usages of 'validateOrder':", len(usages))
	for _, usage := range usages {
		t.Logf("  - Line %d: %s", usage.LineNumber, strings.ReplaceAll(usage.Context, "\n", "\\n"))
	}

	// Should find at least 2 usages (excluding the definition)
	if len(usages) < 2 {
		t.Errorf("Expected to find at least 2 usages of 'validateOrder', found %d", len(usages))
	}
}

func TestJavaParser_GetSymbolContext(t *testing.T) {
	parser, err := NewJavaParser()
	if err != nil {
		t.Fatalf("Failed to create Java parser: %v", err)
	}

	content := `
package com.example;

public class Calculator {
	public int add(int a, int b) {
		return a + b;
	}
	
	public double calculateTotal(List<Item> items) {
		double total = 0.0;
		for (Item item : items) {
			total += item.getPrice() * item.getQuantity();
		}
		return total;
	}
}
`

	symbol := types.Symbol{
		Name:      "calculateTotal",
		StartLine: 9,
		EndLine:   14,
	}

	context, err := parser.GetSymbolContext("Calculator.java", content, symbol)
	if err != nil {
		t.Fatalf("Failed to get symbol context: %v", err)
	}

	t.Logf("Symbol context: %s", context)

	// Context should contain method information
	if !strings.Contains(context, "calculateTotal") {
		t.Errorf("Expected context to contain method name 'calculateTotal'")
	}
	
	if !strings.Contains(context, "Method:") {
		t.Errorf("Expected context to contain 'Method:' prefix")
	}
}

func TestJavaParser_ShouldExcludeFile(t *testing.T) {
	parser, err := NewJavaParser()
	if err != nil {
		t.Fatalf("Failed to create Java parser: %v", err)
	}

	testCases := []struct {
		name     string
		filePath string
		expected bool
	}{
		// Should exclude test files
		{"exclude Test.java files", "src/test/java/UserServiceTest.java", true},
		{"exclude Tests.java files", "src/test/java/UserServiceTests.java", true},
		{"exclude files in test directory", "src/test/java/UserService.java", true},
		{"exclude files in tests directory", "src/tests/java/UserService.java", true},
		
		// Should exclude build directories
		{"exclude target directory", "target/classes/UserService.java", true},
		{"exclude build directory", "build/classes/UserService.java", true},
		{"exclude .gradle directory", ".gradle/cache/UserService.java", true},
		{"exclude bin directory", "bin/UserService.java", true},
		{"exclude out directory", "out/production/UserService.java", true},
		{"exclude .git directory", ".git/hooks/UserService.java", true},
		
		// Should exclude compiled files
		{"exclude .class files", "target/classes/UserService.class", true},
		{"exclude generated files", "target/generated-sources/UserService.java", true},
		
		// Should include regular files
		{"include regular .java files", "src/main/java/UserService.java", false},
		{"include files with 'test' in name but not in test directory", "src/main/java/TestUtils.java", false},
		{"include files with 'Test' in name but not test files", "src/main/java/TestConfiguration.java", false},
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