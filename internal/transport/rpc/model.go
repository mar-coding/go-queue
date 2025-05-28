package rpc

// CommandType represents the type of a command
type CommandType string

const (
	// CommandTypePing is a ping command
	CommandTypePing CommandType = "ping"

	// CommandTypeCreateQueue is a command to create a queue
	CommandTypeCreateQueue CommandType = "createQueue"

	// CommandTypeAppendMessage is a command to append a message to a queue
	CommandTypeAppendMessage CommandType = "appendMessage"

	// CommandTypeReadMessage is a command to read a message from a queue
	CommandTypeReadMessage CommandType = "readMessage"

	// CommandTypeUpdateOffset is a command to update a client's offset
	CommandTypeUpdateOffset CommandType = "updateOffset"
)

// Command represents a command sent between nodes
type Command struct {
	// Type is the type of the command
	Type CommandType `json:"type"`

	// QueueID is the ID of the queue
	QueueID string `json:"queueId,omitempty"`

	// QueueName is the name of the queue
	QueueName string `json:"queueName,omitempty"`

	// MessageID is the ID of the message
	MessageID string `json:"messageId,omitempty"`

	// ClientID is the ID of the client
	ClientID string `json:"clientId,omitempty"`

	// Data is the message payload
	Data []byte `json:"data,omitempty"`

	// Index is a generic index field
	Index int `json:"index,omitempty"`

	// Replicas is a list of replica node IDs
	Replicas []string `json:"replicas,omitempty"`
}

// CommandResponse represents a response to a command
type CommandResponse struct {
	// Success indicates whether the command was successful
	Success bool `json:"success"`

	// Error contains an error message if the command failed
	Error string `json:"error,omitempty"`

	// MessageID is the ID of a message (for read responses)
	MessageID string `json:"messageId,omitempty"`

	// Data is the message payload (for read responses)
	Data []byte `json:"data,omitempty"`
}

// MessageData contains data about a message
type MessageData struct {
	// ID is the unique identifier for the message
	ID string

	// Data is the actual message payload
	Data []byte
}
