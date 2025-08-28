package components

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// EnhancedComponentLoadBalancer implements the documented architecture
// with component graph storage and invisible/visible functionality
type EnhancedComponentLoadBalancer struct {
	// Component identification
	ComponentID   string        `json:"component_id"`
	ComponentType ComponentType `json:"component_type"`

	// Component-level graph storage (key enhancement)
	ComponentGraph *DecisionGraph `json:"component_graph"`

	// Instance management
	instances        map[string]*ComponentInstance `json:"-"`
	nextInstanceID   int                           `json:"next_instance_id"`
	instanceHealth   map[string]float64            `json:"instance_health"`

	// Load balancing configuration
	algorithm        LoadBalancingAlgorithm `json:"algorithm"`
	autoScalingConfig *AutoScalingConfig    `json:"auto_scaling_config"`

	// Load balancing state
	roundRobinIndex int `json:"round_robin_index"`

	// Visibility management (key enhancement)
	isVisible        bool   `json:"is_visible"`
	visibilityReason string `json:"visibility_reason"`

	// Communication channels
	inputChannel  chan *engines.Operation `json:"-"`
	outputChannel chan *engines.Operation `json:"-"`

	// Global registry reference
	globalRegistry GlobalRegistryInterface `json:"-"`

	// Metrics
	metrics *LoadBalancerMetrics `json:"metrics"`

	// Lifecycle management
	ctx     context.Context    `json:"-"`
	cancel  context.CancelFunc `json:"-"`
	running bool               `json:"running"`
	mutex   sync.RWMutex       `json:"-"`
}

// NewEnhancedComponentLoadBalancer creates a new enhanced component load balancer
func NewEnhancedComponentLoadBalancer(componentID string, componentType ComponentType, 
	algorithm LoadBalancingAlgorithm, autoScalingConfig *AutoScalingConfig) *EnhancedComponentLoadBalancer {
	
	return &EnhancedComponentLoadBalancer{
		ComponentID:       componentID,
		ComponentType:     componentType,
		ComponentGraph:    nil, // Will be set later
		instances:         make(map[string]*ComponentInstance),
		nextInstanceID:    1,
		instanceHealth:    make(map[string]float64),
		algorithm:         algorithm,
		autoScalingConfig: autoScalingConfig,
		roundRobinIndex:   0,
		isVisible:         false, // Start invisible
		visibilityReason:  "not_initialized",
		inputChannel:      make(chan *engines.Operation, 1000),
		outputChannel:     make(chan *engines.Operation, 1000),
		metrics: &LoadBalancerMetrics{
			TotalRequests:      0,
			SuccessfulRequests: 0,
			FailedRequests:     0,
			InstanceCount:      0,
			HealthyInstances:   0,
			UnhealthyInstances: 0,
		},
		running: false,
	}
}

// Start starts the enhanced component load balancer
func (eclb *EnhancedComponentLoadBalancer) Start(ctx context.Context) error {
	eclb.mutex.Lock()
	defer eclb.mutex.Unlock()

	if eclb.running {
		return fmt.Errorf("enhanced component load balancer %s is already running", eclb.ComponentID)
	}

	eclb.ctx, eclb.cancel = context.WithCancel(ctx)

	// Configure auto-scaling and create initial instances
	if err := eclb.configureAutoScaling(); err != nil {
		return fmt.Errorf("failed to configure auto-scaling: %w", err)
	}

	// Start all instances
	for _, instance := range eclb.instances {
		if err := instance.Start(eclb.ctx); err != nil {
			return fmt.Errorf("failed to start instance %s: %w", instance.ID, err)
		}
	}

	// Start main processing goroutine
	go eclb.run()

	eclb.running = true
	eclb.updateVisibility()

	log.Printf("EnhancedComponentLoadBalancer %s: Started with %d instances (visible: %t, reason: %s)", 
		eclb.ComponentID, len(eclb.instances), eclb.isVisible, eclb.visibilityReason)

	return nil
}

// Stop stops the enhanced component load balancer
func (eclb *EnhancedComponentLoadBalancer) Stop() error {
	eclb.mutex.Lock()
	defer eclb.mutex.Unlock()

	if !eclb.running {
		return nil
	}

	log.Printf("EnhancedComponentLoadBalancer %s: Stopping", eclb.ComponentID)

	// Cancel context to stop main goroutine
	eclb.cancel()

	// Stop all instances
	for _, instance := range eclb.instances {
		if err := instance.Stop(); err != nil {
			log.Printf("EnhancedComponentLoadBalancer %s: Error stopping instance %s: %v", 
				eclb.ComponentID, instance.ID, err)
		}
	}

	eclb.running = false
	log.Printf("EnhancedComponentLoadBalancer %s: Stopped", eclb.ComponentID)

	return nil
}

// run is the main processing loop
func (eclb *EnhancedComponentLoadBalancer) run() {
	log.Printf("EnhancedComponentLoadBalancer %s: Starting main processing loop", eclb.ComponentID)

	for {
		select {
		case operation := <-eclb.inputChannel:
			if err := eclb.processOperation(operation); err != nil {
				log.Printf("EnhancedComponentLoadBalancer %s: Error processing operation %s: %v", 
					eclb.ComponentID, operation.ID, err)
				eclb.metrics.FailedRequests++
			} else {
				eclb.metrics.TotalRequests++
				eclb.metrics.SuccessfulRequests++
			}

		case <-eclb.ctx.Done():
			log.Printf("EnhancedComponentLoadBalancer %s: Main processing loop stopping", eclb.ComponentID)
			return
		}
	}
}

// processOperation processes an operation using invisible/visible architecture
func (eclb *EnhancedComponentLoadBalancer) processOperation(operation *engines.Operation) error {
	eclb.updateVisibility()

	if !eclb.isVisible {
		// Load balancer is invisible - direct routing to single instance
		return eclb.processOperationInvisible(operation)
	}

	// Load balancer is visible - use load balancing algorithm
	return eclb.processOperationVisible(operation)
}

// processOperationInvisible handles operations when load balancer is invisible
func (eclb *EnhancedComponentLoadBalancer) processOperationInvisible(operation *engines.Operation) error {
	// No load balancing overhead - direct routing to single instance
	singleInstance := eclb.getSingleInstance()
	if singleInstance == nil {
		return fmt.Errorf("no instances available for invisible routing")
	}

	// Direct routing (O(1) operation)
	select {
	case singleInstance.InputChannel <- operation:
		log.Printf("EnhancedComponentLoadBalancer %s: Direct routed operation %s to single instance %s", 
			eclb.ComponentID, operation.ID, singleInstance.ID)
		return nil
	default:
		return fmt.Errorf("single instance %s input channel is full", singleInstance.ID)
	}
}

// processOperationVisible handles operations when load balancer is visible
func (eclb *EnhancedComponentLoadBalancer) processOperationVisible(operation *engines.Operation) error {
	// Algorithm-based instance selection (O(n) operation)
	instance, err := eclb.selectInstance(operation)
	if err != nil {
		return fmt.Errorf("failed to select instance: %w", err)
	}

	// Route to selected instance
	select {
	case instance.InputChannel <- operation:
		log.Printf("EnhancedComponentLoadBalancer %s: Load balanced operation %s to instance %s using %s algorithm", 
			eclb.ComponentID, operation.ID, instance.ID, eclb.algorithm)
		return nil
	default:
		return fmt.Errorf("instance %s input channel is full", instance.ID)
	}
}

// updateVisibility updates the visibility state based on current configuration
func (eclb *EnhancedComponentLoadBalancer) updateVisibility() {
	instanceCount := len(eclb.instances)

	if eclb.autoScalingConfig.Enabled {
		// Always visible when auto-scaling enabled (even with 1 instance)
		eclb.isVisible = true
		eclb.visibilityReason = "auto_scaling_enabled"
	} else if instanceCount > 1 {
		// Visible when multiple fixed instances
		eclb.isVisible = true
		eclb.visibilityReason = "multiple_instances"
	} else {
		// Invisible when single fixed instance
		eclb.isVisible = false
		eclb.visibilityReason = "single_instance"
	}
}

// configureAutoScaling configures auto-scaling and creates initial instances
func (eclb *EnhancedComponentLoadBalancer) configureAutoScaling() error {
	if eclb.autoScalingConfig.Mode == FixedInstances {
		// Create fixed number of instances
		for i := 0; i < eclb.autoScalingConfig.FixedInstances; i++ {
			if err := eclb.createInstance(); err != nil {
				return fmt.Errorf("failed to create fixed instance %d: %w", i, err)
			}
		}

		log.Printf("EnhancedComponentLoadBalancer %s: Configured with %d fixed instances", 
			eclb.ComponentID, eclb.autoScalingConfig.FixedInstances)
	} else {
		// Create minimum instances for auto-scaling
		for i := 0; i < eclb.autoScalingConfig.MinInstances; i++ {
			if err := eclb.createInstance(); err != nil {
				return fmt.Errorf("failed to create initial instance %d: %w", i, err)
			}
		}

		log.Printf("EnhancedComponentLoadBalancer %s: Configured with auto-scaling (min: %d, max: %d)", 
			eclb.ComponentID, eclb.autoScalingConfig.MinInstances, eclb.autoScalingConfig.MaxInstances)
	}

	return nil
}

// createInstance creates a new component instance
func (eclb *EnhancedComponentLoadBalancer) createInstance() error {
	instanceID := fmt.Sprintf("%s-instance-%d", eclb.ComponentID, eclb.nextInstanceID)
	eclb.nextInstanceID++

	// Create instance (simplified for now)
	instance := &ComponentInstance{
		ID:            instanceID,
		ComponentID:   eclb.ComponentID,
		Health:        1.0,
		InputChannel:  make(chan *engines.Operation, 100),
		OutputChannel: make(chan *engines.Operation, 100),
		Engines:       make(map[engines.EngineType]*Engine),
	}

	// Add to instances map
	eclb.instances[instanceID] = instance
	eclb.instanceHealth[instanceID] = 1.0

	// Update metrics
	eclb.metrics.InstanceCount = len(eclb.instances)
	eclb.metrics.HealthyInstances++

	log.Printf("EnhancedComponentLoadBalancer %s: Created instance %s", eclb.ComponentID, instanceID)
	return nil
}

// getSingleInstance returns the single instance when load balancer is invisible
func (eclb *EnhancedComponentLoadBalancer) getSingleInstance() *ComponentInstance {
	for _, instance := range eclb.instances {
		return instance // Return first (and only) instance
	}
	return nil
}

// selectInstance selects an instance using the configured algorithm
func (eclb *EnhancedComponentLoadBalancer) selectInstance(operation *engines.Operation) (*ComponentInstance, error) {
	switch eclb.algorithm {
	case RoundRobin:
		return eclb.roundRobinSelect()
	case HealthBased:
		return eclb.healthBasedSelect()
	case Hybrid:
		return eclb.hybridSelect()
	default:
		return eclb.roundRobinSelect() // Default to round robin
	}
}

// roundRobinSelect selects instance using round robin algorithm
func (eclb *EnhancedComponentLoadBalancer) roundRobinSelect() (*ComponentInstance, error) {
	if len(eclb.instances) == 0 {
		return nil, fmt.Errorf("no instances available")
	}

	// Convert map to slice for round robin
	instances := make([]*ComponentInstance, 0, len(eclb.instances))
	for _, instance := range eclb.instances {
		instances = append(instances, instance)
	}

	// Select using round robin
	eclb.roundRobinIndex = (eclb.roundRobinIndex + 1) % len(instances)
	return instances[eclb.roundRobinIndex], nil
}

// healthBasedSelect selects instance based on health scores
func (eclb *EnhancedComponentLoadBalancer) healthBasedSelect() (*ComponentInstance, error) {
	var bestInstance *ComponentInstance
	var bestHealth float64 = 0

	for instanceID, instance := range eclb.instances {
		health := eclb.instanceHealth[instanceID]
		if health > bestHealth && health > 0.5 {
			bestHealth = health
			bestInstance = instance
		}
	}

	if bestInstance == nil {
		// All instances unhealthy - create new one if auto-scaling enabled
		if eclb.autoScalingConfig.Enabled && len(eclb.instances) < eclb.autoScalingConfig.MaxInstances {
			if err := eclb.createInstance(); err != nil {
				return nil, fmt.Errorf("failed to create new instance: %w", err)
			}
			// Return the newly created instance
			return eclb.healthBasedSelect()
		}
		return nil, fmt.Errorf("no healthy instances available")
	}

	return bestInstance, nil
}

// hybridSelect selects instance using hybrid algorithm (health + load + connections)
func (eclb *EnhancedComponentLoadBalancer) hybridSelect() (*ComponentInstance, error) {
	var bestInstance *ComponentInstance
	var bestScore float64 = 0

	for instanceID, instance := range eclb.instances {
		// Composite score: 50% health + 30% load + 20% connections
		healthScore := eclb.instanceHealth[instanceID] * 0.5
		loadScore := (1.0 - instance.CurrentLoad) * 0.3
		connectionScore := (1.0 - float64(instance.ActiveConnections)/float64(instance.MaxConnections)) * 0.2

		totalScore := healthScore + loadScore + connectionScore

		if totalScore > bestScore {
			bestScore = totalScore
			bestInstance = instance
		}
	}

	if bestInstance == nil {
		return nil, fmt.Errorf("no instances available for hybrid selection")
	}

	return bestInstance, nil
}

// Interface implementation methods

// GetComponentGraph returns the component-level decision graph
func (eclb *EnhancedComponentLoadBalancer) GetComponentGraph() *DecisionGraph {
	eclb.mutex.RLock()
	defer eclb.mutex.RUnlock()
	return eclb.ComponentGraph
}

// SetComponentGraph sets the component-level decision graph
func (eclb *EnhancedComponentLoadBalancer) SetComponentGraph(graph *DecisionGraph) {
	eclb.mutex.Lock()
	defer eclb.mutex.Unlock()
	eclb.ComponentGraph = graph
	log.Printf("EnhancedComponentLoadBalancer %s: Component graph updated", eclb.ComponentID)
}

// GetInstances returns all component instances
func (eclb *EnhancedComponentLoadBalancer) GetInstances() map[string]*ComponentInstance {
	eclb.mutex.RLock()
	defer eclb.mutex.RUnlock()

	// Return a copy to prevent external modification
	instances := make(map[string]*ComponentInstance)
	for id, instance := range eclb.instances {
		instances[id] = instance
	}
	return instances
}

// SelectInstance selects an instance for the given operation
func (eclb *EnhancedComponentLoadBalancer) SelectInstance(operation *engines.Operation) (*ComponentInstance, error) {
	eclb.mutex.RLock()
	defer eclb.mutex.RUnlock()
	return eclb.selectInstance(operation)
}

// GetHealth returns the overall health of the load balancer
func (eclb *EnhancedComponentLoadBalancer) GetHealth() float64 {
	eclb.mutex.RLock()
	defer eclb.mutex.RUnlock()

	if len(eclb.instances) == 0 {
		return 0.0
	}

	totalHealth := 0.0
	for _, health := range eclb.instanceHealth {
		totalHealth += health
	}

	return totalHealth / float64(len(eclb.instances))
}

// GetMetrics returns current load balancer metrics
func (eclb *EnhancedComponentLoadBalancer) GetMetrics() *LoadBalancerMetrics {
	eclb.mutex.RLock()
	defer eclb.mutex.RUnlock()

	// Update instance counts
	eclb.metrics.InstanceCount = len(eclb.instances)
	eclb.metrics.HealthyInstances = 0
	eclb.metrics.UnhealthyInstances = 0

	for _, health := range eclb.instanceHealth {
		if health > 0.5 {
			eclb.metrics.HealthyInstances++
		} else {
			eclb.metrics.UnhealthyInstances++
		}
	}

	return eclb.metrics
}

// GetComponentID returns the component ID
func (eclb *EnhancedComponentLoadBalancer) GetComponentID() string {
	return eclb.ComponentID
}

// GetComponentType returns the component type
func (eclb *EnhancedComponentLoadBalancer) GetComponentType() ComponentType {
	return eclb.ComponentType
}

// IsVisible returns whether the load balancer is currently visible
func (eclb *EnhancedComponentLoadBalancer) IsVisible() bool {
	eclb.mutex.RLock()
	defer eclb.mutex.RUnlock()
	return eclb.isVisible
}

// GetVisibilityReason returns the reason for current visibility state
func (eclb *EnhancedComponentLoadBalancer) GetVisibilityReason() string {
	eclb.mutex.RLock()
	defer eclb.mutex.RUnlock()
	return eclb.visibilityReason
}

// ProcessOperation processes an operation through the load balancer
func (eclb *EnhancedComponentLoadBalancer) ProcessOperation(operation *engines.Operation) error {
	select {
	case eclb.inputChannel <- operation:
		return nil
	default:
		return fmt.Errorf("load balancer %s input channel is full", eclb.ComponentID)
	}
}
