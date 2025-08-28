package components

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// Run starts the centralized output manager goroutine
func (com *CentralizedOutputManager) Run(ctx context.Context) {
	// Ensure sync package is recognized as used
	var _ sync.Mutex
	com.mutex.Lock()
	com.ctx, com.cancel = context.WithCancel(ctx)
	com.running = true
	com.mutex.Unlock()

	log.Printf("CentralizedOutputManager %s: Starting centralized output goroutine", com.InstanceID)

	for {
		select {
		case result := <-com.InputChannel:
			// Handle operation result from last engine
			if err := com.handleOperationResult(result); err != nil {
				log.Printf("CentralizedOutputManager %s: Error handling result %s: %v", 
					com.InstanceID, result.OperationID, err)
			}

		case <-com.ctx.Done():
			log.Printf("CentralizedOutputManager %s: Centralized output goroutine stopping", com.InstanceID)
			return
		}
	}
}

// Stop stops the centralized output manager
func (com *CentralizedOutputManager) Stop() error {
	com.mutex.Lock()
	defer com.mutex.Unlock()

	if !com.running {
		return fmt.Errorf("centralized output manager %s is not running", com.InstanceID)
	}

	log.Printf("CentralizedOutputManager %s: Stopping", com.InstanceID)

	if com.cancel != nil {
		com.cancel()
	}

	com.running = false
	return nil
}

// handleOperationResult processes a result from the last engine and routes to next component
// Enhanced to read system graphs from Global Registry and handle business logic evaluation
func (com *CentralizedOutputManager) handleOperationResult(result *engines.OperationResult) error {
	log.Printf("CentralizedOutputManager %s: Handling result %s", com.InstanceID, result.OperationID)

	// Get request context from global registry
	requestCtx, err := com.getRequestContext(result.OperationID)
	if err != nil {
		return fmt.Errorf("failed to get request context: %w", err)
	}

	// Get system graph from global registry
	systemGraph, err := com.getSystemGraph(requestCtx.SystemFlowID)
	if err != nil {
		return fmt.Errorf("failed to get system graph: %w", err)
	}

	// Determine next component using system graph and business logic evaluation
	nextComponent, subFlowRequired, err := com.evaluateSystemGraph(systemGraph, requestCtx, result)
	if err != nil {
		return fmt.Errorf("failed to evaluate system graph: %w", err)
	}

	// Handle sub-flow execution if needed
	if subFlowRequired != "" {
		return com.executeSubFlow(subFlowRequired, requestCtx, result)
	}

	// If no next component, this is the end of the flow
	if nextComponent == "" {
		log.Printf("CentralizedOutputManager %s: End of flow for result %s", com.InstanceID, result.OperationID)
		return com.routeToEndNode(result)
	}

	// Route to next component via global registry
	return com.routeToNextComponent(nextComponent, result)
}

// getRequestContext gets request context from global registry
func (com *CentralizedOutputManager) getRequestContext(requestID string) (*RequestContext, error) {
	if com.GlobalRegistry == nil {
		return nil, fmt.Errorf("no global registry available")
	}

	// Try to get request context from enhanced global registry
	if enhancedRegistry, ok := com.GlobalRegistry.(GlobalRegistryInterface); ok {
		return enhancedRegistry.GetRequestContext(requestID), nil
	}

	// Fallback: create basic request context
	return &RequestContext{
		RequestID:         requestID,
		SystemFlowID:      "default_flow",
		CurrentSystemNode: "start",
		StartTime:         time.Now().Format(time.RFC3339),
		LastUpdate:        time.Now().Format(time.RFC3339),
	}, nil
}

// getSystemGraph gets system graph from global registry
func (com *CentralizedOutputManager) getSystemGraph(flowID string) (*DecisionGraph, error) {
	if com.GlobalRegistry == nil {
		return nil, fmt.Errorf("no global registry available")
	}

	// Try to get system graph from enhanced global registry
	if enhancedRegistry, ok := com.GlobalRegistry.(GlobalRegistryInterface); ok {
		graph := enhancedRegistry.GetSystemGraph(flowID)
		if graph != nil {
			return graph, nil
		}
	}

	// Fallback: create basic system graph
	return &DecisionGraph{
		Name:      "default_system_graph",
		StartNode: "start",
		Level:     SystemLevel,
		Nodes: map[string]*DecisionNode{
			"start": {
				Target:     com.ComponentID,
				Operation:  "process",
				Next:       "end",
				Conditions: make(map[string]string),
			},
			"end": {
				Target:     "end_node",
				Operation:  "complete",
				Conditions: make(map[string]string),
			},
		},
	}, nil
}

// evaluateSystemGraph evaluates system graph with business logic conditions
func (com *CentralizedOutputManager) evaluateSystemGraph(graph *DecisionGraph, requestCtx *RequestContext, result *engines.OperationResult) (string, string, error) {
	currentNode := graph.Nodes[requestCtx.CurrentSystemNode]
	if currentNode == nil {
		// Start from beginning if no current node
		currentNode = graph.Nodes[graph.StartNode]
		if currentNode == nil {
			return "", "", fmt.Errorf("no start node found in system graph")
		}
	}

	// Evaluate business logic conditions
	for condition, destination := range currentNode.Conditions {
		if com.evaluateBusinessLogicCondition(condition, requestCtx, result) {
			// Check if destination is a sub-flow
			if com.isSubFlow(destination) {
				return "", destination, nil // Return sub-flow for execution
			}
			return destination, "", nil // Return next component
		}
	}

	// Return default next node
	if currentNode.Next != "" {
		return currentNode.Next, "", nil
	}

	// No conditions matched and no default - end of flow
	return "", "", nil
}

// evaluateBusinessLogicCondition evaluates business logic conditions
func (com *CentralizedOutputManager) evaluateBusinessLogicCondition(condition string, requestCtx *RequestContext, result *engines.OperationResult) bool {
	switch condition {
	case "authenticated":
		return com.isUserAuthenticated(requestCtx, result)
	case "not_authenticated":
		return !com.isUserAuthenticated(requestCtx, result)
	case "in_stock":
		return com.isItemInStock(requestCtx, result)
	case "out_of_stock":
		return !com.isItemInStock(requestCtx, result)
	case "payment_success":
		return com.isPaymentSuccessful(requestCtx, result)
	case "payment_failure":
		return !com.isPaymentSuccessful(requestCtx, result)
	case "high_priority":
		return com.isHighPriority(requestCtx, result)
	case "low_priority":
		return !com.isHighPriority(requestCtx, result)
	case "cache_hit":
		return result.Success && result.Data != nil
	case "cache_miss":
		return !result.Success || result.Data == nil
	case "processing_success":
		return result.Success
	case "processing_failure":
		return !result.Success
	default:
		log.Printf("CentralizedOutputManager %s: Unknown business logic condition '%s'", com.InstanceID, condition)
		return false
	}
}

// Business logic evaluation helpers
func (com *CentralizedOutputManager) isUserAuthenticated(requestCtx *RequestContext, result *engines.OperationResult) bool {
	// Check if authentication result is available in result metadata
	if result.Metadata != nil {
		if authResult, exists := result.Metadata["auth_result"]; exists {
			if authMap, ok := authResult.(map[string]interface{}); ok {
				if authenticated, exists := authMap["is_authenticated"]; exists {
					if authBool, ok := authenticated.(bool); ok {
						return authBool
					}
				}
			}
		}
	}

	// Fallback: check if operation was successful and type was authentication
	return result.Success && result.OperationType == "authenticate"
}

func (com *CentralizedOutputManager) isItemInStock(requestCtx *RequestContext, result *engines.OperationResult) bool {
	// Check if inventory result is available in result metadata
	if result.Metadata != nil {
		if inventoryResult, exists := result.Metadata["inventory_result"]; exists {
			if inventoryMap, ok := inventoryResult.(map[string]interface{}); ok {
				if inStock, exists := inventoryMap["in_stock"]; exists {
					if stockBool, ok := inStock.(bool); ok {
						return stockBool
					}
				}
			}
		}
	}

	// Fallback: check if operation was successful and type was inventory check
	return result.Success && result.OperationType == "inventory_check"
}

func (com *CentralizedOutputManager) isPaymentSuccessful(requestCtx *RequestContext, result *engines.OperationResult) bool {
	// Check if payment result is available in result metadata
	if result.Metadata != nil {
		if paymentResult, exists := result.Metadata["payment_result"]; exists {
			if paymentMap, ok := paymentResult.(map[string]interface{}); ok {
				if processed, exists := paymentMap["processed"]; exists {
					if processedBool, ok := processed.(bool); ok {
						return processedBool
					}
				}
			}
		}
	}

	// Fallback: check if operation was successful and type was payment
	return result.Success && result.OperationType == "payment"
}

func (com *CentralizedOutputManager) isHighPriority(requestCtx *RequestContext, result *engines.OperationResult) bool {
	// Check priority in result metadata
	if result.Metadata != nil {
		if priority, exists := result.Metadata["priority"]; exists {
			if priorityInt, ok := priority.(int); ok {
				return priorityInt >= 8 // High priority threshold
			}
		}
	}

	// Fallback: assume normal priority
	return false
}

// isSubFlow checks if destination is a sub-flow
func (com *CentralizedOutputManager) isSubFlow(destination string) bool {
	// Sub-flows typically have a specific naming pattern
	return len(destination) > 9 && destination[:9] == "sub_flow_"
}

// executeSubFlow executes a sub-flow
func (com *CentralizedOutputManager) executeSubFlow(subFlowID string, requestCtx *RequestContext, result *engines.OperationResult) error {
	log.Printf("CentralizedOutputManager %s: Executing sub-flow %s for request %s",
		com.InstanceID, subFlowID, requestCtx.RequestID)

	// Get sub-flow graph from global registry
	subFlowGraph, err := com.getSystemGraph(subFlowID)
	if err != nil {
		return fmt.Errorf("failed to get sub-flow graph %s: %w", subFlowID, err)
	}

	// Create sub-flow context
	subFlowCtx := &RequestContext{
		RequestID:         requestCtx.RequestID + "-" + subFlowID,
		SystemFlowID:      subFlowID,
		CurrentSystemNode: subFlowGraph.StartNode,
		StartTime:         time.Now().Format(time.RFC3339),
		LastUpdate:        time.Now().Format(time.RFC3339),
	}

	// Execute sub-flow (recursive call)
	nextComponent, _, err := com.evaluateSystemGraph(subFlowGraph, subFlowCtx, result)
	if err != nil {
		return fmt.Errorf("failed to execute sub-flow %s: %w", subFlowID, err)
	}

	// Route to first component in sub-flow
	if nextComponent != "" {
		return com.routeToNextComponent(nextComponent, result)
	}

	// Sub-flow completed immediately
	return nil
}

// routeToEndNode routes request to end node for completion
func (com *CentralizedOutputManager) routeToEndNode(result *engines.OperationResult) error {
	log.Printf("CentralizedOutputManager %s: Routing result %s to end node", com.InstanceID, result.OperationID)

	// Send to output channel for external consumption
	select {
	case com.OutputChannel <- result:
		return nil
	default:
		return fmt.Errorf("output channel is full")
	}
}

// determineNextComponent determines the next component based on routing rules and user flow
func (com *CentralizedOutputManager) determineNextComponent(result *engines.OperationResult) (string, error) {
	// Priority 1: Check routing rules for operation type
	if com.RoutingRules != nil {
		if nextComponent, exists := com.RoutingRules[result.OperationType]; exists {
			log.Printf("CentralizedOutputManager %s: Using routing rule %s -> %s", 
				com.InstanceID, result.OperationType, nextComponent)
			return nextComponent, nil
		}
	}

	// Priority 2: Check user flow configuration
	if com.UserFlowConfig != nil {
		nextComponent := com.getNextComponentFromUserFlow(result)
		if nextComponent != "" {
			log.Printf("CentralizedOutputManager %s: Using user flow routing -> %s", 
				com.InstanceID, nextComponent)
			return nextComponent, nil
		}
	}

	// Priority 3: Use default routing
	if com.DefaultRouting != "" {
		log.Printf("CentralizedOutputManager %s: Using default routing -> %s", 
			com.InstanceID, com.DefaultRouting)
		return com.DefaultRouting, nil
	}

	// No routing configured - end of flow
	return "", nil
}

// getNextComponentFromUserFlow determines next component from user flow configuration
func (com *CentralizedOutputManager) getNextComponentFromUserFlow(result *engines.OperationResult) string {
	if com.UserFlowConfig == nil {
		return ""
	}

	// Search through all flows to find routing for current component
	for _, flow := range com.UserFlowConfig.Flows {
		for _, step := range flow.Steps {
			if step.ComponentID == com.ComponentID && step.Operation == result.OperationType {
				// Evaluate conditions for routing
				if len(step.Conditions) > 0 {
					nextComponent := com.evaluateStepConditions(step, result)
					if nextComponent != "" {
						log.Printf("CentralizedOutputManager %s: User flow routing %s -> %s",
							com.InstanceID, com.ComponentID, nextComponent)
						return nextComponent
					}
				}
			}
		}
	}

	// No user flow routing found
	return ""
}

// routeToNextComponent routes the result to the next component via global registry
func (com *CentralizedOutputManager) routeToNextComponent(nextComponentID string, result *engines.OperationResult) error {
	if com.GlobalRegistry == nil {
		compErr := CreateComponentError(
			ErrorCategoryConfiguration,
			ErrorSeverityCritical,
			"no_global_registry",
			"Global registry not configured for routing",
			com.ComponentID,
		)
		compErr.OperationID = result.OperationID
		if GlobalErrorHandler != nil {
			GlobalErrorHandler.HandleError(context.Background(), compErr, com.ComponentID)
		}
		return compErr
	}

	// Check target component health
	targetHealth := com.GlobalRegistry.GetHealth(nextComponentID)
	if targetHealth < 0.5 {
		return com.handleUnhealthyTarget(nextComponentID, result)
	}

	// Check target component load
	targetLoad := com.GlobalRegistry.GetLoad(nextComponentID)
	if targetLoad == BufferStatusEmergency {
		return com.handleOverloadedTarget(nextComponentID, result)
	}

	// Apply backpressure if needed
	if err := com.applyBackpressure(nextComponentID, targetLoad); err != nil {
		compErr := WrapError(err, com.ComponentID, result.OperationID)
		compErr.Category = ErrorCategoryResource
		compErr.Severity = ErrorSeverityHigh
		if GlobalErrorHandler != nil {
			GlobalErrorHandler.HandleError(context.Background(), compErr, com.ComponentID)
		}
		return compErr
	}

	// Use circuit breaker for normal routing to track component health
	if com.CircuitBreakerManager != nil {
		return com.CircuitBreakerManager.ExecuteWithCircuitBreaker(nextComponentID, func() error {
			return com.attemptRoutingDirect(nextComponentID, result)
		})
	}

	// Fallback to direct routing if no circuit breaker
	return com.attemptRoutingDirect(nextComponentID, result)
}

// handleUnhealthyTarget handles routing when target component is unhealthy
func (com *CentralizedOutputManager) handleUnhealthyTarget(targetComponentID string, result *engines.OperationResult) error {
	log.Printf("CentralizedOutputManager %s: Target component %s is unhealthy, trying fallback routing",
		com.InstanceID, targetComponentID)

	// Try fallback routing first
	if com.FallbackRouting != nil && com.FallbackRouting.Enabled {
		if err := com.attemptFallbackRouting(targetComponentID, result, "unhealthy"); err == nil {
			return nil // Fallback routing succeeded
		}
		log.Printf("CentralizedOutputManager %s: Fallback routing failed for unhealthy target %s",
			com.InstanceID, targetComponentID)
	}

	// Fallback to circuit breaker if available
	if com.CircuitBreakerManager == nil {
		return fmt.Errorf("target component %s is unhealthy and no circuit breaker available", targetComponentID)
	}

	// Use circuit breaker to handle unhealthy target
	return com.CircuitBreakerManager.ExecuteWithCircuitBreaker(targetComponentID, func() error {
		// Check if component is still unhealthy
		if com.GlobalRegistry != nil {
			targetHealth := com.GlobalRegistry.GetHealth(targetComponentID)
			if targetHealth < 0.5 {
				return fmt.Errorf("component %s still unhealthy (health: %.2f)", targetComponentID, targetHealth)
			}
		}

		// Try to route to the component
		return com.attemptRouting(targetComponentID, result)
	})
}

// handleOverloadedTarget handles routing when target component is overloaded
func (com *CentralizedOutputManager) handleOverloadedTarget(targetComponentID string, result *engines.OperationResult) error {
	log.Printf("CentralizedOutputManager %s: Target component %s is overloaded, trying fallback routing",
		com.InstanceID, targetComponentID)

	// Try fallback routing first
	if com.FallbackRouting != nil && com.FallbackRouting.Enabled {
		if err := com.attemptFallbackRouting(targetComponentID, result, "overloaded"); err == nil {
			return nil // Fallback routing succeeded
		}
		log.Printf("CentralizedOutputManager %s: Fallback routing failed for overloaded target %s",
			com.InstanceID, targetComponentID)
	}

	// Fallback to circuit breaker if available
	if com.CircuitBreakerManager == nil {
		return fmt.Errorf("target component %s is overloaded and no circuit breaker available", targetComponentID)
	}

	// Use circuit breaker to handle overloaded target with retry logic
	return com.CircuitBreakerManager.ExecuteWithCircuitBreaker(targetComponentID, func() error {
		// Apply backpressure delay before attempting
		if com.BackpressureConfig != nil {
			time.Sleep(com.BackpressureConfig.RetryDelay)
		}

		// Check if component is still overloaded
		if com.GlobalRegistry != nil {
			targetLoad := com.GlobalRegistry.GetLoad(targetComponentID)
			if targetLoad == BufferStatusEmergency {
				return fmt.Errorf("component %s still overloaded (load: %s)", targetComponentID, targetLoad)
			}
		}

		// Try to route to the component
		return com.attemptRouting(targetComponentID, result)
	})
}

// applyBackpressure applies backpressure based on target component load
func (com *CentralizedOutputManager) applyBackpressure(targetComponentID string, load BufferStatus) error {
	if com.BackpressureConfig == nil {
		return nil
	}

	switch load {
	case BufferStatusHigh:
		// Small delay for high load
		time.Sleep(com.BackpressureConfig.RetryDelay / 4)
	case BufferStatusCritical:
		// Medium delay for critical load
		time.Sleep(com.BackpressureConfig.RetryDelay / 2)
	case BufferStatusEmergency:
		// Full delay for emergency load
		time.Sleep(com.BackpressureConfig.RetryDelay)
	default:
		// No delay for normal/low load
	}

	return nil
}

// attemptRouting attempts to route a result to a target component (for circuit breaker retry logic)
func (com *CentralizedOutputManager) attemptRouting(targetComponentID string, result *engines.OperationResult) error {
	if com.GlobalRegistry == nil {
		return fmt.Errorf("no global registry available for routing")
	}

	// Get target component input channel
	targetChannel := com.GlobalRegistry.GetChannel(targetComponentID)
	if targetChannel == nil {
		return fmt.Errorf("target component %s not found in registry", targetComponentID)
	}

	// Convert result back to operation for next component
	nextOperation := com.resultToOperation(result, targetComponentID)

	// Route to target component
	select {
	case targetChannel <- nextOperation:
		log.Printf("CentralizedOutputManager %s: Successfully routed result %s to component %s",
			com.InstanceID, result.OperationID, targetComponentID)
		return nil
	default:
		return fmt.Errorf("target component %s input channel is full", targetComponentID)
	}
}

// attemptRoutingDirect attempts to route using the existing registry interface
func (com *CentralizedOutputManager) attemptRoutingDirect(targetComponentID string, result *engines.OperationResult) error {
	if com.GlobalRegistry == nil {
		return fmt.Errorf("no global registry available for routing")
	}

	// Get target component's input channel
	targetChannel := com.GlobalRegistry.GetChannel(targetComponentID)
	if targetChannel == nil {
		return fmt.Errorf("target component %s not found in registry", targetComponentID)
	}

	// Convert result back to operation for next component
	nextOperation := com.resultToOperation(result, targetComponentID)

	// Route to target component
	select {
	case targetChannel <- nextOperation:
		log.Printf("CentralizedOutputManager %s: Successfully routed result %s to component %s",
			com.InstanceID, result.OperationID, targetComponentID)
		return nil
	default:
		return fmt.Errorf("target component %s input channel is full", targetComponentID)
	}
}

// resultToOperation converts an operation result back to an operation for the next component
func (com *CentralizedOutputManager) resultToOperation(result *engines.OperationResult, nextComponentID string) *engines.Operation {
	// Create new operation based on result
	operation := &engines.Operation{
		ID:            result.OperationID + "-next",
		Type:          result.OperationType,
		DataSize:      1024, // TODO: Get from result metadata
		Complexity:    "O(1)", // TODO: Get from result metadata
		Language:      "go",
		Priority:      5,
		StartTick:     0, // Will be set by receiving component
		NextComponent: nextComponentID,
		Metadata:      make(map[string]interface{}),
	}

	// Copy relevant metadata from result
	if result.Metrics != nil {
		for key, value := range result.Metrics {
			operation.Metadata[key] = value
		}
	}

	// Add routing metadata
	operation.Metadata["source_component"] = com.ComponentID
	operation.Metadata["source_instance"] = com.InstanceID
	operation.Metadata["original_operation_id"] = result.OperationID

	return operation
}

// attemptFallbackRouting attempts to route to fallback targets when primary target fails
func (com *CentralizedOutputManager) attemptFallbackRouting(primaryTarget string, result *engines.OperationResult, reason string) error {
	if com.FallbackRouting == nil || !com.FallbackRouting.Enabled {
		return fmt.Errorf("fallback routing not enabled")
	}

	// Get fallback targets in priority order
	fallbackTargets := com.getFallbackTargets(primaryTarget, result)
	if len(fallbackTargets) == 0 {
		return fmt.Errorf("no fallback targets available for %s", primaryTarget)
	}

	log.Printf("CentralizedOutputManager %s: Attempting fallback routing for %s (reason: %s), targets: %v",
		com.InstanceID, primaryTarget, reason, fallbackTargets)

	// Try each fallback target
	maxAttempts := com.FallbackRouting.MaxFallbackAttempts
	if maxAttempts <= 0 {
		maxAttempts = len(fallbackTargets) // Try all fallbacks by default
	}

	for i, fallbackTarget := range fallbackTargets {
		if i >= maxAttempts {
			break
		}

		// Apply fallback delay if configured
		if i > 0 && com.FallbackRouting.FallbackDelay > 0 {
			time.Sleep(com.FallbackRouting.FallbackDelay)
		}

		// Check fallback target health and load
		if com.GlobalRegistry != nil {
			targetHealth := com.GlobalRegistry.GetHealth(fallbackTarget)
			targetLoad := com.GlobalRegistry.GetLoad(fallbackTarget)

			// Skip unhealthy or overloaded fallback targets
			if targetHealth < 0.5 {
				log.Printf("CentralizedOutputManager %s: Skipping unhealthy fallback target %s (health: %.2f)",
					com.InstanceID, fallbackTarget, targetHealth)
				continue
			}

			if targetLoad == BufferStatusEmergency {
				log.Printf("CentralizedOutputManager %s: Skipping overloaded fallback target %s (load: %s)",
					com.InstanceID, fallbackTarget, targetLoad)
				continue
			}
		}

		// Attempt routing to fallback target
		if err := com.attemptRoutingDirect(fallbackTarget, result); err == nil {
			log.Printf("CentralizedOutputManager %s: Successfully routed to fallback target %s (attempt %d)",
				com.InstanceID, fallbackTarget, i+1)
			return nil
		} else {
			log.Printf("CentralizedOutputManager %s: Failed to route to fallback target %s: %v",
				com.InstanceID, fallbackTarget, err)
		}
	}

	return fmt.Errorf("all fallback targets failed for %s", primaryTarget)
}

// getFallbackTargets returns the list of fallback targets for a primary target
func (com *CentralizedOutputManager) getFallbackTargets(primaryTarget string, result *engines.OperationResult) []string {
	var fallbackTargets []string

	// Priority 1: Specific fallback targets for this primary target
	if targets, exists := com.FallbackRouting.FallbackTargets[primaryTarget]; exists {
		fallbackTargets = append(fallbackTargets, targets...)
	}

	// Priority 2: Operation type specific fallbacks
	if targets, exists := com.FallbackRouting.OperationTypeFallbacks[result.OperationType]; exists {
		fallbackTargets = append(fallbackTargets, targets...)
	}

	// Priority 3: Condition-based fallbacks (check penalty info)
	if result.PenaltyInfo != nil {
		if targets, exists := com.FallbackRouting.ConditionFallbacks[result.PenaltyInfo.RecommendedAction]; exists {
			fallbackTargets = append(fallbackTargets, targets...)
		}
		if targets, exists := com.FallbackRouting.ConditionFallbacks[result.PenaltyInfo.PerformanceGrade]; exists {
			fallbackTargets = append(fallbackTargets, targets...)
		}
	}

	// Remove duplicates and apply strategy
	fallbackTargets = com.removeDuplicates(fallbackTargets)
	return com.applyFallbackStrategy(fallbackTargets)
}

// removeDuplicates removes duplicate targets from the list
func (com *CentralizedOutputManager) removeDuplicates(targets []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, target := range targets {
		if !seen[target] {
			seen[target] = true
			result = append(result, target)
		}
	}

	return result
}

// applyFallbackStrategy applies the configured fallback strategy to order targets
func (com *CentralizedOutputManager) applyFallbackStrategy(targets []string) []string {
	if len(targets) <= 1 {
		return targets
	}

	switch com.FallbackRouting.FallbackStrategy {
	case FallbackStrategySequential:
		// Return targets in original order
		return targets

	case FallbackStrategyRoundRobin:
		// Simple round-robin based on current time
		offset := int(time.Now().UnixNano()) % len(targets)
		result := make([]string, len(targets))
		for i := 0; i < len(targets); i++ {
			result[i] = targets[(i+offset)%len(targets)]
		}
		return result

	case FallbackStrategyHealthBased:
		// Sort by health (best first)
		if com.GlobalRegistry != nil {
			return com.sortTargetsByHealth(targets)
		}
		return targets

	case FallbackStrategyLoadBased:
		// Sort by load (least loaded first)
		if com.GlobalRegistry != nil {
			return com.sortTargetsByLoad(targets)
		}
		return targets

	default:
		// Default to sequential
		return targets
	}
}

// sortTargetsByHealth sorts targets by health score (highest first)
func (com *CentralizedOutputManager) sortTargetsByHealth(targets []string) []string {
	type targetHealth struct {
		target string
		health float64
	}

	targetHealths := make([]targetHealth, len(targets))
	for i, target := range targets {
		targetHealths[i] = targetHealth{
			target: target,
			health: com.GlobalRegistry.GetHealth(target),
		}
	}

	// Sort by health (highest first)
	for i := 0; i < len(targetHealths)-1; i++ {
		for j := i + 1; j < len(targetHealths); j++ {
			if targetHealths[i].health < targetHealths[j].health {
				targetHealths[i], targetHealths[j] = targetHealths[j], targetHealths[i]
			}
		}
	}

	result := make([]string, len(targets))
	for i, th := range targetHealths {
		result[i] = th.target
	}

	return result
}

// sortTargetsByLoad sorts targets by load (lowest first)
func (com *CentralizedOutputManager) sortTargetsByLoad(targets []string) []string {
	type targetLoad struct {
		target string
		load   BufferStatus
	}

	targetLoads := make([]targetLoad, len(targets))
	for i, target := range targets {
		targetLoads[i] = targetLoad{
			target: target,
			load:   com.GlobalRegistry.GetLoad(target),
		}
	}

	// Sort by load (lowest first)
	loadOrder := map[BufferStatus]int{
		BufferStatusNormal:    0,
		BufferStatusWarning:   1,
		BufferStatusHigh:      2,
		BufferStatusOverflow:  3,
		BufferStatusCritical:  4,
		BufferStatusEmergency: 5,
	}

	for i := 0; i < len(targetLoads)-1; i++ {
		for j := i + 1; j < len(targetLoads); j++ {
			if loadOrder[targetLoads[i].load] > loadOrder[targetLoads[j].load] {
				targetLoads[i], targetLoads[j] = targetLoads[j], targetLoads[i]
			}
		}
	}

	result := make([]string, len(targets))
	for i, tl := range targetLoads {
		result[i] = tl.target
	}

	return result
}

// SetFallbackRouting sets the fallback routing configuration
func (com *CentralizedOutputManager) SetFallbackRouting(config *FallbackRoutingConfig) {
	com.mutex.Lock()
	defer com.mutex.Unlock()
	com.FallbackRouting = config
}

// SetUserFlowConfig sets the user flow configuration
func (com *CentralizedOutputManager) SetUserFlowConfig(config *UserFlowConfig) {
	com.mutex.Lock()
	defer com.mutex.Unlock()
	com.UserFlowConfig = config
}

// SetRoutingRules sets the routing rules
func (com *CentralizedOutputManager) SetRoutingRules(rules map[string]string) {
	com.mutex.Lock()
	defer com.mutex.Unlock()
	com.RoutingRules = rules
}

// SetDefaultRouting sets the default routing target
func (com *CentralizedOutputManager) SetDefaultRouting(defaultTarget string) {
	com.mutex.Lock()
	defer com.mutex.Unlock()
	com.DefaultRouting = defaultTarget
}

// evaluateStepConditions evaluates the conditions in a user flow step
func (com *CentralizedOutputManager) evaluateStepConditions(step *UserFlowStep, result *engines.OperationResult) string {
	// Priority order for condition evaluation (default is handled separately as fallback)
	conditionPriority := []string{
		"success", "failure", "error", "timeout",
		"cache_hit", "cache_miss", "database_query", "cache_lookup",
		"high_priority", "low_priority", "large_data", "small_data",
		"authenticated", "unauthorized", "valid_request", "invalid_request",
	}

	// First, try to evaluate conditions based on result data
	for _, conditionName := range conditionPriority {
		if nextComponent, exists := step.Conditions[conditionName]; exists {
			if com.evaluateCondition(conditionName, result) {
				log.Printf("CentralizedOutputManager %s: Condition '%s' matched for routing to %s",
					com.InstanceID, conditionName, nextComponent)
				return nextComponent
			}
		}
	}

	// If no specific conditions matched, check for operation type match
	if nextComponent, exists := step.Conditions[result.OperationType]; exists {
		log.Printf("CentralizedOutputManager %s: Operation type '%s' matched for routing to %s",
			com.InstanceID, result.OperationType, nextComponent)
		return nextComponent
	}

	// Fall back to default if available
	if defaultComponent, exists := step.Conditions["default"]; exists {
		log.Printf("CentralizedOutputManager %s: Using default routing to %s",
			com.InstanceID, defaultComponent)
		return defaultComponent
	}

	// No matching condition found
	return ""
}

// evaluateCondition evaluates a single condition based on operation result
func (com *CentralizedOutputManager) evaluateCondition(conditionName string, result *engines.OperationResult) bool {
	switch conditionName {
	case "success":
		return result.Success

	case "failure", "error":
		return !result.Success

	case "timeout":
		// Check if processing took too long (> 1 second as example)
		return result.ProcessingTime.Seconds() > 1.0

	case "cache_hit":
		if cacheHit, exists := result.Metrics["cache_hit"]; exists {
			if hit, ok := cacheHit.(bool); ok {
				return hit
			}
		}
		return false

	case "cache_miss":
		if cacheHit, exists := result.Metrics["cache_hit"]; exists {
			if hit, ok := cacheHit.(bool); ok {
				return !hit
			}
		}
		return false

	case "database_query":
		return result.OperationType == "database_query" || result.OperationType == "read_request" || result.OperationType == "write_request"

	case "cache_lookup":
		return result.OperationType == "cache_lookup" || result.OperationType == "read_request"

	case "high_priority":
		if priority, exists := result.Metrics["priority"]; exists {
			if p, ok := priority.(int); ok {
				return p > 7
			}
			if p, ok := priority.(float64); ok {
				return p > 7.0
			}
		}
		return false

	case "low_priority":
		if priority, exists := result.Metrics["priority"]; exists {
			if p, ok := priority.(int); ok {
				return p < 3
			}
			if p, ok := priority.(float64); ok {
				return p < 3.0
			}
		}
		return false

	case "high_performance":
		if result.PenaltyInfo != nil {
			return result.PenaltyInfo.PerformanceGrade == "A" || result.PenaltyInfo.PerformanceGrade == "B"
		}
		return false

	case "low_performance":
		if result.PenaltyInfo != nil {
			return result.PenaltyInfo.PerformanceGrade == "D" || result.PenaltyInfo.PerformanceGrade == "F"
		}
		return false

	case "overloaded":
		if result.PenaltyInfo != nil {
			return result.PenaltyInfo.TotalPenaltyFactor > 2.0
		}
		return false

	case "throttled":
		if result.PenaltyInfo != nil {
			return result.PenaltyInfo.RecommendedAction == "throttle"
		}
		return false

	case "redirect_needed":
		if result.PenaltyInfo != nil {
			return result.PenaltyInfo.RecommendedAction == "redirect"
		}
		return false

	case "large_data":
		if dataSize, exists := result.Metrics["data_size"]; exists {
			if size, ok := dataSize.(int64); ok {
				return size > 1000000 // > 1MB
			}
			if size, ok := dataSize.(float64); ok {
				return size > 1000000
			}
		}
		return false

	case "small_data":
		if dataSize, exists := result.Metrics["data_size"]; exists {
			if size, ok := dataSize.(int64); ok {
				return size < 64000 // < 64KB
			}
			if size, ok := dataSize.(float64); ok {
				return size < 64000
			}
		}
		return false

	case "authenticated":
		if auth, exists := result.Metrics["authenticated"]; exists {
			if authenticated, ok := auth.(bool); ok {
				return authenticated
			}
		}
		return false

	case "unauthorized":
		if auth, exists := result.Metrics["authenticated"]; exists {
			if authenticated, ok := auth.(bool); ok {
				return !authenticated
			}
		}
		return false

	case "valid_request":
		if valid, exists := result.Metrics["valid_request"]; exists {
			if isValid, ok := valid.(bool); ok {
				return isValid
			}
		}
		return result.Success // Default to success status

	case "invalid_request":
		if valid, exists := result.Metrics["valid_request"]; exists {
			if isValid, ok := valid.(bool); ok {
				return !isValid
			}
		}
		return !result.Success // Default to failure status

	case "default":
		// Default condition always matches - it's a fallback
		return true

	default:
		// Unknown condition - return false
		log.Printf("CentralizedOutputManager %s: Unknown condition '%s'", com.InstanceID, conditionName)
		return false
	}
}
