//go:build ignore

package cache

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
)

type CacheManager struct {
	cache       map[string][]byte
	mu          sync.RWMutex
	db          *sql.DB
	globalCache map[string][]byte
}

func NewCacheManager(db *sql.DB) *CacheManager {
	return &CacheManager{
		cache: make(map[string][]byte),
		db:    db,
	}
}

// This function properly manages resources (before state)
func (m *CacheManager) ProcessLargeDataset(data []byte) error {
	conn, err := m.db.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close database connection: %v", err)
		}
	}()

	// Process data in chunks
	chunkSize := 1024
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := make([]byte, end-i)
		copy(chunk, data[i:end])

		if err := m.processChunk(conn, chunk); err != nil {
			return err
		}
	}

	return nil
}

func (m *CacheManager) processChunk(conn *sql.Conn, chunk []byte) error {
	// Simulate chunk processing
	return nil
}

func (m *CacheManager) GetCacheSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for _, data := range m.cache {
		total += len(data)
	}
	return total
}
