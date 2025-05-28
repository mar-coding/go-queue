package service

import (
	"context"
	"fmt"
	"github.com/mar-coding/go-queue/internal/entity"
	"github.com/mar-coding/go-queue/internal/repository"
	"github.com/mar-coding/go-queue/internal/transport/rpc"
	"log"
	"sync"
	"time"

	"github.com/mar-coding/go-queue/config"

	"github.com/google/uuid"
)

// QueueService handles queue operations
type QueueService struct {
	config      *config.Config
	queueRepo   repository.QueueRepository
	rpcClient   *rpc.Client
	nodeService *NodeService
}

// NewQueueService creates a new queue service
func NewQueueService(config *config.Config, queueRepo repository.QueueRepository, rpcClient *rpc.Client, nodeService *NodeService) *QueueService {
	return &QueueService{
		config:      config,
		queueRepo:   queueRepo,
		rpcClient:   rpcClient,
		nodeService: nodeService,
	}
}

// GetAliveNodes returns a list of alive nodes
func (s *QueueService) GetAliveNodes() []*entity.Node {
	// Get all alive nodes from the node service
	aliveNodes := s.nodeService.nodeRegistry.GetAliveNodes()
	return aliveNodes
}

// CreateQueue creates a new queue
func (s *QueueService) CreateQueue(ctx context.Context, name string) (string, error) {
	queueID := uuid.New().String()

	// Select replica nodes
	var replicas []string

	// Always include self as a replica
	replicas = append(replicas, s.nodeService.GetNodeID())

	// Select random nodes for additional replicas
	if s.config.ReplicationFactor > 1 {
		randomNodes := s.nodeService.GetRandomAliveNodes(s.config.ReplicationFactor - 1)
		for _, node := range randomNodes {
			replicas = append(replicas, node.ID)
		}
	}

	// Create queue locally
	queue := entity.NewQueue(queueID, name, replicas)
	if err := s.queueRepo.CreateQueue(ctx, queue); err != nil {
		return "", fmt.Errorf("failed to create queue locally: %w", err)
	}

	// Send queue creation requests to other replicas
	var wg sync.WaitGroup
	errorChan := make(chan error, len(replicas)-1)

	for _, replicaID := range replicas {
		if replicaID == s.nodeService.GetNodeID() {
			// Skip self
			continue
		}

		wg.Add(1)
		go func(nodeID string) {
			defer wg.Done()

			address, exists := s.nodeService.GetNodeAddress(nodeID)
			if !exists {
				errorChan <- fmt.Errorf("node %s not found", nodeID)
				return
			}

			err := s.rpcClient.CreateQueue(ctx, address, queueID, name, replicas)
			if err != nil {
				errorChan <- fmt.Errorf("failed to create queue on node %s: %w", nodeID, err)
			}
		}(replicaID)
	}

	// Wait for all replicas to be created
	wg.Wait()
	close(errorChan)

	// Check for errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// Log errors but still return success if at least one replica was created
		for _, err := range errors {
			log.Printf("Error creating queue replica: %v", err)
		}

		// If all replicas failed, return error
		if len(errors) == len(replicas)-1 {
			return "", fmt.Errorf("failed to create queue on any replica nodes")
		}
	}

	return queueID, nil
}

// AppendMessage appends a message to a queue
func (s *QueueService) AppendMessage(ctx context.Context, queueID, clientID string, data []byte) (string, error) {
	// Check if the queue exists locally
	queue, err := s.queueRepo.GetQueue(ctx, queueID)

	if err != nil {
		// Queue doesn't exist locally, forward to a replica node
		return s.forwardAppendMessage(ctx, queueID, clientID, data)
	}

	// Queue exists locally, append the message
	messageID := uuid.New().String()
	index := len(queue.Messages)

	message := entity.NewMessage(messageID, queueID, index, data)

	if err := s.queueRepo.AppendMessage(ctx, queueID, message); err != nil {
		return "", fmt.Errorf("failed to append message: %w", err)
	}

	// Forward the message to other replicas
	go s.replicateMessage(queueID, messageID, data, queue.Replicas)

	return messageID, nil
}

// forwardAppendMessage forwards a message append request to a replica node
func (s *QueueService) forwardAppendMessage(ctx context.Context, queueID, clientID string, data []byte) (string, error) {
	// Check all alive nodes for the queue
	aliveNodes := s.GetAliveNodes()
	var targetNodeID string

	// Try each alive node
	for _, node := range aliveNodes {
		if node.ID == s.nodeService.GetNodeID() {
			continue // Skip self
		}

		// Try to get queue info from this node
		address := node.Address
		// Make a lightweight RPC call to check if queue exists on this node
		err := s.rpcClient.Ping(ctx, address, queueID)
		if err == nil {
			targetNodeID = node.ID
			break
		}
	}

	if targetNodeID == "" {
		return "", fmt.Errorf("queue %s not found on any alive nodes", queueID)
	}

	// Forward to the target node
	address, exists := s.nodeService.GetNodeAddress(targetNodeID)
	if !exists {
		return "", fmt.Errorf("node %s not found", targetNodeID)
	}

	// We need to generate a message ID as the AppendMessage RPC method doesn't return one
	messageID := uuid.New().String()
	err := s.rpcClient.AppendMessage(ctx, address, queueID, messageID, data)
	if err != nil {
		return "", err
	}

	return messageID, nil
}

// replicateMessage replicates a message to all replica nodes
func (s *QueueService) replicateMessage(queueID, messageID string, data []byte, replicas []string) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.NodeTimeout)
	defer cancel()

	var wg sync.WaitGroup

	for _, replicaID := range replicas {
		if replicaID == s.nodeService.GetNodeID() {
			// Skip self
			continue
		}

		if !s.nodeService.IsNodeAlive(replicaID) {
			log.Printf("Skipping replication to dead node %s", replicaID)
			continue
		}

		wg.Add(1)
		go func(nodeID string) {
			defer wg.Done()

			address, exists := s.nodeService.GetNodeAddress(nodeID)
			if !exists {
				log.Printf("Node %s not found for replication", nodeID)
				return
			}

			err := s.rpcClient.AppendMessage(ctx, address, queueID, messageID, data)
			if err != nil {
				log.Printf("Failed to replicate message to node %s: %v", nodeID, err)
			}
		}(replicaID)
	}

	wg.Wait()
}

// ReadMessage reads a message from a queue for a specific client
func (s *QueueService) ReadMessage(ctx context.Context, queueID, clientID string) (*entity.Message, error) {
	// Check if the queue exists locally
	queue, err := s.queueRepo.GetQueue(ctx, queueID)

	if err != nil {
		// Queue doesn't exist locally, forward to a replica node
		return s.forwardReadMessage(ctx, queueID, clientID)
	}

	// Queue exists locally, read the message based on client offset
	message, err := s.queueRepo.ReadMessageForClient(ctx, queueID, clientID)
	if err != nil {
		return nil, err
	}

	clientOffset, err := s.queueRepo.GetClientOffset(ctx, queueID, clientID)
	if err != nil {
		return nil, err
	}

	// Update client offset in other replicas
	go s.syncClientOffset(queueID, clientID, clientOffset, queue.Replicas)

	return message, nil
}

// forwardReadMessage forwards a message read request to a replica node
func (s *QueueService) forwardReadMessage(ctx context.Context, queueID, clientID string) (*entity.Message, error) {
	// Check all alive nodes for the queue
	aliveNodes := s.GetAliveNodes()
	var targetNodeID string

	// Try each alive node
	for _, node := range aliveNodes {
		if node.ID == s.nodeService.GetNodeID() {
			continue // Skip self
		}

		// Try to get queue info from this node
		address := node.Address
		// Make a lightweight RPC call to check if queue exists on this node
		err := s.rpcClient.Ping(ctx, address, queueID)
		if err == nil {
			targetNodeID = node.ID
			break
		}
	}

	if targetNodeID == "" {
		return nil, fmt.Errorf("queue %s not found or no alive replicas", queueID)
	}

	// Forward to the target node
	address, exists := s.nodeService.GetNodeAddress(targetNodeID)
	if !exists {
		return nil, fmt.Errorf("node %s not found", targetNodeID)
	}

	msgData, err := s.rpcClient.ReadMessage(ctx, address, queueID, clientID)
	if err != nil {
		return nil, err
	}

	// Convert to a message entity
	message := &entity.Message{
		ID:      msgData.ID,
		QueueID: queueID,
		Data:    msgData.Data,
	}

	return message, nil
}

// syncClientOffset syncs the client offset to all replica nodes
func (s *QueueService) syncClientOffset(queueID, clientID string, offset int, replicas []string) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.NodeTimeout)
	defer cancel()

	var wg sync.WaitGroup

	for _, replicaID := range replicas {
		if replicaID == s.nodeService.GetNodeID() {
			// Skip self
			continue
		}

		if !s.nodeService.IsNodeAlive(replicaID) {
			log.Printf("Skipping client offset sync to dead node %s", replicaID)
			continue
		}

		wg.Add(1)
		go func(nodeID string) {
			defer wg.Done()

			address, exists := s.nodeService.GetNodeAddress(nodeID)
			if !exists {
				log.Printf("Node %s not found for offset sync", nodeID)
				return
			}

			err := s.rpcClient.UpdateClientOffset(ctx, address, queueID, clientID, offset)
			if err != nil {
				log.Printf("Failed to sync client offset to node %s: %v", nodeID, err)
			}
		}(replicaID)
	}

	wg.Wait()
}

// GetQueueInfo gets information about a queue
func (s *QueueService) GetQueueInfo(ctx context.Context, queueID string) (map[string]interface{}, error) {
	queue, err := s.queueRepo.GetQueue(ctx, queueID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":           queue.ID,
		"name":         queue.Name,
		"messageCount": len(queue.Messages),
		"replicas":     queue.Replicas,
		"createdAt":    queue.CreatedAt,
		"updatedAt":    queue.LastUpdatedAt,
	}, nil
}

// GetNodeStatus returns the status of the node
func (s *QueueService) GetNodeStatus() map[string]interface{} {
	// Get status from node service
	nodeStatus := s.nodeService.GetStatus()

	// Get queue count
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	queues, err := s.queueRepo.ListQueues(ctx)
	queueCount := 0
	if err == nil {
		queueCount = len(queues)
	}

	// Merge with queue information
	nodeStatus["queueCount"] = queueCount

	return nodeStatus
}

// ProcessCommand handles internal commands from other nodes
func (s *QueueService) ProcessCommand(cmd *rpc.Command) *rpc.CommandResponse {
	ctx := context.Background()
	resp := &rpc.CommandResponse{Success: true}

	switch cmd.Type {
	case rpc.CommandTypePing:
		// if queue id is not empty, check if the queue exists
		if cmd.QueueID != "" {
			_, err := s.queueRepo.GetQueue(ctx, cmd.QueueID)
			if err != nil {
				resp.Success = false
				resp.Error = err.Error()
			}
		}
		// Respond with success
		resp.Success = true
		resp.Error = ""

	case rpc.CommandTypeCreateQueue:
		// Create a queue locally
		queue := entity.NewQueue(cmd.QueueID, cmd.QueueName, cmd.Replicas)
		err := s.queueRepo.CreateQueue(ctx, queue)
		if err != nil {
			resp.Success = false
			resp.Error = err.Error()
		}

	case rpc.CommandTypeAppendMessage:
		// Add a message to a queue
		message := entity.NewMessage(cmd.MessageID, cmd.QueueID, cmd.Index, cmd.Data)
		err := s.queueRepo.AppendMessage(ctx, cmd.QueueID, message)
		if err != nil {
			resp.Success = false
			resp.Error = err.Error()
		}

	case rpc.CommandTypeReadMessage:
		message, err := s.queueRepo.ReadMessageForClient(ctx, cmd.QueueID, cmd.ClientID)
		if err != nil {
			resp.Success = false
			resp.Error = err.Error()
			break
		}

		resp.MessageID = message.ID
		resp.Data = message.Data

	case rpc.CommandTypeUpdateOffset:
		// Update client offset
		err := s.queueRepo.UpdateClientOffset(ctx, cmd.QueueID, cmd.ClientID, cmd.Index)
		if err != nil {
			resp.Success = false
			resp.Error = err.Error()
		}

	default:
		resp.Success = false
		resp.Error = "Unknown command type"
	}

	return resp
}
