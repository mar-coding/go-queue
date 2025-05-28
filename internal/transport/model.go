package transport

import "github.com/mar-coding/go-queue/internal/transport/rpc"

// MessageData contains data about a message
type MessageData struct {
	// ID is the unique identifier for the message
	ID string

	// Data is the actual message payload
	Data []byte
}

// QueueInfo contains information about a queue
type QueueInfo struct {
	// ID is the unique identifier for the queue
	ID string

	// Name is a human-readable name for the queue
	Name string

	// MessageCount is the number of messages in the queue
	MessageCount int

	// Replicas is a list of node IDs that have a replica of this queue
	Replicas []string
}

// NodeInfo contains information about a node
type NodeInfo struct {
	// ID is the unique identifier for the node
	ID string

	// Address is the host:port of the node
	Address string

	// Status is the current status of the node
	Status string
}

// Command represents a command sent between nodes
type Command = rpc.Command

// CommandResponse represents a response to a command
type CommandResponse = rpc.CommandResponse

// CommandType represents the type of a command
type CommandType = rpc.CommandType
