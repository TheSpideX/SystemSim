package engines

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// NetworkEngine implements the BaseEngine interface for network operations
type NetworkEngine struct {
	*CommonEngine

	// Complexity control interface
	ComplexityInterface *NetworkInterface `json:"complexity_interface"`

	// Profile storage
	Profile *EngineProfile `json:"profile"`

	// Network-specific properties from profile (NO HARDCODED VALUES)
	BandwidthMbps     int     `json:"bandwidth_mbps"`
	BaseLatencyMs     float64 `json:"base_latency_ms"`
	MaxConnections    int     `json:"max_connections"`
	Protocol          string  `json:"protocol"`          // TCP, UDP, HTTP/1.1, HTTP/2, gRPC
	GeographicDistance float64 `json:"geographic_distance"` // kilometers (legacy)
	NetworkType       string  `json:"network_type"`      // LAN, WAN, Internet

	// Graph-based topology (NEW)
	NetworkTopology   *NetworkTopology `json:"network_topology"`
	CurrentNodeID     string          `json:"current_node_id"`     // This engine's node ID
	
	// Dynamic bandwidth utilization state (real-time adaptation)
	BandwidthState struct {
		CurrentBandwidthMbps int     `json:"current_bandwidth_mbps"`
		BandwidthUtilization float64 `json:"bandwidth_utilization"`
		CongestionFactor     float64 `json:"congestion_factor"`
		PacketLossProbability float64 `json:"packet_loss_probability"`
		CongestionLatency    time.Duration `json:"congestion_latency"`
		LastBandwidthUpdate  int64   `json:"last_bandwidth_update"`
	} `json:"bandwidth_state"`
	
	// Dynamic connection management state
	ConnectionState struct {
		ActiveConnections    int       `json:"active_connections"`
		ConnectionUtilization float64  `json:"connection_utilization"`
		ConnectionPool       []string  `json:"connection_pool"`
		KeepAliveConnections int       `json:"keep_alive_connections"`
		ConnectionEstablishmentCost time.Duration `json:"connection_establishment_cost"`
		LastConnectionUpdate int64    `json:"last_connection_update"`
	} `json:"connection_state"`
	
	// Geographic distance effects (physics-based)
	GeographicState struct {
		PhysicsLatencyMs     float64 `json:"physics_latency_ms"`
		RoutingOverhead      float64 `json:"routing_overhead"`
		PropagationDelay     time.Duration `json:"propagation_delay"`
		FiberOpticFactor     float64 `json:"fiber_optic_factor"`
		SpeedOfLightMps      float64 `json:"speed_of_light_mps"`
	} `json:"geographic_state"`
	
	// Protocol overhead effects (dynamic efficiency)
	ProtocolState struct {
		HeaderOverheadBytes  int     `json:"header_overhead_bytes"`
		ProtocolEfficiency   float64 `json:"protocol_efficiency"`
		CompressionRatio     float64 `json:"compression_ratio"`
		MultiplexingFactor   float64 `json:"multiplexing_factor"`
		KeepAliveEnabled     bool    `json:"keep_alive_enabled"`
	} `json:"protocol_state"`
	
	// Active transmissions tracking
	ActiveTransmissions map[string]*NetworkTransmission `json:"active_transmissions"`
	TransmissionHistory []TransmissionEvent             `json:"transmission_history"`
}

// NetworkTransmission represents an active network transmission
type NetworkTransmission struct {
	OperationID      string    `json:"operation_id"`
	Type             string    `json:"type"`
	DataSizeBytes    int64     `json:"data_size_bytes"`
	StartTick        int64     `json:"start_tick"`
	EstimatedTicks   int64     `json:"estimated_ticks"`
	Protocol         string    `json:"protocol"`
	ConnectionReused bool      `json:"connection_reused"`
}

// TransmissionEvent represents a completed transmission event
type TransmissionEvent struct {
	OperationID      string        `json:"operation_id"`
	Type             string        `json:"type"`
	DataSizeBytes    int64         `json:"data_size_bytes"`
	Duration         time.Duration `json:"duration"`
	CompletedAt      int64         `json:"completed_at"`
	PacketLoss       bool          `json:"packet_loss"`
	Congestion       bool          `json:"congestion"`
}

// NetworkTopology represents a graph-based network topology with real distances
type NetworkTopology struct {
	Nodes map[string]*NetworkNode `json:"nodes"`
	Edges map[string]*NetworkEdge `json:"edges"`
}

// NetworkNode represents a node in the network topology
type NetworkNode struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`        // "server", "switch", "router", "datacenter"
	Location    string                 `json:"location"`    // "us-east-1", "eu-west-1", etc.
	Properties  map[string]interface{} `json:"properties"`
}

// NetworkEdge represents a connection between two nodes with real distance-based latency
type NetworkEdge struct {
	ID              string  `json:"id"`
	SourceNodeID    string  `json:"source_node_id"`
	TargetNodeID    string  `json:"target_node_id"`
	DistanceKm      float64 `json:"distance_km"`      // Real physical distance in kilometers
	BaseLatencyMs   float64 `json:"base_latency_ms"`  // Real measured latency for this edge
	BandwidthMbps   int     `json:"bandwidth_mbps"`   // Bandwidth capacity of this edge
	HopCount        int     `json:"hop_count"`        // Number of network hops
	EdgeType        string  `json:"edge_type"`        // "fiber", "satellite", "wireless", "ethernet"
	QualityFactor   float64 `json:"quality_factor"`   // 0.0-1.0, affects reliability and jitter
}

// NewNetworkEngine creates a new Network engine with Gigabit Ethernet defaults
func NewNetworkEngine(queueCapacity int) *NetworkEngine {
	common := NewCommonEngine(NetworkEngineType, queueCapacity)

	network := &NetworkEngine{
		CommonEngine:        common,
		ComplexityInterface: NewNetworkInterface(ComplexityAdvanced), // Default to advanced complexity
		BandwidthMbps:       1000, // 1 Gbps
		BaseLatencyMs:       1.0,  // 1ms LAN latency
		MaxConnections:      10000,
		Protocol:            "TCP",
		GeographicDistance:  0.1,  // 100m LAN
		NetworkType:         "LAN",
		ActiveTransmissions: make(map[string]*NetworkTransmission),
		TransmissionHistory: make([]TransmissionEvent, 0, 10000),
	}
	
	// Initialize bandwidth state
	network.BandwidthState.CurrentBandwidthMbps = 0
	network.BandwidthState.BandwidthUtilization = 0.0
	network.BandwidthState.CongestionFactor = 1.0
	network.BandwidthState.PacketLossProbability = 0.0
	network.BandwidthState.CongestionLatency = 0
	network.BandwidthState.LastBandwidthUpdate = 0
	
	// Initialize connection state
	network.ConnectionState.ActiveConnections = 0
	network.ConnectionState.ConnectionUtilization = 0.0
	network.ConnectionState.ConnectionPool = make([]string, 0, 1000)
	network.ConnectionState.KeepAliveConnections = 0
	network.ConnectionState.ConnectionEstablishmentCost = 1500 * time.Microsecond // 1.5ms for TCP handshake
	network.ConnectionState.LastConnectionUpdate = 0
	
	// Initialize geographic state (physics-based)
	network.GeographicState.SpeedOfLightMps = 299792458.0 // m/s
	network.GeographicState.FiberOpticFactor = 0.67       // Refractive index of glass
	network.GeographicState.RoutingOverhead = 1.3         // 30% routing overhead
	network.calculatePhysicsLatency()
	
	// Initialize protocol state
	network.ProtocolState.HeaderOverheadBytes = 40 // TCP + IP headers
	network.ProtocolState.ProtocolEfficiency = 1.0
	network.ProtocolState.CompressionRatio = 1.0
	network.ProtocolState.MultiplexingFactor = 1.0
	network.ProtocolState.KeepAliveEnabled = true
	
	// Configure protocol-specific settings
	network.configureProtocol()
	
	// Initialize convergence models
	network.initializeConvergenceModels()
	
	return network
}

// ProcessOperation processes a single network operation with real-world network modeling
func (network *NetworkEngine) ProcessOperation(op *Operation, currentTick int64) *OperationResult {
	network.CurrentTick = currentTick

	// Calculate base transmission time from profile (IOPS and latency)
	baseTime := network.calculateBaseTransmissionTime(op)

	// Apply graph-based topology effects (if enabled)
	topologyTime := baseTime
	if network.ComplexityInterface.ShouldEnableFeature("graph_topology") {
		topologyTime = network.applyGraphTopologyEffects(baseTime, op)
	}

	// Apply geographic distance effects (if enabled - legacy mode)
	geographicTime := topologyTime
	if network.ComplexityInterface.ShouldEnableFeature("geographic_effects") && !network.ComplexityInterface.ShouldEnableFeature("graph_topology") {
		geographicTime = network.applyGeographicEffects(topologyTime)
	}

	// Apply bandwidth saturation effects (if enabled)
	bandwidthTime := geographicTime
	if network.ComplexityInterface.ShouldEnableFeature("bandwidth_limits") {
		bandwidthTime = network.applyBandwidthSaturation(geographicTime)
	}

	// Apply connection management effects (if enabled)
	connectionTime := bandwidthTime
	if network.ComplexityInterface.ShouldEnableFeature("connection_pooling") {
		connectionTime = network.applyConnectionManagement(bandwidthTime, op)
	}

	// Apply protocol overhead effects (if enabled)
	protocolTime := connectionTime
	if network.ComplexityInterface.ShouldEnableFeature("protocol_overhead") {
		protocolTime = network.applyProtocolOverhead(connectionTime, op)
	}

	// Apply jitter modeling (if enabled)
	jitterTime := protocolTime
	if network.ComplexityInterface.ShouldEnableFeature("jitter_modeling") {
		jitterTime = network.applyJitterEffects(protocolTime, op)
	}

	// Apply QoS modeling (if enabled)
	qosTime := jitterTime
	if network.ComplexityInterface.ShouldEnableFeature("qos_modeling") {
		qosTime = network.applyQoSEffects(jitterTime, op)
	}

	// Apply compression effects (if enabled)
	compressionTime := qosTime
	if network.ComplexityInterface.ShouldEnableFeature("compression_effects") {
		compressionTime = network.applyCompressionEffects(qosTime, op)
	}

	// Apply security overhead (if enabled)
	securityTime := compressionTime
	if network.ComplexityInterface.ShouldEnableFeature("security_overhead") {
		securityTime = network.applySecurityOverhead(compressionTime, op)
	}

	// Apply common performance factors (load, queue, health, variance)
	utilization := network.calculateCurrentUtilization()
	finalTime := network.ApplyCommonPerformanceFactors(securityTime, utilization)

	// Update dynamic state tracking (if enabled)
	if network.ComplexityInterface.ShouldEnableFeature("dynamic_behavior") {
		network.updateDynamicState(op, finalTime)
	}

	// Determine success based on packet loss (if enabled)
	success := true
	if network.ComplexityInterface.ShouldEnableFeature("packet_loss") {
		success = !network.checkPacketLoss()
	}

	// Ensure operations take at least 1 tick to complete
	ticksToComplete := network.DurationToTicks(finalTime)
	if ticksToComplete < 1 {
		ticksToComplete = 1
	}

	// Calculate penalty factors for routing decisions
	loadPenalty := 1.0 + (network.BandwidthState.BandwidthUtilization * 0.5) // Bandwidth utilization
	queuePenalty := 1.0 + (float64(network.ConnectionState.ActiveConnections) / 1000.0 * 0.3) // Connection load
	thermalPenalty := 1.0 // Network equipment typically doesn't have thermal issues
	contentionPenalty := network.BandwidthState.CongestionFactor // Direct congestion impact
	healthPenalty := 1.0 + (1.0 - network.GetHealth().Score) * 0.25

	// Network-specific penalties
	latencyPenalty := 1.0 + (network.GeographicState.PhysicsLatencyMs / 100.0) // Latency impact
	packetLossPenalty := 1.0 + (network.BandwidthState.PacketLossProbability * 2.0) // Packet loss is critical
	protocolPenalty := 1.0 / network.ProtocolState.ProtocolEfficiency // Protocol inefficiency

	totalPenaltyFactor := loadPenalty * queuePenalty * contentionPenalty * healthPenalty * latencyPenalty * packetLossPenalty * protocolPenalty

	// Determine performance grade
	performanceGrade := "A"
	recommendedAction := "continue"
	if totalPenaltyFactor > 3.0 {
		performanceGrade = "F"
		recommendedAction = "redirect"
	} else if totalPenaltyFactor > 2.0 {
		performanceGrade = "D"
		recommendedAction = "throttle"
	} else if totalPenaltyFactor > 1.5 {
		performanceGrade = "C"
		recommendedAction = "throttle"
	} else if totalPenaltyFactor > 1.2 {
		performanceGrade = "B"
	}

	// Create result with penalty information
	result := &OperationResult{
		OperationID:    op.ID,
		ProcessingTime: finalTime,
		CompletedTick:  currentTick + ticksToComplete,
		Success:        success,
		PenaltyInfo: &PenaltyInformation{
			EngineType:           NetworkEngineType,
			EngineID:            network.ID,
			BaseProcessingTime:   baseTime,
			ActualProcessingTime: finalTime,
			LoadPenalty:         loadPenalty,
			QueuePenalty:        queuePenalty,
			ThermalPenalty:      thermalPenalty,
			ContentionPenalty:   contentionPenalty,
			HealthPenalty:       healthPenalty,
			TotalPenaltyFactor:  totalPenaltyFactor,
			PerformanceGrade:    performanceGrade,
			RecommendedAction:   recommendedAction,
			NetworkPenalties: &NetworkPenaltyDetails{
				BandwidthUtilization: network.BandwidthState.BandwidthUtilization,
				CongestionFactor:     network.BandwidthState.CongestionFactor,
				PacketLossRate:       network.BandwidthState.PacketLossProbability,
				LatencyPenalty:       latencyPenalty,
				ProtocolEfficiency:   network.ProtocolState.ProtocolEfficiency,
			},
		},
		Metrics: map[string]interface{}{
			"base_time_ms":         float64(baseTime) / float64(time.Millisecond),
			"physics_latency_ms":   network.GeographicState.PhysicsLatencyMs,
			"bandwidth_utilization": network.BandwidthState.BandwidthUtilization,
			"congestion_factor":    network.BandwidthState.CongestionFactor,
			"packet_loss_prob":     network.BandwidthState.PacketLossProbability,
			"active_connections":   network.ConnectionState.ActiveConnections,
			"protocol_efficiency":  network.ProtocolState.ProtocolEfficiency,
			"geographic_distance":  network.GeographicDistance,
		},
	}
	
	// Update operation history for convergence
	network.AddOperationToHistory(finalTime)
	if result.Success {
		network.CompletedOps++
	} else {
		network.FailedOps++
	}
	
	return result
}

// ProcessTick processes one simulation tick
func (network *NetworkEngine) ProcessTick(currentTick int64) []OperationResult {
	network.CurrentTick = currentTick
	results := make([]OperationResult, 0)
	
	// Update bandwidth utilization state
	network.updateBandwidthUtilization()
	
	// Update connection state
	network.updateConnectionState()
	
	// Update protocol efficiency based on traffic patterns
	network.updateProtocolEfficiency()
	
	// Process queued operations if bandwidth available
	for network.BandwidthState.CurrentBandwidthMbps < network.BandwidthMbps && network.GetQueueLength() > 0 {
		queuedOp := network.DequeueOperation()
		if queuedOp != nil {
			result := network.ProcessOperation(queuedOp.Operation, currentTick)
			results = append(results, *result)
		}
	}
	
	// Update health metrics
	network.UpdateHealth()
	
	// Update dynamic behavior
	network.UpdateDynamicBehavior()
	
	return results
}

// calculateBaseTransmissionTime calculates base transmission time
func (network *NetworkEngine) calculateBaseTransmissionTime(op *Operation) time.Duration {
	// Base latency from profile
	baseLatencyMs := network.BaseLatencyMs
	
	if network.Profile != nil {
		if val, ok := network.Profile.BaselinePerformance["base_latency_ms"]; ok {
			baseLatencyMs = val
		}
	}
	
	// Calculate transmission time based on data size and bandwidth
	if op.DataSize > 0 {
		// Convert bandwidth from Mbps to bytes per millisecond
		bandwidthBytesPerMs := float64(network.BandwidthMbps) * 1000000 / 8 / 1000
		transmissionTimeMs := float64(op.DataSize) / bandwidthBytesPerMs
		baseLatencyMs += transmissionTimeMs
	}
	
	return time.Duration(baseLatencyMs * float64(time.Millisecond))
}

// calculateCurrentUtilization calculates current network utilization
func (network *NetworkEngine) calculateCurrentUtilization() float64 {
	if network.BandwidthMbps == 0 {
		return 0.0
	}
	return float64(network.BandwidthState.CurrentBandwidthMbps) / float64(network.BandwidthMbps)
}

// calculatePhysicsLatency calculates physics-based latency from distance
func (network *NetworkEngine) calculatePhysicsLatency() {
	// Physics-Based Latency Calculation (Cannot Be Violated)
	distanceMeters := network.GeographicDistance * 1000 // Convert km to meters
	
	// Speed of light in fiber optic cable
	effectiveSpeed := network.GeographicState.SpeedOfLightMps * network.GeographicState.FiberOpticFactor
	
	// Theoretical minimum latency (one way)
	theoreticalLatencySeconds := distanceMeters / effectiveSpeed
	
	// Apply routing overhead (non-direct paths)
	actualLatencySeconds := theoreticalLatencySeconds * network.GeographicState.RoutingOverhead
	
	// Convert to milliseconds
	network.GeographicState.PhysicsLatencyMs = actualLatencySeconds * 1000
	network.GeographicState.PropagationDelay = time.Duration(actualLatencySeconds * float64(time.Second))
}

// applyGraphTopologyEffects applies graph-based distance modeling (NEW FEATURE)
func (network *NetworkEngine) applyGraphTopologyEffects(baseTime time.Duration, op *Operation) time.Duration {
	// If no topology is configured, fall back to geographic effects
	if network.NetworkTopology == nil || network.CurrentNodeID == "" {
		return network.applyGeographicEffects(baseTime)
	}

	// For now, use a simple approach - in a real implementation, this would:
	// 1. Determine target node from operation metadata
	// 2. Find shortest path using Dijkstra's algorithm
	// 3. Sum latencies along the path
	// 4. Apply bandwidth constraints of bottleneck edge

	// Placeholder: Use geographic effects as baseline
	// TODO: Implement full graph-based routing when operation metadata includes target node
	return network.applyGeographicEffects(baseTime)
}

// applyJitterEffects applies network jitter modeling
func (network *NetworkEngine) applyJitterEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Network jitter is typically 1-5% of base latency
	jitterFactor := 1.0 + (rand.Float64()-0.5)*0.05 // ±2.5% jitter
	return time.Duration(float64(baseTime) * jitterFactor)
}

// applyQoSEffects applies Quality of Service prioritization
func (network *NetworkEngine) applyQoSEffects(baseTime time.Duration, op *Operation) time.Duration {
	// QoS prioritization based on operation type
	// High priority operations get better treatment during congestion
	utilization := network.calculateCurrentUtilization()

	if utilization < 0.7 {
		return baseTime // No QoS effects under normal load
	}

	// Apply QoS based on operation type
	qosFactor := 1.0
	switch op.Type {
	case "network_high_priority":
		qosFactor = 0.8 // 20% better performance
	case "network_low_priority":
		qosFactor = 1.3 // 30% worse performance
	default:
		qosFactor = 1.0 // Normal priority
	}

	return time.Duration(float64(baseTime) * qosFactor)
}

// applyCompressionEffects applies data compression impact
func (network *NetworkEngine) applyCompressionEffects(baseTime time.Duration, op *Operation) time.Duration {
	if op.DataSize <= 0 {
		return baseTime
	}

	// Compression reduces data size but adds CPU overhead
	compressionRatio := 0.7 // 30% compression typical
	cpuOverhead := 1.05     // 5% CPU overhead

	// Reduce transmission time due to smaller data
	transmissionReduction := compressionRatio

	// Add CPU processing time
	processingOverhead := cpuOverhead

	return time.Duration(float64(baseTime) * transmissionReduction * processingOverhead)
}

// applySecurityOverhead applies TLS/encryption processing costs
func (network *NetworkEngine) applySecurityOverhead(baseTime time.Duration, op *Operation) time.Duration {
	// TLS handshake and encryption overhead
	if op.DataSize <= 0 {
		return baseTime
	}

	// Security overhead is typically 5-15% for established connections
	securityOverhead := 1.08 // 8% overhead typical

	return time.Duration(float64(baseTime) * securityOverhead)
}

// configureProtocol configures protocol-specific settings
func (network *NetworkEngine) configureProtocol() {
	switch network.Protocol {
	case "HTTP/1.1":
		network.ProtocolState.HeaderOverheadBytes = 200 // HTTP headers
		network.ProtocolState.ProtocolEfficiency = 0.8  // Connection per request overhead
		network.ProtocolState.MultiplexingFactor = 1.0  // No multiplexing
		network.ProtocolState.KeepAliveEnabled = false
		
	case "HTTP/2":
		network.ProtocolState.HeaderOverheadBytes = 50  // Compressed headers
		network.ProtocolState.ProtocolEfficiency = 1.2  // Multiplexing efficiency
		network.ProtocolState.MultiplexingFactor = 4.0  // 4x multiplexing
		network.ProtocolState.KeepAliveEnabled = true
		
	case "gRPC":
		network.ProtocolState.HeaderOverheadBytes = 30  // Minimal headers
		network.ProtocolState.ProtocolEfficiency = 1.3  // Binary protocol efficiency
		network.ProtocolState.MultiplexingFactor = 8.0  // High multiplexing
		network.ProtocolState.KeepAliveEnabled = true
		
	case "UDP":
		network.ProtocolState.HeaderOverheadBytes = 8   // UDP header only
		network.ProtocolState.ProtocolEfficiency = 1.1  // No connection overhead
		network.ProtocolState.MultiplexingFactor = 1.0  // No multiplexing
		network.ProtocolState.KeepAliveEnabled = false
		
	default: // TCP
		network.ProtocolState.HeaderOverheadBytes = 40  // TCP + IP
		network.ProtocolState.ProtocolEfficiency = 1.0  // Baseline
		network.ProtocolState.MultiplexingFactor = 1.0  // No multiplexing
		network.ProtocolState.KeepAliveEnabled = true
	}
}

// checkPacketLoss checks if packet loss occurred
func (network *NetworkEngine) checkPacketLoss() bool {
	return rand.Float64() < network.BandwidthState.PacketLossProbability
}

// applyGeographicEffects applies physics-based distance effects (NOT random)
func (network *NetworkEngine) applyGeographicEffects(baseTime time.Duration) time.Duration {
	// Add physics-based propagation delay (cannot be less than speed of light allows)
	return baseTime + network.GeographicState.PropagationDelay
}

// applyBandwidthSaturation applies bandwidth saturation effects (congestion-based performance)
func (network *NetworkEngine) applyBandwidthSaturation(baseTime time.Duration) time.Duration {
	utilization := network.BandwidthState.BandwidthUtilization

	// Gigabit Ethernet Congestion Behavior (Based on Network Equipment Documentation)
	var congestionFactor float64
	switch {
	case utilization < 0.70:
		congestionFactor = 1.0 // Optimal performance (documented range)
	case utilization < 0.85:
		// Linear increase from 1.0 to 1.5 (mild congestion)
		congestionFactor = 1.0 + (utilization-0.70)*3.33
	case utilization < 0.95:
		// Linear increase from 1.5 to 3.0 (significant congestion)
		congestionFactor = 1.5 + (utilization-0.85)*15.0
	default:
		// Exponential increase above 95% (severe congestion + packet loss)
		excess := utilization - 0.95
		congestionFactor = 3.0 + excess*140.0 // Up to 10x slower

		// Packet loss probability increases with congestion
		network.BandwidthState.PacketLossProbability = math.Min(excess*100.0, 0.05) // Up to 5% loss
	}

	network.BandwidthState.CongestionFactor = congestionFactor

	// Apply congestion latency
	if utilization > 0.70 {
		congestionLatencyMs := (utilization - 0.70) * 10.0 // Up to 3ms additional latency
		network.BandwidthState.CongestionLatency = time.Duration(congestionLatencyMs * float64(time.Millisecond))
	} else {
		network.BandwidthState.CongestionLatency = 0
		network.BandwidthState.PacketLossProbability = 0.0
	}

	return time.Duration(float64(baseTime)*congestionFactor) + network.BandwidthState.CongestionLatency
}

// applyConnectionManagement applies TCP connection lifecycle costs
func (network *NetworkEngine) applyConnectionManagement(baseTime time.Duration, op *Operation) time.Duration {
	// Check if we can reuse an existing connection
	connectionReused := false

	if network.ProtocolState.KeepAliveEnabled && len(network.ConnectionState.ConnectionPool) > 0 {
		// 80% chance of connection reuse with keep-alive
		if rand.Float64() < 0.8 {
			connectionReused = true
		}
	}

	if !connectionReused {
		// New connection required - add establishment cost
		establishmentCost := network.ConnectionState.ConnectionEstablishmentCost

		// TCP 3-way handshake cost (1.5x round-trip time)
		rtt := network.GeographicState.PropagationDelay * 2 // Round trip
		tcpHandshakeCost := time.Duration(float64(rtt) * 1.5)

		// Use the larger of configured cost or calculated handshake cost
		if tcpHandshakeCost > establishmentCost {
			establishmentCost = tcpHandshakeCost
		}

		// Add connection to pool if keep-alive enabled
		if network.ProtocolState.KeepAliveEnabled {
			network.ConnectionState.ConnectionPool = append(network.ConnectionState.ConnectionPool, op.ID)
			// Limit pool size
			if len(network.ConnectionState.ConnectionPool) > 100 {
				network.ConnectionState.ConnectionPool = network.ConnectionState.ConnectionPool[1:]
			}
		}

		return baseTime + establishmentCost
	}

	// Connection reused - minimal overhead
	return baseTime + time.Microsecond*10 // 10μs reuse overhead
}

// applyProtocolOverhead applies protocol-specific overhead effects
func (network *NetworkEngine) applyProtocolOverhead(baseTime time.Duration, op *Operation) time.Duration {
	// Calculate protocol overhead based on message size
	dataSize := float64(op.DataSize)
	headerSize := float64(network.ProtocolState.HeaderOverheadBytes)

	// Protocol efficiency factor
	efficiencyFactor := network.ProtocolState.ProtocolEfficiency

	// For small messages, header overhead dominates
	if dataSize < headerSize*10 {
		// High overhead ratio for small messages
		overheadRatio := headerSize / (dataSize + headerSize)
		efficiencyFactor *= (1.0 - overheadRatio*0.5) // Up to 50% efficiency loss
	}

	// Apply multiplexing benefits for HTTP/2 and gRPC
	if network.ProtocolState.MultiplexingFactor > 1.0 && dataSize > 1024 {
		// Multiplexing reduces per-message overhead for larger messages
		multiplexingBenefit := math.Min(network.ProtocolState.MultiplexingFactor/4.0, 0.3) // Up to 30% improvement
		efficiencyFactor *= (1.0 + multiplexingBenefit)
	}

	return time.Duration(float64(baseTime) / efficiencyFactor)
}

// updateBandwidthUtilization updates bandwidth utilization state
func (network *NetworkEngine) updateBandwidthUtilization() {
	// Calculate current bandwidth usage from active transmissions
	currentBandwidth := 0
	for _, transmission := range network.ActiveTransmissions {
		// Estimate bandwidth per transmission (simplified)
		transmissionBandwidth := int(float64(transmission.DataSizeBytes) / 1000.0) // Rough estimate
		currentBandwidth += transmissionBandwidth
	}

	network.BandwidthState.CurrentBandwidthMbps = currentBandwidth
	network.BandwidthState.BandwidthUtilization = network.calculateCurrentUtilization()
	network.BandwidthState.LastBandwidthUpdate = network.CurrentTick
}

// updateConnectionState updates connection management state
func (network *NetworkEngine) updateConnectionState() {
	// Count active connections
	network.ConnectionState.ActiveConnections = len(network.ActiveTransmissions)

	// Calculate connection utilization
	if network.MaxConnections > 0 {
		network.ConnectionState.ConnectionUtilization = float64(network.ConnectionState.ActiveConnections) / float64(network.MaxConnections)
	}

	// Update keep-alive connections
	network.ConnectionState.KeepAliveConnections = len(network.ConnectionState.ConnectionPool)

	// Clean up old connections from pool (simulate timeout)
	if len(network.ConnectionState.ConnectionPool) > 50 {
		// Remove oldest 10 connections
		network.ConnectionState.ConnectionPool = network.ConnectionState.ConnectionPool[10:]
	}

	network.ConnectionState.LastConnectionUpdate = network.CurrentTick
}

// updateProtocolEfficiency updates protocol efficiency based on traffic patterns
func (network *NetworkEngine) updateProtocolEfficiency() {
	// Analyze recent transmission patterns
	if len(network.TransmissionHistory) < 10 {
		return
	}

	recentTransmissions := network.TransmissionHistory[len(network.TransmissionHistory)-10:]

	// Calculate average message size
	totalSize := int64(0)
	for _, transmission := range recentTransmissions {
		totalSize += transmission.DataSizeBytes
	}
	avgSize := float64(totalSize) / 10.0

	// Adjust protocol efficiency based on message size patterns
	if avgSize < 1024 { // Small messages
		// Reduce efficiency for small message overhead
		network.ProtocolState.ProtocolEfficiency *= 0.95
	} else if avgSize > 64*1024 { // Large messages
		// Improve efficiency for large message amortization
		network.ProtocolState.ProtocolEfficiency *= 1.02
	}

	// Keep efficiency within reasonable bounds
	network.ProtocolState.ProtocolEfficiency = math.Max(0.5, math.Min(2.0, network.ProtocolState.ProtocolEfficiency))
}

// updateDynamicState updates all dynamic state after processing an operation
func (network *NetworkEngine) updateDynamicState(op *Operation, processingTime time.Duration) {
	// Add to active transmissions
	transmission := &NetworkTransmission{
		OperationID:      op.ID,
		Type:             op.Type,
		DataSizeBytes:    op.DataSize,
		StartTick:        network.CurrentTick,
		EstimatedTicks:   network.DurationToTicks(processingTime),
		Protocol:         network.Protocol,
		ConnectionReused: len(network.ConnectionState.ConnectionPool) > 0,
	}
	network.ActiveTransmissions[op.ID] = transmission

	// Add to transmission history
	event := TransmissionEvent{
		OperationID:   op.ID,
		Type:          op.Type,
		DataSizeBytes: op.DataSize,
		Duration:      processingTime,
		CompletedAt:   network.CurrentTick,
		PacketLoss:    network.checkPacketLoss(),
		Congestion:    network.BandwidthState.CongestionFactor > 1.5,
	}
	network.TransmissionHistory = append(network.TransmissionHistory, event)

	// Keep history limited
	if len(network.TransmissionHistory) > 10000 {
		network.TransmissionHistory = network.TransmissionHistory[len(network.TransmissionHistory)-10000:]
	}

	// Update utilization in health metrics
	network.Health.Utilization = network.calculateCurrentUtilization()

	// Update convergence state
	network.ConvergenceState.OperationCount++
	network.ConvergenceState.DataProcessed += op.DataSize
}

// initializeConvergenceModels initializes statistical convergence models
func (network *NetworkEngine) initializeConvergenceModels() {
	network.ConvergenceState.Models["bandwidth_behavior"] = &StatisticalModel{
		Name:             "bandwidth_behavior",
		ConvergencePoint: 1.0, // No congestion under normal load
		BaseVariance:     0.02,
		MinOperations:    5000,
		CurrentValue:     1.0,
		IsConverged:      false,
	}

	network.ConvergenceState.Models["connection_behavior"] = &StatisticalModel{
		Name:             "connection_behavior",
		ConvergencePoint: 0.8, // 80% connection reuse efficiency
		BaseVariance:     0.05,
		MinOperations:    1000,
		CurrentValue:     0.5, // Start moderate
		IsConverged:      false,
	}

	network.ConvergenceState.Models["protocol_behavior"] = &StatisticalModel{
		Name:             "protocol_behavior",
		ConvergencePoint: 1.0, // Optimal protocol efficiency
		BaseVariance:     0.03,
		MinOperations:    2000,
		CurrentValue:     1.0,
		IsConverged:      false,
	}
}

// updateConvergenceModels updates statistical convergence models based on recent operations
func (network *NetworkEngine) updateConvergenceModels() {
	// Update bandwidth behavior model
	if bandwidthModel, exists := network.ConvergenceState.Models["bandwidth_behavior"]; exists {
		// Update based on current congestion factor
		bandwidthModel.CurrentValue = network.BandwidthState.CongestionFactor

		// Check convergence
		if network.CompletedOps >= bandwidthModel.MinOperations {
			variance := math.Abs(bandwidthModel.CurrentValue - bandwidthModel.ConvergencePoint)
			bandwidthModel.IsConverged = variance <= bandwidthModel.BaseVariance
		}
	}

	// Update connection behavior model
	if connectionModel, exists := network.ConvergenceState.Models["connection_behavior"]; exists {
		// Update based on connection reuse efficiency
		if network.ConnectionState.ActiveConnections > 0 {
			reuseRatio := float64(network.ConnectionState.KeepAliveConnections) / float64(network.ConnectionState.ActiveConnections)
			connectionModel.CurrentValue = reuseRatio
		}

		// Check convergence
		if network.CompletedOps >= connectionModel.MinOperations {
			variance := math.Abs(connectionModel.CurrentValue - connectionModel.ConvergencePoint)
			connectionModel.IsConverged = variance <= connectionModel.BaseVariance
		}
	}

	// Update protocol behavior model
	if protocolModel, exists := network.ConvergenceState.Models["protocol_behavior"]; exists {
		// Update based on protocol efficiency
		protocolModel.CurrentValue = network.ProtocolState.ProtocolEfficiency

		// Check convergence
		if network.CompletedOps >= protocolModel.MinOperations {
			variance := math.Abs(protocolModel.CurrentValue - protocolModel.ConvergencePoint)
			protocolModel.IsConverged = variance <= protocolModel.BaseVariance
		}
	}
}

// LoadProfile loads a network profile into the engine (BaseEngine interface)
func (network *NetworkEngine) LoadProfile(profile *EngineProfile) error {
	if profile == nil {
		return fmt.Errorf("profile cannot be nil")
	}

	// Store the profile
	network.Profile = profile

	// Load baseline performance settings
	if bandwidth, ok := profile.BaselinePerformance["bandwidth_mbps"]; ok {
		network.BandwidthMbps = int(bandwidth)
	}

	if latency, ok := profile.BaselinePerformance["base_latency_ms"]; ok {
		network.BaseLatencyMs = latency
	}

	if maxConn, ok := profile.BaselinePerformance["max_connections"]; ok {
		network.MaxConnections = int(maxConn)
	}

	// Load technology specs
	if protocol, ok := profile.TechnologySpecs["protocol"]; ok {
		if protocolStr, ok := protocol.(string); ok {
			network.Protocol = protocolStr
		}
	}

	if networkType, ok := profile.TechnologySpecs["network_type"]; ok {
		if networkTypeStr, ok := networkType.(string); ok {
			network.NetworkType = networkTypeStr
		}
	}

	// Load engine-specific settings
	if engineSpecific, ok := profile.EngineSpecific["geographic"]; ok {
		if geographic, ok := engineSpecific.(map[string]interface{}); ok {
			if distance, ok := geographic["distance_km"]; ok {
				if distanceFloat, ok := distance.(float64); ok {
					network.GeographicDistance = distanceFloat
				}
			}
		}
	}

	// Configure protocol-specific settings
	network.configureProtocol()

	// Initialize convergence models
	network.initializeConvergenceModels()

	return nil
}

// GetProfile returns the current profile (BaseEngine interface)
func (network *NetworkEngine) GetProfile() *EngineProfile {
	return network.Profile
}

// SetComplexityLevel sets the network simulation complexity level (BaseEngine interface)
func (network *NetworkEngine) SetComplexityLevel(level int) error {
	if err := ValidateNetworkComplexityLevel(NetworkComplexityLevel(level)); err != nil {
		return err
	}
	return network.ComplexityInterface.SetComplexityLevel(NetworkComplexityLevel(level))
}

// GetComplexityLevel returns the current complexity level (BaseEngine interface)
func (network *NetworkEngine) GetComplexityLevel() int {
	return int(network.ComplexityInterface.ComplexityLevel)
}

// GetDynamicState returns the current dynamic state with network-specific data (BaseEngine interface)
func (network *NetworkEngine) GetDynamicState() *DynamicState {
	return &DynamicState{
		CurrentUtilization:  network.calculateCurrentUtilization(),
		PerformanceFactor:   network.BandwidthState.CongestionFactor,
		ConvergenceProgress: network.CommonEngine.calculateConvergenceProgress(),
		HardwareSpecific: map[string]interface{}{
			"bandwidth_utilization":    network.BandwidthState.BandwidthUtilization,
			"active_connections":       network.ConnectionState.ActiveConnections,
			"connection_utilization":   network.ConnectionState.ConnectionUtilization,
			"packet_loss_probability":  network.BandwidthState.PacketLossProbability,
			"congestion_factor":        network.BandwidthState.CongestionFactor,
			"protocol_efficiency":      network.ProtocolState.ProtocolEfficiency,
			"physics_latency_ms":       network.GeographicState.PhysicsLatencyMs,
			"active_transmissions":     len(network.ActiveTransmissions),
			"protocol":                 network.Protocol,
			"network_type":             network.NetworkType,
			"bandwidth_mbps":           network.BandwidthMbps,
			"base_latency_ms":          network.BaseLatencyMs,
		},
		LastUpdated: network.CurrentTick,
	}
}

// UpdateDynamicBehavior updates network convergence models and dynamic state (BaseEngine interface)
func (network *NetworkEngine) UpdateDynamicBehavior() {
	// Update convergence models based on recent operations
	if network.ComplexityInterface.ShouldEnableFeature("statistical_modeling") {
		network.updateConvergenceModels()
	}

	// Update bandwidth utilization patterns
	if network.ComplexityInterface.ShouldEnableFeature("dynamic_behavior") {
		network.updateBandwidthUtilization()
		network.updateConnectionState()
		network.updateProtocolEfficiency()
	}

	// Update health metrics
	network.UpdateHealth()
}

// GetConvergenceMetrics returns network convergence metrics (BaseEngine interface)
func (network *NetworkEngine) GetConvergenceMetrics() *ConvergenceMetrics {
	// Calculate overall convergence progress
	totalModels := len(network.ConvergenceState.Models)
	convergedModels := 0
	totalVariance := 0.0

	for _, model := range network.ConvergenceState.Models {
		if model.IsConverged {
			convergedModels++
		}
		variance := math.Abs(model.CurrentValue - model.ConvergencePoint)
		totalVariance += variance
	}

	// Calculate average variance
	avgVariance := 0.0
	if totalModels > 0 {
		avgVariance = totalVariance / float64(totalModels)
	}

	// Create convergence factors map
	convergenceFactors := make(map[string]float64)
	for name, model := range network.ConvergenceState.Models {
		convergenceFactors[name] = model.CurrentValue
	}

	return &ConvergenceMetrics{
		OperationCount:     network.CompletedOps,
		ConvergencePoint:   1.0, // Target convergence point
		CurrentVariance:    avgVariance,
		IsConverged:        convergedModels == totalModels && totalModels > 0,
		TimeToConvergence:  network.CurrentTick,
		ConvergenceFactors: convergenceFactors,
	}
}

// Reset resets the network engine to initial state (BaseEngine interface)
func (network *NetworkEngine) Reset() {
	// Reset bandwidth state
	network.BandwidthState.CurrentBandwidthMbps = 0
	network.BandwidthState.BandwidthUtilization = 0.0
	network.BandwidthState.CongestionFactor = 1.0
	network.BandwidthState.PacketLossProbability = 0.0
	network.BandwidthState.CongestionLatency = 0
	network.BandwidthState.LastBandwidthUpdate = 0

	// Reset connection state
	network.ConnectionState.ActiveConnections = 0
	network.ConnectionState.ConnectionUtilization = 0.0
	network.ConnectionState.ConnectionPool = make([]string, 0, 1000)
	network.ConnectionState.KeepAliveConnections = 0
	network.ConnectionState.LastConnectionUpdate = 0

	// Reset protocol state
	network.ProtocolState.ProtocolEfficiency = 1.0
	network.ProtocolState.CompressionRatio = 1.0
	network.ProtocolState.MultiplexingFactor = 1.0

	// Clear active transmissions and history
	network.ActiveTransmissions = make(map[string]*NetworkTransmission)
	network.TransmissionHistory = make([]TransmissionEvent, 0, 10000)

	// Reset convergence models
	network.initializeConvergenceModels()

	// Reset common engine state
	network.CommonEngine.Reset()
}

// GetCurrentState returns the current network engine state (BaseEngine interface)
func (network *NetworkEngine) GetCurrentState() map[string]interface{} {
	return map[string]interface{}{
		"engine_type":              network.GetEngineType().String(),
		"engine_id":                network.GetEngineID(),
		"complexity_level":         network.GetComplexityLevel(),
		"bandwidth_mbps":           network.BandwidthMbps,
		"base_latency_ms":          network.BaseLatencyMs,
		"max_connections":          network.MaxConnections,
		"protocol":                 network.Protocol,
		"network_type":             network.NetworkType,
		"geographic_distance_km":   network.GeographicDistance,

		// Dynamic state
		"bandwidth_utilization":    network.BandwidthState.BandwidthUtilization,
		"active_connections":       network.ConnectionState.ActiveConnections,
		"connection_utilization":   network.ConnectionState.ConnectionUtilization,
		"packet_loss_probability":  network.BandwidthState.PacketLossProbability,
		"congestion_factor":        network.BandwidthState.CongestionFactor,
		"protocol_efficiency":      network.ProtocolState.ProtocolEfficiency,
		"physics_latency_ms":       network.GeographicState.PhysicsLatencyMs,
		"active_transmissions":     len(network.ActiveTransmissions),

		// Queue state
		"queue_length":             network.GetQueueLength(),
		"queue_capacity":           network.GetQueueCapacity(),
		"queue_utilization":        network.Health.QueueUtilization,

		// Health metrics
		"health_score":             network.Health.Score,
		"utilization":              network.Health.Utilization,
		"error_rate":               network.Health.ErrorRate,
		"average_latency":          network.Health.AverageLatency,
		"throughput_ops":           network.Health.ThroughputOps,

		// Convergence state
		"convergence_progress":     network.CommonEngine.calculateConvergenceProgress(),
		"operations_processed":     network.CompletedOps,
		"current_tick":             network.CurrentTick,
		"last_updated":             network.Health.LastUpdated,
	}
}
