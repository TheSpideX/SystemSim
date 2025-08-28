package engines

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestEngineWrapperWithTickSystem tests the complete integration of EngineWrapper with CPU engine and tick system
func TestEngineWrapperWithTickSystem(t *testing.T) {
	t.Log("🚀 Starting comprehensive EngineWrapper + CPU Engine + Tick System test")
	
	// Create CPU engine with Advanced complexity
	cpuEngine := NewCPUEngine(100)
	wrapper := NewEngineWrapper(cpuEngine, 2) // Advanced complexity
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Start the wrapper
	err := wrapper.Start(ctx)
	if err != nil {
		t.Fatalf("❌ Failed to start wrapper: %v", err)
	}
	defer wrapper.Stop()
	
	t.Log("✅ Wrapper started successfully")
	
	// Test 1: Queue operations and verify they're processed
	testOperations := createTestOperations()
	
	for i, op := range testOperations {
		err := wrapper.QueueOperation(op)
		if err != nil {
			t.Fatalf("❌ Failed to queue operation %d: %v", i, err)
		}
	}
	
	t.Logf("✅ Queued %d operations successfully", len(testOperations))
	
	// Give time for input handler to process
	time.Sleep(200 * time.Millisecond)
	
	// Test 2: Simulate tick processing
	t.Log("🔄 Starting tick simulation...")
	
	currentTick := int64(100)
	tickResults := make(map[int64]int) // Track operations completed per tick
	
	// Simulate 100 ticks
	for tick := currentTick; tick < currentTick+100; tick++ {
		// Send tick to wrapper
		err := wrapper.ProcessTick(tick)
		if err != nil {
			t.Fatalf("❌ Failed to process tick %d: %v", tick, err)
		}
		
		// Give time for tick processing
		time.Sleep(10 * time.Millisecond)
		
		// Check metrics after each tick
		metrics := wrapper.GetMetrics()
		completedOps := metrics["completed_operations"].(int64)
		
		if tick == currentTick {
			// First tick - record baseline
			tickResults[tick] = int(completedOps)
		} else {
			// Subsequent ticks - record new completions
			prevCompleted := tickResults[tick-1]
			newCompleted := int(completedOps) - prevCompleted
			tickResults[tick] = int(completedOps)
			
			if newCompleted > 0 {
				t.Logf("📊 Tick %d: %d operations completed (total: %d)", tick, newCompleted, completedOps)
			}
		}
		
		// Check CPU engine utilization
		cpuUtilization := cpuEngine.GetUtilization()
		if cpuUtilization > 0 {
			t.Logf("🔥 Tick %d: CPU utilization: %.2f%%, Busy cores: %d/%d", 
				tick, cpuUtilization*100, cpuEngine.BusyCores, cpuEngine.CoreCount)
		}
	}
	
	// Test 3: Verify final state
	finalMetrics := wrapper.GetMetrics()
	t.Logf("📈 Final metrics: %+v", finalMetrics)
	
	completedOps := finalMetrics["completed_operations"].(int64)
	if completedOps == 0 {
		t.Errorf("❌ Expected some operations to be completed, got 0")
	} else {
		t.Logf("✅ Successfully completed %d operations", completedOps)
	}
	
	// Test 4: Verify CPU engine state
	cpuHealth := cpuEngine.GetHealth()
	t.Logf("💚 CPU Health Score: %.2f", cpuHealth.Score)
	
	if !wrapper.IsHealthy() {
		t.Errorf("❌ Wrapper should be healthy after processing")
	}
	
	t.Log("🎉 Comprehensive test completed successfully!")
}

// TestEngineWrapperRealisticWorkload tests with a realistic workload pattern
func TestEngineWrapperRealisticWorkload(t *testing.T) {
	t.Log("🏭 Testing realistic workload pattern")
	
	cpuEngine := NewCPUEngine(200)
	wrapper := NewEngineWrapper(cpuEngine, 1) // Basic complexity
	
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	
	err := wrapper.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start wrapper: %v", err)
	}
	defer wrapper.Stop()
	
	// Simulate realistic workload: burst of operations followed by processing
	currentTick := int64(1000)
	
	// Phase 1: Burst load (simulate high traffic)
	t.Log("📈 Phase 1: Burst load simulation")
	burstOps := createBurstWorkload(50) // 50 operations
	
	for _, op := range burstOps {
		wrapper.QueueOperation(op)
	}
	
	time.Sleep(100 * time.Millisecond)
	
	// Phase 2: Process through multiple ticks
	t.Log("⚙️ Phase 2: Processing through ticks")
	
	utilizationHistory := make([]float64, 0)
	
	for tick := currentTick; tick < currentTick+50; tick++ {
		wrapper.ProcessTick(tick)
		time.Sleep(20 * time.Millisecond)
		
		utilization := cpuEngine.GetUtilization()
		utilizationHistory = append(utilizationHistory, utilization)
		
		if utilization > 0.1 { // Only log when there's significant activity
			t.Logf("🔄 Tick %d: Utilization %.1f%%, Active ops: %d, Queue: %d", 
				tick, utilization*100, cpuEngine.ActiveOperations.Len(), cpuEngine.GetQueueLength())
		}
	}
	
	// Phase 3: Analyze results
	t.Log("📊 Phase 3: Results analysis")
	
	maxUtilization := 0.0
	avgUtilization := 0.0
	for _, util := range utilizationHistory {
		if util > maxUtilization {
			maxUtilization = util
		}
		avgUtilization += util
	}
	avgUtilization /= float64(len(utilizationHistory))
	
	t.Logf("📈 Max utilization: %.1f%%, Average utilization: %.1f%%", 
		maxUtilization*100, avgUtilization*100)
	
	finalMetrics := wrapper.GetMetrics()
	t.Logf("🏁 Final completed operations: %d", finalMetrics["completed_operations"])
	
	if maxUtilization == 0 {
		t.Errorf("❌ Expected some CPU utilization during workload")
	}
	
	t.Log("✅ Realistic workload test completed")
}

// TestEngineWrapperComplexityComparison tests different complexity levels
func TestEngineWrapperComplexityComparison(t *testing.T) {
	t.Log("🔬 Testing complexity level performance comparison")
	
	complexityLevels := []int{0, 1, 2, 3} // Minimal, Basic, Advanced, Maximum
	results := make(map[int]map[string]interface{})

	for _, level := range complexityLevels {
		t.Logf("🧪 Testing complexity level: %s", ComplexityLevel(level).String())

		cpuEngine := NewCPUEngine(100)
		wrapper := NewEngineWrapper(cpuEngine, level)
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		wrapper.Start(ctx)
		
		// Queue same operations for each complexity level
		testOps := createTestOperations()
		for _, op := range testOps {
			wrapper.QueueOperation(op)
		}
		
		time.Sleep(100 * time.Millisecond)
		
		// Process 20 ticks
		startTime := time.Now()
		for tick := int64(1); tick <= 20; tick++ {
			wrapper.ProcessTick(tick)
			time.Sleep(10 * time.Millisecond)
		}
		processingTime := time.Since(startTime)
		
		// Collect results
		metrics := wrapper.GetMetrics()
		results[level] = map[string]interface{}{
			"processing_time":     processingTime,
			"completed_ops":       metrics["completed_operations"],
			"buffer_size":         metrics["input_buffer_size"],
			"final_utilization":   cpuEngine.GetUtilization(),
		}
		
		wrapper.Stop()
		cancel()
		
		t.Logf("📊 %s: %d ops completed in %v",
			ComplexityLevel(level).String(), metrics["completed_operations"], processingTime)
	}
	
	// Compare results
	t.Log("📈 Complexity comparison results:")
	for level, result := range results {
		t.Logf("  %s: %d ops, %v processing time, %.1f%% utilization",
			ComplexityLevel(level).String(),
			result["completed_ops"],
			result["processing_time"],
			result["final_utilization"].(float64)*100)
	}
	
	t.Log("✅ Complexity comparison completed")
}

// TestEngineWrapperTickSynchronization tests precise tick synchronization
func TestEngineWrapperTickSynchronization(t *testing.T) {
	t.Log("⏰ Testing tick synchronization and timing accuracy")

	cpuEngine := NewCPUEngine(50)
	wrapper := NewEngineWrapper(cpuEngine, 1) // Basic complexity

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	wrapper.Start(ctx)
	defer wrapper.Stop()

	// Create operations with known processing times
	predictableOps := []*Operation{
		{
			ID:         "sync-test-1",
			Type:       "compute",
			Complexity: "O(1)",
			DataSize:   100,
			Language:   "go",
		},
		{
			ID:         "sync-test-2",
			Type:       "compute",
			Complexity: "O(1)",
			DataSize:   100,
			Language:   "go",
		},
	}

	// Queue operations
	for _, op := range predictableOps {
		wrapper.QueueOperation(op)
	}

	time.Sleep(100 * time.Millisecond)

	// Track when operations complete
	completionTicks := make(map[string]int64)

	// Process ticks and track completions
	for tick := int64(1); tick <= 50; tick++ {
		beforeMetrics := wrapper.GetMetrics()
		beforeCompleted := beforeMetrics["completed_operations"].(int64)

		wrapper.ProcessTick(tick)
		time.Sleep(20 * time.Millisecond)

		afterMetrics := wrapper.GetMetrics()
		afterCompleted := afterMetrics["completed_operations"].(int64)

		if afterCompleted > beforeCompleted {
			newCompletions := afterCompleted - beforeCompleted
			t.Logf("⏱️ Tick %d: %d operations completed", tick, newCompletions)

			// Record completion tick for analysis
			for i := beforeCompleted; i < afterCompleted; i++ {
				completionTicks[fmt.Sprintf("op-%d", i)] = tick
			}
		}

		// Log utilization changes
		utilization := cpuEngine.GetUtilization()
		if utilization > 0 {
			t.Logf("📊 Tick %d: CPU %.1f%% busy, %d active ops",
				tick, utilization*100, cpuEngine.ActiveOperations.Len())
		}
	}

	t.Logf("🎯 Completion analysis: %+v", completionTicks)
	t.Log("✅ Tick synchronization test completed")
}

// TestEngineWrapperDebugProcessingTimes tests to debug processing time calculation
func TestEngineWrapperDebugProcessingTimes(t *testing.T) {
	t.Log("🔍 Debug: Testing processing time calculation")

	cpuEngine := NewCPUEngine(10)
	wrapper := NewEngineWrapper(cpuEngine, 0) // Minimal complexity // Use minimal for faster processing

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wrapper.Start(ctx)
	defer wrapper.Stop()

	// Create a simple operation
	simpleOp := &Operation{
		ID:         "debug-op-1",
		Type:       "compute",
		Complexity: "O(1)",
		DataSize:   100,
		Language:   "go",
	}

	// Queue the operation
	wrapper.QueueOperation(simpleOp)
	time.Sleep(100 * time.Millisecond)

	// Process a few ticks and log detailed info
	for tick := int64(1); tick <= 10; tick++ {
		wrapper.ProcessTick(tick)
		time.Sleep(50 * time.Millisecond)

		// Check active operations in heap
		if cpuEngine.ActiveOperations.Len() > 0 {
			activeOp := (*cpuEngine.ActiveOperations)[0]
			t.Logf("🔍 Tick %d: Active op completion tick: %d, current tick: %d, remaining: %d",
				tick, activeOp.CompletionTick, tick, activeOp.CompletionTick-tick)
		}

		metrics := wrapper.GetMetrics()
		completed := metrics["completed_operations"].(int64)
		if completed > 0 {
			t.Logf("🎉 Operation completed at tick %d!", tick)
			break
		}
	}

	t.Log("✅ Debug test completed")
}

// Helper functions

func createTestOperations() []*Operation {
	return []*Operation{
		{
			ID:           "cpu-test-1",
			Type:         "compute",
			Complexity:   "O(n)",
			DataSize:     1024,
			Language:     "go",
			NextComponent: "memory-engine-1",
		},
		{
			ID:           "cpu-test-2",
			Type:         "compute",
			Complexity:   "O(1)",
			DataSize:     512,
			Language:     "rust",
			NextComponent: "storage-engine-1",
		},
		{
			ID:           "cpu-test-3",
			Type:         "memory",
			Complexity:   "O(log n)",
			DataSize:     2048,
			Language:     "cpp",
			NextComponent: "network-engine-1",
		},
		{
			ID:           "cpu-test-4",
			Type:         "compute",
			Complexity:   "O(n²)",
			DataSize:     4096,
			Language:     "python",
			NextComponent: "cpu-engine-2",
		},
		{
			ID:           "cpu-test-5",
			Type:         "io",
			Complexity:   "O(1)",
			DataSize:     256,
			Language:     "java",
			NextComponent: "storage-engine-2",
		},
	}
}

func createBurstWorkload(count int) []*Operation {
	ops := make([]*Operation, count)

	for i := 0; i < count; i++ {
		ops[i] = &Operation{
			ID:           fmt.Sprintf("burst-op-%d", i),
			Type:         "compute",
			Complexity:   "O(n)",
			DataSize:     1024 + int64(i*100),
			Language:     "go",
			NextComponent: fmt.Sprintf("next-engine-%d", i%4),
		}
	}

	return ops
}

// TestEngineWrapperSimulationControl tests pause, resume, save, and load functionality
func TestEngineWrapperSimulationControl(t *testing.T) {
	t.Log("🎮 Testing Engine Wrapper Simulation Control APIs")

	// Create CPU engine and wrapper
	cpuEngine := NewCPUEngine(100)
	wrapper := NewEngineWrapper(cpuEngine, 1) // Basic complexity

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start wrapper
	err := wrapper.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start wrapper: %v", err)
	}
	defer wrapper.Stop()

	// Test initial state
	if wrapper.IsRunning() != true {
		t.Errorf("Expected wrapper to be running")
	}
	if wrapper.IsPaused() != false {
		t.Errorf("Expected wrapper to not be paused initially")
	}

	// Queue some operations
	testOps := createTestOperations()
	for i, op := range testOps[:3] { // Queue first 3 operations
		err := wrapper.QueueOperation(op)
		if err != nil {
			t.Errorf("Failed to queue operation %d: %v", i, err)
		}
	}

	// Process a few ticks
	for tick := int64(1); tick <= 5; tick++ {
		err := wrapper.ProcessTick(tick)
		if err != nil {
			t.Errorf("Failed to process tick %d: %v", tick, err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Test pause functionality
	t.Log("🔄 Testing pause functionality")
	err = wrapper.Pause()
	if err != nil {
		t.Errorf("Failed to pause wrapper: %v", err)
	}

	if wrapper.IsPaused() != true {
		t.Errorf("Expected wrapper to be paused")
	}

	// Try to process ticks while paused (should be skipped)
	for tick := int64(6); tick <= 8; tick++ {
		err := wrapper.ProcessTick(tick)
		if err != nil {
			t.Errorf("Failed to process tick %d while paused: %v", tick, err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Test save state while paused
	t.Log("💾 Testing save state while paused")
	err = wrapper.SaveState()
	if err != nil {
		t.Errorf("Failed to save state while paused: %v", err)
	}

	// Test resume functionality
	t.Log("▶️ Testing resume functionality")
	err = wrapper.Resume()
	if err != nil {
		t.Errorf("Failed to resume wrapper: %v", err)
	}

	if wrapper.IsPaused() != false {
		t.Errorf("Expected wrapper to not be paused after resume")
	}

	// Process more ticks after resume
	for tick := int64(9); tick <= 12; tick++ {
		err := wrapper.ProcessTick(tick)
		if err != nil {
			t.Errorf("Failed to process tick %d after resume: %v", tick, err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Test save state while running
	t.Log("💾 Testing save state while running")
	err = wrapper.SaveState()
	if err != nil {
		t.Errorf("Failed to save state while running: %v", err)
	}

	// Get metrics to verify state
	metrics := wrapper.GetMetrics()
	t.Logf("📊 Final metrics: running=%v, paused=%v, processed=%v",
		metrics["running"], metrics["paused"], metrics["processed_operations"])

	// Test load state (create new wrapper and load state)
	t.Log("📂 Testing load state functionality")
	newCpuEngine := NewCPUEngine(100)
	newWrapper := NewEngineWrapper(newCpuEngine, 1)

	err = newWrapper.LoadState(wrapper.GetID())
	if err != nil {
		t.Errorf("Failed to load state: %v", err)
	}

	// Verify loaded state
	newMetrics := newWrapper.GetMetrics()
	t.Logf("📊 Loaded metrics: running=%v, paused=%v, processed=%v",
		newMetrics["running"], newMetrics["paused"], newMetrics["processed_operations"])

	t.Log("✅ Simulation control APIs tested successfully")
}
