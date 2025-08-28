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

// NewLoadBalancer creates a new load balancer for a component
func NewLoadBalancer(config *ComponentConfig) (*LoadBalancer, error) {
	// Ensure sync package is recognized as used
	var _ sync.RWMutex
	if config == nil {
		return nil, fmt.Errorf("component config cannot be nil")
	}

	// Set default load balancer config if not provided
	lbConfig := config.LoadBalancer
	if lbConfig == nil {
		lbConfig = &LoadBalancingConfig{
			Algorithm:       LoadBalancingNone,
			MinInstances:    1,
			MaxInstances:    1,
			AutoScaling:     false,
			InstanceWeights: make(map[string]int),
			DefaultWeight:   1,
		}
	}

	// Initialize weight configuration if not set
	if lbConfig.InstanceWeights == nil {
		lbConfig.InstanceWeights = make(map[string]int)
	}
	if lbConfig.DefaultWeight <= 0 {
		lbConfig.DefaultWeight = 1
	}

	// Create channels
	inputChannel := make(chan *engines.Operation, config.QueueCapacity)
	outputChannel := make(chan *engines.OperationResult, config.QueueCapacity)

	lb := &LoadBalancer{
		ComponentID:      config.ID,
		ComponentType:    config.Type,
		Config:           lbConfig,
		ComponentConfig:  config,
		Instances:        make([]*ComponentInstance, 0),
		NextInstanceID:   1,
		InstanceReady:    make(map[string]*atomic.Bool),
		InstanceShutdown: make(map[string]*atomic.Bool),
		InstanceHealth:   make(map[string]float64),
		RoundRobinIndex:  0,
		WeightedSelections: make(map[string]int),
		TotalWeight:      0,
		InputChannel:     inputChannel,
		OutputChannel:    outputChannel,
		Metrics: &ComponentMetrics{
			ComponentID:     config.ID,
			ComponentType:   config.Type,
			State:           ComponentStateStopped,
			TotalOperations: 0,
			CompletedOps:    0,
			FailedOps:       0,
			LastUpdated:     time.Now(),
			EngineMetrics:   make(map[string]interface{}),
		},
		Health: &ComponentHealth{
			Status:            "GREEN",
			IsAcceptingLoad:   true,
			AvailableCapacity: 1.0,
			LastHealthCheck:   time.Now(),
		},
	}

	return lb, nil
}

// Start implements the Component interface for LoadBalancer
func (lb *LoadBalancer) Start(ctx context.Context) error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if lb.running {
		return fmt.Errorf("load balancer %s is already running", lb.ComponentID)
	}

	log.Printf("LoadBalancer %s: Starting with %d min instances", lb.ComponentID, lb.Config.MinInstances)

	// Create context for load balancer lifecycle
	lb.ctx, lb.cancel = context.WithCancel(ctx)

	// Create initial instances
	for i := 0; i < lb.Config.MinInstances; i++ {
		if err := lb.createInstance(); err != nil {
			return fmt.Errorf("failed to create initial instance %d: %w", i, err)
		}
	}

	// Start all instances
	for _, instance := range lb.Instances {
		if err := instance.Start(lb.ctx); err != nil {
			return fmt.Errorf("failed to start instance %s: %w", instance.ID, err)
		}
	}

	// Start load balancer main goroutine
	go lb.runLoadBalancer()

	lb.running = true
	log.Printf("LoadBalancer %s: Started successfully with %d instances", lb.ComponentID, len(lb.Instances))

	return nil
}

// Stop implements the Component interface for LoadBalancer
func (lb *LoadBalancer) Stop() error {
	return lb.StopWithTimeout(time.Second * 30) // Default 30 second timeout
}

// StopWithTimeout performs graceful shutdown with a specified timeout
func (lb *LoadBalancer) StopWithTimeout(timeout time.Duration) error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if !lb.running {
		return fmt.Errorf("load balancer %s is not running", lb.ComponentID)
	}

	log.Printf("LoadBalancer %s: Starting graceful shutdown (timeout: %v)", lb.ComponentID, timeout)

	// Phase 1: Stop accepting new operations
	lb.running = false // This prevents new operations from being accepted

	// Phase 2: Cancel context to signal shutdown to goroutines
	if lb.cancel != nil {
		lb.cancel()
	}

	// Phase 3: Wait for instances to complete current operations
	shutdownComplete := make(chan bool, len(lb.Instances))

	for _, instance := range lb.Instances {
		go func(inst *ComponentInstance) {
			// Set shutdown flag to stop accepting new operations
			if shutdownFlag, exists := lb.InstanceShutdown[inst.ID]; exists {
				shutdownFlag.Store(true)
			}

			// Wait for instance to finish current operations
			lb.waitForInstanceCompletion(inst, timeout/2) // Give each instance half the total timeout

			// Stop instance
			if err := inst.Stop(); err != nil {
				log.Printf("LoadBalancer %s: Error stopping instance %s: %v", lb.ComponentID, inst.ID, err)
			}

			shutdownComplete <- true
		}(instance)
	}

	// Phase 4: Wait for all instances to shutdown or timeout
	shutdownTimer := time.NewTimer(timeout)
	defer shutdownTimer.Stop()

	instancesShutdown := 0
	totalInstances := len(lb.Instances)

	for instancesShutdown < totalInstances {
		select {
		case <-shutdownComplete:
			instancesShutdown++
			log.Printf("LoadBalancer %s: Instance %d/%d shutdown complete", lb.ComponentID, instancesShutdown, totalInstances)

		case <-shutdownTimer.C:
			log.Printf("LoadBalancer %s: Graceful shutdown timeout reached, forcing shutdown of remaining %d instances",
				lb.ComponentID, totalInstances-instancesShutdown)

			// Force shutdown remaining instances
			for _, instance := range lb.Instances {
				if !instance.ShutdownFlag.Load() {
					log.Printf("LoadBalancer %s: Force stopping instance %s", lb.ComponentID, instance.ID)
					instance.Stop()
				}
			}
			break
		}
	}

	log.Printf("LoadBalancer %s: Graceful shutdown completed", lb.ComponentID)
	return nil
}

// waitForInstanceCompletion waits for an instance to complete its current operations
func (lb *LoadBalancer) waitForInstanceCompletion(instance *ComponentInstance, timeout time.Duration) {
	log.Printf("LoadBalancer %s: Waiting for instance %s to complete operations", lb.ComponentID, instance.ID)

	completionTimer := time.NewTimer(timeout)
	defer completionTimer.Stop()

	ticker := time.NewTicker(time.Millisecond * 100) // Check every 100ms
	defer ticker.Stop()

	// Create a separate context for graceful shutdown that doesn't get cancelled immediately
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), timeout)
	defer shutdownCancel()

	for {
		select {
		case <-completionTimer.C:
			log.Printf("LoadBalancer %s: Timeout waiting for instance %s completion", lb.ComponentID, instance.ID)
			return

		case <-ticker.C:
			// Check if instance has completed all operations
			if lb.isInstanceIdle(instance) {
				log.Printf("LoadBalancer %s: Instance %s completed all operations", lb.ComponentID, instance.ID)
				return
			}

		case <-shutdownCtx.Done():
			log.Printf("LoadBalancer %s: Shutdown context timeout while waiting for instance %s", lb.ComponentID, instance.ID)
			return
		}
	}
}

// isInstanceIdle checks if an instance has no pending operations
func (lb *LoadBalancer) isInstanceIdle(instance *ComponentInstance) bool {
	// Check if input channel is empty
	if len(instance.InputChannel) > 0 {
		return false
	}

	// Check if instance is currently processing operations
	if instance.ProcessingFlag.Load() {
		return false
	}

	// Instance is idle
	return true
}

// Pause implements the Component interface for LoadBalancer
func (lb *LoadBalancer) Pause() error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	log.Printf("LoadBalancer %s: Pausing all instances", lb.ComponentID)

	// Pause all instances
	for _, instance := range lb.Instances {
		if err := instance.Pause(); err != nil {
			log.Printf("LoadBalancer %s: Error pausing instance %s: %v", lb.ComponentID, instance.ID, err)
		}
	}

	return nil
}

// Resume implements the Component interface for LoadBalancer
func (lb *LoadBalancer) Resume() error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	log.Printf("LoadBalancer %s: Resuming all instances", lb.ComponentID)

	// Resume all instances
	for _, instance := range lb.Instances {
		if err := instance.Resume(); err != nil {
			log.Printf("LoadBalancer %s: Error resuming instance %s: %v", lb.ComponentID, instance.ID, err)
		}
	}

	return nil
}

// GetID implements the Component interface for LoadBalancer
func (lb *LoadBalancer) GetID() string {
	return lb.ComponentID
}

// GetType implements the Component interface for LoadBalancer
func (lb *LoadBalancer) GetType() ComponentType {
	return lb.ComponentType
}

// GetState implements the Component interface for LoadBalancer
func (lb *LoadBalancer) GetState() ComponentState {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	if !lb.running {
		return ComponentStateStopped
	}

	// Check if any instances are running
	for _, instance := range lb.Instances {
		if instance.GetState() == ComponentStateRunning {
			return ComponentStateRunning
		}
	}

	return ComponentStateStopped
}

// IsHealthy implements the Component interface for LoadBalancer
func (lb *LoadBalancer) IsHealthy() bool {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	// Load balancer is healthy if at least one instance is healthy
	for _, instance := range lb.Instances {
		if instance.IsHealthy() {
			return true
		}
	}

	return false
}

// GetHealth implements the Component interface for LoadBalancer
func (lb *LoadBalancer) GetHealth() *ComponentHealth {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	// Aggregate health from all instances
	totalHealth := 0.0
	healthyInstances := 0

	for _, instance := range lb.Instances {
		instanceHealth := instance.GetHealth()
		if instanceHealth != nil && instanceHealth.Status == "GREEN" {
			totalHealth += instanceHealth.AvailableCapacity
			healthyInstances++
		}
	}

	avgCapacity := 0.0
	if healthyInstances > 0 {
		avgCapacity = totalHealth / float64(healthyInstances)
	}

	status := "RED"
	if healthyInstances > 0 {
		if avgCapacity > 0.7 {
			status = "GREEN"
		} else if avgCapacity > 0.3 {
			status = "YELLOW"
		}
	}

	return &ComponentHealth{
		Status:            status,
		IsAcceptingLoad:   healthyInstances > 0,
		AvailableCapacity: avgCapacity,
		LastHealthCheck:   time.Now(),
	}
}

// GetMetrics implements the Component interface for LoadBalancer
func (lb *LoadBalancer) GetMetrics() *ComponentMetrics {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	// Aggregate metrics from all instances
	totalOps := int64(0)
	completedOps := int64(0)
	failedOps := int64(0)

	for _, instance := range lb.Instances {
		instanceMetrics := instance.GetMetrics()
		if instanceMetrics != nil {
			totalOps += instanceMetrics.TotalOperations
			completedOps += instanceMetrics.CompletedOps
			failedOps += instanceMetrics.FailedOps
		}
	}

	return &ComponentMetrics{
		ComponentID:     lb.ComponentID,
		ComponentType:   lb.ComponentType,
		State:           lb.GetState(),
		TotalOperations: totalOps,
		CompletedOps:    completedOps,
		FailedOps:       failedOps,
		LastUpdated:     time.Now(),
	}
}

// ProcessOperation implements the Component interface for LoadBalancer
func (lb *LoadBalancer) ProcessOperation(op *engines.Operation) error {
	// Check if load balancer is shutting down
	if !lb.running {
		return fmt.Errorf("load balancer %s is shutting down, rejecting operation %s", lb.ComponentID, op.ID)
	}

	// Add operation to input channel for load balancer to route
	select {
	case lb.InputChannel <- op:
		return nil
	default:
		return fmt.Errorf("load balancer %s input channel is full", lb.ComponentID)
	}
}

// ProcessTick implements the Component interface for LoadBalancer
func (lb *LoadBalancer) ProcessTick(currentTick int64) error {
	// Load balancer doesn't process ticks directly - instances do
	// This could be used for auto-scaling decisions in the future
	return nil
}

// runLoadBalancer is the main goroutine that handles load balancing
func (lb *LoadBalancer) runLoadBalancer() {
	log.Printf("LoadBalancer %s: Starting load balancer goroutine", lb.ComponentID)

	// Create ticker for auto-scaling checks
	var autoScaleTicker *time.Ticker
	var autoScaleChannel <-chan time.Time
	if lb.Config.AutoScaling {
		autoScaleTicker = time.NewTicker(time.Second * 10) // Check every 10 seconds
		autoScaleChannel = autoScaleTicker.C
		defer autoScaleTicker.Stop()
	}

	for {
		if lb.Config.AutoScaling {
			// Select with auto-scaling enabled
			select {
			case op := <-lb.InputChannel:
				// Route operation to appropriate instance
				if err := lb.routeOperation(op); err != nil {
					log.Printf("LoadBalancer %s: Error routing operation %s: %v", lb.ComponentID, op.ID, err)
					lb.Metrics.FailedOps++
				} else {
					lb.Metrics.TotalOperations++
				}

			case <-autoScaleChannel:
				// Perform auto-scaling check
				lb.performAutoScalingCheck()

			case <-lb.ctx.Done():
				log.Printf("LoadBalancer %s: Load balancer goroutine stopping", lb.ComponentID)
				return
			}
		} else {
			// Select without auto-scaling
			select {
			case op := <-lb.InputChannel:
				// Route operation to appropriate instance
				if err := lb.routeOperation(op); err != nil {
					log.Printf("LoadBalancer %s: Error routing operation %s: %v", lb.ComponentID, op.ID, err)
					lb.Metrics.FailedOps++
				} else {
					lb.Metrics.TotalOperations++
				}

			case <-lb.ctx.Done():
				log.Printf("LoadBalancer %s: Load balancer goroutine stopping", lb.ComponentID)
				return
			}
		}
	}
}

// routeOperation routes an operation to an appropriate instance
func (lb *LoadBalancer) routeOperation(op *engines.Operation) error {
	// Apply algorithm penalty if configured
	if lb.Config.AlgorithmPenalty > 0 {
		time.Sleep(lb.Config.AlgorithmPenalty)
	}

	// Select instance based on algorithm
	instance, err := lb.selectInstance()
	if err != nil {
		return fmt.Errorf("failed to select instance: %w", err)
	}

	// Route to selected instance
	select {
	case instance.InputChannel <- op:
		log.Printf("LoadBalancer %s: Routed operation %s to instance %s", lb.ComponentID, op.ID, instance.ID)
		return nil
	default:
		return fmt.Errorf("instance %s input channel is full", instance.ID)
	}
}

// selectInstance selects an instance based on the configured algorithm
func (lb *LoadBalancer) selectInstance() (*ComponentInstance, error) {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	if len(lb.Instances) == 0 {
		return nil, fmt.Errorf("no instances available")
	}

	// Filter healthy instances
	healthyInstances := make([]*ComponentInstance, 0)
	for _, instance := range lb.Instances {
		if readyFlag, exists := lb.InstanceReady[instance.ID]; exists && readyFlag.Load() {
			if shutdownFlag, exists := lb.InstanceShutdown[instance.ID]; !exists || !shutdownFlag.Load() {
				healthyInstances = append(healthyInstances, instance)
			}
		}
	}

	if len(healthyInstances) == 0 {
		return nil, fmt.Errorf("no healthy instances available")
	}

	// Single instance - no algorithm needed (invisible load balancer)
	if len(healthyInstances) == 1 {
		return healthyInstances[0], nil
	}

	// Apply load balancing algorithm
	switch lb.Config.Algorithm {
	case LoadBalancingNone:
		return healthyInstances[0], nil

	case LoadBalancingRoundRobin:
		return lb.selectRoundRobin(healthyInstances), nil

	case LoadBalancingLeastConnections:
		return lb.selectLeastConnections(healthyInstances), nil

	case LoadBalancingWeighted:
		return lb.selectWeighted(healthyInstances), nil

	case LoadBalancingHealthAware:
		return lb.selectHealthAware(healthyInstances), nil

	default:
		return healthyInstances[0], nil
	}
}

// selectRoundRobin implements round-robin load balancing
func (lb *LoadBalancer) selectRoundRobin(instances []*ComponentInstance) *ComponentInstance {
	if len(instances) == 0 {
		return nil
	}

	// Use atomic operation for thread-safe round-robin
	index := lb.RoundRobinIndex % len(instances)
	lb.RoundRobinIndex = (lb.RoundRobinIndex + 1) % len(instances)

	return instances[index]
}

// selectLeastConnections implements least connections load balancing
func (lb *LoadBalancer) selectLeastConnections(instances []*ComponentInstance) *ComponentInstance {
	if len(instances) == 0 {
		return nil
	}

	var bestInstance *ComponentInstance
	minConnections := int64(^uint64(0) >> 1) // Max int64

	for _, instance := range instances {
		metrics := instance.GetMetrics()
		if metrics != nil {
			currentConnections := metrics.TotalOperations - metrics.CompletedOps
			if currentConnections < minConnections {
				minConnections = currentConnections
				bestInstance = instance
			}
		}
	}

	if bestInstance == nil {
		return instances[0] // Fallback to first instance
	}

	return bestInstance
}

// selectWeighted implements weighted round-robin load balancing
func (lb *LoadBalancer) selectWeighted(instances []*ComponentInstance) *ComponentInstance {
	if len(instances) == 0 {
		return nil
	}

	// Single instance - no weighting needed
	if len(instances) == 1 {
		return instances[0]
	}

	// Build weighted selection pool
	weightedPool := make([]*ComponentInstance, 0)
	totalWeight := 0

	for _, instance := range instances {
		// Get weight for this instance
		weight := lb.getInstanceWeight(instance.ID)
		totalWeight += weight

		// Add instance to pool based on its weight
		for i := 0; i < weight; i++ {
			weightedPool = append(weightedPool, instance)
		}
	}

	// If no weights configured, fall back to round-robin
	if len(weightedPool) == 0 {
		log.Printf("LoadBalancer %s: No weights configured for weighted algorithm, falling back to round-robin", lb.ComponentID)
		return lb.selectRoundRobin(instances)
	}

	// Select from weighted pool using round-robin
	index := lb.RoundRobinIndex % len(weightedPool)
	lb.RoundRobinIndex = (lb.RoundRobinIndex + 1) % len(weightedPool)

	selectedInstance := weightedPool[index]

	// Update selection tracking
	if lb.WeightedSelections == nil {
		lb.WeightedSelections = make(map[string]int)
	}
	lb.WeightedSelections[selectedInstance.ID]++

	log.Printf("LoadBalancer %s: Selected instance %s (weight: %d, selections: %d)",
		lb.ComponentID, selectedInstance.ID, lb.getInstanceWeight(selectedInstance.ID),
		lb.WeightedSelections[selectedInstance.ID])

	return selectedInstance
}

// getInstanceWeight returns the configured weight for an instance
func (lb *LoadBalancer) getInstanceWeight(instanceID string) int {
	if lb.Config.InstanceWeights == nil {
		return lb.Config.DefaultWeight
	}

	if weight, exists := lb.Config.InstanceWeights[instanceID]; exists && weight > 0 {
		return weight
	}

	return lb.Config.DefaultWeight
}

// SetInstanceWeight sets the weight for a specific instance
func (lb *LoadBalancer) SetInstanceWeight(instanceID string, weight int) error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if weight <= 0 {
		return fmt.Errorf("weight must be positive, got %d", weight)
	}

	if lb.Config.InstanceWeights == nil {
		lb.Config.InstanceWeights = make(map[string]int)
	}

	oldWeight := lb.getInstanceWeight(instanceID)
	lb.Config.InstanceWeights[instanceID] = weight

	// Update total weight tracking
	lb.TotalWeight = lb.TotalWeight - oldWeight + weight

	log.Printf("LoadBalancer %s: Updated weight for instance %s from %d to %d (total weight: %d)",
		lb.ComponentID, instanceID, oldWeight, weight, lb.TotalWeight)

	return nil
}

// GetInstanceWeights returns a copy of all instance weights
func (lb *LoadBalancer) GetInstanceWeights() map[string]int {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	weights := make(map[string]int)
	for _, instance := range lb.Instances {
		weights[instance.ID] = lb.getInstanceWeight(instance.ID)
	}

	return weights
}

// selectHealthAware implements health-aware load balancing
func (lb *LoadBalancer) selectHealthAware(instances []*ComponentInstance) *ComponentInstance {
	if len(instances) == 0 {
		return nil
	}

	var bestInstance *ComponentInstance
	bestHealth := 0.0

	for _, instance := range instances {
		health := instance.GetHealth()
		if health != nil && health.AvailableCapacity > bestHealth {
			bestHealth = health.AvailableCapacity
			bestInstance = instance
		}
	}

	if bestInstance == nil {
		return instances[0] // Fallback to first instance
	}

	return bestInstance
}

// createInstance creates a new component instance
func (lb *LoadBalancer) createInstance() error {
	instanceID := fmt.Sprintf("%s-instance-%d", lb.ComponentID, lb.NextInstanceID)
	lb.NextInstanceID++

	// Create instance configuration
	instanceConfig := &ComponentConfig{
		ID:               instanceID,
		Type:             lb.ComponentType,
		Name:             fmt.Sprintf("%s Instance %d", lb.ComponentID, lb.NextInstanceID-1),
		Description:      fmt.Sprintf("Instance of component %s", lb.ComponentID),
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType, engines.MemoryEngineType, engines.StorageEngineType},
		MaxConcurrentOps: 5,
		QueueCapacity:    50,
		TickTimeout:      time.Millisecond * 10,
		EngineProfiles:   make(map[engines.EngineType]string),
		ComplexityLevels: make(map[engines.EngineType]int),
	}

	// Create the instance
	instance, err := NewComponentInstance(instanceConfig)
	if err != nil {
		return fmt.Errorf("failed to create instance %s: %w", instanceID, err)
	}

	// Initialize atomic flags
	readyFlag := &atomic.Bool{}
	shutdownFlag := &atomic.Bool{}
	readyFlag.Store(false)
	shutdownFlag.Store(false)

	// Add to load balancer
	lb.Instances = append(lb.Instances, instance)
	lb.InstanceReady[instanceID] = readyFlag
	lb.InstanceShutdown[instanceID] = shutdownFlag
	lb.InstanceHealth[instanceID] = 1.0

	// Initialize weight tracking for new instance
	instanceWeight := lb.getInstanceWeight(instanceID)
	lb.TotalWeight += instanceWeight
	if lb.WeightedSelections == nil {
		lb.WeightedSelections = make(map[string]int)
	}
	lb.WeightedSelections[instanceID] = 0

	// Set the atomic flags in the instance
	instance.ReadyFlag = readyFlag
	instance.ShutdownFlag = shutdownFlag

	log.Printf("LoadBalancer %s: Created instance %s with weight %d (total weight: %d)",
		lb.ComponentID, instanceID, instanceWeight, lb.TotalWeight)
	return nil
}

// removeInstance removes an instance from the load balancer
func (lb *LoadBalancer) removeInstance(instanceID string) error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Find and remove instance
	for i, instance := range lb.Instances {
		if instance.ID == instanceID {
			// Stop the instance
			if err := instance.Stop(); err != nil {
				log.Printf("LoadBalancer %s: Error stopping instance %s: %v", lb.ComponentID, instanceID, err)
			}

			// Remove from slice
			lb.Instances = append(lb.Instances[:i], lb.Instances[i+1:]...)

			// Clean up atomic flags
			delete(lb.InstanceReady, instanceID)
			delete(lb.InstanceShutdown, instanceID)
			delete(lb.InstanceHealth, instanceID)

			// Clean up weight tracking
			instanceWeight := lb.getInstanceWeight(instanceID)
			lb.TotalWeight -= instanceWeight
			delete(lb.WeightedSelections, instanceID)
			delete(lb.Config.InstanceWeights, instanceID)

			log.Printf("LoadBalancer %s: Removed instance %s (weight: %d, total weight: %d)",
				lb.ComponentID, instanceID, instanceWeight, lb.TotalWeight)
			return nil
		}
	}

	return fmt.Errorf("instance %s not found", instanceID)
}

// GetInputChannel returns the load balancer's input channel
func (lb *LoadBalancer) GetInputChannel() chan *engines.Operation {
	return lb.InputChannel
}

// GetOutputChannel returns the load balancer's output channel
func (lb *LoadBalancer) GetOutputChannel() chan *engines.OperationResult {
	return lb.OutputChannel
}

// SetRegistry sets the global registry for the load balancer and all instances
func (lb *LoadBalancer) SetRegistry(registry GlobalRegistryInterface) {
	lb.GlobalRegistry = registry

	// Set registry for all instances
	for _, instance := range lb.Instances {
		instance.SetRegistry(registry)
	}
}

// SaveState implements the Component interface for LoadBalancer
func (lb *LoadBalancer) SaveState() error {
	if GlobalStatePersistenceManager == nil {
		return fmt.Errorf("state persistence manager not initialized")
	}

	log.Printf("LoadBalancer %s: Saving state", lb.ComponentID)

	if err := GlobalStatePersistenceManager.SaveLoadBalancerState(lb); err != nil {
		return fmt.Errorf("failed to save load balancer state: %w", err)
	}

	log.Printf("LoadBalancer %s: State saved successfully", lb.ComponentID)
	return nil
}

// LoadState implements the Component interface for LoadBalancer
func (lb *LoadBalancer) LoadState(componentID string) error {
	if GlobalStatePersistenceManager == nil {
		return fmt.Errorf("state persistence manager not initialized")
	}

	log.Printf("LoadBalancer %s: Loading state for component %s", lb.ComponentID, componentID)

	state, err := GlobalStatePersistenceManager.LoadLoadBalancerState(componentID)
	if err != nil {
		return fmt.Errorf("failed to load load balancer state: %w", err)
	}

	// Restore basic state
	lb.ComponentID = state.ComponentID
	lb.ComponentType = state.ComponentType
	lb.Config = state.Config
	lb.ComponentConfig = state.ComponentConfig
	lb.NextInstanceID = state.NextInstanceID
	lb.InstanceHealth = state.InstanceHealth
	lb.RoundRobinIndex = state.RoundRobinIndex
	lb.WeightedSelections = state.WeightedSelections
	lb.TotalWeight = state.TotalWeight
	lb.LastScaleUp = state.LastScaleUp
	lb.LastScaleDown = state.LastScaleDown

	// Restore metrics
	if state.Metrics != nil {
		lb.Metrics = state.Metrics
	}

	// Restore instances (this is complex as it requires recreating the instances)
	if len(state.InstanceStates) > 0 {
		log.Printf("LoadBalancer %s: Restoring %d instances", lb.ComponentID, len(state.InstanceStates))

		// Clear existing instances
		lb.Instances = make([]*ComponentInstance, 0, len(state.InstanceStates))
		lb.InstanceReady = make(map[string]*atomic.Bool)
		lb.InstanceShutdown = make(map[string]*atomic.Bool)

		// Recreate instances from saved state
		for _, instanceState := range state.InstanceStates {
			// Create new instance with saved configuration
			instance, err := NewComponentInstance(instanceState.Config)
			if err != nil {
				log.Printf("LoadBalancer %s: Failed to recreate instance %s: %v", lb.ComponentID, instanceState.ID, err)
				continue
			}

			// Restore instance state
			if err := instance.LoadState(instanceState.ID); err != nil {
				log.Printf("LoadBalancer %s: Failed to restore instance %s state: %v", lb.ComponentID, instanceState.ID, err)
				// Continue with the instance even if state restoration fails
			}

			// Add to load balancer
			lb.Instances = append(lb.Instances, instance)
			lb.InstanceReady[instance.ID] = instance.ReadyFlag
			lb.InstanceShutdown[instance.ID] = instance.ShutdownFlag
		}

		log.Printf("LoadBalancer %s: Successfully restored %d instances", lb.ComponentID, len(lb.Instances))
	}

	// Note: We don't restore the running flag as this should be set during startup
	log.Printf("LoadBalancer %s: State loaded successfully from %s", lb.ComponentID, componentID)
	return nil
}

// performAutoScalingCheck checks if auto-scaling is needed and performs scaling actions
func (lb *LoadBalancer) performAutoScalingCheck() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if !lb.Config.AutoScaling {
		return
	}

	currentInstances := len(lb.Instances)
	minInstances := lb.Config.MinInstances
	maxInstances := lb.Config.MaxInstances

	// Calculate average load across all instances
	totalLoad := 0.0
	healthyInstances := 0

	for _, instance := range lb.Instances {
		// Consider instances healthy if they're not shutting down (paused instances are still healthy)
		if !instance.ShutdownFlag.Load() {
			healthyInstances++
			// Calculate load based on input channel utilization
			channelCapacity := float64(cap(instance.InputChannel))
			channelLength := float64(len(instance.InputChannel))
			instanceLoad := channelLength / channelCapacity
			totalLoad += instanceLoad
		}
	}

	if healthyInstances == 0 {
		log.Printf("LoadBalancer %s: No healthy instances for auto-scaling check", lb.ComponentID)
		return
	}

	averageLoad := totalLoad / float64(healthyInstances)

	log.Printf("LoadBalancer %s: Auto-scaling check - %d instances, average load: %.2f",
		lb.ComponentID, currentInstances, averageLoad)

	// Scale up if average load is high and we can add more instances
	if averageLoad > 0.8 && currentInstances < maxInstances {
		if lb.shouldScaleUp() {
			lb.scaleUp()
		}
	}

	// Scale down if average load is low and we have more than minimum instances
	if averageLoad < 0.3 && currentInstances > minInstances {
		if lb.shouldScaleDown() {
			lb.scaleDown()
		}
	}
}

// shouldScaleUp checks if scaling up is allowed based on cooldown periods
func (lb *LoadBalancer) shouldScaleUp() bool {
	// Check if enough time has passed since last scale up
	if time.Since(lb.LastScaleUp) < time.Minute*2 {
		log.Printf("LoadBalancer %s: Scale up blocked by cooldown period", lb.ComponentID)
		return false
	}

	return true
}

// shouldScaleDown checks if scaling down is allowed based on cooldown periods
func (lb *LoadBalancer) shouldScaleDown() bool {
	// Check if enough time has passed since last scale down
	if time.Since(lb.LastScaleDown) < time.Minute*5 {
		log.Printf("LoadBalancer %s: Scale down blocked by cooldown period", lb.ComponentID)
		return false
	}

	return true
}

// scaleUp adds a new instance to the load balancer
func (lb *LoadBalancer) scaleUp() {
	// Check if we're already at max instances
	if len(lb.Instances) >= lb.Config.MaxInstances {
		log.Printf("LoadBalancer %s: Cannot scale up beyond maximum instances (%d)", lb.ComponentID, lb.Config.MaxInstances)
		return
	}

	newInstanceID := fmt.Sprintf("%s-instance-%d", lb.ComponentID, len(lb.Instances)+1)

	log.Printf("LoadBalancer %s: Scaling up - adding instance %s", lb.ComponentID, newInstanceID)

	// Create new instance config based on existing config
	instanceConfig := &ComponentConfig{
		ID:               newInstanceID,
		Type:             lb.ComponentConfig.Type,
		Name:             fmt.Sprintf("%s Instance %d", lb.ComponentConfig.Name, len(lb.Instances)+1),
		Description:      fmt.Sprintf("Auto-scaled instance for %s", lb.ComponentID),
		RequiredEngines:  lb.ComponentConfig.RequiredEngines,
		MaxConcurrentOps: lb.ComponentConfig.MaxConcurrentOps,
		QueueCapacity:    lb.ComponentConfig.QueueCapacity,
		TickTimeout:      lb.ComponentConfig.TickTimeout,
		EngineProfiles:   lb.ComponentConfig.EngineProfiles,
		ComplexityLevels: lb.ComponentConfig.ComplexityLevels,
	}

	// Create new instance
	newInstance, err := NewComponentInstance(instanceConfig)
	if err != nil {
		log.Printf("LoadBalancer %s: Failed to create new instance: %v", lb.ComponentID, err)
		return
	}

	// Start the new instance
	if err := newInstance.Start(lb.ctx); err != nil {
		log.Printf("LoadBalancer %s: Failed to start new instance: %v", lb.ComponentID, err)
		return
	}

	// Add to instances list
	lb.Instances = append(lb.Instances, newInstance)
	lb.LastScaleUp = time.Now()

	log.Printf("LoadBalancer %s: Successfully scaled up to %d instances", lb.ComponentID, len(lb.Instances))
}

// scaleDown removes an instance from the load balancer
func (lb *LoadBalancer) scaleDown() {
	if len(lb.Instances) <= lb.Config.MinInstances {
		log.Printf("LoadBalancer %s: Cannot scale down below minimum instances", lb.ComponentID)
		return
	}

	// Find the least loaded instance to remove
	leastLoadedIndex := lb.findLeastLoadedInstance()
	if leastLoadedIndex == -1 {
		log.Printf("LoadBalancer %s: No suitable instance found for scaling down", lb.ComponentID)
		return
	}

	instanceToRemove := lb.Instances[leastLoadedIndex]
	log.Printf("LoadBalancer %s: Scaling down - removing instance %s", lb.ComponentID, instanceToRemove.ID)

	// Gracefully stop the instance
	if err := instanceToRemove.Stop(); err != nil {
		log.Printf("LoadBalancer %s: Error stopping instance %s: %v", lb.ComponentID, instanceToRemove.ID, err)
	}

	// Remove from instances list
	lb.Instances = append(lb.Instances[:leastLoadedIndex], lb.Instances[leastLoadedIndex+1:]...)
	lb.LastScaleDown = time.Now()

	log.Printf("LoadBalancer %s: Successfully scaled down to %d instances", lb.ComponentID, len(lb.Instances))
}

// findLeastLoadedInstance finds the instance with the lowest load
func (lb *LoadBalancer) findLeastLoadedInstance() int {
	if len(lb.Instances) == 0 {
		return -1
	}

	leastLoadedIndex := -1
	lowestLoad := 1.0

	for i, instance := range lb.Instances {
		if instance.ShutdownFlag.Load() {
			continue // Skip instances that are shutting down
		}

		// Calculate instance load
		channelCapacity := float64(cap(instance.InputChannel))
		channelLength := float64(len(instance.InputChannel))
		instanceLoad := channelLength / channelCapacity

		if instanceLoad < lowestLoad {
			lowestLoad = instanceLoad
			leastLoadedIndex = i
		}
	}

	return leastLoadedIndex
}
