package mesh

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"github.com/systemsim/auth-service/internal/discovery"
)

// PoolManager manages connection pools for multiple services
type PoolManager struct {
	pools           map[string]*ConnectionPool
	serviceDiscovery *discovery.ServiceDiscovery
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	
	// Configuration
	minConnections  int
	maxConnections  int
	refreshInterval time.Duration
}

// NewPoolManager creates a new connection pool manager
func NewPoolManager(serviceDiscovery *discovery.ServiceDiscovery, minConns, maxConns int) *PoolManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PoolManager{
		pools:           make(map[string]*ConnectionPool),
		serviceDiscovery: serviceDiscovery,
		ctx:             ctx,
		cancel:          cancel,
		minConnections:  minConns,
		maxConnections:  maxConns,
		refreshInterval: 30 * time.Second, // Refresh service discovery every 30 seconds
	}
}

// Start begins managing connection pools
func (pm *PoolManager) Start() error {
	log.Printf("Starting connection pool manager with %d-%d connections per service", 
		pm.minConnections, pm.maxConnections)
	
	// Initial service discovery
	if err := pm.refreshServicePools(); err != nil {
		log.Printf("Warning: Initial service discovery failed: %v", err)
	}
	
	// Start periodic service discovery refresh
	go pm.serviceDiscoveryLoop()
	
	log.Printf("Connection pool manager started")
	return nil
}

// Stop stops all connection pools
func (pm *PoolManager) Stop() error {
	log.Printf("Stopping connection pool manager")
	
	pm.cancel() // Stop service discovery loop
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Stop all pools
	for serviceName, pool := range pm.pools {
		if err := pool.Stop(); err != nil {
			log.Printf("Error stopping pool for %s: %v", serviceName, err)
		}
	}
	
	pm.pools = make(map[string]*ConnectionPool)
	log.Printf("Connection pool manager stopped")
	return nil
}

// GetConnection returns a connection to the specified service
func (pm *PoolManager) GetConnection(serviceName string) (*grpc.ClientConn, error) {
	pm.mu.RLock()
	pool, exists := pm.pools[serviceName]
	pm.mu.RUnlock()
	
	if !exists {
		// Try to create pool for this service
		if err := pm.createPoolForService(serviceName); err != nil {
			return nil, fmt.Errorf("service %s not available and failed to create pool: %v", serviceName, err)
		}
		
		// Retry getting the connection
		pm.mu.RLock()
		pool, exists = pm.pools[serviceName]
		pm.mu.RUnlock()
		
		if !exists {
			return nil, fmt.Errorf("failed to create connection pool for service %s", serviceName)
		}
	}
	
	return pool.GetConnection()
}

// GetPoolMetrics returns metrics for all connection pools
func (pm *PoolManager) GetPoolMetrics() map[string]*PoolMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	metrics := make(map[string]*PoolMetrics)
	for serviceName, pool := range pm.pools {
		metrics[serviceName] = pool.GetMetrics()
	}
	
	return metrics
}

// GetServiceNames returns all services with active connection pools
func (pm *PoolManager) GetServiceNames() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	services := make([]string, 0, len(pm.pools))
	for serviceName := range pm.pools {
		services = append(services, serviceName)
	}
	
	return services
}

// serviceDiscoveryLoop periodically refreshes service discovery
func (pm *PoolManager) serviceDiscoveryLoop() {
	ticker := time.NewTicker(pm.refreshInterval)
	defer ticker.Stop()
	
	log.Printf("Started service discovery loop for connection pools")
	
	for {
		select {
		case <-ticker.C:
			if err := pm.refreshServicePools(); err != nil {
				log.Printf("Service discovery refresh failed: %v", err)
			}
		case <-pm.ctx.Done():
			log.Printf("Service discovery loop stopped")
			return
		}
	}
}

// refreshServicePools discovers services and creates/updates connection pools
func (pm *PoolManager) refreshServicePools() error {
	// Discover all services
	allServices, err := pm.serviceDiscovery.GetAllServices()
	if err != nil {
		return fmt.Errorf("failed to discover services: %v", err)
	}
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Track which services we found
	foundServices := make(map[string]bool)
	
	// Create or update pools for discovered services
	for serviceName, instances := range allServices {
		// Skip self (auth-service)
		if serviceName == "auth-service" {
			continue
		}
		
		foundServices[serviceName] = true
		
		// Check if we already have a pool for this service
		if _, exists := pm.pools[serviceName]; !exists {
			// Find a healthy instance to connect to
			var targetAddress string
			for _, instance := range instances {
				if instance.Status == "healthy" {
					targetAddress = fmt.Sprintf("%s:%d", instance.Host, instance.GRPCPort)
					break
				}
			}
			
			if targetAddress == "" {
				log.Printf("No healthy instances found for service %s", serviceName)
				continue
			}
			
			// Create new connection pool
			pool := NewConnectionPool(serviceName, targetAddress, pm.minConnections, pm.maxConnections)
			if err := pool.Start(); err != nil {
				log.Printf("Failed to start connection pool for %s: %v", serviceName, err)
				continue
			}
			
			pm.pools[serviceName] = pool
			log.Printf("Created connection pool for service %s at %s", serviceName, targetAddress)
		}
	}
	
	// Remove pools for services that are no longer available
	for serviceName, pool := range pm.pools {
		if !foundServices[serviceName] {
			log.Printf("Service %s no longer available, stopping connection pool", serviceName)
			pool.Stop()
			delete(pm.pools, serviceName)
		}
	}
	
	return nil
}

// createPoolForService creates a connection pool for a specific service
func (pm *PoolManager) createPoolForService(serviceName string) error {
	// Discover instances of this service
	instances, err := pm.serviceDiscovery.DiscoverServices(serviceName)
	if err != nil {
		return fmt.Errorf("failed to discover service %s: %v", serviceName, err)
	}
	
	if len(instances) == 0 {
		return fmt.Errorf("no instances found for service %s", serviceName)
	}
	
	// Find a healthy instance
	var targetAddress string
	for _, instance := range instances {
		if instance.Status == "healthy" {
			targetAddress = fmt.Sprintf("%s:%d", instance.Host, instance.GRPCPort)
			break
		}
	}
	
	if targetAddress == "" {
		return fmt.Errorf("no healthy instances found for service %s", serviceName)
	}
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Create and start connection pool
	pool := NewConnectionPool(serviceName, targetAddress, pm.minConnections, pm.maxConnections)
	if err := pool.Start(); err != nil {
		return fmt.Errorf("failed to start connection pool for %s: %v", serviceName, err)
	}
	
	pm.pools[serviceName] = pool
	log.Printf("Created connection pool for service %s at %s", serviceName, targetAddress)
	
	return nil
}

// HealthCheck returns health status of all connection pools
func (pm *PoolManager) HealthCheck() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	health := make(map[string]interface{})
	
	totalPools := len(pm.pools)
	healthyPools := 0
	totalConnections := int64(0)
	healthyConnections := int64(0)
	
	poolDetails := make(map[string]interface{})
	
	for serviceName, pool := range pm.pools {
		metrics := pool.GetMetrics()
		
		poolHealth := map[string]interface{}{
			"total_connections":   metrics.TotalConnections,
			"healthy_connections": metrics.HealthyConnections,
			"total_requests":      metrics.TotalRequests,
			"failed_requests":     metrics.FailedRequests,
		}
		
		if metrics.HealthyConnections > 0 {
			healthyPools++
		}
		
		totalConnections += metrics.TotalConnections
		healthyConnections += metrics.HealthyConnections
		
		poolDetails[serviceName] = poolHealth
	}
	
	health["total_pools"] = totalPools
	health["healthy_pools"] = healthyPools
	health["total_connections"] = totalConnections
	health["healthy_connections"] = healthyConnections
	health["pool_details"] = poolDetails
	
	return health
}

// DefaultPoolManager creates a pool manager with default settings (5-20 connections)
func DefaultPoolManager(serviceDiscovery *discovery.ServiceDiscovery) *PoolManager {
	return NewPoolManager(serviceDiscovery, 5, 20) // 5 min, 20 max connections per service
}
