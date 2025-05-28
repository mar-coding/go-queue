package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds the broker configuration
type Config struct {
	// NodeID is the unique identifier for this node
	NodeID string `json:"nodeId"`

	// HTTPPort is the port for the HTTP API
	HTTPPort string `json:"httpPort"`

	// RPCPort is the port for inter-node communication
	RPCPort string `json:"rpcPort"`

	// Nodes is a list of all nodes in the cluster (hostname:port)
	Nodes []string `json:"nodes"`

	// ReplicationFactor is the number of nodes to replicate each queue to
	ReplicationFactor int `json:"replicationFactor"`

	// HealthCheckInterval is the interval between node health checks
	HealthCheckInterval time.Duration

	// NodeTimeout is the timeout for node-to-node communication
	NodeTimeout time.Duration

	// ReadTimeout is the timeout for read operations
	ReadTimeout time.Duration
}

// configJSON is a temporary struct to unmarshal JSON into before processing time values
type configJSON struct {
	NodeID              string   `json:"nodeId"`
	HTTPPort            string   `json:"httpPort"`
	RPCPort             string   `json:"rpcPort"`
	Nodes               []string `json:"nodes"`
	ReplicationFactor   int      `json:"replicationFactor"`
	HealthCheckInterval string   `json:"healthCheckInterval"`
	NodeTimeout         string   `json:"nodeTimeout"`
	ReadTimeout         string   `json:"readTimeout"`
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// First, parse the JSON into our temporary struct
	var jsonConfig configJSON
	if err := json.NewDecoder(file).Decode(&jsonConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Create the actual config with the basic fields
	config := &Config{
		NodeID:            jsonConfig.NodeID,
		HTTPPort:          jsonConfig.HTTPPort,
		RPCPort:           jsonConfig.RPCPort,
		Nodes:             jsonConfig.Nodes,
		ReplicationFactor: jsonConfig.ReplicationFactor,
	}

	// Parse string durations from JSON into time.Duration
	healthCheckInterval, err := time.ParseDuration(jsonConfig.HealthCheckInterval)
	if err != nil {
		healthCheckInterval = 10 * time.Second
	}
	config.HealthCheckInterval = healthCheckInterval

	nodeTimeout, err := time.ParseDuration(jsonConfig.NodeTimeout)
	if err != nil {
		nodeTimeout = 5 * time.Second
	}
	config.NodeTimeout = nodeTimeout

	readTimeout, err := time.ParseDuration(jsonConfig.ReadTimeout)
	if err != nil {
		readTimeout = 2 * time.Second
	}
	config.ReadTimeout = readTimeout

	// Validate required fields
	if config.NodeID == "" {
		return nil, fmt.Errorf("nodeId is required")
	}

	if config.HTTPPort == "" {
		config.HTTPPort = "8000"
	}

	if config.RPCPort == "" {
		config.RPCPort = "8001"
	}

	if config.ReplicationFactor < 1 {
		return nil, fmt.Errorf("replicationFactor must be at least 1")
	}

	if len(config.Nodes) < config.ReplicationFactor {
		return nil, fmt.Errorf("not enough nodes for requested replication factor")
	}

	return config, nil
}
