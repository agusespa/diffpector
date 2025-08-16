//go:build ignore

package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type User struct {
	ID        int       `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	Role      string    `json:"role" db:"role"`
}

type UserRepository struct {
	conn   *sql.DB
	logger *log.Logger
}

func NewUserRepository(conn *sql.DB, logger *log.Logger) *UserRepository {
	return &UserRepository{
		conn:   conn,
		logger: logger,
	}
}

// GetUserByEmail retrieves a user by email address
func (r *UserRepository) GetUserByEmail(email string) (*User, error) {
	// Safe parameterized query (before state)
	query := "SELECT id, email, name FROM users WHERE email = ?"

	r.logger.Printf("Executing query: %s", query)

	row := r.conn.QueryRow(query, email)

	var user User
	err := row.Scan(&user.ID, &user.Email, &user.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Printf("Error scanning user: %v", err)
		return nil, err
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID (safer implementation)
func (r *UserRepository) GetUserByID(userID int) (*User, error) {
	query := "SELECT id, email, name, created_at, updated_at, is_active, role FROM users WHERE id = ? AND is_active = true"

	row := r.conn.QueryRow(query, userID)

	var user User
	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt, &user.IsActive, &user.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

// CreateUser creates a new user record
func (r *UserRepository) CreateUser(email, name, role string) (*User, error) {
	query := `
		INSERT INTO users (email, name, role, created_at, updated_at, is_active) 
		VALUES (?, ?, ?, ?, ?, true)
	`

	now := time.Now()
	result, err := r.conn.Exec(query, email, name, role, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return &User{
		ID:        int(id),
		Email:     email,
		Name:      name,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  true,
	}, nil
}

// UpdateUser updates an existing user
func (r *UserRepository) UpdateUser(userID int, name, role string) error {
	query := "UPDATE users SET name = ?, role = ?, updated_at = ? WHERE id = ?"

	_, err := r.conn.Exec(query, name, role, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser soft deletes a user (sets is_active to false)
func (r *UserRepository) DeleteUser(userID string) error {
	// Safe parameterized query (before state)
	query := "DELETE FROM users WHERE id = ?"

	_, err := r.conn.Exec(query, userID)
	return err
}

// GetActiveUsers retrieves all active users with pagination
func (r *UserRepository) GetActiveUsers(limit, offset int) ([]*User, error) {
	query := `
		SELECT id, email, name, created_at, updated_at, is_active, role 
		FROM users 
		WHERE is_active = true 
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?
	`

	rows, err := r.conn.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query active users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt, &user.IsActive, &user.Role)
		if err != nil {
			r.logger.Printf("Error scanning user row: %v", err)
			continue
		}
		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

// GetUsersByRole retrieves users by their role
func (r *UserRepository) GetUsersByRole(role string) ([]*User, error) {
	query := "SELECT id, email, name, created_at, updated_at, is_active, role FROM users WHERE role = ? AND is_active = true"

	rows, err := r.conn.Query(query, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		rows.Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt, &user.IsActive, &user.Role)
		users = append(users, &user)
	}

	return users, nil
}
