package mesh

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthMonitor_Creation(t *testing.T) {
	tests := []struct {
		name           string
		serviceName    string
		checkInterval  time.Duration
		expectError    bool
		errorMsg       string
	}{
		{
			name:          "valid_monitor_creation",
			serviceName:   "test-service",
			checkInterval: 30 * time.Second,
			expectError:   false,
		},
		{
			name:          "short_check_interval",
			serviceName:   "fast-service",
			checkInterval: 1 * time.Second,
			expectError:   false,
		},
		{
			name:          "long_check_interval",
			serviceName:   "slow-service",
			checkInterval: 5 * time.Minute,
			expectError:   false,
		},
		{
			name:          "zero_check_interval",
			serviceName:   "zero-service",
			checkInterval: 0,
			expectError:   false, // Should use default interval
		},
		{
			name:          "negative_check_interval",
			serviceName:   "negative-service",
			checkInterval: -1 * time.Second,
			expectError:   false, // Should use default interval
		},
		{
			name:          "empty_service_name",
			serviceName:   "",
			checkInterval: 30 * time.Second,
			expectError:   false, // Should be allowed for testing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := NewHealthMonitor(tt.serviceName, tt.checkInterval)

			if tt.expectError {
				assert.Nil(t, monitor, "Monitor should be nil for invalid parameters")
			} else {
				assert.NotNil(t, monitor, "Monitor should not be nil")
				assert.Equal(t, tt.serviceName, monitor.serviceName)
				
				expectedInterval := tt.checkInterval
				if expectedInterval <= 0 {
					expectedInterval = 30 * time.Second // Default interval
				}
				assert.Equal(t, expectedInterval, monitor.checkInterval)
				
				assert.NotNil(t, monitor.healthStatus, "Health status should be initialized")
				assert.NotNil(t, monitor.metrics, "Metrics should be initialized")
				assert.NotNil(t, monitor.ctx, "Context should be initialized")
				assert.NotNil(t, monitor.cancel, "Cancel function should be initialized")

				// Clean up
				monitor.Stop()
			}
		})
	}
}

func TestHealthMonitor_HealthStatus(t *testing.T) {
	monitor := NewHealthMonitor("status-test", 1*time.Second)
	defer monitor.Stop()

	t.Run("initial_health_status", func(t *testing.T) {
		status := monitor.GetHealthStatus()
		assert.NotNil(t, status, "Health status should not be nil")
		assert.Equal(t, "status-test", status.ServiceName)
		assert.False(t, status.IsHealthy, "Initial health status should be false")
		assert.Equal(t, int64(0), status.TotalChecks, "Initial total checks should be 0")
		assert.Equal(t, int64(0), status.FailedChecks, "Initial failed checks should be 0")
		assert.True(t, status.LastCheck.IsZero(), "Initial last check should be zero time")
		assert.Empty(t, status.LastError, "Initial last error should be empty")
	})

	t.Run("health_status_updates", func(t *testing.T) {
		// Simulate health check updates
		monitor.healthStatus.mu.Lock()
		monitor.healthStatus.IsHealthy = true
		monitor.healthStatus.TotalChecks = 10
		monitor.healthStatus.FailedChecks = 2
		monitor.healthStatus.LastCheck = time.Now()
		monitor.healthStatus.LastError = "test error"
		monitor.healthStatus.mu.Unlock()

		status := monitor.GetHealthStatus()
		assert.True(t, status.IsHealthy, "Health status should be updated to true")
		assert.Equal(t, int64(10), status.TotalChecks, "Total checks should be updated")
		assert.Equal(t, int64(2), status.FailedChecks, "Failed checks should be updated")
		assert.False(t, status.LastCheck.IsZero(), "Last check should be updated")
		assert.Equal(t, "test error", status.LastError, "Last error should be updated")
	})

	t.Run("concurrent_health_status_access", func(t *testing.T) {
		const numGoroutines = 20
		const operationsPerGoroutine = 50

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Simulate concurrent health status updates and reads
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					switch j % 3 {
					case 0:
						// Update health status
						monitor.healthStatus.mu.Lock()
						monitor.healthStatus.TotalChecks++
						if j%10 == 0 {
							monitor.healthStatus.FailedChecks++
							monitor.healthStatus.IsHealthy = false
						} else {
							monitor.healthStatus.IsHealthy = true
						}
						monitor.healthStatus.LastCheck = time.Now()
						monitor.healthStatus.mu.Unlock()
					case 1:
						// Read health status
						_ = monitor.GetHealthStatus()
					case 2:
						// Update last error
						monitor.healthStatus.mu.Lock()
						if j%15 == 0 {
							monitor.healthStatus.LastError = "concurrent error"
						}
						monitor.healthStatus.mu.Unlock()
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify final state is consistent
		status := monitor.GetHealthStatus()
		assert.True(t, status.TotalChecks >= 0, "Total checks should be non-negative")
		assert.True(t, status.FailedChecks >= 0, "Failed checks should be non-negative")
		assert.True(t, status.FailedChecks <= status.TotalChecks, "Failed checks should not exceed total")
	})
}

func TestHealthMonitor_Metrics(t *testing.T) {
	monitor := NewHealthMonitor("metrics-test", 1*time.Second)
	defer monitor.Stop()

	t.Run("initial_metrics", func(t *testing.T) {
		metrics := monitor.GetMetrics()
		assert.NotNil(t, metrics, "Metrics should not be nil")
		assert.Equal(t, int64(0), metrics.TotalHealthChecks, "Initial total health checks should be 0")
		assert.Equal(t, int64(0), metrics.FailedHealthChecks, "Initial failed health checks should be 0")
		assert.Equal(t, float64(0), metrics.HealthCheckSuccessRate, "Initial success rate should be 0")
		assert.Equal(t, time.Duration(0), metrics.AverageResponseTime, "Initial average response time should be 0")
		assert.True(t, metrics.LastHealthCheck.IsZero(), "Initial last health check should be zero time")
	})

	t.Run("metrics_calculations", func(t *testing.T) {
		// Simulate health check metrics
		monitor.metrics.mu.Lock()
		monitor.metrics.TotalHealthChecks = 100
		monitor.metrics.FailedHealthChecks = 15
		monitor.metrics.LastHealthCheck = time.Now()
		
		// Simulate response times
		totalResponseTime := 50 * time.Millisecond * 100 // 50ms average for 100 checks
		monitor.metrics.TotalResponseTime = totalResponseTime
		monitor.metrics.mu.Unlock()

		metrics := monitor.GetMetrics()
		expectedSuccessRate := float64(85) / float64(100) * 100 // 85% success rate
		expectedAvgResponseTime := totalResponseTime / 100

		assert.Equal(t, int64(100), metrics.TotalHealthChecks, "Total health checks should match")
		assert.Equal(t, int64(15), metrics.FailedHealthChecks, "Failed health checks should match")
		assert.Equal(t, expectedSuccessRate, metrics.HealthCheckSuccessRate, "Success rate should be calculated correctly")
		assert.Equal(t, expectedAvgResponseTime, metrics.AverageResponseTime, "Average response time should be calculated correctly")
		assert.False(t, metrics.LastHealthCheck.IsZero(), "Last health check should be set")
	})

	t.Run("metrics_thread_safety", func(t *testing.T) {
		const numGoroutines = 15
		const operationsPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Simulate concurrent metric updates
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					monitor.metrics.mu.Lock()
					monitor.metrics.TotalHealthChecks++
					monitor.metrics.TotalResponseTime += time.Millisecond * time.Duration(j%100)
					if j%20 == 0 {
						monitor.metrics.FailedHealthChecks++
					}
					monitor.metrics.LastHealthCheck = time.Now()
					monitor.metrics.mu.Unlock()
				}
			}()
		}

		wg.Wait()

		metrics := monitor.GetMetrics()
		expectedTotal := int64(numGoroutines * operationsPerGoroutine)
		expectedFailed := int64(numGoroutines * (operationsPerGoroutine / 20))

		assert.Equal(t, expectedTotal, metrics.TotalHealthChecks, "Total health checks should match expected")
		assert.Equal(t, expectedFailed, metrics.FailedHealthChecks, "Failed health checks should match expected")
		assert.True(t, metrics.HealthCheckSuccessRate >= 0 && metrics.HealthCheckSuccessRate <= 100, 
			"Success rate should be between 0 and 100")
		assert.True(t, metrics.AverageResponseTime >= 0, "Average response time should be non-negative")
	})
}

func TestHealthMonitor_Lifecycle(t *testing.T) {
	t.Run("start_and_stop", func(t *testing.T) {
		monitor := NewHealthMonitor("lifecycle-test", 100*time.Millisecond)

		// Test starting
		err := monitor.Start()
		assert.NoError(t, err, "Start should succeed")

		// Verify context is active
		select {
		case <-monitor.ctx.Done():
			t.Fatal("Context should not be cancelled after start")
		default:
			// Context is active, which is expected
		}

		// Wait a bit to allow some health checks
		time.Sleep(250 * time.Millisecond)

		// Check that health checks are running
		status := monitor.GetHealthStatus()
		assert.True(t, status.TotalChecks > 0, "Health checks should have run")

		// Test stopping
		monitor.Stop()

		// Verify context is cancelled
		select {
		case <-monitor.ctx.Done():
			// Context is cancelled, which is expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Context should be cancelled after stop")
		}
	})

	t.Run("multiple_start_calls", func(t *testing.T) {
		monitor := NewHealthMonitor("multi-start-test", 1*time.Second)
		defer monitor.Stop()

		// Multiple start calls should be handled gracefully
		err1 := monitor.Start()
		err2 := monitor.Start()
		err3 := monitor.Start()

		assert.NoError(t, err1, "First start should succeed")
		assert.NoError(t, err2, "Second start should not error")
		assert.NoError(t, err3, "Third start should not error")
	})

	t.Run("multiple_stop_calls", func(t *testing.T) {
		monitor := NewHealthMonitor("multi-stop-test", 1*time.Second)

		err := monitor.Start()
		assert.NoError(t, err, "Start should succeed")

		// Multiple stop calls should be handled gracefully
		monitor.Stop()
		monitor.Stop()
		monitor.Stop()

		// Verify context is cancelled
		select {
		case <-monitor.ctx.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Context should be cancelled after stop")
		}
	})
}
