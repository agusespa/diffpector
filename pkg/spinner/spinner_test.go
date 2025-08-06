package spinner

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	message := "Testing spinner"
	s := New(message)

	if s.message != message {
		t.Errorf("Expected message %s, got %s", message, s.message)
	}

	if s.active {
		t.Error("Expected spinner to be inactive initially")
	}

	if len(s.chars) == 0 {
		t.Error("Expected spinner to have characters")
	}

	if s.delay == 0 {
		t.Error("Expected spinner to have delay")
	}

	if s.stopChan == nil {
		t.Error("Expected spinner to have stop channel")
	}
}

func TestSpinnerStartStop(t *testing.T) {
	s := New("Test message")

	// Test start
	s.Start()
	if !s.active {
		t.Error("Expected spinner to be active after start")
	}

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test stop
	s.Stop()
	if s.active {
		t.Error("Expected spinner to be inactive after stop")
	}
}

func TestSpinnerDoubleStart(t *testing.T) {
	s := New("Test message")

	s.Start()
	if !s.active {
		t.Error("Expected spinner to be active after first start")
	}

	// Starting again should not cause issues
	s.Start()
	if !s.active {
		t.Error("Expected spinner to still be active after second start")
	}

	s.Stop()
}

func TestSpinnerDoubleStop(t *testing.T) {
	s := New("Test message")

	s.Start()
	s.Stop()
	if s.active {
		t.Error("Expected spinner to be inactive after stop")
	}

	// Stopping again should not cause issues
	s.Stop()
	if s.active {
		t.Error("Expected spinner to still be inactive after second stop")
	}
}

func TestSpinnerUpdate(t *testing.T) {
	s := New("Initial message")
	newMessage := "Updated message"

	s.Update(newMessage)

	if s.message != newMessage {
		t.Errorf("Expected message %s, got %s", newMessage, s.message)
	}
}

func TestSpinnerUpdateWhileRunning(t *testing.T) {
	s := New("Initial message")
	newMessage := "Updated message"

	s.Start()
	s.Update(newMessage)

	if s.message != newMessage {
		t.Errorf("Expected message %s, got %s", newMessage, s.message)
	}

	s.Stop()
}