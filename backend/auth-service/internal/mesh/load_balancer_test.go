package mesh

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBalancer_Creation(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		strategy    LoadBalancingStrategy
		expectError bool
		errorMsg    string
	}{
		{
			name:        "round_robin_strategy",
			serviceName: "rr-service",
			strategy:    RoundRobin,
			expectError: false,
		},
		{
			name:        "least_connections_strategy",
			serviceName: "lc-service",
			strategy:    LeastConnections,
			expectError: false,
		},
		{
			name:        "weighted_round_robin_strategy",
			serviceName: "wrr-service",
			strategy:    WeightedRoundRobin,
			expectError: false,
		},
		{
			name:        "random_strategy",
			serviceName: "random-service",
			strategy:    Random,
			expectError: false,
		},
		{
			name:        "empty_service_name",
			serviceName: "",
			strategy:    RoundRobin,
			expectError: false, // Should be allowed for testing
		},
		{
			name:        "invalid_strategy",
			serviceName: "invalid-service",
			strategy:    LoadBalancingStrategy(999),
			expectError: false, // Should default to round robin
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := NewLoadBalancer(tt.serviceName, tt.strategy)

			if tt.expectError {
				assert.Nil(t, lb, "Load balancer should be nil for invalid parameters")
			} else {
				assert.NotNil(t, lb, "Load balancer should not be nil")
				assert.Equal(t, tt.serviceName, lb.serviceName)
				
				expectedStrategy := tt.strategy
				if expectedStrategy < RoundRobin || expectedStrategy > Random {
					expectedStrategy = RoundRobin // Default strategy
				}
				assert.Equal(t, expectedStrategy, lb.strategy)
				
				assert.NotNil(t, lb.backends, "Backends slice should be initialized")
				assert.Equal(t, 0, len(lb.backends), "Initial backends should be empty")
				assert.Equal(t, int64(0), lb.currentIndex, "Initial current index should be 0")
				assert.NotNil(t, lb.metrics, "Metrics should be initialized")
			}
		})
	}
}

func TestLoadBalancer_BackendManagement(t *testing.T) {
	lb := NewLoadBalancer("backend-test", RoundRobin)

	t.Run("add_backends", func(t *testing.T) {
		backends := []*Backend{
			{ID: "backend-1", Address: "localhost:9001", Weight: 1, Healthy: true},
			{ID: "backend-2", Address: "localhost:9002", Weight: 2, Healthy: true},
			{ID: "backend-3", Address: "localhost:9003", Weight: 1, Healthy: false},
		}

		for _, backend := range backends {
			err := lb.AddBackend(backend)
			assert.NoError(t, err, "Adding backend should succeed")
		}

		assert.Len(t, lb.backends, 3, "Should have 3 backends")
		
		// Verify backends are stored correctly
		for i, backend := range backends {
			assert.Equal(t, backend.ID, lb.backends[i].ID)
			assert.Equal(t, backend.Address, lb.backends[i].Address)
			assert.Equal(t, backend.Weight, lb.backends[i].Weight)
			assert.Equal(t, backend.Healthy, lb.backends[i].Healthy)
		}
	})

	t.Run("add_duplicate_backend", func(t *testing.T) {
		duplicateBackend := &Backend{
			ID: "backend-1", // Same ID as existing backend
			Address: "localhost:9004",
			Weight: 1,
			Healthy: true,
		}

		err := lb.AddBackend(duplicateBackend)
		assert.Error(t, err, "Adding duplicate backend should fail")
		assert.Contains(t, err.Error(), "backend already exists", "Error should indicate duplicate backend")
		
		// Verify backend count didn't change
		assert.Len(t, lb.backends, 3, "Backend count should remain unchanged")
	})

	t.Run("remove_backend", func(t *testing.T) {
		err := lb.RemoveBackend("backend-2")
		assert.NoError(t, err, "Removing existing backend should succeed")
		
		assert.Len(t, lb.backends, 2, "Should have 2 backends after removal")
		
		// Verify correct backend was removed
		for _, backend := range lb.backends {
			assert.NotEqual(t, "backend-2", backend.ID, "Removed backend should not be present")
		}
	})

	t.Run("remove_nonexistent_backend", func(t *testing.T) {
		err := lb.RemoveBackend("nonexistent-backend")
		assert.Error(t, err, "Removing nonexistent backend should fail")
		assert.Contains(t, err.Error(), "backend not found", "Error should indicate backend not found")
		
		// Verify backend count didn't change
		assert.Len(t, lb.backends, 2, "Backend count should remain unchanged")
	})

	t.Run("update_backend_health", func(t *testing.T) {
		err := lb.UpdateBackendHealth("backend-1", false)
		assert.NoError(t, err, "Updating backend health should succeed")
		
		// Find and verify the backend
		var updatedBackend *Backend
		for _, backend := range lb.backends {
			if backend.ID == "backend-1" {
				updatedBackend = backend
				break
			}
		}
		
		require.NotNil(t, updatedBackend, "Backend should be found")
		assert.False(t, updatedBackend.Healthy, "Backend health should be updated to false")
	})

	t.Run("update_nonexistent_backend_health", func(t *testing.T) {
		err := lb.UpdateBackendHealth("nonexistent-backend", true)
		assert.Error(t, err, "Updating nonexistent backend health should fail")
		assert.Contains(t, err.Error(), "backend not found", "Error should indicate backend not found")
	})
}

func TestLoadBalancer_RoundRobinStrategy(t *testing.T) {
	lb := NewLoadBalancer("rr-test", RoundRobin)

	// Add healthy backends
	backends := []*Backend{
		{ID: "rr-1", Address: "localhost:9001", Weight: 1, Healthy: true},
		{ID: "rr-2", Address: "localhost:9002", Weight: 1, Healthy: true},
		{ID: "rr-3", Address: "localhost:9003", Weight: 1, Healthy: true},
	}

	for _, backend := range backends {
		err := lb.AddBackend(backend)
		require.NoError(t, err)
	}

	t.Run("round_robin_selection", func(t *testing.T) {
		// Test round-robin behavior
		selectedBackends := make([]string, 6)
		for i := 0; i < 6; i++ {
			backend, err := lb.SelectBackend()
			assert.NoError(t, err, "Backend selection should succeed")
			assert.NotNil(t, backend, "Selected backend should not be nil")
			selectedBackends[i] = backend.ID
		}

		// Verify round-robin pattern: should cycle through backends twice
		expectedPattern := []string{"rr-1", "rr-2", "rr-3", "rr-1", "rr-2", "rr-3"}
		assert.Equal(t, expectedPattern, selectedBackends, "Should follow round-robin pattern")
	})

	t.Run("skip_unhealthy_backends", func(t *testing.T) {
		// Mark one backend as unhealthy
		err := lb.UpdateBackendHealth("rr-2", false)
		require.NoError(t, err)

		// Test selection with unhealthy backend
		selectedBackends := make([]string, 4)
		for i := 0; i < 4; i++ {
			backend, err := lb.SelectBackend()
			assert.NoError(t, err, "Backend selection should succeed")
			assert.NotNil(t, backend, "Selected backend should not be nil")
			selectedBackends[i] = backend.ID
		}

		// Should skip unhealthy backend
		expectedPattern := []string{"rr-3", "rr-1", "rr-3", "rr-1"}
		assert.Equal(t, expectedPattern, selectedBackends, "Should skip unhealthy backend")
	})
}

func TestLoadBalancer_LeastConnectionsStrategy(t *testing.T) {
	lb := NewLoadBalancer("lc-test", LeastConnections)

	// Add backends with different connection counts
	backends := []*Backend{
		{ID: "lc-1", Address: "localhost:9001", Weight: 1, Healthy: true, ActiveConnections: 5},
		{ID: "lc-2", Address: "localhost:9002", Weight: 1, Healthy: true, ActiveConnections: 2},
		{ID: "lc-3", Address: "localhost:9003", Weight: 1, Healthy: true, ActiveConnections: 8},
	}

	for _, backend := range backends {
		err := lb.AddBackend(backend)
		require.NoError(t, err)
	}

	t.Run("least_connections_selection", func(t *testing.T) {
		// Should select backend with least connections (lc-2 with 2 connections)
		backend, err := lb.SelectBackend()
		assert.NoError(t, err, "Backend selection should succeed")
		assert.NotNil(t, backend, "Selected backend should not be nil")
		assert.Equal(t, "lc-2", backend.ID, "Should select backend with least connections")
	})

	t.Run("update_connections_and_reselect", func(t *testing.T) {
		// Update connection counts
		lb.mu.Lock()
		for _, backend := range lb.backends {
			switch backend.ID {
			case "lc-1":
				backend.ActiveConnections = 1 // Now has least connections
			case "lc-2":
				backend.ActiveConnections = 10 // Now has most connections
			case "lc-3":
				backend.ActiveConnections = 5 // Middle
			}
		}
		lb.mu.Unlock()

		// Should now select lc-1
		backend, err := lb.SelectBackend()
		assert.NoError(t, err, "Backend selection should succeed")
		assert.NotNil(t, backend, "Selected backend should not be nil")
		assert.Equal(t, "lc-1", backend.ID, "Should select backend with updated least connections")
	})
}

func TestLoadBalancer_WeightedRoundRobinStrategy(t *testing.T) {
	lb := NewLoadBalancer("wrr-test", WeightedRoundRobin)

	// Add backends with different weights
	backends := []*Backend{
		{ID: "wrr-1", Address: "localhost:9001", Weight: 1, Healthy: true},
		{ID: "wrr-2", Address: "localhost:9002", Weight: 3, Healthy: true}, // Higher weight
		{ID: "wrr-3", Address: "localhost:9003", Weight: 2, Healthy: true},
	}

	for _, backend := range backends {
		err := lb.AddBackend(backend)
		require.NoError(t, err)
	}

	t.Run("weighted_distribution", func(t *testing.T) {
		// Collect selections over multiple rounds
		selections := make(map[string]int)
		totalSelections := 60 // Multiple of total weight (1+3+2=6)

		for i := 0; i < totalSelections; i++ {
			backend, err := lb.SelectBackend()
			assert.NoError(t, err, "Backend selection should succeed")
			assert.NotNil(t, backend, "Selected backend should not be nil")
			selections[backend.ID]++
		}

		// Verify distribution matches weights
		totalWeight := 6
		expectedWrr1 := totalSelections * 1 / totalWeight // 10 selections
		expectedWrr2 := totalSelections * 3 / totalWeight // 30 selections
		expectedWrr3 := totalSelections * 2 / totalWeight // 20 selections

		// Allow some tolerance for weighted round-robin implementation
		tolerance := 2
		assert.InDelta(t, expectedWrr1, selections["wrr-1"], float64(tolerance), "wrr-1 should get proportional selections")
		assert.InDelta(t, expectedWrr2, selections["wrr-2"], float64(tolerance), "wrr-2 should get proportional selections")
		assert.InDelta(t, expectedWrr3, selections["wrr-3"], float64(tolerance), "wrr-3 should get proportional selections")
	})
}

func TestLoadBalancer_ConcurrentAccess(t *testing.T) {
	lb := NewLoadBalancer("concurrent-test", RoundRobin)

	// Add backends
	backends := []*Backend{
		{ID: "conc-1", Address: "localhost:9001", Weight: 1, Healthy: true},
		{ID: "conc-2", Address: "localhost:9002", Weight: 1, Healthy: true},
		{ID: "conc-3", Address: "localhost:9003", Weight: 1, Healthy: true},
	}

	for _, backend := range backends {
		err := lb.AddBackend(backend)
		require.NoError(t, err)
	}

	t.Run("concurrent_backend_selection", func(t *testing.T) {
		const numGoroutines = 20
		const selectionsPerGoroutine = 50

		var wg sync.WaitGroup
		results := make(chan string, numGoroutines*selectionsPerGoroutine)

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < selectionsPerGoroutine; j++ {
					backend, err := lb.SelectBackend()
					if err == nil && backend != nil {
						results <- backend.ID
					}
				}
			}()
		}

		wg.Wait()
		close(results)

		// Collect results
		selections := make(map[string]int)
		totalSelections := 0
		for backendID := range results {
			selections[backendID]++
			totalSelections++
		}

		// Verify all backends were selected
		assert.True(t, selections["conc-1"] > 0, "conc-1 should be selected")
		assert.True(t, selections["conc-2"] > 0, "conc-2 should be selected")
		assert.True(t, selections["conc-3"] > 0, "conc-3 should be selected")
		
		// Verify total selections
		assert.Equal(t, numGoroutines*selectionsPerGoroutine, totalSelections, "Total selections should match expected")
	})

	t.Run("concurrent_backend_management", func(t *testing.T) {
		const numGoroutines = 10

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Concurrent backend additions and removals
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()
				
				// Add a backend
				backend := &Backend{
					ID:      fmt.Sprintf("temp-%d", goroutineID),
					Address: fmt.Sprintf("localhost:%d", 10000+goroutineID),
					Weight:  1,
					Healthy: true,
				}
				
				err := lb.AddBackend(backend)
				if err == nil {
					// If addition succeeded, try to remove it
					time.Sleep(10 * time.Millisecond) // Small delay
					lb.RemoveBackend(backend.ID)
				}
			}(i)
		}

		wg.Wait()

		// Verify original backends are still present
		lb.mu.RLock()
		backendCount := len(lb.backends)
		originalBackendsPresent := 0
		for _, backend := range lb.backends {
			if backend.ID == "conc-1" || backend.ID == "conc-2" || backend.ID == "conc-3" {
				originalBackendsPresent++
			}
		}
		lb.mu.RUnlock()

		assert.Equal(t, 3, originalBackendsPresent, "Original backends should still be present")
		assert.True(t, backendCount >= 3, "Should have at least original backends")
	})
}
