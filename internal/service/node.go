package service

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/mar-coding/go-queue/config"
	"github.com/mar-coding/go-queue/internal/entity"
	"github.com/mar-coding/go-queue/internal/repository"
	"github.com/mar-coding/go-queue/internal/transport/rpc"
)

// NodeService handles node management and health checking
type NodeService struct {
	config       *config.Config
	nodeRegistry *entity.NodeRegistry
	queueRepo    repository.QueueRepository
	rpcClient    *rpc.Client
	startTime    time.Time
	mutex        sync.RWMutex
}

// NewNodeService creates a new node service
func NewNodeService(config *config.Config, queueRepo repository.QueueRepository, rpcClient *rpc.Client) *NodeService {
	nodeRegistry := entity.NewNodeRegistry()

	// Register self node
	selfAddress := "localhost:" + config.RPCPort
	nodeRegistry.RegisterNode(config.NodeID, selfAddress)

	// Register other nodes from config
	for i, nodeAddr := range config.Nodes {
		if nodeAddr == selfAddress {
			continue
		}

		nodeID := "node" + string(rune('1'+i))
		if nodeID == config.NodeID {
			nodeID = "node" + string(rune('2'+i))
		}

		nodeRegistry.RegisterNode(nodeID, nodeAddr)
	}

	return &NodeService{
		config:       config,
		nodeRegistry: nodeRegistry,
		queueRepo:    queueRepo,
		rpcClient:    rpcClient,
		startTime:    time.Now(),
	}
}

// StartHealthCheck starts the health check ticker
func (s *NodeService) StartHealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.checkNodeHealth()
	}
}

// checkNodeHealth pings all nodes to check their health
func (s *NodeService) checkNodeHealth() {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	nodes := s.nodeRegistry.GetAliveNodes()

	// Don't ping self
	for _, node := range nodes {
		if node.ID == s.config.NodeID {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), s.config.NodeTimeout)

		// Ping the node
		err := s.rpcClient.Ping(ctx, node.Address, "")

		if err != nil {
			log.Printf("Node %s (%s) is unreachable: %v", node.ID, node.Address, err)
			s.nodeRegistry.UpdateNodeStatus(node.ID, entity.NodeStatusDead)
			// Handle node failure - reassign queues
			go s.handleNodeFailure(node.ID)
		} else {
			// Node is alive
			s.nodeRegistry.UpdateNodeStatus(node.ID, entity.NodeStatusAlive)
		}

		cancel()
	}
}

// handleNodeFailure handles the failure of a node
func (s *NodeService) handleNodeFailure(nodeID string) {
	// Get all queues that have the failed node as replica
	ctx := context.Background()
	queues, err := s.queueRepo.GetQueuesByReplica(ctx, nodeID)
	if err != nil {
		log.Printf("Failed to get queues by replica for node %s: %v", nodeID, err)
		return
	}

	for _, queue := range queues {
		// Find a new node to replicate this queue
		newReplicaNode := s.findNewReplicaNode(queue.Replicas)
		if newReplicaNode == "" {
			log.Printf("No available nodes to replace failed node %s for queue %s", nodeID, queue.ID)
			continue
		}

		// Create new replicas list without the failed node
		newReplicas := make([]string, 0, len(queue.Replicas))
		for _, replica := range queue.Replicas {
			if replica != nodeID {
				newReplicas = append(newReplicas, replica)
			}
		}

		// Add the new replica
		newReplicas = append(newReplicas, newReplicaNode)

		// Update queue replicas
		err := s.queueRepo.UpdateQueueReplicas(ctx, queue.ID, newReplicas)
		if err != nil {
			log.Printf("Failed to update replicas for queue %s: %v", queue.ID, err)
			continue
		}

		// Sync the queue data to the new replica
		s.syncQueueToNewReplica(ctx, queue.ID, newReplicaNode)
	}
}

// findNewReplicaNode finds a new alive node that's not already a replica
func (s *NodeService) findNewReplicaNode(currentReplicas []string) string {
	aliveNodes := s.nodeRegistry.GetAliveNodes()

	// Create a map of current replicas for fast lookup
	replicaMap := make(map[string]bool)
	for _, replica := range currentReplicas {
		replicaMap[replica] = true
	}

	// Find nodes that are alive and not already replicas
	var candidates []string
	for _, node := range aliveNodes {
		if !replicaMap[node.ID] {
			candidates = append(candidates, node.ID)
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	// Pick a random node from the candidates
	newReplica := candidates[rand.Intn(len(candidates))]

	log.Printf("Selected new replica node: %s", newReplica)

	return newReplica
}

// syncQueueToNewReplica syncs queue data to a new replica
func (s *NodeService) syncQueueToNewReplica(ctx context.Context, queueID, nodeID string) {
	// Get the queue data
	queue, err := s.queueRepo.GetQueue(ctx, queueID)
	if err != nil {
		log.Printf("Failed to get queue %s for syncing: %v", queueID, err)
		return
	}

	// Get the node address
	node, exists := s.nodeRegistry.GetNodeByID(nodeID)
	if !exists {
		log.Printf("Node %s not found for syncing queue %s", nodeID, queueID)
		return
	}

	// Create the queue on the new replica
	err = s.rpcClient.CreateQueue(ctx, node.Address, queue.ID, queue.Name, queue.Replicas)
	if err != nil {
		log.Printf("Failed to create queue %s on node %s: %v", queueID, nodeID, err)
		return
	}

	// Send all messages to the new replica
	for _, message := range queue.Messages {
		err := s.rpcClient.ReplicateMessage(ctx, node.Address, message.QueueID, message.ID, message.Data, message.Index)
		if err != nil {
			log.Printf("Failed to sync message %s to node %s: %v", message.ID, nodeID, err)
			continue
		}
	}

	// Send client offsets to the new replica
	for clientID, offset := range queue.ClientOffsets {
		err := s.rpcClient.UpdateClientOffset(ctx, node.Address, queue.ID, clientID, offset)
		if err != nil {
			log.Printf("Failed to update offset for client %s on node %s: %v", clientID, nodeID, err)
			continue
		}
	}

	log.Printf("Queue %s successfully synced to node %s", queueID, nodeID)
}

// GetRandomAliveNodes returns a specified number of random alive nodes
func (s *NodeService) GetRandomAliveNodes(count int) []*entity.Node {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	aliveNodes := s.nodeRegistry.GetAliveNodes()

	// Remove self from the list
	filteredNodes := make([]*entity.Node, 0, len(aliveNodes))
	for _, node := range aliveNodes {
		if node.ID != s.config.NodeID {
			filteredNodes = append(filteredNodes, node)
		}
	}

	// Shuffle the nodes
	rand.Shuffle(len(filteredNodes), func(i, j int) {
		filteredNodes[i], filteredNodes[j] = filteredNodes[j], filteredNodes[i]
	})

	// Return at most count nodes
	if count > len(filteredNodes) {
		count = len(filteredNodes)
	}

	return filteredNodes[:count]
}

// GetStatus returns the current status of the node
func (s *NodeService) GetStatus() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	aliveCount := s.nodeRegistry.GetAliveNodeCount()
	totalCount := s.nodeRegistry.GetNodeCount()

	return map[string]interface{}{
		"nodeId":      s.config.NodeID,
		"aliveNodes":  aliveCount,
		"totalNodes":  totalCount,
		"deadNodes":   totalCount - aliveCount,
		"uptimeHours": int(time.Since(s.startTime).Hours()),
		"version":     "1.0.0",
	}
}

// GetNodeAddress returns the address of a node by ID
func (s *NodeService) GetNodeAddress(nodeID string) (string, bool) {
	node, exists := s.nodeRegistry.GetNodeByID(nodeID)
	if !exists {
		return "", false
	}

	return node.Address, true
}

// GetNodeID returns the ID of the current node
func (s *NodeService) GetNodeID() string {
	return s.config.NodeID
}

// IsNodeAlive checks if a node is alive
func (s *NodeService) IsNodeAlive(nodeID string) bool {
	return s.nodeRegistry.GetNodeStatus(nodeID) == entity.NodeStatusAlive
}
