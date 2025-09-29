package evaluation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agusespa/diffpector/internal/types"
)

// TestEnvironmentBuilder handles creation and management of test environments
type TestEnvironmentBuilder struct {
	suiteBaseDir string
	mockFilesDir string
}

func NewTestEnvironmentBuilder(suiteBaseDir, mockFilesDir string) *TestEnvironmentBuilder {
	return &TestEnvironmentBuilder{
		suiteBaseDir: suiteBaseDir,
		mockFilesDir: mockFilesDir,
	}
}

// CreateTestEnvironment creates a test environment from a test case
func (b *TestEnvironmentBuilder) CreateTestEnvironment(testCase types.TestCase) (*types.TestEnvironment, error) {
	if testCase.DiffFile == "" {
		return &types.TestEnvironment{
			Files: make(map[string]string),
			Diff:  "",
		}, nil
	}

	diffPath := filepath.Join(b.suiteBaseDir, testCase.DiffFile)
	diffContent, err := os.ReadFile(diffPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read diff file %s: %w", diffPath, err)
	}

	files, err := b.parseDiffToFiles(string(diffContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	return &types.TestEnvironment{
		Files: files,
		Diff:  string(diffContent),
	}, nil
}



func (b *TestEnvironmentBuilder) parseDiffToFiles(diff string) (map[string]string, error) {
	filenames := b.ExtractFilenamesFromDiff(diff)

	files := make(map[string]string)
	for _, filename := range filenames {
		content, err := b.loadMockFileContent(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to load mock content for %s: %w", filename, err)
		}
		files[filename] = content
	}

	return files, nil
}

func (b *TestEnvironmentBuilder) ExtractFilenamesFromDiff(diff string) []string {
	var filenames []string
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "+++") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				filename := strings.TrimPrefix(parts[1], "b/")
				if filename != "/dev/null" {
					filenames = append(filenames, filename)
				}
			}
		}
	}

	return filenames
}

func (b *TestEnvironmentBuilder) getAbsPath(filename string) string {
	return filepath.Join(b.mockFilesDir, filename)
}

func (b *TestEnvironmentBuilder) loadMockFileContent(filename string) (string, error) {
	if b.mockFilesDir == "" {
		return "", fmt.Errorf("mock files directory not configured")
	}

	fullPath := filepath.Join(b.mockFilesDir, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read mock file %s: %w", fullPath, err)
	}

	return string(content), nil
}




