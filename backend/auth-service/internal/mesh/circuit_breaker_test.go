package mesh

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_Creation(t *testing.T) {
	tests := []struct {
		name               string
		serviceName        string
		failureThreshold   int
		recoveryTimeout    time.Duration
		requestTimeout     time.Duration
		expectError        bool
		errorMsg           string
	}{
		{
			name:             "valid_circuit_breaker",
			serviceName:      "test-service",
			failureThreshold: 5,
			recoveryTimeout:  30 * time.Second,
			requestTimeout:   5 * time.Second,
			expectError:      false,
		},
		{
			name:             "zero_failure_threshold",
			serviceName:      "zero-threshold-service",
			failureThreshold: 0,
			recoveryTimeout:  30 * time.Second,
			requestTimeout:   5 * time.Second,
			expectError:      false, // Should use default threshold
		},
		{
			name:             "negative_failure_threshold",
			serviceName:      "negative-threshold-service",
			failureThreshold: -1,
			recoveryTimeout:  30 * time.Second,
			requestTimeout:   5 * time.Second,
			expectError:      false, // Should use default threshold
		},
		{
			name:             "zero_recovery_timeout",
			serviceName:      "zero-recovery-service",
			failureThreshold: 5,
			recoveryTimeout:  0,
			requestTimeout:   5 * time.Second,
			expectError:      false, // Should use default timeout
		},
		{
			name:             "zero_request_timeout",
			serviceName:      "zero-request-service",
			failureThreshold: 5,
			recoveryTimeout:  30 * time.Second,
			requestTimeout:   0,
			requestTimeout:   5 * time.Second,
			expectError:      false, // Should use default timeout
		},
		{
			name:             "high_failure_threshold",
			serviceName:      "high-threshold-service",
			failureThreshold: 1000,
			recoveryTimeout:  30 * time.Second,
			requestTimeout:   5 * time.Second,
			expectError:      false,
		},
		{
			name:             "short_timeouts",
			serviceName:      "short-timeout-service",
			failureThreshold: 3,
			recoveryTimeout:  1 * time.Second,
			requestTimeout:   100 * time.Millisecond,
			expectError:      false,
		},
		{
			name:             "empty_service_name",
			serviceName:      "",
			failureThreshold: 5,
			recoveryTimeout:  30 * time.Second,
			requestTimeout:   5 * time.Second,
			expectError:      false, // Should be allowed for testing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewCircuitBreaker(tt.serviceName, tt.failureThreshold, tt.recoveryTimeout, tt.requestTimeout)

			if tt.expectError {
				assert.Nil(t, cb, "Circuit breaker should be nil for invalid parameters")
			} else {
				assert.NotNil(t, cb, "Circuit breaker should not be nil")
				assert.Equal(t, tt.serviceName, cb.serviceName)
				
				expectedThreshold := tt.failureThreshold
				if expectedThreshold <= 0 {
					expectedThreshold = 5 // Default threshold
				}
				assert.Equal(t, expectedThreshold, cb.failureThreshold)
				
				expectedRecoveryTimeout := tt.recoveryTimeout
				if expectedRecoveryTimeout <= 0 {
					expectedRecoveryTimeout = 30 * time.Second // Default timeout
				}
				assert.Equal(t, expectedRecoveryTimeout, cb.recoveryTimeout)
				
				expectedRequestTimeout := tt.requestTimeout
				if expectedRequestTimeout <= 0 {
					expectedRequestTimeout = 5 * time.Second // Default timeout
				}
				assert.Equal(t, expectedRequestTimeout, cb.requestTimeout)
				
				assert.Equal(t, StateClosed, cb.state, "Initial state should be closed")
				assert.Equal(t, 0, cb.failureCount, "Initial failure count should be 0")
				assert.True(t, cb.lastFailureTime.IsZero(), "Initial last failure time should be zero")
				assert.NotNil(t, cb.metrics, "Metrics should be initialized")
			}
		})
	}
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	cb := NewCircuitBreaker("state-test", 3, 1*time.Second, 100*time.Millisecond)

	t.Run("closed_to_open_transition", func(t *testing.T) {
		// Reset circuit breaker
		cb.mu.Lock()
		cb.state = StateClosed
		cb.failureCount = 0
		cb.mu.Unlock()

		// Simulate failures to trigger state change
		for i := 0; i < 3; i++ {
			err := cb.Execute(func() error {
				return errors.New("simulated failure")
			})
			assert.Error(t, err, "Execute should return error for failure")
		}

		// Check state transition
		cb.mu.RLock()
		state := cb.state
		failureCount := cb.failureCount
		cb.mu.RUnlock()

		assert.Equal(t, StateOpen, state, "State should transition to open after threshold failures")
		assert.Equal(t, 3, failureCount, "Failure count should match threshold")
	})

	t.Run("open_to_half_open_transition", func(t *testing.T) {
		// Set circuit breaker to open state
		cb.mu.Lock()
		cb.state = StateOpen
		cb.failureCount = 3
		cb.lastFailureTime = time.Now().Add(-2 * time.Second) // Past recovery timeout
		cb.mu.Unlock()

		// Try to execute - should transition to half-open
		err := cb.Execute(func() error {
			return nil // Success
		})
		assert.NoError(t, err, "Execute should succeed in half-open state")

		// Check state transition
		cb.mu.RLock()
		state := cb.state
		cb.mu.RUnlock()

		assert.Equal(t, StateClosed, state, "State should transition to closed after successful execution in half-open")
	})

	t.Run("half_open_to_open_on_failure", func(t *testing.T) {
		// Set circuit breaker to half-open state
		cb.mu.Lock()
		cb.state = StateHalfOpen
		cb.failureCount = 2
		cb.mu.Unlock()

		// Execute with failure
		err := cb.Execute(func() error {
			return errors.New("failure in half-open")
		})
		assert.Error(t, err, "Execute should return error for failure")

		// Check state transition
		cb.mu.RLock()
		state := cb.state
		cb.mu.RUnlock()

		assert.Equal(t, StateOpen, state, "State should transition back to open on failure in half-open")
	})

	t.Run("half_open_to_closed_on_success", func(t *testing.T) {
		// Set circuit breaker to half-open state
		cb.mu.Lock()
		cb.state = StateHalfOpen
		cb.failureCount = 2
		cb.mu.Unlock()

		// Execute with success
		err := cb.Execute(func() error {
			return nil // Success
		})
		assert.NoError(t, err, "Execute should succeed")

		// Check state transition
		cb.mu.RLock()
		state := cb.state
		failureCount := cb.failureCount
		cb.mu.RUnlock()

		assert.Equal(t, StateClosed, state, "State should transition to closed on success in half-open")
		assert.Equal(t, 0, failureCount, "Failure count should be reset")
	})
}

func TestCircuitBreaker_Execute(t *testing.T) {
	cb := NewCircuitBreaker("execute-test", 2, 1*time.Second, 100*time.Millisecond)

	t.Run("successful_execution", func(t *testing.T) {
		// Reset circuit breaker
		cb.mu.Lock()
		cb.state = StateClosed
		cb.failureCount = 0
		cb.mu.Unlock()

		result := "success"
		err := cb.Execute(func() error {
			return nil
		})

		assert.NoError(t, err, "Execute should succeed for successful function")
		
		// Check metrics
		metrics := cb.GetMetrics()
		assert.Equal(t, int64(1), metrics.TotalRequests, "Total requests should be incremented")
		assert.Equal(t, int64(1), metrics.SuccessfulRequests, "Successful requests should be incremented")
		assert.Equal(t, int64(0), metrics.FailedRequests, "Failed requests should remain 0")
	})

	t.Run("failed_execution", func(t *testing.T) {
		// Reset circuit breaker
		cb.mu.Lock()
		cb.state = StateClosed
		cb.failureCount = 0
		cb.mu.Unlock()

		expectedError := errors.New("test failure")
		err := cb.Execute(func() error {
			return expectedError
		})

		assert.Error(t, err, "Execute should return error for failed function")
		assert.Equal(t, expectedError, err, "Error should match the function error")
		
		// Check failure count
		cb.mu.RLock()
		failureCount := cb.failureCount
		cb.mu.RUnlock()
		
		assert.Equal(t, 1, failureCount, "Failure count should be incremented")
	})

	t.Run("execution_blocked_when_open", func(t *testing.T) {
		// Set circuit breaker to open state
		cb.mu.Lock()
		cb.state = StateOpen
		cb.lastFailureTime = time.Now()
		cb.mu.Unlock()

		err := cb.Execute(func() error {
			t.Fatal("Function should not be executed when circuit is open")
			return nil
		})

		assert.Error(t, err, "Execute should return error when circuit is open")
		assert.Contains(t, err.Error(), "circuit breaker is open", "Error should indicate circuit is open")
	})

	t.Run("timeout_handling", func(t *testing.T) {
		cb := NewCircuitBreaker("timeout-test", 5, 1*time.Second, 50*time.Millisecond)
		
		// Reset circuit breaker
		cb.mu.Lock()
		cb.state = StateClosed
		cb.failureCount = 0
		cb.mu.Unlock()

		err := cb.Execute(func() error {
			time.Sleep(100 * time.Millisecond) // Longer than timeout
			return nil
		})

		assert.Error(t, err, "Execute should timeout for slow function")
		assert.Contains(t, err.Error(), "timeout", "Error should indicate timeout")
		
		// Check that timeout is treated as failure
		cb.mu.RLock()
		failureCount := cb.failureCount
		cb.mu.RUnlock()
		
		assert.Equal(t, 1, failureCount, "Timeout should increment failure count")
	})
}

func TestCircuitBreaker_Metrics(t *testing.T) {
	cb := NewCircuitBreaker("metrics-test", 3, 1*time.Second, 100*time.Millisecond)

	t.Run("initial_metrics", func(t *testing.T) {
		metrics := cb.GetMetrics()
		assert.NotNil(t, metrics, "Metrics should not be nil")
		assert.Equal(t, int64(0), metrics.TotalRequests, "Initial total requests should be 0")
		assert.Equal(t, int64(0), metrics.SuccessfulRequests, "Initial successful requests should be 0")
		assert.Equal(t, int64(0), metrics.FailedRequests, "Initial failed requests should be 0")
		assert.Equal(t, int64(0), metrics.RejectedRequests, "Initial rejected requests should be 0")
		assert.Equal(t, StateClosed, metrics.CurrentState, "Initial state should be closed")
		assert.Equal(t, 0, metrics.FailureCount, "Initial failure count should be 0")
		assert.True(t, metrics.LastFailureTime.IsZero(), "Initial last failure time should be zero")
	})

	t.Run("metrics_updates", func(t *testing.T) {
		// Execute successful requests
		for i := 0; i < 5; i++ {
			err := cb.Execute(func() error {
				return nil
			})
			assert.NoError(t, err)
		}

		// Execute failed requests
		for i := 0; i < 2; i++ {
			err := cb.Execute(func() error {
				return errors.New("test failure")
			})
			assert.Error(t, err)
		}

		metrics := cb.GetMetrics()
		assert.Equal(t, int64(7), metrics.TotalRequests, "Total requests should be 7")
		assert.Equal(t, int64(5), metrics.SuccessfulRequests, "Successful requests should be 5")
		assert.Equal(t, int64(2), metrics.FailedRequests, "Failed requests should be 2")
		assert.Equal(t, 2, metrics.FailureCount, "Failure count should be 2")
	})
}
