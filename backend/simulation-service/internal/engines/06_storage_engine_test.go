package engines

import (
	"testing"
)

// TestStorageEngineCreation tests basic storage engine creation
func TestStorageEngineCreation(t *testing.T) {
	storage := NewStorageEngine(100)
	
	if storage == nil {
		t.Fatal("Storage engine should not be nil")
	}
	
	if storage.GetEngineType() != StorageEngineType {
		t.Errorf("Expected engine type %v, got %v", StorageEngineType, storage.GetEngineType())
	}
	
	if storage.ComplexityInterface == nil {
		t.Fatal("ComplexityInterface should not be nil")
	}
	
	if storage.ActiveOperations == nil {
		t.Fatal("ActiveOperations heap should not be nil")
	}
	
	// Test default complexity level
	if storage.GetComplexityLevel() != int(ComplexityAdvanced) {
		t.Errorf("Expected default complexity level %d, got %d", ComplexityAdvanced, storage.GetComplexityLevel())
	}
}

// TestStorageEngineComplexityLevels tests complexity level management
func TestStorageEngineComplexityLevels(t *testing.T) {
	storage := NewStorageEngine(100)
	
	// Test setting different complexity levels
	levels := []int{
		int(ComplexityMinimal),
		int(ComplexityBasic),
		int(ComplexityAdvanced),
		int(ComplexityMaximum),
	}
	
	for _, level := range levels {
		err := storage.SetComplexityLevel(level)
		if err != nil {
			t.Errorf("Failed to set complexity level %d: %v", level, err)
		}
		
		if storage.GetComplexityLevel() != level {
			t.Errorf("Expected complexity level %d, got %d", level, storage.GetComplexityLevel())
		}
	}
	
	// Test invalid complexity level
	err := storage.SetComplexityLevel(-1)
	if err == nil {
		t.Error("Expected error for invalid complexity level")
	}
}

// TestStorageEngineFeatureControl tests feature enabling/disabling
func TestStorageEngineFeatureControl(t *testing.T) {
	storage := NewStorageEngine(100)
	
	// Test minimal complexity features
	storage.SetComplexityLevel(int(ComplexityMinimal))
	
	if !storage.ComplexityInterface.ShouldEnableFeature("iops_limits") {
		t.Error("IOPS limits should be enabled in minimal complexity")
	}
	
	if storage.ComplexityInterface.ShouldEnableFeature("sequential_optimization") {
		t.Error("Sequential optimization should be disabled in minimal complexity")
	}
	
	// Test maximum complexity features
	storage.SetComplexityLevel(int(ComplexityMaximum))
	
	expectedFeatures := []string{
		"iops_limits",
		"sequential_optimization",
		"queue_depth_management",
		"controller_cache",
		"thermal_throttling",
		"advanced_prefetching",
		"multi_stream_io",
		"zoned_storage",
	}
	
	for _, feature := range expectedFeatures {
		if !storage.ComplexityInterface.ShouldEnableFeature(feature) {
			t.Errorf("Feature %s should be enabled in maximum complexity", feature)
		}
	}
}

// TestStorageEngineProfileLoading tests profile loading functionality
func TestStorageEngineProfileLoading(t *testing.T) {
	storage := NewStorageEngine(100)
	
	// Create test profile
	profile := &EngineProfile{
		Name:        "Test Storage Profile",
		Type:        StorageEngineType,
		Description: "Test storage profile for unit testing",
		Version:     "1.0",
		BaselinePerformance: map[string]float64{
			"storage_type":       0, // Will be set as string separately
			"capacity_gb":        1024,
			"iops_read":          1000000,
			"iops_write":         1000000,
			"latency_read_us":    20.0,
			"latency_write_us":   25.0,
			"bandwidth_mbps":     7000.0,
			"queue_depth":        128,
			"block_size_bytes":   4096,
			"controller_cache_mb": 1024,
			"thermal_limit_c":    70.0,
		},
	}
	
	// Set storage type separately (since it's a string)
	profile.BaselinePerformance["storage_type"] = 0 // Will be handled in LoadProfile
	
	err := storage.LoadProfile(profile)
	if err != nil {
		t.Fatalf("Failed to load profile: %v", err)
	}
	
	// Verify profile values were loaded
	if storage.CapacityGB != 1024 {
		t.Errorf("Expected capacity 1024GB, got %d", storage.CapacityGB)
	}
	
	if storage.IOPSRead != 1000000 {
		t.Errorf("Expected read IOPS 1000000, got %d", storage.IOPSRead)
	}
	
	if storage.LatencyReadUs != 20.0 {
		t.Errorf("Expected read latency 20.0us, got %f", storage.LatencyReadUs)
	}
	
	if storage.BandwidthMBps != 7000.0 {
		t.Errorf("Expected bandwidth 7000MB/s, got %f", storage.BandwidthMBps)
	}
}

// TestStorageEngineBasicOperations tests basic storage operations
func TestStorageEngineBasicOperations(t *testing.T) {
	storage := NewStorageEngine(100)
	
	// Load a test profile first
	profile := createTestStorageProfile()
	err := storage.LoadProfile(profile)
	if err != nil {
		t.Fatalf("Failed to load test profile: %v", err)
	}
	
	// Test different operation types
	operations := []*Operation{
		{
			ID:            "read_op_1",
			Type:          OpStorageRead,
			DataSize:      4096,
			NextComponent: "test_component",
		},
		{
			ID:            "write_op_1",
			Type:          OpStorageWrite,
			DataSize:      4096,
			NextComponent: "test_component",
		},
		{
			ID:            "seek_op_1",
			Type:          OpStorageSeek,
			DataSize:      0,
			NextComponent: "test_component",
		},
	}
	
	for _, op := range operations {
		result := storage.ProcessOperation(op, 1000)
		
		if result == nil {
			t.Fatalf("ProcessOperation returned nil for operation %s", op.ID)
		}
		
		if result.OperationID != op.ID {
			t.Errorf("Expected operation ID %s, got %s", op.ID, result.OperationID)
		}
		
		if result.ProcessingTime <= 0 {
			t.Errorf("Processing time should be positive, got %v", result.ProcessingTime)
		}
		
		if !result.Success {
			t.Errorf("Operation %s should succeed", op.ID)
		}
		
		// Verify metrics are present
		if result.Metrics == nil {
			t.Errorf("Metrics should not be nil for operation %s", op.ID)
		}
	}
}

// TestStorageEngineTickProcessing tests tick-based processing
func TestStorageEngineTickProcessing(t *testing.T) {
	storage := NewStorageEngine(100)
	
	// Load test profile
	profile := createTestStorageProfile()
	err := storage.LoadProfile(profile)
	if err != nil {
		t.Fatalf("Failed to load test profile: %v", err)
	}
	
	// Queue some operations
	operations := []*Operation{
		{ID: "op1", Type: OpStorageRead, DataSize: 4096, NextComponent: "test"},
		{ID: "op2", Type: OpStorageWrite, DataSize: 4096, NextComponent: "test"},
		{ID: "op3", Type: OpStorageRead, DataSize: 8192, NextComponent: "test"},
	}
	
	for _, op := range operations {
		storage.QueueOperation(op)
	}
	
	// Process several ticks
	currentTick := int64(1000)
	totalResults := 0
	
	for i := 0; i < 10; i++ {
		results := storage.ProcessTick(currentTick)
		totalResults += len(results)
		currentTick++
		
		// Verify results
		for _, result := range results {
			if result.OperationID == "" {
				t.Error("Result should have operation ID")
			}
			
			if result.ProcessingTime <= 0 {
				t.Error("Processing time should be positive")
			}
		}
		
		// Break if queue is empty and no active operations
		if storage.GetQueueLength() == 0 && storage.ActiveOperations.Len() == 0 {
			break
		}
	}
	
	// Should have processed some operations
	if totalResults == 0 {
		t.Error("Should have processed some operations")
	}
}

// TestStorageEngineAccessPatterns tests access pattern optimization
func TestStorageEngineAccessPatterns(t *testing.T) {
	storage := NewStorageEngine(100)
	storage.SetComplexityLevel(int(ComplexityAdvanced))
	
	// Load test profile
	profile := createTestStorageProfile()
	err := storage.LoadProfile(profile)
	if err != nil {
		t.Fatalf("Failed to load test profile: %v", err)
	}
	
	// Test sequential vs random access patterns
	sequentialOp := &Operation{
		ID:       "seq_op",
		Type:     OpStorageRead,
		DataSize: 65536, // Large sequential read
	}
	
	randomOp := &Operation{
		ID:       "rand_op",
		Type:     OpStorageRead,
		DataSize: 4096, // Small random read
	}
	
	seqResult := storage.ProcessOperation(sequentialOp, 1000)
	randResult := storage.ProcessOperation(randomOp, 1001)
	
	// Access patterns should be determined
	seqPattern := storage.determineAccessPattern(sequentialOp)
	randPattern := storage.determineAccessPattern(randomOp)
	
	if seqPattern == "" {
		t.Error("Sequential operation should have access pattern")
	}
	
	if randPattern == "" {
		t.Error("Random operation should have access pattern")
	}
	
	// Results should have metrics
	if seqResult.Metrics["access_pattern"] == nil {
		t.Error("Sequential result should have access pattern metric")
	}
	
	if randResult.Metrics["access_pattern"] == nil {
		t.Error("Random result should have access pattern metric")
	}
}

// createTestStorageProfile creates a test storage profile for testing
func createTestStorageProfile() *EngineProfile {
	return &EngineProfile{
		Name:        "Test NVMe SSD",
		Type:        StorageEngineType,
		Description: "Test profile for storage engine testing",
		Version:     "1.0",
		BaselinePerformance: map[string]float64{
			"capacity_gb":         1024,
			"iops_read":          500000,
			"iops_write":         500000,
			"latency_read_us":    50.0,
			"latency_write_us":   60.0,
			"bandwidth_mbps":     3500.0,
			"queue_depth":        64,
			"block_size_bytes":   4096,
			"controller_cache_mb": 512,
			"thermal_limit_c":    70.0,
		},
		// PerformanceFactors are now handled in the profile structure
	}
}
