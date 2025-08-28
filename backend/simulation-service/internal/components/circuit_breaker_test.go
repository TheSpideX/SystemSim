package components

import (
	"fmt"
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestCircuitBreaker_BasicOperation(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:         100 * time.Millisecond,
		MaxRetries:      3,
		RetryDelay:      10 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test-component", config)

	// Test initial state
	if cb.GetState() != CircuitBreakerClosed {
		t.Errorf("Expected initial state to be CLOSED, got %s", cb.GetState())
	}

	// Test successful execution
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected successful execution, got error: %v", err)
	}

	// Test that circuit remains closed after success
	if cb.GetState() != CircuitBreakerClosed {
		t.Errorf("Expected state to remain CLOSED after success, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_FailureThreshold(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:         100 * time.Millisecond,
		MaxRetries:      3,
		RetryDelay:      10 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test-component", config)

	// Cause failures to reach threshold
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			return fmt.Errorf("simulated failure %d", i+1)
		})
		if err == nil {
			t.Errorf("Expected failure %d to return error", i+1)
		}
	}

	// Circuit should now be open
	if cb.GetState() != CircuitBreakerOpen {
		t.Errorf("Expected state to be OPEN after %d failures, got %s", config.FailureThreshold, cb.GetState())
	}

	// Test that execution is blocked when circuit is open
	if cb.CanExecute() {
		t.Error("Expected CanExecute to return false when circuit is OPEN")
	}

	err := cb.Execute(func() error {
		return nil
	})
	if err == nil {
		t.Error("Expected execution to fail when circuit is OPEN")
	}
}

func TestCircuitBreaker_TimeoutAndRecovery(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:         50 * time.Millisecond,
		MaxRetries:      3,
		RetryDelay:      10 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test-component", config)

	// Cause failures to open circuit
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("failure %d", i+1)
		})
	}

	// Verify circuit is open
	if cb.GetState() != CircuitBreakerOpen {
		t.Errorf("Expected state to be OPEN, got %s", cb.GetState())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Circuit should allow execution (transitioning to half-open)
	if !cb.CanExecute() {
		t.Error("Expected CanExecute to return true after timeout")
	}

	// Execute successfully to transition to half-open
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected successful execution after timeout, got error: %v", err)
	}

	// Should be in half-open state
	if cb.GetState() != CircuitBreakerHalfOpen {
		t.Errorf("Expected state to be HALF_OPEN after successful execution, got %s", cb.GetState())
	}

	// Another success should close the circuit
	err = cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected second successful execution, got error: %v", err)
	}

	// Should be closed now
	if cb.GetState() != CircuitBreakerClosed {
		t.Errorf("Expected state to be CLOSED after %d successes, got %s", config.SuccessThreshold, cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:         50 * time.Millisecond,
		MaxRetries:      3,
		RetryDelay:      10 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test-component", config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("failure %d", i+1)
		})
	}

	// Wait for timeout and transition to half-open
	time.Sleep(60 * time.Millisecond)
	cb.Execute(func() error {
		return nil
	})

	// Verify half-open state
	if cb.GetState() != CircuitBreakerHalfOpen {
		t.Errorf("Expected state to be HALF_OPEN, got %s", cb.GetState())
	}

	// Cause a failure in half-open state
	err := cb.Execute(func() error {
		return fmt.Errorf("failure in half-open")
	})
	if err == nil {
		t.Error("Expected failure to return error")
	}

	// Should be open again
	if cb.GetState() != CircuitBreakerOpen {
		t.Errorf("Expected state to be OPEN after failure in HALF_OPEN, got %s", cb.GetState())
	}
}

func TestCircuitBreakerManager(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:         50 * time.Millisecond,
		MaxRetries:      3,
		RetryDelay:      10 * time.Millisecond,
	}

	manager := NewCircuitBreakerManager(config)

	// Test getting circuit breakers for different components
	cb1 := manager.GetCircuitBreaker("component-1")
	cb2 := manager.GetCircuitBreaker("component-2")

	if cb1 == cb2 {
		t.Error("Expected different circuit breakers for different components")
	}

	// Test that getting the same component returns the same circuit breaker
	cb1Again := manager.GetCircuitBreaker("component-1")
	if cb1 != cb1Again {
		t.Error("Expected same circuit breaker for same component")
	}

	// Test ExecuteWithCircuitBreaker
	err := manager.ExecuteWithCircuitBreaker("component-1", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected successful execution, got error: %v", err)
	}

	// Test failure
	err = manager.ExecuteWithCircuitBreaker("component-1", func() error {
		return fmt.Errorf("test failure")
	})
	if err == nil {
		t.Error("Expected failure to return error")
	}

	// Test stats
	stats := manager.GetAllStats()
	if len(stats) != 2 {
		t.Errorf("Expected stats for 2 components, got %d", len(stats))
	}

	if _, exists := stats["component-1"]; !exists {
		t.Error("Expected stats for component-1")
	}

	if _, exists := stats["component-2"]; !exists {
		t.Error("Expected stats for component-2")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:         50 * time.Millisecond,
		MaxRetries:      3,
		RetryDelay:      10 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test-component", config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("failure %d", i+1)
		})
	}

	// Verify circuit is open
	if cb.GetState() != CircuitBreakerOpen {
		t.Errorf("Expected state to be OPEN, got %s", cb.GetState())
	}

	// Reset the circuit breaker
	cb.Reset()

	// Should be closed now
	if cb.GetState() != CircuitBreakerClosed {
		t.Errorf("Expected state to be CLOSED after reset, got %s", cb.GetState())
	}

	// Should allow execution
	if !cb.CanExecute() {
		t.Error("Expected CanExecute to return true after reset")
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:         100 * time.Millisecond,
		MaxRetries:      3,
		RetryDelay:      10 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test-component", config)

	// Execute some operations
	cb.Execute(func() error { return nil })
	cb.Execute(func() error { return fmt.Errorf("failure") })
	cb.Execute(func() error { return nil })

	stats := cb.GetStats()

	if stats["state"] != "CLOSED" {
		t.Errorf("Expected state to be CLOSED, got %s", stats["state"])
	}

	if stats["target_component"] != "test-component" {
		t.Errorf("Expected target_component to be test-component, got %s", stats["target_component"])
	}

	if stats["failure_count"] != 0 { // Should be reset after success
		t.Errorf("Expected failure_count to be 0, got %v", stats["failure_count"])
	}
}

func TestCentralizedOutputManager_CircuitBreakerIntegration(t *testing.T) {
	// Create a mock global registry
	registry := NewGlobalRegistry()

	// Create target component channel
	targetChannel := make(chan *engines.Operation, 1)
	registry.Register("target-component", targetChannel)

	// Create centralized output manager with circuit breaker
	com := &CentralizedOutputManager{
		InstanceID:            "test-instance",
		ComponentID:           "test-component",
		GlobalRegistry:        registry,
		CircuitBreakerManager: NewCircuitBreakerManager(DefaultCircuitBreakerConfig()),
	}

	// Create a test operation result
	result := &engines.OperationResult{
		OperationID:   "test-op-1",
		OperationType: "test_operation",
		Success:       true,
	}

	// Test successful routing through circuit breaker
	err := com.CircuitBreakerManager.ExecuteWithCircuitBreaker("target-component", func() error {
		return com.attemptRouting("target-component", result)
	})
	if err != nil {
		t.Errorf("Expected successful routing, got error: %v", err)
	}

	// Verify operation was routed
	select {
	case op := <-targetChannel:
		if op.ID != "test-op-1-next" {
			t.Errorf("Expected operation ID 'test-op-1-next', got %s", op.ID)
		}
	default:
		t.Error("Expected operation to be routed to target channel")
	}

	// Test circuit breaker with full channel (should fail and open circuit)
	// Fill the channel to capacity
	targetChannel <- &engines.Operation{ID: "filler"}

	// This should fail due to full channel and trigger circuit breaker
	for i := 0; i < 5; i++ { // Trigger multiple failures to open circuit
		err = com.CircuitBreakerManager.ExecuteWithCircuitBreaker("target-component", func() error {
			return com.attemptRouting("target-component", result)
		})
		if err == nil {
			t.Errorf("Expected routing failure %d to return error", i+1)
		}
	}

	// Get circuit breaker stats
	stats := com.CircuitBreakerManager.GetAllStats()
	if len(stats) == 0 {
		t.Error("Expected circuit breaker stats to be available")
	}

	// Verify circuit breaker was created for target component
	if _, exists := stats["target-component"]; !exists {
		t.Error("Expected circuit breaker stats for target-component")
	}
}
