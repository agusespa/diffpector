package utils

import (
	"reflect"
	"testing"
)

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		input  string
		expect *LineRange
	}{
		{"@@ -1,4 +10,6 @@", &LineRange{Start: 10, Count: 6}},
		{"@@ -5 +20 @@", &LineRange{Start: 20, Count: 1}},   // single line change, no count
		{"@@ -1,3 +4,0 @@", &LineRange{Start: 4, Count: 0}}, // deletion hunk, count 0
		{"@@ -1,3 +a,b @@", nil},                            // invalid numbers
		{"invalid header", nil},                             // malformed input
	}

	for _, tt := range tests {
		got := ParseHunkHeader(tt.input)
		if !reflect.DeepEqual(got, tt.expect) {
			t.Errorf("parseHunkHeader(%q) = %v; want %v", tt.input, got, tt.expect)
		}
	}
}

func TestGetDiffContext(t *testing.T) {
	diff := `
diff --git a/file1.go b/file1.go
index 83db48f..f735c20 100644
--- a/file1.go
+++ b/file1.go
@@ -1,3 +1,4 @@
+added line
diff --git a/file2.go b/file2.go
index 83db48f..f735c20 100644
--- a/file2.go
+++ b/file2.go
@@ -10,2 +10,3 @@
+another added line
`

	expect := map[string][]LineRange{
		"file1.go": {{Start: 1, Count: 1}}, // Only the +added line
		"file2.go": {{Start: 10, Count: 1}}, // Only the +another added line
	}

	got := GetDiffContext(diff)
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("GetDiffContext() = %v; want %v", got, expect)
	}
}

func TestGetDiffContextMultipleHunks(t *testing.T) {
	diff := `
diff --git a/file.go b/file.go
index 83db48f..f735c20 100644
--- a/file.go
+++ b/file.go
@@ -1,2 +1,3 @@
+line1
@@ -10,3 +11,4 @@
+line2
`

	expect := map[string][]LineRange{
		"file.go": {
			{Start: 1, Count: 1},  // Only the +line1
			{Start: 11, Count: 1}, // Only the +line2
		},
	}

	got := GetDiffContext(diff)
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("GetDiffContext() with multiple hunks = %v; want %v", got, expect)
	}
}

func TestGetDiffContextNewFileAndDeletion(t *testing.T) {
	diff := `
diff --git a/newfile.go b/newfile.go
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/newfile.go
@@ -0,0 +1,5 @@
+line1
+line2
`

	expect := map[string][]LineRange{
		"newfile.go": {
			{Start: 1, Count: 2}, // Only the two +line1 and +line2
		},
	}

	got := GetDiffContext(diff)
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("GetDiffContext() new file = %v; want %v", got, expect)
	}
}

func TestGetDiffContextRealWorldScenario(t *testing.T) {
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

	got := GetDiffContext(diff)
	
	expect := map[string][]LineRange{
		"internal/database/user.go": {{Start: 18, Count: 2}, {Start: 31, Count: 2}},
	}
	
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("GetDiffContext() = %v; want %v", got, expect)
	}
}

func TestGetDiffContextMixedChanges(t *testing.T) {
	diff := `diff --git a/example.go b/example.go
index 1234567..abcdefg 100644
--- a/example.go
+++ b/example.go
@@ -5,10 +5,12 @@ func example() {
 	fmt.Println("line 5")
 	fmt.Println("line 6")
-	fmt.Println("old line 7")
-	fmt.Println("old line 8")
+	fmt.Println("new line 7")
+	fmt.Println("new line 8")
+	fmt.Println("added line 9")
 	fmt.Println("line 10")
 }`

	got := GetDiffContext(diff)
	
	// Should detect lines 7-9 as changed (2 replacements + 1 addition)
	expect := map[string][]LineRange{
		"example.go": {{Start: 7, Count: 3}},
	}
	
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("GetDiffContext() mixed changes = %v; want %v", got, expect)
	}
}

func TestGetDiffContextConsecutiveRanges(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
index 1234567..abcdefg 100644
--- a/file.go
+++ b/file.go
@@ -1,10 +1,10 @@
+line 1 changed
+line 2 changed
+line 3 changed
 line 4 unchanged
 line 5 unchanged
+line 6 changed
+line 7 changed
 line 8 unchanged
 `

	got := GetDiffContext(diff)
	
	// Should group consecutive changes into ranges
	expect := map[string][]LineRange{
		"file.go": {
			{Start: 1, Count: 3}, // Lines 1-3
			{Start: 6, Count: 2}, // Lines 6-7
		},
	}
	
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("GetDiffContext() consecutive ranges = %v; want %v", got, expect)
	}
}

func TestGetDiffContextEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string][]LineRange
	}{
		{
			name:  "empty input should return empty map",
			input: "",
			want:  map[string][]LineRange{},
		},
		{
			name: "malformed hunk header should be ignored",
			input: `diff --git a/file.go b/file.go
+++ b/file.go
@@ -1,3 +a,b @@
+some line
`,
			want: map[string][]LineRange{}, // Invalid hunk header, no ranges detected
		},
		{
			name: "diff without changes should return empty",
			input: `diff --git a/file.go b/file.go
index 1234567..abcdefg 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,3 @@
 line 1
 line 2
 line 3
`,
			want: map[string][]LineRange{},
		},
		{
			name: "only deletions should still track line positions",
			input: `diff --git a/file.go b/file.go
index 1234567..abcdefg 100644
--- a/file.go
+++ b/file.go
@@ -5,3 +5,1 @@
 line 5
-line 6
-line 7
`,
			want: map[string][]LineRange{
				"file.go": {{Start: 6, Count: 1}}, // Only one deletion position tracked
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDiffContext(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDiffContext() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestGetDiffContextRequirements(t *testing.T) {
	t.Run("requirement: only changed lines should be detected", func(t *testing.T) {
		diff := `diff --git a/file.go b/file.go
index 1234567..abcdefg 100644
--- a/file.go
+++ b/file.go
@@ -10,5 +10,5 @@ func example() {
 	line 10 // unchanged context
 	line 11 // unchanged context  
-	line 12 // this line changed
+	line 12 // NEW VERSION
 	line 13 // unchanged context
 	line 14 // unchanged context
`
		
		got := GetDiffContext(diff)
		
		// Should detect only line 12, not the entire hunk (10-14)
		expect := map[string][]LineRange{
			"file.go": {{Start: 12, Count: 1}},
		}
		
		if !reflect.DeepEqual(got, expect) {
			t.Errorf("Failed requirement: should detect only changed line 12, got %v", got)
		}
	})
	
	t.Run("requirement: multiple files should be handled independently", func(t *testing.T) {
		diff := `diff --git a/file1.go b/file1.go
index 1234567..abcdefg 100644
--- a/file1.go
+++ b/file1.go
@@ -1,1 +1,2 @@
+change in file1
diff --git a/file2.go b/file2.go
index 1234567..abcdefg 100644
--- a/file2.go
+++ b/file2.go
@@ -5,1 +5,2 @@
+change in file2
`
		
		got := GetDiffContext(diff)
		
		// Each file should have its own ranges
		if len(got) != 2 {
			t.Errorf("Should handle 2 files independently, got %d files", len(got))
			return
		}
		
		if len(got["file1.go"]) == 0 || len(got["file2.go"]) == 0 {
			t.Errorf("Both files should have ranges, got %v", got)
			return
		}
		
		if got["file1.go"][0].Start != 1 || got["file2.go"][0].Start != 5 {
			t.Errorf("Files should have independent line numbering, got %v", got)
		}
	})
}
