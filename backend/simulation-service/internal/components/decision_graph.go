package components

import (
	"fmt"
	"sync"

	"github.com/systemsim/simulation-service/internal/engines"
)

// DecisionGraph manages the routing of operations between engines within a component
// Enhanced to support probability-based routing, dynamic state-based routing, and custom routing logic
type DecisionGraph struct {
	// Graph structure
	Name      string                    `json:"name"`
	StartNode string                    `json:"start_node"`
	EndNodes  []string                  `json:"end_nodes"`
	Nodes     map[string]*DecisionNode  `json:"nodes"`
	Level     GraphLevel                `json:"level"` // Component or System level

	// Engine references
	Engines   map[engines.EngineType]*engines.EngineWrapper `json:"-"`

	// Routing channels for dynamic decisions
	EngineOutputChannels map[engines.EngineType]chan *engines.OperationResult `json:"-"`
	DecisionChannel      chan *RoutingDecision                                 `json:"-"`

	// Component context
	ComponentID   string        `json:"component_id"`
	InstanceID    string        `json:"instance_id"`

	// Enhanced routing features
	ProbabilityEngine *ProbabilityEngine `json:"-"` // For probability-based routing
	StateMonitor      *StateMonitor      `json:"-"` // For dynamic state-based routing
	CustomLogicMap    map[string]CustomRoutingFunc `json:"-"` // For custom routing logic

	// State tracking
	mutex         sync.RWMutex  `json:"-"`
	isRunning     bool          `json:"-"`

	// Performance tracking
	DecisionStats map[string]*DecisionStats `json:"-"`
}

// NewDecisionGraph creates a new decision graph from configuration
func NewDecisionGraph(config *DecisionGraphConfig, engines map[engines.EngineType]*engines.EngineWrapper) *DecisionGraph {
	return &DecisionGraph{
		StartNode: config.StartNode,
		EndNodes:  config.EndNodes,
		Nodes:     config.Nodes,
		Engines:   engines,
	}
}

// ExecuteOperation processes an operation through the decision graph
func (dg *DecisionGraph) ExecuteOperation(op *engines.Operation) error {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()
	
	// Start at the start node
	currentNodeID := dg.StartNode
	
	// Track the path through the graph for debugging
	executionPath := []string{currentNodeID}
	
	// Execute the operation through the graph
	for {
		// Get current node
		currentNode, exists := dg.Nodes[currentNodeID]
		if !exists {
			return fmt.Errorf("node %s not found in decision graph", currentNodeID)
		}
		
		// Check if we've reached an end node
		if dg.isEndNode(currentNodeID) {
			// Operation completed successfully
			return nil
		}
		
		// Process the operation at this node
		nextNodeID, err := dg.processNode(currentNode, op)
		if err != nil {
			return fmt.Errorf("failed to process node %s: %w", currentNodeID, err)
		}
		
		// Move to next node
		currentNodeID = nextNodeID
		executionPath = append(executionPath, currentNodeID)
		
		// Prevent infinite loops
		if len(executionPath) > 100 {
			return fmt.Errorf("execution path too long, possible infinite loop: %v", executionPath)
		}
	}
}

// isEndNode checks if a node is an end node
func (dg *DecisionGraph) isEndNode(nodeID string) bool {
	for _, endNode := range dg.EndNodes {
		if nodeID == endNode {
			return true
		}
	}
	return false
}

// processNode processes an operation at a specific node
func (dg *DecisionGraph) processNode(node *DecisionNode, op *engines.Operation) (string, error) {
	switch node.Type {
	case "engine":
		return dg.processEngineNode(node, op)
	case "decision":
		return dg.processDecisionNode(node, op)
	case "end":
		return "", nil // End nodes don't have next nodes
	default:
		return "", fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// processEngineNode handles engine nodes that process operations
func (dg *DecisionGraph) processEngineNode(node *DecisionNode, op *engines.Operation) (string, error) {
	// Get the engine for this node
	engine, exists := dg.Engines[node.EngineType]
	if !exists {
		return "", fmt.Errorf("engine %s not found for node %s", node.EngineType, node.ID)
	}
	
	// Process operation through the engine
	if err := engine.QueueOperation(op); err != nil {
		return "", fmt.Errorf("engine %s failed to queue operation: %w", node.EngineType, err)
	}
	
	// Determine next node based on operation result or default routing
	nextNode := dg.getNextNode(node, op)
	return nextNode, nil
}

// processDecisionNode handles decision nodes that route based on conditions
// Enhanced to support probability-based, dynamic state-based, and custom routing
func (dg *DecisionGraph) processDecisionNode(node *DecisionNode, op *engines.Operation) (string, error) {
	// Check routing type and use appropriate evaluation method
	switch node.RoutingType {
	case "probability_based":
		return dg.processProbabilityBasedRouting(node, op)
	case "dynamic_state_based":
		return dg.processStateBasedRouting(node, op)
	case "custom_logic":
		return dg.processCustomLogicRouting(node, op)
	default:
		return dg.processStandardRouting(node, op)
	}
}

// processStandardRouting handles standard condition-based routing
func (dg *DecisionGraph) processStandardRouting(node *DecisionNode, op *engines.Operation) (string, error) {
	// Evaluate conditions to determine next node
	for condition, nextNodeID := range node.Conditions {
		if dg.evaluateCondition(condition, op) {
			return nextNodeID, nil
		}
	}

	// If no conditions match, use default routing
	if defaultNext, exists := node.Conditions["default"]; exists {
		return defaultNext, nil
	}

	return "", fmt.Errorf("no matching condition found for decision node %s", node.ID)
}

// processProbabilityBasedRouting handles probability-based routing decisions
func (dg *DecisionGraph) processProbabilityBasedRouting(node *DecisionNode, op *engines.Operation) (string, error) {
	if node.ProbabilityConfig == nil {
		return dg.processStandardRouting(node, op) // Fallback to standard routing
	}

	if dg.ProbabilityEngine == nil {
		dg.ProbabilityEngine = NewProbabilityEngine()
	}

	// Generate routing decision based on probability configuration
	decision := dg.ProbabilityEngine.MakeDecision(node.ProbabilityConfig, op)

	// Look up destination for the decision
	if destination, exists := node.ProbabilityConfig.Conditions[decision]; exists {
		return destination, nil
	}

	// Fallback to standard routing if probability decision not found
	return dg.processStandardRouting(node, op)
}

// processStateBasedRouting handles dynamic state-based routing decisions
func (dg *DecisionGraph) processStateBasedRouting(node *DecisionNode, op *engines.Operation) (string, error) {
	if node.StateConfig == nil {
		return dg.processStandardRouting(node, op) // Fallback to standard routing
	}

	if dg.StateMonitor == nil {
		dg.StateMonitor = NewStateMonitor(dg.ComponentID)
	}

	// Get current system state
	currentState := dg.StateMonitor.GetCurrentState()

	// Evaluate state-based conditions
	for stateCondition, destination := range node.StateConfig.StateChecks {
		if dg.evaluateStateCondition(stateCondition, currentState, op) {
			return destination, nil
		}
	}

	// Use fallback destination if no state conditions match
	if node.StateConfig.Fallback != "" {
		return node.StateConfig.Fallback, nil
	}

	// Final fallback to standard routing
	return dg.processStandardRouting(node, op)
}

// processCustomLogicRouting handles custom routing logic
func (dg *DecisionGraph) processCustomLogicRouting(node *DecisionNode, op *engines.Operation) (string, error) {
	if dg.CustomLogicMap == nil {
		return dg.processStandardRouting(node, op) // Fallback to standard routing
	}

	// Look up custom routing function
	customFunc, exists := dg.CustomLogicMap[node.ID]
	if !exists {
		return dg.processStandardRouting(node, op) // Fallback if no custom logic defined
	}

	// Execute custom routing logic
	destination, err := customFunc(node, op, dg)
	if err != nil {
		return "", fmt.Errorf("custom routing logic failed for node %s: %w", node.ID, err)
	}

	return destination, nil
}

// evaluateStateCondition evaluates state-based conditions
func (dg *DecisionGraph) evaluateStateCondition(condition string, state *SystemState, op *engines.Operation) bool {
	switch condition {
	case "high_load":
		return state.SystemLoad > 0.8
	case "low_load":
		return state.SystemLoad < 0.3
	case "high_memory_usage":
		return state.MemoryUsage > 0.9
	case "low_memory_usage":
		return state.MemoryUsage < 0.5
	case "storage_fast":
		return state.StorageLatency < 10 // milliseconds
	case "storage_slow":
		return state.StorageLatency > 100 // milliseconds
	case "network_congested":
		return state.NetworkLatency > 50 // milliseconds
	case "network_clear":
		return state.NetworkLatency < 10 // milliseconds
	case "peak_hours":
		return state.IsPeakHours
	case "off_peak_hours":
		return !state.IsPeakHours
	default:
		return false
	}
}

// evaluateCondition evaluates a condition against an operation
func (dg *DecisionGraph) evaluateCondition(condition string, op *engines.Operation) bool {
	switch condition {
	case "default":
		return true
	case "cpu_operation":
		return op.Type == "cpu_compute" || op.Type == "cpu_algorithm"
	case "memory_operation":
		return op.Type == "memory_read" || op.Type == "memory_write"
	case "storage_operation":
		return op.Type == "storage_read" || op.Type == "storage_write"
	case "network_operation":
		return op.Type == "network_send" || op.Type == "network_recv"
	case "high_priority":
		return op.Priority > 5
	case "low_priority":
		return op.Priority <= 5
	default:
		// Check if condition is in operation metadata
		if value, exists := op.Metadata[condition]; exists {
			if boolValue, ok := value.(bool); ok {
				return boolValue
			}
		}
		return false
	}
}

// getNextNode determines the next node based on operation characteristics
func (dg *DecisionGraph) getNextNode(node *DecisionNode, op *engines.Operation) string {
	// Check if operation specifies next component/node
	if nextNode, exists := op.Metadata["next_node"]; exists {
		if nextNodeStr, ok := nextNode.(string); ok {
			return nextNodeStr
		}
	}
	
	// Use default routing from node conditions
	if defaultNext, exists := node.Conditions["default"]; exists {
		return defaultNext
	}
	
	// If no routing specified, assume operation is complete
	if len(dg.EndNodes) > 0 {
		return dg.EndNodes[0] // Return first end node
	}
	
	return "end" // Fallback
}

// AddNode adds a new node to the decision graph
func (dg *DecisionGraph) AddNode(node *DecisionNode) {
	dg.mutex.Lock()
	defer dg.mutex.Unlock()
	
	if dg.Nodes == nil {
		dg.Nodes = make(map[string]*DecisionNode)
	}
	
	dg.Nodes[node.ID] = node
}

// RemoveNode removes a node from the decision graph
func (dg *DecisionGraph) RemoveNode(nodeID string) {
	dg.mutex.Lock()
	defer dg.mutex.Unlock()
	
	delete(dg.Nodes, nodeID)
}

// GetNodes returns all nodes in the decision graph
func (dg *DecisionGraph) GetNodes() map[string]*DecisionNode {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()
	
	// Return a copy to prevent external modification
	nodesCopy := make(map[string]*DecisionNode)
	for nodeID, node := range dg.Nodes {
		nodesCopy[nodeID] = node
	}
	return nodesCopy
}

// Validate checks if the decision graph is valid
func (dg *DecisionGraph) Validate() error {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()
	
	// Check if start node exists
	if _, exists := dg.Nodes[dg.StartNode]; !exists {
		return fmt.Errorf("start node %s not found", dg.StartNode)
	}
	
	// Check if all end nodes exist
	for _, endNode := range dg.EndNodes {
		if _, exists := dg.Nodes[endNode]; !exists {
			return fmt.Errorf("end node %s not found", endNode)
		}
	}
	
	// Check if all referenced nodes in conditions exist
	for nodeID, node := range dg.Nodes {
		for _, nextNodeID := range node.Conditions {
			if _, exists := dg.Nodes[nextNodeID]; !exists {
				return fmt.Errorf("node %s references non-existent node %s", nodeID, nextNodeID)
			}
		}
	}
	
	return nil
}
