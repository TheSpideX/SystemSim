package engines

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestMemoryEngineBasicOperations tests basic memory engine functionality
func TestMemoryEngineBasicOperations(t *testing.T) {
	t.Log("🧠 MEMORY ENGINE BASIC OPERATIONS TEST")
	
	profileLoader := NewProfileLoader("../../profiles")
	wrapper, err := NewEngineWrapperWithProfile(MemoryEngineType, "ddr4_3200_dual_channel", 2, profileLoader)
	if err != nil {
		t.Fatalf("Failed to create memory wrapper: %v", err)
	}
	defer wrapper.Stop()
	
	wrapper.SetDefaultRouting("drain")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err = wrapper.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start memory wrapper: %v", err)
	}
	
	// Test different memory operation types
	testOps := []*Operation{
		{
			ID:         "mem-read-1",
			Type:       "memory_read",
			Complexity: "O(1)",
			DataSize:   64,    // Cache line size
			Language:   "cpp",
		},
		{
			ID:         "mem-write-1",
			Type:       "memory_write", 
			Complexity: "O(1)",
			DataSize:   1024,  // 1KB
			Language:   "rust",
		},
		{
			ID:         "mem-large-1",
			Type:       "memory_read",
			Complexity: "O(1)",
			DataSize:   1048576, // 1MB - should use multiple channels
			Language:   "cpp",
		},
	}
	
	t.Log("📥 Queueing memory operations:")
	for _, op := range testOps {
		err := wrapper.QueueOperation(op)
		if err != nil {
			t.Errorf("Failed to queue operation %s: %v", op.ID, err)
		} else {
			t.Logf("  ✅ Queued %s (%d bytes)", op.ID, op.DataSize)
		}
	}
	
	// Let operations process
	time.Sleep(200 * time.Millisecond)
	
	// Check metrics
	metrics := wrapper.GetMetrics()
	t.Logf("📊 Final metrics:")
	t.Logf("  - Architecture: %v", metrics["architecture"])
	t.Logf("  - Processed operations: %v", metrics["processed_operations"])
	t.Logf("  - Engine utilization: %v", metrics["engine_utilization"])
	
	// Verify architecture
	if arch := metrics["architecture"]; arch != "single_goroutine_sequential" {
		t.Errorf("❌ Expected single_goroutine_sequential, got: %v", arch)
	}
	
	t.Log("✅ MEMORY ENGINE BASIC OPERATIONS TEST COMPLETED")
}

// TestMemoryEngineRealisticBehavior tests realistic memory behavior modeling
func TestMemoryEngineRealisticBehavior(t *testing.T) {
	t.Log("🧠 MEMORY ENGINE REALISTIC BEHAVIOR TEST")
	
	profileLoader := NewProfileLoader("../../profiles")
	wrapper, err := NewEngineWrapperWithProfile(MemoryEngineType, "ddr4_3200_dual_channel", 2, profileLoader)
	if err != nil {
		t.Fatalf("Failed to create memory wrapper: %v", err)
	}
	defer wrapper.Stop()
	
	wrapper.SetDefaultRouting("drain")
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	err = wrapper.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start memory wrapper: %v", err)
	}
	
	// Test different access patterns
	accessPatterns := []struct {
		name     string
		dataSize int64
		expected string
	}{
		{"sequential_large", 2097152, "sequential"}, // 2MB
		{"random_small", 64, "random"},              // 64B
		{"stride_medium", 65536, "stride"},          // 64KB
	}
	
	t.Log("🔍 Testing memory access patterns:")
	for i, pattern := range accessPatterns {
		op := &Operation{
			ID:         fmt.Sprintf("pattern-test-%d", i),
			Type:       "memory_read",
			Complexity: "O(1)",
			DataSize:   pattern.dataSize,
			Language:   "cpp",
		}
		
		err := wrapper.QueueOperation(op)
		if err != nil {
			t.Errorf("Failed to queue %s operation: %v", pattern.name, err)
		} else {
			t.Logf("  ✅ Queued %s pattern (%d bytes)", pattern.name, pattern.dataSize)
		}
	}
	
	// Let operations process and converge
	time.Sleep(500 * time.Millisecond)
	
	// Check final metrics
	metrics := wrapper.GetMetrics()
	t.Logf("📊 Realistic behavior metrics:")
	t.Logf("  - Processed operations: %v", metrics["processed_operations"])
	t.Logf("  - Engine health: %v", metrics["engine_health"])
	
	if processed := metrics["processed_operations"].(int64); processed < 3 {
		t.Errorf("❌ Expected at least 3 processed operations, got: %d", processed)
	}
	
	t.Log("✅ MEMORY ENGINE REALISTIC BEHAVIOR TEST COMPLETED")
}

// TestMemoryEngineStatePersistence tests memory engine state save/restore
func TestMemoryEngineStatePersistence(t *testing.T) {
	t.Log("🧠 MEMORY ENGINE STATE PERSISTENCE TEST")
	
	// This test validates that our updated memory engine works with state persistence
	profileLoader := NewProfileLoader("../../profiles")
	wrapper, err := NewEngineWrapperWithProfile(MemoryEngineType, "ddr4_3200_dual_channel", 2, profileLoader)
	if err != nil {
		t.Fatalf("Failed to create memory wrapper: %v", err)
	}
	defer wrapper.Stop()
	
	wrapper.SetDefaultRouting("drain")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err = wrapper.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start memory wrapper: %v", err)
	}
	
	// Process some operations to create state
	op := &Operation{
		ID:         "persist-mem-1",
		Type:       "memory_read",
		Complexity: "O(1)",
		DataSize:   4096,
		Language:   "cpp",
	}
	
	err = wrapper.QueueOperation(op)
	if err != nil {
		t.Fatalf("Failed to queue operation: %v", err)
	}
	
	time.Sleep(100 * time.Millisecond)
	
	// Test that state persistence works with consolidated wrapper
	wrapper.SetStateDirectory("/tmp/memory_test")
	defer os.RemoveAll("/tmp/memory_test")

	// This should not fail with our consolidated wrapper state persistence
	err = wrapper.SaveState()
	if err != nil {
		t.Errorf("❌ Failed to save memory engine state: %v", err)
	} else {
		t.Log("✅ Memory engine state saved successfully")
	}
	
	t.Log("✅ MEMORY ENGINE STATE PERSISTENCE TEST COMPLETED")
}
