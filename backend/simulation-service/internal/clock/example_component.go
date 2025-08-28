package clock

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// ExampleComponent demonstrates how to implement the Component interface
// This is a simple example that can be used for testing the clock system
type ExampleComponent struct {
	ID                string
	ProcessingQueue   []ProcessingOperation
	InputQueue        []Operation
	OutputQueue       []OperationResult
	Health            bool
	ProcessedCount    int64
	LastProcessedTick int64

	// Goroutine communication
	tickChannel       chan int64      // Receives tick notifications
	stopChannel       chan struct{}   // Signals shutdown
	running           bool            // Component state
	mutex             sync.RWMutex    // Protects component state
}

// ProcessingOperation represents an operation being processed
type ProcessingOperation struct {
	Operation      Operation
	StartTick      int64
	CompletionTick int64
}

// Operation represents a unit of work
type Operation struct {
	ID          string
	Type        string
	Data        interface{}
	ProcessTime time.Duration
}

// OperationResult represents the result of a completed operation
type OperationResult struct {
	OperationID   string
	CompletedTick int64
	ProcessTime   time.Duration
	Success       bool
	Data          interface{}
}

// NewExampleComponent creates a new example component
func NewExampleComponent(id string) *ExampleComponent {
	return &ExampleComponent{
		ID:                id,
		ProcessingQueue:   make([]ProcessingOperation, 0),
		InputQueue:        make([]Operation, 0),
		OutputQueue:       make([]OperationResult, 0),
		Health:            true,
		ProcessedCount:    0,
		LastProcessedTick: 0,

		// Goroutine communication channels
		tickChannel:       make(chan int64, 100),  // Large buffer for reliability
		stopChannel:       make(chan struct{}),
		running:           false,
	}
}

// ProcessTick implements the Component interface
// This method is called from the component's goroutine when a tick is received
func (ec *ExampleComponent) ProcessTick(currentTick int64) error {
	// ATOMIC tick processing - guaranteed to complete
	ec.mutex.Lock()
	defer ec.mutex.Unlock()

	ec.LastProcessedTick = currentTick

	// Stage 1: Start new operations from input queue (instant)
	err := ec.startNewOperations(currentTick)
	if err != nil {
		return fmt.Errorf("failed to start new operations: %w", err)
	}

	// Stage 2: Check for completed operations (instant)
	err = ec.completeFinishedOperations(currentTick)
	if err != nil {
		return fmt.Errorf("failed to complete operations: %w", err)
	}

	// Stage 3: Update component state (instant)
	ec.updateComponentState(currentTick)

	// Processing complete - ready for next tick
	return nil
}

// GetID implements the Component interface
func (ec *ExampleComponent) GetID() string {
	return ec.ID
}

// IsHealthy implements the Component interface
func (ec *ExampleComponent) IsHealthy() bool {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	return ec.Health
}

// GetTickChannel implements the Component interface
func (ec *ExampleComponent) GetTickChannel() chan int64 {
	return ec.tickChannel
}

// Start implements the Component interface - starts the component's goroutine
func (ec *ExampleComponent) Start(ctx context.Context) error {
	ec.mutex.Lock()
	if ec.running {
		ec.mutex.Unlock()
		return fmt.Errorf("component %s already running", ec.ID)
	}
	ec.running = true
	ec.mutex.Unlock()

	// Start the component's main goroutine
	go ec.runComponentLoop(ctx)

	log.Printf("Component %s: Started goroutine", ec.ID)
	return nil
}

// Stop implements the Component interface - gracefully shuts down the component
func (ec *ExampleComponent) Stop() error {
	ec.mutex.Lock()
	if !ec.running {
		ec.mutex.Unlock()
		return fmt.Errorf("component %s not running", ec.ID)
	}
	ec.running = false
	ec.mutex.Unlock()

	// Signal stop
	close(ec.stopChannel)

	log.Printf("Component %s: Stopped", ec.ID)
	return nil
}

// runComponentLoop is the main goroutine loop that receives tick notifications
func (ec *ExampleComponent) runComponentLoop(ctx context.Context) {
	log.Printf("Component %s: Starting main goroutine loop", ec.ID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Component %s: Stopping due to context cancellation", ec.ID)
			return

		case <-ec.stopChannel:
			log.Printf("Component %s: Stopping due to stop signal", ec.ID)
			return

		case currentTick := <-ec.tickChannel:
			// PERFECT TICK RECEPTION!
			// This goroutine receives the tick and processes it
			startTime := time.Now()

			err := ec.ProcessTick(currentTick)

			processingTime := time.Since(startTime)

			// Monitor processing time to detect performance issues
			if processingTime > TICK_DURATION/2 {
				log.Printf("WARNING: Component %s took %v to process tick %d (> 50%% of tick duration)",
					ec.ID, processingTime, currentTick)
			}

			if err != nil {
				log.Printf("Component %s: Error processing tick %d: %v",
					ec.ID, currentTick, err)
			}
		}
	}
}

// startNewOperations processes new operations from the input queue
func (ec *ExampleComponent) startNewOperations(currentTick int64) error {
	// Process up to 3 operations per tick (simulating capacity limits)
	maxNewOperations := 3
	processed := 0
	
	for len(ec.InputQueue) > 0 && processed < maxNewOperations {
		operation := ec.InputQueue[0]
		ec.InputQueue = ec.InputQueue[1:] // Remove from input queue
		
		// Calculate completion tick
		processingTicks := DurationToTicks(operation.ProcessTime)
		completionTick := currentTick + processingTicks
		
		// Add to processing queue
		ec.ProcessingQueue = append(ec.ProcessingQueue, ProcessingOperation{
			Operation:      operation,
			StartTick:      currentTick,
			CompletionTick: completionTick,
		})
		
		processed++
		log.Printf("Component %s: Started operation %s (completes at tick %d)", 
			ec.ID, operation.ID, completionTick)
	}
	
	return nil
}

// completeFinishedOperations moves completed operations to output queue
func (ec *ExampleComponent) completeFinishedOperations(currentTick int64) error {
	// Check processing queue for completed operations
	for i := len(ec.ProcessingQueue) - 1; i >= 0; i-- {
		procOp := ec.ProcessingQueue[i]
		
		if currentTick >= procOp.CompletionTick {
			// Operation is complete
			result := OperationResult{
				OperationID:   procOp.Operation.ID,
				CompletedTick: currentTick,
				ProcessTime:   procOp.Operation.ProcessTime,
				Success:       true, // Assume success for this example
				Data:          procOp.Operation.Data,
			}
			
			// Add to output queue
			ec.OutputQueue = append(ec.OutputQueue, result)
			
			// Remove from processing queue
			ec.ProcessingQueue = append(ec.ProcessingQueue[:i], ec.ProcessingQueue[i+1:]...)
			
			ec.ProcessedCount++
			log.Printf("Component %s: Completed operation %s at tick %d", 
				ec.ID, result.OperationID, currentTick)
		}
	}
	
	return nil
}

// updateComponentState updates internal component state
func (ec *ExampleComponent) updateComponentState(currentTick int64) {
	// Simulate occasional health issues (very rare)
	if rand.Float64() < 0.001 { // 0.1% chance per tick
		ec.Health = false
		log.Printf("Component %s: Health degraded at tick %d", ec.ID, currentTick)
	} else if !ec.Health && rand.Float64() < 0.1 { // 10% chance to recover
		ec.Health = true
		log.Printf("Component %s: Health recovered at tick %d", ec.ID, currentTick)
	}
}

// AddOperation adds a new operation to the input queue
func (ec *ExampleComponent) AddOperation(op Operation) {
	ec.InputQueue = append(ec.InputQueue, op)
}

// GetOutputResults returns and clears the output queue
func (ec *ExampleComponent) GetOutputResults() []OperationResult {
	results := make([]OperationResult, len(ec.OutputQueue))
	copy(results, ec.OutputQueue)
	ec.OutputQueue = ec.OutputQueue[:0] // Clear the queue
	return results
}

// GetStatus returns current component status
func (ec *ExampleComponent) GetStatus() ComponentStatus {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()

	return ComponentStatus{
		ID:                ec.ID,
		Health:            ec.Health,
		ProcessedCount:    ec.ProcessedCount,
		LastProcessedTick: ec.LastProcessedTick,
		InputQueueLength:  len(ec.InputQueue),
		ProcessingCount:   len(ec.ProcessingQueue),
		OutputQueueLength: len(ec.OutputQueue),
		Running:           ec.running,
		TickChannelLength: len(ec.tickChannel),
	}
}

// ComponentStatus represents the status of a component
type ComponentStatus struct {
	ID                string `json:"id"`
	Health            bool   `json:"health"`
	ProcessedCount    int64  `json:"processed_count"`
	LastProcessedTick int64  `json:"last_processed_tick"`
	InputQueueLength  int    `json:"input_queue_length"`
	ProcessingCount   int    `json:"processing_count"`
	OutputQueueLength int    `json:"output_queue_length"`
	Running           bool   `json:"running"`
	TickChannelLength int    `json:"tick_channel_length"`
}

// CreateExampleOperations creates sample operations for testing
func CreateExampleOperations(count int) []Operation {
	operations := make([]Operation, count)
	
	for i := 0; i < count; i++ {
		// Random processing time between 0.1ms and 5ms
		processTime := time.Duration(rand.Intn(4900)+100) * time.Microsecond
		
		operations[i] = Operation{
			ID:          fmt.Sprintf("op_%d", i+1),
			Type:        "example",
			Data:        fmt.Sprintf("data_%d", i+1),
			ProcessTime: processTime,
		}
	}
	
	return operations
}

// RunExampleSimulation demonstrates how to use the clock system with goroutine-based components
func RunExampleSimulation() {
	log.Println("Starting example simulation with goroutine-based tick notification...")

	// Create global tick coordinator
	coordinator := NewGlobalTickCoordinator()

	// Create example components
	comp1 := NewExampleComponent("component_1")
	comp2 := NewExampleComponent("component_2")
	comp3 := NewExampleComponent("component_3")

	// Create context for component lifecycle management
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Register and start components (each gets its own goroutine)
	err := coordinator.RegisterComponent(comp1, ctx)
	if err != nil {
		log.Printf("Failed to register component 1: %v", err)
		return
	}

	err = coordinator.RegisterComponent(comp2, ctx)
	if err != nil {
		log.Printf("Failed to register component 2: %v", err)
		return
	}

	err = coordinator.RegisterComponent(comp3, ctx)
	if err != nil {
		log.Printf("Failed to register component 3: %v", err)
		return
	}

	// Add some example operations
	operations := CreateExampleOperations(10)
	for i, op := range operations {
		// Distribute operations across components
		switch i % 3 {
		case 0:
			comp1.AddOperation(op)
		case 1:
			comp2.AddOperation(op)
		case 2:
			comp3.AddOperation(op)
		}
	}

	// Start simulation (this starts the global tick coordinator)
	err = coordinator.Start(ctx)
	if err != nil {
		log.Printf("Failed to start simulation: %v", err)
		return
	}

	log.Println("Simulation running... Each component is processing ticks in its own goroutine")

	// Run for 5 seconds
	time.Sleep(5 * time.Second)

	// Get final metrics
	metrics := coordinator.GetPerformanceMetrics()
	log.Printf("Simulation completed:")
	log.Printf("  Total ticks: %d", metrics.TotalTicks)
	log.Printf("  Simulation time: %v", metrics.SimulationTime)
	log.Printf("  Real time elapsed: %v", metrics.RealTimeElapsed)
	log.Printf("  Ticks per second: %.2f", metrics.TicksPerSecond)
	log.Printf("  Average tick time: %v", metrics.AverageTickTime)
	log.Printf("  Efficiency ratio: %.2f", metrics.EfficiencyRatio)

	// Get component statuses
	log.Println("Component statuses:")
	log.Printf("  Component 1: %+v", comp1.GetStatus())
	log.Printf("  Component 2: %+v", comp2.GetStatus())
	log.Printf("  Component 3: %+v", comp3.GetStatus())

	// Stop simulation (this stops all component goroutines)
	coordinator.Stop()

	log.Println("Example simulation completed. All goroutines stopped.")
}
