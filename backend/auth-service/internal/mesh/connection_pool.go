package mesh

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// ConnectionPool manages dynamic gRPC connections to a target service
type ConnectionPool struct {
	targetService  string
	targetAddress  string
	minConnections int
	maxConnections int
	
	// Connection management
	connections    []*PooledConnection
	mu             sync.RWMutex
	currentIndex   int64 // For round-robin load balancing
	
	// Health monitoring
	healthChecker  *ConnectionHealthChecker
	ctx            context.Context
	cancel         context.CancelFunc
	
	// Metrics
	metrics        *PoolMetrics
}

// PooledConnection wraps a gRPC connection with health information
type PooledConnection struct {
	conn         *grpc.ClientConn
	id           string
	createdAt    time.Time
	lastUsed     time.Time
	requestCount int64
	healthy      bool
	mu           sync.RWMutex
}

// PoolMetrics tracks connection pool performance
type PoolMetrics struct {
	TotalConnections    int64
	HealthyConnections  int64
	UnhealthyConnections int64
	TotalRequests       int64
	FailedRequests      int64
	AverageLatency      time.Duration
	mu                  sync.RWMutex
}

// ConnectionHealthChecker monitors connection health
type ConnectionHealthChecker struct {
	pool           *ConnectionPool
	checkInterval  time.Duration
	ctx            context.Context
}

// NewConnectionPool creates a new dynamic connection pool
func NewConnectionPool(targetService, targetAddress string, minConns, maxConns int) *ConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &ConnectionPool{
		targetService:  targetService,
		targetAddress:  targetAddress,
		minConnections: minConns,
		maxConnections: maxConns,
		connections:    make([]*PooledConnection, 0, maxConns),
		ctx:            ctx,
		cancel:         cancel,
		metrics:        &PoolMetrics{},
	}
	
	// Initialize health checker
	pool.healthChecker = &ConnectionHealthChecker{
		pool:          pool,
		checkInterval: 30 * time.Second,
		ctx:           ctx,
	}
	
	return pool
}

// Start initializes the connection pool with minimum connections
func (cp *ConnectionPool) Start() error {
	log.Printf("Starting connection pool for %s (%s) with %d-%d connections", 
		cp.targetService, cp.targetAddress, cp.minConnections, cp.maxConnections)
	
	// Create minimum connections
	for i := 0; i < cp.minConnections; i++ {
		if err := cp.createConnection(); err != nil {
			log.Printf("Failed to create initial connection %d for %s: %v", i, cp.targetService, err)
			// Continue creating other connections
		}
	}
	
	// Start health monitoring
	go cp.healthChecker.start()
	
	log.Printf("Connection pool started for %s with %d initial connections", 
		cp.targetService, len(cp.connections))
	
	return nil
}

// Stop closes all connections and stops the pool
func (cp *ConnectionPool) Stop() error {
	log.Printf("Stopping connection pool for %s", cp.targetService)
	
	cp.cancel() // Stop health checker
	
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	// Close all connections
	for _, conn := range cp.connections {
		if err := conn.conn.Close(); err != nil {
			log.Printf("Error closing connection to %s: %v", cp.targetService, err)
		}
	}
	
	cp.connections = nil
	log.Printf("Connection pool stopped for %s", cp.targetService)
	return nil
}

// GetConnection returns a healthy connection using round-robin load balancing
func (cp *ConnectionPool) GetConnection() (*grpc.ClientConn, error) {
	cp.mu.RLock()
	
	if len(cp.connections) == 0 {
		cp.mu.RUnlock()
		return nil, fmt.Errorf("no connections available for %s", cp.targetService)
	}
	
	// Find healthy connections
	healthyConns := make([]*PooledConnection, 0, len(cp.connections))
	for _, conn := range cp.connections {
		if cp.isConnectionHealthy(conn) {
			healthyConns = append(healthyConns, conn)
		}
	}
	
	cp.mu.RUnlock()
	
	if len(healthyConns) == 0 {
		// Try to create a new connection if we're under max
		if len(cp.connections) < cp.maxConnections {
			if err := cp.createConnection(); err != nil {
				return nil, fmt.Errorf("no healthy connections and failed to create new one: %v", err)
			}
			return cp.GetConnection() // Retry with new connection
		}
		return nil, fmt.Errorf("no healthy connections available for %s", cp.targetService)
	}
	
	// Round-robin load balancing
	index := atomic.AddInt64(&cp.currentIndex, 1) % int64(len(healthyConns))
	selectedConn := healthyConns[index]
	
	// Update usage statistics
	selectedConn.mu.Lock()
	selectedConn.lastUsed = time.Now()
	selectedConn.requestCount++
	selectedConn.mu.Unlock()
	
	// Update metrics
	atomic.AddInt64(&cp.metrics.TotalRequests, 1)
	
	return selectedConn.conn, nil
}

// createConnection creates a new gRPC connection and adds it to the pool
func (cp *ConnectionPool) createConnection() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	if len(cp.connections) >= cp.maxConnections {
		return fmt.Errorf("connection pool full (%d/%d) for %s", len(cp.connections), cp.maxConnections, cp.targetService)
	}
	
	// Create gRPC connection
	conn, err := grpc.Dial(cp.targetAddress, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to create connection to %s: %v", cp.targetAddress, err)
	}
	
	// Create pooled connection
	pooledConn := &PooledConnection{
		conn:         conn,
		id:           fmt.Sprintf("%s-%d", cp.targetService, len(cp.connections)+1),
		createdAt:    time.Now(),
		lastUsed:     time.Now(),
		requestCount: 0,
		healthy:      true,
	}
	
	cp.connections = append(cp.connections, pooledConn)
	atomic.AddInt64(&cp.metrics.TotalConnections, 1)
	atomic.AddInt64(&cp.metrics.HealthyConnections, 1)
	
	log.Printf("Created new connection %s for %s (total: %d/%d)", 
		pooledConn.id, cp.targetService, len(cp.connections), cp.maxConnections)
	
	return nil
}

// isConnectionHealthy checks if a connection is healthy
func (cp *ConnectionPool) isConnectionHealthy(conn *PooledConnection) bool {
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	
	if !conn.healthy {
		return false
	}
	
	// Check gRPC connection state
	state := conn.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// removeUnhealthyConnection removes a connection from the pool
func (cp *ConnectionPool) removeUnhealthyConnection(conn *PooledConnection) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	for i, c := range cp.connections {
		if c.id == conn.id {
			// Close the connection
			c.conn.Close()
			
			// Remove from slice
			cp.connections = append(cp.connections[:i], cp.connections[i+1:]...)
			
			// Update metrics
			atomic.AddInt64(&cp.metrics.TotalConnections, -1)
			atomic.AddInt64(&cp.metrics.UnhealthyConnections, 1)
			
			log.Printf("Removed unhealthy connection %s for %s (remaining: %d)", 
				conn.id, cp.targetService, len(cp.connections))
			
			break
		}
	}
	
	// Ensure we maintain minimum connections
	if len(cp.connections) < cp.minConnections {
		go func() {
			if err := cp.createConnection(); err != nil {
				log.Printf("Failed to replace unhealthy connection for %s: %v", cp.targetService, err)
			}
		}()
	}
}

// GetMetrics returns current pool metrics
func (cp *ConnectionPool) GetMetrics() *PoolMetrics {
	cp.metrics.mu.RLock()
	defer cp.metrics.mu.RUnlock()
	
	// Update current connection counts
	cp.mu.RLock()
	healthyCount := int64(0)
	for _, conn := range cp.connections {
		if cp.isConnectionHealthy(conn) {
			healthyCount++
		}
	}
	cp.mu.RUnlock()
	
	return &PoolMetrics{
		TotalConnections:     int64(len(cp.connections)),
		HealthyConnections:   healthyCount,
		UnhealthyConnections: int64(len(cp.connections)) - healthyCount,
		TotalRequests:        atomic.LoadInt64(&cp.metrics.TotalRequests),
		FailedRequests:       atomic.LoadInt64(&cp.metrics.FailedRequests),
		AverageLatency:       cp.metrics.AverageLatency,
	}
}

// start begins health checking for connections
func (hc *ConnectionHealthChecker) start() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()
	
	log.Printf("Started health checker for %s connection pool", hc.pool.targetService)
	
	for {
		select {
		case <-ticker.C:
			hc.checkAllConnections()
		case <-hc.ctx.Done():
			log.Printf("Health checker stopped for %s", hc.pool.targetService)
			return
		}
	}
}

// checkAllConnections checks health of all connections in the pool
func (hc *ConnectionHealthChecker) checkAllConnections() {
	hc.pool.mu.RLock()
	connections := make([]*PooledConnection, len(hc.pool.connections))
	copy(connections, hc.pool.connections)
	hc.pool.mu.RUnlock()
	
	for _, conn := range connections {
		if !hc.pool.isConnectionHealthy(conn) {
			log.Printf("Connection %s for %s is unhealthy, removing", conn.id, hc.pool.targetService)
			hc.pool.removeUnhealthyConnection(conn)
		}
	}
}
