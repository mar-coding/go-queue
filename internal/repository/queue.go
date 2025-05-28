package repository

import (
	"context"
	"github.com/mar-coding/go-queue/internal/entity"
)

// QueueRepository defines the interface for queue storage
type QueueRepository interface {
	// CreateQueue creates a new queue
	CreateQueue(ctx context.Context, queue *entity.Queue) error

	// GetQueue retrieves a queue by ID
	GetQueue(ctx context.Context, queueID string) (*entity.Queue, error)

	// ListQueues lists all queues
	ListQueues(ctx context.Context) ([]*entity.Queue, error)

	// AppendMessage adds a message to a queue
	AppendMessage(ctx context.Context, queueID string, message *entity.Message) error

	// GetMessage retrieves a message by ID
	GetMessage(ctx context.Context, queueID, messageID string) (*entity.Message, error)

	// GetMessages retrieves all messages for a queue
	GetMessages(ctx context.Context, queueID string) ([]*entity.Message, error)

	// GetMessagesFrom retrieves messages starting from an index
	GetMessagesFrom(ctx context.Context, queueID string, fromIndex int) ([]*entity.Message, error)

	// ReadMessageForClient retrieves a message for a specific client and queue and updates the read offset
	ReadMessageForClient(ctx context.Context, queueID, clientID string) (*entity.Message, error)

	// UpdateClientOffset updates the read offset for a client
	UpdateClientOffset(ctx context.Context, queueID, clientID string, offset int) error

	// GetClientOffset gets the read offset for a client
	GetClientOffset(ctx context.Context, queueID, clientID string) (int, error)

	// UpdateQueueReplicas updates the replica list for a queue
	UpdateQueueReplicas(ctx context.Context, queueID string, replicas []string) error

	// GetQueuesByReplica gets all queues that have a specific node as a replica
	GetQueuesByReplica(ctx context.Context, nodeID string) ([]*entity.Queue, error)
}
