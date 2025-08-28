package components

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestComponentArchitecture_AtomicFlagCoordination(t *testing.T) {
	// Test atomic flag coordination between load balancer and instances
	config := &ComponentConfig{
		ID:               "test-atomic-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Atomic Component",
		Description:      "Test component for atomic flag coordination",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 2,
			MaxInstances: 2,
			AutoScaling:  false,
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

	// Verify instances are created and ready
	if len(lb.Instances) != 2 {
		t.Fatalf("Expected 2 instances, got %d", len(lb.Instances))
	}

	// Test atomic flag states
	for i, instance := range lb.Instances {
		// Check ready flag
		if !instance.ReadyFlag.Load() {
			t.Errorf("Instance %d should be ready", i)
		}

		// Check shutdown flag (should be false initially)
		if instance.ShutdownFlag.Load() {
			t.Errorf("Instance %d should not be shutting down initially", i)
		}

		// Check processing flag (should be false initially)
		if instance.ProcessingFlag.Load() {
			t.Errorf("Instance %d should not be processing initially", i)
		}
	}

	// Test pause functionality
	instance := lb.Instances[0]
	instance.Pause()

	// Ready flag should be false after pause
	if instance.ReadyFlag.Load() {
		t.Error("Instance should not be ready after pause")
	}

	// Resume and check ready flag
	instance.Resume()
	time.Sleep(time.Millisecond * 50) // Give time for resume

	if !instance.ReadyFlag.Load() {
		t.Error("Instance should be ready after resume")
	}
}

func TestComponentArchitecture_MultiInstanceManagement(t *testing.T) {
	// Test multi-instance management with load distribution
	config := &ComponentConfig{
		ID:               "test-multi-mgmt-component",
		Type:             ComponentTypeDatabase,
		Name:             "Test Multi Management Component",
		Description:      "Test component for multi-instance management",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 3,
			MaxInstances: 5,
			AutoScaling:  true,
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

	// Verify initial instance count
	if len(lb.Instances) != 3 {
		t.Fatalf("Expected 3 initial instances, got %d", len(lb.Instances))
	}

	// Test load distribution across instances
	operationCount := 15
	var wg sync.WaitGroup
	successCount := int64(0)
	var mutex sync.Mutex

	for i := 0; i < operationCount; i++ {
		wg.Add(1)
		go func(opIndex int) {
			defer wg.Done()

			operation := &engines.Operation{
				ID:       fmt.Sprintf("multi-mgmt-op-%d", opIndex),
				Type:     "test",
				DataSize: 1024,
				Priority: 5,
				Metadata: make(map[string]interface{}),
			}

			err := lb.ProcessOperation(operation)
			if err == nil {
				mutex.Lock()
				successCount++
				mutex.Unlock()
			}
			// Note: Some operations may fail due to backpressure, which is expected behavior
		}(i)
	}

	wg.Wait()

	// Give time for processing
	time.Sleep(time.Millisecond * 200)

	// Verify operations were distributed across instances
	totalOps := int64(0)
	for _, instance := range lb.Instances {
		metrics := instance.GetMetrics()
		if metrics != nil {
			totalOps += metrics.TotalOperations
		}
	}

	if successCount == 0 {
		t.Error("Expected some operations to be processed successfully")
	}

	if totalOps == 0 {
		t.Error("Expected operations to be distributed across instances")
	}

	// Test instance health
	for i, instance := range lb.Instances {
		if !instance.IsHealthy() {
			t.Errorf("Instance %d should be healthy", i)
		}
	}
}

func TestComponentArchitecture_LoadBalancingAlgorithms(t *testing.T) {
	// Test different load balancing algorithms
	algorithms := []LoadBalancingAlgorithm{
		LoadBalancingRoundRobin,
		LoadBalancingLeastConnections,
		LoadBalancingWeighted,
	}

	for _, algorithm := range algorithms {
		t.Run(fmt.Sprintf("Algorithm_%v", algorithm), func(t *testing.T) {
			config := &ComponentConfig{
				ID:               fmt.Sprintf("test-algo-%v-component", algorithm),
				Type:             ComponentTypeWebServer,
				Name:             fmt.Sprintf("Test Algorithm %v Component", algorithm),
				Description:      fmt.Sprintf("Test component for %v algorithm", algorithm),
				LoadBalancer: &LoadBalancingConfig{
					Algorithm:    algorithm,
					MinInstances: 3,
					MaxInstances: 3,
					AutoScaling:  false,
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

			// Verify algorithm is set correctly
			if lb.Config.Algorithm != algorithm {
				t.Errorf("Expected algorithm %v, got %v", algorithm, lb.Config.Algorithm)
			}

			// Test operation routing with the algorithm
			for i := 0; i < 9; i++ { // 9 operations across 3 instances
				operation := &engines.Operation{
					ID:       fmt.Sprintf("algo-test-op-%d", i),
					Type:     "test",
					DataSize: 1024,
					Priority: 5,
					Metadata: make(map[string]interface{}),
				}

				err := lb.ProcessOperation(operation)
				if err != nil {
					t.Errorf("Failed to process operation %d: %v", i, err)
				}
			}

			// Give time for processing
			time.Sleep(time.Millisecond * 100)

			// Verify operations were processed
			totalOps := int64(0)
			for _, instance := range lb.Instances {
				metrics := instance.GetMetrics()
				if metrics != nil {
					totalOps += metrics.TotalOperations
				}
			}

			if totalOps == 0 {
				t.Errorf("Expected operations to be processed with %v algorithm", algorithm)
			}
		})
	}
}

func TestComponentArchitecture_EngineCoordination(t *testing.T) {
	// Test engine coordination within instances
	config := &ComponentConfig{
		ID:               "test-engine-coord-component",
		Type:             ComponentTypeDatabase,
		Name:             "Test Engine Coordination Component",
		Description:      "Test component for engine coordination",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingNone,
			MinInstances: 1,
			MaxInstances: 1,
			AutoScaling:  false,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType, engines.MemoryEngineType, engines.StorageEngineType},
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

	// Get the single instance
	if len(lb.Instances) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(lb.Instances))
	}

	instance := lb.Instances[0]

	// Verify engines are initialized
	if len(instance.Engines) == 0 {
		t.Fatal("Expected engines to be initialized")
	}

	// Test operation processing through engines
	operation := &engines.Operation{
		ID:       "engine-coord-test-op",
		Type:     "test",
		DataSize: 1024,
		Priority: 5,
		Metadata: make(map[string]interface{}),
	}

	err = lb.ProcessOperation(operation)
	if err != nil {
		t.Fatalf("Failed to process operation: %v", err)
	}

	// Give time for processing through all engines
	time.Sleep(time.Millisecond * 200)

	// Verify operation was processed
	metrics := instance.GetMetrics()
	if metrics == nil || metrics.TotalOperations == 0 {
		t.Error("Expected operation to be processed through engines")
	}
}

func TestComponentArchitecture_ConcurrentOperations(t *testing.T) {
	// Test concurrent operation processing
	config := &ComponentConfig{
		ID:               "test-concurrent-arch-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Concurrent Architecture Component",
		Description:      "Test component for concurrent operations",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 2,
			MaxInstances: 2,
			AutoScaling:  false,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 10,
		QueueCapacity:    20,
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

	// Test concurrent operation processing
	operationCount := 50
	var wg sync.WaitGroup
	successCount := int64(0)
	var mutex sync.Mutex

	for i := 0; i < operationCount; i++ {
		wg.Add(1)
		go func(opIndex int) {
			defer wg.Done()

			operation := &engines.Operation{
				ID:       fmt.Sprintf("concurrent-arch-op-%d", opIndex),
				Type:     "test",
				DataSize: 1024,
				Priority: 5,
				Metadata: make(map[string]interface{}),
			}

			err := lb.ProcessOperation(operation)
			if err == nil {
				mutex.Lock()
				successCount++
				mutex.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Give time for processing
	time.Sleep(time.Millisecond * 300)

	// Verify most operations were processed successfully
	if successCount == 0 {
		t.Error("Expected some operations to be processed successfully")
	}

	// Verify load balancer is still healthy
	if !lb.IsHealthy() {
		t.Error("Load balancer should be healthy after concurrent operations")
	}
}
