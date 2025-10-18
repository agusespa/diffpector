//go:build ignore

package database

import (
	"database/sql"
	"fmt"
)

func (db *Database) UpdateUserProfile(user *User) error {
	query := "UPDATE users SET name = ?, bio = ? WHERE id = ?"
	_, err := db.conn.Exec(query, user.Name, user.Bio, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}
	return nil
}

func (db *Database) GetUserProfile(userID int) (*User, error) {
	query := "SELECT id, name, bio FROM users WHERE id = ?"
	row := db.conn.QueryRow(query, userID)
	
	var user User
	err := row.Scan(&user.ID, &user.Name, &user.Bio)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	
	return &user, nil
}