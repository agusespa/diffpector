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
		"file1.go": {{Start: 1, Count: 4}},
		"file2.go": {{Start: 10, Count: 3}},
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
			{Start: 1, Count: 3},
			{Start: 11, Count: 4},
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
			{Start: 1, Count: 5},
		},
	}

	got := GetDiffContext(diff)
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("GetDiffContext() new file = %v; want %v", got, expect)
	}
}

func TestGetDiffContextEmptyAndMalformed(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string][]LineRange
	}{
		{
			name:  "empty input",
			input: "",
			want:  map[string][]LineRange{},
		},
		{
			name: "malformed input",
			input: `
+++ b/file.go
@@ -1,3 +a,b @@
`,
			want: map[string][]LineRange{},
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
