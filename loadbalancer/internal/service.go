package internal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Node struct {
	ID                string
	Address           string
	Port              string
	Healthy           bool
	LastSeen          time.Time
	ActiveConnections int64
}

type Service struct {
	nodes         map[string]*Node
	mu            sync.RWMutex
	checkInterval time.Duration
	rpcClient     RPCClient
}

// RPCClient interface for minimal RPC operations
type RPCClient interface {
	Ping(ctx context.Context, address string, msg string) error
}

func NewService(checkInterval time.Duration, rpcClient RPCClient) *Service {
	s := &Service{
		nodes:         make(map[string]*Node),
		checkInterval: checkInterval,
		rpcClient:     rpcClient,
	}
	go s.startHealthCheck()
	return s
}

func (s *Service) startHealthCheck() {
	ticker := time.NewTicker(s.checkInterval)
	for range ticker.C {
		s.checkNodesHealth()
	}
}

func (s *Service) checkNodesHealth() {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, node := range s.nodes {
		// Ping the node
		err := s.rpcClient.Ping(ctx, node.Address+":"+node.Port, "")
		if err != nil {
			node.Healthy = false
		} else {
			node.Healthy = true
			node.LastSeen = time.Now()
		}
	}
}

// GetHealthyNode returns a healthy node using least connection strategy
func (s *Service) GetHealthyNode() (*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var healthyNodes []*Node
	for _, node := range s.nodes {
		if node.Healthy {
			healthyNodes = append(healthyNodes, node)
		}
	}

	if len(healthyNodes) == 0 {
		return nil, ErrNoHealthyNodes
	}

	// Implement least connections strategy
	var selectedNode *Node
	minConnections := int64(^uint64(0) >> 1)

	for _, node := range healthyNodes {
		connections := atomic.LoadInt64(&node.ActiveConnections)
		if connections < minConnections {
			minConnections = connections
			selectedNode = node
		}
	}

	// Check if we found a node
	if selectedNode == nil {
		return nil, ErrNoHealthyNodes
	}

	atomic.AddInt64(&selectedNode.ActiveConnections, 1)
	return selectedNode, nil
}

// RegisterNode registers a new node with the load balancer
func (s *Service) RegisterNode(id, address, port string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the node already exists
	if _, exists := s.nodes[id]; exists {
		return fmt.Errorf("node with ID %s already exists", id)
	}

	// Create a new node
	s.nodes[id] = &Node{
		ID:                id,
		Address:           address,
		Port:              port,
		Healthy:           true,
		LastSeen:          time.Now(),
		ActiveConnections: 0,
	}

	log.Printf("Registered new node: %s at %s:%s", id, address, port)
	return nil
}

// ReleaseNode decrements the active connection count for a node
func (s *Service) ReleaseNode(nodeID string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if node, exists := s.nodes[nodeID]; exists {
		atomic.AddInt64(&node.ActiveConnections, -1)
	}
}

// GetHealthyNodes returns a list of all healthy nodes
func (s *Service) GetHealthyNodes() []*Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var healthyNodes []*Node
	for _, node := range s.nodes {
		if node.Healthy {
			healthyNodes = append(healthyNodes, node)
		}
	}
	return healthyNodes
}

// GetAllNodes returns a list of all nodes regardless of health status
func (s *Service) GetAllNodes() []*Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]*Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetNodeStats returns detailed statistics about all nodes
func (s *Service) GetNodeStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	for id, node := range s.nodes {
		stats[id] = map[string]interface{}{
			"address":            node.Address + ":" + node.Port,
			"healthy":            node.Healthy,
			"last_seen":          node.LastSeen,
			"active_connections": atomic.LoadInt64(&node.ActiveConnections),
		}
	}
	return stats
}
