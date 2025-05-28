package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Server handles RPC requests from other nodes
type Server struct {
	server       *http.Server
	queueService interface {
		ProcessCommand(*Command) *CommandResponse
	}
}

// NewServer creates a new RPC server
func NewServer() *Server {
	return &Server{}
}

// Start starts the RPC server
func (s *Server) Start(port string, queueService interface {
	ProcessCommand(*Command) *CommandResponse
}) error {
	s.queueService = queueService

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", s.handleRPC)

	s.server = &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: mux,
	}

	log.Printf("Starting RPC server on port %s", port)
	return s.server.ListenAndServe()
}

// Stop stops the RPC server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}

	return nil
}

// handleRPC handles RPC requests
func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}

	// Unmarshal command
	var cmd Command
	if err := json.Unmarshal(body, &cmd); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal command: %v", err), http.StatusBadRequest)
		return
	}

	// Process command
	resp := s.queueService.ProcessCommand(&cmd)

	// Marshal response
	respData, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}
