package rest

// CreateQueueRequest is the request for creating a queue
type CreateQueueRequest struct {
	Name string `json:"name"`
}

// CreateQueueResponse is the response for creating a queue
type CreateQueueResponse struct {
	QueueID string `json:"queueId"`
	Message string `json:"message"`
}

// AppendDataRequest is the request for appending data to a queue
type AppendDataRequest struct {
	QueueID  string `json:"queueId"`
	ClientID string `json:"clientId"`
	Data     string `json:"data"`
}

// AppendDataResponse is the response for appending data to a queue
type AppendDataResponse struct {
	MessageID string `json:"messageId"`
	Message   string `json:"message"`
}

// ReadDataResponse is the response for reading data from a queue
type ReadDataResponse struct {
	MessageID string `json:"messageId"`
	Data      string `json:"data"`
}

// NodeStatus represents the status of the node
type NodeStatus struct {
	NodeID      string `json:"nodeId"`
	Status      string `json:"status"`
	QueueCount  int    `json:"queueCount"`
	NodeCount   int    `json:"nodeCount"`
	AliveNodes  int    `json:"aliveNodes"`
	DeadNodes   int    `json:"deadNodes"`
	Version     string `json:"version"`
	UptimeHours int    `json:"uptimeHours"`
}
