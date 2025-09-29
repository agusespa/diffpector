//go:build ignore

package analytics

import (
	"encoding/json"
	"time"
)

type EventProcessor struct {
	events []Event
}

type Event struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      string    `json:"data"`
}

func (p *EventProcessor) ProcessEvents(rawEvents []string) error {
	for _, rawEvent := range rawEvents {
		var event Event
		err := json.Unmarshal([]byte(rawEvent), &event)
		if err != nil {
			return err
		}
		p.events = append(p.events, event)
	}
	return nil
}

func (p *EventProcessor) GetEventsByUser(userID int) []Event {
	var userEvents []Event
	for _, event := range p.events {
		if event.UserID == userID {
			userEvents = append(userEvents, event)
		}
	}
	return userEvents
}
