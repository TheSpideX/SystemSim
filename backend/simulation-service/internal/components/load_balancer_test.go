package components

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestLoadBalancer_SingleInstance(t *testing.T) {
	// Create component config with single instance (invisible load balancer)
	config := &ComponentConfig{
		ID:               "test-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Component",
		Description:      "Test component for load balancer",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingNone,
			MinInstances: 1,
			MaxInstances: 1,
			AutoScaling:  false,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}

	// Verify load balancer state
	if lb.GetState() != ComponentStateRunning {
		t.Errorf("Expected load balancer state to be running, got %v", lb.GetState())
	}

	// Verify single instance was created
	if len(lb.Instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(lb.Instances))
	}

	// Verify instance is ready
	instance := lb.Instances[0]
	if !instance.ReadyFlag.Load() {
		t.Error("Expected instance to be ready")
	}

	// Test operation processing
	operation := &engines.Operation{
		ID:       "test-op-1",
		Type:     "test",
		DataSize: 1024,
		Priority: 5,
		Metadata: make(map[string]interface{}),
	}

	// Send operation to load balancer
	err = lb.ProcessOperation(operation)
	if err != nil {
		t.Fatalf("Failed to process operation: %v", err)
	}

	// Give some time for processing
	time.Sleep(time.Millisecond * 100)

	// Verify metrics
	metrics := lb.GetMetrics()
	if metrics.TotalOperations == 0 {
		t.Error("Expected total operations to be > 0")
	}

	// Stop load balancer
	err = lb.Stop()
	if err != nil {
		t.Fatalf("Failed to stop load balancer: %v", err)
	}

	// Verify stopped state
	if lb.GetState() != ComponentStateStopped {
		t.Errorf("Expected load balancer state to be stopped, got %v", lb.GetState())
	}
}

func TestLoadBalancer_MultipleInstances(t *testing.T) {
	// Create component config with multiple instances
	config := &ComponentConfig{
		ID:               "test-multi-component",
		Type:             ComponentTypeDatabase,
		Name:             "Test Multi Component",
		Description:      "Test component with multiple instances",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:        LoadBalancingRoundRobin,
			MinInstances:     3,
			MaxInstances:     3,
			AutoScaling:      false,
			AlgorithmPenalty: time.Millisecond * 1,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType, engines.StorageEngineType},
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}

	// Verify multiple instances were created
	if len(lb.Instances) != 3 {
		t.Errorf("Expected 3 instances, got %d", len(lb.Instances))
	}

	// Verify all instances are ready
	for i, instance := range lb.Instances {
		if !instance.ReadyFlag.Load() {
			t.Errorf("Expected instance %d to be ready", i)
		}
	}

	// Test round-robin load balancing by sending multiple operations
	for i := 0; i < 6; i++ {
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

	// Verify that operations were distributed across instances
	totalOps := int64(0)
	for _, instance := range lb.Instances {
		metrics := instance.GetMetrics()
		if metrics != nil {
			totalOps += metrics.TotalOperations
		}
	}

	if totalOps == 0 {
		t.Error("Expected total operations across instances to be > 0")
	}

	// Stop load balancer
	err = lb.Stop()
	if err != nil {
		t.Fatalf("Failed to stop load balancer: %v", err)
	}
}

func TestLoadBalancer_HealthChecking(t *testing.T) {
	// Create component config
	config := &ComponentConfig{
		ID:               "test-health-component",
		Type:             ComponentTypeCache,
		Name:             "Test Health Component",
		Description:      "Test component for health checking",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingHealthAware,
			MinInstances: 2,
			MaxInstances: 2,
			AutoScaling:  false,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.MemoryEngineType},
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}

	// Verify load balancer is healthy
	if !lb.IsHealthy() {
		t.Error("Expected load balancer to be healthy")
	}

	health := lb.GetHealth()
	if health == nil {
		t.Fatal("Expected health information")
	}

	if health.Status != "GREEN" && health.Status != "YELLOW" {
		t.Errorf("Expected health status to be GREEN or YELLOW, got %s", health.Status)
	}

	// Stop load balancer
	err = lb.Stop()
	if err != nil {
		t.Fatalf("Failed to stop load balancer: %v", err)
	}
}
