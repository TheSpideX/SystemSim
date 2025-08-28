package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EngineWrapper wraps a BaseEngine to provide single-goroutine sequential processing
// that matches real CPU architecture with cycle-based execution
type EngineWrapper struct {
	// Core engine being wrapped
	engine BaseEngine

	// Complexity management (passed to engine during creation)
	complexityLevel int

	// Communication channels (single goroutine responsibility)
	inputQueue   chan *Operation      // FROM other engines/components TO this engine
	tickChannel  chan int64           // FROM global tick coordinator
	stopChannel  chan struct{}        // FOR graceful shutdown

	// Routing and coordination (single goroutine responsibility)
	routingTable map[string]string    // Maps operation types to next destination
	// TODO: Add inter-engine communication when global registry is implemented

	// State management (single goroutine)
	running      bool
	paused       bool              // NEW: Pause state for simulation control
	pauseChannel chan struct{}     // NEW: Channel to signal pause
	resumeChannel chan struct{}    // NEW: Channel to signal resume
	mutex        sync.RWMutex
	wg           sync.WaitGroup

	// Metrics aggregation (single goroutine responsibility)
	processedOps    int64
	queuedOps       int64
	completedOps    int64
	lastTickTime    time.Time

	// Configuration (profile-driven queue sizing)
	inputBufferSize  int
	tickTimeout      time.Duration

	// Sequential processing state (matches real CPU pipeline)
	pendingResults  []*OperationResult // Results waiting to be routed (like CPU write buffer)

	// State persistence (built-in)
	stateDir        string             // Directory for state files
}

// WrapperState represents the complete wrapper state for persistence
type WrapperState struct {
	// Engine identification
	EngineID   string     `json:"engine_id"`
	EngineType EngineType `json:"engine_type"`
	ProfileName string    `json:"profile_name"`

	// Simulation state
	CurrentTick     int64     `json:"current_tick"`
	TotalOperations int64     `json:"total_operations"`
	CompletedOps    int64     `json:"completed_operations"`
	FailedOps       int64     `json:"failed_operations"`
	SavedAt         time.Time `json:"saved_at"`

	// Wrapper-specific state
	Architecture         string                    `json:"architecture"`
	IsRunning           bool                      `json:"is_running"`
	IsPaused            bool                      `json:"is_paused"`        // NEW: Pause state
	ComplexityLevel     int                       `json:"complexity_level"`
	InputQueueOps       []*Operation              `json:"input_queue_operations"`
	PendingResults      []*OperationResult        `json:"pending_results"`
	RoutingTable        map[string]string         `json:"routing_table"`
	ProcessedOps        int64                     `json:"processed_operations"`
	QueuedOps           int64                     `json:"queued_operations"`
	CompletedOperations int64                     `json:"completed_operations"`

	// Engine state (embedded)
	EngineState         map[string]interface{}    `json:"engine_state"`
}

// NewEngineWrapper creates a new wrapper with single-goroutine sequential processing
func NewEngineWrapper(engine BaseEngine, complexity int) *EngineWrapper {
	// Calculate realistic queue sizes based on engine profile and complexity level
	// This ensures realistic behavior where high input rates can cause queue overflow
	var inputBufferSize int
	var tickTimeout time.Duration

	// Get profile-driven queue sizes (realistic modeling)
	inputBufferSize, _ = calculateProfileDrivenQueueSizes(engine, ComplexityLevel(complexity))

	// If profile-driven calculation fails, fall back to engine type defaults
	if inputBufferSize == 0 {
		baseInputSize, _ := calculateEngineQueueSizes(engine.GetEngineType())
		complexityMultiplier := getComplexityQueueMultiplier(complexity)
		inputBufferSize = int(float64(baseInputSize) * complexityMultiplier)
	}

	// Timeout varies by complexity level for processing accuracy
	// Scaled for 0.01ms tick duration - timeouts must be very fast
	switch complexity {
	case 0: // Minimal
		tickTimeout = 50 * time.Microsecond  // Ultra-fast timeout for minimal complexity
	case 1: // Basic
		tickTimeout = 100 * time.Microsecond // Fast timeout for basic complexity
	case 2: // Advanced
		tickTimeout = 200 * time.Microsecond // Moderate timeout for advanced complexity
	case 3: // Maximum
		tickTimeout = 500 * time.Microsecond // Longer timeout for maximum complexity
	default:
		// Default to Basic settings
		tickTimeout = 100 * time.Microsecond
	}

	wrapper := &EngineWrapper{
		engine:           engine,
		complexityLevel:  complexity,
		inputQueue:       make(chan *Operation, inputBufferSize),
		tickChannel:      make(chan int64, 1),
		stopChannel:      make(chan struct{}),
		pauseChannel:     make(chan struct{}),     // NEW: Initialize pause channel
		resumeChannel:    make(chan struct{}),     // NEW: Initialize resume channel
		routingTable:     make(map[string]string), // Initialize routing table for inter-engine communication
		inputBufferSize:  inputBufferSize,
		tickTimeout:      tickTimeout,
		running:          false,
		paused:           false,                   // NEW: Initialize pause state
		pendingResults:   make([]*OperationResult, 0, 10), // Pre-allocate for efficiency
		stateDir:         "./engine_states", // Default state directory
	}

	// Set tick duration to match the global clock coordinator (0.01ms for precision)
	engine.SetTickDuration(10 * time.Microsecond)

	return wrapper
}

// NewEngineWrapperWithProfile creates a new wrapper with profile-driven engine creation and queue sizing
func NewEngineWrapperWithProfile(engineType EngineType, profileName string, complexity int, profileLoader *ProfileLoader) (*EngineWrapper, error) {
	// Load the profile
	profilePath := profileLoader.GetProfilePath(engineType, profileName)
	profile, err := profileLoader.LoadProfileFromFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile %s: %w", profileName, err)
	}

	// Calculate optimal queue size from profile
	queueCapacity := calculateOptimalQueueSizeFromProfile(profile, ComplexityLevel(complexity))

	// Create engine with calculated queue size
	var engine BaseEngine
	switch engineType {
	case CPUEngineType:
		engine = NewCPUEngine(queueCapacity)
	case MemoryEngineType:
		engine = NewMemoryEngine(queueCapacity)
	case StorageEngineType:
		engine = NewStorageEngine(queueCapacity)
	case NetworkEngineType:
		engine = NewNetworkEngine(queueCapacity)
	default:
		return nil, fmt.Errorf("unknown engine type: %v", engineType)
	}

	// Load profile into engine
	if err := engine.LoadProfile(profile); err != nil {
		return nil, fmt.Errorf("failed to load profile into engine: %w", err)
	}

	// Set complexity level in engine (engines need to know complexity for processing decisions)
	if err := engine.SetComplexityLevel(complexity); err != nil {
		return nil, fmt.Errorf("failed to set complexity level in engine: %w", err)
	}

	// Create wrapper with profile-driven queue sizes
	return NewEngineWrapper(engine, complexity), nil
}

// Start begins the wrapper's single goroutine (matches real CPU architecture)
func (ew *EngineWrapper) Start(ctx context.Context) error {
	ew.mutex.Lock()
	defer ew.mutex.Unlock()

	if ew.running {
		return fmt.Errorf("engine wrapper already running")
	}

	ew.running = true
	ew.lastTickTime = time.Now()

	// Start single goroutine for sequential processing (like real CPU control unit)
	ew.wg.Add(1)

	go ew.sequentialProcessor(ctx)  // Single control unit with sequential cycles

	return nil
}

// Stop gracefully shuts down the wrapper
func (ew *EngineWrapper) Stop() error {
	ew.mutex.Lock()
	defer ew.mutex.Unlock()

	if !ew.running {
		return nil
	}

	ew.running = false
	close(ew.stopChannel)

	// Wait for single goroutine to finish
	ew.wg.Wait()

	return nil
}

// Pause temporarily suspends simulation processing while keeping the wrapper running
func (ew *EngineWrapper) Pause() error {
	ew.mutex.Lock()
	defer ew.mutex.Unlock()

	if !ew.running {
		return fmt.Errorf("cannot pause: wrapper is not running")
	}

	if ew.paused {
		return fmt.Errorf("wrapper is already paused")
	}

	ew.paused = true

	// Signal pause to the processing goroutine
	select {
	case ew.pauseChannel <- struct{}{}:
		fmt.Printf("⏸️ Paused engine wrapper %s\n", ew.engine.GetEngineID())
		return nil
	case <-time.After(ew.tickTimeout):
		ew.paused = false // Reset pause state on timeout
		return fmt.Errorf("pause timeout after %v", ew.tickTimeout)
	}
}

// Resume continues simulation processing after a pause
func (ew *EngineWrapper) Resume() error {
	ew.mutex.Lock()
	defer ew.mutex.Unlock()

	if !ew.running {
		return fmt.Errorf("cannot resume: wrapper is not running")
	}

	if !ew.paused {
		return fmt.Errorf("wrapper is not paused")
	}

	ew.paused = false

	// Signal resume to the processing goroutine
	select {
	case ew.resumeChannel <- struct{}{}:
		fmt.Printf("▶️ Resumed engine wrapper %s\n", ew.engine.GetEngineID())
		return nil
	case <-time.After(ew.tickTimeout):
		ew.paused = true // Reset pause state on timeout
		return fmt.Errorf("resume timeout after %v", ew.tickTimeout)
	}
}

// IsPaused returns whether the wrapper is currently paused
func (ew *EngineWrapper) IsPaused() bool {
	ew.mutex.RLock()
	defer ew.mutex.RUnlock()
	return ew.paused
}

// IsRunning returns whether the wrapper is currently running
func (ew *EngineWrapper) IsRunning() bool {
	ew.mutex.RLock()
	defer ew.mutex.RUnlock()
	return ew.running
}

// QueueOperation adds an operation to the input queue
func (ew *EngineWrapper) QueueOperation(op *Operation) error {
	select {
	case ew.inputQueue <- op:
		ew.mutex.Lock()
		ew.queuedOps++
		ew.mutex.Unlock()
		return nil
	default:
		return fmt.Errorf("input queue is full (capacity: %d)", ew.inputBufferSize)
	}
}

// sequentialProcessor implements single-goroutine sequential processing (matches real CPU)
func (ew *EngineWrapper) sequentialProcessor(ctx context.Context) {
	defer ew.wg.Done()

	for {
		select {
		// Priority 1: Handle pause/resume signals
		case <-ew.pauseChannel:
			fmt.Printf("⏸️ Engine wrapper %s entering pause state\n", ew.engine.GetEngineID())
			// Wait for resume signal or shutdown
			select {
			case <-ew.resumeChannel:
				fmt.Printf("▶️ Engine wrapper %s resuming from pause\n", ew.engine.GetEngineID())
				continue
			case <-ew.stopChannel:
				return
			case <-ctx.Done():
				return
			}

		// Priority 2: Process ticks (most important - like real CPU clock cycles)
		case currentTick := <-ew.tickChannel:
			// Skip tick processing if paused
			if ew.IsPaused() {
				continue
			}

			ew.lastTickTime = time.Now()

			// SEQUENTIAL PROCESSING CYCLES (like real CPU pipeline stages):

			// CYCLE 1: Input Stage (Fetch) - Process incoming operations
			ew.processInputCycle()

			// CYCLE 2: Execute Stage (Decode + Execute) - Process tick in engine
			results := ew.engine.ProcessTick(currentTick)

			// CYCLE 3: Output Stage (Write Back) - Route completed operations
			ew.processOutputCycle(results)

		// Priority 3: Handle new input operations (when not processing ticks)
		case op := <-ew.inputQueue:
			// Skip operation processing if paused
			if ew.IsPaused() {
				// Put operation back in queue to process after resume
				select {
				case ew.inputQueue <- op:
					// Successfully put back
				default:
					// Queue full - this is realistic behavior during pause
					fmt.Printf("Warning: Dropping operation %s during pause (queue full)\n", op.ID)
				}
				continue
			}

			// Check if engine can accept operation BEFORE taking from inputQueue
			if ew.engine.GetQueueLength() < ew.engine.GetQueueCapacity() {
				// Engine has space - transfer operation
				if err := ew.engine.QueueOperation(op); err != nil {
					// This should rarely happen since we checked capacity
					fmt.Printf("Warning: Engine queue operation failed: %v\n", err)
				} else {
					ew.mutex.Lock()
					ew.processedOps++
					ew.mutex.Unlock()
				}
			} else {
				// ✅ PROPER BACKPRESSURE: Engine queue full - put operation back
				// This creates backpressure by blocking the inputQueue
				select {
				case ew.inputQueue <- op:
					// Successfully put back - will retry later
				default:
					// InputQueue also full - this is realistic overload
					fmt.Printf("Warning: System overloaded, dropping operation %s\n", op.ID)
				}
			}

		// Priority 4: Shutdown
		case <-ew.stopChannel:
			return
		case <-ctx.Done():
			return

		// Priority 5: Periodic maintenance (when idle)
		default:
			// Skip maintenance if paused
			if !ew.IsPaused() {
				// Process any pending results that couldn't be routed immediately
				if len(ew.pendingResults) > 0 {
					ew.processPendingResults()
				}
			}
			// Small sleep to prevent busy waiting
			time.Sleep(1 * time.Microsecond)
		}
	}
}

// processInputCycle handles the input stage (like CPU fetch stage)
func (ew *EngineWrapper) processInputCycle() {
	// Process multiple input operations in one cycle (like CPU can fetch multiple instructions)
	maxInputsPerCycle := 3 // Realistic limit based on CPU fetch width

	for i := 0; i < maxInputsPerCycle; i++ {
		// Check if engine can accept more operations
		if ew.engine.GetQueueLength() >= ew.engine.GetQueueCapacity() {
			// Engine queue full - stop processing input to create backpressure
			break
		}

		select {
		case op := <-ew.inputQueue:
			// Engine has space - transfer operation
			if err := ew.engine.QueueOperation(op); err != nil {
				// This should rarely happen since we checked capacity
				fmt.Printf("Input cycle: Engine queue operation failed: %v\n", err)
			} else {
				ew.mutex.Lock()
				ew.queuedOps++
				ew.mutex.Unlock()
			}
		default:
			// No more input operations available
			break
		}
	}
}

// processOutputCycle handles the output stage (like CPU write-back stage)
func (ew *EngineWrapper) processOutputCycle(results []OperationResult) {
	// Route completed operations immediately (like CPU write-back)
	for _, result := range results {
		// Try to route immediately
		if ew.routeCompletedOperation(&result) {
			ew.mutex.Lock()
			ew.completedOps++
			ew.mutex.Unlock()
		} else {
			// If routing fails, add to pending results (like CPU write buffer)
			ew.pendingResults = append(ew.pendingResults, &result)
		}
	}
}

// processPendingResults handles results that couldn't be routed immediately
func (ew *EngineWrapper) processPendingResults() {
	// Try to route pending results (like draining CPU write buffer)
	remainingResults := make([]*OperationResult, 0, len(ew.pendingResults))

	for _, result := range ew.pendingResults {
		if ew.routeCompletedOperation(result) {
			ew.mutex.Lock()
			ew.completedOps++
			ew.mutex.Unlock()
		} else {
			// Still can't route - keep in pending
			remainingResults = append(remainingResults, result)
		}
	}

	ew.pendingResults = remainingResults
}

// routeCompletedOperation routes a completed operation to its next destination
// Returns true if routing was successful, false if it should be retried later
func (ew *EngineWrapper) routeCompletedOperation(result *OperationResult) bool {
	// Determine next destination using routing table
	var nextDestination string

	// First check if operation specifies next component
	if result.NextComponent != "" {
		nextDestination = result.NextComponent
	} else {
		// Use routing table based on operation type
		if destination, exists := ew.GetRouting(result.OperationType); exists {
			nextDestination = destination
		} else {
			// Fall back to default routing
			if defaultDest, exists := ew.GetRouting("default"); exists {
				nextDestination = defaultDest
			} else {
				nextDestination = "completed" // Final destination
			}
		}
	}

	// Handle routing destinations
	switch nextDestination {
	case "drain", "completed":
		// INTRA-ENGINE: Operation completed within this engine
		// Drain the operation (prevents backup)
		ew.drainCompletedOperation(result)
		return true // Successfully drained

	case "memory_engine", "storage_engine", "network_engine":
		// TODO: INTER-ENGINE: Route to different engine types via global registry
		// This will be implemented when global registry is available
		// For now, drain to prevent backup
		ew.drainCompletedOperation(result)
		return true // Successfully drained

	case "same_engine":
		// INTRA-ENGINE: Route back to same engine for further processing
		// This handles multi-stage operations within the same engine
		return ew.routeIntraEngine(result)

	default:
		// Unknown destination - drain to prevent backup
		ew.drainCompletedOperation(result)
		return true // Successfully drained
	}
}

// Component interface implementation for clock coordinator integration

// ProcessTick implements the Component interface for the clock coordinator
func (ew *EngineWrapper) ProcessTick(currentTick int64) error {
	// If paused, don't send tick but don't timeout either
	if ew.IsPaused() {
		return nil // Silently skip tick processing when paused
	}

	select {
	case ew.tickChannel <- currentTick:
		return nil
	case <-time.After(ew.tickTimeout):
		return fmt.Errorf("tick processing timeout after %v", ew.tickTimeout)
	}
}

// GetID returns the wrapped engine's ID
func (ew *EngineWrapper) GetID() string {
	return ew.engine.GetEngineID()
}

// GetTickChannel returns the tick channel for global coordinator integration
func (ew *EngineWrapper) GetTickChannel() chan int64 {
	return ew.tickChannel
}

// IsHealthy returns whether the wrapper and engine are functioning properly
func (ew *EngineWrapper) IsHealthy() bool {
	ew.mutex.RLock()
	defer ew.mutex.RUnlock()

	// Check if wrapper is running and engine is healthy
	health := ew.engine.GetHealth()
	return ew.running && health.Score > 0.5
}

// Removed duplicate methods - using the ones defined above

// GetMetrics returns wrapper-specific metrics
func (ew *EngineWrapper) GetMetrics() map[string]interface{} {
	ew.mutex.RLock()
	defer ew.mutex.RUnlock()

	return map[string]interface{}{
		"complexity_level":     ComplexityLevel(ew.complexityLevel).String(),
		"input_buffer_size":    ew.inputBufferSize,
		"input_queue_length":   len(ew.inputQueue),
		"pending_results":      len(ew.pendingResults),
		"processed_operations": ew.processedOps,
		"queued_operations":    ew.queuedOps,
		"completed_operations": ew.completedOps,
		"last_tick_time":       ew.lastTickTime,
		"running":              ew.running,
		"paused":               ew.paused,              // NEW: Include pause state
		"engine_utilization":   ew.engine.GetUtilization(),
		"engine_health":        ew.engine.GetHealth(),
		"architecture":         "single_goroutine_sequential", // New metric
	}
}

// GetComplexityLevel returns the current complexity level
func (ew *EngineWrapper) GetComplexityLevel() int {
	return ew.complexityLevel
}

// SetComplexityLevel changes the complexity level (requires restart)
func (ew *EngineWrapper) SetComplexityLevel(level int) error {
	if ew.running {
		return fmt.Errorf("cannot change complexity level while running")
	}

	ew.complexityLevel = level

	// Update the engine's complexity level
	return ew.engine.SetComplexityLevel(level)
}

// SetRouting configures where to send completed operations based on operation type
func (ew *EngineWrapper) SetRouting(operationType string, nextDestination string) {
	ew.mutex.Lock()
	defer ew.mutex.Unlock()
	ew.routingTable[operationType] = nextDestination
}

// GetRouting returns the next destination for an operation type
func (ew *EngineWrapper) GetRouting(operationType string) (string, bool) {
	ew.mutex.RLock()
	defer ew.mutex.RUnlock()
	destination, exists := ew.routingTable[operationType]
	return destination, exists
}

// SetDefaultRouting sets a default destination for operations without specific routing
func (ew *EngineWrapper) SetDefaultRouting(defaultDestination string) {
	ew.SetRouting("default", defaultDestination)
}

// GetQueueLength returns the current input queue length
func (ew *EngineWrapper) GetQueueLength() int {
	return len(ew.inputQueue)
}

// GetPendingResultsLength returns the current pending results length (replaces output queue)
func (ew *EngineWrapper) GetPendingResultsLength() int {
	return len(ew.pendingResults)
}

// drainCompletedOperation drains a completed operation (intra-engine completion)
func (ew *EngineWrapper) drainCompletedOperation(result *OperationResult) {
	// INTRA-ENGINE: Operation completed within this engine
	// This prevents output queue backup when there's no global registry
	// In production, completed operations would be sent to global registry

	// Log completion for debugging/monitoring
	// fmt.Printf("✅ Operation %s completed in %v (drained)\n",
	//     result.OperationID, result.ProcessingTime)

	// Operation is successfully drained - no further action needed
	// This maintains realistic queue flow without external dependencies
}

// routeIntraEngine handles intra-engine routing for multi-stage operations
// Returns true if routing was successful, false if it should be retried later
func (ew *EngineWrapper) routeIntraEngine(result *OperationResult) bool {
	// INTRA-ENGINE: Route operation back to same engine for further processing
	// This handles cases where an operation needs multiple processing stages

	// Create new operation for next stage
	nextStageOp := &Operation{
		ID:         result.OperationID + "-stage2", // Append stage identifier
		Type:       result.OperationType,
		Complexity: "O(1)", // Next stage might have different complexity
		DataSize:   1024,    // Data size might change between stages
		Language:   "cpp",
		Metadata: map[string]interface{}{
			"previous_stage_result": result,
			"stage": 2,
		},
	}

	// Queue the next stage operation back to this engine
	err := ew.QueueOperation(nextStageOp)
	if err != nil {
		// If queue is full, routing failed - should retry later
		return false
	}

	// Successfully routed for next stage processing
	return true

	// Note: This enables complex multi-stage processing within a single engine
	// Examples: CPU instruction decode → execute → writeback stages
}

// TODO: INTER-ENGINE COMMUNICATION METHODS (to be implemented with global registry)

// TODO: routeToMemoryEngine routes operations to memory engine
// func (ew *EngineWrapper) routeToMemoryEngine(result *OperationResult) {
//     // Will route to memory engine via global registry
//     // globalRegistry.RouteToEngine("memory", result)
// }

// TODO: routeToStorageEngine routes operations to storage engine
// func (ew *EngineWrapper) routeToStorageEngine(result *OperationResult) {
//     // Will route to storage engine via global registry
//     // globalRegistry.RouteToEngine("storage", result)
// }

// TODO: routeToNetworkEngine routes operations to network engine
// func (ew *EngineWrapper) routeToNetworkEngine(result *OperationResult) {
//     // Will route to network engine via global registry
//     // globalRegistry.RouteToEngine("network", result)
// }

// TODO: registerWithGlobalRegistry registers this wrapper with global registry
// func (ew *EngineWrapper) registerWithGlobalRegistry(registry *GlobalRegistry) error {
//     // Will register this wrapper for inter-engine communication
//     // return registry.RegisterEngine(ew.engine.GetEngineType(), ew)
// }

// TODO: handleInterEngineMessage handles messages from other engines
// func (ew *EngineWrapper) handleInterEngineMessage(message *InterEngineMessage) error {
//     // Will handle operations routed from other engines
//     // return ew.QueueOperation(message.Operation)
// }

// calculateProfileDrivenQueueSizes calculates queue sizes based on engine profile characteristics
// Note: Only returns inputSize since single-goroutine architecture doesn't need output queues
func calculateProfileDrivenQueueSizes(engine BaseEngine, complexity ComplexityLevel) (inputSize, outputSize int) {
	profile := engine.GetProfile()
	if profile == nil {
		return 0, 0 // Fall back to engine type defaults
	}

	switch profile.Type {
	case CPUEngineType:
		inputSize, _ = calculateCPUQueueSizeFromProfile(profile, complexity)
		return inputSize, 0 // No output queue needed
	case MemoryEngineType:
		inputSize, _ = calculateMemoryQueueSizeFromProfile(profile, complexity)
		return inputSize, 0 // No output queue needed
	case StorageEngineType:
		inputSize, _ = calculateStorageQueueSizeFromProfile(profile, complexity)
		return inputSize, 0 // No output queue needed
	case NetworkEngineType:
		inputSize, _ = calculateNetworkQueueSizeFromProfile(profile, complexity)
		return inputSize, 0 // No output queue needed
	default:
		return 0, 0
	}
}

// calculateOptimalQueueSizeFromProfile calculates optimal internal engine queue size from profile
func calculateOptimalQueueSizeFromProfile(profile *EngineProfile, complexity ComplexityLevel) int {
	switch profile.Type {
	case CPUEngineType:
		// CPU queue calculation: cores × realistic_operation_ticks × buffer_factor
		cores := int(profile.BaselinePerformance["cores"])
		baseProcessingTime := profile.BaselinePerformance["base_processing_time"] // milliseconds (for O(1))

		// Calculate realistic operation time considering complexity and language multipliers
		realisticOperationTime := calculateRealisticOperationTime(profile, baseProcessingTime)

		// Convert to ticks (10 microseconds per tick)
		tickDurationMs := 0.01
		avgOperationTicks := int(realisticOperationTime / tickDurationMs)

		// Operations per tick (from profile or default to 3)
		opsPerTick := 3
		if profile.EngineSpecific != nil {
			if queueConfig, ok := profile.EngineSpecific["queue_processing"].(map[string]interface{}); ok {
				if maxOps, ok := queueConfig["max_ops_per_tick"].(float64); ok {
					opsPerTick = int(maxOps)
				}
			}
		}

		// Calculate: (cores × operation_duration_ticks) ÷ ops_per_tick × buffer_factor
		bufferFactor := 2.0 // 2x buffer for safety
		queueSize := int(float64(cores*avgOperationTicks) / float64(opsPerTick) * bufferFactor)

		// Ensure minimum queue size for realistic modeling
		if queueSize < 1000 {
			queueSize = 1000
		}

		// Apply complexity scaling
		complexityMultiplier := getComplexityQueueMultiplier(int(complexity))
		return int(float64(queueSize) * complexityMultiplier)

	case MemoryEngineType:
		// Memory queue calculation: channels × realistic_operation_ticks × buffer_factor (DYNAMIC like CPU)
		channels := int(profile.BaselinePerformance["channels"])
		accessTimeNs := profile.BaselinePerformance["access_time"]

		// Calculate realistic operation time for memory operations
		realisticOperationTime := calculateRealisticMemoryOperationTime(profile, accessTimeNs)

		// Convert to ticks (10 microseconds per tick)
		tickDurationMs := 0.01
		avgOperationTicks := int(realisticOperationTime / tickDurationMs)

		// Operations per tick (from profile or default to 3)
		opsPerTick := 3
		if profile.EngineSpecific != nil {
			if queueConfig, ok := profile.EngineSpecific["queue_processing"].(map[string]interface{}); ok {
				if maxOps, ok := queueConfig["max_ops_per_tick"].(float64); ok {
					opsPerTick = int(maxOps)
				}
			}
		}

		// Calculate: (channels × operation_duration_ticks) ÷ ops_per_tick × buffer_factor
		bufferFactor := 1.5 // Smaller buffer than wrapper to ensure engine queue fills first
		queueSize := int(float64(channels*avgOperationTicks) / float64(opsPerTick) * bufferFactor)

		// Ensure minimum queue size for realistic modeling
		if queueSize < 200 {
			queueSize = 200
		}

		// Apply complexity scaling
		complexityMultiplier := getComplexityQueueMultiplier(int(complexity))
		return int(float64(queueSize) * complexityMultiplier)

	case StorageEngineType:
		// Storage has variable latency, larger queue needed
		return int(10000 * getComplexityQueueMultiplier(int(complexity)))

	case NetworkEngineType:
		// Network depends on bandwidth and latency
		bandwidth := profile.BaselinePerformance["bandwidth_mbps"]
		latency := profile.BaselinePerformance["base_latency_ms"]

		// Higher bandwidth or latency = larger queue needed
		queueSize := int(bandwidth/100 + latency*100) // Heuristic calculation
		return int(float64(queueSize) * getComplexityQueueMultiplier(int(complexity)))

	default:
		return 5000 // Default fallback
	}
}

// calculateEngineQueueSizes returns optimal queue sizes for different engine types
func calculateEngineQueueSizes(engineType EngineType) (inputSize, outputSize int) {
	switch engineType {
	case CPUEngineType:
		// CPU: 3 ops/tick, ~200 ticks/op, 24 cores
		// Calculation: (24 cores × 200 ticks) ÷ 3 ops/tick × 2 buffer = 3200
		return 3200, 3200

	case MemoryEngineType:
		// Memory: Much faster operations (~1-10 ticks), higher throughput
		// Smaller queue sufficient due to fast processing
		return 800, 800

	case StorageEngineType:
		// Storage: Variable latency (2-1000 ticks), lower throughput
		// Larger queue needed for variable operation durations
		return 8000, 8000

	case NetworkEngineType:
		// Network: High latency (100-10000 ticks), bandwidth dependent
		// Medium queue size for network buffering
		return 2000, 2000

	default:
		// Default to CPU sizing for unknown engine types
		return 3200, 3200
	}
}

// getComplexityQueueMultiplier returns queue size multiplier based on complexity level
func getComplexityQueueMultiplier(complexity int) float64 {
	switch complexity {
	case 0: // Minimal
		// Minimal complexity: Fast processing, smaller queues sufficient
		return 0.5 // 50% of base size

	case 1: // Basic
		// Basic complexity: Standard processing, standard queues
		return 0.75 // 75% of base size

	case 2: // Advanced
		// Advanced complexity: Detailed modeling, full queues needed
		return 1.0 // 100% of base size (realistic)

	case 3: // Maximum
		// Maximum complexity: Most detailed modeling, larger queues for accuracy
		return 1.5 // 150% of base size

	default:
		// Default to Advanced settings
		return 1.0
	}
}

// calculateCPUQueueSizeFromProfile calculates CPU wrapper queue sizes from profile
func calculateCPUQueueSizeFromProfile(profile *EngineProfile, complexity ComplexityLevel) (inputSize, outputSize int) {
	cores := int(profile.BaselinePerformance["cores"])
	baseProcessingTime := profile.BaselinePerformance["base_processing_time"] // milliseconds

	// Calculate realistic operation time (with complexity, language, cache factors)
	realisticOperationTime := calculateRealisticOperationTime(profile, baseProcessingTime)

	// Convert to ticks and calculate queue requirements
	tickDurationMs := 0.01
	avgOperationTicks := int(realisticOperationTime / tickDurationMs)

	// Base queue size calculation: (cores × operation_duration_ticks) ÷ ops_per_tick × buffer_factor
	opsPerTick := 3 // Standard 3 operations per tick
	bufferFactor := 2.0 // 2x buffer for safety
	baseQueueSize := int(float64(cores*avgOperationTicks) / float64(opsPerTick) * bufferFactor)

	// Ensure minimum queue size
	if baseQueueSize < 500 {
		baseQueueSize = 500
	}

	// Apply complexity scaling
	complexityMultiplier := getComplexityQueueMultiplier(int(complexity))
	queueSize := int(float64(baseQueueSize) * complexityMultiplier)

	return queueSize, queueSize
}

// calculateMemoryQueueSizeFromProfile calculates Memory wrapper queue sizes from profile (DYNAMIC like CPU)
func calculateMemoryQueueSizeFromProfile(profile *EngineProfile, complexity ComplexityLevel) (inputSize, outputSize int) {
	// Dynamic calculation based on memory characteristics (like CPU engine)
	channels := int(profile.BaselinePerformance["channels"])
	accessTimeNs := profile.BaselinePerformance["access_time"]

	// Calculate realistic operation time for memory operations
	realisticOperationTime := calculateRealisticMemoryOperationTime(profile, accessTimeNs)

	// Convert to ticks and calculate queue requirements
	tickDurationMs := 0.01
	avgOperationTicks := int(realisticOperationTime / tickDurationMs)

	// Operations per tick (from profile or default to 3)
	opsPerTick := 3
	if profile.EngineSpecific != nil {
		if queueConfig, ok := profile.EngineSpecific["queue_processing"].(map[string]interface{}); ok {
			if maxOps, ok := queueConfig["max_ops_per_tick"].(float64); ok {
				opsPerTick = int(maxOps)
			}
		}
	}

	// Base queue size calculation: (channels × operation_duration_ticks) ÷ ops_per_tick × buffer_factor
	bufferFactor := 3.0 // 3x buffer for memory (higher than CPU due to burst nature)
	baseQueueSize := int(float64(channels*avgOperationTicks) / float64(opsPerTick) * bufferFactor)

	// Ensure minimum queue size for memory operations
	if baseQueueSize < 400 {
		baseQueueSize = 400
	}

	// Apply complexity scaling
	complexityMultiplier := getComplexityQueueMultiplier(int(complexity))
	queueSize := int(float64(baseQueueSize) * complexityMultiplier)

	return queueSize, queueSize
}

// calculateStorageQueueSizeFromProfile calculates Storage wrapper queue sizes from profile
func calculateStorageQueueSizeFromProfile(profile *EngineProfile, complexity ComplexityLevel) (inputSize, outputSize int) {
	// Storage has variable latency, larger queues needed
	maxIOPS := profile.BaselinePerformance["max_iops"]
	avgLatency := profile.BaselinePerformance["avg_latency_ms"]

	// Higher IOPS = more concurrent operations = larger queue
	// Higher latency = longer operation duration = larger queue
	baseSize := int(maxIOPS/1000 + avgLatency*1000) // Heuristic
	if baseSize < 2000 {
		baseSize = 2000 // Minimum for storage
	}

	complexityMultiplier := getComplexityQueueMultiplier(int(complexity))
	queueSize := int(float64(baseSize) * complexityMultiplier)

	return queueSize, queueSize
}

// calculateNetworkQueueSizeFromProfile calculates Network wrapper queue sizes from profile
func calculateNetworkQueueSizeFromProfile(profile *EngineProfile, complexity ComplexityLevel) (inputSize, outputSize int) {
	bandwidth := profile.BaselinePerformance["bandwidth_mbps"]
	latency := profile.BaselinePerformance["base_latency_ms"]

	// Higher bandwidth = more concurrent transfers = larger queue
	// Higher latency = longer transfer duration = larger queue
	baseSize := int(bandwidth/100 + latency*100) // Heuristic
	if baseSize < 1000 {
		baseSize = 1000 // Minimum for network
	}

	complexityMultiplier := getComplexityQueueMultiplier(int(complexity))
	queueSize := int(float64(baseSize) * complexityMultiplier)

	return queueSize, queueSize
}

// calculateRealisticMemoryOperationTime calculates realistic memory operation time (FULLY PROFILE-DRIVEN like CPU)
func calculateRealisticMemoryOperationTime(profile *EngineProfile, accessTimeNs float64) float64 {
	// Convert nanoseconds to milliseconds for consistency
	baseTimeMs := accessTimeNs / 1000000.0

	// ✅ PROFILE-DRIVEN: Get base complexity factor from profile (like CPU engine)
	complexityFactor := getProfileFloatFromEngineProfile(profile, "queue_processing", "base_complexity_factor", 1.2)

	// ✅ PROFILE-DRIVEN: Apply memory-specific factors from profile
	if profile.EngineSpecific != nil {
		if factors, ok := profile.EngineSpecific["realistic_factors"].(map[string]interface{}); ok {
			// ✅ PROFILE-DRIVEN: Refresh overhead from profile
			if overhead, ok := factors["refresh_overhead"].(float64); ok {
				complexityFactor += overhead
			}
			// ✅ PROFILE-DRIVEN: Bank conflict probability from profile
			if conflicts, ok := factors["bank_conflict_probability"].(float64); ok {
				bankConflictMultiplier := getProfileFloatFromEngineProfile(profile, "realistic_factors", "bank_conflict_multiplier", 0.5)
				complexityFactor += conflicts * bankConflictMultiplier
			}
			// ✅ PROFILE-DRIVEN: Queue depth impact from profile
			if queueImpact, ok := factors["queue_depth_impact"].(map[string]interface{}); ok {
				if mediumLoad, ok := queueImpact["medium_load"].(float64); ok {
					// Assume medium load for queue sizing calculations
					complexityFactor *= mediumLoad
				}
			}
		}

		// ✅ PROFILE-DRIVEN: Bandwidth saturation effects from profile
		if bandwidth, ok := profile.EngineSpecific["bandwidth_characteristics"].(map[string]interface{}); ok {
			if saturationThreshold, ok := bandwidth["saturation_threshold"].(float64); ok {
				// Apply saturation effects for queue sizing (assume 80% utilization)
				utilizationFactor := 0.8 / saturationThreshold
				if utilizationFactor > 1.0 {
					complexityFactor *= utilizationFactor
				}
			}
		}
	}

	return baseTimeMs * complexityFactor
}

// getProfileFloatFromEngineProfile gets a float value from engine profile (helper function like CPU engine)
func getProfileFloatFromEngineProfile(profile *EngineProfile, section, key string, defaultValue float64) float64 {
	if profile == nil || profile.EngineSpecific == nil {
		return defaultValue
	}

	if sectionData, ok := profile.EngineSpecific[section]; ok {
		if sectionMap, ok := sectionData.(map[string]interface{}); ok {
			if value, ok := sectionMap[key].(float64); ok {
				return value
			}
		}
	}

	return defaultValue
}

// calculateRealisticOperationTime calculates realistic operation time considering complexity and language factors
func calculateRealisticOperationTime(profile *EngineProfile, baseTime float64) float64 {
	// Start with base processing time (for O(1) operations)
	realisticTime := baseTime

	// Apply complexity factor (assume O(n²) for realistic queue sizing)
	if profile.EngineSpecific != nil {
		if complexityFactors, ok := profile.EngineSpecific["complexity_factors"].(map[string]interface{}); ok {
			if factor, ok := complexityFactors["O(n²)"].(float64); ok {
				realisticTime *= factor // Apply O(n²) complexity
			}
		}

		// Apply language multiplier (assume C++ for realistic sizing)
		if langMultipliers, ok := profile.EngineSpecific["language_multipliers"].(map[string]interface{}); ok {
			if multiplier, ok := langMultipliers["cpp"].(float64); ok {
				realisticTime *= multiplier // Apply C++ language overhead
			}
		}

		// Apply cache miss penalty (assume some cache misses for realistic sizing)
		if cacheBehavior, ok := profile.EngineSpecific["cache_behavior"].(map[string]interface{}); ok {
			if memoryMultiplier, ok := cacheBehavior["memory_access_multiplier"].(float64); ok {
				// Assume 10% cache miss rate for realistic calculation
				cacheMissRate := 0.1
				realisticTime *= (1.0 + cacheMissRate*(memoryMultiplier-1.0))
			}
		}
	}

	// Ensure minimum realistic operation time (at least 1ms for complex operations)
	if realisticTime < 1.0 {
		realisticTime = 1.0
	}

	return realisticTime
}

// extractWrapperState extracts the complete wrapper state for persistence
func (ew *EngineWrapper) extractWrapperState() *WrapperState {
	// Extract input queue operations
	inputQueueOps := make([]*Operation, 0)
	queueLength := len(ew.inputQueue)
	tempOps := make([]*Operation, 0, queueLength)

	// Drain input queue to capture state
	for i := 0; i < queueLength; i++ {
		select {
		case op := <-ew.inputQueue:
			tempOps = append(tempOps, op)
			inputQueueOps = append(inputQueueOps, op)
		default:
			break
		}
	}

	// Put operations back in queue
	for _, op := range tempOps {
		ew.inputQueue <- op
	}

	// Get engine state
	engineState := ew.engine.GetCurrentState()

	// Try to get detailed engine state if the engine supports it
	var detailedEngineState map[string]interface{}
	if memEngine, ok := ew.engine.(*MemoryEngine); ok {
		// For memory engine, get complete state
		if stateData, err := memEngine.SaveEngineState(); err == nil {
			detailedEngineState = map[string]interface{}{
				"type": "memory_engine_detailed",
				"data": string(stateData),
			}
		}
	}

	if detailedEngineState == nil {
		detailedEngineState = engineState
	}

	// Create wrapper state
	state := &WrapperState{
		EngineID:            ew.engine.GetEngineID(),
		EngineType:          ew.engine.GetEngineType(),
		ProfileName:         "",
		CurrentTick:         engineState["current_tick"].(int64),
		TotalOperations:     engineState["total_operations"].(int64),
		CompletedOps:        engineState["completed_ops"].(int64),
		FailedOps:           engineState["failed_ops"].(int64),
		SavedAt:             time.Now(),
		Architecture:        "single_goroutine_sequential",
		IsRunning:           ew.running,
		IsPaused:            ew.paused,           // NEW: Include pause state
		ComplexityLevel:     ew.complexityLevel,
		InputQueueOps:       inputQueueOps,
		PendingResults:      ew.pendingResults,
		RoutingTable:        make(map[string]string),
		ProcessedOps:        ew.processedOps,
		QueuedOps:           ew.queuedOps,
		CompletedOperations: ew.completedOps,
		EngineState:         detailedEngineState,
	}

	// Copy routing table
	for opType, destination := range ew.routingTable {
		state.RoutingTable[opType] = destination
	}

	// Add profile name if available
	if profile := ew.engine.GetProfile(); profile != nil {
		state.ProfileName = profile.Name
	}

	return state
}

// restoreWrapperState restores the wrapper state from saved data
func (ew *EngineWrapper) restoreWrapperState(state *WrapperState) error {
	// Restore wrapper-specific state
	ew.complexityLevel = state.ComplexityLevel
	ew.paused = state.IsPaused           // NEW: Restore pause state
	ew.processedOps = state.ProcessedOps
	ew.queuedOps = state.QueuedOps
	ew.completedOps = state.CompletedOperations

	// Restore routing table
	ew.routingTable = make(map[string]string)
	for opType, destination := range state.RoutingTable {
		ew.routingTable[opType] = destination
	}

	// Restore input queue operations
	for _, op := range state.InputQueueOps {
		select {
		case ew.inputQueue <- op:
			// Successfully queued
		default:
			// Queue is full, skip this operation
			fmt.Printf("Warning: Skipped restoring operation %s - queue full\n", op.ID)
		}
	}

	// Restore pending results
	ew.pendingResults = make([]*OperationResult, len(state.PendingResults))
	copy(ew.pendingResults, state.PendingResults)

	// Restore engine complexity level
	err := ew.engine.SetComplexityLevel(state.ComplexityLevel)
	if err != nil {
		return fmt.Errorf("failed to restore engine complexity level: %w", err)
	}

	// Try to restore detailed engine state if available
	if engineState, ok := state.EngineState["type"].(string); ok && engineState == "memory_engine_detailed" {
		if stateData, ok := state.EngineState["data"].(string); ok {
			if memEngine, ok := ew.engine.(*MemoryEngine); ok {
				if err := memEngine.LoadEngineState([]byte(stateData)); err != nil {
					fmt.Printf("Warning: Failed to restore detailed memory engine state: %v\n", err)
				} else {
					fmt.Printf("✅ Restored detailed memory engine state\n")
				}
			}
		}
	}

	fmt.Printf("Restored wrapper state: %d queued ops, %d pending results, routing rules: %d\n",
		len(state.InputQueueOps), len(state.PendingResults), len(state.RoutingTable))

	return nil
}

// SetStateDirectory sets the directory for state persistence files
func (ew *EngineWrapper) SetStateDirectory(dir string) {
	ew.mutex.Lock()
	defer ew.mutex.Unlock()
	ew.stateDir = dir
}

// SaveState saves the complete wrapper and engine state to a file
func (ew *EngineWrapper) SaveState() error {
	ew.mutex.RLock()
	defer ew.mutex.RUnlock()

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(ew.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Extract complete state
	state := ew.extractWrapperState()

	// Generate filename
	filename := fmt.Sprintf("%s_%s_state.json",
		state.EngineType.String(),
		state.EngineID)
	filePath := filepath.Join(ew.stateDir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal wrapper state: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	fmt.Printf("✅ Saved wrapper state for %s (tick: %d) to %s\n",
		state.EngineID, state.CurrentTick, filePath)
	return nil
}

// LoadState loads and restores the complete wrapper state from a file
func (ew *EngineWrapper) LoadState(engineID string) error {
	ew.mutex.Lock()
	defer ew.mutex.Unlock()

	// Generate filename
	filename := fmt.Sprintf("%s_%s_state.json",
		ew.engine.GetEngineType().String(),
		engineID)
	filePath := filepath.Join(ew.stateDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("state file not found: %s", filePath)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// Unmarshal JSON
	var state WrapperState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal wrapper state: %w", err)
	}

	// Restore state
	err = ew.restoreWrapperState(&state)
	if err != nil {
		return fmt.Errorf("failed to restore wrapper state: %w", err)
	}

	fmt.Printf("✅ Restored wrapper state for %s (tick: %d) from %s\n",
		engineID, state.CurrentTick, filePath)
	return nil
}
