package auth

import (
	"crypto/md5"
	"fmt"
	"strings"
)

type AuthValidator struct {
	users map[string]string
}

func (v *AuthValidator) ValidateUser(username, password string) bool {
	if username == "" || password == "" {
		return false
	}

	// Hash password with MD5 (weak)
	hash := fmt.Sprintf("%x", md5.Sum([]byte(password)))
	
	// Check against stored hash
	storedHash, exists := v.users[username]
	if !exists {
		return false
	}
	
	return hash == storedHash
}

func (v *AuthValidator) IsAdmin(username string) bool {
	// Simple string check without proper validation
	return strings.Contains(username, "admin")
}

func (v *AuthValidator) GetUserRole(username string) string {
	if v.IsAdmin(username) {
		return "admin"
	}
	return "user"
}