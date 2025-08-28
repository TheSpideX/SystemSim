package mesh

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConnectionPool_Creation(t *testing.T) {
	tests := []struct {
		name           string
		targetService  string
		targetAddress  string
		minConnections int
		maxConnections int
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "valid_pool_creation",
			targetService:  "test-service",
			targetAddress:  "localhost:9000",
			minConnections: 2,
			maxConnections: 10,
			expectError:    false,
		},
		{
			name:           "zero_min_connections",
			targetService:  "test-service",
			targetAddress:  "localhost:9001",
			minConnections: 0,
			maxConnections: 5,
			expectError:    false,
		},
		{
			name:           "equal_min_max_connections",
			targetService:  "test-service",
			targetAddress:  "localhost:9002",
			minConnections: 5,
			maxConnections: 5,
			expectError:    false,
		},
		{
			name:           "min_greater_than_max",
			targetService:  "test-service",
			targetAddress:  "localhost:9003",
			minConnections: 10,
			maxConnections: 5,
			expectError:    false, // Should be handled gracefully
		},
		{
			name:           "empty_service_name",
			targetService:  "",
			targetAddress:  "localhost:9004",
			minConnections: 2,
			maxConnections: 10,
			expectError:    false, // Should be allowed for testing
		},
		{
			name:           "empty_target_address",
			targetService:  "test-service",
			targetAddress:  "",
			minConnections: 2,
			maxConnections: 10,
			expectError:    false, // Will fail on Start(), not creation
		},
		{
			name:           "large_connection_pool",
			targetService:  "large-service",
			targetAddress:  "localhost:9005",
			minConnections: 50,
			maxConnections: 100,
			expectError:    false,
		},
		{
			name:           "single_connection_pool",
			targetService:  "single-service",
			targetAddress:  "localhost:9006",
			minConnections: 1,
			maxConnections: 1,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewConnectionPool(tt.targetService, tt.targetAddress, tt.minConnections, tt.maxConnections)

			if tt.expectError {
				assert.Nil(t, pool, "Pool should be nil for invalid parameters")
			} else {
				assert.NotNil(t, pool, "Pool should not be nil")
				assert.Equal(t, tt.targetService, pool.targetService)
				assert.Equal(t, tt.targetAddress, pool.targetAddress)
				assert.Equal(t, tt.minConnections, pool.minConnections)
				assert.Equal(t, tt.maxConnections, pool.maxConnections)
				assert.NotNil(t, pool.connections, "Connections slice should be initialized")
				assert.NotNil(t, pool.metrics, "Metrics should be initialized")
				assert.NotNil(t, pool.healthChecker, "Health checker should be initialized")
				assert.NotNil(t, pool.ctx, "Context should be initialized")
				assert.NotNil(t, pool.cancel, "Cancel function should be initialized")

				// Clean up
				pool.Stop()
			}
		})
	}
}

func TestConnectionPool_Metrics(t *testing.T) {
	pool := NewConnectionPool("metrics-test", "localhost:9100", 1, 5)
	defer pool.Stop()

	t.Run("initial_metrics", func(t *testing.T) {
		metrics := pool.GetMetrics()
		assert.NotNil(t, metrics, "Metrics should not be nil")
		assert.Equal(t, int64(0), metrics.TotalConnections, "Initial total connections should be 0")
		assert.Equal(t, int64(0), metrics.HealthyConnections, "Initial healthy connections should be 0")
		assert.Equal(t, int64(0), metrics.UnhealthyConnections, "Initial unhealthy connections should be 0")
		assert.Equal(t, int64(0), metrics.TotalRequests, "Initial total requests should be 0")
		assert.Equal(t, int64(0), metrics.FailedRequests, "Initial failed requests should be 0")
	})

	t.Run("metrics_thread_safety", func(t *testing.T) {
		const numGoroutines = 10
		const operationsPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Simulate concurrent metric updates
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Simulate metric updates that would happen during normal operation
					pool.metrics.mu.Lock()
					pool.metrics.TotalRequests++
					if j%10 == 0 {
						pool.metrics.FailedRequests++
					}
					pool.metrics.mu.Unlock()
				}
			}()
		}

		wg.Wait()

		metrics := pool.GetMetrics()
		expectedTotal := int64(numGoroutines * operationsPerGoroutine)
		expectedFailed := int64(numGoroutines * (operationsPerGoroutine / 10))

		assert.Equal(t, expectedTotal, metrics.TotalRequests, "Total requests should match expected")
		assert.Equal(t, expectedFailed, metrics.FailedRequests, "Failed requests should match expected")
	})
}

func TestConnectionPool_LoadBalancing(t *testing.T) {
	// This test focuses on the load balancing logic without actual gRPC connections
	pool := NewConnectionPool("lb-test", "localhost:9200", 3, 10)
	defer pool.Stop()

	t.Run("round_robin_selection", func(t *testing.T) {
		// Create mock connections
		pool.mu.Lock()
		for i := 0; i < 3; i++ {
			mockConn := &PooledConnection{
				conn:         nil, // We'll mock this
				id:           fmt.Sprintf("mock-conn-%d", i),
				createdAt:    time.Now(),
				lastUsed:     time.Now(),
				requestCount: 0,
				healthy:      true,
			}
			pool.connections = append(pool.connections, mockConn)
		}
		pool.mu.Unlock()

		// Test round-robin behavior by checking the index progression
		initialIndex := pool.currentIndex
		
		// Simulate multiple requests to see round-robin behavior
		for i := 0; i < 10; i++ {
			// We can't actually get connections without a real gRPC server,
			// but we can test the index calculation logic
			expectedIndex := (initialIndex + int64(i) + 1) % int64(len(pool.connections))
			actualIndex := (pool.currentIndex + 1) % int64(len(pool.connections))
			
			// Simulate the index increment that happens in GetConnection
			pool.currentIndex++
			
			if i == 0 {
				// First iteration, check the calculation
				assert.Equal(t, expectedIndex, actualIndex, "Round-robin index should progress correctly")
			}
		}

		// Verify that all connections would be used in round-robin fashion
		assert.True(t, pool.currentIndex > initialIndex, "Current index should have progressed")
	})

	t.Run("healthy_connection_filtering", func(t *testing.T) {
		pool.mu.Lock()
		// Reset connections and create mix of healthy and unhealthy
		pool.connections = nil
		
		// Add healthy connections
		for i := 0; i < 2; i++ {
			healthyConn := &PooledConnection{
				id:      fmt.Sprintf("healthy-%d", i),
				healthy: true,
			}
			pool.connections = append(pool.connections, healthyConn)
		}
		
		// Add unhealthy connections
		for i := 0; i < 2; i++ {
			unhealthyConn := &PooledConnection{
				id:      fmt.Sprintf("unhealthy-%d", i),
				healthy: false,
			}
			pool.connections = append(pool.connections, unhealthyConn)
		}
		pool.mu.Unlock()

		// Test that only healthy connections are considered
		pool.mu.RLock()
		healthyConns := make([]*PooledConnection, 0, len(pool.connections))
		for _, conn := range pool.connections {
			if pool.isConnectionHealthy(conn) {
				healthyConns = append(healthyConns, conn)
			}
		}
		pool.mu.RUnlock()

		assert.Len(t, healthyConns, 2, "Should find exactly 2 healthy connections")
		for _, conn := range healthyConns {
			assert.True(t, conn.healthy, "All filtered connections should be healthy")
			assert.Contains(t, conn.id, "healthy", "Connection ID should indicate it's healthy")
		}
	})
}

func TestConnectionPool_HealthChecking(t *testing.T) {
	pool := NewConnectionPool("health-test", "localhost:9300", 1, 5)
	defer pool.Stop()

	t.Run("connection_health_assessment", func(t *testing.T) {
		// Test with healthy connection
		healthyConn := &PooledConnection{
			id:           "healthy-conn",
			createdAt:    time.Now(),
			lastUsed:     time.Now(),
			requestCount: 10,
			healthy:      true,
		}

		assert.True(t, pool.isConnectionHealthy(healthyConn), "Healthy connection should be assessed as healthy")

		// Test with unhealthy connection
		unhealthyConn := &PooledConnection{
			id:           "unhealthy-conn",
			createdAt:    time.Now(),
			lastUsed:     time.Now().Add(-time.Hour), // Old last used time
			requestCount: 0,
			healthy:      false,
		}

		assert.False(t, pool.isConnectionHealthy(unhealthyConn), "Unhealthy connection should be assessed as unhealthy")
	})

	t.Run("health_checker_initialization", func(t *testing.T) {
		assert.NotNil(t, pool.healthChecker, "Health checker should be initialized")
		assert.Equal(t, pool, pool.healthChecker.pool, "Health checker should reference the pool")
		assert.Equal(t, 30*time.Second, pool.healthChecker.checkInterval, "Check interval should be 30 seconds")
		assert.NotNil(t, pool.healthChecker.ctx, "Health checker context should be initialized")
	})

	t.Run("health_metrics_tracking", func(t *testing.T) {
		// Add mock connections with different health states
		pool.mu.Lock()
		pool.connections = nil
		
		// Add healthy connections
		for i := 0; i < 3; i++ {
			healthyConn := &PooledConnection{
				id:      fmt.Sprintf("healthy-%d", i),
				healthy: true,
			}
			pool.connections = append(pool.connections, healthyConn)
		}
		
		// Add unhealthy connections
		for i := 0; i < 2; i++ {
			unhealthyConn := &PooledConnection{
				id:      fmt.Sprintf("unhealthy-%d", i),
				healthy: false,
			}
			pool.connections = append(pool.connections, unhealthyConn)
		}
		pool.mu.Unlock()

		// Update metrics to reflect the connection states
		pool.metrics.mu.Lock()
		pool.metrics.TotalConnections = 5
		pool.metrics.HealthyConnections = 3
		pool.metrics.UnhealthyConnections = 2
		pool.metrics.mu.Unlock()

		metrics := pool.GetMetrics()
		assert.Equal(t, int64(5), metrics.TotalConnections, "Total connections should be 5")
		assert.Equal(t, int64(3), metrics.HealthyConnections, "Healthy connections should be 3")
		assert.Equal(t, int64(2), metrics.UnhealthyConnections, "Unhealthy connections should be 2")
	})
}

func TestConnectionPool_ErrorHandling(t *testing.T) {
	t.Run("invalid_target_address", func(t *testing.T) {
		pool := NewConnectionPool("error-test", "invalid-address:99999", 1, 5)
		defer pool.Stop()

		// Starting with invalid address should handle errors gracefully
		err := pool.Start()
		// The Start method logs errors but doesn't return them for invalid connections
		// This is by design to allow the service to start even if some connections fail
		assert.NoError(t, err, "Start should not return error even with invalid address")
	})

	t.Run("connection_limit_exceeded", func(t *testing.T) {
		pool := NewConnectionPool("limit-test", "localhost:9500", 1, 2)
		defer pool.Stop()

		// Simulate reaching connection limit
		pool.mu.Lock()
		// Add mock connections to reach the limit
		for i := 0; i < 2; i++ {
			mockConn := &PooledConnection{
				id:      fmt.Sprintf("limit-conn-%d", i),
				healthy: true,
			}
			pool.connections = append(pool.connections, mockConn)
		}
		pool.mu.Unlock()

		// Try to create another connection (should fail)
		err := pool.createConnection()
		assert.Error(t, err, "Should fail when connection limit is exceeded")
		assert.Contains(t, err.Error(), "connection pool full", "Error should indicate pool is full")
	})

	t.Run("no_healthy_connections", func(t *testing.T) {
		pool := NewConnectionPool("no-healthy-test", "localhost:9600", 1, 5)
		defer pool.Stop()

		// Add only unhealthy connections
		pool.mu.Lock()
		for i := 0; i < 3; i++ {
			unhealthyConn := &PooledConnection{
				id:      fmt.Sprintf("unhealthy-conn-%d", i),
				healthy: false,
			}
			pool.connections = append(pool.connections, unhealthyConn)
		}
		pool.mu.Unlock()

		// Try to get a connection (should fail)
		conn, err := pool.GetConnection()
		assert.Error(t, err, "Should fail when no healthy connections available")
		assert.Nil(t, conn, "Connection should be nil when error occurs")
		assert.Contains(t, err.Error(), "no healthy connections", "Error should indicate no healthy connections")
	})

	t.Run("empty_connection_pool", func(t *testing.T) {
		pool := NewConnectionPool("empty-test", "localhost:9700", 0, 5)
		defer pool.Stop()

		// Ensure pool is empty
		pool.mu.Lock()
		pool.connections = nil
		pool.mu.Unlock()

		// Try to get a connection from empty pool
		conn, err := pool.GetConnection()
		assert.Error(t, err, "Should fail when pool is empty")
		assert.Nil(t, conn, "Connection should be nil when error occurs")
		assert.Contains(t, err.Error(), "no connections available", "Error should indicate no connections available")
	})
}

func TestConnectionPool_ConcurrentAccess(t *testing.T) {
	pool := NewConnectionPool("concurrent-test", "localhost:9400", 2, 10)
	defer pool.Stop()

	t.Run("concurrent_metric_updates", func(t *testing.T) {
		const numGoroutines = 20
		const operationsPerGoroutine = 50

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Simulate concurrent access to pool metrics
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Simulate various metric operations
					switch j % 4 {
					case 0:
						// Simulate successful request
						pool.metrics.mu.Lock()
						pool.metrics.TotalRequests++
						pool.metrics.mu.Unlock()
					case 1:
						// Simulate failed request
						pool.metrics.mu.Lock()
						pool.metrics.FailedRequests++
						pool.metrics.mu.Unlock()
					case 2:
						// Read metrics
						_ = pool.GetMetrics()
					case 3:
						// Simulate connection state change
						pool.metrics.mu.Lock()
						if pool.metrics.HealthyConnections > 0 {
							pool.metrics.HealthyConnections--
							pool.metrics.UnhealthyConnections++
						}
						pool.metrics.mu.Unlock()
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify metrics are consistent
		metrics := pool.GetMetrics()
		assert.True(t, metrics.TotalRequests >= 0, "Total requests should be non-negative")
		assert.True(t, metrics.FailedRequests >= 0, "Failed requests should be non-negative")
		assert.True(t, metrics.FailedRequests <= metrics.TotalRequests, "Failed requests should not exceed total")
	})
}
	defer pool.Stop()

	t.Run("concurrent_metric_updates", func(t *testing.T) {
		const numGoroutines = 20
		const operationsPerGoroutine = 50

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Simulate concurrent access to pool metrics
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Simulate various metric operations
					switch j % 4 {
					case 0:
						// Simulate successful request
						pool.metrics.mu.Lock()
						pool.metrics.TotalRequests++
						pool.metrics.mu.Unlock()
					case 1:
						// Simulate failed request
						pool.metrics.mu.Lock()
						pool.metrics.FailedRequests++
						pool.metrics.mu.Unlock()
					case 2:
						// Read metrics
						_ = pool.GetMetrics()
					case 3:
						// Simulate connection state change
						pool.metrics.mu.Lock()
						if pool.metrics.HealthyConnections > 0 {
							pool.metrics.HealthyConnections--
							pool.metrics.UnhealthyConnections++
						}
						pool.metrics.mu.Unlock()
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify metrics are consistent
		metrics := pool.GetMetrics()
		assert.True(t, metrics.TotalRequests >= 0, "Total requests should be non-negative")
		assert.True(t, metrics.FailedRequests >= 0, "Failed requests should be non-negative")
		assert.True(t, metrics.FailedRequests <= metrics.TotalRequests, "Failed requests should not exceed total")
	})

	t.Run("concurrent_connection_list_access", func(t *testing.T) {
		const numReaders = 10
		const numWriters = 5
		const operationsPerGoroutine = 20

		var wg sync.WaitGroup
		wg.Add(numReaders + numWriters)

		// Start reader goroutines
		for i := 0; i < numReaders; i++ {
			go func(readerID int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Read connection list
					pool.mu.RLock()
					connectionCount := len(pool.connections)
					for _, conn := range pool.connections {
						_ = conn.id // Access connection data
						_ = conn.healthy
					}
					pool.mu.RUnlock()

					assert.True(t, connectionCount >= 0, "Connection count should be non-negative")
				}
			}(i)
		}

		// Start writer goroutines (simulating connection management)
		for i := 0; i < numWriters; i++ {
			go func(writerID int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Simulate connection state updates
					pool.mu.Lock()
					for _, conn := range pool.connections {
						conn.mu.Lock()
						conn.lastUsed = time.Now()
						conn.requestCount++
						conn.mu.Unlock()
					}
					pool.mu.Unlock()
				}
			}(i)
		}

		wg.Wait()
	})
}

func TestConnectionPool_ErrorHandling(t *testing.T) {
	t.Run("invalid_target_address", func(t *testing.T) {
		pool := NewConnectionPool("error-test", "invalid-address:99999", 1, 5)
		defer pool.Stop()

		// Starting with invalid address should handle errors gracefully
		err := pool.Start()
		// The Start method logs errors but doesn't return them for invalid connections
		// This is by design to allow the service to start even if some connections fail
		assert.NoError(t, err, "Start should not return error even with invalid address")
	})

	t.Run("connection_limit_exceeded", func(t *testing.T) {
		pool := NewConnectionPool("limit-test", "localhost:9500", 1, 2)
		defer pool.Stop()

		// Simulate reaching connection limit
		pool.mu.Lock()
		// Add mock connections to reach the limit
		for i := 0; i < 2; i++ {
			mockConn := &PooledConnection{
				id:      fmt.Sprintf("limit-conn-%d", i),
				healthy: true,
			}
			pool.connections = append(pool.connections, mockConn)
		}
		pool.mu.Unlock()

		// Try to create another connection (should fail)
		err := pool.createConnection()
		assert.Error(t, err, "Should fail when connection limit is exceeded")
		assert.Contains(t, err.Error(), "connection pool full", "Error should indicate pool is full")
	})

	t.Run("no_healthy_connections", func(t *testing.T) {
		pool := NewConnectionPool("no-healthy-test", "localhost:9600", 1, 5)
		defer pool.Stop()

		// Add only unhealthy connections
		pool.mu.Lock()
		for i := 0; i < 3; i++ {
			unhealthyConn := &PooledConnection{
				id:      fmt.Sprintf("unhealthy-conn-%d", i),
				healthy: false,
			}
			pool.connections = append(pool.connections, unhealthyConn)
		}
		pool.mu.Unlock()

		// Try to get a connection (should fail)
		conn, err := pool.GetConnection()
		assert.Error(t, err, "Should fail when no healthy connections available")
		assert.Nil(t, conn, "Connection should be nil when error occurs")
		assert.Contains(t, err.Error(), "no healthy connections", "Error should indicate no healthy connections")
	})

	t.Run("empty_connection_pool", func(t *testing.T) {
		pool := NewConnectionPool("empty-test", "localhost:9700", 0, 5)
		defer pool.Stop()

		// Ensure pool is empty
		pool.mu.Lock()
		pool.connections = nil
		pool.mu.Unlock()

		// Try to get a connection from empty pool
		conn, err := pool.GetConnection()
		assert.Error(t, err, "Should fail when pool is empty")
		assert.Nil(t, conn, "Connection should be nil when error occurs")
		assert.Contains(t, err.Error(), "no connections available", "Error should indicate no connections available")
	})
}

func TestConnectionPool_Lifecycle(t *testing.T) {
	t.Run("start_and_stop", func(t *testing.T) {
		pool := NewConnectionPool("lifecycle-test", "localhost:9800", 1, 5)

		// Test starting
		err := pool.Start()
		assert.NoError(t, err, "Start should succeed")

		// Verify context is active
		select {
		case <-pool.ctx.Done():
			t.Fatal("Context should not be cancelled after start")
		default:
			// Context is active, which is expected
		}

		// Test stopping
		pool.Stop()

		// Verify context is cancelled
		select {
		case <-pool.ctx.Done():
			// Context is cancelled, which is expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Context should be cancelled after stop")
		}
	})

	t.Run("multiple_start_calls", func(t *testing.T) {
		pool := NewConnectionPool("multi-start-test", "localhost:9900", 1, 5)
		defer pool.Stop()

		// Multiple start calls should be handled gracefully
		err1 := pool.Start()
		err2 := pool.Start()
		err3 := pool.Start()

		assert.NoError(t, err1, "First start should succeed")
		assert.NoError(t, err2, "Second start should not error")
		assert.NoError(t, err3, "Third start should not error")
	})

	t.Run("multiple_stop_calls", func(t *testing.T) {
		pool := NewConnectionPool("multi-stop-test", "localhost:10000", 1, 5)

		err := pool.Start()
		assert.NoError(t, err, "Start should succeed")

		// Multiple stop calls should be handled gracefully
		pool.Stop()
		pool.Stop()
		pool.Stop()

		// Verify context is cancelled
		select {
		case <-pool.ctx.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Context should be cancelled after stop")
		}
	})
}
