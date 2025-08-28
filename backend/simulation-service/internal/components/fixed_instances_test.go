package components

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestLoadBalancer_FixedInstances_NoAutoScaling(t *testing.T) {
	// Create component config with fixed instances (no auto-scaling)
	config := &ComponentConfig{
		ID:               "test-fixed-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Fixed Component",
		Description:      "Test component with fixed instances",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 3,
			MaxInstances: 3, // Same min/max = fixed instances
			AutoScaling:  false, // No auto-scaling
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    10,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Should start with exactly 3 instances (fixed)
	if len(lb.Instances) != 3 {
		t.Errorf("Expected 3 fixed instances, got %d", len(lb.Instances))
	}

	// Verify all instances are ready
	for i, instance := range lb.Instances {
		if !instance.ReadyFlag.Load() {
			t.Errorf("Expected instance %d to be ready", i)
		}
	}

	// Verify auto-scaling is disabled
	if lb.Config.AutoScaling {
		t.Error("Auto-scaling should be disabled for fixed instances")
	}

	// Test load distribution across fixed instances
	operations := make([]*engines.Operation, 6)
	for i := 0; i < 6; i++ {
		operations[i] = &engines.Operation{
			ID:       fmt.Sprintf("test-op-%d", i),
			Type:     "test",
			DataSize: 1024,
			Priority: 5,
			Metadata: make(map[string]interface{}),
		}

		err = lb.ProcessOperation(operations[i])
		if err != nil {
			t.Fatalf("Failed to process operation %d: %v", i, err)
		}
	}

	// Give some time for processing
	time.Sleep(time.Millisecond * 100)

	// Verify operations were distributed (total operations should be > 0)
	totalOps := int64(0)
	for _, instance := range lb.Instances {
		metrics := instance.GetMetrics()
		if metrics != nil {
			totalOps += metrics.TotalOperations
		}
	}

	if totalOps == 0 {
		t.Error("Expected operations to be distributed across fixed instances")
	}
}

func TestLoadBalancer_FixedInstances_LoadDistribution(t *testing.T) {
	// Create component config with fixed instances
	config := &ComponentConfig{
		ID:               "test-distribution-component",
		Type:             ComponentTypeDatabase,
		Name:             "Test Distribution Component",
		Description:      "Test component for load distribution",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 4,
			MaxInstances: 4, // Fixed 4 instances
			AutoScaling:  false,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType, engines.StorageEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    10,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Should have exactly 4 instances
	if len(lb.Instances) != 4 {
		t.Errorf("Expected 4 fixed instances, got %d", len(lb.Instances))
	}

	// Test round-robin distribution
	for i := 0; i < 12; i++ { // 12 operations across 4 instances = 3 each
		operation := &engines.Operation{
			ID:       fmt.Sprintf("test-op-%d", i),
			Type:     "test",
			DataSize: 1024,
			Priority: 5,
			Metadata: make(map[string]interface{}),
		}

		err = lb.ProcessOperation(operation)
		if err != nil {
			t.Fatalf("Failed to process operation %d: %v", i, err)
		}
	}

	// Give some time for processing
	time.Sleep(time.Millisecond * 200)

	// Verify load balancer health
	if !lb.IsHealthy() {
		t.Error("Load balancer should be healthy with fixed instances")
	}

	health := lb.GetHealth()
	if health == nil {
		t.Fatal("Expected health information")
	}

	if health.Status != "GREEN" && health.Status != "YELLOW" {
		t.Errorf("Expected health status GREEN or YELLOW, got %s", health.Status)
	}
}

func TestLoadBalancer_FixedInstances_SingleInstance(t *testing.T) {
	// Test fixed single instance (invisible load balancer)
	config := &ComponentConfig{
		ID:               "test-single-fixed-component",
		Type:             ComponentTypeCache,
		Name:             "Test Single Fixed Component",
		Description:      "Test component with single fixed instance",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingNone, // Should be invisible
			MinInstances: 1,
			MaxInstances: 1, // Fixed single instance
			AutoScaling:  false,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.MemoryEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    10,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Should have exactly 1 instance (invisible load balancer)
	if len(lb.Instances) != 1 {
		t.Errorf("Expected 1 fixed instance (invisible load balancer), got %d", len(lb.Instances))
	}

	// Verify load balancing algorithm is None for single instance
	if lb.Config.Algorithm != LoadBalancingNone {
		t.Errorf("Expected LoadBalancingNone for single instance, got %v", lb.Config.Algorithm)
	}

	// Test operation processing
	operation := &engines.Operation{
		ID:       "test-single-op",
		Type:     "test",
		DataSize: 1024,
		Priority: 5,
		Metadata: make(map[string]interface{}),
	}

	err = lb.ProcessOperation(operation)
	if err != nil {
		t.Fatalf("Failed to process operation: %v", err)
	}

	// Give some time for processing
	time.Sleep(time.Millisecond * 100)

	// Verify the single instance processed the operation
	instance := lb.Instances[0]
	metrics := instance.GetMetrics()
	if metrics != nil && metrics.TotalOperations == 0 {
		t.Error("Expected single instance to process the operation")
	}
}

func TestLoadBalancer_FixedInstances_NoScalingAttempts(t *testing.T) {
	// Create component config with fixed instances
	config := &ComponentConfig{
		ID:               "test-no-scaling-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test No Scaling Component",
		Description:      "Test component that should not scale",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingLeastConnections,
			MinInstances: 2,
			MaxInstances: 2, // Fixed 2 instances
			AutoScaling:  false, // No auto-scaling
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    5, // Small queue to potentially trigger scaling if it were enabled
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Should start with exactly 2 instances
	initialInstanceCount := len(lb.Instances)
	if initialInstanceCount != 2 {
		t.Errorf("Expected 2 fixed instances, got %d", initialInstanceCount)
	}

	// Manually trigger auto-scaling check (should do nothing since auto-scaling is disabled)
	lb.performAutoScalingCheck()

	// Instance count should remain the same
	if len(lb.Instances) != initialInstanceCount {
		t.Errorf("Instance count should not change with auto-scaling disabled. Expected %d, got %d", 
			initialInstanceCount, len(lb.Instances))
	}

	// Try manual scaling operations (should respect limits)
	lb.scaleUp() // Should not add instance (already at max)
	if len(lb.Instances) != 2 {
		t.Errorf("Manual scale up should not exceed max instances. Expected 2, got %d", len(lb.Instances))
	}

	lb.scaleDown() // Should not remove instance (already at min)
	if len(lb.Instances) != 2 {
		t.Errorf("Manual scale down should not go below min instances. Expected 2, got %d", len(lb.Instances))
	}
}
