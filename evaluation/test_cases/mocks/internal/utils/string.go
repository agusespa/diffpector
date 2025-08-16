//go:build ignore

package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// StringValidator provides validation utilities for user input
type StringValidator struct {
	minLength int
	maxLength int
	patterns  map[string]*regexp.Regexp
}

// NewStringValidator creates a new validator with default settings
func NewStringValidator() *StringValidator {
	return &StringValidator{
		minLength: 1,
		maxLength: 255,
		patterns: map[string]*regexp.Regexp{
			"email":    regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
			"username": regexp.MustCompile(`^[a-zA-Z0-9_-]{3,20}$`),
			"phone":    regexp.MustCompile(`^\+?[1-9]\d{1,14}$`),
		},
	}
}

// ValidateEmail checks if the provided string is a valid email format
func (v *StringValidator) ValidateEmail(email string) error {
	if len(email) == 0 {
		return fmt.Errorf("email cannot be empty")
	}

	if len(email) > v.maxLength {
		return fmt.Errorf("email too long: maximum %d characters", v.maxLength)
	}

	if !v.patterns["email"].MatchString(email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// ValidateUsername checks if username meets requirements
func (v *StringValidator) ValidateUsername(username string) error {
	if len(username) < 3 {
		return fmt.Errorf("username too short: minimum 3 characters")
	}

	if len(username) > 20 {
		return fmt.Errorf("username too long: maximum 20 characters")
	}

	if !v.patterns["username"].MatchString(username) {
		return fmt.Errorf("username contains invalid characters")
	}

	return nil
}

// SanitizeInput removes potentially dangerous characters from user input
func SanitizeInput(input string) string {
	// Remove null bytes and control characters
	cleaned := strings.Map(func(r rune) rune {
		if r == 0 || (unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t') {
			return -1
		}
		return r
	}, input)

	// Trim whitespace
	cleaned = strings.TrimSpace(cleaned)

	// Remove potential SQL injection patterns (basic)
	dangerous := []string{"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_"}
	for _, pattern := range dangerous {
		cleaned = strings.ReplaceAll(cleaned, pattern, "")
	}

	return cleaned
}

// GenerateSecureToken creates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("token length must be positive")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// TruncateString safely truncates a string to the specified length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Try to break at word boundary
	if maxLen > 3 {
		truncated := s[:maxLen-3]
		if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLen/2 {
			return truncated[:lastSpace] + "..."
		}
	}

	return s[:maxLen-3] + "..."
}

// ParseCSV parses a simple CSV line (basic implementation)
func ParseCSV(line string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false

	for i, char := range line {
		switch char {
		case '"':
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				// Escaped quote
				current.WriteRune('"')
				i++ // Skip next quote
			} else {
				inQuotes = !inQuotes
			}
		case ',':
			if !inQuotes {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	result = append(result, current.String())
	return result
}

// HashPassword creates a simple hash of a password (NOT for production use)
func HashPassword(password string) string {
	// This is intentionally weak for testing purposes
	hash := 0
	for _, char := range password {
		hash = hash*31 + int(char)
	}
	return fmt.Sprintf("hash_%d", hash)
}

// Legacy functions for backward compatibility
func TrimAndLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func IsEmpty(s string) bool {
	return TrimAndLower(s) == ""
}
