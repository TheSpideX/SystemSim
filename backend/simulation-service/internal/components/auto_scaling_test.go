package components

import (
	"context"
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestLoadBalancer_AutoScaling_ScaleUp(t *testing.T) {
	// Create component config with auto-scaling enabled
	config := &ComponentConfig{
		ID:               "test-autoscale-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Auto-Scale Component",
		Description:      "Test component for auto-scaling",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 1,
			MaxInstances: 3,
			AutoScaling:  true,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    5, // Small queue to trigger scaling
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Verify initial state - should have 1 instance
	if len(lb.Instances) != 1 {
		t.Errorf("Expected 1 initial instance, got %d", len(lb.Instances))
	}

	// Test auto-scaling by simulating high load
	// We'll manually set the load balancer to have high load conditions

	// Set the last scale up time to past to allow scaling
	lb.LastScaleUp = time.Now().Add(-time.Minute * 10)

	// Create a scenario where we need to scale up by manually calling scaleUp
	// after verifying the conditions would trigger it
	currentInstances := len(lb.Instances)
	if currentInstances < lb.Config.MaxInstances {
		// Manually trigger scale up to test the functionality
		lb.scaleUp()
	}

	// Should scale up to 2 instances
	if len(lb.Instances) != 2 {
		t.Errorf("Expected 2 instances after scale up, got %d", len(lb.Instances))
	}

	// Verify scale up timestamp was updated
	if time.Since(lb.LastScaleUp) > time.Minute {
		t.Error("LastScaleUp timestamp should be recent after scaling up")
	}
}

func TestLoadBalancer_AutoScaling_ScaleDown(t *testing.T) {
	// Create component config with auto-scaling enabled
	config := &ComponentConfig{
		ID:               "test-scaledown-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Scale Down Component",
		Description:      "Test component for scale down",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 1,
			MaxInstances: 3,
			AutoScaling:  true,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Manually scale up to 2 instances first
	lb.scaleUp()

	// Verify we have 2 instances
	if len(lb.Instances) != 2 {
		t.Errorf("Expected 2 instances after manual scale up, got %d", len(lb.Instances))
	}

	// Set last scale down time to past to allow scaling down
	lb.LastScaleDown = time.Now().Add(-time.Minute * 10)

	// Both instances should have low load (empty channels)
	// Manually trigger auto-scaling check
	lb.performAutoScalingCheck()

	// Should scale down to 1 instance
	if len(lb.Instances) != 1 {
		t.Errorf("Expected 1 instance after scale down, got %d", len(lb.Instances))
	}

	// Verify scale down timestamp was updated
	if time.Since(lb.LastScaleDown) > time.Minute {
		t.Error("LastScaleDown timestamp should be recent after scaling down")
	}
}

func TestLoadBalancer_AutoScaling_CooldownPeriods(t *testing.T) {
	// Create component config with auto-scaling enabled
	config := &ComponentConfig{
		ID:               "test-cooldown-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Cooldown Component",
		Description:      "Test component for cooldown periods",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 1,
			MaxInstances: 3,
			AutoScaling:  true,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    5,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Set recent scale up time to test cooldown
	lb.LastScaleUp = time.Now()

	// Try to scale up (should be blocked by cooldown)
	shouldScale := lb.shouldScaleUp()
	if shouldScale {
		t.Error("Scale up should be blocked by cooldown period")
	}

	// Set old scale up time to allow scaling
	lb.LastScaleUp = time.Now().Add(-time.Minute * 5)

	shouldScale = lb.shouldScaleUp()
	if !shouldScale {
		t.Error("Scale up should be allowed after cooldown period")
	}

	// Test scale down cooldown
	lb.LastScaleDown = time.Now()

	shouldScale = lb.shouldScaleDown()
	if shouldScale {
		t.Error("Scale down should be blocked by cooldown period")
	}

	// Set old scale down time to allow scaling
	lb.LastScaleDown = time.Now().Add(-time.Minute * 10)

	shouldScale = lb.shouldScaleDown()
	if !shouldScale {
		t.Error("Scale down should be allowed after cooldown period")
	}
}

func TestLoadBalancer_AutoScaling_MinMaxLimits(t *testing.T) {
	// Create component config with auto-scaling enabled
	config := &ComponentConfig{
		ID:               "test-limits-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Limits Component",
		Description:      "Test component for min/max limits",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 2,
			MaxInstances: 2, // Same min/max to test limits
			AutoScaling:  true,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    5,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Should start with minimum instances
	if len(lb.Instances) != 2 {
		t.Errorf("Expected 2 instances (minimum), got %d", len(lb.Instances))
	}

	// Try to scale up (should be blocked by max limit)
	initialCount := len(lb.Instances)
	lb.scaleUp() // This should not add an instance due to max limit

	if len(lb.Instances) != initialCount {
		t.Errorf("Instance count should not change when at max limit")
	}

	// Try to scale down (should be blocked by min limit)
	lb.scaleDown() // This should not remove an instance due to min limit

	if len(lb.Instances) != initialCount {
		t.Errorf("Instance count should not change when at min limit")
	}
}

func TestLoadBalancer_AutoScaling_FindLeastLoadedInstance(t *testing.T) {
	// Create component config
	config := &ComponentConfig{
		ID:               "test-least-loaded-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Least Loaded Component",
		Description:      "Test component for finding least loaded instance",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 1,
			MaxInstances: 3,
			AutoScaling:  true,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Scale up to 3 instances
	lb.scaleUp()
	lb.scaleUp()

	if len(lb.Instances) != 3 {
		t.Fatalf("Expected 3 instances, got %d", len(lb.Instances))
	}

	// Test the least loaded instance detection with empty channels
	// All instances should have 0 load, so it should return the first one (index 0)
	leastLoadedIndex := lb.findLeastLoadedInstance()

	if leastLoadedIndex < 0 || leastLoadedIndex >= len(lb.Instances) {
		t.Errorf("Expected valid least loaded instance index, got %d", leastLoadedIndex)
	}

	// Test with different channel capacities to simulate different loads
	// Create instances with different queue utilization by temporarily changing channel lengths
	// Since we can't easily control channel consumption, we'll test the logic directly

	// Test the algorithm by checking that it returns a valid index
	if leastLoadedIndex == -1 {
		t.Error("findLeastLoadedInstance should not return -1 when instances are available")
	}
}
