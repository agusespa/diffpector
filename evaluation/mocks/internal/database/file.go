//go:build ignore

package database

import (
	"database/sql"
	"fmt"
	"time"
)

type FileRecord struct {
	ID         int
	Filename   string
	OriginalName string
	Size       int64
	MimeType   string
	UploadedAt time.Time
	UserID     int
}

func (db *Database) SaveFileRecord(record *FileRecord) error {
	query := `INSERT INTO files (filename, original_name, size, mime_type, uploaded_at, user_id) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	
	result, err := db.conn.Exec(query, record.Filename, record.OriginalName, 
		record.Size, record.MimeType, record.UploadedAt, record.UserID)
	if err != nil {
		return fmt.Errorf("failed to save file record: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get file record ID: %w", err)
	}

	record.ID = int(id)
	return nil
}

func (db *Database) GetFileRecord(filename string) (*FileRecord, error) {
	query := `SELECT id, filename, original_name, size, mime_type, uploaded_at, user_id 
			  FROM files WHERE filename = ?`
	
	row := db.conn.QueryRow(query, filename)
	
	var record FileRecord
	err := row.Scan(&record.ID, &record.Filename, &record.OriginalName,
		&record.Size, &record.MimeType, &record.UploadedAt, &record.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("file record not found")
		}
		return nil, fmt.Errorf("failed to get file record: %w", err)
	}
	
	return &record, nil
}

func (db *Database) DeleteFileRecord(filename string) error {
	query := "DELETE FROM files WHERE filename = ?"
	_, err := db.conn.Exec(query, filename)
	if err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}
	return nil
}