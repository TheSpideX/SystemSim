package components

import (
	"math/rand"
	"sync"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// ProbabilityEngine handles probability-based routing decisions
type ProbabilityEngine struct {
	random *rand.Rand
	mutex  sync.Mutex
}

// NewProbabilityEngine creates a new probability engine
func NewProbabilityEngine() *ProbabilityEngine {
	return &ProbabilityEngine{
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// MakeDecision makes a probability-based routing decision
func (pe *ProbabilityEngine) MakeDecision(config *ProbabilityConfig, op *engines.Operation) string {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()

	randomValue := pe.random.Float64()

	switch op.Type {
	case "cache_lookup":
		if randomValue < config.CacheHitRate {
			return "cache_hit"
		}
		return "cache_miss"

	case "database_lookup", "storage_read":
		if randomValue < config.SuccessRate {
			return "data_found"
		}
		return "data_not_found"

	case "network_request", "api_call":
		if randomValue < config.SuccessRate {
			return "request_success"
		}
		return "request_timeout"

	case "payment_process":
		if randomValue < config.SuccessRate {
			return "payment_success"
		}
		return "payment_failure"

	case "authentication":
		if randomValue < config.SuccessRate {
			return "authenticated"
		}
		return "not_authenticated"

	default:
		// Generic success/failure based on success rate
		if randomValue < config.SuccessRate {
			return "success"
		}
		return "failure"
	}
}

// StateMonitor monitors system state for dynamic routing decisions
type StateMonitor struct {
	componentID   string
	currentState  *SystemState
	mutex         sync.RWMutex
	updateTicker  *time.Ticker
	stopChan      chan struct{}
}

// SystemState represents the current state of the system
type SystemState struct {
	SystemLoad      float64 `json:"system_load"`       // 0.0 to 1.0
	MemoryUsage     float64 `json:"memory_usage"`      // 0.0 to 1.0
	StorageLatency  float64 `json:"storage_latency"`   // milliseconds
	NetworkLatency  float64 `json:"network_latency"`   // milliseconds
	IsPeakHours     bool    `json:"is_peak_hours"`
	LastUpdate      time.Time `json:"last_update"`
}

// NewStateMonitor creates a new state monitor
func NewStateMonitor(componentID string) *StateMonitor {
	sm := &StateMonitor{
		componentID: componentID,
		currentState: &SystemState{
			SystemLoad:     0.5,
			MemoryUsage:    0.4,
			StorageLatency: 20.0,
			NetworkLatency: 15.0,
			IsPeakHours:    false,
			LastUpdate:     time.Now(),
		},
		updateTicker: time.NewTicker(5 * time.Second), // Update every 5 seconds
		stopChan:     make(chan struct{}),
	}

	// Start monitoring goroutine
	go sm.monitorState()

	return sm
}

// GetCurrentState returns the current system state
func (sm *StateMonitor) GetCurrentState() *SystemState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Return a copy to prevent external modification
	return &SystemState{
		SystemLoad:     sm.currentState.SystemLoad,
		MemoryUsage:    sm.currentState.MemoryUsage,
		StorageLatency: sm.currentState.StorageLatency,
		NetworkLatency: sm.currentState.NetworkLatency,
		IsPeakHours:    sm.currentState.IsPeakHours,
		LastUpdate:     sm.currentState.LastUpdate,
	}
}

// monitorState continuously monitors and updates system state
func (sm *StateMonitor) monitorState() {
	for {
		select {
		case <-sm.updateTicker.C:
			sm.updateState()
		case <-sm.stopChan:
			return
		}
	}
}

// updateState updates the current system state with realistic variations
func (sm *StateMonitor) updateState() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	
	// Simulate realistic state changes
	sm.currentState.SystemLoad = sm.simulateSystemLoad(now)
	sm.currentState.MemoryUsage = sm.simulateMemoryUsage()
	sm.currentState.StorageLatency = sm.simulateStorageLatency()
	sm.currentState.NetworkLatency = sm.simulateNetworkLatency()
	sm.currentState.IsPeakHours = sm.isPeakHours(now)
	sm.currentState.LastUpdate = now
}

// simulateSystemLoad simulates realistic system load variations
func (sm *StateMonitor) simulateSystemLoad(now time.Time) float64 {
	// Base load varies by time of day
	hour := now.Hour()
	var baseLoad float64

	if hour >= 9 && hour <= 17 {
		// Business hours - higher load
		baseLoad = 0.7
	} else if hour >= 18 && hour <= 22 {
		// Evening - moderate load
		baseLoad = 0.5
	} else {
		// Night/early morning - lower load
		baseLoad = 0.2
	}

	// Add random variation (±20%)
	variation := (rand.Float64() - 0.5) * 0.4
	load := baseLoad + variation

	// Clamp to valid range
	if load < 0.0 {
		load = 0.0
	}
	if load > 1.0 {
		load = 1.0
	}

	return load
}

// simulateMemoryUsage simulates memory usage patterns
func (sm *StateMonitor) simulateMemoryUsage() float64 {
	// Memory usage tends to be more stable but can spike
	currentUsage := sm.currentState.MemoryUsage
	
	// Small random changes (±5%)
	change := (rand.Float64() - 0.5) * 0.1
	newUsage := currentUsage + change

	// Occasional spikes (5% chance of significant increase)
	if rand.Float64() < 0.05 {
		newUsage += rand.Float64() * 0.3
	}

	// Clamp to valid range
	if newUsage < 0.1 {
		newUsage = 0.1
	}
	if newUsage > 0.95 {
		newUsage = 0.95
	}

	return newUsage
}

// simulateStorageLatency simulates storage latency variations
func (sm *StateMonitor) simulateStorageLatency() float64 {
	// Base latency with random variation
	baseLatency := 20.0 // 20ms base
	variation := (rand.Float64() - 0.5) * 30.0 // ±15ms variation
	
	latency := baseLatency + variation

	// Occasional slow operations (10% chance)
	if rand.Float64() < 0.1 {
		latency += rand.Float64() * 100.0 // Add up to 100ms
	}

	if latency < 1.0 {
		latency = 1.0
	}

	return latency
}

// simulateNetworkLatency simulates network latency variations
func (sm *StateMonitor) simulateNetworkLatency() float64 {
	// Base latency with random variation
	baseLatency := 15.0 // 15ms base
	variation := (rand.Float64() - 0.5) * 20.0 // ±10ms variation
	
	latency := baseLatency + variation

	// Occasional network congestion (8% chance)
	if rand.Float64() < 0.08 {
		latency += rand.Float64() * 80.0 // Add up to 80ms
	}

	if latency < 1.0 {
		latency = 1.0
	}

	return latency
}

// isPeakHours determines if current time is during peak hours
func (sm *StateMonitor) isPeakHours(now time.Time) bool {
	hour := now.Hour()
	// Peak hours: 9 AM - 5 PM and 7 PM - 10 PM
	return (hour >= 9 && hour <= 17) || (hour >= 19 && hour <= 22)
}

// Stop stops the state monitor
func (sm *StateMonitor) Stop() {
	close(sm.stopChan)
	sm.updateTicker.Stop()
}

// CustomRoutingFunc defines the signature for custom routing functions
type CustomRoutingFunc func(node *DecisionNode, op *engines.Operation, graph *DecisionGraph) (string, error)

// Example custom routing functions

// LoadBalancedRouting routes based on current load across multiple destinations
func LoadBalancedRouting(node *DecisionNode, op *engines.Operation, graph *DecisionGraph) (string, error) {
	// This would implement load-balanced routing logic
	// For now, return a simple round-robin approach
	destinations := []string{"destination_1", "destination_2", "destination_3"}
	index := int(time.Now().UnixNano()) % len(destinations)
	return destinations[index], nil
}

// PriorityBasedRouting routes based on operation priority
func PriorityBasedRouting(node *DecisionNode, op *engines.Operation, graph *DecisionGraph) (string, error) {
	if op.Priority >= 8 {
		return "high_priority_queue", nil
	} else if op.Priority >= 5 {
		return "medium_priority_queue", nil
	}
	return "low_priority_queue", nil
}

// ContentBasedRouting routes based on operation content/type
func ContentBasedRouting(node *DecisionNode, op *engines.Operation, graph *DecisionGraph) (string, error) {
	switch op.Type {
	case "read", "select", "get":
		return "read_optimized_engine", nil
	case "write", "insert", "update", "delete":
		return "write_optimized_engine", nil
	case "compute", "calculate", "process":
		return "compute_optimized_engine", nil
	default:
		return "general_purpose_engine", nil
	}
}
