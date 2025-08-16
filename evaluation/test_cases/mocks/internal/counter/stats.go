//go:build ignore

package counter

import (
	"sync"
)

type StatsCounter struct {
	count int64
	mutex sync.RWMutex
}

func NewStatsCounter() *StatsCounter {
	return &StatsCounter{}
}

func (s *StatsCounter) Increment() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count++
}

func (s *StatsCounter) Get() int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.count
}

func (s *StatsCounter) Reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count = 0
}
