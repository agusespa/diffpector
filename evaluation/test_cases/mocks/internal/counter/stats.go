package counter

import (
	"sync"
	"time"
)

type StatsCounter struct {
	mu    sync.Mutex
	count int64
	data  map[string]int
}

func NewStatsCounter() *StatsCounter {
	return &StatsCounter{
		data: make(map[string]int),
	}
}

// This function has a race condition - it reads and writes shared data without proper locking
func (s *StatsCounter) IncrementAsync(key string) {
	go func() {
		// Race condition: accessing shared data without mutex
		current := s.data[key]
		time.Sleep(time.Millisecond) // Simulate some work
		s.data[key] = current + 1
		s.count++
	}()
}

func (s *StatsCounter) GetCount() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

func (s *StatsCounter) GetData() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string]int)
	for k, v := range s.data {
		result[k] = v
	}
	return result
}