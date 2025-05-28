package entity

import "time"

// Message represents a single message in a queue
type Message struct {
	// ID is the unique identifier for the message
	ID string

	// QueueID is the ID of the queue this message belongs to
	QueueID string

	// Index is the position of the message in the queue
	Index int

	// Data is the actual message payload
	Data []byte

	// CreatedAt is the timestamp when the message was created
	CreatedAt time.Time
}

// NewMessage creates a new message
func NewMessage(id, queueID string, index int, data []byte) *Message {
	return &Message{
		ID:        id,
		QueueID:   queueID,
		Index:     index,
		Data:      data,
		CreatedAt: time.Now(),
	}
}
