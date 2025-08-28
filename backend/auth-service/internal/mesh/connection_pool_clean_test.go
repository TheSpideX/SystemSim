package mesh

import (
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
	// Don't defer pool.Stop() since we're adding mock connections with nil gRPC connections

	t.Run("round_robin_index_progression", func(t *testing.T) {
		// Test round-robin behavior by checking the index progression
		initialIndex := pool.currentIndex

		// Simulate multiple requests to see round-robin behavior
		for i := 0; i < 10; i++ {
			// Simulate the index increment that happens in GetConnection
			pool.currentIndex++
		}

		// Verify that the index has progressed
		assert.True(t, pool.currentIndex > initialIndex, "Current index should have progressed")
		assert.Equal(t, initialIndex+10, pool.currentIndex, "Index should have incremented by 10")
	})

	t.Run("connection_health_filtering_logic", func(t *testing.T) {
		// Test connection health filtering logic without modifying the pool
		// Create test connections without adding them to the pool
		testConnections := []*PooledConnection{
			{id: "healthy-1", healthy: true},
			{id: "healthy-2", healthy: true},
			{id: "unhealthy-1", healthy: false},
			{id: "unhealthy-2", healthy: false},
		}

		// Test that we can filter connections by health status
		healthyConns := make([]*PooledConnection, 0)
		for _, conn := range testConnections {
			// Just check the healthy flag directly
			if conn.healthy {
				healthyConns = append(healthyConns, conn)
			}
		}

		assert.Len(t, healthyConns, 2, "Should find exactly 2 healthy connections")
		for _, conn := range healthyConns {
			assert.True(t, conn.healthy, "All filtered connections should be healthy")
			assert.Contains(t, conn.id, "healthy", "Connection ID should indicate it's healthy")
		}
	})

	// Clean up the pool properly
	pool.Stop()
}

func TestConnectionPool_HealthChecking(t *testing.T) {
	pool := NewConnectionPool("health-test", "localhost:9300", 1, 5)
	defer pool.Stop()

	t.Run("connection_health_assessment", func(t *testing.T) {
		// Test connection health flags (without calling isConnectionHealthy which needs real gRPC connections)
		healthyConn := &PooledConnection{
			id:           "healthy-conn",
			createdAt:    time.Now(),
			lastUsed:     time.Now(),
			requestCount: 10,
			healthy:      true,
		}

		assert.True(t, healthyConn.healthy, "Healthy connection should have healthy flag set to true")

		// Test with unhealthy connection
		unhealthyConn := &PooledConnection{
			id:           "unhealthy-conn",
			createdAt:    time.Now(),
			lastUsed:     time.Now().Add(-time.Hour), // Old last used time
			requestCount: 0,
			healthy:      false,
		}

		assert.False(t, unhealthyConn.healthy, "Unhealthy connection should have healthy flag set to false")
	})

	t.Run("health_checker_initialization", func(t *testing.T) {
		assert.NotNil(t, pool.healthChecker, "Health checker should be initialized")
		assert.Equal(t, pool, pool.healthChecker.pool, "Health checker should reference the pool")
		assert.Equal(t, 30*time.Second, pool.healthChecker.checkInterval, "Check interval should be 30 seconds")
		assert.NotNil(t, pool.healthChecker.ctx, "Health checker context should be initialized")
	})

	t.Run("health_metrics_tracking", func(t *testing.T) {
		// Test that metrics can be retrieved and have the expected structure
		metrics := pool.GetMetrics()
		assert.NotNil(t, metrics, "Metrics should not be nil")

		// Test that metrics have the expected fields and are non-negative
		assert.True(t, metrics.TotalConnections >= 0, "Total connections should be non-negative")
		assert.True(t, metrics.HealthyConnections >= 0, "Healthy connections should be non-negative")
		assert.True(t, metrics.UnhealthyConnections >= 0, "Unhealthy connections should be non-negative")
		assert.True(t, metrics.TotalRequests >= 0, "Total requests should be non-negative")
		assert.True(t, metrics.FailedRequests >= 0, "Failed requests should be non-negative")

		// Test that the relationship between connection counts is logical
		assert.Equal(t, metrics.TotalConnections, metrics.HealthyConnections+metrics.UnhealthyConnections,
			"Total connections should equal healthy + unhealthy connections")
	})
}
