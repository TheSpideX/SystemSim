package components

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// EngineOutputQueue handles ALL routing decisions (both internal and external)
// It's the single point for routing with dynamic graph lookup
type EngineOutputQueue struct {
	// Identity
	EngineType  engines.EngineType `json:"engine_type"`
	ComponentID string             `json:"component_id"`
	InstanceID  string             `json:"instance_id"`

	// Reference to Component LB for graph lookup
	ComponentLB ComponentLoadBalancerInterface `json:"-"`

	// Reference to Global Registry for system-level routing
	GlobalRegistry GlobalRegistryInterface `json:"-"`

	// Communication channels
	InputChannel  chan *EngineOutputRequest `json:"-"`
	OutputChannel chan *EngineOutputRequest `json:"-"`

	// Timeout management
	TimeoutManager *RequestTimeoutManager `json:"-"`

	// Context for lifecycle management
	ctx    context.Context    `json:"-"`
	cancel context.CancelFunc `json:"-"`

	// Metrics
	RoutingMetrics *EngineOutputMetrics `json:"-"`
}

// EngineOutputRequest represents a request with engine result for routing
type EngineOutputRequest struct {
	Request      *Request                   `json:"request"`
	EngineResult *engines.OperationResult   `json:"engine_result"`
	Timestamp    time.Time                  `json:"timestamp"`
}

// EngineOutputMetrics tracks routing performance
type EngineOutputMetrics struct {
	TotalRoutingDecisions   int64         `json:"total_routing_decisions"`
	InternalRoutingCount    int64         `json:"internal_routing_count"`
	ExternalRoutingCount    int64         `json:"external_routing_count"`
	RoutingErrors           int64         `json:"routing_errors"`
	AverageRoutingLatency   time.Duration `json:"average_routing_latency"`
	TimeoutCount            int64         `json:"timeout_count"`
}

// NewEngineOutputQueue creates a new engine output queue
func NewEngineOutputQueue(engineType engines.EngineType, componentID, instanceID string, 
	componentLB ComponentLoadBalancerInterface, globalRegistry GlobalRegistryInterface) *EngineOutputQueue {
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &EngineOutputQueue{
		EngineType:     engineType,
		ComponentID:    componentID,
		InstanceID:     instanceID,
		ComponentLB:    componentLB,
		GlobalRegistry: globalRegistry,
		InputChannel:   make(chan *EngineOutputRequest, 1000),
		OutputChannel:  make(chan *EngineOutputRequest, 1000),
		TimeoutManager: NewRequestTimeoutManager(30 * time.Second),
		ctx:            ctx,
		cancel:         cancel,
		RoutingMetrics: &EngineOutputMetrics{},
	}
}

// Start starts the engine output queue processing
func (eoq *EngineOutputQueue) Start() {
	log.Printf("EngineOutputQueue %s-%s: Starting engine output queue", eoq.ComponentID, eoq.EngineType)
	go eoq.run()
	go eoq.TimeoutManager.Start()
}

// Stop stops the engine output queue processing
func (eoq *EngineOutputQueue) Stop() {
	log.Printf("EngineOutputQueue %s-%s: Stopping engine output queue", eoq.ComponentID, eoq.EngineType)
	eoq.cancel()
	eoq.TimeoutManager.Stop()
}

// run is the main processing loop
func (eoq *EngineOutputQueue) run() {
	for {
		select {
		case request := <-eoq.InputChannel:
			// Handle ALL routing decisions here
			if err := eoq.handleAllRouting(request); err != nil {
				log.Printf("EngineOutputQueue %s-%s: Routing error for request %s: %v", 
					eoq.ComponentID, eoq.EngineType, request.Request.ID, err)
				eoq.RoutingMetrics.RoutingErrors++
			}

		case <-eoq.ctx.Done():
			log.Printf("EngineOutputQueue %s-%s: Engine output queue stopping", eoq.ComponentID, eoq.EngineType)
			return
		}
	}
}

// handleAllRouting handles ALL routing decisions - both internal and external
func (eoq *EngineOutputQueue) handleAllRouting(request *EngineOutputRequest) error {
	startTime := time.Now()
	defer func() {
		eoq.RoutingMetrics.TotalRoutingDecisions++
		latency := time.Since(startTime)
		eoq.updateAverageLatency(latency)
	}()

	// Update request position
	request.Request.SetCurrentPosition(eoq.ComponentID, string(eoq.EngineType), request.Request.CurrentNode)
	request.Request.IncrementEngineCount()

	// Add to history if tracking enabled
	request.Request.AddToHistory(eoq.ComponentID, string(eoq.EngineType), 
		request.EngineResult.OperationType, request.EngineResult.Success)

	// Register with timeout manager
	eoq.TimeoutManager.RegisterRequest(request.Request.ID, eoq.ComponentID, string(eoq.EngineType))

	// 1. Look up component-level graph for routing decision
	componentGraph := eoq.ComponentLB.GetComponentGraph()
	if componentGraph == nil {
		return fmt.Errorf("no component graph available for component %s", eoq.ComponentID)
	}

	// 2. Determine next destination (internal or external)
	nextDestination, err := eoq.evaluateRoutingConditions(componentGraph, request)
	if err != nil {
		return fmt.Errorf("failed to evaluate routing conditions: %w", err)
	}

	// 3. Route based on destination type
	if eoq.isInternalEngine(nextDestination) {
		// Internal routing - route to next engine within component
		eoq.RoutingMetrics.InternalRoutingCount++
		return eoq.routeToInternalEngine(nextDestination, request)
	} else if eoq.isExternalComponent(nextDestination) {
		// External routing - route to different component
		eoq.RoutingMetrics.ExternalRoutingCount++
		return eoq.routeToExternalComponent(nextDestination, request)
	} else if eoq.isEndNode(nextDestination) {
		// End routing - route to end node
		return eoq.routeToEndNode(request)
	}

	return fmt.Errorf("unknown destination type: %s", nextDestination)
}

// evaluateRoutingConditions evaluates routing conditions using dynamic graph lookup
func (eoq *EngineOutputQueue) evaluateRoutingConditions(graph *DecisionGraph, request *EngineOutputRequest) (string, error) {
	currentNode := graph.Nodes[request.Request.CurrentNode]
	if currentNode == nil {
		// If no current node specified, start from beginning
		currentNode = graph.Nodes[graph.StartNode]
		if currentNode == nil {
			return "", fmt.Errorf("no start node found in component graph")
		}
	}

	// Check for probability-based routing
	if currentNode.RoutingType == "probability_based" {
		return eoq.evaluateProbabilityBasedRouting(currentNode, request)
	}

	// Check for dynamic state-based routing
	if currentNode.RoutingType == "dynamic_state_based" {
		return eoq.evaluateStateBasedRouting(currentNode, request)
	}

	// Standard condition evaluation
	return eoq.evaluateStandardRouting(currentNode, request)
}

// evaluateStandardRouting evaluates standard routing conditions
func (eoq *EngineOutputQueue) evaluateStandardRouting(node *DecisionNode, request *EngineOutputRequest) (string, error) {
	// Evaluate conditions based on engine result
	for condition, destination := range node.Conditions {
		if eoq.evaluateCondition(condition, request) {
			return destination, nil
		}
	}

	// Return default next node
	if node.Next != "" {
		return node.Next, nil
	}

	// If no conditions match and no default, this might be an end node
	return "end_node", nil
}

// evaluateCondition evaluates a single routing condition
func (eoq *EngineOutputQueue) evaluateCondition(condition string, request *EngineOutputRequest) bool {
	result := request.EngineResult

	switch condition {
	case "cache_hit":
		return result.Success && result.Data != nil
	case "cache_miss":
		return !result.Success || result.Data == nil
	case "parse_success":
		return result.Success
	case "parse_failure":
		return !result.Success
	case "authenticated":
		return result.Success && request.Request.Data.AuthResult != nil && request.Request.Data.AuthResult.IsAuthenticated
	case "not_authenticated":
		return !result.Success || request.Request.Data.AuthResult == nil || !request.Request.Data.AuthResult.IsAuthenticated
	case "in_stock":
		return result.Success && request.Request.Data.InventoryResult != nil && request.Request.Data.InventoryResult.InStock
	case "out_of_stock":
		return !result.Success || request.Request.Data.InventoryResult == nil || !request.Request.Data.InventoryResult.InStock
	case "payment_success":
		return result.Success && request.Request.Data.PaymentResult != nil && request.Request.Data.PaymentResult.Processed
	case "payment_failure":
		return !result.Success || request.Request.Data.PaymentResult == nil || !request.Request.Data.PaymentResult.Processed
	default:
		// Unknown condition - default to false
		log.Printf("EngineOutputQueue %s-%s: Unknown condition '%s'", eoq.ComponentID, eoq.EngineType, condition)
		return false
	}
}

// evaluateProbabilityBasedRouting evaluates probability-based routing
func (eoq *EngineOutputQueue) evaluateProbabilityBasedRouting(node *DecisionNode, request *EngineOutputRequest) (string, error) {
	if node.ProbabilityConfig == nil {
		return eoq.evaluateStandardRouting(node, request)
	}

	// Generate random number for probability decision
	randomValue := rand.Float64()

	switch request.EngineResult.OperationType {
	case "cache_lookup":
		if randomValue < node.ProbabilityConfig.CacheHitRate {
			return node.ProbabilityConfig.Conditions["cache_hit"], nil
		}
		return node.ProbabilityConfig.Conditions["cache_miss"], nil

	case "database_lookup":
		if randomValue < node.ProbabilityConfig.SuccessRate {
			return node.ProbabilityConfig.Conditions["data_found"], nil
		}
		return node.ProbabilityConfig.Conditions["data_not_found"], nil

	case "network_request":
		if randomValue < node.ProbabilityConfig.SuccessRate {
			return node.ProbabilityConfig.Conditions["request_success"], nil
		}
		return node.ProbabilityConfig.Conditions["request_timeout"], nil
	}

	// Fallback to standard routing
	return eoq.evaluateStandardRouting(node, request)
}

// evaluateStateBasedRouting evaluates dynamic state-based routing
func (eoq *EngineOutputQueue) evaluateStateBasedRouting(node *DecisionNode, request *EngineOutputRequest) (string, error) {
	// Dynamic routing based on current system state
	for condition, destination := range node.Conditions {
		if eoq.evaluateCurrentStateCondition(condition, request) {
			return destination, nil
		}
	}

	// Fallback to standard routing
	return eoq.evaluateStandardRouting(node, request)
}

// evaluateCurrentStateCondition evaluates conditions based on current system state
func (eoq *EngineOutputQueue) evaluateCurrentStateCondition(condition string, request *EngineOutputRequest) bool {
	switch condition {
	case "high_load":
		return eoq.getCurrentSystemLoad() > 0.8
	case "low_memory":
		return eoq.getAvailableMemory() < 0.2
	case "storage_fast":
		return eoq.getStorageLatency() < 10*time.Millisecond
	case "read_only":
		return request.Request.Data.Operation == "read" || request.Request.Data.Operation == "select"
	case "write_query":
		return request.Request.Data.Operation == "write" || request.Request.Data.Operation == "insert" || request.Request.Data.Operation == "update"
	default:
		return false
	}
}

// Routing destination type checks
func (eoq *EngineOutputQueue) isInternalEngine(destination string) bool {
	// Check if destination is an engine type within the same component
	engineTypes := []string{"cpu", "memory", "storage", "network"}
	for _, engineType := range engineTypes {
		if destination == engineType {
			return true
		}
	}
	return false
}

func (eoq *EngineOutputQueue) isExternalComponent(destination string) bool {
	// Check if destination is a component ID in the global registry
	return eoq.GlobalRegistry.GetChannel(destination) != nil
}

func (eoq *EngineOutputQueue) isEndNode(destination string) bool {
	return destination == "end_node" || destination == "end" || destination == "complete"
}

// Routing implementations
func (eoq *EngineOutputQueue) routeToInternalEngine(engineType string, request *EngineOutputRequest) error {
	// Route to next engine within the same component instance
	// This would typically go through the component instance's engine manager
	log.Printf("EngineOutputQueue %s-%s: Routing request %s to internal engine %s",
		eoq.ComponentID, eoq.EngineType, request.Request.ID, engineType)

	// Update request position
	request.Request.CurrentEngine = engineType

	// Send to output channel for component instance to handle
	select {
	case eoq.OutputChannel <- request:
		return nil
	default:
		return fmt.Errorf("output channel full for internal routing to %s", engineType)
	}
}

func (eoq *EngineOutputQueue) routeToExternalComponent(componentID string, request *EngineOutputRequest) error {
	// Route to different component via global registry
	log.Printf("EngineOutputQueue %s-%s: Routing request %s to external component %s",
		eoq.ComponentID, eoq.EngineType, request.Request.ID, componentID)

	// Update request position
	request.Request.CurrentComponent = componentID
	request.Request.IncrementComponentCount()

	// Get target component channel from global registry
	targetChannel := eoq.GlobalRegistry.GetChannel(componentID)
	if targetChannel == nil {
		return fmt.Errorf("component %s not found in global registry", componentID)
	}

	// Convert to operation for target component
	operation := &engines.Operation{
		ID:            request.Request.ID,
		Type:          request.Request.Data.Operation,
		Data:          request.Request.Data.Payload,
		RequestID:     request.Request.ID,
		ComponentID:   componentID,
		Priority:      1, // Default priority
		Timestamp:     time.Now(),
	}

	// Route to target component
	select {
	case targetChannel <- operation:
		return nil
	default:
		return fmt.Errorf("target component %s channel is full", componentID)
	}
}

func (eoq *EngineOutputQueue) routeToEndNode(request *EngineOutputRequest) error {
	// Route to end node for completion
	log.Printf("EngineOutputQueue %s-%s: Routing request %s to end node",
		eoq.ComponentID, eoq.EngineType, request.Request.ID)

	// Mark request as complete
	request.Request.MarkComplete()

	// Unregister from timeout manager
	eoq.TimeoutManager.UnregisterRequest(request.Request.ID)

	// Send to output channel for final processing
	select {
	case eoq.OutputChannel <- request:
		return nil
	default:
		return fmt.Errorf("output channel full for end node routing")
	}
}

// System state helpers (these would be implemented based on actual system monitoring)
func (eoq *EngineOutputQueue) getCurrentSystemLoad() float64 {
	// Placeholder - would get actual system load
	return 0.5
}

func (eoq *EngineOutputQueue) getAvailableMemory() float64 {
	// Placeholder - would get actual available memory percentage
	return 0.7
}

func (eoq *EngineOutputQueue) getStorageLatency() time.Duration {
	// Placeholder - would get actual storage latency
	return 5 * time.Millisecond
}

// Metrics helpers
func (eoq *EngineOutputQueue) updateAverageLatency(latency time.Duration) {
	// Simple moving average calculation
	if eoq.RoutingMetrics.AverageRoutingLatency == 0 {
		eoq.RoutingMetrics.AverageRoutingLatency = latency
	} else {
		// Weighted average with 90% old, 10% new
		eoq.RoutingMetrics.AverageRoutingLatency = time.Duration(
			float64(eoq.RoutingMetrics.AverageRoutingLatency)*0.9 + float64(latency)*0.1)
	}
}

// GetMetrics returns current routing metrics
func (eoq *EngineOutputQueue) GetMetrics() *EngineOutputMetrics {
	return eoq.RoutingMetrics
}
