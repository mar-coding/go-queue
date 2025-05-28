package entity

import (
	"sync"
	"time"
)

// Client represents a client connected to the queue system
type Client struct {
	// ID is the unique identifier for the client
	ID string

	// LastActivity is the timestamp of the last client activity
	LastActivity time.Time

	// mutex protects the client data from concurrent access
	mutex sync.RWMutex
}

// NewClient creates a new client
func NewClient(id string) *Client {
	return &Client{
		ID:           id,
		LastActivity: time.Now(),
	}
}

// UpdateLastActivity updates the last activity time	stamp
func (c *Client) UpdateLastActivity() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.LastActivity = time.Now()
}

// ClientRegistry manages all clients connected to the system
type ClientRegistry struct {
	clients map[string]*Client
	mutex   sync.RWMutex
}

// NewClientRegistry creates a new client registry
func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients: make(map[string]*Client),
	}
}

// GetOrCreateClient gets an existing client or creates a new one
func (r *ClientRegistry) GetOrCreateClient(clientID string) *Client {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	client, exists := r.clients[clientID]
	if !exists {
		client = NewClient(clientID)
		r.clients[clientID] = client
	}

	return client
}

// GetClient gets a client by ID
func (r *ClientRegistry) GetClient(clientID string) (*Client, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	client, exists := r.clients[clientID]
	return client, exists
}

// RemoveClient removes a client from the registry
func (r *ClientRegistry) RemoveClient(clientID string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.clients, clientID)
}

// CleanupInactiveClients removes clients that have been inactive for a given duration
func (r *ClientRegistry) CleanupInactiveClients(threshold time.Duration) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	for id, client := range r.clients {
		if now.Sub(client.LastActivity) > threshold {
			delete(r.clients, id)
		}
	}
}
