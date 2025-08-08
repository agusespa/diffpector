package file

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type ProcessResult struct {
	LinesProcessed int
	WordCount      int
	Errors         []string
}

// This function has missing error handling
func ProcessFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		
		// Missing error handling - this could fail
		processLine(line)
		
		// Missing error handling - this could also fail
		words := strings.Fields(line)
		for _, word := range words {
			validateWord(word)
		}
	}

	// Missing error handling for scanner errors
	fmt.Printf("Processed %d lines\n", lineCount)
	return nil
}

func processLine(line string) error {
	if len(line) > 1000 {
		return fmt.Errorf("line too long: %d characters", len(line))
	}
	// Simulate processing that could fail
	return nil
}

func validateWord(word string) error {
	if strings.Contains(word, "invalid") {
		return fmt.Errorf("invalid word found: %s", word)
	}
	return nil
}