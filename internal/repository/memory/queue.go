package memory

import (
	"context"
	"fmt"
	"github.com/mar-coding/go-queue/internal/entity"
	"github.com/mar-coding/go-queue/internal/repository"
	"sync"
)

// QueueRepository implements the QueueRepository interface using in-memory storage
type QueueRepository struct {
	queues map[string]*entity.Queue
	mutex  sync.RWMutex
}

// Ensure QueueRepository implements repository.QueueRepository
var _ repository.QueueRepository = &QueueRepository{}

// NewQueueRepository creates a new in-memory queue repository
func NewQueueRepository() *QueueRepository {
	return &QueueRepository{
		queues: make(map[string]*entity.Queue),
	}
}

// CreateQueue creates a new queue
func (r *QueueRepository) CreateQueue(ctx context.Context, queue *entity.Queue) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.queues[queue.ID]; exists {
		return fmt.Errorf("queue with ID %s already exists", queue.ID)
	}

	r.queues[queue.ID] = queue
	return nil
}

// GetQueue retrieves a queue by ID
func (r *QueueRepository) GetQueue(ctx context.Context, queueID string) (*entity.Queue, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	queue, exists := r.queues[queueID]
	if !exists {
		return nil, fmt.Errorf("queue with ID %s not found", queueID)
	}

	return queue, nil
}

// ListQueues lists all queues
func (r *QueueRepository) ListQueues(ctx context.Context) ([]*entity.Queue, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	queues := make([]*entity.Queue, 0, len(r.queues))
	for _, queue := range r.queues {
		queues = append(queues, queue)
	}

	return queues, nil
}

// AppendMessage adds a message to a queue
func (r *QueueRepository) AppendMessage(ctx context.Context, queueID string, message *entity.Message) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	queue, exists := r.queues[queueID]
	if !exists {
		return fmt.Errorf("queue with ID %s not found", queueID)
	}

	queue.AppendMessage(message)
	return nil
}

// GetMessage retrieves a message by ID
func (r *QueueRepository) GetMessage(ctx context.Context, queueID, messageID string) (*entity.Message, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	queue, exists := r.queues[queueID]
	if !exists {
		return nil, fmt.Errorf("queue with ID %s not found", queueID)
	}

	for _, message := range queue.Messages {
		if message.ID == messageID {
			return message, nil
		}
	}

	return nil, fmt.Errorf("message with ID %s not found in queue %s", messageID, queueID)
}

// GetMessages retrieves all messages for a queue
func (r *QueueRepository) GetMessages(ctx context.Context, queueID string) ([]*entity.Message, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	queue, exists := r.queues[queueID]
	if !exists {
		return nil, fmt.Errorf("queue with ID %s not found", queueID)
	}

	return queue.GetMessages(), nil
}

// GetMessagesFrom retrieves messages starting from an index
func (r *QueueRepository) GetMessagesFrom(ctx context.Context, queueID string, fromIndex int) ([]*entity.Message, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	queue, exists := r.queues[queueID]
	if !exists {
		return nil, fmt.Errorf("queue with ID %s not found", queueID)
	}

	return queue.GetMessagesFrom(fromIndex), nil
}

// UpdateClientOffset updates the read offset for a client
func (r *QueueRepository) UpdateClientOffset(ctx context.Context, queueID, clientID string, offset int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	queue, exists := r.queues[queueID]
	if !exists {
		return fmt.Errorf("queue with ID %s not found", queueID)
	}

	queue.SetClientOffset(clientID, offset)

	return nil
}

// GetClientOffset gets the read offset for a client
func (r *QueueRepository) GetClientOffset(ctx context.Context, queueID, clientID string) (int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	queue, exists := r.queues[queueID]
	if !exists {
		return 0, fmt.Errorf("queue with ID %s not found", queueID)
	}

	return queue.GetClientOffset(clientID), nil
}

// UpdateQueueReplicas updates the replica list for a queue
func (r *QueueRepository) UpdateQueueReplicas(ctx context.Context, queueID string, replicas []string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	queue, exists := r.queues[queueID]
	if !exists {
		return fmt.Errorf("queue with ID %s not found", queueID)
	}

	queue.Replicas = replicas
	return nil
}

// GetQueuesByReplica gets all queues that have a specific node as a replica
func (r *QueueRepository) GetQueuesByReplica(ctx context.Context, nodeID string) ([]*entity.Queue, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*entity.Queue
	for _, queue := range r.queues {
		if queue.HasReplica(nodeID) {
			result = append(result, queue)
		}
	}

	return result, nil
}

// ReadMessageForClient retrieves a message for a specific client and queue and updates the read offset
func (r *QueueRepository) ReadMessageForClient(ctx context.Context, queueID, clientID string) (*entity.Message, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	queue, exists := r.queues[queueID]
	if !exists {
		return nil, fmt.Errorf("queue with ID %s not found", queueID)
	}

	message, err := queue.ReadMessageForClient(clientID)
	if err != nil {
		return nil, err
	}

	return message, nil
}
