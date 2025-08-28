package components

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// NewComponentInstance creates a new component instance
func NewComponentInstance(config *ComponentConfig) (*ComponentInstance, error) {
	if config == nil {
		return nil, fmt.Errorf("component config cannot be nil")
	}

	// Create channels
	inputChannel := make(chan *engines.Operation, config.QueueCapacity)
	outputChannel := make(chan *engines.OperationResult, config.QueueCapacity)
	internalChannels := make(map[engines.EngineType]chan *engines.Operation)

	// Create internal channels for each engine
	for _, engineType := range config.RequiredEngines {
		internalChannels[engineType] = make(chan *engines.Operation, config.QueueCapacity)
	}

	// Initialize atomic flags
	readyFlag := &atomic.Bool{}
	shutdownFlag := &atomic.Bool{}
	processingFlag := &atomic.Bool{}

	readyFlag.Store(false)
	shutdownFlag.Store(false)
	processingFlag.Store(false)

	// Create engine router
	engineRouter := NewEngineRouter(config.Type, config.RequiredEngines)

	instance := &ComponentInstance{
		ID:               config.ID,
		ComponentID:      config.ID,
		ComponentType:    config.Type,
		Config:           config,
		ReadyFlag:        readyFlag,
		ShutdownFlag:     shutdownFlag,
		ProcessingFlag:   processingFlag,
		Engines:          make(map[engines.EngineType]*engines.EngineWrapper),
		EngineGoroutines: make(map[engines.EngineType]GoroutineTracker),
		EngineRouter:     engineRouter,
		InputChannel:     inputChannel,
		OutputChannel:    outputChannel,
		InternalChannels: internalChannels,
		Health: &ComponentHealth{
			Status:            "GREEN", // Start as healthy
			IsAcceptingLoad:   true,    // Accept load when ready
			AvailableCapacity: 1.0,     // Full capacity initially
			LastHealthCheck:   time.Now(),
		},
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
		ErrorHandler: NewErrorHandler(),
		StartTime:    time.Now(),
	}

	// Initialize engines (placeholder for now)
	if err := instance.initializeEngines(); err != nil {
		return nil, fmt.Errorf("failed to initialize engines: %w", err)
	}

	// Create decision graph (placeholder for now)
	instance.DecisionGraph = &DecisionGraph{
		StartNode: "input_network",
		EndNodes:  []string{"output_network"},
		Nodes:     make(map[string]*DecisionNode),
		Engines:   instance.Engines,
	}

	// Create centralized output manager
	instance.CentralizedOutput = &CentralizedOutputManager{
		InstanceID:         instance.ID,
		ComponentID:        instance.ComponentID,
		InputChannel:       make(chan *engines.OperationResult, config.QueueCapacity),
		OutputChannel:      outputChannel,
		RoutingRules:       config.RoutingRules,
		BackpressureConfig: &BackpressureConfig{
			MaxRetries:              3,
			RetryDelay:              time.Millisecond * 10,
			CircuitBreakerThreshold: 5,
			HealthCheckInterval:     time.Second * 5,
		},
		CircuitBreakerManager: NewCircuitBreakerManager(DefaultCircuitBreakerConfig()),
	}

	return instance, nil
}

// Start implements the Component interface for ComponentInstance
func (ci *ComponentInstance) Start(ctx context.Context) error {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	if ci.running {
		return fmt.Errorf("component instance %s is already running", ci.ID)
	}

	log.Printf("ComponentInstance %s: Starting instance", ci.ID)

	// Create context for instance lifecycle
	ci.ctx, ci.cancel = context.WithCancel(ctx)

	// Start independent engine goroutines
	for engineType, engine := range ci.Engines {
		if engine != nil {
			// Start each engine in its own goroutine
			if err := engine.Start(ci.ctx); err != nil {
				log.Printf("ComponentInstance %s: Failed to start engine %s: %v", ci.ID, engineType, err)
				// Continue with other engines even if one fails
			} else {
				log.Printf("ComponentInstance %s: Started independent goroutine for engine %s", ci.ID, engineType)

				// Update goroutine tracker
				if tracker, exists := ci.EngineGoroutines[engineType]; exists {
					tracker.LastActivity = time.Now()
					ci.EngineGoroutines[engineType] = tracker
				}
			}
		} else {
			log.Printf("ComponentInstance %s: Skipping start of nil engine %s", ci.ID, engineType)
		}
	}

	// Start centralized output manager
	if ci.CentralizedOutput != nil {
		go ci.CentralizedOutput.Run(ci.ctx)
	}

	// Start main instance goroutine
	go ci.runInstance()

	// Mark as ready
	ci.ReadyFlag.Store(true)
	ci.running = true

	log.Printf("ComponentInstance %s: Started successfully", ci.ID)
	return nil
}

// Stop implements the Component interface for ComponentInstance
func (ci *ComponentInstance) Stop() error {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	if !ci.running {
		return fmt.Errorf("component instance %s is not running", ci.ID)
	}

	log.Printf("ComponentInstance %s: Stopping instance", ci.ID)

	// Mark as not ready
	ci.ReadyFlag.Store(false)

	// Cancel context to signal shutdown
	if ci.cancel != nil {
		ci.cancel()
	}

	// Stop independent engine goroutines
	for engineType, engine := range ci.Engines {
		if engine != nil {
			// Stop each engine goroutine
			if err := engine.Stop(); err != nil {
				log.Printf("ComponentInstance %s: Error stopping engine %s: %v", ci.ID, engineType, err)
			} else {
				log.Printf("ComponentInstance %s: Stopped independent goroutine for engine %s", ci.ID, engineType)
			}
		} else {
			log.Printf("ComponentInstance %s: Skipping stop of nil engine %s", ci.ID, engineType)
		}
	}

	ci.running = false
	log.Printf("ComponentInstance %s: Stopped successfully", ci.ID)

	return nil
}

// Pause implements the Component interface for ComponentInstance
func (ci *ComponentInstance) Pause() error {
	log.Printf("ComponentInstance %s: Pausing instance", ci.ID)

	// Mark as not ready to receive new operations
	ci.ReadyFlag.Store(false)

	// Pause independent engine goroutines
	for engineType, engine := range ci.Engines {
		if engine != nil {
			// Pause each engine goroutine
			if err := engine.Pause(); err != nil {
				log.Printf("ComponentInstance %s: Error pausing engine %s: %v", ci.ID, engineType, err)
			} else {
				log.Printf("ComponentInstance %s: Paused independent goroutine for engine %s", ci.ID, engineType)
			}
		}
	}

	return nil
}

// Resume implements the Component interface for ComponentInstance
func (ci *ComponentInstance) Resume() error {
	log.Printf("ComponentInstance %s: Resuming instance", ci.ID)

	// Resume independent engine goroutines
	for engineType, engine := range ci.Engines {
		if engine != nil {
			// Resume each engine goroutine
			if err := engine.Resume(); err != nil {
				log.Printf("ComponentInstance %s: Error resuming engine %s: %v", ci.ID, engineType, err)
			} else {
				log.Printf("ComponentInstance %s: Resumed independent goroutine for engine %s", ci.ID, engineType)
			}
		}
	}

	// Mark as ready to receive operations
	ci.ReadyFlag.Store(true)

	return nil
}

// GetID implements the Component interface for ComponentInstance
func (ci *ComponentInstance) GetID() string {
	return ci.ID
}

// GetType implements the Component interface for ComponentInstance
func (ci *ComponentInstance) GetType() ComponentType {
	return ci.ComponentType
}

// GetState implements the Component interface for ComponentInstance
func (ci *ComponentInstance) GetState() ComponentState {
	ci.mutex.RLock()
	defer ci.mutex.RUnlock()

	if !ci.running {
		return ComponentStateStopped
	}

	if ci.ReadyFlag.Load() {
		return ComponentStateRunning
	}

	return ComponentStatePaused
}

// IsHealthy implements the Component interface for ComponentInstance
func (ci *ComponentInstance) IsHealthy() bool {
	return ci.Health != nil && ci.Health.Status == "GREEN"
}

// GetHealth implements the Component interface for ComponentInstance
func (ci *ComponentInstance) GetHealth() *ComponentHealth {
	return ci.Health
}

// GetMetrics implements the Component interface for ComponentInstance
func (ci *ComponentInstance) GetMetrics() *ComponentMetrics {
	return ci.Metrics
}

// ProcessOperation implements the Component interface for ComponentInstance
func (ci *ComponentInstance) ProcessOperation(op *engines.Operation) error {
	if !ci.ReadyFlag.Load() {
		err := fmt.Errorf("component instance %s is not ready", ci.ID)
		compErr := WrapError(err, ci.ID, op.ID)
		compErr.Category = ErrorCategoryResource
		compErr.Severity = ErrorSeverityMedium
		ci.ErrorHandler.HandleError(context.Background(), compErr, ci.ID)
		return compErr
	}

	// Add operation to input channel
	select {
	case ci.InputChannel <- op:
		log.Printf("ComponentInstance %s: Accepted operation %s", ci.ID, op.ID)
		return nil
	default:
		err := fmt.Errorf("component instance %s input channel is full", ci.ID)
		compErr := WrapError(err, ci.ID, op.ID)
		compErr.Category = ErrorCategoryResource
		compErr.Severity = ErrorSeverityHigh
		ci.ErrorHandler.HandleError(context.Background(), compErr, ci.ID)
		return compErr
	}
}

// ProcessTick implements the Component interface for ComponentInstance
func (ci *ComponentInstance) ProcessTick(currentTick int64) error {
	// Instance doesn't process ticks directly - engines do
	// This could be used for health monitoring in the future
	ci.updateComponentState()
	return nil
}

// GetInputChannel returns the instance's input channel
func (ci *ComponentInstance) GetInputChannel() chan *engines.Operation {
	return ci.InputChannel
}

// GetOutputChannel returns the instance's output channel
func (ci *ComponentInstance) GetOutputChannel() chan *engines.OperationResult {
	return ci.OutputChannel
}

// SetRegistry sets the global registry for the instance
func (ci *ComponentInstance) SetRegistry(registry GlobalRegistryInterface) {
	if ci.CentralizedOutput != nil {
		ci.CentralizedOutput.GlobalRegistry = registry
	}
}

// SaveState implements the Component interface for ComponentInstance
func (ci *ComponentInstance) SaveState() error {
	if GlobalStatePersistenceManager == nil {
		return fmt.Errorf("state persistence manager not initialized")
	}

	log.Printf("ComponentInstance %s: Saving state", ci.ID)

	if err := GlobalStatePersistenceManager.SaveComponentInstanceState(ci); err != nil {
		return fmt.Errorf("failed to save component instance state: %w", err)
	}

	log.Printf("ComponentInstance %s: State saved successfully", ci.ID)
	return nil
}

// LoadState implements the Component interface for ComponentInstance
func (ci *ComponentInstance) LoadState(instanceID string) error {
	if GlobalStatePersistenceManager == nil {
		return fmt.Errorf("state persistence manager not initialized")
	}

	log.Printf("ComponentInstance %s: Loading state for instance %s", ci.ID, instanceID)

	state, err := GlobalStatePersistenceManager.LoadComponentInstanceState(instanceID)
	if err != nil {
		return fmt.Errorf("failed to load component instance state: %w", err)
	}

	// Restore basic state
	ci.ID = state.ID
	ci.ComponentID = state.ComponentID
	ci.ComponentType = state.ComponentType
	ci.Config = state.Config
	ci.StartTime = state.StartTime
	ci.LastTickTime = state.LastTick

	// Restore health and metrics
	if state.Health != nil {
		ci.Health = state.Health
	}
	if state.Metrics != nil {
		ci.Metrics = state.Metrics
	}

	// Restore engine states
	for engineType, engineState := range state.EngineStates {
		if wrapper, exists := ci.Engines[engineType]; exists && wrapper != nil {
			if err := wrapper.LoadState(engineState.EngineID); err != nil {
				log.Printf("ComponentInstance %s: Failed to restore engine %s state: %v", ci.ID, engineType, err)
				// Continue with other engines rather than failing completely
			} else {
				log.Printf("ComponentInstance %s: Restored engine %s state", ci.ID, engineType)
			}
		}
	}

	// Note: We don't restore running/ready/processing flags as these should be set during startup
	log.Printf("ComponentInstance %s: State loaded successfully from %s", ci.ID, instanceID)
	return nil
}

// runInstance is the main goroutine for the component instance
func (ci *ComponentInstance) runInstance() {
	log.Printf("ComponentInstance %s: Starting instance goroutine", ci.ID)

	for {
		select {
		case op := <-ci.InputChannel:
			// Check if we should shutdown
			if ci.ShutdownFlag.Load() {
				log.Printf("ComponentInstance %s: Shutdown flag set, rejecting operation %s", ci.ID, op.ID)
				continue
			}

			// Mark as processing
			ci.ProcessingFlag.Store(true)

			// Process operation through decision graph
			if err := ci.processOperationThroughEngines(op); err != nil {
				log.Printf("ComponentInstance %s: Error processing operation %s: %v", ci.ID, op.ID, err)
				ci.Metrics.FailedOps++
			} else {
				ci.Metrics.CompletedOps++
			}

			ci.Metrics.TotalOperations++

			// Mark as not processing
			ci.ProcessingFlag.Store(false)

		case <-ci.ctx.Done():
			log.Printf("ComponentInstance %s: Instance goroutine stopping", ci.ID)
			return
		}
	}
}

// processOperationThroughEngines processes an operation through the engine decision graph
func (ci *ComponentInstance) processOperationThroughEngines(op *engines.Operation) error {
	// Initialize queue position tracking metadata
	if op.Metadata == nil {
		op.Metadata = make(map[string]interface{})
	}

	// Set initial queue position tracking
	totalEngines := len(ci.Config.RequiredEngines)
	op.Metadata["total_engines"] = totalEngines
	op.Metadata["current_engine_index"] = 0
	op.Metadata["queue_position"] = totalEngines
	op.Metadata["engines_remaining"] = totalEngines
	op.Metadata["processing_path"] = make([]string, 0, totalEngines)

	log.Printf("ComponentInstance %s: Processing operation %s through %d engines", ci.ID, op.ID, totalEngines)

	// Process through engines using decision graph flow
	return ci.executeDecisionGraphFlow(op)
}

// executeDecisionGraphFlow executes the operation through the decision graph
func (ci *ComponentInstance) executeDecisionGraphFlow(op *engines.Operation) error {
	totalEngines := op.Metadata["total_engines"].(int)

	// For now, simulate sequential processing through engines
	// TODO: Replace with actual decision graph execution when decision graph is fully implemented
	for i, engineType := range ci.Config.RequiredEngines {
		// Update queue position tracking
		currentIndex := i + 1
		remainingEngines := totalEngines - i

		op.Metadata["current_engine_index"] = currentIndex
		op.Metadata["queue_position"] = remainingEngines
		op.Metadata["engines_remaining"] = remainingEngines - 1
		op.Metadata["current_engine"] = engineType.String()

		// Add to processing path
		if path, ok := op.Metadata["processing_path"].([]string); ok {
			op.Metadata["processing_path"] = append(path, engineType.String())
		}

		log.Printf("ComponentInstance %s: Processing operation %s through engine %s (position %d/%d, remaining: %d)",
			ci.ID, op.ID, engineType, currentIndex, totalEngines, remainingEngines-1)

		// Process through engine (simulate for now)
		result := ci.processOperationThroughSingleEngine(op, engineType)
		if result == nil {
			return fmt.Errorf("failed to process operation %s through engine %s", op.ID, engineType)
		}

		// Copy queue position metadata to result metrics
		if result.Metrics == nil {
			result.Metrics = make(map[string]interface{})
		}
		for key, value := range op.Metadata {
			result.Metrics[key] = value
		}

		// Check if this is the last engine (queue_position == 1)
		if remainingEngines == 1 {
			log.Printf("ComponentInstance %s: Last engine %s detected for operation %s (queue_position=1)",
				ci.ID, engineType, op.ID)

			// Mark as final result
			result.Metrics["is_final_result"] = true
			result.Metrics["final_engine"] = engineType.String()

			// Send to centralized output manager
			return ci.sendToOutputManager(result)
		}

		log.Printf("ComponentInstance %s: Processed operation %s through engine %s, continuing to next engine",
			ci.ID, op.ID, engineType)
	}

	return nil
}

// processOperationThroughSingleEngine processes an operation through a single engine
func (ci *ComponentInstance) processOperationThroughSingleEngine(op *engines.Operation, engineType engines.EngineType) *engines.OperationResult {
	// Check if engine exists and is available
	engine, exists := ci.Engines[engineType]
	if !exists || engine == nil {
		log.Printf("ComponentInstance %s: Engine %s not available, simulating processing", ci.ID, engineType)
		// Simulate processing for missing engines
		return &engines.OperationResult{
			OperationID:    op.ID,
			OperationType:  op.Type,
			Success:        true,
			ProcessingTime: time.Millisecond * 10, // Placeholder
			Metrics:        make(map[string]interface{}),
		}
	}

	// TODO: Send operation to actual engine goroutine when engines are properly implemented
	// For now, simulate engine processing
	result := &engines.OperationResult{
		OperationID:    op.ID,
		OperationType:  op.Type,
		Success:        true,
		ProcessingTime: time.Millisecond * 10, // Placeholder
		Metrics:        make(map[string]interface{}),
	}

	// Add engine-specific metadata
	result.Metrics["engine_type"] = engineType.String()
	result.Metrics["processing_time_ms"] = 10
	result.Metrics["simulated"] = true

	return result
}

// sendToOutputManager sends the final result to the centralized output manager
func (ci *ComponentInstance) sendToOutputManager(result *engines.OperationResult) error {
	if ci.CentralizedOutput == nil {
		return fmt.Errorf("centralized output manager not available")
	}

	select {
	case ci.CentralizedOutput.InputChannel <- result:
		log.Printf("ComponentInstance %s: Sent final result to centralized output manager", ci.ID)
		return nil
	default:
		return fmt.Errorf("centralized output manager channel is full")
	}
}

// initializeEngines creates and configures engines for the instance
func (ci *ComponentInstance) initializeEngines() error {
	// Create engine factory for proper engine initialization
	engineFactory := engines.NewEngineFactoryWithPaths("./profiles") // TODO: Make configurable

	// Try to load profiles from files (ignore errors for now)
	if err := engineFactory.LoadProfilesFromFiles(); err != nil {
		log.Printf("ComponentInstance %s: Warning - could not load profiles from files: %v", ci.ID, err)
	}

	for _, engineType := range ci.Config.RequiredEngines {
		// Get profile name for this engine type
		profileName := ci.getEngineProfileName(engineType)

		// Get complexity level for this engine type
		complexityLevel := ci.getEngineComplexityLevel(engineType)

		// Create the base engine using factory
		baseEngine, err := engineFactory.CreateEngine(engineType, profileName, ci.Config.QueueCapacity)
		if err != nil {
			log.Printf("ComponentInstance %s: Failed to create %s engine, using placeholder: %v", ci.ID, engineType, err)
			// Create placeholder engine wrapper for testing - set to nil to avoid crashes
			ci.Engines[engineType] = nil
		} else {
			// Create proper engine wrapper with the base engine
			engineWrapper := engines.NewEngineWrapper(baseEngine, complexityLevel)
			ci.Engines[engineType] = engineWrapper

			log.Printf("ComponentInstance %s: Created %s engine with profile %s", ci.ID, engineType, profileName)
		}

		// Track goroutine information
		ci.EngineGoroutines[engineType] = GoroutineTracker{
			GoroutineID:         fmt.Sprintf("%s-%s", ci.ID, engineType),
			StartTime:           time.Now(),
			LastActivity:        time.Now(),
			OperationsProcessed: 0,
			CurrentLoad:         0.0,
		}
	}

	log.Printf("ComponentInstance %s: Initialized %d engines", ci.ID, len(ci.Engines))
	return nil
}

// getEngineProfileName returns the profile name for a given engine type
func (ci *ComponentInstance) getEngineProfileName(engineType engines.EngineType) string {
	if ci.Config.EngineProfiles != nil {
		if profileName, exists := ci.Config.EngineProfiles[engineType]; exists {
			return profileName
		}
	}

	// Return default profile names based on component type and engine type
	switch ci.ComponentType {
	case ComponentTypeWebServer:
		switch engineType {
		case engines.NetworkEngineType:
			return "web_server_network"
		case engines.CPUEngineType:
			return "web_server_cpu"
		case engines.MemoryEngineType:
			return "web_server_memory"
		case engines.StorageEngineType:
			return "web_server_storage"
		}
	case ComponentTypeDatabase:
		switch engineType {
		case engines.NetworkEngineType:
			return "database_network"
		case engines.CPUEngineType:
			return "database_cpu"
		case engines.MemoryEngineType:
			return "database_memory"
		case engines.StorageEngineType:
			return "database_storage"
		}
	case ComponentTypeCache:
		switch engineType {
		case engines.NetworkEngineType:
			return "cache_network"
		case engines.CPUEngineType:
			return "cache_cpu"
		case engines.MemoryEngineType:
			return "cache_memory"
		case engines.StorageEngineType:
			return "cache_storage"
		}
	case ComponentTypeLoadBalancer:
		switch engineType {
		case engines.NetworkEngineType:
			return "load_balancer_network"
		case engines.CPUEngineType:
			return "load_balancer_cpu"
		case engines.MemoryEngineType:
			return "load_balancer_memory"
		case engines.StorageEngineType:
			return "load_balancer_storage"
		}
	}

	// Fallback to generic profiles
	return fmt.Sprintf("default_%s", engineType.String())
}

// getEngineComplexityLevel returns the complexity level for a given engine type
func (ci *ComponentInstance) getEngineComplexityLevel(engineType engines.EngineType) int {
	if ci.Config.ComplexityLevels != nil {
		if level, exists := ci.Config.ComplexityLevels[engineType]; exists {
			return level
		}
	}

	// Default complexity level based on component type
	switch ci.ComponentType {
	case ComponentTypeWebServer:
		return 1 // Basic complexity for web servers
	case ComponentTypeDatabase:
		return 2 // Advanced complexity for databases
	case ComponentTypeCache:
		return 1 // Basic complexity for caches
	case ComponentTypeLoadBalancer:
		return 0 // Minimal complexity for load balancers
	default:
		return 1 // Basic complexity as default
	}
}

// updateComponentState updates the instance's health and metrics
func (ci *ComponentInstance) updateComponentState() {
	// Update health based on engine status
	healthyEngines := 0
	totalEngines := len(ci.Engines)

	for _, engine := range ci.Engines {
		if engine != nil {
			healthyEngines++
		}
	}

	// Calculate health
	healthRatio := float64(healthyEngines) / float64(totalEngines)
	ci.Health.AvailableCapacity = healthRatio

	if healthRatio >= 0.8 {
		ci.Health.Status = "GREEN"
		ci.Health.IsAcceptingLoad = true
	} else if healthRatio >= 0.5 {
		ci.Health.Status = "YELLOW"
		ci.Health.IsAcceptingLoad = true
	} else {
		ci.Health.Status = "RED"
		ci.Health.IsAcceptingLoad = false
	}

	ci.Health.LastHealthCheck = time.Now()

	// Update metrics
	ci.Metrics.State = ci.GetState()
	ci.Metrics.LastUpdated = time.Now()
}
