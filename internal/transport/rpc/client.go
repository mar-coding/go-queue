package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an HTTP client for node-to-node communication
type Client struct {
	client  *http.Client
	timeout time.Duration
}

// NewClient creates a new RPC client
func NewClient(timeout time.Duration) *Client {
	return &Client{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Ping sends a ping to a node
func (c *Client) Ping(ctx context.Context, address string, queueID string) error {
	cmd := &Command{
		Type:    CommandTypePing,
		QueueID: queueID,
	}

	_, err := c.sendCommand(ctx, address, cmd)
	return err
}

// CreateQueue sends a request to create a queue on another node
func (c *Client) CreateQueue(ctx context.Context, address, queueID, queueName string, replicas []string) error {
	cmd := &Command{
		Type:      CommandTypeCreateQueue,
		QueueID:   queueID,
		QueueName: queueName,
		Replicas:  replicas,
	}

	_, err := c.sendCommand(ctx, address, cmd)
	return err
}

// AppendMessage sends a request to append a message to a queue on another node
func (c *Client) AppendMessage(ctx context.Context, address, queueID, messageID string, data []byte) error {
	cmd := &Command{
		Type:      CommandTypeAppendMessage,
		QueueID:   queueID,
		MessageID: messageID,
		Data:      data,
	}

	_, err := c.sendCommand(ctx, address, cmd)
	return err
}

// ReadMessage sends a request to read a message from a queue on another node
func (c *Client) ReadMessage(ctx context.Context, address, queueID, clientID string) (*MessageData, error) {
	cmd := &Command{
		Type:     CommandTypeReadMessage,
		QueueID:  queueID,
		ClientID: clientID,
	}

	resp, err := c.sendCommand(ctx, address, cmd)
	if err != nil {
		return nil, err
	}

	return &MessageData{
		ID:   resp.MessageID,
		Data: resp.Data,
	}, nil
}

// UpdateClientOffset sends a request to update a client's offset on another node
func (c *Client) UpdateClientOffset(ctx context.Context, address, queueID, clientID string, offset int) error {
	cmd := &Command{
		Type:     CommandTypeUpdateOffset,
		QueueID:  queueID,
		ClientID: clientID,
		Index:    offset,
	}

	_, err := c.sendCommand(ctx, address, cmd)
	return err
}

// sendCommand sends a command to another node and returns the response
func (c *Client) sendCommand(ctx context.Context, address string, cmd *Command) (*CommandResponse, error) {
	// Ensure address has http:// prefix
	if !strings.HasPrefix(address, "http://") {
		address = "http://" + address
	}

	// Marshal command to JSON
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", address+"/rpc", strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respData))
	}

	// Unmarshal response
	var cmdResp CommandResponse
	if err := json.Unmarshal(respData, &cmdResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for error in response
	if !cmdResp.Success {
		return nil, fmt.Errorf("command failed: %s", cmdResp.Error)
	}

	return &cmdResp, nil
}
