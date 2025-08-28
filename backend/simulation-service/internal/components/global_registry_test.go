package components

import (
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestGlobalRegistry_BasicOperations(t *testing.T) {
	registry := NewGlobalRegistry()

	// Test registration
	channel1 := make(chan *engines.Operation, 10)
	registry.Register("component-1", channel1)

	// Test getting channel
	retrievedChannel := registry.GetChannel("component-1")
	if retrievedChannel != channel1 {
		t.Error("Retrieved channel does not match registered channel")
	}

	// Test health and load defaults
	health := registry.GetHealth("component-1")
	if health != 1.0 {
		t.Errorf("Expected initial health 1.0, got %f", health)
	}

	load := registry.GetLoad("component-1")
	if load != BufferStatusNormal {
		t.Errorf("Expected initial load BufferStatusNormal, got %v", load)
	}

	// Test unregistration
	registry.Unregister("component-1")
	retrievedChannel = registry.GetChannel("component-1")
	if retrievedChannel != nil {
		t.Error("Channel should be nil after unregistration")
	}
}

func TestGlobalRegistry_MultipleComponents(t *testing.T) {
	registry := NewGlobalRegistry()

	// Register multiple components
	channel1 := make(chan *engines.Operation, 10)
	channel2 := make(chan *engines.Operation, 20)
	channel3 := make(chan *engines.Operation, 30)

	registry.Register("component-1", channel1)
	registry.Register("component-2", channel2)
	registry.Register("component-3", channel3)

	// Test getting all components
	allComponents := registry.GetAllComponents()
	if len(allComponents) != 3 {
		t.Errorf("Expected 3 components, got %d", len(allComponents))
	}

	if allComponents["component-1"] != channel1 {
		t.Error("Component-1 channel mismatch")
	}

	if allComponents["component-2"] != channel2 {
		t.Error("Component-2 channel mismatch")
	}

	if allComponents["component-3"] != channel3 {
		t.Error("Component-3 channel mismatch")
	}

	// Test getting registered components list
	registeredComponents := registry.GetRegisteredComponents()
	if len(registeredComponents) != 3 {
		t.Errorf("Expected 3 registered components, got %d", len(registeredComponents))
	}

	// Verify all components are in the list
	componentMap := make(map[string]bool)
	for _, id := range registeredComponents {
		componentMap[id] = true
	}

	if !componentMap["component-1"] || !componentMap["component-2"] || !componentMap["component-3"] {
		t.Error("Not all components found in registered components list")
	}
}

func TestGlobalRegistry_HealthAndLoadUpdates(t *testing.T) {
	registry := NewGlobalRegistry()

	// Register a component
	channel := make(chan *engines.Operation, 10)
	registry.Register("test-component", channel)

	// Test health updates
	registry.UpdateHealth("test-component", 0.8)
	health := registry.GetHealth("test-component")
	if health != 0.8 {
		t.Errorf("Expected health 0.8, got %f", health)
	}

	// Test health clamping
	registry.UpdateHealth("test-component", 1.5) // Should be clamped to 1.0
	health = registry.GetHealth("test-component")
	if health != 1.0 {
		t.Errorf("Expected health 1.0 (clamped), got %f", health)
	}

	registry.UpdateHealth("test-component", -0.5) // Should be clamped to 0.0
	health = registry.GetHealth("test-component")
	if health != 0.0 {
		t.Errorf("Expected health 0.0 (clamped), got %f", health)
	}

	// Test load updates
	registry.UpdateLoad("test-component", BufferStatusCritical)
	load := registry.GetLoad("test-component")
	if load != BufferStatusCritical {
		t.Errorf("Expected load BufferStatusCritical, got %v", load)
	}
}

func TestGlobalRegistry_NonExistentComponent(t *testing.T) {
	registry := NewGlobalRegistry()

	// Test getting channel for non-existent component
	channel := registry.GetChannel("nonexistent")
	if channel != nil {
		t.Error("Expected nil channel for non-existent component")
	}

	// Test getting health for non-existent component
	health := registry.GetHealth("nonexistent")
	if health != 0.0 {
		t.Errorf("Expected health 0.0 for non-existent component, got %f", health)
	}

	// Test getting load for non-existent component
	load := registry.GetLoad("nonexistent")
	if load != BufferStatusEmergency {
		t.Errorf("Expected load BufferStatusEmergency for non-existent component, got %v", load)
	}

	// Test updating health for non-existent component (should not crash)
	registry.UpdateHealth("nonexistent", 0.5)

	// Test updating load for non-existent component (should not crash)
	registry.UpdateLoad("nonexistent", BufferStatusHigh)
}

func TestGlobalRegistry_StartStop(t *testing.T) {
	registry := NewGlobalRegistry()

	// Test starting
	err := registry.Start()
	if err != nil {
		t.Fatalf("Failed to start registry: %v", err)
	}

	// Test starting again (should fail)
	err = registry.Start()
	if err == nil {
		t.Error("Expected error when starting already running registry")
	}

	// Test stopping
	err = registry.Stop()
	if err != nil {
		t.Fatalf("Failed to stop registry: %v", err)
	}

	// Test stopping again (should fail)
	err = registry.Stop()
	if err == nil {
		t.Error("Expected error when stopping already stopped registry")
	}
}

func TestGlobalRegistry_HealthMonitoring(t *testing.T) {
	registry := NewGlobalRegistry()

	// Register a component with a small channel
	channel := make(chan *engines.Operation, 5)
	registry.Register("test-component", channel)

	// Start the registry to enable health monitoring
	err := registry.Start()
	if err != nil {
		t.Fatalf("Failed to start registry: %v", err)
	}
	defer registry.Stop()

	// Fill the channel partially
	for i := 0; i < 3; i++ {
		channel <- &engines.Operation{ID: "test-op"}
	}

	// Wait for health monitoring to run
	time.Sleep(time.Millisecond * 100)

	// Manually trigger health check
	registry.performHealthCheck()

	// Check that health was updated based on channel utilization
	health := registry.GetHealth("test-component")
	expectedHealth := 1.0 - (3.0 / 5.0) // 1.0 - 0.6 = 0.4
	if health != expectedHealth {
		t.Errorf("Expected health %f, got %f", expectedHealth, health)
	}

	// Check that load status was updated
	load := registry.GetLoad("test-component")
	// 3/5 = 0.6 utilization should be BufferStatusOverflow (0.6 is in 0.6-0.8 range)
	if load != BufferStatusOverflow {
		t.Errorf("Expected load BufferStatusOverflow, got %v", load)
	}
}

func TestGlobalRegistry_ComponentStats(t *testing.T) {
	registry := NewGlobalRegistry()

	// Register multiple components
	channel1 := make(chan *engines.Operation, 10)
	channel2 := make(chan *engines.Operation, 20)

	registry.Register("component-1", channel1)
	registry.Register("component-2", channel2)

	// Update health and load
	registry.UpdateHealth("component-1", 0.9)
	registry.UpdateLoad("component-1", BufferStatusWarning)

	registry.UpdateHealth("component-2", 0.7)
	registry.UpdateLoad("component-2", BufferStatusHigh)

	// Get component stats
	stats := registry.GetComponentStats()

	if len(stats) != 2 {
		t.Errorf("Expected 2 component stats, got %d", len(stats))
	}

	// Check component-1 stats
	if stats["component-1"].Health != 0.9 {
		t.Errorf("Expected component-1 health 0.9, got %f", stats["component-1"].Health)
	}

	if stats["component-1"].Load != BufferStatusWarning {
		t.Errorf("Expected component-1 load BufferStatusWarning, got %v", stats["component-1"].Load)
	}

	if !stats["component-1"].Registered {
		t.Error("Expected component-1 to be registered")
	}

	// Check component-2 stats
	if stats["component-2"].Health != 0.7 {
		t.Errorf("Expected component-2 health 0.7, got %f", stats["component-2"].Health)
	}

	if stats["component-2"].Load != BufferStatusHigh {
		t.Errorf("Expected component-2 load BufferStatusHigh, got %v", stats["component-2"].Load)
	}

	if !stats["component-2"].Registered {
		t.Error("Expected component-2 to be registered")
	}
}

func TestGlobalRegistry_InvalidRegistration(t *testing.T) {
	registry := NewGlobalRegistry()

	// Test registering with empty component ID
	channel := make(chan *engines.Operation, 10)
	registry.Register("", channel) // Should not crash

	// Test registering with nil channel
	registry.Register("test-component", nil) // Should not crash

	// Verify no components were registered
	allComponents := registry.GetAllComponents()
	if len(allComponents) != 0 {
		t.Errorf("Expected 0 components after invalid registrations, got %d", len(allComponents))
	}
}
