package rpc

import (
	"context"
	"fmt"
	myType "github.com/mar-coding/go-queue/loadbalancer/types"
	"net/rpc"
	"time"
)

// ClientRPC implements the Client interface
type ClientRPC struct {
	timeout time.Duration
}

// NewClient creates a new RPC client
func NewClient(timeout time.Duration) Client {
	return &ClientRPC{
		timeout: timeout,
	}
}

// Ping Implementation of interface methods...
func (c *ClientRPC) Ping(ctx context.Context, address string, msg string) error {
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to node: %w", err)
	}
	defer client.Close()

	var reply string
	err = client.Call("QueueService.Ping", msg, &reply)
	if err != nil {
		return fmt.Errorf("RPC Ping failed: %w", err)
	}

	return nil
}

// CreateQueue implements the ClientRPC interface
func (c *ClientRPC) CreateQueue(ctx context.Context, address string, req *myType.CreateQueueRequest) (*myType.CreateQueueResponse, error) {
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %w", err)
	}
	defer client.Close()

	var resp myType.CreateQueueResponse
	err = client.Call("QueueService.CreateQueue", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("RPC CreateQueue failed: %w", err)
	}

	return &resp, nil
}

// AppendData implements the ClientRPC interface
func (c *ClientRPC) AppendData(ctx context.Context, address string, req *myType.AppendDataRequest) (*myType.AppendDataResponse, error) {
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %w", err)
	}
	defer client.Close()

	var resp myType.AppendDataResponse
	err = client.Call("QueueService.AppendData", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("RPC AppendData failed: %w", err)
	}

	return &resp, nil
}

// ReadData implements the ClientRPC interface
func (c *ClientRPC) ReadData(ctx context.Context, address string, req *myType.ReadDataRequest) (*myType.ReadDataResponse, error) {
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %w", err)
	}
	defer client.Close()

	var resp myType.ReadDataResponse
	err = client.Call("QueueService.ReadData", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("RPC ReadData failed: %w", err)
	}

	return &resp, nil
}
