package grpc_clients

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"server-service/internal/config"
)

// ClientPool manages gRPC connections to backend services
type ClientPool struct {
	authPool       *ServicePool
	projectPool    *ServicePool
	simulationPool *ServicePool

	// Connection monitoring
	totalConnections int64
	activeRequests   int64
}

// ServicePool manages connections to a specific service with dynamic scaling
type ServicePool struct {
	serviceName string
	address     string
	connections []*grpc.ClientConn
	roundRobin  int64
	config      config.ServiceConfig

	// Dynamic scaling configuration
	minConnections int
	maxConnections int
	scaleUpThreshold   float64 // Scale up when utilization > this
	scaleDownThreshold float64 // Scale down when utilization < this

	// Performance monitoring
	activeRequests int64
	totalRequests  int64
	errorCount     int64
	avgLatency     time.Duration

	// Load tracking for dynamic scaling
	requestHistory []int64 // Recent request counts for load calculation
	lastScaleTime  time.Time
	scaleInterval  time.Duration

	mutex sync.RWMutex
}

// NewClientPool creates a new gRPC client pool for all backend services (resilient initialization)
func NewClientPool(servicesConfig config.ServicesConfig) (*ClientPool, error) {
	pool := &ClientPool{}
	var totalConnections int64 = 0
	var errors []string

	// Create auth service pool (critical service)
	authPool, err := NewServicePool("auth-service", servicesConfig.AuthService)
	if err != nil {
		log.Printf("Warning: Failed to create auth service pool: %v", err)
		errors = append(errors, fmt.Sprintf("auth service: %v", err))
		// Create a dummy pool to prevent nil pointer issues
		authPool = &ServicePool{
			serviceName: "auth-service",
			address:     servicesConfig.AuthService.GRPCAddress,
			connections: make([]*grpc.ClientConn, 0),
		}
	} else {
		totalConnections += int64(len(authPool.connections))
		log.Printf("✅ Auth service pool: %d connections to %s", len(authPool.connections), authPool.address)
	}
	pool.authPool = authPool

	// Create project service pool (optional service)
	projectPool, err := NewServicePool("project-service", servicesConfig.ProjectService)
	if err != nil {
		log.Printf("Warning: Failed to create project service pool: %v", err)
		errors = append(errors, fmt.Sprintf("project service: %v", err))
		// Create a dummy pool to prevent nil pointer issues
		projectPool = &ServicePool{
			serviceName: "project-service",
			address:     servicesConfig.ProjectService.GRPCAddress,
			connections: make([]*grpc.ClientConn, 0),
		}
	} else {
		totalConnections += int64(len(projectPool.connections))
		log.Printf("✅ Project service pool: %d connections to %s", len(projectPool.connections), projectPool.address)
	}
	pool.projectPool = projectPool

	// Create simulation service pool (optional service)
	simulationPool, err := NewServicePool("simulation-service", servicesConfig.SimulationService)
	if err != nil {
		log.Printf("Warning: Failed to create simulation service pool: %v", err)
		errors = append(errors, fmt.Sprintf("simulation service: %v", err))
		// Create a dummy pool to prevent nil pointer issues
		simulationPool = &ServicePool{
			serviceName: "simulation-service",
			address:     servicesConfig.SimulationService.GRPCAddress,
			connections: make([]*grpc.ClientConn, 0),
		}
	} else {
		totalConnections += int64(len(simulationPool.connections))
		log.Printf("✅ Simulation service pool: %d connections to %s", len(simulationPool.connections), simulationPool.address)
	}
	pool.simulationPool = simulationPool

	// Set total connections
	pool.totalConnections = totalConnections

	if totalConnections > 0 {
		log.Printf("🎉 gRPC client pool initialized with %d total connections", totalConnections)
		if len(errors) > 0 {
			log.Printf("⚠️  Some services unavailable: %v", errors)
		}
		return pool, nil
	} else {
		log.Printf("❌ No gRPC services available: %v", errors)
		return pool, fmt.Errorf("no gRPC services available: %v", errors)
	}
}

// NewServicePool creates a dynamic connection pool for a specific service
func NewServicePool(serviceName string, config config.ServiceConfig) (*ServicePool, error) {
	pool := &ServicePool{
		serviceName: serviceName,
		address:     config.GRPCAddress,
		config:      config,

		// Dynamic scaling configuration (5-20 connections based on load)
		minConnections:     5,
		maxConnections:     20,
		scaleUpThreshold:   0.8,  // Scale up when 80% utilization
		scaleDownThreshold: 0.3,  // Scale down when 30% utilization
		scaleInterval:      5 * time.Second, // Check scaling every 5 seconds
		lastScaleTime:      time.Now(),
		requestHistory:     make([]int64, 0, 10), // Track last 10 intervals

		connections: make([]*grpc.ClientConn, 0, 20), // Max capacity
	}

	// Create initial minimum connections
	for i := 0; i < pool.minConnections; i++ {
		conn, err := createGRPCConnection(config)
		if err != nil {
			// Close existing connections on error
			pool.Close()
			return nil, fmt.Errorf("failed to create initial connection %d: %w", i, err)
		}
		pool.connections = append(pool.connections, conn)
	}

	// Start dynamic scaling goroutine
	go pool.dynamicScaler()

	log.Printf("Created dynamic gRPC pool for %s: %d initial connections (min: %d, max: %d)",
		serviceName, len(pool.connections), pool.minConnections, pool.maxConnections)

	return pool, nil
}

// createGRPCConnection creates a single optimized gRPC connection
func createGRPCConnection(config config.ServiceConfig) (*grpc.ClientConn, error) {
	// Connection options for high performance
	opts := []grpc.DialOption{
		// Use insecure connection for internal mesh (development)
		grpc.WithTransportCredentials(insecure.NewCredentials()),

		// Keep-alive settings for connection reuse
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second, // Send ping every 10 seconds
			Timeout:             3 * time.Second,  // Wait 3 seconds for ping response
			PermitWithoutStream: true,             // Allow pings without active streams
		}),

		// Message size limits
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB receive limit
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB send limit
		),

		// Connection timeout
		grpc.WithTimeout(config.ConnectionTimeout),

		// Disable service config for performance
		grpc.WithDisableServiceConfig(),

		// Connection state monitoring
		grpc.WithBlock(), // Block until connection is ready
	}

	// Create connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, config.GRPCAddress, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", config.GRPCAddress, err)
	}

	return conn, nil
}

// GetAuthConnection returns a connection to the auth service
func (cp *ClientPool) GetAuthConnection() *grpc.ClientConn {
	if cp.authPool == nil {
		return nil
	}
	return cp.authPool.GetConnection()
}

// GetProjectConnection returns a connection to the project service
func (cp *ClientPool) GetProjectConnection() *grpc.ClientConn {
	return cp.projectPool.GetConnection()
}

// GetSimulationConnection returns a connection to the simulation service
func (cp *ClientPool) GetSimulationConnection() *grpc.ClientConn {
	return cp.simulationPool.GetConnection()
}

// GetAuthPool returns the auth service pool
func (cp *ClientPool) GetAuthPool() *ServicePool {
	if cp.authPool == nil {
		return &ServicePool{
			serviceName: "auth-service",
			connections: make([]*grpc.ClientConn, 0),
		}
	}
	return cp.authPool
}

// GetProjectPool returns the project service pool
func (cp *ClientPool) GetProjectPool() *ServicePool {
	return cp.projectPool
}

// GetSimulationPool returns the simulation service pool
func (cp *ClientPool) GetSimulationPool() *ServicePool {
	return cp.simulationPool
}

// GetConnection returns a connection using round-robin load balancing
func (sp *ServicePool) GetConnection() *grpc.ClientConn {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	if len(sp.connections) == 0 {
		return nil
	}

	// Round-robin selection (atomic for thread safety)
	idx := atomic.AddInt64(&sp.roundRobin, 1) % int64(len(sp.connections))

	// Track active request
	newActive := atomic.AddInt64(&sp.activeRequests, 1)
	newTotal := atomic.AddInt64(&sp.totalRequests, 1)

	log.Printf("GetConnection for %s: active=%d, total=%d", sp.serviceName, newActive, newTotal)

	return sp.connections[idx]
}

// dynamicScaler runs in background to scale connections based on load
func (sp *ServicePool) dynamicScaler() {
	log.Printf("Starting dynamic scaler for %s (interval: %v)", sp.serviceName, sp.scaleInterval)
	ticker := time.NewTicker(sp.scaleInterval)
	defer ticker.Stop()

	for range ticker.C {
		log.Printf("Dynamic scaler tick for %s", sp.serviceName)
		sp.checkAndScale()
	}
}

// checkAndScale evaluates current load and scales connections accordingly
func (sp *ServicePool) checkAndScale() {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	currentConnections := len(sp.connections)
	activeReqs := atomic.LoadInt64(&sp.activeRequests)
	totalReqs := atomic.LoadInt64(&sp.totalRequests)

	// Calculate request rate (requests per second over the interval)
	var requestRate float64
	if len(sp.requestHistory) > 0 {
		lastTotal := sp.requestHistory[len(sp.requestHistory)-1]
		requestRate = float64(totalReqs-lastTotal) / sp.scaleInterval.Seconds()
	}

	// Calculate utilization based on request rate (requests per connection per second)
	// Assume each connection can handle ~10 requests per second optimally
	optimalRatePerConnection := 10.0
	utilization := requestRate / (float64(currentConnections) * optimalRatePerConnection)

	log.Printf("Scaling check for %s: connections=%d, active=%d, total=%d, rate=%.1f/s, utilization=%.2f",
		sp.serviceName, currentConnections, activeReqs, totalReqs, requestRate, utilization)

	// Track request history for trend analysis (store total requests, not active)
	sp.requestHistory = append(sp.requestHistory, totalReqs)
	if len(sp.requestHistory) > 10 {
		sp.requestHistory = sp.requestHistory[1:] // Keep last 10 samples
	}

	// Prevent too frequent scaling
	if time.Since(sp.lastScaleTime) < sp.scaleInterval {
		return
	}

	// Scale up logic
	if utilization > sp.scaleUpThreshold && currentConnections < sp.maxConnections {
		newConnections := min(currentConnections+2, sp.maxConnections) // Add 2 connections at a time
		sp.scaleUp(newConnections - currentConnections)
		log.Printf("Scaled UP %s pool: %d -> %d connections (utilization: %.2f)",
			sp.serviceName, currentConnections, newConnections, utilization)
		sp.lastScaleTime = time.Now()
	}

	// Scale down logic (only if utilization is consistently low)
	if utilization < sp.scaleDownThreshold && currentConnections > sp.minConnections {
		// Check if utilization has been consistently low
		if sp.isUtilizationConsistentlyLow() {
			newConnections := max(currentConnections-1, sp.minConnections) // Remove 1 connection at a time
			sp.scaleDown(currentConnections - newConnections)
			log.Printf("Scaled DOWN %s pool: %d -> %d connections (utilization: %.2f)",
				sp.serviceName, currentConnections, newConnections, utilization)
			sp.lastScaleTime = time.Now()
		}
	}
}

// scaleUp adds new connections to the pool
func (sp *ServicePool) scaleUp(count int) {
	for i := 0; i < count; i++ {
		conn, err := createGRPCConnection(sp.config)
		if err != nil {
			log.Printf("Failed to create new connection for %s: %v", sp.serviceName, err)
			sp.RecordError()
			continue
		}
		sp.connections = append(sp.connections, conn)
	}
}

// scaleDown removes connections from the pool
func (sp *ServicePool) scaleDown(count int) {
	for i := 0; i < count && len(sp.connections) > sp.minConnections; i++ {
		// Remove the last connection
		lastIdx := len(sp.connections) - 1
		conn := sp.connections[lastIdx]
		sp.connections = sp.connections[:lastIdx]

		// Close the connection gracefully
		go func(c *grpc.ClientConn) {
			if err := c.Close(); err != nil {
				log.Printf("Error closing connection for %s: %v", sp.serviceName, err)
			}
		}(conn)
	}
}

// isUtilizationConsistentlyLow checks if utilization has been low for multiple intervals
func (sp *ServicePool) isUtilizationConsistentlyLow() bool {
	if len(sp.requestHistory) < 3 {
		return false // Need at least 3 samples
	}

	// Check if last 3 intervals show low request rate
	recentSamples := sp.requestHistory[len(sp.requestHistory)-3:]
	optimalRatePerConnection := 10.0

	for i := 1; i < len(recentSamples); i++ {
		requestRate := float64(recentSamples[i]-recentSamples[i-1]) / sp.scaleInterval.Seconds()
		utilization := requestRate / (float64(len(sp.connections)) * optimalRatePerConnection)
		if utilization >= sp.scaleDownThreshold {
			return false
		}
	}

	return true
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ReleaseConnection marks a connection as available (for monitoring)
func (sp *ServicePool) ReleaseConnection() {
	newActive := atomic.AddInt64(&sp.activeRequests, -1)
	log.Printf("ReleaseConnection for %s: active=%d", sp.serviceName, newActive)
}

// GetPoolStats returns detailed pool statistics for monitoring
func (sp *ServicePool) GetPoolStats() map[string]interface{} {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	activeReqs := atomic.LoadInt64(&sp.activeRequests)
	totalReqs := atomic.LoadInt64(&sp.totalRequests)
	errors := atomic.LoadInt64(&sp.errorCount)
	currentConnections := len(sp.connections)

	var utilization float64
	if currentConnections > 0 {
		utilization = float64(activeReqs) / float64(currentConnections)
	}

	var errorRate float64
	if totalReqs > 0 {
		errorRate = float64(errors) / float64(totalReqs) * 100
	}

	return map[string]interface{}{
		"service_name":        sp.serviceName,
		"current_connections": currentConnections,
		"min_connections":     sp.minConnections,
		"max_connections":     sp.maxConnections,
		"active_requests":     activeReqs,
		"total_requests":      totalReqs,
		"error_count":         errors,
		"error_rate_percent":  errorRate,
		"utilization":         utilization,
		"scale_up_threshold":  sp.scaleUpThreshold,
		"scale_down_threshold": sp.scaleDownThreshold,
		"last_scale_time":     sp.lastScaleTime.Format(time.RFC3339),
		"request_history":     sp.requestHistory,
	}
}

// GetStats returns pool statistics
func (cp *ClientPool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_connections":  atomic.LoadInt64(&cp.totalConnections),
		"active_requests":    atomic.LoadInt64(&cp.activeRequests),
		"auth_service":       cp.authPool.GetPoolStats(),
		"project_service":    cp.projectPool.GetStats(),
		"simulation_service": cp.simulationPool.GetStats(),
	}
}

// GetStats returns service pool statistics
func (sp *ServicePool) GetStats() map[string]interface{} {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	return map[string]interface{}{
		"service_name":      sp.serviceName,
		"address":           sp.address,
		"total_connections": len(sp.connections),
		"active_requests":   atomic.LoadInt64(&sp.activeRequests),
		"total_requests":    atomic.LoadInt64(&sp.totalRequests),
		"error_count":       atomic.LoadInt64(&sp.errorCount),
		"avg_latency_ms":    int64(sp.avgLatency.Milliseconds()),
		"utilization":       func() float64 {
			active := atomic.LoadInt64(&sp.activeRequests)
			conns := len(sp.connections)
			if conns > 0 {
				return float64(active) / float64(conns)
			}
			return 0.0
		}(),
	}
}

// HealthCheck checks the health of all connections
func (cp *ClientPool) HealthCheck() map[string]bool {
	return map[string]bool{
		"auth_service":       cp.authPool.HealthCheck(),
		"project_service":    cp.projectPool.HealthCheck(),
		"simulation_service": cp.simulationPool.HealthCheck(),
	}
}

// HealthCheck checks if the service pool is healthy
func (sp *ServicePool) HealthCheck() bool {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	healthyConnections := 0
	for _, conn := range sp.connections {
		if conn.GetState().String() == "READY" {
			healthyConnections++
		}
	}

	// Consider healthy if at least 50% of connections are ready
	return healthyConnections >= len(sp.connections)/2
}

// Close closes all connections in the pool
func (cp *ClientPool) Close() error {
	var errors []error

	if err := cp.authPool.Close(); err != nil {
		errors = append(errors, fmt.Errorf("auth pool: %w", err))
	}

	if err := cp.projectPool.Close(); err != nil {
		errors = append(errors, fmt.Errorf("project pool: %w", err))
	}

	if err := cp.simulationPool.Close(); err != nil {
		errors = append(errors, fmt.Errorf("simulation pool: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing pools: %v", errors)
	}

	log.Println("All gRPC connection pools closed")
	return nil
}

// Close closes all connections in the service pool
func (sp *ServicePool) Close() error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	var errors []error
	for i, conn := range sp.connections {
		if err := conn.Close(); err != nil {
			errors = append(errors, fmt.Errorf("connection %d: %w", i, err))
		}
	}

	sp.connections = nil

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	return nil
}

// RecordError records an error for monitoring
func (sp *ServicePool) RecordError() {
	atomic.AddInt64(&sp.errorCount, 1)
}

// RecordLatency records request latency for monitoring
func (sp *ServicePool) RecordLatency(latency time.Duration) {
	// Simple moving average (can be improved with more sophisticated metrics)
	sp.avgLatency = (sp.avgLatency + latency) / 2
}
