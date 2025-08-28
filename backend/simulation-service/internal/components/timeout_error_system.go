package components

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// TimeoutErrorSystem manages request timeouts, backpressure, and failure injection
type TimeoutErrorSystem struct {
	// Configuration
	config *TimeoutErrorConfig

	// Timeout management
	timeoutManager *RequestTimeoutManager
	
	// Backpressure management
	backpressureManager *BackpressureManager
	
	// Failure injection for educational scenarios
	failureInjector *FailureInjector
	
	// Error handling
	errorHandler *ErrorHandler
	
	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// TimeoutErrorConfig defines timeout and error handling configuration
type TimeoutErrorConfig struct {
	// Timeout settings
	DefaultTimeout      time.Duration `json:"default_timeout"`
	ComponentTimeouts   map[string]time.Duration `json:"component_timeouts"`
	EngineTimeouts      map[string]time.Duration `json:"engine_timeouts"`
	
	// Backpressure settings
	BackpressureEnabled bool          `json:"backpressure_enabled"`
	MaxQueueSize        int           `json:"max_queue_size"`
	BackpressureThreshold float64     `json:"backpressure_threshold"`
	
	// Failure injection settings
	FailureInjectionEnabled bool      `json:"failure_injection_enabled"`
	FailureRate            float64    `json:"failure_rate"`
	EducationalMode        bool       `json:"educational_mode"`
	
	// Error handling settings
	RetryEnabled           bool       `json:"retry_enabled"`
	MaxRetries            int        `json:"max_retries"`
	RetryBackoff          time.Duration `json:"retry_backoff"`
}

// BackpressureManager manages natural backpressure in the system
type BackpressureManager struct {
	// Configuration
	enabled           bool
	maxQueueSize      int
	threshold         float64
	
	// Queue monitoring
	queueSizes        map[string]int
	queueCapacities   map[string]int
	backpressureState map[string]bool
	mutex             sync.RWMutex
	
	// Metrics
	backpressureEvents int64
	droppedRequests    int64
	
	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
}

// FailureInjector injects failures for educational scenarios
type FailureInjector struct {
	// Configuration
	enabled         bool
	failureRate     float64
	educationalMode bool
	
	// Failure scenarios
	scenarios       []FailureScenario
	currentScenario int
	
	// Random generator
	random *rand.Rand
	mutex  sync.Mutex
	
	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
}

// FailureScenario defines an educational failure scenario
type FailureScenario struct {
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	FailureType     FailureType   `json:"failure_type"`
	TargetComponent string        `json:"target_component"`
	Duration        time.Duration `json:"duration"`
	Probability     float64       `json:"probability"`
}

// FailureType defines types of failures that can be injected
type FailureType string

const (
	FailureTimeout     FailureType = "timeout"
	FailureNetworkLoss FailureType = "network_loss"
	FailureHighLatency FailureType = "high_latency"
	FailureMemoryLeak  FailureType = "memory_leak"
	FailureCPUSpike    FailureType = "cpu_spike"
	FailureStorageFull FailureType = "storage_full"
)

// ErrorHandler handles various types of errors and recovery
type ErrorHandler struct {
	// Configuration
	retryEnabled bool
	maxRetries   int
	retryBackoff time.Duration
	
	// Error tracking
	errorCounts    map[string]int64
	errorTypes     map[string]map[string]int64
	recoveryStats  map[string]*RecoveryStats
	mutex          sync.RWMutex
	
	// Global registry for error routing
	globalRegistry GlobalRegistryInterface
}

// RecoveryStats tracks error recovery statistics
type RecoveryStats struct {
	TotalErrors     int64         `json:"total_errors"`
	RecoveredErrors int64         `json:"recovered_errors"`
	FailedRecovery  int64         `json:"failed_recovery"`
	AvgRecoveryTime time.Duration `json:"avg_recovery_time"`
}

// NewTimeoutErrorSystem creates a new timeout and error system
func NewTimeoutErrorSystem(config *TimeoutErrorConfig, globalRegistry GlobalRegistryInterface) *TimeoutErrorSystem {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &TimeoutErrorSystem{
		config:              config,
		timeoutManager:      NewRequestTimeoutManager(config.DefaultTimeout),
		backpressureManager: NewBackpressureManager(config),
		failureInjector:     NewFailureInjector(config),
		errorHandler:        NewErrorHandler(config, globalRegistry),
		ctx:                 ctx,
		cancel:              cancel,
	}
}

// Start starts the timeout and error system
func (tes *TimeoutErrorSystem) Start() error {
	log.Printf("TimeoutErrorSystem: Starting timeout and error handling system")
	
	tes.timeoutManager.Start()
	tes.backpressureManager.Start()
	tes.errorHandler.Start()
	
	if tes.config.FailureInjectionEnabled {
		tes.failureInjector.Start()
	}
	
	return nil
}

// Stop stops the timeout and error system
func (tes *TimeoutErrorSystem) Stop() error {
	log.Printf("TimeoutErrorSystem: Stopping timeout and error handling system")
	
	tes.cancel()
	tes.timeoutManager.Stop()
	tes.backpressureManager.Stop()
	tes.failureInjector.Stop()
	tes.errorHandler.Stop()
	
	return nil
}

// NewBackpressureManager creates a new backpressure manager
func NewBackpressureManager(config *TimeoutErrorConfig) *BackpressureManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &BackpressureManager{
		enabled:           config.BackpressureEnabled,
		maxQueueSize:      config.MaxQueueSize,
		threshold:         config.BackpressureThreshold,
		queueSizes:        make(map[string]int),
		queueCapacities:   make(map[string]int),
		backpressureState: make(map[string]bool),
		ctx:               ctx,
		cancel:            cancel,
		ticker:            time.NewTicker(1 * time.Second),
	}
}

// Start starts the backpressure manager
func (bm *BackpressureManager) Start() {
	if !bm.enabled {
		return
	}
	
	log.Printf("BackpressureManager: Starting backpressure monitoring")
	go bm.run()
}

// Stop stops the backpressure manager
func (bm *BackpressureManager) Stop() {
	bm.cancel()
	bm.ticker.Stop()
}

// run monitors queue sizes and applies backpressure
func (bm *BackpressureManager) run() {
	for {
		select {
		case <-bm.ticker.C:
			bm.checkBackpressure()
		case <-bm.ctx.Done():
			return
		}
	}
}

// checkBackpressure checks if backpressure should be applied
func (bm *BackpressureManager) checkBackpressure() {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()
	
	for componentID, queueSize := range bm.queueSizes {
		capacity := bm.queueCapacities[componentID]
		if capacity == 0 {
			capacity = bm.maxQueueSize
		}
		
		utilization := float64(queueSize) / float64(capacity)
		
		if utilization > bm.threshold {
			if !bm.backpressureState[componentID] {
				bm.backpressureState[componentID] = true
				bm.backpressureEvents++
				log.Printf("BackpressureManager: Applying backpressure to %s (utilization: %.2f)", 
					componentID, utilization)
			}
		} else if utilization < bm.threshold*0.8 {
			if bm.backpressureState[componentID] {
				bm.backpressureState[componentID] = false
				log.Printf("BackpressureManager: Releasing backpressure from %s (utilization: %.2f)", 
					componentID, utilization)
			}
		}
	}
}

// ShouldApplyBackpressure checks if backpressure should be applied to a component
func (bm *BackpressureManager) ShouldApplyBackpressure(componentID string) bool {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	
	return bm.backpressureState[componentID]
}

// UpdateQueueSize updates the queue size for a component
func (bm *BackpressureManager) UpdateQueueSize(componentID string, size, capacity int) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()
	
	bm.queueSizes[componentID] = size
	bm.queueCapacities[componentID] = capacity
}

// NewFailureInjector creates a new failure injector
func NewFailureInjector(config *TimeoutErrorConfig) *FailureInjector {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &FailureInjector{
		enabled:         config.FailureInjectionEnabled,
		failureRate:     config.FailureRate,
		educationalMode: config.EducationalMode,
		scenarios:       createDefaultFailureScenarios(),
		currentScenario: 0,
		random:          rand.New(rand.NewSource(time.Now().UnixNano())),
		ctx:             ctx,
		cancel:          cancel,
		ticker:          time.NewTicker(10 * time.Second),
	}
}

// Start starts the failure injector
func (fi *FailureInjector) Start() {
	if !fi.enabled {
		return
	}
	
	log.Printf("FailureInjector: Starting failure injection (rate: %.2f)", fi.failureRate)
	go fi.run()
}

// Stop stops the failure injector
func (fi *FailureInjector) Stop() {
	fi.cancel()
	fi.ticker.Stop()
}

// run executes failure injection scenarios
func (fi *FailureInjector) run() {
	for {
		select {
		case <-fi.ticker.C:
			if fi.educationalMode {
				fi.executeScenario()
			} else {
				fi.injectRandomFailure()
			}
		case <-fi.ctx.Done():
			return
		}
	}
}

// ShouldInjectFailure determines if a failure should be injected
func (fi *FailureInjector) ShouldInjectFailure(componentID string, operationType string) (bool, FailureType) {
	fi.mutex.Lock()
	defer fi.mutex.Unlock()
	
	if !fi.enabled {
		return false, ""
	}
	
	if fi.random.Float64() < fi.failureRate {
		// Select random failure type
		failureTypes := []FailureType{
			FailureTimeout, FailureNetworkLoss, FailureHighLatency,
		}
		
		selectedType := failureTypes[fi.random.Intn(len(failureTypes))]
		return true, selectedType
	}
	
	return false, ""
}

// executeScenario executes an educational failure scenario
func (fi *FailureInjector) executeScenario() {
	if len(fi.scenarios) == 0 {
		return
	}
	
	scenario := fi.scenarios[fi.currentScenario]
	log.Printf("FailureInjector: Executing scenario '%s': %s", scenario.Name, scenario.Description)
	
	// Execute scenario (simplified implementation)
	fi.currentScenario = (fi.currentScenario + 1) % len(fi.scenarios)
}

// injectRandomFailure injects random failures for testing
func (fi *FailureInjector) injectRandomFailure() {
	// Random failure injection logic
	if fi.random.Float64() < fi.failureRate {
		log.Printf("FailureInjector: Injecting random failure")
	}
}

// createDefaultFailureScenarios creates default failure scenarios
func createDefaultFailureScenarios() []FailureScenario {
	return []FailureScenario{
		{
			Name:            "Network Timeout",
			Description:     "Simulate network timeout to demonstrate timeout handling",
			FailureType:     FailureTimeout,
			TargetComponent: "api_gateway",
			Duration:        30 * time.Second,
			Probability:     0.1,
		},
		{
			Name:            "Database Connection Loss",
			Description:     "Simulate database connection loss for error recovery demonstration",
			FailureType:     FailureNetworkLoss,
			TargetComponent: "database",
			Duration:        45 * time.Second,
			Probability:     0.05,
		},
		{
			Name:            "High Latency Spike",
			Description:     "Simulate high latency to show backpressure effects",
			FailureType:     FailureHighLatency,
			TargetComponent: "storage",
			Duration:        60 * time.Second,
			Probability:     0.15,
		},
	}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(config *TimeoutErrorConfig, globalRegistry GlobalRegistryInterface) *ErrorHandler {
	return &ErrorHandler{
		retryEnabled:   config.RetryEnabled,
		maxRetries:     config.MaxRetries,
		retryBackoff:   config.RetryBackoff,
		errorCounts:    make(map[string]int64),
		errorTypes:     make(map[string]map[string]int64),
		recoveryStats:  make(map[string]*RecoveryStats),
		globalRegistry: globalRegistry,
	}
}

// Start starts the error handler
func (eh *ErrorHandler) Start() {
	log.Printf("ErrorHandler: Starting error handling system")
}

// Stop stops the error handler
func (eh *ErrorHandler) Stop() {
	log.Printf("ErrorHandler: Stopping error handling system")
}

// HandleError handles various types of errors with recovery attempts
func (eh *ErrorHandler) HandleError(componentID string, errorType string, err error, request *Request) error {
	eh.mutex.Lock()
	defer eh.mutex.Unlock()
	
	// Track error
	eh.errorCounts[componentID]++
	
	if eh.errorTypes[componentID] == nil {
		eh.errorTypes[componentID] = make(map[string]int64)
	}
	eh.errorTypes[componentID][errorType]++
	
	// Initialize recovery stats if needed
	if eh.recoveryStats[componentID] == nil {
		eh.recoveryStats[componentID] = &RecoveryStats{}
	}
	
	stats := eh.recoveryStats[componentID]
	stats.TotalErrors++
	
	log.Printf("ErrorHandler: Handling %s error in %s: %v", errorType, componentID, err)
	
	// Attempt recovery based on error type
	if eh.retryEnabled {
		return eh.attemptRecovery(componentID, errorType, err, request, stats)
	}
	
	// Route to error end node
	return eh.routeToErrorEndNode(request, errorType, err.Error())
}

// attemptRecovery attempts to recover from an error
func (eh *ErrorHandler) attemptRecovery(componentID, errorType string, err error, request *Request, stats *RecoveryStats) error {
	startTime := time.Now()
	
	for attempt := 1; attempt <= eh.maxRetries; attempt++ {
		log.Printf("ErrorHandler: Recovery attempt %d/%d for %s in %s", 
			attempt, eh.maxRetries, errorType, componentID)
		
		// Wait before retry
		time.Sleep(eh.retryBackoff * time.Duration(attempt))
		
		// Attempt recovery (simplified - would contain actual recovery logic)
		if eh.simulateRecovery(errorType) {
			stats.RecoveredErrors++
			recoveryTime := time.Since(startTime)
			eh.updateAverageRecoveryTime(stats, recoveryTime)
			
			log.Printf("ErrorHandler: Successfully recovered from %s error in %s after %d attempts", 
				errorType, componentID, attempt)
			return nil
		}
	}
	
	// Recovery failed
	stats.FailedRecovery++
	log.Printf("ErrorHandler: Failed to recover from %s error in %s after %d attempts", 
		errorType, componentID, eh.maxRetries)
	
	return eh.routeToErrorEndNode(request, errorType, err.Error())
}

// simulateRecovery simulates recovery success/failure
func (eh *ErrorHandler) simulateRecovery(errorType string) bool {
	// Different recovery rates for different error types
	switch errorType {
	case "timeout":
		return rand.Float64() < 0.7 // 70% recovery rate
	case "network_loss":
		return rand.Float64() < 0.5 // 50% recovery rate
	case "high_latency":
		return rand.Float64() < 0.8 // 80% recovery rate
	default:
		return rand.Float64() < 0.6 // 60% default recovery rate
	}
}

// routeToErrorEndNode routes a failed request to error end node
func (eh *ErrorHandler) routeToErrorEndNode(request *Request, errorType, errorMessage string) error {
	log.Printf("ErrorHandler: Routing request %s to error end node (error: %s)", 
		request.ID, errorType)
	
	// Mark request as failed
	request.MarkFailed()
	
	// Add error information to request history
	request.AddToHistory("error_handler", "error", errorType, errorMessage)
	
	// In a full implementation, this would route to an actual error end node
	return fmt.Errorf("request %s failed with %s: %s", request.ID, errorType, errorMessage)
}

// updateAverageRecoveryTime updates the average recovery time
func (eh *ErrorHandler) updateAverageRecoveryTime(stats *RecoveryStats, recoveryTime time.Duration) {
	if stats.AvgRecoveryTime == 0 {
		stats.AvgRecoveryTime = recoveryTime
	} else {
		// Weighted average
		stats.AvgRecoveryTime = time.Duration(
			float64(stats.AvgRecoveryTime)*0.9 + float64(recoveryTime)*0.1)
	}
}

// GetErrorStats returns error statistics for a component
func (eh *ErrorHandler) GetErrorStats(componentID string) *RecoveryStats {
	eh.mutex.RLock()
	defer eh.mutex.RUnlock()
	
	if stats, exists := eh.recoveryStats[componentID]; exists {
		// Return a copy
		return &RecoveryStats{
			TotalErrors:     stats.TotalErrors,
			RecoveredErrors: stats.RecoveredErrors,
			FailedRecovery:  stats.FailedRecovery,
			AvgRecoveryTime: stats.AvgRecoveryTime,
		}
	}
	
	return &RecoveryStats{}
}
