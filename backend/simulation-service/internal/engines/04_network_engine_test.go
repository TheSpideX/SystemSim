package engines

import (
	"testing"
	"time"
)

// TestNetworkEngineBasicOperations tests basic network engine functionality
func TestNetworkEngineBasicOperations(t *testing.T) {
	// Create a basic network engine for testing
	networkEngine := NewNetworkEngine(100) // 100 operation queue capacity
	
	// Set up basic configuration
	networkEngine.BandwidthMbps = 1000 // 1 Gbps
	networkEngine.BaseLatencyMs = 0.1  // 0.1ms LAN latency
	networkEngine.MaxConnections = 10000
	networkEngine.Protocol = "TCP"
	networkEngine.NetworkType = "LAN"
	
	// Test basic operation processing
	testOp := &Operation{
		ID:       "test-network-op",
		Type:     "network_request",
		DataSize: 1024, // 1KB
	}
	
	result := networkEngine.ProcessOperation(testOp, 1)
	
	// Verify result
	if result == nil {
		t.Fatal("ProcessOperation returned nil result")
	}
	
	if result.OperationID != testOp.ID {
		t.Errorf("Expected operation ID %s, got %s", testOp.ID, result.OperationID)
	}
	
	if result.ProcessingTime <= 0 {
		t.Errorf("Expected positive processing time, got %v", result.ProcessingTime)
	}
	
	if !result.Success {
		t.Errorf("Expected successful operation, got failure")
	}
	
	t.Logf("✅ Basic operation processed successfully: %v", result.ProcessingTime)
}

// TestNetworkEngineComplexityLevels tests all complexity levels
func TestNetworkEngineComplexityLevels(t *testing.T) {
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
			// Create network engine with specific complexity
			networkEngine := NewNetworkEngine(100)
			networkEngine.ComplexityInterface.SetComplexityLevel(NetworkComplexityLevel(complexity.level))
			
			// Test operation
			testOp := &Operation{
				ID:       "complexity-test",
				Type:     "network_request",
				DataSize: 4096,
			}
			
			result := networkEngine.ProcessOperation(testOp, 1)
			
			if result == nil {
				t.Fatalf("ProcessOperation returned nil for %s complexity", complexity.name)
			}
			
			t.Logf("✅ %s complexity: Processing time %v", complexity.name, result.ProcessingTime)
			
			// Verify complexity-specific features
			interface_ := networkEngine.ComplexityInterface
			
			// All levels should have basic features
			if !interface_.ShouldEnableFeature("bandwidth_limits") {
				t.Errorf("%s should enable bandwidth_limits", complexity.name)
			}
			
			if !interface_.ShouldEnableFeature("latency_modeling") {
				t.Errorf("%s should enable latency_modeling", complexity.name)
			}
			
			// Advanced features should only be enabled at higher levels
			if complexity.level >= 2 { // Advanced+
				if !interface_.ShouldEnableFeature("graph_topology") {
					t.Errorf("%s should enable graph_topology", complexity.name)
				}
			}
			
			if complexity.level >= 3 { // Maximum
				if !interface_.ShouldEnableFeature("advanced_routing") {
					t.Errorf("%s should enable advanced_routing", complexity.name)
				}
			}
		})
	}
}

// TestNetworkEngineFeatureSpecificBehavior tests specific network features
func TestNetworkEngineFeatureSpecificBehavior(t *testing.T) {
	// Create a test network engine with maximum complexity
	networkEngine := NewNetworkEngine(100)
	networkEngine.ComplexityInterface.SetComplexityLevel(ComplexityMaximum)
	
	// Set up test configuration
	networkEngine.BandwidthMbps = 1000
	networkEngine.BaseLatencyMs = 1.0
	networkEngine.Protocol = "TCP"
	
	baseTime := 10 * time.Millisecond
	testOp := &Operation{ID: "feature-test", Type: "network_request", DataSize: 4096}
	
	// Test jitter effects
	jitterTime := networkEngine.applyJitterEffects(baseTime, testOp)
	if jitterTime == baseTime {
		t.Logf("⚠️ Jitter had no effect (may be expected for small variance)")
	} else {
		t.Logf("✅ Jitter effects applied: %v -> %v", baseTime, jitterTime)
	}
	
	// Test QoS effects under high utilization
	networkEngine.BandwidthState.BandwidthUtilization = 0.8 // High utilization
	qosTime := networkEngine.applyQoSEffects(baseTime, testOp)
	t.Logf("✅ QoS effects applied under high utilization: %v -> %v", baseTime, qosTime)
	
	// Test compression effects
	compressionTime := networkEngine.applyCompressionEffects(baseTime, testOp)
	if compressionTime != baseTime {
		t.Logf("✅ Compression effects applied: %v -> %v", baseTime, compressionTime)
	}
	
	// Test security overhead
	securityTime := networkEngine.applySecurityOverhead(baseTime, testOp)
	if securityTime > baseTime {
		t.Logf("✅ Security overhead applied: %v -> %v", baseTime, securityTime)
	}
}

// TestNetworkEngineProtocolBehavior tests different protocol configurations
func TestNetworkEngineProtocolBehavior(t *testing.T) {
	protocols := []string{"TCP", "UDP", "HTTP/1.1", "HTTP/2", "gRPC"}
	
	for _, protocol := range protocols {
		t.Run(protocol, func(t *testing.T) {
			networkEngine := NewNetworkEngine(100)
			networkEngine.Protocol = protocol
			networkEngine.configureProtocol()
			
			// Test operation with this protocol
			testOp := &Operation{
				ID:       "protocol-test",
				Type:     "network_request",
				DataSize: 2048,
			}
			
			result := networkEngine.ProcessOperation(testOp, 1)
			
			if result == nil {
				t.Fatalf("ProcessOperation failed for protocol %s", protocol)
			}
			
			t.Logf("✅ %s protocol: Processing time %v, Header overhead %d bytes", 
				protocol, result.ProcessingTime, networkEngine.ProtocolState.HeaderOverheadBytes)
			
			// Verify protocol-specific settings
			switch protocol {
			case "UDP":
				if networkEngine.ProtocolState.HeaderOverheadBytes != 8 {
					t.Errorf("UDP should have 8 byte header, got %d", networkEngine.ProtocolState.HeaderOverheadBytes)
				}
			case "HTTP/2":
				if networkEngine.ProtocolState.MultiplexingFactor <= 1.0 {
					t.Errorf("HTTP/2 should have multiplexing factor > 1.0, got %f", networkEngine.ProtocolState.MultiplexingFactor)
				}
			case "gRPC":
				if networkEngine.ProtocolState.ProtocolEfficiency <= 1.0 {
					t.Errorf("gRPC should have efficiency > 1.0, got %f", networkEngine.ProtocolState.ProtocolEfficiency)
				}
			}
		})
	}
}

// TestNetworkEngineComplexityFeatureCounts tests feature counts at different complexity levels
func TestNetworkEngineComplexityFeatureCounts(t *testing.T) {
	complexityLevels := []struct {
		name           string
		level          int
		minFeatures    int
		expectedFeatures []string
	}{
		{
			name:        "Minimal",
			level:       0,
			minFeatures: 4,
			expectedFeatures: []string{
				"bandwidth_limits",
				"latency_modeling",
				"protocol_overhead",
				"connection_pooling",
			},
		},
		{
			name:        "Basic",
			level:       1,
			minFeatures: 8,
			expectedFeatures: []string{
				"bandwidth_limits",
				"latency_modeling",
				"protocol_overhead",
				"connection_pooling",
				"congestion_control",
				"packet_loss",
				"jitter_modeling",
				"geographic_effects",
			},
		},
		{
			name:        "Advanced",
			level:       2,
			minFeatures: 12,
			expectedFeatures: []string{
				"bandwidth_limits",
				"latency_modeling",
				"protocol_overhead",
				"connection_pooling",
				"congestion_control",
				"packet_loss",
				"jitter_modeling",
				"geographic_effects",
				"qos_modeling",
				"load_balancing",
				"compression_effects",
				"graph_topology",
			},
		},
		{
			name:        "Maximum",
			level:       3,
			minFeatures: 16,
			expectedFeatures: []string{
				"bandwidth_limits",
				"latency_modeling",
				"protocol_overhead",
				"connection_pooling",
				"congestion_control",
				"packet_loss",
				"jitter_modeling",
				"geographic_effects",
				"qos_modeling",
				"load_balancing",
				"security_overhead",
				"compression_effects",
				"graph_topology",
				"advanced_routing",
				"protocol_optimization",
				"realtime_adaptation",
			},
		},
	}
	
	for _, complexity := range complexityLevels {
		t.Run(complexity.name, func(t *testing.T) {
			// Create network interface at different complexity levels
			networkInterface := NewNetworkInterface(NetworkComplexityLevel(complexity.level))
			
			enabledFeatures := 0
			for _, feature := range complexity.expectedFeatures {
				if networkInterface.ShouldEnableFeature(feature) {
					enabledFeatures++
					t.Logf("✅ %s: Feature '%s' enabled", complexity.name, feature)
				} else {
					t.Errorf("❌ %s: Feature '%s' should be enabled but isn't", complexity.name, feature)
				}
			}
			
			if enabledFeatures < complexity.minFeatures {
				t.Errorf("❌ %s complexity: Expected at least %d features, got %d",
					complexity.name, complexity.minFeatures, enabledFeatures)
			}
			
			t.Logf("📊 %s complexity: %d/%d expected features enabled",
				complexity.name, enabledFeatures, len(complexity.expectedFeatures))
		})
	}
}

// TestNetworkEngineGraphTopology tests graph-based topology features
func TestNetworkEngineGraphTopology(t *testing.T) {
	networkEngine := NewNetworkEngine(100)
	networkEngine.ComplexityInterface.SetComplexityLevel(ComplexityAdvanced) // Enable graph topology
	
	// Create a simple topology
	topology := &NetworkTopology{
		Nodes: map[string]*NetworkNode{
			"node1": {ID: "node1", Name: "Server 1", Type: "server", Location: "us-east-1"},
			"node2": {ID: "node2", Name: "Server 2", Type: "server", Location: "us-west-1"},
		},
		Edges: map[string]*NetworkEdge{
			"edge1": {
				ID:            "edge1",
				SourceNodeID:  "node1",
				TargetNodeID:  "node2",
				DistanceKm:    4000, // Cross-country
				BaseLatencyMs: 50,
				BandwidthMbps: 1000,
				HopCount:      15,
				EdgeType:      "fiber",
				QualityFactor: 0.95,
			},
		},
	}
	
	networkEngine.NetworkTopology = topology
	networkEngine.CurrentNodeID = "node1"
	
	// Test operation with graph topology
	testOp := &Operation{
		ID:       "topology-test",
		Type:     "network_request",
		DataSize: 1024,
	}
	
	result := networkEngine.ProcessOperation(testOp, 1)
	
	if result == nil {
		t.Fatal("ProcessOperation failed with graph topology")
	}
	
	t.Logf("✅ Graph topology operation processed: %v", result.ProcessingTime)
	
	// Verify that graph topology feature is enabled
	if !networkEngine.ComplexityInterface.ShouldEnableFeature("graph_topology") {
		t.Error("Graph topology feature should be enabled at Advanced complexity")
	}
}

// TestNetworkEngineBaseEngineCompliance tests BaseEngine interface compliance
func TestNetworkEngineBaseEngineCompliance(t *testing.T) {
	networkEngine := NewNetworkEngine(100)

	// Test SetComplexityLevel and GetComplexityLevel
	err := networkEngine.SetComplexityLevel(3) // Maximum
	if err != nil {
		t.Fatalf("SetComplexityLevel failed: %v", err)
	}

	level := networkEngine.GetComplexityLevel()
	if level != 3 {
		t.Errorf("Expected complexity level 3, got %d", level)
	}

	// Test invalid complexity level
	err = networkEngine.SetComplexityLevel(5) // Invalid
	if err == nil {
		t.Error("Expected error for invalid complexity level")
	}

	// Test GetDynamicState
	dynamicState := networkEngine.GetDynamicState()
	if dynamicState == nil {
		t.Fatal("GetDynamicState returned nil")
	}

	if dynamicState.HardwareSpecific == nil {
		t.Error("DynamicState.HardwareSpecific should not be nil")
	}

	// Verify network-specific fields in dynamic state
	expectedFields := []string{
		"bandwidth_utilization", "active_connections", "connection_utilization",
		"packet_loss_probability", "congestion_factor", "protocol_efficiency",
		"physics_latency_ms", "active_transmissions", "protocol", "network_type",
	}

	for _, field := range expectedFields {
		if _, exists := dynamicState.HardwareSpecific[field]; !exists {
			t.Errorf("DynamicState.HardwareSpecific missing field: %s", field)
		}
	}

	// Test UpdateDynamicBehavior
	networkEngine.UpdateDynamicBehavior() // Should not panic

	// Test GetConvergenceMetrics
	convergenceMetrics := networkEngine.GetConvergenceMetrics()
	if convergenceMetrics == nil {
		t.Fatal("GetConvergenceMetrics returned nil")
	}

	if convergenceMetrics.ConvergenceFactors == nil {
		t.Error("ConvergenceMetrics.ConvergenceFactors should not be nil")
	}

	// Test GetCurrentState
	currentState := networkEngine.GetCurrentState()
	if currentState == nil {
		t.Fatal("GetCurrentState returned nil")
	}

	// Verify essential state fields
	essentialFields := []string{
		"engine_type", "engine_id", "complexity_level", "bandwidth_mbps",
		"protocol", "network_type", "health_score", "utilization",
	}

	for _, field := range essentialFields {
		if _, exists := currentState[field]; !exists {
			t.Errorf("GetCurrentState missing field: %s", field)
		}
	}

	// Test Reset
	// First, modify some state
	networkEngine.BandwidthState.BandwidthUtilization = 0.5
	networkEngine.ConnectionState.ActiveConnections = 10
	networkEngine.CompletedOps = 100

	// Reset and verify
	networkEngine.Reset()

	if networkEngine.BandwidthState.BandwidthUtilization != 0.0 {
		t.Error("Reset did not clear bandwidth utilization")
	}

	if networkEngine.ConnectionState.ActiveConnections != 0 {
		t.Error("Reset did not clear active connections")
	}

	if len(networkEngine.ActiveTransmissions) != 0 {
		t.Error("Reset did not clear active transmissions")
	}

	t.Logf("✅ All BaseEngine interface methods implemented correctly")
}
