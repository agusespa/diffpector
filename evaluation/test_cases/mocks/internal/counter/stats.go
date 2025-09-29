//go:build ignore

package counter

import (
	"sync"
	"time"
)

type StatsCounter struct {
	mu    sync.RWMutex
	count int64
	stats map[string]int64
}

func (s *StatsCounter) Increment() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.count++
}

func (s *StatsCounter) GetCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

func (s *StatsCounter) UpdateStats() {
	for {
		time.Sleep(1 * time.Second)
		current := s.GetCount()
		s.stats["current"] = current
		s.stats["timestamp"] = time.Now().Unix()
	}
}
