//go:build ignore

package database

import (
	"database/sql"
	"fmt"
)

type UserResourcePermission struct {
	UserID       int
	ResourceID   string
	PermissionLevel int
}

func (db *Database) GetUserResourcePermission(userID int, resourceID string) (int, error) {
	query := "SELECT permission_level FROM user_permissions WHERE user_id = ? AND resource_id = ?"
	row := db.conn.QueryRow(query, userID, resourceID)
	
	var permissionLevel int
	err := row.Scan(&permissionLevel)
	if err != nil {
		if err == sql.ErrNoRows {
			// No explicit permission found, return read-only by default
			return 1, nil
		}
		return 0, fmt.Errorf("failed to get user permission: %w", err)
	}
	
	return permissionLevel, nil
}

func (db *Database) SetUserResourcePermission(userID int, resourceID string, level int) error {
	query := `INSERT OR REPLACE INTO user_permissions (user_id, resource_id, permission_level) 
			  VALUES (?, ?, ?)`
	
	_, err := db.conn.Exec(query, userID, resourceID, level)
	if err != nil {
		return fmt.Errorf("failed to set user permission: %w", err)
	}
	
	return nil
}