package engines

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestMemoryEngineQueueDynamics tests realistic queue behavior with proper overflow and recovery
func TestMemoryEngineQueueDynamics(t *testing.T) {
	t.Log("🔄 MEMORY ENGINE QUEUE DYNAMICS TEST")
	
	// Create memory engine with basic complexity for predictable behavior
	profileLoader := NewProfileLoader("../../profiles")
	wrapper, err := NewEngineWrapperWithProfile(MemoryEngineType, "ddr4_3200_dual_channel", 1, profileLoader)
	if err != nil {
		t.Fatalf("Failed to create memory wrapper: %v", err)
	}
	defer wrapper.Stop()
	
	wrapper.SetDefaultRouting("drain")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	err = wrapper.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start memory wrapper: %v", err)
	}
	
	memEngine := wrapper.engine.(*MemoryEngine)
	
	// Get actual queue capacities
	wrapperCap := cap(wrapper.inputQueue)
	engineCap := memEngine.GetQueueCapacity()
	
	t.Logf("📊 Queue Configuration:")
	t.Logf("  - Wrapper Queue Capacity: %d", wrapperCap)
	t.Logf("  - Engine Queue Capacity: %d", engineCap)
	t.Logf("  - Memory Channels: %d", memEngine.Channels)
	
	// Test 1: Fill queue to capacity without processing
	testQueueFillWithoutProcessing(t, wrapper, memEngine, wrapperCap)
	
	// Test 2: Queue overflow with rejection
	testQueueOverflowWithRejection(t, wrapper, memEngine, wrapperCap)
	
	// Test 3: Gradual draining and refilling
	testGradualDrainingAndRefilling(t, wrapper, memEngine)
	
	// Test 4: Burst load with backpressure
	testBurstLoadWithBackpressure(t, wrapper, memEngine)
}

// testQueueFillWithoutProcessing fills queue to capacity without processing ticks
func testQueueFillWithoutProcessing(t *testing.T, wrapper *EngineWrapper, memEngine *MemoryEngine, capacity int) {
	t.Log("📋 Test 1: Fill Queue to Capacity (No Processing)")
	
	successfulQueues := 0
	rejectedQueues := 0
	
	// Fill queue exactly to capacity
	for i := 0; i < capacity+10; i++ {
		op := &Operation{
			ID:         fmt.Sprintf("fill-test-%d", i),
			Type:       "memory_read",
			Complexity: "O(1)",
			DataSize:   1024,
			Language:   "cpp",
		}
		
		err := wrapper.QueueOperation(op)
		if err == nil {
			successfulQueues++
		} else {
			rejectedQueues++
			if rejectedQueues == 1 {
				t.Logf("  📍 First rejection at operation %d", i)
			}
		}
	}
	
	t.Logf("📊 Fill Results:")
	t.Logf("  - Successful queues: %d", successfulQueues)
	t.Logf("  - Rejected queues: %d", rejectedQueues)
	t.Logf("  - Queue length: %d", memEngine.GetQueueLength())
	
	if successfulQueues == capacity && rejectedQueues > 0 {
		t.Log("  ✅ Queue capacity enforcement working correctly")
	} else if rejectedQueues == 0 {
		t.Log("  ⚠️  No rejections - queue may be larger than expected")
	} else {
		t.Errorf("  ❌ Unexpected queue behavior: %d successful, %d rejected", successfulQueues, rejectedQueues)
	}
}

// testQueueOverflowWithRejection tests overflow behavior with immediate rejection
func testQueueOverflowWithRejection(t *testing.T, wrapper *EngineWrapper, memEngine *MemoryEngine, capacity int) {
	t.Log("📋 Test 2: Queue Overflow with Rejection")
	
	// Queue should already be full from previous test
	initialQueueLen := memEngine.GetQueueLength()
	t.Logf("  Initial queue length: %d", initialQueueLen)
	
	// Try to add more operations - should be rejected immediately
	overflowAttempts := 20
	rejectedCount := 0
	
	for i := 0; i < overflowAttempts; i++ {
		op := &Operation{
			ID:         fmt.Sprintf("overflow-test-%d", i),
			Type:       "memory_write",
			Complexity: "O(1)",
			DataSize:   2048,
			Language:   "rust",
		}
		
		err := wrapper.QueueOperation(op)
		if err != nil {
			rejectedCount++
		}
	}
	
	finalQueueLen := memEngine.GetQueueLength()
	
	t.Logf("📊 Overflow Results:")
	t.Logf("  - Overflow attempts: %d", overflowAttempts)
	t.Logf("  - Rejected operations: %d", rejectedCount)
	t.Logf("  - Queue length unchanged: %d -> %d", initialQueueLen, finalQueueLen)
	
	if rejectedCount == overflowAttempts && finalQueueLen == initialQueueLen {
		t.Log("  ✅ Queue overflow rejection working correctly")
	} else {
		t.Errorf("  ❌ Queue overflow not handled properly")
	}
}

// testGradualDrainingAndRefilling tests queue draining and refilling dynamics
func testGradualDrainingAndRefilling(t *testing.T, wrapper *EngineWrapper, memEngine *MemoryEngine) {
	t.Log("📋 Test 3: Gradual Draining and Refilling")
	
	initialQueueLen := memEngine.GetQueueLength()
	t.Logf("  Starting queue length: %d", initialQueueLen)
	
	// Process ticks to drain queue gradually
	drainTicks := int64(50)
	queueLengths := make([]int, 0, drainTicks)
	processedCounts := make([]int64, 0, drainTicks)
	
	for tick := int64(1); tick <= drainTicks; tick++ {
		// Process tick to drain queue
		wrapper.ProcessTick(tick)
		
		queueLen := memEngine.GetQueueLength()
		queueLengths = append(queueLengths, queueLen)
		
		metrics := wrapper.GetMetrics()
		processed := metrics["processed_operations"].(int64)
		processedCounts = append(processedCounts, processed)
		
		// Try to add new operation every 5 ticks
		if tick%5 == 0 {
			op := &Operation{
				ID:         fmt.Sprintf("refill-test-%d", tick),
				Type:       "memory_allocate",
				Complexity: "O(1)",
				DataSize:   4096,
				Language:   "java",
			}
			
			err := wrapper.QueueOperation(op)
			if err == nil {
				t.Logf("  ✅ Tick %d: Refilled queue (len=%d)", tick, queueLen+1)
			} else {
				t.Logf("  ⚠️  Tick %d: Refill rejected (len=%d)", tick, queueLen)
			}
		}
		
		// Log progress every 10 ticks
		if tick%10 == 0 {
			t.Logf("  📊 Tick %d: Queue=%d, Processed=%d, Active=%d", 
				tick, queueLen, processed, memEngine.ActiveOperations.Len())
		}
	}
	
	finalQueueLen := memEngine.GetQueueLength()
	finalProcessed := processedCounts[len(processedCounts)-1]
	
	t.Logf("📊 Draining Results:")
	t.Logf("  - Initial queue: %d", initialQueueLen)
	t.Logf("  - Final queue: %d", finalQueueLen)
	t.Logf("  - Operations processed: %d", finalProcessed)
	t.Logf("  - Queue reduction: %d", initialQueueLen-finalQueueLen)
	
	if finalProcessed > 0 && finalQueueLen < initialQueueLen {
		t.Log("  ✅ Queue draining and refilling working correctly")
	} else {
		t.Error("  ❌ Queue draining not working properly")
	}
}

// testBurstLoadWithBackpressure tests burst loading with realistic backpressure
func testBurstLoadWithBackpressure(t *testing.T, wrapper *EngineWrapper, memEngine *MemoryEngine) {
	t.Log("📋 Test 4: Burst Load with Backpressure")
	
	// Create burst load scenario
	burstSize := 100
	burstInterval := int64(10) // Every 10 ticks
	totalTicks := int64(100)
	
	burstCount := 0
	totalQueued := 0
	totalRejected := 0
	
	for tick := int64(100); tick < int64(100)+totalTicks; tick++ {
		// Process tick first (draining)
		wrapper.ProcessTick(tick)
		
		// Create burst every burstInterval ticks
		if tick%burstInterval == 0 {
			burstCount++
			t.Logf("  💥 Burst %d at tick %d", burstCount, tick)
			
			burstQueued := 0
			burstRejected := 0
			
			// Attempt to queue burst of operations
			for i := 0; i < burstSize; i++ {
				op := &Operation{
					ID:         fmt.Sprintf("burst-%d-%d", burstCount, i),
					Type:       "memory_read",
					Complexity: "O(1)",
					DataSize:   1024,
					Language:   "cpp",
				}
				
				err := wrapper.QueueOperation(op)
				if err == nil {
					burstQueued++
					totalQueued++
				} else {
					burstRejected++
					totalRejected++
				}
			}
			
			queueLen := memEngine.GetQueueLength()
			activeOps := memEngine.ActiveOperations.Len()
			
			t.Logf("    📊 Burst %d: Queued=%d, Rejected=%d, QueueLen=%d, Active=%d", 
				burstCount, burstQueued, burstRejected, queueLen, activeOps)
		}
		
		// Log system state every 20 ticks
		if tick%20 == 0 {
			metrics := wrapper.GetMetrics()
			processed := metrics["processed_operations"].(int64)
			utilization := memEngine.GetUtilization()
			
			t.Logf("  📈 Tick %d: Processed=%d, Util=%.1f%%, Channels=%d/%d", 
				tick, processed, utilization*100, memEngine.BusyChannels, memEngine.Channels)
		}
	}
	
	// Final processing to clear remaining operations
	for tick := int64(200); tick <= int64(220); tick++ {
		wrapper.ProcessTick(tick)
	}
	
	finalMetrics := wrapper.GetMetrics()
	finalProcessed := finalMetrics["processed_operations"].(int64)
	finalQueueLen := memEngine.GetQueueLength()
	
	t.Logf("📊 Burst Load Results:")
	t.Logf("  - Total bursts: %d", burstCount)
	t.Logf("  - Operations queued: %d", totalQueued)
	t.Logf("  - Operations rejected: %d", totalRejected)
	t.Logf("  - Operations processed: %d", finalProcessed)
	t.Logf("  - Final queue length: %d", finalQueueLen)
	t.Logf("  - Rejection rate: %.1f%%", float64(totalRejected)/float64(totalQueued+totalRejected)*100)
	
	if totalRejected > 0 && finalProcessed > 0 {
		t.Log("  ✅ Burst load with backpressure working correctly")
	} else if totalRejected == 0 {
		t.Log("  ⚠️  No backpressure detected - system may be over-provisioned")
	} else {
		t.Error("  ❌ Burst load handling failed")
	}
	
	t.Log("✅ MEMORY ENGINE QUEUE DYNAMICS TEST COMPLETED")
}
