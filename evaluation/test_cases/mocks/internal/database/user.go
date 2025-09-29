//go:build ignore

package database

import (
	"database/sql"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func GetUser(db *sql.DB, userID int) (*User, error) {
	query := "SELECT id, name, email FROM users WHERE id = ?"
	row := db.QueryRow(query, userID)

	var user User
	err := row.Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func CreateUser(db *sql.DB, name, email string) error {
	query := "INSERT INTO users (name, email) VALUES (?, ?)"
	_, err := db.Exec(query, name, email)
	return err
}
