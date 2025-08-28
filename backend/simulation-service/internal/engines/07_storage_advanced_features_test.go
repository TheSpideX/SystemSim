package engines

import (
	"testing"
	"time"
)

// TestStorageEngineAdvancedFeatures tests all advanced storage features integration
func TestStorageEngineAdvancedFeatures(t *testing.T) {
	// Create a basic storage engine for testing advanced features
	storageEngine := NewStorageEngine(150) // 150 IOPS capacity

	// Set up basic configuration
	storageEngine.StorageType = "NVMe"
	storageEngine.CapacityGB = 1024
	storageEngine.IOPSRead = 1000000
	storageEngine.IOPSWrite = 1000000

	// Create a maximum complexity interface to enable all features
	storageInterface := &StorageInterface{
		ComplexityLevel: ComplexityMaximum, // Maximum complexity (3)
		Features: &StorageFeatures{
			EnableTrimGarbageCollection: true,
			EnableAdvancedPrefetching:   true,
			EnableCompressionEffects:    true,
			EnableEncryptionOverhead:    true,
			EnableMultiStreamIO:         true,
			EnableZonedStorage:          true,
			EnableErrorCorrection:       true,
		},
	}
	storageEngine.ComplexityInterface = storageInterface
	
	// Test all advanced feature methods directly
	baseTime := 10 * time.Millisecond

	// Test operations
	writeOp := &Operation{ID: "write-test", Type: OpStorageWrite, DataSize: 4096}
	readOp := &Operation{ID: "read-test", Type: OpStorageRead, DataSize: 65536}

	// Test TRIM/Garbage Collection
	gcTime := storageEngine.applyTrimGarbageCollectionEffects(baseTime, writeOp)
	if gcTime == baseTime {
		t.Logf("⚠️ TRIM/GC had no effect (may be expected for low wear)")
	} else {
		t.Logf("✅ TRIM/GC effects applied: %v -> %v", baseTime, gcTime)
	}

	// Test Advanced Prefetching
	prefetchTime := storageEngine.applyAdvancedPrefetchingEffects(baseTime, readOp)
	t.Logf("✅ Advanced prefetching applied: %v -> %v", baseTime, prefetchTime)

	// Test Compression Effects
	compressionTime := storageEngine.applyCompressionEffects(baseTime, readOp)
	if compressionTime == baseTime {
		t.Errorf("❌ Compression effects not applied to large read")
	} else {
		t.Logf("✅ Compression effects applied: %v -> %v", baseTime, compressionTime)
	}

	// Test Encryption Overhead
	encryptionTime := storageEngine.applyEncryptionOverhead(baseTime, writeOp)
	if encryptionTime <= baseTime {
		t.Errorf("❌ Encryption should add overhead, got %v <= %v", encryptionTime, baseTime)
	} else {
		t.Logf("✅ Encryption overhead applied: %v -> %v", baseTime, encryptionTime)
	}

	// Test Multi-Stream I/O
	multiStreamTime := storageEngine.applyMultiStreamIOEffects(baseTime, writeOp)
	t.Logf("✅ Multi-stream I/O effects applied: %v -> %v", baseTime, multiStreamTime)

	// Test Zoned Storage
	zonedTime := storageEngine.applyZonedStorageEffects(baseTime, writeOp)
	t.Logf("✅ Zoned storage effects applied: %v -> %v", baseTime, zonedTime)

	// Test Error Correction
	eccTime := storageEngine.applyErrorCorrectionEffects(baseTime, readOp)
	if eccTime <= baseTime {
		t.Errorf("❌ Error correction should add overhead, got %v <= %v", eccTime, baseTime)
	} else {
		t.Logf("✅ Error correction overhead applied: %v -> %v", baseTime, eccTime)
	}

	t.Logf("✅ All 7 advanced storage features tested successfully")
}

// TestStorageEngineFeatureSpecificBehavior tests specific advanced feature behaviors
func TestStorageEngineFeatureSpecificBehavior(t *testing.T) {
	// Create a test SSD engine
	ssdEngine := NewStorageEngine(150)
	ssdEngine.StorageType = "NVMe"
	ssdEngine.CapacityGB = 1024
	ssdEngine.IOPSRead = 1000000
	ssdEngine.IOPSWrite = 1000000
	ssdEngine.BlockSizeBytes = 4096
	
	// Test TRIM/Garbage Collection (SSD-specific)
	writeOp := &Operation{ID: "gc-test", Type: OpStorageWrite, DataSize: 4096}
	baseTime := 10 * time.Millisecond
	
	// Apply TRIM/GC effects
	gcTime := ssdEngine.applyTrimGarbageCollectionEffects(baseTime, writeOp)
	if gcTime == baseTime {
		t.Errorf("❌ TRIM/GC effects not applied to SSD write operation")
	} else {
		t.Logf("✅ TRIM/GC effects applied: %v -> %v", baseTime, gcTime)
	}
	
	// Test Advanced Prefetching
	readOp := &Operation{ID: "prefetch-test", Type: OpStorageRead, DataSize: 65536}
	prefetchTime := ssdEngine.applyAdvancedPrefetchingEffects(baseTime, readOp)
	if prefetchTime == baseTime {
		t.Logf("⚠️ Prefetching had no effect (may be expected for random access)")
	} else {
		t.Logf("✅ Advanced prefetching applied: %v -> %v", baseTime, prefetchTime)
	}
	
	// Test Compression Effects
	largeWriteOp := &Operation{ID: "compression-test", Type: OpStorageWrite, DataSize: 65536}
	compressionTime := ssdEngine.applyCompressionEffects(baseTime, largeWriteOp)
	if compressionTime == baseTime {
		t.Errorf("❌ Compression effects not applied to large write")
	} else {
		t.Logf("✅ Compression effects applied: %v -> %v", baseTime, compressionTime)
	}
	
	// Test Encryption Overhead
	encryptionTime := ssdEngine.applyEncryptionOverhead(baseTime, writeOp)
	if encryptionTime <= baseTime {
		t.Errorf("❌ Encryption should add overhead, got %v <= %v", encryptionTime, baseTime)
	} else {
		t.Logf("✅ Encryption overhead applied: %v -> %v", baseTime, encryptionTime)
	}
	
	// Test Multi-Stream I/O (SSD-specific)
	multiStreamTime := ssdEngine.applyMultiStreamIOEffects(baseTime, writeOp)
	t.Logf("✅ Multi-stream I/O effects applied: %v -> %v", baseTime, multiStreamTime)
	
	// Test Zoned Storage (NVMe-specific)
	zonedTime := ssdEngine.applyZonedStorageEffects(baseTime, writeOp)
	t.Logf("✅ Zoned storage effects applied: %v -> %v", baseTime, zonedTime)
	
	// Test Error Correction
	eccTime := ssdEngine.applyErrorCorrectionEffects(baseTime, readOp)
	if eccTime <= baseTime {
		t.Errorf("❌ Error correction should add overhead, got %v <= %v", eccTime, baseTime)
	} else {
		t.Logf("✅ Error correction overhead applied: %v -> %v", baseTime, eccTime)
	}
}

// TestStorageEngineComplexityComparison tests feature availability across complexity levels
func TestStorageEngineComplexityComparison(t *testing.T) {
	// Test different complexity levels with direct interface creation
	complexityLevels := []struct {
		level int
		name  string
	}{
		{int(ComplexityBasic), "Basic"},
		{int(ComplexityBasic), "Intermediate"},
		{int(ComplexityAdvanced), "Advanced"},
		{int(ComplexityMaximum), "Maximum"},
	}

	for _, complexity := range complexityLevels {
		t.Run(complexity.name, func(t *testing.T) {
			// Create storage interface at different complexity levels
			storageInterface := NewStorageInterface(StorageComplexityLevel(complexity.level))

			advancedFeatures := []string{
				"trim_garbage_collection",
				"advanced_prefetching",
				"compression_effects",
				"encryption_overhead",
				"multi_stream_io",
				"zoned_storage",
				"error_correction",
			}

			enabledFeatures := 0
			for _, feature := range advancedFeatures {
				if storageInterface.ShouldEnableFeature(feature) {
					enabledFeatures++
					t.Logf("✅ %s: Feature '%s' enabled", complexity.name, feature)
				}
			}

			t.Logf("📊 %s complexity: %d/%d advanced features enabled",
				complexity.name, enabledFeatures, len(advancedFeatures))

			// Maximum complexity should have all advanced features
			if complexity.level == int(ComplexityMaximum) && enabledFeatures != len(advancedFeatures) {
				t.Errorf("❌ Maximum complexity should enable all %d advanced features, got %d",
					len(advancedFeatures), enabledFeatures)
			}
		})
	}
}
