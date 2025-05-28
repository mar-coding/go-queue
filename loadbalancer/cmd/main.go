package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "github.com/mar-coding/go-queue/loadbalancer/config"
	lb "github.com/mar-coding/go-queue/loadbalancer/internal"
	"github.com/mar-coding/go-queue/loadbalancer/internal/middleware"
	"github.com/mar-coding/go-queue/loadbalancer/internal/transport/rpc"
)

func main() {
	// Configure logging with timestamp and microsecond precision
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Parse command line flags
	configPath := flag.String("config", "config/loadbalancer.json", "Path to config file")
	flag.Parse()

	// Load and validate configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize components
	rpcClient := rpc.NewClient(cfg.NodeTimeout)
	lbService := lb.NewService(cfg.HealthCheckInterval, rpcClient)
	handler := lb.NewHandler(lbService)

	rpcServer := rpc.NewServer()
	go func() {
		log.Printf("Starting RPC server on port %s", cfg.RPCPort)
		if err := rpcServer.Start(":"+cfg.RPCPort, lbService); err != nil {
			log.Fatalf("Failed to start RPC server: %v", err)
		}
	}()

	// Configure an HTTP server with middleware and timeouts
	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      middleware.LoggingMiddleware(handler.SetupRoutes()),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start a server in a goroutine
	go startServer(server, cfg.HTTPPort)

	// Wait for a shutdown signal
	gracefulShutdown(server)
}

func startServer(server *http.Server, port string) {
	log.Printf("Starting load balancer HTTP server on port %s", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func gracefulShutdown(server *http.Server) {
	// Create channel for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a shutdown signal
	<-quit
	log.Println("Shutting down load balancer...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt a graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Load balancer forced to shutdown: %v", err)
	}

	log.Println("Load balancer exited properly")
}
