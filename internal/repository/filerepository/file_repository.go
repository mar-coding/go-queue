package filerepository

import (
	"context"
	"errors"
	"fmt"
	"github.com/mar-coding/go-queue/internal/entity"
	"github.com/mar-coding/go-queue/internal/pkg/filestorage"
	"github.com/mar-coding/go-queue/internal/repository"
	"sync"
)

type QueueRepository struct {
	queueStorage *filestorage.FileStorage
	mu           sync.Mutex
}

// NewQueueRepository creates a new QueueRepository
func NewQueueRepository(queueStorage *filestorage.FileStorage) *QueueRepository {
	return &QueueRepository{
		queueStorage: queueStorage,
	}
}

// Ensure QueueRepository implements repository.QueueRepository
var _ repository.QueueRepository = &QueueRepository{}

// CreateQueue creates a new queue
func (r *QueueRepository) CreateQueue(ctx context.Context, queue *entity.Queue) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// check if the queue already exists
	var existingQueue entity.Queue
	err := r.queueStorage.Load(queue.ID, &existingQueue)

	if err != nil {
		if !errors.Is(err, filestorage.ErrFileNotFound) {
			return fmt.Errorf("failed to check if queue exists: %w", err)
		}
	}

	if err == nil {
		return fmt.Errorf("queue with ID %s already exists", queue.ID)
	}

	// Save the queue data
	return r.queueStorage.Save(queue.ID, queue)
}

// GetQueue retrieves a queue by ID
func (r *QueueRepository) GetQueue(ctx context.Context, queueID string) (*entity.Queue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var queue entity.Queue
	err := r.queueStorage.Load(queueID, &queue)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve queue: %w", err)
	}

	return &queue, nil
}

// ListQueues lists all queues
func (r *QueueRepository) ListQueues(ctx context.Context) ([]*entity.Queue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// List the queue IDs
	queueIDs, err := r.queueStorage.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}

	// Load each queue from storage
	var queues []*entity.Queue
	for _, id := range queueIDs {
		var queue entity.Queue
		if err := r.queueStorage.Load(id, &queue); err == nil {
			queues = append(queues, &queue)
		}
	}

	return queues, nil
}

// AppendMessage adds a message to a queue
func (r *QueueRepository) AppendMessage(ctx context.Context, queueID string, message *entity.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load the queue
	var queue entity.Queue
	if err := r.queueStorage.Load(queueID, &queue); err != nil {
		return fmt.Errorf("failed to load queue: %w", err)
	}

	// Append the message to the queue
	queue.AppendMessage(message)

	// Save updated queue and message list
	if err := r.queueStorage.Save(queueID, &queue); err != nil {
		return fmt.Errorf("failed to save queue: %w", err)
	}

	return nil
}

// GetMessage retrieves a message by ID
func (r *QueueRepository) GetMessage(ctx context.Context, queueID, messageID string) (*entity.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load the queue
	var queue entity.Queue
	if err := r.queueStorage.Load(queueID, &queue); err != nil {
		return nil, fmt.Errorf("failed to load queue: %w", err)
	}

	messages := queue.GetMessages()

	// Find the message by ID
	for _, message := range messages {
		if message.ID == messageID {
			return message, nil
		}
	}

	return nil, fmt.Errorf("message not found")
}

// GetMessages retrieves all messages for a queue
func (r *QueueRepository) GetMessages(ctx context.Context, queueID string) ([]*entity.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load the queue
	var queue entity.Queue
	if err := r.queueStorage.Load(queueID, &queue); err != nil {
		return nil, fmt.Errorf("failed to load queue: %w", err)
	}

	return queue.GetMessages(), nil
}

// GetMessagesFrom retrieves messages starting from an index
func (r *QueueRepository) GetMessagesFrom(ctx context.Context, queueID string, fromIndex int) ([]*entity.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load the queue
	var queue entity.Queue
	if err := r.queueStorage.Load(queueID, &queue); err != nil {
		return nil, fmt.Errorf("failed to load queue: %w", err)
	}

	return queue.GetMessagesFrom(fromIndex), nil
}

// UpdateClientOffset updates the read offset for a client
func (r *QueueRepository) UpdateClientOffset(ctx context.Context, queueID, clientID string, offset int) error {
	// Load the queue
	var queue entity.Queue
	if err := r.queueStorage.Load(queueID, &queue); err != nil {
		return fmt.Errorf("failed to load queue: %w", err)
	}

	// Update the client's offset
	queue.SetClientOffset(clientID, offset)

	// Save the updated queue
	if err := r.queueStorage.Save(queueID, &queue); err != nil {
		return fmt.Errorf("failed to save queue: %w", err)
	}

	return nil
}

// GetClientOffset gets the read offset for a client
func (r *QueueRepository) GetClientOffset(ctx context.Context, queueID, clientID string) (int, error) {
	// Load the queue
	var queue entity.Queue
	if err := r.queueStorage.Load(queueID, &queue); err != nil {
		return 0, fmt.Errorf("failed to load queue: %w", err)
	}

	return queue.GetClientOffset(clientID), nil
}

// UpdateQueueReplicas updates the replica list for a queue
func (r *QueueRepository) UpdateQueueReplicas(ctx context.Context, queueID string, replicas []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load the queue
	var queue entity.Queue
	if err := r.queueStorage.Load(queueID, &queue); err != nil {
		return fmt.Errorf("failed to load queue: %w", err)
	}

	// Update replicas
	queue.Replicas = replicas

	// Save the updated queue
	return r.queueStorage.Save(queueID, &queue)
}

// GetQueuesByReplica gets all queues that have a specific node as a replica
func (r *QueueRepository) GetQueuesByReplica(ctx context.Context, nodeID string) ([]*entity.Queue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// List all queues and filter by the nodeID
	queues, err := r.ListQueues(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}

	var result []*entity.Queue
	for _, queue := range queues {
		for _, replica := range queue.Replicas {
			if replica == nodeID {
				result = append(result, queue)
				break
			}
		}
	}

	return result, nil
}

// ReadMessageForClient retrieves a message for a specific client and queue and updates the read offset
func (r *QueueRepository) ReadMessageForClient(ctx context.Context, queueID, clientID string) (*entity.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load the queue
	var queue entity.Queue
	if err := r.queueStorage.Load(queueID, &queue); err != nil {
		return nil, fmt.Errorf("failed to load queue: %w", err)
	}

	message, err := queue.ReadMessageForClient(clientID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, fmt.Errorf("no message available for client[%s]", clientID)
	}

	// Save the updated queue
	if err := r.queueStorage.Save(queueID, &queue); err != nil {
		return nil, fmt.Errorf("failed to save queue: %w", err)
	}

	return message, nil
}
