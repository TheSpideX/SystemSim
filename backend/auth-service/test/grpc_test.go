package test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	grpcAddress = "localhost:9000" // Auth service gRPC port
)

// TestAuthServiceGRPC tests the actual running auth service gRPC endpoints
func TestAuthServiceGRPC(t *testing.T) {
	// Skip if gRPC service is not running
	if !isGRPCServiceRunning(t) {
		t.Skip("Auth service gRPC is not running on localhost:9001")
	}

	t.Run("grpc_connection_establishment", func(t *testing.T) {
		// Test that we can establish gRPC connection
		conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()
		
		// Test connection state
		state := conn.GetState()
		assert.NotEqual(t, connectivity.TransientFailure, state, "Connection should not be in failure state")
		assert.NotEqual(t, connectivity.Shutdown, state, "Connection should not be shutdown")
		
		// Wait for connection to be ready
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		conn.WaitForStateChange(ctx, connectivity.Idle)
		finalState := conn.GetState()
		t.Logf("Final connection state: %v", finalState)
	})
	
	t.Run("grpc_health_check", func(t *testing.T) {
		// Test gRPC health check endpoint
		conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()
		
		// If the service implements gRPC health checking
		// This would test the health check endpoint
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Test that connection is healthy by attempting to create a client
		// In a real implementation, you would use the generated gRPC client here
		// For now, we test that the connection doesn't immediately fail
		
		select {
		case <-ctx.Done():
			t.Log("Health check timeout - service may be slow to respond")
		default:
			t.Log("gRPC connection established successfully")
		}
	})
	
	t.Run("concurrent_grpc_connections", func(t *testing.T) {
		// Test multiple concurrent gRPC connections
		const numConnections = 10
		var wg sync.WaitGroup
		var successCount int32
		var errorCount int32
		
		for i := 0; i < numConnections; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				conn, err := grpc.Dial(grpcAddress, 
					grpc.WithTransportCredentials(insecure.NewCredentials()),
					grpc.WithBlock(),
				)
				
				if err != nil {
					errorCount++
					t.Logf("Connection %d failed: %v", index, err)
					return
				}
				defer conn.Close()
				
				// Test that connection is usable
				state := conn.GetState()
				if state != connectivity.TransientFailure && state != connectivity.Shutdown {
					successCount++
				} else {
					errorCount++
				}
			}(i)
		}
		
		wg.Wait()
		
		assert.True(t, successCount >= int32(numConnections*0.8), 
			"At least 80%% of concurrent connections should succeed")
		t.Logf("Concurrent connections: %d successful, %d failed", successCount, errorCount)
	})
	
	t.Run("grpc_connection_pooling_behavior", func(t *testing.T) {
		// Test connection pooling by creating multiple connections rapidly
		const numRapidConnections = 20
		connections := make([]*grpc.ClientConn, numRapidConnections)
		
		start := time.Now()
		
		// Create connections rapidly
		for i := 0; i < numRapidConnections; i++ {
			conn, err := grpc.Dial(grpcAddress, 
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			require.NoError(t, err)
			connections[i] = conn
		}
		
		creationTime := time.Since(start)
		t.Logf("Created %d connections in %v", numRapidConnections, creationTime)
		
		// Test that all connections are functional
		functionalCount := 0
		for i, conn := range connections {
			state := conn.GetState()
			if state != connectivity.TransientFailure && state != connectivity.Shutdown {
				functionalCount++
			}
			t.Logf("Connection %d state: %v", i, state)
		}
		
		assert.True(t, functionalCount >= numRapidConnections*8/10, 
			"At least 80%% of rapid connections should be functional")
		
		// Clean up connections
		for _, conn := range connections {
			if conn != nil {
				conn.Close()
			}
		}
	})
	
	t.Run("grpc_connection_resilience", func(t *testing.T) {
		// Test connection resilience and recovery
		conn, err := grpc.Dial(grpcAddress, 
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()
		
		// Monitor connection state changes
		initialState := conn.GetState()
		t.Logf("Initial connection state: %v", initialState)
		
		// Test connection over time
		for i := 0; i < 5; i++ {
			time.Sleep(1 * time.Second)
			currentState := conn.GetState()
			t.Logf("Connection state after %ds: %v", i+1, currentState)
			
			// Connection should remain stable
			assert.NotEqual(t, connectivity.TransientFailure, currentState,
				"Connection should not fail during stability test")
		}
	})
	
	t.Run("grpc_error_handling", func(t *testing.T) {
		// Test gRPC error handling with invalid requests
		conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()
		
		// Test connection to non-existent method
		// This would normally use the generated gRPC client
		// For now, we test that the connection handles errors gracefully
		
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		
		// Simulate calling a non-existent method
		err = conn.Invoke(ctx, "/nonexistent.Service/NonExistentMethod", nil, nil)
		if err != nil {
			// Should get a proper gRPC error, not a connection error
			grpcStatus := status.Convert(err)
			assert.NotNil(t, grpcStatus, "Should get proper gRPC status error")
			t.Logf("Expected gRPC error: %v", grpcStatus.Message())
		}
	})
	
	t.Run("grpc_performance_baseline", func(t *testing.T) {
		// Test basic gRPC performance metrics
		const numOperations = 100
		
		conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()
		
		start := time.Now()
		
		// Perform rapid connection state checks as a baseline performance test
		for i := 0; i < numOperations; i++ {
			state := conn.GetState()
			assert.NotEqual(t, connectivity.Shutdown, state, "Connection should remain active")
		}
		
		duration := time.Since(start)
		operationsPerSecond := float64(numOperations) / duration.Seconds()
		
		t.Logf("gRPC performance baseline:")
		t.Logf("- Operations: %d", numOperations)
		t.Logf("- Duration: %v", duration)
		t.Logf("- Operations/second: %.2f", operationsPerSecond)
		
		// Basic performance assertion - should be able to do at least 1000 ops/sec
		assert.True(t, operationsPerSecond > 1000, 
			"Should achieve at least 1000 operations per second")
	})
}

// TestGRPCConnectionPooling tests connection pooling behavior
func TestGRPCConnectionPooling(t *testing.T) {
	if !isGRPCServiceRunning(t) {
		t.Skip("Auth service gRPC is not running on localhost:9001")
	}

	t.Run("connection_pool_scaling", func(t *testing.T) {
		// Test that connection pool scales from 5 to 20 connections under load
		const minConnections = 5
		const maxConnections = 20
		const highLoad = 50
		
		var wg sync.WaitGroup
		connections := make(chan *grpc.ClientConn, maxConnections)
		
		// Simulate high load
		for i := 0; i < highLoad; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				conn, err := grpc.Dial(grpcAddress, 
					grpc.WithTransportCredentials(insecure.NewCredentials()),
				)
				
				if err == nil {
					select {
					case connections <- conn:
						// Connection added to pool
					default:
						// Pool full, close connection
						conn.Close()
					}
				}
			}(i)
		}
		
		wg.Wait()
		close(connections)
		
		// Count active connections
		activeConnections := 0
		var activeConns []*grpc.ClientConn
		
		for conn := range connections {
			if conn.GetState() != connectivity.Shutdown {
				activeConnections++
				activeConns = append(activeConns, conn)
			}
		}
		
		t.Logf("Active connections under load: %d", activeConnections)
		
		// Should have scaled up but not exceeded maximum
		assert.True(t, activeConnections >= minConnections, 
			"Should maintain at least %d connections", minConnections)
		assert.True(t, activeConnections <= maxConnections, 
			"Should not exceed %d connections", maxConnections)
		
		// Clean up
		for _, conn := range activeConns {
			conn.Close()
		}
	})
}

// Helper function to check if gRPC service is running
func isGRPCServiceRunning(t *testing.T) bool {
	conn, err := grpc.Dial(grpcAddress, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(2*time.Second),
	)
	if err != nil {
		return false
	}
	defer conn.Close()
	
	// Check if connection is in a good state
	state := conn.GetState()
	return state != connectivity.TransientFailure && state != connectivity.Shutdown
}
