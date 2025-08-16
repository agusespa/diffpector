package tools

import (
	"reflect"
	"strings"
	"testing"

	"github.com/agusespa/diffpector/internal/types"
	"github.com/agusespa/diffpector/internal/utils"
)

func TestFilterAffectedSymbols(t *testing.T) {
	testCases := []struct {
		name        string
		symbols     []types.Symbol
		diffContext map[string][]utils.LineRange
		want        []types.Symbol
	}{
		{
			name: "Basic case with multiple changes",
			symbols: []types.Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
				{Name: "bar", FilePath: "file.go", StartLine: 15, EndLine: 20},
				{Name: "baz", FilePath: "file.go", StartLine: 25, EndLine: 30},
			},
			diffContext: map[string][]utils.LineRange{
				"file.go": {
					{Start: 8, Count: 2},  // Change inside "foo"
					{Start: 18, Count: 1}, // Change inside "bar"
				},
			},
			want: []types.Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
				{Name: "bar", FilePath: "file.go", StartLine: 15, EndLine: 20},
			},
		},
		{
			name: "Change on start and end lines",
			symbols: []types.Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
				{Name: "bar", FilePath: "file.go", StartLine: 11, EndLine: 15},
			},
			diffContext: map[string][]utils.LineRange{
				"file.go": {
					{Start: 5, Count: 1},  // Change on the start line of "foo"
					{Start: 15, Count: 1}, // Change on the end line of "bar"
				},
			},
			want: []types.Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
				{Name: "bar", FilePath: "file.go", StartLine: 11, EndLine: 15},
			},
		},
		{
			name: "No affected symbols",
			symbols: []types.Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
			},
			diffContext: map[string][]utils.LineRange{
				"file.go": {
					{Start: 12, Count: 2}, // Change outside any symbol range
				},
			},
			want: []types.Symbol{},
		},
		{
			name: "Multiple symbols on the same line",
			symbols: []types.Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 5},
				{Name: "bar", FilePath: "file.go", StartLine: 5, EndLine: 5},
			},
			diffContext: map[string][]utils.LineRange{
				"file.go": {
					{Start: 5, Count: 1}, // Change on the single line where both symbols reside
				},
			},
			want: []types.Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 5},
				{Name: "bar", FilePath: "file.go", StartLine: 5, EndLine: 5},
			},
		},
		{
			name: "Change in an unrelated file",
			symbols: []types.Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
			},
			diffContext: map[string][]utils.LineRange{
				"other_file.go": {
					{Start: 1, Count: 1},
				},
			},
			want: []types.Symbol{},
		},
		{
			name: "Single symbol with multiple overlapping changes",
			symbols: []types.Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
			},
			diffContext: map[string][]utils.LineRange{
				"file.go": {
					{Start: 6, Count: 1}, // First change inside "foo"
					{Start: 8, Count: 1}, // Second change inside "foo"
				},
			},
			want: []types.Symbol{
				{Name: "foo", FilePath: "file.go", StartLine: 5, EndLine: 10},
			},
		},
		{
			name: "Precise line filtering - should exclude symbols outside changed lines",
			symbols: []types.Symbol{
				{Name: "User", FilePath: "internal/database/user.go", StartLine: 12, EndLine: 20},     // Struct definition
				{Name: "Name", FilePath: "internal/database/user.go", StartLine: 15, EndLine: 15},     // Field outside changed lines
				{Name: "CreatedAt", FilePath: "internal/database/user.go", StartLine: 16, EndLine: 16}, // Field outside changed lines
				{Name: "IsActive", FilePath: "internal/database/user.go", StartLine: 18, EndLine: 18},  // Field on changed line
				{Name: "DeleteUser", FilePath: "internal/database/user.go", StartLine: 30, EndLine: 35}, // Function containing lines 31-32
			},
			diffContext: map[string][]utils.LineRange{
				"internal/database/user.go": {
					{Start: 18, Count: 2}, // Only lines 18-19 changed (precise, not hunk range 15-22)
					{Start: 31, Count: 2}, // Only lines 31-32 changed (precise, not hunk range 28-35)
				},
			},
			want: []types.Symbol{
				// Should only include symbols that actually overlap with lines 18-19 and 31-32
				{Name: "User", FilePath: "internal/database/user.go", StartLine: 12, EndLine: 20},        // Overlaps with 18-19
				{Name: "IsActive", FilePath: "internal/database/user.go", StartLine: 18, EndLine: 18},   // Exactly on line 18
				{Name: "DeleteUser", FilePath: "internal/database/user.go", StartLine: 30, EndLine: 35}, // Contains lines 31-32
				// Should NOT include Name (line 15) or CreatedAt (line 16) - they're outside changed lines
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterAffectedSymbols(tc.symbols, tc.diffContext)

			if len(got) == 0 && len(tc.want) == 0 {
				return
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSymbolContextToolEndToEnd(t *testing.T) {
	registry := NewParserRegistry()
	tool := NewSymbolContextTool(".", registry)

	diff := `diff --git a/internal/database/user.go b/internal/database/user.go
index 1234567..abcdefg 100644
--- a/internal/database/user.go
+++ b/internal/database/user.go
@@ -15,8 +15,8 @@ type User struct {
 }
 
 func (db *Database) GetUserByEmail(email string) (*User, error) {
-	query := "SELECT id, email, name FROM users WHERE email = ?"
-	row := db.conn.QueryRow(query, email)
+	query := fmt.Sprintf("SELECT id, email, name FROM users WHERE email = '%s'", email)
+	row := db.conn.QueryRow(query)
 	
 	var user User
 	err := row.Scan(&user.ID, &user.Email, &user.Name)
@@ -28,8 +28,8 @@ func (db *Database) GetUserByEmail(email string) (*User, error) {
 }
 
 func (db *Database) DeleteUser(userID string) error {
-	query := "DELETE FROM users WHERE id = ?"
-	_, err := db.conn.Exec(query, userID)
+	query := fmt.Sprintf("DELETE FROM users WHERE id = %s", userID)
+	_, err := db.conn.Exec(query)
 	return err
 }`

	fileContent := `package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type User struct {
	ID        int       // line 12
	Email     string    // line 13
	Name      string    // line 14 - should NOT be included (outside changed lines)
	CreatedAt time.Time // line 15 - should NOT be included (outside changed lines)
	UpdatedAt time.Time // line 16 - should NOT be included (outside changed lines)
	IsActive  bool      // line 17 - should NOT be included (outside changed lines)
	Role      string    // line 18 - SHOULD be included (on changed line)
}

type Database struct {
	conn *sql.DB
}

func (db *Database) GetUserByEmail(email string) (*User, error) {
	// This function contains the changed lines 18-19
	query := "SELECT id, email, name FROM users WHERE email = ?"
	row := db.conn.QueryRow(query, email)
	
	var user User
	err := row.Scan(&user.ID, &user.Email, &user.Name)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *Database) DeleteUser(userID string) error {
	// This function contains the changed lines 31-32
	query := "DELETE FROM users WHERE id = ?"
	_, err := db.conn.Exec(query, userID)
	return err
}`

	args := map[string]any{
		"file_contents": map[string]string{
			"internal/database/user.go": fileContent,
		},
		"diff":             diff,
		"primary_language": "go",
	}

	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	
	if strings.Contains(result, "CreatedAt") || strings.Contains(result, "UpdatedAt") {
		t.Errorf("REQUIREMENT VIOLATION: Found symbols outside changed lines. This indicates diff parser is using hunk ranges instead of precise lines. Result: %s", result)
	}
	
	if strings.Contains(result, "Name") && !strings.Contains(result, "GetUserByEmail") {
		t.Errorf("REQUIREMENT VIOLATION: Found field 'Name' but not function 'GetUserByEmail'. This suggests imprecise symbol filtering. Result: %s", result)
	}

	lineCount := strings.Count(result, "\n")
	if lineCount > 50 {
		t.Errorf("REQUIREMENT VIOLATION: Context is too verbose (%d lines). Should be focused on changed symbols only. Result: %s", lineCount, result)
	}

	t.Logf("✅ End-to-end test passed - precise symbol filtering working correctly")
}
