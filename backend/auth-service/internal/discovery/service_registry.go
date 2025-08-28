package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ServiceInfo represents information about a service instance
type ServiceInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	GRPCPort    int       `json:"grpc_port"`
	HTTPPort    int       `json:"http_port"`
	Host        string    `json:"host"`
	Status      string    `json:"status"`
	LastSeen    time.Time `json:"last_seen"`
	StartedAt   time.Time `json:"started_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ServiceRegistry handles service registration and discovery
type ServiceRegistry struct {
	redis       *redis.Client
	ctx         context.Context
	cancel      context.CancelFunc
	serviceInfo *ServiceInfo
	heartbeatInterval time.Duration
	ttl         time.Duration
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(redisClient *redis.Client, serviceInfo *ServiceInfo) *ServiceRegistry {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ServiceRegistry{
		redis:             redisClient,
		ctx:               ctx,
		cancel:            cancel,
		serviceInfo:       serviceInfo,
		heartbeatInterval: 15 * time.Second, // Send heartbeat every 15 seconds
		ttl:               45 * time.Second, // Service expires after 45 seconds without heartbeat
	}
}

// Start begins service registration and heartbeat
func (sr *ServiceRegistry) Start() error {
	// Initial registration
	if err := sr.register(); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	log.Printf("Service registered: %s (ID: %s) on %s:%d (gRPC), %s:%d (HTTP)", 
		sr.serviceInfo.Name, sr.serviceInfo.ID, sr.serviceInfo.Host, 
		sr.serviceInfo.GRPCPort, sr.serviceInfo.Host, sr.serviceInfo.HTTPPort)

	// Start heartbeat goroutine
	go sr.heartbeatLoop()

	return nil
}

// Stop stops the service registry and deregisters the service
func (sr *ServiceRegistry) Stop() error {
	log.Printf("Stopping service registry for %s (ID: %s)", sr.serviceInfo.Name, sr.serviceInfo.ID)
	
	// Cancel heartbeat
	sr.cancel()
	
	// Deregister service
	return sr.deregister()
}

// register registers the service in Redis
func (sr *ServiceRegistry) register() error {
	sr.serviceInfo.LastSeen = time.Now()
	sr.serviceInfo.Status = "healthy"
	
	data, err := json.Marshal(sr.serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	key := sr.getServiceKey()
	
	// Set service info with TTL
	if err := sr.redis.Set(sr.ctx, key, data, sr.ttl).Err(); err != nil {
		return fmt.Errorf("failed to register service in Redis: %w", err)
	}

	// Add to service list
	listKey := fmt.Sprintf("services:%s:instances", sr.serviceInfo.Name)
	if err := sr.redis.SAdd(sr.ctx, listKey, sr.serviceInfo.ID).Err(); err != nil {
		return fmt.Errorf("failed to add service to list: %w", err)
	}

	return nil
}

// deregister removes the service from Redis
func (sr *ServiceRegistry) deregister() error {
	key := sr.getServiceKey()
	
	// Remove service info
	if err := sr.redis.Del(sr.ctx, key).Err(); err != nil {
		log.Printf("Warning: failed to remove service key %s: %v", key, err)
	}

	// Remove from service list
	listKey := fmt.Sprintf("services:%s:instances", sr.serviceInfo.Name)
	if err := sr.redis.SRem(sr.ctx, listKey, sr.serviceInfo.ID).Err(); err != nil {
		log.Printf("Warning: failed to remove service from list: %v", err)
	}

	log.Printf("Service deregistered: %s (ID: %s)", sr.serviceInfo.Name, sr.serviceInfo.ID)
	return nil
}

// heartbeatLoop sends periodic heartbeats to keep the service registered
func (sr *ServiceRegistry) heartbeatLoop() {
	ticker := time.NewTicker(sr.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sr.sendHeartbeat(); err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
			}
		case <-sr.ctx.Done():
			log.Printf("Heartbeat loop stopped for service %s", sr.serviceInfo.Name)
			return
		}
	}
}

// sendHeartbeat updates the service's last seen timestamp
func (sr *ServiceRegistry) sendHeartbeat() error {
	sr.serviceInfo.LastSeen = time.Now()
	
	data, err := json.Marshal(sr.serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	key := sr.getServiceKey()
	
	// Update service info with new TTL
	if err := sr.redis.Set(sr.ctx, key, data, sr.ttl).Err(); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	return nil
}

// getServiceKey returns the Redis key for this service instance
func (sr *ServiceRegistry) getServiceKey() string {
	return fmt.Sprintf("services:%s:instance:%s", sr.serviceInfo.Name, sr.serviceInfo.ID)
}

// UpdateStatus updates the service status
func (sr *ServiceRegistry) UpdateStatus(status string) error {
	sr.serviceInfo.Status = status
	return sr.sendHeartbeat()
}

// GetServiceInfo returns the current service information
func (sr *ServiceRegistry) GetServiceInfo() *ServiceInfo {
	return sr.serviceInfo
}

// ServiceDiscovery handles discovering other services
type ServiceDiscovery struct {
	redis *redis.Client
	ctx   context.Context
}

// NewServiceDiscovery creates a new service discovery client
func NewServiceDiscovery(redisClient *redis.Client) *ServiceDiscovery {
	return &ServiceDiscovery{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// DiscoverServices returns all instances of a specific service
func (sd *ServiceDiscovery) DiscoverServices(serviceName string) ([]*ServiceInfo, error) {
	listKey := fmt.Sprintf("services:%s:instances", serviceName)
	
	// Get all instance IDs
	instanceIDs, err := sd.redis.SMembers(sd.ctx, listKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get service instances: %w", err)
	}

	var services []*ServiceInfo
	
	for _, instanceID := range instanceIDs {
		key := fmt.Sprintf("services:%s:instance:%s", serviceName, instanceID)
		
		data, err := sd.redis.Get(sd.ctx, key).Result()
		if err != nil {
			// Instance might have expired, remove from list
			sd.redis.SRem(sd.ctx, listKey, instanceID)
			continue
		}

		var serviceInfo ServiceInfo
		if err := json.Unmarshal([]byte(data), &serviceInfo); err != nil {
			log.Printf("Failed to unmarshal service info for %s: %v", instanceID, err)
			continue
		}

		services = append(services, &serviceInfo)
	}

	return services, nil
}

// GetAllServices returns all registered services
func (sd *ServiceDiscovery) GetAllServices() (map[string][]*ServiceInfo, error) {
	// Get all service names
	pattern := "services:*:instances"
	keys, err := sd.redis.Keys(sd.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get service keys: %w", err)
	}

	result := make(map[string][]*ServiceInfo)

	for _, key := range keys {
		// Extract service name from key: services:SERVICE_NAME:instances
		serviceName := key[9 : len(key)-10] // Remove "services:" and ":instances"
		
		services, err := sd.DiscoverServices(serviceName)
		if err != nil {
			log.Printf("Failed to discover services for %s: %v", serviceName, err)
			continue
		}

		if len(services) > 0 {
			result[serviceName] = services
		}
	}

	return result, nil
}

// CreateServiceInfo creates a ServiceInfo for the auth service
func CreateAuthServiceInfo(grpcPort, http2Port int, version string) *ServiceInfo {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}

	return &ServiceInfo{
		ID:        uuid.New().String(),
		Name:      "auth-service",
		Version:   version,
		GRPCPort:  grpcPort,
		HTTPPort:  http2Port, // Now HTTP/2 port
		Host:      hostname,
		Status:    "starting",
		StartedAt: time.Now(),
		Metadata: map[string]string{
			"environment":    os.Getenv("ENVIRONMENT"),
			"region":         os.Getenv("REGION"),
			"protocol":       "http2",
			"tls_enabled":    os.Getenv("TLS_ENABLED"),
			"http2_enabled":  os.Getenv("HTTP2_ENABLED"),
		},
	}
}
