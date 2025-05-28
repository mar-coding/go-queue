package loadbalancer

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds the load balancer configuration
type Config struct {
	NodeID              string        `json:"nodeId"`
	HTTPPort            string        `json:"httpPort"`
	HealthCheckInterval time.Duration `json:"-"`
	NodeTimeout         time.Duration `json:"-"`
	ReadTimeout         time.Duration `json:"-"`
}

// configJSON is a temporary struct to unmarshal JSON into before processing time values
type configJSON struct {
	NodeID              string `json:"nodeId"`
	HTTPPort            string `json:"httpPort"`
	HealthCheckInterval string `json:"healthCheckInterval"`
	NodeTimeout         string `json:"nodeTimeout"`
	ReadTimeout         string `json:"readTimeout"`
}

// LoadConfig loads the load balancer configuration from a file
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
		NodeID:   jsonConfig.NodeID,
		HTTPPort: jsonConfig.HTTPPort,
	}

	// Parse string durations from JSON into time.Duration
	healthCheckInterval, err := time.ParseDuration(jsonConfig.HealthCheckInterval)
	if err != nil {
		healthCheckInterval = 10 * time.Second // default value
	}
	config.HealthCheckInterval = healthCheckInterval

	nodeTimeout, err := time.ParseDuration(jsonConfig.NodeTimeout)
	if err != nil {
		nodeTimeout = 5 * time.Second // default value
	}
	config.NodeTimeout = nodeTimeout

	readTimeout, err := time.ParseDuration(jsonConfig.ReadTimeout)
	if err != nil {
		readTimeout = 2 * time.Second // default value
	}
	config.ReadTimeout = readTimeout

	// Validate required fields
	if config.NodeID == "" {
		return nil, fmt.Errorf("nodeId is required")
	}

	if config.HTTPPort == "" {
		config.HTTPPort = "8080" // default port for load balancer
	}

	return config, nil
}
