package components

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// ComponentInstanceExecutor executes engine sequences provided by Load Balancer
// without routing decisions or graph knowledge - pure execution engine
type ComponentInstanceExecutor struct {
	// Identity
	InstanceID    string `json:"instance_id"`
	ComponentID   string `json:"component_id"`
	
	// Engine management
	engines       map[engines.EngineType]*engines.EngineWrapper `json:"-"`
	engineOrder   []engines.EngineType                          `json:"engine_order"`
	
	// Execution state
	isRunning     bool                                          `json:"is_running"`
	currentLoad   float64                                       `json:"current_load"`
	health        float64                                       `json:"health"`
	
	// Communication channels
	inputChannel  chan *EngineSequenceRequest                   `json:"-"`
	outputChannel chan *EngineSequenceResult                    `json:"-"`
	
	// Execution metrics
	metrics       *ExecutionMetrics                             `json:"metrics"`
	
	// Lifecycle management
	ctx           context.Context                               `json:"-"`
	cancel        context.CancelFunc                            `json:"-"`
	mutex         sync.RWMutex                                  `json:"-"`
}

// EngineSequenceRequest represents a request with predefined engine sequence
type EngineSequenceRequest struct {
	// Request information
	Request       *Request                                      `json:"request"`
	
	// Execution sequence (provided by Load Balancer)
	EngineSequence []engines.EngineType                        `json:"engine_sequence"`
	
	// Execution parameters
	Parameters    map[string]interface{}                       `json:"parameters"`
	
	// Timing
	Timestamp     time.Time                                    `json:"timestamp"`
	Timeout       time.Duration                                `json:"timeout"`
	
	// Tracking
	SequenceID    string                                       `json:"sequence_id"`
}

// EngineSequenceResult represents the result of engine sequence execution
type EngineSequenceResult struct {
	// Request information
	Request       *Request                                      `json:"request"`
	SequenceID    string                                       `json:"sequence_id"`
	
	// Execution results
	EngineResults []EngineExecutionResult                      `json:"engine_results"`
	
	// Overall result
	Success       bool                                         `json:"success"`
	ErrorMessage  string                                       `json:"error_message,omitempty"`
	
	// Timing
	StartTime     time.Time                                    `json:"start_time"`
	EndTime       time.Time                                    `json:"end_time"`
	TotalLatency  time.Duration                                `json:"total_latency"`
	
	// Metrics
	EngineCount   int                                          `json:"engine_count"`
}

// EngineExecutionResult represents the result of a single engine execution
type EngineExecutionResult struct {
	EngineType    engines.EngineType                           `json:"engine_type"`
	OperationID   string                                       `json:"operation_id"`
	Success       bool                                         `json:"success"`
	Result        *engines.OperationResult                     `json:"result"`
	StartTime     time.Time                                    `json:"start_time"`
	EndTime       time.Time                                    `json:"end_time"`
	Latency       time.Duration                                `json:"latency"`
	ErrorMessage  string                                       `json:"error_message,omitempty"`
}

// ExecutionMetrics tracks execution performance
type ExecutionMetrics struct {
	TotalSequences        int64         `json:"total_sequences"`
	SuccessfulSequences   int64         `json:"successful_sequences"`
	FailedSequences       int64         `json:"failed_sequences"`
	AvgSequenceLatency    time.Duration `json:"avg_sequence_latency"`
	AvgEnginesPerSequence float64       `json:"avg_engines_per_sequence"`
	TotalEngineExecutions int64         `json:"total_engine_executions"`
	EngineSuccessRate     map[engines.EngineType]float64 `json:"engine_success_rate"`
	LastUpdate            time.Time     `json:"last_update"`
}

// NewComponentInstanceExecutor creates a new component instance executor
func NewComponentInstanceExecutor(instanceID, componentID string) *ComponentInstanceExecutor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ComponentInstanceExecutor{
		InstanceID:    instanceID,
		ComponentID:   componentID,
		engines:       make(map[engines.EngineType]*engines.EngineWrapper),
		engineOrder:   make([]engines.EngineType, 0),
		isRunning:     false,
		currentLoad:   0.0,
		health:        1.0,
		inputChannel:  make(chan *EngineSequenceRequest, 100),
		outputChannel: make(chan *EngineSequenceResult, 100),
		metrics: &ExecutionMetrics{
			EngineSuccessRate: make(map[engines.EngineType]float64),
			LastUpdate:        time.Now(),
		},
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the component instance executor
func (cie *ComponentInstanceExecutor) Start() error {
	cie.mutex.Lock()
	defer cie.mutex.Unlock()
	
	if cie.isRunning {
		return fmt.Errorf("component instance executor %s is already running", cie.InstanceID)
	}
	
	log.Printf("ComponentInstanceExecutor %s: Starting execution engine", cie.InstanceID)
	
	// Start main execution loop
	go cie.run()
	
	cie.isRunning = true
	log.Printf("ComponentInstanceExecutor %s: Started successfully", cie.InstanceID)
	
	return nil
}

// Stop stops the component instance executor
func (cie *ComponentInstanceExecutor) Stop() error {
	cie.mutex.Lock()
	defer cie.mutex.Unlock()
	
	if !cie.isRunning {
		return nil
	}
	
	log.Printf("ComponentInstanceExecutor %s: Stopping execution engine", cie.InstanceID)
	
	// Cancel context to stop main loop
	cie.cancel()
	
	cie.isRunning = false
	log.Printf("ComponentInstanceExecutor %s: Stopped successfully", cie.InstanceID)
	
	return nil
}

// RegisterEngine registers an engine with the executor
func (cie *ComponentInstanceExecutor) RegisterEngine(engineType engines.EngineType, engine *engines.EngineWrapper) error {
	cie.mutex.Lock()
	defer cie.mutex.Unlock()
	
	cie.engines[engineType] = engine
	
	// Add to engine order if not already present
	for _, existingType := range cie.engineOrder {
		if existingType == engineType {
			log.Printf("ComponentInstanceExecutor %s: Updated engine %s", cie.InstanceID, engineType)
			return nil
		}
	}
	
	cie.engineOrder = append(cie.engineOrder, engineType)
	log.Printf("ComponentInstanceExecutor %s: Registered engine %s", cie.InstanceID, engineType)
	
	return nil
}

// ExecuteSequence executes an engine sequence (external API)
func (cie *ComponentInstanceExecutor) ExecuteSequence(request *EngineSequenceRequest) error {
	select {
	case cie.inputChannel <- request:
		return nil
	default:
		return fmt.Errorf("executor %s input channel is full", cie.InstanceID)
	}
}

// GetOutputChannel returns the output channel for results
func (cie *ComponentInstanceExecutor) GetOutputChannel() <-chan *EngineSequenceResult {
	return cie.outputChannel
}

// run is the main execution loop
func (cie *ComponentInstanceExecutor) run() {
	log.Printf("ComponentInstanceExecutor %s: Starting main execution loop", cie.InstanceID)
	
	for {
		select {
		case request := <-cie.inputChannel:
			result := cie.executeEngineSequence(request)
			
			// Send result to output channel
			select {
			case cie.outputChannel <- result:
				// Result sent successfully
			default:
				log.Printf("ComponentInstanceExecutor %s: Output channel full, dropping result for sequence %s", 
					cie.InstanceID, request.SequenceID)
			}
			
		case <-cie.ctx.Done():
			log.Printf("ComponentInstanceExecutor %s: Main execution loop stopping", cie.InstanceID)
			return
		}
	}
}

// executeEngineSequence executes the provided engine sequence
func (cie *ComponentInstanceExecutor) executeEngineSequence(request *EngineSequenceRequest) *EngineSequenceResult {
	startTime := time.Now()
	
	log.Printf("ComponentInstanceExecutor %s: Executing sequence %s with %d engines", 
		cie.InstanceID, request.SequenceID, len(request.EngineSequence))
	
	result := &EngineSequenceResult{
		Request:       request.Request,
		SequenceID:    request.SequenceID,
		EngineResults: make([]EngineExecutionResult, 0, len(request.EngineSequence)),
		Success:       true,
		StartTime:     startTime,
		EngineCount:   len(request.EngineSequence),
	}
	
	// Execute engines in sequence
	var lastResult *engines.OperationResult
	
	for i, engineType := range request.EngineSequence {
		engineResult := cie.executeEngine(engineType, request, lastResult, i)
		result.EngineResults = append(result.EngineResults, engineResult)
		
		// If engine execution failed, stop sequence
		if !engineResult.Success {
			result.Success = false
			result.ErrorMessage = fmt.Sprintf("Engine %s failed: %s", engineType, engineResult.ErrorMessage)
			break
		}
		
		// Use this engine's result as input for next engine
		lastResult = engineResult.Result
	}
	
	// Complete result
	result.EndTime = time.Now()
	result.TotalLatency = result.EndTime.Sub(result.StartTime)
	
	// Update metrics
	cie.updateMetrics(result)
	
	// Update request with final result
	if result.Success {
		request.Request.MarkComplete()
	} else {
		request.Request.MarkFailed()
	}
	
	log.Printf("ComponentInstanceExecutor %s: Completed sequence %s (success: %t, latency: %v)", 
		cie.InstanceID, request.SequenceID, result.Success, result.TotalLatency)
	
	return result
}

// executeEngine executes a single engine in the sequence
func (cie *ComponentInstanceExecutor) executeEngine(engineType engines.EngineType, request *EngineSequenceRequest, 
	previousResult *engines.OperationResult, sequenceIndex int) EngineExecutionResult {
	
	startTime := time.Now()
	
	engineResult := EngineExecutionResult{
		EngineType:  engineType,
		OperationID: fmt.Sprintf("%s_%s_%d", request.SequenceID, engineType, sequenceIndex),
		StartTime:   startTime,
	}
	
	// Get engine
	engine, exists := cie.engines[engineType]
	if !exists {
		engineResult.Success = false
		engineResult.ErrorMessage = fmt.Sprintf("Engine %s not found", engineType)
		engineResult.EndTime = time.Now()
		engineResult.Latency = engineResult.EndTime.Sub(engineResult.StartTime)
		return engineResult
	}
	
	// Create operation for engine
	operation := cie.createOperationForEngine(engineType, request, previousResult, sequenceIndex)
	
	// Execute operation on engine
	result, err := cie.executeOperationOnEngine(engine, operation)
	if err != nil {
		engineResult.Success = false
		engineResult.ErrorMessage = err.Error()
	} else {
		engineResult.Success = result.Success
		engineResult.Result = result
		if !result.Success {
			engineResult.ErrorMessage = "Engine operation failed"
		}
	}
	
	engineResult.EndTime = time.Now()
	engineResult.Latency = engineResult.EndTime.Sub(engineResult.StartTime)
	
	// Update request position
	request.Request.SetCurrentPosition(cie.ComponentID, string(engineType), "executing")
	request.Request.IncrementEngineCount()
	
	// Add to request history if tracking enabled
	request.Request.AddToHistory(cie.ComponentID, string(engineType), operation.Type, engineResult.Success)
	
	return engineResult
}

// createOperationForEngine creates an operation for a specific engine
func (cie *ComponentInstanceExecutor) createOperationForEngine(engineType engines.EngineType, 
	request *EngineSequenceRequest, previousResult *engines.OperationResult, sequenceIndex int) *engines.Operation {
	
	// Determine operation type based on engine type and previous result
	operationType := cie.determineOperationType(engineType, previousResult)
	
	// Determine operation data
	var operationData interface{}
	if previousResult != nil {
		operationData = previousResult.Data
	} else {
		operationData = request.Request.Data.Payload
	}
	
	return &engines.Operation{
		ID:          fmt.Sprintf("%s_%s_%d", request.SequenceID, engineType, sequenceIndex),
		Type:        operationType,
		Data:        operationData,
		RequestID:   request.Request.ID,
		ComponentID: cie.ComponentID,
		Priority:    1, // Default priority
		Timestamp:   time.Now(),
		Metadata:    request.Parameters,
	}
}

// determineOperationType determines the operation type for an engine
func (cie *ComponentInstanceExecutor) determineOperationType(engineType engines.EngineType, previousResult *engines.OperationResult) string {
	switch engineType {
	case engines.CPUEngine:
		if previousResult != nil {
			return "process_data"
		}
		return "parse_request"
		
	case engines.MemoryEngine:
		if previousResult != nil && previousResult.Success {
			return "cache_store"
		}
		return "cache_lookup"
		
	case engines.StorageEngine:
		if previousResult != nil && previousResult.Success {
			return "data_write"
		}
		return "data_read"
		
	default:
		return "generic_operation"
	}
}

// executeOperationOnEngine executes an operation on a specific engine
func (cie *ComponentInstanceExecutor) executeOperationOnEngine(engine *engines.EngineWrapper, operation *engines.Operation) (*engines.OperationResult, error) {
	// Submit operation to engine
	if err := engine.SubmitOperation(operation); err != nil {
		return nil, fmt.Errorf("failed to submit operation: %w", err)
	}
	
	// Wait for result (simplified - in real implementation would use channels)
	// For now, simulate engine execution
	time.Sleep(10 * time.Millisecond) // Simulate processing time
	
	// Create mock result (in real implementation, would get from engine)
	result := &engines.OperationResult{
		OperationID:   operation.ID,
		OperationType: operation.Type,
		Success:       true,
		Data:          operation.Data,
		Timestamp:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}
	
	return result, nil
}

// updateMetrics updates execution metrics
func (cie *ComponentInstanceExecutor) updateMetrics(result *EngineSequenceResult) {
	cie.mutex.Lock()
	defer cie.mutex.Unlock()
	
	cie.metrics.TotalSequences++
	
	if result.Success {
		cie.metrics.SuccessfulSequences++
	} else {
		cie.metrics.FailedSequences++
	}
	
	// Update average sequence latency
	if cie.metrics.AvgSequenceLatency == 0 {
		cie.metrics.AvgSequenceLatency = result.TotalLatency
	} else {
		// Weighted average
		cie.metrics.AvgSequenceLatency = time.Duration(
			float64(cie.metrics.AvgSequenceLatency)*0.9 + float64(result.TotalLatency)*0.1)
	}
	
	// Update engines per sequence
	engineCount := float64(result.EngineCount)
	if cie.metrics.AvgEnginesPerSequence == 0 {
		cie.metrics.AvgEnginesPerSequence = engineCount
	} else {
		cie.metrics.AvgEnginesPerSequence = cie.metrics.AvgEnginesPerSequence*0.9 + engineCount*0.1
	}
	
	// Update engine-specific metrics
	cie.metrics.TotalEngineExecutions += int64(result.EngineCount)
	
	for _, engineResult := range result.EngineResults {
		currentRate := cie.metrics.EngineSuccessRate[engineResult.EngineType]
		if engineResult.Success {
			cie.metrics.EngineSuccessRate[engineResult.EngineType] = currentRate*0.9 + 0.1
		} else {
			cie.metrics.EngineSuccessRate[engineResult.EngineType] = currentRate * 0.9
		}
	}
	
	cie.metrics.LastUpdate = time.Now()
}

// GetMetrics returns current execution metrics
func (cie *ComponentInstanceExecutor) GetMetrics() *ExecutionMetrics {
	cie.mutex.RLock()
	defer cie.mutex.RUnlock()
	
	// Return a copy
	metrics := &ExecutionMetrics{
		TotalSequences:        cie.metrics.TotalSequences,
		SuccessfulSequences:   cie.metrics.SuccessfulSequences,
		FailedSequences:       cie.metrics.FailedSequences,
		AvgSequenceLatency:    cie.metrics.AvgSequenceLatency,
		AvgEnginesPerSequence: cie.metrics.AvgEnginesPerSequence,
		TotalEngineExecutions: cie.metrics.TotalEngineExecutions,
		EngineSuccessRate:     make(map[engines.EngineType]float64),
		LastUpdate:            cie.metrics.LastUpdate,
	}
	
	for engineType, rate := range cie.metrics.EngineSuccessRate {
		metrics.EngineSuccessRate[engineType] = rate
	}
	
	return metrics
}

// GetHealth returns the current health of the executor
func (cie *ComponentInstanceExecutor) GetHealth() float64 {
	cie.mutex.RLock()
	defer cie.mutex.RUnlock()
	return cie.health
}

// GetCurrentLoad returns the current load of the executor
func (cie *ComponentInstanceExecutor) GetCurrentLoad() float64 {
	cie.mutex.RLock()
	defer cie.mutex.RUnlock()
	return cie.currentLoad
}

// IsRunning returns whether the executor is running
func (cie *ComponentInstanceExecutor) IsRunning() bool {
	cie.mutex.RLock()
	defer cie.mutex.RUnlock()
	return cie.isRunning
}
