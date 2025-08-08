package cache

import (
	"database/sql"
	"fmt"
	"sync"
)

type CacheManager struct {
	cache map[string][]byte
	mu    sync.RWMutex
	db    *sql.DB
}

func NewCacheManager(db *sql.DB) *CacheManager {
	return &CacheManager{
		cache: make(map[string][]byte),
		db:    db,
	}
}

// This function has a memory leak - it stores large datasets in memory without cleanup
func (m *CacheManager) ProcessLargeDataset(data []byte) error {
	if m.db == nil {
		return fmt.Errorf("failed to connect to database: %w", fmt.Errorf("database connection is nil"))
	}

	// Memory leak: storing large data without any cleanup mechanism
	key := fmt.Sprintf("dataset_%d", len(data))
	
	m.mu.Lock()
	// This keeps growing without bounds
	m.cache[key] = make([]byte, len(data))
	copy(m.cache[key], data)
	m.mu.Unlock()

	// Process the data (simulate heavy computation)
	for i := 0; i < len(data); i++ {
		// Create more temporary allocations that aren't cleaned up
		temp := make([]byte, 1024)
		_ = temp
	}

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