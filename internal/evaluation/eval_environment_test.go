package evaluation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agusespa/diffpector/internal/types"
)

// Helper function to create test environment builder
func createTestBuilder(suiteBaseDir, mockFilesDir string) *TestEnvironmentBuilder {
	return NewTestEnvironmentBuilder(suiteBaseDir, mockFilesDir)
}



func TestNewTestEnvironmentBuilder(t *testing.T) {
	builder := createTestBuilder("/base", "/mock")

	if builder.suiteBaseDir != "/base" {
		t.Errorf("Expected suiteBaseDir to be '/base', got '%s'", builder.suiteBaseDir)
	}

	if builder.mockFilesDir != "/mock" {
		t.Errorf("Expected mockFilesDir to be '/mock', got '%s'", builder.mockFilesDir)
	}
}

func TestCreateTestEnvironment_EmptyDiffFile(t *testing.T) {
	builder := createTestBuilder("", "")

	testCase := types.TestCase{
		Name:     "test",
		DiffFile: "",
	}

	env, err := builder.CreateTestEnvironment(testCase)
	if err != nil {
		t.Fatalf("CreateTestEnvironment failed: %v", err)
	}

	if env.Diff != "" {
		t.Errorf("Expected empty diff, got '%s'", env.Diff)
	}

	if len(env.Files) != 0 {
		t.Errorf("Expected empty files map, got %d files", len(env.Files))
	}
}

func TestCreateTestEnvironment_WithDiffFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-env-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock files directory
	mockDir := filepath.Join(tempDir, "mock")
	err = os.MkdirAll(mockDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create mock dir: %v", err)
	}

	// Create a test file
	testContent := "package main\n\nfunc main() {}\n"
	err = os.WriteFile(filepath.Join(mockDir, "main.go"), []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a diff file
	diffContent := `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
 
+import "fmt"
 func main() {}`

	diffFile := filepath.Join(tempDir, "test.diff")
	err = os.WriteFile(diffFile, []byte(diffContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create diff file: %v", err)
	}

	builder := createTestBuilder(tempDir, mockDir)

	testCase := types.TestCase{
		Name:     "test",
		DiffFile: "test.diff",
	}

	env, err := builder.CreateTestEnvironment(testCase)
	if err != nil {
		t.Fatalf("CreateTestEnvironment failed: %v", err)
	}

	if env.Diff != diffContent {
		t.Errorf("Expected diff content to match")
	}

	if len(env.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(env.Files))
	}

	if env.Files["main.go"] != testContent {
		t.Errorf("Expected file content to match")
	}
}

func TestExtractFilenamesFromDiff(t *testing.T) {
	builder := createTestBuilder("", "")

	diff := `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
 
+import "fmt"
 func main() {
diff --git a/utils.go b/utils.go
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/utils.go
@@ -0,0 +1,5 @@
+package main
+
+func helper() {
+	// helper function
+}`

	filenames := builder.extractFilenamesFromDiff(diff)

	expected := []string{"main.go", "utils.go"}
	if len(filenames) != len(expected) {
		t.Errorf("Expected %d filenames, got %d", len(expected), len(filenames))
	}

	for i, expected := range expected {
		if i >= len(filenames) || filenames[i] != expected {
			t.Errorf("Expected filename %s at index %d, got %s", expected, i, filenames[i])
		}
	}
}

func TestExtractFilenamesFromDiff_IgnoreDevNull(t *testing.T) {
	builder := createTestBuilder("", "")

	diff := `diff --git a/deleted.go b/deleted.go
deleted file mode 100644
index 1234567..0000000
--- a/deleted.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package main
-
-func deleted() {}`

	filenames := builder.extractFilenamesFromDiff(diff)

	if len(filenames) != 0 {
		t.Errorf("Expected 0 filenames (should ignore /dev/null), got %d", len(filenames))
	}
}

func TestLoadMockFileContent_NoMockDir(t *testing.T) {
	builder := createTestBuilder("", "")

	_, err := builder.loadMockFileContent("test.go")
	if err == nil {
		t.Error("Expected error when mock files directory not configured")
	}

	if err.Error() != "mock files directory not configured" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestLoadMockFileContent_FileNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-mock-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	builder := createTestBuilder("", tempDir)

	_, err = builder.loadMockFileContent("nonexistent.go")
	if err == nil {
		t.Error("Expected error when file doesn't exist")
	}
}


