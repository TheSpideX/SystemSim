package components

import (
	"testing"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestFallbackRouting_BasicFallback(t *testing.T) {
	// Create a mock global registry
	registry := NewMockGlobalRegistry()
	
	// Register components with different health states
	registry.RegisterComponent("primary-component", make(chan *engines.Operation, 10))
	registry.RegisterComponent("fallback-1", make(chan *engines.Operation, 10))
	registry.RegisterComponent("fallback-2", make(chan *engines.Operation, 10))
	
	// Set primary component as unhealthy
	registry.SetHealth("primary-component", 0.3) // Unhealthy
	registry.SetHealth("fallback-1", 0.9)        // Healthy
	registry.SetHealth("fallback-2", 0.8)        // Healthy
	
	// Create centralized output manager with fallback routing
	com := &CentralizedOutputManager{
		InstanceID:      "test-instance",
		ComponentID:     "test-component",
		GlobalRegistry:  registry,
		InputChannel:    make(chan *engines.OperationResult, 10),
		OutputChannel:   make(chan *engines.OperationResult, 10),
		FallbackRouting: &FallbackRoutingConfig{
			Enabled: true,
			FallbackTargets: map[string][]string{
				"primary-component": {"fallback-1", "fallback-2"},
			},
			MaxFallbackAttempts: 2,
			FallbackStrategy:    FallbackStrategySequential,
		},
	}

	// Create a test operation result
	result := &engines.OperationResult{
		OperationID:   "test-op-1",
		OperationType: "compute",
		Success:       true,
	}

	// Attempt fallback routing
	err := com.attemptFallbackRouting("primary-component", result, "unhealthy")
	if err != nil {
		t.Fatalf("Expected fallback routing to succeed, got error: %v", err)
	}

	// Verify that the operation was routed to fallback-1
	fallback1Channel := registry.GetChannel("fallback-1")
	select {
	case op := <-fallback1Channel:
		if op.ID != "test-op-1-next" {
			t.Errorf("Expected operation ID 'test-op-1-next', got %s", op.ID)
		}
	default:
		t.Error("Expected operation to be routed to fallback-1")
	}
}

func TestFallbackRouting_HealthBasedStrategy(t *testing.T) {
	// Create a mock global registry
	registry := NewMockGlobalRegistry()
	
	// Register components with different health states
	registry.RegisterComponent("fallback-1", make(chan *engines.Operation, 10))
	registry.RegisterComponent("fallback-2", make(chan *engines.Operation, 10))
	registry.RegisterComponent("fallback-3", make(chan *engines.Operation, 10))
	
	// Set different health levels
	registry.SetHealth("fallback-1", 0.6) // Medium health
	registry.SetHealth("fallback-2", 0.9) // Best health
	registry.SetHealth("fallback-3", 0.7) // Good health
	
	// Create centralized output manager with health-based fallback
	com := &CentralizedOutputManager{
		InstanceID:      "test-instance",
		ComponentID:     "test-component",
		GlobalRegistry:  registry,
		InputChannel:    make(chan *engines.OperationResult, 10),
		OutputChannel:   make(chan *engines.OperationResult, 10),
		FallbackRouting: &FallbackRoutingConfig{
			Enabled: true,
			FallbackTargets: map[string][]string{
				"primary-component": {"fallback-1", "fallback-2", "fallback-3"},
			},
			FallbackStrategy: FallbackStrategyHealthBased,
		},
	}

	// Get fallback targets and verify they're sorted by health
	result := &engines.OperationResult{
		OperationID:   "test-op-1",
		OperationType: "compute",
		Success:       true,
	}

	targets := com.getFallbackTargets("primary-component", result)
	
	// Should be sorted by health: fallback-2 (0.9), fallback-3 (0.7), fallback-1 (0.6)
	expectedOrder := []string{"fallback-2", "fallback-3", "fallback-1"}
	if len(targets) != len(expectedOrder) {
		t.Fatalf("Expected %d targets, got %d", len(expectedOrder), len(targets))
	}

	for i, expected := range expectedOrder {
		if targets[i] != expected {
			t.Errorf("Expected target %d to be %s, got %s", i, expected, targets[i])
		}
	}
}

func TestFallbackRouting_LoadBasedStrategy(t *testing.T) {
	// Create a mock global registry
	registry := NewMockGlobalRegistry()
	
	// Register components with different load states
	registry.RegisterComponent("fallback-1", make(chan *engines.Operation, 10))
	registry.RegisterComponent("fallback-2", make(chan *engines.Operation, 10))
	registry.RegisterComponent("fallback-3", make(chan *engines.Operation, 10))
	
	// Set different load levels
	registry.SetLoad("fallback-1", BufferStatusHigh)     // High load
	registry.SetLoad("fallback-2", BufferStatusNormal)   // Normal load (best)
	registry.SetLoad("fallback-3", BufferStatusWarning)  // Warning load
	
	// Create centralized output manager with load-based fallback
	com := &CentralizedOutputManager{
		InstanceID:      "test-instance",
		ComponentID:     "test-component",
		GlobalRegistry:  registry,
		InputChannel:    make(chan *engines.OperationResult, 10),
		OutputChannel:   make(chan *engines.OperationResult, 10),
		FallbackRouting: &FallbackRoutingConfig{
			Enabled: true,
			FallbackTargets: map[string][]string{
				"primary-component": {"fallback-1", "fallback-2", "fallback-3"},
			},
			FallbackStrategy: FallbackStrategyLoadBased,
		},
	}

	// Get fallback targets and verify they're sorted by load
	result := &engines.OperationResult{
		OperationID:   "test-op-1",
		OperationType: "compute",
		Success:       true,
	}

	targets := com.getFallbackTargets("primary-component", result)
	
	// Should be sorted by load: fallback-2 (Normal), fallback-3 (Warning), fallback-1 (High)
	expectedOrder := []string{"fallback-2", "fallback-3", "fallback-1"}
	if len(targets) != len(expectedOrder) {
		t.Fatalf("Expected %d targets, got %d", len(expectedOrder), len(targets))
	}

	for i, expected := range expectedOrder {
		if targets[i] != expected {
			t.Errorf("Expected target %d to be %s, got %s", i, expected, targets[i])
		}
	}
}

func TestFallbackRouting_OperationTypeFallbacks(t *testing.T) {
	// Create a mock global registry
	registry := NewMockGlobalRegistry()
	
	// Register components
	registry.RegisterComponent("compute-fallback", make(chan *engines.Operation, 10))
	registry.RegisterComponent("memory-fallback", make(chan *engines.Operation, 10))
	
	// Create centralized output manager with operation type fallbacks
	com := &CentralizedOutputManager{
		InstanceID:      "test-instance",
		ComponentID:     "test-component",
		GlobalRegistry:  registry,
		InputChannel:    make(chan *engines.OperationResult, 10),
		OutputChannel:   make(chan *engines.OperationResult, 10),
		FallbackRouting: &FallbackRoutingConfig{
			Enabled: true,
			OperationTypeFallbacks: map[string][]string{
				"compute": {"compute-fallback"},
				"memory":  {"memory-fallback"},
			},
			FallbackStrategy: FallbackStrategySequential,
		},
	}

	// Test compute operation fallback
	computeResult := &engines.OperationResult{
		OperationID:   "test-compute-1",
		OperationType: "compute",
		Success:       true,
	}

	targets := com.getFallbackTargets("unknown-primary", computeResult)
	if len(targets) != 1 || targets[0] != "compute-fallback" {
		t.Errorf("Expected compute fallback target, got %v", targets)
	}

	// Test memory operation fallback
	memoryResult := &engines.OperationResult{
		OperationID:   "test-memory-1",
		OperationType: "memory",
		Success:       true,
	}

	targets = com.getFallbackTargets("unknown-primary", memoryResult)
	if len(targets) != 1 || targets[0] != "memory-fallback" {
		t.Errorf("Expected memory fallback target, got %v", targets)
	}
}

func TestFallbackRouting_PenaltyBasedFallbacks(t *testing.T) {
	// Create a mock global registry
	registry := NewMockGlobalRegistry()
	
	// Register components
	registry.RegisterComponent("throttle-fallback", make(chan *engines.Operation, 10))
	registry.RegisterComponent("redirect-fallback", make(chan *engines.Operation, 10))
	
	// Create centralized output manager with penalty-based fallbacks
	com := &CentralizedOutputManager{
		InstanceID:      "test-instance",
		ComponentID:     "test-component",
		GlobalRegistry:  registry,
		InputChannel:    make(chan *engines.OperationResult, 10),
		OutputChannel:   make(chan *engines.OperationResult, 10),
		FallbackRouting: &FallbackRoutingConfig{
			Enabled: true,
			ConditionFallbacks: map[string][]string{
				"throttle": {"throttle-fallback"},
				"redirect": {"redirect-fallback"},
				"D":        {"redirect-fallback"}, // Performance grade D
			},
			FallbackStrategy: FallbackStrategySequential,
		},
	}

	// Test throttle recommendation fallback
	throttleResult := &engines.OperationResult{
		OperationID:   "test-throttle-1",
		OperationType: "compute",
		Success:       true,
		PenaltyInfo: &engines.PenaltyInformation{
			RecommendedAction: "throttle",
			PerformanceGrade:  "C",
		},
	}

	targets := com.getFallbackTargets("unknown-primary", throttleResult)
	if len(targets) != 1 || targets[0] != "throttle-fallback" {
		t.Errorf("Expected throttle fallback target, got %v", targets)
	}

	// Test redirect recommendation fallback
	redirectResult := &engines.OperationResult{
		OperationID:   "test-redirect-1",
		OperationType: "compute",
		Success:       true,
		PenaltyInfo: &engines.PenaltyInformation{
			RecommendedAction: "redirect",
			PerformanceGrade:  "D",
		},
	}

	targets = com.getFallbackTargets("unknown-primary", redirectResult)
	// Should get both redirect and D grade fallbacks (deduplicated)
	if len(targets) != 1 || targets[0] != "redirect-fallback" {
		t.Errorf("Expected redirect fallback target, got %v", targets)
	}
}

func TestFallbackRouting_MaxAttempts(t *testing.T) {
	// Create a mock global registry with all components failing
	registry := NewMockGlobalRegistry()
	
	// Register components but make their channels full to simulate failure
	registry.RegisterComponent("fallback-1", make(chan *engines.Operation, 0)) // Full channel
	registry.RegisterComponent("fallback-2", make(chan *engines.Operation, 0)) // Full channel
	registry.RegisterComponent("fallback-3", make(chan *engines.Operation, 0)) // Full channel
	
	// Create centralized output manager with limited attempts
	com := &CentralizedOutputManager{
		InstanceID:      "test-instance",
		ComponentID:     "test-component",
		GlobalRegistry:  registry,
		InputChannel:    make(chan *engines.OperationResult, 10),
		OutputChannel:   make(chan *engines.OperationResult, 10),
		FallbackRouting: &FallbackRoutingConfig{
			Enabled: true,
			FallbackTargets: map[string][]string{
				"primary-component": {"fallback-1", "fallback-2", "fallback-3"},
			},
			MaxFallbackAttempts: 2, // Only try first 2 fallbacks
			FallbackStrategy:    FallbackStrategySequential,
		},
	}

	// Create a test operation result
	result := &engines.OperationResult{
		OperationID:   "test-op-1",
		OperationType: "compute",
		Success:       true,
	}

	// Attempt fallback routing - should fail after 2 attempts
	err := com.attemptFallbackRouting("primary-component", result, "test")
	if err == nil {
		t.Error("Expected fallback routing to fail due to max attempts limit")
	}

	// Verify error message mentions all fallback targets failed
	expectedError := "all fallback targets failed for primary-component"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}
