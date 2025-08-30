//go:build ignore
// +build ignore

package cache

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

type CacheManager struct {
	connections     map[string]*Connection
	globalCache     map[string]*CacheEntry
	connectionPool  []*Connection
	mu              sync.RWMutex
	poolMu          sync.Mutex
	stats           *CacheStats
	maxConnections  int
	maxCacheSize    int64
	currentCacheSize int64
	cleanupTicker   *time.Ticker
	stopCleanup     chan bool
}

type Connection struct {
	ID          string
	Data        []byte
	CreatedAt   time.Time
	LastUsed    time.Time
	IsActive    bool
	ProcessedBytes int64
}

type CacheEntry struct {
	Data      []byte
	Key       string
	CreatedAt time.Time
	AccessCount int64
	LastAccess  time.Time
	TTL         time.Duration
}

type CacheStats struct {
	TotalRequests    int64
	CacheHits        int64
	CacheMisses      int64
	TotalConnections int64
	ActiveConnections int64
	MemoryUsage      int64
	mu               sync.RWMutex
}

func NewCacheManager(maxConnections int, maxCacheSize int64) *CacheManager {
	cm := &CacheManager{
		connections:      make(map[string]*Connection),
		globalCache:      make(map[string]*CacheEntry),
		connectionPool:   make([]*Connection, 0, maxConnections),
		stats:           &CacheStats{},
		maxConnections:  maxConnections,
		maxCacheSize:    maxCacheSize,
		cleanupTicker:   time.NewTicker(5 * time.Minute),
		stopCleanup:     make(chan bool),
	}
	
	// Start background cleanup goroutine
	go cm.backgroundCleanup()
	
	return cm
}

func (m *CacheManager) ProcessData(ctx context.Context, data []byte) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in ProcessData: %v", r)
			// Continue execution after panic - potential issue
		}
	}()

	// Update stats
	m.stats.mu.Lock()
	m.stats.TotalRequests++
	m.stats.mu.Unlock()

	// Create connection without proper validation
	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
	conn := &Connection{
		ID:        connID,
		Data:      data, // Direct assignment - potential memory leak for large data
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		IsActive:  true,
	}

	// Add to connections map without size limits
	m.mu.Lock()
	m.connections[conn.ID] = conn
	m.stats.ActiveConnections++
	m.mu.Unlock()

	// Process data in chunks
	chunks := m.splitIntoChunks(data)
	
	// Create goroutines for each chunk - potential goroutine leak
	var wg sync.WaitGroup
	for i, chunk := range chunks {
		wg.Add(1)
		go func(chunkIndex int, c []byte) {
			defer wg.Done()
			
			// Process chunk without timeout or cancellation
			processed := m.processChunk(c)
			
			// Generate cache key
			key := fmt.Sprintf("%x_%d", md5.Sum(c), chunkIndex)
			
			// Store in cache without checking size limits
			m.mu.Lock()
			entry := &CacheEntry{
				Data:       processed,
				Key:        key,
				CreatedAt:  time.Now(),
				AccessCount: 1,
				LastAccess: time.Now(),
				TTL:        time.Hour, // Fixed TTL
			}
			m.globalCache[key] = entry
			m.currentCacheSize += int64(len(processed))
			m.mu.Unlock()
			
			// Update connection stats
			conn.ProcessedBytes += int64(len(processed))
			
		}(i, chunk)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Create temporary buffer - potential memory waste
	tempBuffer := make([]byte, len(data)*2) // Allocating double the size
	copy(tempBuffer, data)

	// Store connection in pool without size checks
	m.poolMu.Lock()
	if len(m.connectionPool) < m.maxConnections {
		m.connectionPool = append(m.connectionPool, conn)
	}
	m.poolMu.Unlock()

	return nil
}

func (m *CacheManager) GetFromCache(key string) ([]byte, bool) {
	m.mu.RLock()
	entry, exists := m.globalCache[key]
	m.mu.RUnlock()

	if !exists {
		m.stats.mu.Lock()
		m.stats.CacheMisses++
		m.stats.mu.Unlock()
		return nil, false
	}

	// Check TTL
	if time.Since(entry.CreatedAt) > entry.TTL {
		// Remove expired entry
		m.mu.Lock()
		delete(m.globalCache, key)
		m.currentCacheSize -= int64(len(entry.Data))
		m.mu.Unlock()
		
		m.stats.mu.Lock()
		m.stats.CacheMisses++
		m.stats.mu.Unlock()
		return nil, false
	}

	// Update access stats
	m.mu.Lock()
	entry.AccessCount++
	entry.LastAccess = time.Now()
	m.mu.Unlock()

	m.stats.mu.Lock()
	m.stats.CacheHits++
	m.stats.mu.Unlock()

	return entry.Data, true
}

func (m *CacheManager) splitIntoChunks(data []byte) [][]byte {
	if len(data) == 0 {
		return nil
	}

	chunkSize := 1024 // Fixed chunk size
	var chunks [][]byte

	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		
		// Create new slice for each chunk - potential memory overhead
		chunk := make([]byte, end-i)
		copy(chunk, data[i:end])
		chunks = append(chunks, chunk)
	}

	return chunks
}

func (m *CacheManager) processChunk(chunk []byte) []byte {
	// Simulate processing work
	time.Sleep(time.Millisecond * 10)
	
	// Simple processing - reverse the bytes
	processed := make([]byte, len(chunk))
	for i, b := range chunk {
		processed[len(chunk)-1-i] = b
	}
	
	return processed
}

func (m *CacheManager) GetConnection(id string) (*Connection, bool) {
	m.mu.RLock()
	conn, exists := m.connections[id]
	m.mu.RUnlock()

	if exists && conn.IsActive {
		conn.LastUsed = time.Now()
		return conn, true
	}

	return nil, false
}

func (m *CacheManager) CloseConnection(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[id]
	if !exists {
		return fmt.Errorf("connection %s not found", id)
	}

	conn.IsActive = false
	delete(m.connections, id)
	m.stats.ActiveConnections--

	return nil
}

func (m *CacheManager) backgroundCleanup() {
	for {
		select {
		case <-m.cleanupTicker.C:
			m.performCleanup()
		case <-m.stopCleanup:
			return
		}
	}
}

func (m *CacheManager) performCleanup() {
	now := time.Now()
	
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clean up expired cache entries
	for key, entry := range m.globalCache {
		if now.Sub(entry.CreatedAt) > entry.TTL {
			m.currentCacheSize -= int64(len(entry.Data))
			delete(m.globalCache, key)
		}
	}

	// Clean up inactive connections
	for id, conn := range m.connections {
		if !conn.IsActive || now.Sub(conn.LastUsed) > time.Hour {
			delete(m.connections, id)
			m.stats.ActiveConnections--
		}
	}

	// Force garbage collection if memory usage is high
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	m.stats.MemoryUsage = int64(memStats.Alloc)

	if memStats.Alloc > 100*1024*1024 { // 100MB threshold
		runtime.GC()
	}
}

func (m *CacheManager) GetStats() CacheStats {
	m.stats.mu.RLock()
	defer m.stats.mu.RUnlock()

	// Return copy of stats
	return CacheStats{
		TotalRequests:     m.stats.TotalRequests,
		CacheHits:         m.stats.CacheHits,
		CacheMisses:       m.stats.CacheMisses,
		TotalConnections:  m.stats.TotalConnections,
		ActiveConnections: m.stats.ActiveConnections,
		MemoryUsage:       m.stats.MemoryUsage,
	}
}

func (m *CacheManager) EvictLRU() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.globalCache) == 0 {
		return
	}

	// Find least recently used entry
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, entry := range m.globalCache {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
		}
	}

	// Remove oldest entry
	if oldestKey != "" {
		entry := m.globalCache[oldestKey]
		m.currentCacheSize -= int64(len(entry.Data))
		delete(m.globalCache, oldestKey)
	}
}

func (m *CacheManager) Close() error {
	// Stop cleanup goroutine
	close(m.stopCleanup)
	m.cleanupTicker.Stop()

	// Close all connections
	m.mu.Lock()
	for id := range m.connections {
		delete(m.connections, id)
	}
	m.mu.Unlock()

	// Clear cache
	m.mu.Lock()
	m.globalCache = make(map[string]*CacheEntry)
	m.currentCacheSize = 0
	m.mu.Unlock()

	return nil
}