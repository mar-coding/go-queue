package rpc

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
)

type Server struct {
	listener net.Listener
	server   *rpc.Server
}

func NewServer() *Server {
	return &Server{
		server: rpc.NewServer(),
	}
}

// Start starts the RPC server on the given address with the provided service
func (s *Server) Start(port string, service interface{}) error {
	// Register the service
	if err := s.server.RegisterName("NodeService", service); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// Create listener
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	log.Printf("RPC Server listening on %s", port)

	// Start accepting connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if the server was stopped
			if err, ok := err.(*net.OpError); ok && err.Op == "accept" {
				return nil
			}
			log.Printf("Accept error: %v", err)
			continue
		}
		go s.server.ServeConn(conn)
	}
}

// Stop gracefully stops the RPC server
func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}
