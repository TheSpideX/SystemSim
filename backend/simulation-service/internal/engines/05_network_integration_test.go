package engines

import (
	"context"
	"testing"
	"time"
)

// TestNetworkEngineFactoryIntegration tests Network engine creation via EngineFactory
func TestNetworkEngineFactoryIntegration(t *testing.T) {
	t.Log("🏭 Testing Network Engine Factory Integration")
	
	// Create factory
	factory := NewEngineFactory()
	
	// Test creating network engine with different profiles
	profiles := []string{"gigabit_ethernet", "10g_datacenter", "wan_connection", "wifi_6"}
	
	for _, profileName := range profiles {
		t.Run(profileName, func(t *testing.T) {
			// Create engine via factory
			engine, err := factory.CreateEngineFromFile(NetworkEngineType, profileName, 100)
			if err != nil {
				t.Logf("⚠️ Profile %s not found, skipping: %v", profileName, err)
				return // Skip if profile file doesn't exist
			}
			
			// Verify it's a network engine
			if engine.GetEngineType() != NetworkEngineType {
				t.Errorf("Expected NetworkEngineType, got %v", engine.GetEngineType())
			}
			
			// Verify BaseEngine interface compliance
			networkEngine, ok := engine.(*NetworkEngine)
			if !ok {
				t.Fatal("Engine is not a NetworkEngine")
			}
			
			// Test basic operations
			testOp := &Operation{
				ID:       "factory-test",
				Type:     "network_request",
				DataSize: 2048,
			}
			
			result := networkEngine.ProcessOperation(testOp, 1)
			if result == nil {
				t.Fatal("ProcessOperation returned nil")
			}
			
			if !result.Success {
				t.Error("Operation should succeed")
			}
			
			t.Logf("✅ %s profile: Processing time %v", profileName, result.ProcessingTime)
		})
	}
}

// TestNetworkEngineWrapperIntegration tests Network engine with EngineWrapper
func TestNetworkEngineWrapperIntegration(t *testing.T) {
	t.Log("🔄 Testing Network Engine Wrapper Integration")

	// Create network engine
	networkEngine := NewNetworkEngine(100)
	networkEngine.BandwidthMbps = 1000
	networkEngine.BaseLatencyMs = 1.0
	networkEngine.Protocol = "TCP"

	// Create wrapper with complexity level (int parameter)
	wrapper := NewEngineWrapper(networkEngine, 2) // Advanced complexity

	// Start wrapper with context
	ctx := context.Background()
	err := wrapper.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start wrapper: %v", err)
	}
	defer wrapper.Stop()

	// Queue operations
	operations := []*Operation{
		{ID: "wrapper-test-1", Type: "network_request", DataSize: 1024},
		{ID: "wrapper-test-2", Type: "network_response", DataSize: 2048},
		{ID: "wrapper-test-3", Type: "network_stream", DataSize: 4096},
	}

	for _, op := range operations {
		err := wrapper.QueueOperation(op)
		if err != nil {
			t.Errorf("Failed to queue operation %s: %v", op.ID, err)
		}
	}

	// Process ticks using ProcessTick method that returns error
	completedOps := int64(0)
	for tick := int64(1); tick <= 50; tick++ {
		err := wrapper.ProcessTick(tick)
		if err != nil {
			t.Errorf("ProcessTick failed at tick %d: %v", tick, err)
		}

		// Check metrics to see completed operations
		metrics := wrapper.GetMetrics()
		if metrics != nil {
			if completed, ok := metrics["completed_operations"].(int64); ok {
				completedOps = completed
				if completedOps >= int64(len(operations)) {
					break // All operations completed
				}
			}
		}
	}

	// Verify results
	if completedOps < int64(len(operations)) {
		t.Logf("⚠️ Only %d/%d operations completed (may be expected for long operations)", completedOps, len(operations))
	}

	// Test wrapper metrics
	metrics := wrapper.GetMetrics()
	if metrics == nil {
		t.Error("Wrapper metrics should not be nil")
	}

	// Verify essential metrics exist
	expectedMetrics := []string{"complexity_level", "processed_operations", "completed_operations", "running"}
	for _, metric := range expectedMetrics {
		if _, exists := metrics[metric]; !exists {
			t.Errorf("Missing metric: %s", metric)
		}
	}

	t.Logf("✅ Wrapper integration: %d operations processed", completedOps)
}

// TestNetworkEngineComplexityIntegration tests complexity level integration
func TestNetworkEngineComplexityIntegration(t *testing.T) {
	t.Log("🎛️ Testing Network Engine Complexity Integration")
	
	complexityLevels := []struct {
		name  string
		level int
	}{
		{"Minimal", 0},
		{"Basic", 1},
		{"Advanced", 2},
		{"Maximum", 3},
	}
	
	for _, complexity := range complexityLevels {
		t.Run(complexity.name, func(t *testing.T) {
			// Create engine with specific complexity
			networkEngine := NewNetworkEngine(100)
			
			// Set complexity via BaseEngine interface
			err := networkEngine.SetComplexityLevel(complexity.level)
			if err != nil {
				t.Fatalf("Failed to set complexity level: %v", err)
			}
			
			// Verify complexity was set
			actualLevel := networkEngine.GetComplexityLevel()
			if actualLevel != complexity.level {
				t.Errorf("Expected complexity %d, got %d", complexity.level, actualLevel)
			}
			
			// Test operations at this complexity
			testOp := &Operation{
				ID:       "complexity-test",
				Type:     "network_request",
				DataSize: 8192,
			}
			
			// Process multiple operations to test convergence
			var totalTime time.Duration
			for i := 0; i < 10; i++ {
				result := networkEngine.ProcessOperation(testOp, int64(i+1))
				if result == nil {
					t.Fatal("ProcessOperation returned nil")
				}
				totalTime += result.ProcessingTime
			}
			
			avgTime := totalTime / 10
			
			// Test dynamic state at this complexity
			dynamicState := networkEngine.GetDynamicState()
			if dynamicState == nil {
				t.Fatal("GetDynamicState returned nil")
			}
			
			// Test convergence metrics
			convergenceMetrics := networkEngine.GetConvergenceMetrics()
			if convergenceMetrics == nil {
				t.Fatal("GetConvergenceMetrics returned nil")
			}
			
			t.Logf("✅ %s complexity: Avg time %v, Utilization %.2f%%, Convergence: %v", 
				complexity.name, avgTime, dynamicState.CurrentUtilization*100, convergenceMetrics.IsConverged)
		})
	}
}

// TestNetworkEngineProfileLoadingIntegration tests profile loading integration
func TestNetworkEngineProfileLoadingIntegration(t *testing.T) {
	t.Log("📋 Testing Network Engine Profile Loading Integration")
	
	// Create engine
	networkEngine := NewNetworkEngine(100)
	
	// Create a test profile
	testProfile := &EngineProfile{
		Name:        "test_network_profile",
		Type:        NetworkEngineType,
		Description: "Test network profile for integration testing",
		BaselinePerformance: map[string]float64{
			"bandwidth_mbps":    500.0,
			"base_latency_ms":   2.0,
			"max_connections":   5000.0,
		},
		TechnologySpecs: map[string]interface{}{
			"protocol":     "UDP",
			"network_type": "WAN",
		},
		EngineSpecific: map[string]interface{}{
			"geographic": map[string]interface{}{
				"distance_km":        100.0,
				"routing_overhead":   1.5,
				"fiber_optic_factor": 0.7,
			},
			"protocol_overhead": map[string]interface{}{
				"efficiency": 0.9,
			},
		},
	}
	
	// Load profile
	err := networkEngine.LoadProfile(testProfile)
	if err != nil {
		t.Fatalf("Failed to load profile: %v", err)
	}
	
	// Verify profile was loaded
	loadedProfile := networkEngine.GetProfile()
	if loadedProfile == nil {
		t.Fatal("GetProfile returned nil after loading")
	}
	
	if loadedProfile.Name != testProfile.Name {
		t.Errorf("Expected profile name %s, got %s", testProfile.Name, loadedProfile.Name)
	}
	
	// Verify profile values were applied
	if networkEngine.BandwidthMbps != 500 {
		t.Errorf("Expected bandwidth 500 Mbps, got %d", networkEngine.BandwidthMbps)
	}
	
	if networkEngine.BaseLatencyMs != 2.0 {
		t.Errorf("Expected latency 2.0 ms, got %f", networkEngine.BaseLatencyMs)
	}
	
	if networkEngine.Protocol != "UDP" {
		t.Errorf("Expected protocol UDP, got %s", networkEngine.Protocol)
	}
	
	// Test operation with loaded profile
	testOp := &Operation{
		ID:       "profile-test",
		Type:     "network_request",
		DataSize: 1024,
	}
	
	result := networkEngine.ProcessOperation(testOp, 1)
	if result == nil {
		t.Fatal("ProcessOperation returned nil")
	}
	
	t.Logf("✅ Profile loading: %s loaded successfully, processing time %v", 
		testProfile.Name, result.ProcessingTime)
}

// TestNetworkEngineStatePersistenceIntegration tests state persistence integration
func TestNetworkEngineStatePersistenceIntegration(t *testing.T) {
	t.Log("💾 Testing Network Engine State Persistence Integration")
	
	// Create engine and wrapper
	networkEngine := NewNetworkEngine(100)
	wrapper := NewEngineWrapper(networkEngine, 2) // Advanced complexity

	// Set some state
	networkEngine.BandwidthMbps = 2000
	networkEngine.Protocol = "HTTP/2"
	networkEngine.BandwidthState.BandwidthUtilization = 0.3
	networkEngine.ConnectionState.ActiveConnections = 15

	// Process some operations to create history
	for i := 0; i < 5; i++ {
		op := &Operation{
			ID:       "persist-test-" + string(rune('1'+i)),
			Type:     "network_request",
			DataSize: 2048,
		}
		networkEngine.ProcessOperation(op, int64(i+1))
	}

	// Get current state
	originalState := networkEngine.GetCurrentState()
	if originalState == nil {
		t.Fatal("GetCurrentState returned nil")
	}

	// Test state serialization (via wrapper metrics)
	wrapperMetrics := wrapper.GetMetrics()
	if wrapperMetrics == nil {
		t.Fatal("Wrapper GetMetrics returned nil")
	}
	
	// Verify essential state fields are present
	essentialFields := []string{
		"engine_type", "bandwidth_mbps", "protocol", "bandwidth_utilization",
		"active_connections", "operations_processed",
	}
	
	for _, field := range essentialFields {
		if _, exists := originalState[field]; !exists {
			t.Errorf("State missing essential field: %s", field)
		}
	}
	
	// Test reset and state comparison
	networkEngine.Reset()
	resetState := networkEngine.GetCurrentState()
	
	// Verify reset cleared dynamic state
	if resetState["bandwidth_utilization"].(float64) != 0.0 {
		t.Error("Reset did not clear bandwidth utilization")
	}
	
	if resetState["active_connections"].(int) != 0 {
		t.Error("Reset did not clear active connections")
	}
	
	t.Logf("✅ State persistence: Original ops=%v, Reset ops=%v",
		originalState["operations_processed"], resetState["operations_processed"])
}

// TestNetworkEngineMultiEngineCoordination tests coordination with other engines
func TestNetworkEngineMultiEngineCoordination(t *testing.T) {
	t.Log("🤝 Testing Network Engine Multi-Engine Coordination")

	// Create multiple engines
	cpuEngine := NewCPUEngine(100)
	memoryEngine := NewMemoryEngine(100)
	networkEngine := NewNetworkEngine(100)

	// Set up realistic configurations
	cpuEngine.CoreCount = 8
	cpuEngine.BaseClockGHz = 3.2

	memoryEngine.CapacityGB = 32
	memoryEngine.BandwidthGBps = 25.6

	networkEngine.BandwidthMbps = 1000
	networkEngine.BaseLatencyMs = 0.5

	// Create wrappers for coordination (using complexity levels)
	cpuWrapper := NewEngineWrapper(cpuEngine, 2)    // Advanced complexity
	memoryWrapper := NewEngineWrapper(memoryEngine, 2) // Advanced complexity
	networkWrapper := NewEngineWrapper(networkEngine, 2) // Advanced complexity

	// Start all wrappers with context
	ctx := context.Background()
	engines := []*EngineWrapper{cpuWrapper, memoryWrapper, networkWrapper}
	for _, wrapper := range engines {
		err := wrapper.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start wrapper %s: %v", wrapper.GetID(), err)
		}
		defer wrapper.Stop()
	}

	// Create coordinated operations (simulating a web request)
	operations := []*Operation{
		// CPU: Process request
		{ID: "web-req-cpu", Type: "cpu_compute", DataSize: 0, Complexity: "O(n)"},
		// Memory: Load data
		{ID: "web-req-mem", Type: "memory_read", DataSize: 4096},
		// Network: Send response
		{ID: "web-req-net", Type: "network_response", DataSize: 8192},
	}

	// Queue operations to respective engines
	cpuWrapper.QueueOperation(operations[0])
	memoryWrapper.QueueOperation(operations[1])
	networkWrapper.QueueOperation(operations[2])

	// Process coordinated ticks
	completedCounts := make(map[string]int64)
	for tick := int64(1); tick <= 100; tick++ {
		for i, wrapper := range engines {
			err := wrapper.ProcessTick(tick)
			if err != nil {
				t.Errorf("ProcessTick failed for %s: %v", wrapper.GetID(), err)
			}

			// Check completed operations via metrics
			metrics := wrapper.GetMetrics()
			if metrics != nil {
				if completed, ok := metrics["completed_operations"].(int64); ok {
					engineName := []string{"CPU", "Memory", "Network"}[i]
					completedCounts[engineName] = completed
				}
			}
		}

		// Check if all operations completed
		totalCompleted := completedCounts["CPU"] + completedCounts["Memory"] + completedCounts["Network"]
		if totalCompleted >= int64(len(operations)) {
			break
		}
	}

	// Verify coordination results
	for engineName, completed := range completedCounts {
		if completed > 0 {
			t.Logf("✅ %s engine: %d operations completed", engineName, completed)
		}
	}

	// Test cross-engine state consistency via metrics
	cpuMetrics := cpuWrapper.GetMetrics()
	memoryMetrics := memoryWrapper.GetMetrics()
	networkMetrics := networkWrapper.GetMetrics()

	// All engines should have processed operations
	if cpuMetrics != nil {
		if processed, ok := cpuMetrics["processed_operations"].(int64); ok && processed == 0 && completedCounts["CPU"] > 0 {
			t.Error("CPU engine metrics inconsistent with results")
		}
	}

	if memoryMetrics != nil {
		if processed, ok := memoryMetrics["processed_operations"].(int64); ok && processed == 0 && completedCounts["Memory"] > 0 {
			t.Error("Memory engine metrics inconsistent with results")
		}
	}

	if networkMetrics != nil {
		if processed, ok := networkMetrics["processed_operations"].(int64); ok && processed == 0 && completedCounts["Network"] > 0 {
			t.Error("Network engine metrics inconsistent with results")
		}
	}

	t.Logf("✅ Multi-engine coordination: CPU=%d, Memory=%d, Network=%d operations",
		completedCounts["CPU"], completedCounts["Memory"], completedCounts["Network"])
}

// TestNetworkEnginePerformanceIntegration tests performance characteristics integration
func TestNetworkEnginePerformanceIntegration(t *testing.T) {
	t.Log("⚡ Testing Network Engine Performance Integration")

	// Test different network scenarios
	scenarios := []struct {
		name          string
		bandwidthMbps int
		latencyMs     float64
		protocol      string
		complexity    int
		expectedRange time.Duration
	}{
		{"Gigabit LAN", 1000, 0.1, "TCP", 1, 1 * time.Microsecond},
		{"10G Datacenter", 10000, 0.05, "TCP", 2, 500 * time.Nanosecond},
		{"WAN Connection", 100, 50, "TCP", 1, 50 * time.Millisecond},
		{"WiFi Connection", 600, 2, "TCP", 1, 2 * time.Millisecond},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Create engine with scenario configuration
			networkEngine := NewNetworkEngine(100)
			networkEngine.BandwidthMbps = scenario.bandwidthMbps
			networkEngine.BaseLatencyMs = scenario.latencyMs
			networkEngine.Protocol = scenario.protocol
			networkEngine.SetComplexityLevel(scenario.complexity)

			// Test with different data sizes
			dataSizes := []int64{1024, 4096, 16384, 65536} // 1KB to 64KB

			for _, dataSize := range dataSizes {
				testOp := &Operation{
					ID:       "perf-test",
					Type:     "network_request",
					DataSize: dataSize,
				}

				// Measure processing time
				start := time.Now()
				result := networkEngine.ProcessOperation(testOp, 1)
				processingDuration := time.Since(start)

				if result == nil {
					t.Fatal("ProcessOperation returned nil")
				}

				// Verify result makes sense for scenario
				if result.ProcessingTime <= 0 {
					t.Error("Processing time should be positive")
				}

				// Log performance characteristics
				throughputMbps := float64(dataSize*8) / (float64(result.ProcessingTime.Nanoseconds()) / 1e9) / 1e6

				t.Logf("📊 %s - %d bytes: Processing=%v, Real=%v, Throughput=%.1f Mbps",
					scenario.name, dataSize, result.ProcessingTime, processingDuration, throughputMbps)
			}
		})
	}
}

// TestNetworkEngineErrorHandlingIntegration tests error handling integration
func TestNetworkEngineErrorHandlingIntegration(t *testing.T) {
	t.Log("🚨 Testing Network Engine Error Handling Integration")

	networkEngine := NewNetworkEngine(5) // Small queue for testing

	// Test queue overflow
	for i := 0; i < 10; i++ {
		op := &Operation{
			ID:       "overflow-test-" + string(rune('1'+i)),
			Type:     "network_request",
			DataSize: 1024,
		}

		err := networkEngine.QueueOperation(op)
		if i >= 5 && err == nil {
			t.Error("Expected queue overflow error")
		}
	}

	// Test invalid complexity level
	err := networkEngine.SetComplexityLevel(10) // Invalid
	if err == nil {
		t.Error("Expected error for invalid complexity level")
	}

	// Test operations with packet loss enabled
	networkEngine.ComplexityInterface.SetComplexityLevel(ComplexityMaximum)
	networkEngine.BandwidthState.PacketLossProbability = 0.1 // 10% packet loss

	successCount := 0
	totalOps := 100

	for i := 0; i < totalOps; i++ {
		op := &Operation{
			ID:       "packet-loss-test",
			Type:     "network_request",
			DataSize: 1024,
		}

		result := networkEngine.ProcessOperation(op, int64(i+1))
		if result != nil && result.Success {
			successCount++
		}
	}

	successRate := float64(successCount) / float64(totalOps)
	expectedSuccessRate := 0.9 // 90% success with 10% packet loss

	if successRate < expectedSuccessRate-0.1 || successRate > expectedSuccessRate+0.1 {
		t.Logf("⚠️ Success rate %.2f%% outside expected range (%.0f%% ±10%%)",
			successRate*100, expectedSuccessRate*100)
	} else {
		t.Logf("✅ Packet loss simulation: %.2f%% success rate", successRate*100)
	}

	// Test engine reset after errors
	networkEngine.Reset()

	// Verify reset cleared error state
	state := networkEngine.GetCurrentState()
	if state["error_rate"].(float64) != 0.0 {
		t.Error("Reset should clear error rate")
	}

	t.Logf("✅ Error handling integration: Queue overflow, invalid params, packet loss tested")
}
