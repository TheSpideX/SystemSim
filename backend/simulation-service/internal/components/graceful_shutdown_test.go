package components

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestLoadBalancer_GracefulShutdown_RejectsNewOperations(t *testing.T) {
	// Create component config
	config := &ComponentConfig{
		ID:               "test-graceful-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Graceful Component",
		Description:      "Test component for graceful shutdown",
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

	// Process some operations before shutdown
	for i := 0; i < 3; i++ {
		operation := &engines.Operation{
			ID:       fmt.Sprintf("pre-shutdown-op-%d", i),
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

	// Start graceful shutdown
	go func() {
		time.Sleep(time.Millisecond * 50) // Small delay to ensure operations are being processed
		lb.Stop()
	}()

	// Try to process operations after shutdown starts
	time.Sleep(time.Millisecond * 100) // Wait for shutdown to start

	operation := &engines.Operation{
		ID:       "post-shutdown-op",
		Type:     "test",
		DataSize: 1024,
		Priority: 5,
		Metadata: make(map[string]interface{}),
	}

	err = lb.ProcessOperation(operation)
	if err == nil {
		t.Error("Expected operation to be rejected after shutdown started")
	}

	if err != nil && err.Error() != fmt.Sprintf("load balancer %s is shutting down, rejecting operation %s", lb.ComponentID, operation.ID) {
		t.Errorf("Expected shutdown rejection error, got: %v", err)
	}
}

func TestLoadBalancer_GracefulShutdown_WaitsForCompletion(t *testing.T) {
	// Create component config
	config := &ComponentConfig{
		ID:               "test-completion-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Completion Component",
		Description:      "Test component for completion waiting",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 1,
			MaxInstances: 1,
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

	// Process operations
	for i := 0; i < 5; i++ {
		operation := &engines.Operation{
			ID:       fmt.Sprintf("completion-op-%d", i),
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

	// Measure shutdown time
	shutdownStart := time.Now()
	
	// Perform graceful shutdown with timeout
	err = lb.StopWithTimeout(time.Second * 5)
	if err != nil {
		t.Fatalf("Failed to shutdown gracefully: %v", err)
	}

	shutdownDuration := time.Since(shutdownStart)

	// Shutdown should take some time to complete operations
	if shutdownDuration < time.Millisecond*10 {
		t.Error("Shutdown completed too quickly, may not have waited for operations")
	}

	// But shouldn't take longer than timeout
	if shutdownDuration > time.Second*6 {
		t.Error("Shutdown took longer than expected timeout")
	}
}

func TestLoadBalancer_GracefulShutdown_Timeout(t *testing.T) {
	// Create component config
	config := &ComponentConfig{
		ID:               "test-timeout-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Timeout Component",
		Description:      "Test component for shutdown timeout",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 1,
			MaxInstances: 1,
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

	// Measure shutdown time with very short timeout
	shutdownStart := time.Now()
	
	// Perform graceful shutdown with very short timeout
	err = lb.StopWithTimeout(time.Millisecond * 100) // Very short timeout
	if err != nil {
		t.Fatalf("Failed to shutdown: %v", err)
	}

	shutdownDuration := time.Since(shutdownStart)

	// Should complete within timeout period (plus some buffer)
	if shutdownDuration > time.Millisecond*200 {
		t.Errorf("Shutdown took %v, expected around 100ms", shutdownDuration)
	}
}

func TestLoadBalancer_GracefulShutdown_ConcurrentOperations(t *testing.T) {
	// Create component config
	config := &ComponentConfig{
		ID:               "test-concurrent-component",
		Type:             ComponentTypeWebServer,
		Name:             "Test Concurrent Component",
		Description:      "Test component for concurrent operations during shutdown",
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
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

	// Start concurrent operation processing
	var wg sync.WaitGroup
	operationCount := 20
	successfulOps := 0
	rejectedOps := 0
	var mutex sync.Mutex

	for i := 0; i < operationCount; i++ {
		wg.Add(1)
		go func(opIndex int) {
			defer wg.Done()

			operation := &engines.Operation{
				ID:       fmt.Sprintf("concurrent-op-%d", opIndex),
				Type:     "test",
				DataSize: 1024,
				Priority: 5,
				Metadata: make(map[string]interface{}),
			}

			err := lb.ProcessOperation(operation)
			
			mutex.Lock()
			if err != nil {
				rejectedOps++
			} else {
				successfulOps++
			}
			mutex.Unlock()

			// Small delay to simulate processing time
			time.Sleep(time.Millisecond * 10)
		}(i)
	}

	// Start shutdown after some operations have started
	go func() {
		time.Sleep(time.Millisecond * 50)
		lb.StopWithTimeout(time.Second * 3)
	}()

	// Wait for all operations to complete
	wg.Wait()

	// Verify that operations were processed
	if successfulOps == 0 {
		t.Error("Expected some operations to be processed successfully")
	}

	// During graceful shutdown, it's acceptable for all operations to be processed
	// or for some to be rejected depending on timing. Both are valid outcomes.
	if successfulOps + rejectedOps != operationCount {
		t.Errorf("Expected total operations to equal %d, got %d successful + %d rejected = %d",
			operationCount, successfulOps, rejectedOps, successfulOps + rejectedOps)
	}

	t.Logf("Processed %d operations successfully, rejected %d operations", successfulOps, rejectedOps)
}
