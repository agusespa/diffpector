package spinner

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Spinner struct {
	chars    []string
	delay    time.Duration
	message  string
	active   bool
	mu       sync.Mutex
	stopChan chan bool
}

func New(message string) *Spinner {
	return &Spinner{
		chars:    []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		delay:    100 * time.Millisecond,
		message:  message,
		stopChan: make(chan bool, 1),
	}
}

func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		i := 0
		for {
			select {
			case <-s.stopChan:
				return
			default:
				s.mu.Lock()
				if !s.active {
					s.mu.Unlock()
					return
				}
				fmt.Printf("\r%s %s", s.chars[i%len(s.chars)], s.message)
				s.mu.Unlock()
				i++
				time.Sleep(s.delay)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	s.stopChan <- true
	
	// Clear the spinner line completely and move to next line
	fmt.Print("\r" + strings.Repeat(" ", len(s.message)+10) + "\r")
}

func (s *Spinner) Update(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}