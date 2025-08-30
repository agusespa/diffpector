//go:build ignore
// +build ignore

package helper

import (
	"fmt"
	"strings"
)

func FormatUserData(name, email string, age int) string {
	if name == "" {
		name = "Unknown"
	}
	if email == "" {
		email = "no-email@example.com"
	}
	
	return fmt.Sprintf("%s <%s> (age: %d)", name, email, age)
}

func ProcessNames(names []string) []string {
	var result []string
	for _, name := range names {
		processed := strings.TrimSpace(name)
		if processed != "" {
			result = append(result, processed)
		}
	}
	return result
}