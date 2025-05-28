package main

import (
	"context"
	"flag"
	"github.com/mar-coding/go-queue/internal/transport/rpc"
	config "github.com/mar-coding/go-queue/loadbalancer/config"
	lb "github.com/mar-coding/go-queue/loadbalancer/internal"
	"github.com/mar-coding/go-queue/loadbalancer/internal/middleware"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	configPath := flag.String("config", "config/loadbalancer.json", "Path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup RPC client for health checks
	rpcClient := rpc.NewClient(cfg.NodeTimeout)

	// Setup load balancer service
	lbService := lb.NewService(cfg.HealthCheckInterval, rpcClient)

	// Setup HTTP handler with logging middleware
	handler := lb.NewHandler(lbService)

	// Create server with middleware
	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      middleware.LoggingMiddleware(handler.SetupRoutes()),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.ReadTimeout,
	}

	// Start HTTP server
	go func() {
		log.Printf("Starting load balancer HTTP server on port %s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down load balancer...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Gracefully shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Load balancer forced to shutdown: %v", err)
	}

	log.Println("Load balancer exited properly")
}
