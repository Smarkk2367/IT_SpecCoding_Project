package event

import "time"

const Version = "1.0"

type Envelope[T any] struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Payload   T         `json:"payload"`
}
