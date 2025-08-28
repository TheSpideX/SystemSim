package components

import (
	"context"
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestLoadBalancer_WeightedSelection(t *testing.T) {
	// Create component config with weighted load balancing
	config := &ComponentConfig{
		ID:               "test-weighted-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Weighted Component",
		Description:      "Test component for weighted load balancing",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:       LoadBalancingWeighted,
			MinInstances:    3,
			MaxInstances:    3,
			AutoScaling:     false,
			InstanceWeights: map[string]int{
				"test-weighted-component-instance-1": 1,
				"test-weighted-component-instance-2": 2,
				"test-weighted-component-instance-3": 3,
			},
			DefaultWeight: 1,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    50,
		TickTimeout:      time.Millisecond * 10,
		EngineProfiles:   make(map[engines.EngineType]string),
		ComplexityLevels: make(map[engines.EngineType]int),
	}

	// Create load balancer
	lb, err := NewLoadBalancer(config)
	if err != nil {
		t.Fatalf("Failed to create load balancer: %v", err)
	}

	// Start load balancer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Wait for instances to be ready
	time.Sleep(100 * time.Millisecond)

	// Verify instances were created with correct weights
	if len(lb.Instances) != 3 {
		t.Fatalf("Expected 3 instances, got %d", len(lb.Instances))
	}

	// Test weight configuration
	weights := lb.GetInstanceWeights()
	expectedWeights := map[string]int{
		"test-weighted-component-instance-1": 1,
		"test-weighted-component-instance-2": 2,
		"test-weighted-component-instance-3": 3,
	}

	for instanceID, expectedWeight := range expectedWeights {
		if actualWeight, exists := weights[instanceID]; !exists {
			t.Errorf("Expected weight for instance %s not found", instanceID)
		} else if actualWeight != expectedWeight {
			t.Errorf("Expected weight %d for instance %s, got %d", expectedWeight, instanceID, actualWeight)
		}
	}

	// Test weighted selection distribution
	selectionCounts := make(map[string]int)
	totalSelections := 600 // Should be divisible by total weight (1+2+3=6)

	// Perform selections
	for i := 0; i < totalSelections; i++ {
		// Get healthy instances
		healthyInstances := make([]*ComponentInstance, 0)
		for _, instance := range lb.Instances {
			if readyFlag, exists := lb.InstanceReady[instance.ID]; exists && readyFlag.Load() {
				if shutdownFlag, exists := lb.InstanceShutdown[instance.ID]; !exists || !shutdownFlag.Load() {
					healthyInstances = append(healthyInstances, instance)
				}
			}
		}

		if len(healthyInstances) == 0 {
			t.Fatal("No healthy instances available for selection")
		}

		// Select instance using weighted algorithm
		selected := lb.selectWeighted(healthyInstances)
		if selected == nil {
			t.Fatal("selectWeighted returned nil")
		}

		selectionCounts[selected.ID]++
	}

	// Verify distribution matches weights
	totalWeight := 6 // 1 + 2 + 3
	for instanceID, expectedWeight := range expectedWeights {
		expectedSelections := (totalSelections * expectedWeight) / totalWeight
		actualSelections := selectionCounts[instanceID]

		// Allow for some variance due to round-robin within weighted pool
		tolerance := expectedSelections / 10 // 10% tolerance
		if actualSelections < expectedSelections-tolerance || actualSelections > expectedSelections+tolerance {
			t.Errorf("Instance %s: expected ~%d selections (weight %d), got %d", 
				instanceID, expectedSelections, expectedWeight, actualSelections)
		}

		t.Logf("Instance %s: weight=%d, selections=%d (expected ~%d)", 
			instanceID, expectedWeight, actualSelections, expectedSelections)
	}
}

func TestLoadBalancer_WeightedSelectionFallback(t *testing.T) {
	// Create component config with weighted algorithm but no weights configured
	config := &ComponentConfig{
		ID:               "test-fallback-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Fallback Component",
		Description:      "Test component for weighted fallback",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingWeighted,
			MinInstances: 2,
			MaxInstances: 2,
			AutoScaling:  false,
			DefaultWeight: 1,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    50,
		TickTimeout:      time.Millisecond * 10,
		EngineProfiles:   make(map[engines.EngineType]string),
		ComplexityLevels: make(map[engines.EngineType]int),
	}

	// Create load balancer
	lb, err := NewLoadBalancer(config)
	if err != nil {
		t.Fatalf("Failed to create load balancer: %v", err)
	}

	// Start load balancer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Wait for instances to be ready
	time.Sleep(100 * time.Millisecond)

	// Test that all instances get default weight
	weights := lb.GetInstanceWeights()
	for instanceID, weight := range weights {
		if weight != 1 {
			t.Errorf("Expected default weight 1 for instance %s, got %d", instanceID, weight)
		}
	}

	// Test that selection works (should fall back to round-robin behavior)
	healthyInstances := make([]*ComponentInstance, 0)
	for _, instance := range lb.Instances {
		if readyFlag, exists := lb.InstanceReady[instance.ID]; exists && readyFlag.Load() {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) != 2 {
		t.Fatalf("Expected 2 healthy instances, got %d", len(healthyInstances))
	}

	// Test multiple selections
	selectionCounts := make(map[string]int)
	for i := 0; i < 100; i++ {
		selected := lb.selectWeighted(healthyInstances)
		if selected == nil {
			t.Fatal("selectWeighted returned nil")
		}
		selectionCounts[selected.ID]++
	}

	// With equal weights, should be roughly equal distribution
	for instanceID, count := range selectionCounts {
		if count < 40 || count > 60 { // Allow 40-60 range for 50 expected
			t.Errorf("Instance %s: expected ~50 selections, got %d", instanceID, count)
		}
	}
}

func TestLoadBalancer_SetInstanceWeight(t *testing.T) {
	// Create component config
	config := &ComponentConfig{
		ID:               "test-weight-update-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Weight Update Component",
		Description:      "Test component for weight updates",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingWeighted,
			MinInstances: 2,
			MaxInstances: 2,
			AutoScaling:  false,
			DefaultWeight: 1,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    50,
		TickTimeout:      time.Millisecond * 10,
		EngineProfiles:   make(map[engines.EngineType]string),
		ComplexityLevels: make(map[engines.EngineType]int),
	}

	// Create load balancer
	lb, err := NewLoadBalancer(config)
	if err != nil {
		t.Fatalf("Failed to create load balancer: %v", err)
	}

	// Start load balancer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Wait for instances to be ready
	time.Sleep(100 * time.Millisecond)

	// Get first instance ID
	if len(lb.Instances) < 1 {
		t.Fatal("No instances created")
	}
	firstInstanceID := lb.Instances[0].ID

	// Test setting weight
	err = lb.SetInstanceWeight(firstInstanceID, 5)
	if err != nil {
		t.Errorf("Failed to set instance weight: %v", err)
	}

	// Verify weight was set
	weights := lb.GetInstanceWeights()
	if weights[firstInstanceID] != 5 {
		t.Errorf("Expected weight 5 for instance %s, got %d", firstInstanceID, weights[firstInstanceID])
	}

	// Test invalid weight
	err = lb.SetInstanceWeight(firstInstanceID, 0)
	if err == nil {
		t.Error("Expected error when setting weight to 0")
	}

	err = lb.SetInstanceWeight(firstInstanceID, -1)
	if err == nil {
		t.Error("Expected error when setting negative weight")
	}

	// Test setting weight for non-existent instance
	err = lb.SetInstanceWeight("non-existent-instance", 3)
	if err != nil {
		t.Logf("Expected behavior: setting weight for non-existent instance: %v", err)
	}
}
