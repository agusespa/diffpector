//go:build ignore

package auth

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}

type AuthValidator struct {
	db *sql.DB
}

func NewAuthValidator(db *sql.DB) *AuthValidator {
	return &AuthValidator{db: db}
}

// ValidateUser securely validates user credentials (before state)
func (v *AuthValidator) ValidateUser(username, password string) (*User, error) {
	// Step 1: Hash the provided password for comparison
	hashedPassword, err := v.hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Step 2: Use parameterized query to prevent SQL injection
	query := "SELECT id, username, password_hash, role FROM users WHERE username = ?"
	row := v.db.QueryRow(query, username)

	var user User
	err = row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Step 3: Verify the password against stored hash
	if !v.verifyPassword(hashedPassword, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	return &user, nil
}

func (v *AuthValidator) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (v *AuthValidator) verifyPassword(hashedPassword, storedHash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(hashedPassword))
	return err == nil
}
