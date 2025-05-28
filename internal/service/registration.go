package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func RegisterWithLoadBalancer(nodeID, port, rpcPort, loadBalancerURL, registerPath string) error {
	registerRequest := struct {
		ID      string `json:"id"`
		Port    string `json:"port"`
		RPCPort string `json:"rpcPort"`
	}{
		ID:      nodeID,
		Port:    port,
		RPCPort: rpcPort,
	}

	jsonData, err := json.Marshal(registerRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}

	resp, err := http.Post(loadBalancerURL+registerPath, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to register with load balancer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("load balancer returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}
