//go:build ignore

package helper

import (
	"fmt"
)

// formatUserName formats a user's name for display
func formatUserName(firstName, lastName string) string {
	if firstName == "" && lastName == "" {
		return "Anonymous"
	}

	if firstName == "" {
		return lastName
	}

	if lastName == "" {
		return firstName
	}

	return fmt.Sprintf("%s %s", firstName, lastName)
}

// Helper function with unclear name
func process(data interface{}) interface{} {
	return data
}
