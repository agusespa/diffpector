//go:build ignore

package file

import (
	"fmt"
	"io"
	"os"
)

type ProcessResult struct {
	LinesProcessed int
	WordCount      int
	Errors         []string
}

// This function has proper error handling (before state)
func ProcessFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	processed := processData(data)

	if err := writeToCache(filename, processed); err != nil {
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	return nil
}

func processData(data []byte) []byte {
	// Simulate data processing
	return data
}

func writeToCache(filename string, data []byte) error {
	// Simulate cache writing that could fail
	return nil
}

func sendToAnalytics(filename string, size int) error {
	// Simulate analytics call that could fail
	return nil
}
