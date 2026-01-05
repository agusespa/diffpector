package tools

import (
	"strings"
	"testing"
)

func TestTypeScriptParser_ParseJSXFile(t *testing.T) {
	parser, err := NewTypeScriptParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// React component with multiple usages of the same component
	content := []byte(`import React from 'react';

export function Button({ label, onClick }) {
  return <button onClick={onClick}>{label}</button>;
}

export function App() {
  return (
    <div>
      <h1>My App</h1>
      <Button label="Click me" onClick={() => console.log('clicked')} />
      <Button label="Submit" onClick={() => console.log('submit')} />
      <Button label="Cancel" onClick={() => console.log('cancel')} />
    </div>
  );
}
`)

	symbols, err := parser.ParseFile("test.tsx", content)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Count Button usages
	buttonUsages := 0
	for _, s := range symbols {
		if s.Name == "Button" && strings.Contains(s.Type, "jsx_component_usage") {
			buttonUsages++
			t.Logf("Found Button usage at line %d (type: %s)", s.StartLine, s.Type)
		}
	}

	if buttonUsages < 3 {
		t.Errorf("Expected at least 3 JSX usages of Button component, got %d", buttonUsages)
	}

	// Verify we also captured the Button declaration
	foundButtonDecl := false
	for _, s := range symbols {
		if s.Name == "Button" && strings.HasSuffix(s.Type, "_decl") {
			foundButtonDecl = true
			break
		}
	}

	if !foundButtonDecl {
		t.Error("Expected to find Button declaration")
	}
}

func TestTypeScriptParser_ParseRegularTSFile(t *testing.T) {
	parser, err := NewTypeScriptParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Regular TypeScript file without JSX
	content := []byte(`export function add(a: number, b: number): number {
  return a + b;
}

export function calculate() {
  const result1 = add(1, 2);
  const result2 = add(3, 4);
  const result3 = add(5, 6);
  return result1 + result2 + result3;
}
`)

	symbols, err := parser.ParseFile("test.ts", content)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Count add usages
	addUsages := 0
	for _, s := range symbols {
		if s.Name == "add" && strings.HasSuffix(s.Type, "_usage") {
			addUsages++
			t.Logf("Found add usage at line %d (type: %s)", s.StartLine, s.Type)
		}
	}

	if addUsages < 3 {
		t.Errorf("Expected at least 3 usages of add function, got %d", addUsages)
	}
}

func TestTypeScriptParser_TypeReferences(t *testing.T) {
	parser, err := NewTypeScriptParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// TypeScript file with multiple type references
	content := []byte(`import type { OrderInformation, RecurringOrderInformation } from '@commercetools/platform-sdk';

type ExcludedProperties = 'state';
type AllowedOrderKeys = Exclude<keyof OrderInformation | keyof RecurringOrderInformation, ExcludedProperties>;

export type OrderInformationEntry = {
  rowKey: AllowedOrderKeys;
  value: (OrderInformation & RecurringOrderInformation)[AllowedOrderKeys];
};
`)

	symbols, err := parser.ParseFile("test.tsx", content)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Count RecurringOrderInformation usages
	recurringOrderUsages := 0
	for _, s := range symbols {
		if s.Name == "RecurringOrderInformation" {
			t.Logf("Found RecurringOrderInformation at line %d (type: %s)", s.StartLine, s.Type)
			if s.Type == "type_usage" || strings.HasSuffix(s.Type, "_usage") {
				recurringOrderUsages++
			}
		}
	}

	t.Logf("Total type usages of 'RecurringOrderInformation': %d", recurringOrderUsages)

	// We should have at least 2 type usages (lines 4 and 8 in the example)
	// Line 1 is an import, line 4 is in keyof, line 8 is in the intersection type
	if recurringOrderUsages < 2 {
		t.Errorf("Expected at least 2 type usages of RecurringOrderInformation, got %d", recurringOrderUsages)
	}

	// Count OrderInformation usages
	orderInfoUsages := 0
	for _, s := range symbols {
		if s.Name == "OrderInformation" {
			if s.Type == "type_usage" || strings.HasSuffix(s.Type, "_usage") {
				orderInfoUsages++
				t.Logf("Found OrderInformation usage at line %d (type: %s)", s.StartLine, s.Type)
			}
		}
	}

	t.Logf("Total type usages of 'OrderInformation': %d", orderInfoUsages)

	if orderInfoUsages < 2 {
		t.Errorf("Expected at least 2 type usages of OrderInformation, got %d", orderInfoUsages)
	}
}

func TestTypeScriptParser_TypeReferencesInRegularTS(t *testing.T) {
	parser, err := NewTypeScriptParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Regular TypeScript file with type references
	content := []byte(`export interface User {
  id: string;
  name: string;
}

export function getUser(id: string): User {
  return { id, name: 'Test' };
}

export function processUser(user: User): void {
  console.log(user.name);
}

export const users: User[] = [];
`)

	symbols, err := parser.ParseFile("test.ts", content)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Count User type usages
	userTypeUsages := 0
	for _, s := range symbols {
		if s.Name == "User" && (s.Type == "type_usage" || strings.HasSuffix(s.Type, "_usage")) {
			userTypeUsages++
			t.Logf("Found User type usage at line %d (type: %s)", s.StartLine, s.Type)
		}
	}

	t.Logf("Total type usages of 'User': %d", userTypeUsages)

	// We should have at least 3 type usages (return type, parameter type, array type)
	if userTypeUsages < 3 {
		t.Errorf("Expected at least 3 type usages of User, got %d", userTypeUsages)
	}
}
