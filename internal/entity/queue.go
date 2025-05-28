package entity

import (
	"fmt"
	"sync"
	"time"
)

// Queue represents a distributed queue
type Queue struct {
	// ID is the unique identifier for the queue
	ID string

	// Name is a human-readable name for the queue
	Name string

	// Replicas is a list of node IDs that have a replica of this queue
	Replicas []string

	// Messages is the ordered list of messages in the queue
	Messages []*Message

	// ClientOffsets tracks the read position for each client
	ClientOffsets map[string]int

	// CreatedAt is the timestamp when the queue was created
	CreatedAt time.Time

	// LastUpdatedAt is the timestamp when the queue was last updated
	LastUpdatedAt time.Time

	// mutex protects the queue from concurrent access
	mutex sync.RWMutex
}

// NewQueue creates a new queue
func NewQueue(id, name string, replicas []string) *Queue {
	now := time.Now()
	return &Queue{
		ID:            id,
		Name:          name,
		Replicas:      replicas,
		Messages:      make([]*Message, 0),
		ClientOffsets: make(map[string]int),
		CreatedAt:     now,
		LastUpdatedAt: now,
	}
}

// AppendMessage adds a message to the queue
func (q *Queue) AppendMessage(message *Message) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.Messages = append(q.Messages, message)
	q.LastUpdatedAt = time.Now()
}

// ReadMessageForClient reads a message for a specific client
func (q *Queue) ReadMessageForClient(clientID string) (*Message, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	offset, exists := q.ClientOffsets[clientID]
	if !exists {
		// If this is the first read for this client, start at the beginning
		offset = 0
		q.ClientOffsets[clientID] = offset
	}

	if offset >= len(q.Messages) {
		return nil, fmt.Errorf("no more messages in queue for client %s", clientID)
	}

	message := q.Messages[offset]

	// Update client offset
	q.ClientOffsets[clientID] = offset + 1

	return message, nil
}

// GetMessageCount returns the number of messages in the queue
func (q *Queue) GetMessageCount() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	return len(q.Messages)
}

// GetClientOffset returns the current read offset for a client
func (q *Queue) GetClientOffset(clientID string) int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	offset, exists := q.ClientOffsets[clientID]
	if !exists {
		return 0
	}

	return offset
}

// SetClientOffset sets the read offset for a client
func (q *Queue) SetClientOffset(clientID string, offset int) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.ClientOffsets[clientID] = offset
}

// HasReplica checks if a node is a replica for this queue
func (q *Queue) HasReplica(nodeID string) bool {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	for _, replica := range q.Replicas {
		if replica == nodeID {
			return true
		}
	}

	return false
}

// AddReplica adds a node as a replica for this queue
func (q *Queue) AddReplica(nodeID string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Check if node is already a replica
	for _, replica := range q.Replicas {
		if replica == nodeID {
			return
		}
	}

	q.Replicas = append(q.Replicas, nodeID)
	q.LastUpdatedAt = time.Now()
}

// RemoveReplica removes a node as a replica for this queue
func (q *Queue) RemoveReplica(nodeID string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	for i, replica := range q.Replicas {
		if replica == nodeID {
			q.Replicas = append(q.Replicas[:i], q.Replicas[i+1:]...)
			q.LastUpdatedAt = time.Now()
			return
		}
	}
}

// GetMessages returns a copy of all messages in the queue
func (q *Queue) GetMessages() []*Message {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	messagesCopy := make([]*Message, len(q.Messages))
	copy(messagesCopy, q.Messages)

	return messagesCopy
}

// GetMessagesFrom returns a copy of messages starting from a specific index
func (q *Queue) GetMessagesFrom(startIndex int) []*Message {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	if startIndex >= len(q.Messages) {
		return []*Message{}
	}

	messagesCopy := make([]*Message, len(q.Messages)-startIndex)
	copy(messagesCopy, q.Messages[startIndex:])

	return messagesCopy
}
