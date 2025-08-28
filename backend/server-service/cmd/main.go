package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"server-service/internal/config"
	"server-service/internal/gateway"
	"server-service/internal/grpc_clients"
	"server-service/internal/health"
	"server-service/internal/redis_client"
	"server-service/internal/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting API Gateway on port %s", cfg.Server.Port)

	// Initialize Redis client for real-time events (continue on failure for testing)
	redisClient, err := redis_client.New(cfg.Redis)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		log.Println("Continuing without Redis (real-time events disabled)")
		// Create a dummy Redis client for testing
		redisClient = nil
	} else {
		defer redisClient.Close()
	}

	// Initialize gRPC clients to backend services (continue on failure for testing)
	grpcClients, err := grpc_clients.NewClientPool(cfg.Services)
	if err != nil {
		log.Printf("Warning: Failed to initialize gRPC clients: %v", err)
		log.Println("Continuing without backend services (API calls will return errors)")
		// Create a dummy gRPC client pool for testing
		grpcClients = nil
	} else {
		defer grpcClients.Close()
	}

	// Initialize WebSocket hub for real-time connections
	wsHub := websocket.NewHub(redisClient) // Can handle nil redisClient
	go wsHub.Run()

	// Get health manager from WebSocket hub
	healthManager := wsHub.GetHealthManager()

	// Initialize gRPC health checker
	grpcHealthChecker := health.NewGRPCHealthChecker()

	// Add auth service to health monitoring
	if err := grpcHealthChecker.AddService("auth", "localhost:9000", 30*time.Second); err != nil {
		log.Printf("Warning: Failed to add auth service to health monitoring: %v", err)
	}

	// Set up health change callback to update WebSocket clients
	grpcHealthChecker.OnHealthChange(func(service string, healthStatus health.ServiceHealth) {
		wsHealthStatus := websocket.HealthStatus{
			Service:   service,
			Status:    healthStatus.Status,
			Timestamp: healthStatus.LastChecked,
			Details:   healthStatus.Details,
		}
		healthManager.UpdateHealth(service, wsHealthStatus)
	})

	// Initialize Redis health subscriber if Redis is available
	if redisClient != nil {
		redisHealthSub, err := health.NewRedisHealthSubscriber(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			log.Printf("Warning: Failed to initialize Redis health subscriber: %v", err)
		} else {
			defer redisHealthSub.Close()

			// Set up Redis health change callback
			redisHealthSub.OnHealthChange(func(service string, healthStatus health.ServiceHealth) {
				wsHealthStatus := websocket.HealthStatus{
					Service:   service,
					Status:    healthStatus.Status,
					Timestamp: healthStatus.LastChecked,
					Details:   healthStatus.Details,
				}
				healthManager.UpdateHealth(service, wsHealthStatus)
			})
		}
	}

	// Initialize API Gateway with HTTP/2 support
	apiGateway := gateway.New(&gateway.Config{
		ServerConfig: &cfg.Server,
		GRPCClients:  grpcClients,
		WebSocketHub: wsHub,
		RedisClient:  redisClient,
	})

	// Start WebSocket server on port 8002 (HTTP/1.1 for WebSocket upgrade)
	go func() {
		log.Println("Starting WebSocket server on port 8002 (HTTP/1.1)")
		if err := apiGateway.StartWebSocketServer(8002); err != nil {
			log.Printf("WebSocket server failed: %v", err)
		}
	}()

	// Start main HTTP/2 server on port 8000
	go func() {
		log.Println("Starting HTTP/2 API server on port 8000")
		if err := apiGateway.Start(); err != nil {
			log.Fatalf("HTTP/2 server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API Gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiGateway.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("API Gateway stopped")
}
