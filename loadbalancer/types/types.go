package types

type RegisterNodeRequest struct {
	ID      string `json:"id"`
	Port    string `json:"port"`
	RPCPort string `json:"rpcPort"`
}

type CreateQueueRequest struct {
	Name     string `json:"name"`
	ClientID string `json:"clientId"`
}

type AppendDataRequest struct {
	QueueID  string      `json:"queueId"`
	ClientID string      `json:"clientId"`
	Data     interface{} `json:"data"`
}

type ReadDataRequest struct {
	QueueID  string `json:"queueId"`
	ClientID string `json:"clientId"`
}

type RegisterNodeResponse struct {
	Status string `json:"status"`
	NodeID string `json:"node_id"`
}

type CreateQueueResponse struct {
	QueueID string `json:"queueId"`
}

type AppendDataResponse struct {
	MessageID string `json:"messageId"`
}

type ReadDataResponse struct {
	MessageID string      `json:"messageId"`
	Data      interface{} `json:"data"`
}

type StatusResponse struct {
	HealthyNodes int `json:"healthy_nodes"`
	TotalNodes   int `json:"total_nodes"`
}

type NodeStats struct {
	Address           string `json:"address"`
	Healthy           bool   `json:"healthy"`
	LastSeen          string `json:"last_seen"`
	ActiveConnections int64  `json:"active_connections"`
}

type StatsResponse map[string]NodeStats
