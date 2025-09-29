//go:build ignore

package utils

import (
	"strings"
	"unicode"
)

func ProcessString(input string) string {
	if input == "" {
		return ""
	}

	result := strings.ToLower(input)
	result = strings.TrimSpace(result)
	return result
}

func IsValidName(name string) bool {
	if len(name) < 2 {
		return false
	}

	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' {
			return false
		}
	}
	return true
}
