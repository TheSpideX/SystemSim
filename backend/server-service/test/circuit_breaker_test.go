package test

import (
	"errors"
	"testing"
	"time"

	"server-service/internal/circuit"
)

// TestCircuitBreakerBasicFunctionality tests basic circuit breaker functionality
func TestCircuitBreakerBasicFunctionality(t *testing.T) {
	config := circuit.Config{
		MaxRequests: 3,
		Interval:    time.Second,
		Timeout:     time.Second,
		ReadyToTrip: func(counts circuit.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	}

	cb := circuit.NewCircuitBreaker("test", config)

	// Test initial state
	if cb.State() != circuit.StateClosed {
		t.Errorf("Expected initial state to be CLOSED, got %s", cb.State())
	}

	// Test successful requests
	for i := 0; i < 5; i++ {
		result, err := cb.Execute(func() (interface{}, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != "success" {
			t.Errorf("Expected result 'success', got %v", result)
		}
	}

	// Circuit should still be closed
	if cb.State() != circuit.StateClosed {
		t.Errorf("Expected state to be CLOSED after successful requests, got %s", cb.State())
	}
}

// TestCircuitBreakerFailureHandling tests failure handling
func TestCircuitBreakerFailureHandling(t *testing.T) {
	config := circuit.Config{
		MaxRequests: 1,
		Interval:    time.Second,
		Timeout:     100 * time.Millisecond,
		ReadyToTrip: func(counts circuit.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	}

	cb := circuit.NewCircuitBreaker("test", config)

	// Generate failures to trip the circuit
	for i := 0; i < 2; i++ {
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, errors.New("test error")
		})

		if err == nil {
			t.Error("Expected error from failing request")
		}
	}

	// Circuit should now be open
	if cb.State() != circuit.StateOpen {
		t.Errorf("Expected state to be OPEN after failures, got %s", cb.State())
	}

	// Requests should fail fast
	_, err := cb.Execute(func() (interface{}, error) {
		return "should not execute", nil
	})

	if err != circuit.ErrOpenState {
		t.Errorf("Expected ErrOpenState, got %v", err)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit should now be half-open
	if cb.State() != circuit.StateHalfOpen {
		t.Errorf("Expected state to be HALF_OPEN after timeout, got %s", cb.State())
	}

	// Successful request should close the circuit
	_, err = cb.Execute(func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected no error in half-open state, got %v", err)
	}

	// Circuit should be closed again
	if cb.State() != circuit.StateClosed {
		t.Errorf("Expected state to be CLOSED after successful request in half-open, got %s", cb.State())
	}
}

// TestCircuitBreakerCounts tests request counting
func TestCircuitBreakerCounts(t *testing.T) {
	config := circuit.DefaultConfig()
	cb := circuit.NewCircuitBreaker("test", config)

	// Execute some successful requests
	for i := 0; i < 3; i++ {
		cb.Execute(func() (interface{}, error) {
			return "success", nil
		})
	}

	// Execute some failed requests
	for i := 0; i < 2; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("test error")
		})
	}

	counts := cb.Counts()

	if counts.Requests != 5 {
		t.Errorf("Expected 5 total requests, got %d", counts.Requests)
	}

	if counts.TotalSuccesses != 3 {
		t.Errorf("Expected 3 total successes, got %d", counts.TotalSuccesses)
	}

	if counts.TotalFailures != 2 {
		t.Errorf("Expected 2 total failures, got %d", counts.TotalFailures)
	}

	if counts.ConsecutiveFailures != 2 {
		t.Errorf("Expected 2 consecutive failures, got %d", counts.ConsecutiveFailures)
	}
}

// TestCircuitBreakerManager tests the circuit breaker manager
func TestCircuitBreakerManager(t *testing.T) {
	manager := circuit.NewManager()

	// Get breakers for different services
	authBreaker := manager.GetBreaker("auth")
	projectBreaker := manager.GetBreaker("project")
	simulationBreaker := manager.GetBreaker("simulation")

	if authBreaker == nil {
		t.Error("Expected auth breaker to be created")
	}

	if projectBreaker == nil {
		t.Error("Expected project breaker to be created")
	}

	if simulationBreaker == nil {
		t.Error("Expected simulation breaker to be created")
	}

	// Test that getting the same service returns the same breaker
	authBreaker2 := manager.GetBreaker("auth")
	if authBreaker != authBreaker2 {
		t.Error("Expected same breaker instance for same service")
	}

	// Test stats
	stats := manager.GetStats()
	if len(stats) != 3 {
		t.Errorf("Expected 3 breakers in stats, got %d", len(stats))
	}

	expectedServices := []string{"auth", "project", "simulation"}
	for _, service := range expectedServices {
		if _, exists := stats[service]; !exists {
			t.Errorf("Expected stats for service %s", service)
		}
	}
}

// TestCircuitBreakerStateTransitions tests state transitions
func TestCircuitBreakerStateTransitions(t *testing.T) {
	stateChanges := make([]string, 0)

	config := circuit.Config{
		MaxRequests: 1,
		Interval:    0,
		Timeout:     50 * time.Millisecond,
		ReadyToTrip: func(counts circuit.Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
		OnStateChange: func(name string, from circuit.State, to circuit.State) {
			stateChanges = append(stateChanges, from.String()+"->"+to.String())
		},
	}

	cb := circuit.NewCircuitBreaker("test", config)

	// Trigger failure to open circuit
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	// Wait for timeout to go to half-open
	time.Sleep(60 * time.Millisecond)

	// Access state to trigger transition
	cb.State()

	// Successful request to close circuit
	cb.Execute(func() (interface{}, error) {
		return "success", nil
	})

	expectedTransitions := []string{"CLOSED->OPEN", "OPEN->HALF_OPEN", "HALF_OPEN->CLOSED"}
	if len(stateChanges) != len(expectedTransitions) {
		t.Errorf("Expected %d state changes, got %d: %v", len(expectedTransitions), len(stateChanges), stateChanges)
	}

	for i, expected := range expectedTransitions {
		if i < len(stateChanges) && stateChanges[i] != expected {
			t.Errorf("Expected state change %s, got %s", expected, stateChanges[i])
		}
	}
}

// TestCircuitBreakerHalfOpenMaxRequests tests half-open state request limiting
func TestCircuitBreakerHalfOpenMaxRequests(t *testing.T) {
	config := circuit.Config{
		MaxRequests: 2,
		Interval:    0,
		Timeout:     50 * time.Millisecond,
		ReadyToTrip: func(counts circuit.Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	}

	cb := circuit.NewCircuitBreaker("test", config)

	// Trip the circuit
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// First two requests should be allowed in half-open state
	for i := 0; i < 2; i++ {
		_, err := cb.Execute(func() (interface{}, error) {
			return "success", nil
		})
		if err != nil {
			t.Errorf("Expected request %d to succeed in half-open state, got error: %v", i+1, err)
		}
	}

	// Third request should be rejected
	_, err := cb.Execute(func() (interface{}, error) {
		return "should not execute", nil
	})

	if err != circuit.ErrTooManyRequests {
		t.Errorf("Expected ErrTooManyRequests for third request in half-open state, got %v", err)
	}
}
