package health

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Service     string
	Status      string // healthy, unhealthy, degraded
	LastChecked time.Time
	Details     string
}

// GRPCHealthChecker manages health checks for gRPC services
type GRPCHealthChecker struct {
	mu           sync.RWMutex
	services     map[string]*serviceConfig
	healthStatus map[string]ServiceHealth
	callbacks    []func(string, ServiceHealth)
}

type serviceConfig struct {
	name     string
	address  string
	interval time.Duration
	conn     *grpc.ClientConn
	client   grpc_health_v1.HealthClient
	cancel   context.CancelFunc
}

// NewGRPCHealthChecker creates a new gRPC health checker
func NewGRPCHealthChecker() *GRPCHealthChecker {
	return &GRPCHealthChecker{
		services:     make(map[string]*serviceConfig),
		healthStatus: make(map[string]ServiceHealth),
		callbacks:    make([]func(string, ServiceHealth), 0),
	}
}

// AddService adds a service to monitor
func (hc *GRPCHealthChecker) AddService(name, address string, interval time.Duration) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Create gRPC connection
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to %s at %s: %v", name, address, err)
	}

	client := grpc_health_v1.NewHealthClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	config := &serviceConfig{
		name:     name,
		address:  address,
		interval: interval,
		conn:     conn,
		client:   client,
		cancel:   cancel,
	}

	hc.services[name] = config

	// Start health checking goroutine
	go hc.healthCheckLoop(ctx, config)

	log.Printf("Started health monitoring for service %s at %s", name, address)
	return nil
}

// RemoveService stops monitoring a service
func (hc *GRPCHealthChecker) RemoveService(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if config, exists := hc.services[name]; exists {
		config.cancel()
		config.conn.Close()
		delete(hc.services, name)
		delete(hc.healthStatus, name)
		log.Printf("Stopped health monitoring for service %s", name)
	}
}

// OnHealthChange registers a callback for health status changes
func (hc *GRPCHealthChecker) OnHealthChange(callback func(string, ServiceHealth)) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.callbacks = append(hc.callbacks, callback)
}

// GetHealthStatus returns the current health status of a service
func (hc *GRPCHealthChecker) GetHealthStatus(service string) (ServiceHealth, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	status, exists := hc.healthStatus[service]
	return status, exists
}

// GetAllHealthStatus returns all service health statuses
func (hc *GRPCHealthChecker) GetAllHealthStatus() map[string]ServiceHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	result := make(map[string]ServiceHealth)
	for k, v := range hc.healthStatus {
		result[k] = v
	}
	return result
}

// healthCheckLoop performs periodic health checks for a service
func (hc *GRPCHealthChecker) healthCheckLoop(ctx context.Context, config *serviceConfig) {
	ticker := time.NewTicker(config.interval)
	defer ticker.Stop()

	// Initial health check
	hc.performHealthCheck(config)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.performHealthCheck(config)
		}
	}
}

// performHealthCheck executes a single health check
func (hc *GRPCHealthChecker) performHealthCheck(config *serviceConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &grpc_health_v1.HealthCheckRequest{
		Service: config.name,
	}

	resp, err := config.client.Check(ctx, req)
	
	var health ServiceHealth
	health.Service = config.name
	health.LastChecked = time.Now()

	if err != nil {
		health.Status = "unhealthy"
		health.Details = fmt.Sprintf("gRPC health check failed: %v", err)
		log.Printf("Health check failed for %s: %v", config.name, err)
	} else {
		switch resp.Status {
		case grpc_health_v1.HealthCheckResponse_SERVING:
			health.Status = "healthy"
			health.Details = "Service is serving"
		case grpc_health_v1.HealthCheckResponse_NOT_SERVING:
			health.Status = "unhealthy"
			health.Details = "Service is not serving"
		case grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN:
			health.Status = "unhealthy"
			health.Details = "Service unknown"
		default:
			health.Status = "degraded"
			health.Details = "Unknown health status"
		}
	}

	// Update status and notify callbacks if changed
	hc.mu.Lock()
	previousStatus, exists := hc.healthStatus[config.name]
	hc.healthStatus[config.name] = health
	
	// Check if status changed
	statusChanged := !exists || previousStatus.Status != health.Status
	callbacks := make([]func(string, ServiceHealth), len(hc.callbacks))
	copy(callbacks, hc.callbacks)
	hc.mu.Unlock()

	// Notify callbacks if status changed
	if statusChanged {
		for _, callback := range callbacks {
			go callback(config.name, health)
		}
	}
}

// Close stops all health checking and closes connections
func (hc *GRPCHealthChecker) Close() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	for name, config := range hc.services {
		config.cancel()
		config.conn.Close()
		log.Printf("Closed health monitoring for service %s", name)
	}

	hc.services = make(map[string]*serviceConfig)
	hc.healthStatus = make(map[string]ServiceHealth)
}
