//go:build ignore

package database

import (
	"database/sql"
	"fmt"
)

type User struct {
	ID    int
	Email string
	Name  string
	Bio   string
}

type Database struct {
	conn *sql.DB
}

func NewDatabase(conn *sql.DB) *Database {
	return &Database{conn: conn}
}

func (db *Database) SearchUsers(name string) ([]*User, error) {
	query := fmt.Sprintf("SELECT id, email, name FROM users WHERE name = '%s'", name)
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}
