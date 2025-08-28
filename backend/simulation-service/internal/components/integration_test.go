package components

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestIntegration_EndToEndOperationFlow(t *testing.T) {
	// Test complete operation flow from input to output
	config := &ComponentConfig{
		ID:               "test-e2e-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test End-to-End Component",
		Description:      "Test component for end-to-end operation flow",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 2,
			MaxInstances: 2,
			AutoScaling:  false,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType, engines.MemoryEngineType},
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

	// Create test operations
	operations := []*engines.Operation{
		{
			ID:       "e2e-op-1",
			Type:     "web_request",
			DataSize: 1024,
			Priority: 5,
			Metadata: map[string]interface{}{
				"url":    "/api/users",
				"method": "GET",
			},
		},
		{
			ID:       "e2e-op-2",
			Type:     "web_request",
			DataSize: 2048,
			Priority: 3,
			Metadata: map[string]interface{}{
				"url":    "/api/orders",
				"method": "POST",
			},
		},
		{
			ID:       "e2e-op-3",
			Type:     "web_request",
			DataSize: 512,
			Priority: 7,
			Metadata: map[string]interface{}{
				"url":    "/api/health",
				"method": "GET",
			},
		},
	}

	// Process operations
	for _, op := range operations {
		err := lb.ProcessOperation(op)
		if err != nil {
			t.Errorf("Failed to process operation %s: %v", op.ID, err)
		}
	}

	// Give time for processing through all engines
	time.Sleep(time.Millisecond * 300)

	// Verify operations were processed
	totalOps := int64(0)
	for _, instance := range lb.Instances {
		metrics := instance.GetMetrics()
		if metrics != nil {
			totalOps += metrics.TotalOperations
		}
	}

	if totalOps == 0 {
		t.Error("Expected operations to be processed through complete flow")
	}

	// Verify load balancer health
	if !lb.IsHealthy() {
		t.Error("Load balancer should be healthy after processing operations")
	}

	health := lb.GetHealth()
	if health == nil {
		t.Fatal("Expected health information")
	}

	if health.Status != "GREEN" && health.Status != "YELLOW" {
		t.Errorf("Expected health status GREEN or YELLOW, got %s", health.Status)
	}
}

func TestIntegration_MultiComponentSystem(t *testing.T) {
	// Test a multi-component system with different component types
	components := make([]*LoadBalancer, 0)

	// Create web server component
	webConfig := &ComponentConfig{
		ID:               "web-server-component",
		Type:             ComponentTypeWebServer,
		Name:             "Web Server",
		Description:      "Frontend web server component",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 2,
			MaxInstances: 4,
			AutoScaling:  true,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 10,
		QueueCapacity:    20,
		TickTimeout:      time.Millisecond * 10,
		EngineProfiles:   make(map[engines.EngineType]string),
		ComplexityLevels: make(map[engines.EngineType]int),
	}

	webServer, err := NewLoadBalancer(webConfig)
	if err != nil {
		t.Fatalf("Failed to create web server: %v", err)
	}
	components = append(components, webServer)

	// Create database component
	dbConfig := &ComponentConfig{
		ID:               "database-component",
		Type:             ComponentTypeDatabase,
		Name:             "Database Server",
		Description:      "Backend database component",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingLeastConnections,
			MinInstances: 1,
			MaxInstances: 3,
			AutoScaling:  true,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType, engines.StorageEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    15,
		TickTimeout:      time.Millisecond * 10,
		EngineProfiles:   make(map[engines.EngineType]string),
		ComplexityLevels: make(map[engines.EngineType]int),
	}

	database, err := NewLoadBalancer(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	components = append(components, database)

	// Create cache component
	cacheConfig := &ComponentConfig{
		ID:               "cache-component",
		Type:             ComponentTypeCache,
		Name:             "Cache Server",
		Description:      "In-memory cache component",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingNone, // Single instance cache
			MinInstances: 1,
			MaxInstances: 1,
			AutoScaling:  false,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.MemoryEngineType},
		MaxConcurrentOps: 15,
		QueueCapacity:    25,
		TickTimeout:      time.Millisecond * 10,
		EngineProfiles:   make(map[engines.EngineType]string),
		ComplexityLevels: make(map[engines.EngineType]int),
	}

	cache, err := NewLoadBalancer(cacheConfig)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	components = append(components, cache)

	// Start all components
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	for _, component := range components {
		err := component.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start component %s: %v", component.ComponentID, err)
		}
		defer component.Stop()
	}

	// Simulate realistic workload
	var wg sync.WaitGroup
	operationCount := 30

	for i := 0; i < operationCount; i++ {
		wg.Add(1)
		go func(opIndex int) {
			defer wg.Done()

			// Simulate different types of operations
			var targetComponent *LoadBalancer
			var operation *engines.Operation

			switch opIndex % 3 {
			case 0: // Web request
				targetComponent = webServer
				operation = &engines.Operation{
					ID:       fmt.Sprintf("web-op-%d", opIndex),
					Type:     "web_request",
					DataSize: 1024 + int64(opIndex*100),
					Priority: 5,
					Metadata: map[string]interface{}{
						"url":    fmt.Sprintf("/api/endpoint-%d", opIndex),
						"method": "GET",
					},
				}

			case 1: // Database query
				targetComponent = database
				operation = &engines.Operation{
					ID:       fmt.Sprintf("db-op-%d", opIndex),
					Type:     "database_query",
					DataSize: 512 + int64(opIndex*50),
					Priority: 3,
					Metadata: map[string]interface{}{
						"query": fmt.Sprintf("SELECT * FROM table_%d", opIndex),
						"table": fmt.Sprintf("table_%d", opIndex%5),
					},
				}

			case 2: // Cache operation
				targetComponent = cache
				operation = &engines.Operation{
					ID:       fmt.Sprintf("cache-op-%d", opIndex),
					Type:     "cache_operation",
					DataSize: 256 + int64(opIndex*25),
					Priority: 8,
					Metadata: map[string]interface{}{
						"key":    fmt.Sprintf("cache_key_%d", opIndex),
						"action": "get",
					},
				}
			}

			err := targetComponent.ProcessOperation(operation)
			if err != nil {
				// Some operations may fail due to backpressure, which is expected
				t.Logf("Operation %s failed (expected under load): %v", operation.ID, err)
			}
		}(i)
	}

	wg.Wait()

	// Give time for processing
	time.Sleep(time.Millisecond * 500)

	// Verify all components are healthy
	for _, component := range components {
		if !component.IsHealthy() {
			t.Errorf("Component %s should be healthy", component.ComponentID)
		}

		health := component.GetHealth()
		if health == nil {
			t.Errorf("Component %s should have health information", component.ComponentID)
			continue
		}

		if health.Status == "RED" {
			t.Errorf("Component %s should not have RED health status", component.ComponentID)
		}

		// Verify instances are running
		if len(component.Instances) == 0 {
			t.Errorf("Component %s should have at least one instance", component.ComponentID)
		}

		// Verify some operations were processed
		totalOps := int64(0)
		for _, instance := range component.Instances {
			metrics := instance.GetMetrics()
			if metrics != nil {
				totalOps += metrics.TotalOperations
			}
		}

		t.Logf("Component %s processed %d operations", component.ComponentID, totalOps)
	}
}

func TestIntegration_SystemUnderLoad(t *testing.T) {
	// Test system behavior under high load
	config := &ComponentConfig{
		ID:               "load-test-component",
		Type:             ComponentTypeWebServer,
		Name:             "Load Test Component",
		Description:      "Component for load testing",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 2,
			MaxInstances: 5,
			AutoScaling:  true,
		},
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    10, // Small queue to trigger backpressure
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	err = lb.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}
	defer lb.Stop()

	// Generate high load
	var wg sync.WaitGroup
	operationCount := 100
	successCount := int64(0)
	failureCount := int64(0)
	var mutex sync.Mutex

	startTime := time.Now()

	for i := 0; i < operationCount; i++ {
		wg.Add(1)
		go func(opIndex int) {
			defer wg.Done()

			operation := &engines.Operation{
				ID:       fmt.Sprintf("load-op-%d", opIndex),
				Type:     "high_load_request",
				DataSize: 1024,
				Priority: 5,
				Metadata: map[string]interface{}{
					"load_test": true,
					"batch":     opIndex / 10,
				},
			}

			err := lb.ProcessOperation(operation)
			
			mutex.Lock()
			if err != nil {
				failureCount++
			} else {
				successCount++
			}
			mutex.Unlock()

			// Small delay to simulate realistic load pattern
			time.Sleep(time.Millisecond * 5)
		}(i)
	}

	wg.Wait()
	loadDuration := time.Since(startTime)

	// Give time for processing
	time.Sleep(time.Millisecond * 500)

	// Verify system handled load appropriately
	t.Logf("Load test completed in %v", loadDuration)
	t.Logf("Successful operations: %d", successCount)
	t.Logf("Failed operations: %d", failureCount)
	t.Logf("Success rate: %.2f%%", float64(successCount)/float64(operationCount)*100)

	// Should have processed some operations successfully
	if successCount == 0 {
		t.Error("Expected some operations to succeed under load")
	}

	// System should still be healthy (may be degraded but not failed)
	if !lb.IsHealthy() {
		t.Error("Load balancer should still be healthy after load test")
	}

	// Check if auto-scaling occurred
	if len(lb.Instances) > 2 {
		t.Logf("Auto-scaling occurred: %d instances created", len(lb.Instances))
	}

	// Verify instances are still functional
	for i, instance := range lb.Instances {
		if !instance.IsHealthy() {
			t.Errorf("Instance %d should be healthy after load test", i)
		}
	}
}
