package entity

import (
	"sync"
	"time"
)

// NodeStatus represents the status of a node
type NodeStatus string

const (
	// NodeStatusAlive indicates the node is reachable and functioning
	NodeStatusAlive NodeStatus = "alive"

	// NodeStatusDead indicates the node is unreachable
	NodeStatusDead NodeStatus = "dead"

	// NodeStatusSuspect indicates the node is suspected to be unreachable
	NodeStatusSuspect NodeStatus = "suspect"
)

// Node represents a broker node in the cluster
type Node struct {
	// ID is the unique identifier for the node
	ID string

	// Address is the host:port of the node
	Address string

	// Status is the current status of the node
	Status NodeStatus

	// LastSeen is the timestamp when the node was last seen
	LastSeen time.Time
}

// NodeRegistry keeps track of all nodes in the cluster
type NodeRegistry struct {
	nodes map[string]*Node
	mutex sync.RWMutex
}

// NewNodeRegistry creates a new node registry
func NewNodeRegistry() *NodeRegistry {
	return &NodeRegistry{
		nodes: make(map[string]*Node),
	}
}

// RegisterNode adds a node to the registry
func (r *NodeRegistry) RegisterNode(id, address string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.nodes[id] = &Node{
		ID:       id,
		Address:  address,
		Status:   NodeStatusAlive,
		LastSeen: time.Now(),
	}
}

// UpdateNodeStatus updates the status of a node
func (r *NodeRegistry) UpdateNodeStatus(id string, status NodeStatus) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if node, exists := r.nodes[id]; exists {
		node.Status = status
		if status == NodeStatusAlive {
			node.LastSeen = time.Now()
		}
	}
}

// GetAliveNodes returns a list of all alive nodes
func (r *NodeRegistry) GetAliveNodes() []*Node {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var aliveNodes []*Node
	for _, node := range r.nodes {
		if node.Status == NodeStatusAlive {
			aliveNodes = append(aliveNodes, node)
		}
	}

	return aliveNodes
}

// GetNodeByID returns a node by its ID
func (r *NodeRegistry) GetNodeByID(id string) (*Node, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	node, exists := r.nodes[id]
	return node, exists
}

// GetNodeStatus returns the status of a node
func (r *NodeRegistry) GetNodeStatus(id string) NodeStatus {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if node, exists := r.nodes[id]; exists {
		return node.Status
	}

	return NodeStatusDead
}

// GetNodeCount returns the total number of nodes
func (r *NodeRegistry) GetNodeCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.nodes)
}

// GetAliveNodeCount returns the number of alive nodes
func (r *NodeRegistry) GetAliveNodeCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := 0
	for _, node := range r.nodes {
		if node.Status == NodeStatusAlive {
			count++
		}
	}

	return count
}
