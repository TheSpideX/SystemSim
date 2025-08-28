package components

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitBreakerClosed:
		return "CLOSED"
	case CircuitBreakerOpen:
		return "OPEN"
	case CircuitBreakerHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig defines configuration for circuit breaker behavior
type CircuitBreakerConfig struct {
	// Failure threshold to open the circuit
	FailureThreshold int `json:"failure_threshold"`
	
	// Success threshold to close the circuit from half-open
	SuccessThreshold int `json:"success_threshold"`
	
	// Timeout before transitioning from open to half-open
	Timeout time.Duration `json:"timeout"`
	
	// Maximum number of retry attempts
	MaxRetries int `json:"max_retries"`
	
	// Delay between retry attempts
	RetryDelay time.Duration `json:"retry_delay"`
	
	// Fallback component ID to use when circuit is open
	FallbackComponent string `json:"fallback_component"`
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold:  5,
		SuccessThreshold:  3,
		Timeout:          30 * time.Second,
		MaxRetries:       3,
		RetryDelay:       100 * time.Millisecond,
		FallbackComponent: "",
	}
}

// CircuitBreaker implements the circuit breaker pattern for component routing
type CircuitBreaker struct {
	// Configuration
	config *CircuitBreakerConfig
	
	// Current state
	state CircuitBreakerState
	
	// Failure and success counters
	failureCount   int
	successCount   int
	
	// Last failure time (for timeout calculation)
	lastFailureTime time.Time
	
	// Target component ID this circuit breaker protects
	targetComponent string
	
	// Mutex for thread safety
	mutex sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker for a target component
func NewCircuitBreaker(targetComponent string, config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}
	
	return &CircuitBreaker{
		config:          config,
		state:           CircuitBreakerClosed,
		targetComponent: targetComponent,
		failureCount:    0,
		successCount:    0,
	}
}

// CanExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		// Check if timeout has passed to transition to half-open
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			return true // Will transition to half-open on next call
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// Execute attempts to execute an operation through the circuit breaker
func (cb *CircuitBreaker) Execute(operation func() error) error {
	// Check if we can execute
	if !cb.CanExecute() {
		return fmt.Errorf("circuit breaker is OPEN for component %s", cb.targetComponent)
	}
	
	// Transition to half-open if we're open and timeout has passed
	cb.mutex.Lock()
	if cb.state == CircuitBreakerOpen && time.Since(cb.lastFailureTime) >= cb.config.Timeout {
		cb.state = CircuitBreakerHalfOpen
		cb.successCount = 0
		log.Printf("CircuitBreaker %s: Transitioning to HALF_OPEN", cb.targetComponent)
	}
	cb.mutex.Unlock()
	
	// Execute the operation
	err := operation()
	
	// Record the result
	if err != nil {
		cb.recordFailure()
		return err
	} else {
		cb.recordSuccess()
		return nil
	}
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	
	switch cb.state {
	case CircuitBreakerClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = CircuitBreakerOpen
			log.Printf("CircuitBreaker %s: Opening circuit after %d failures", 
				cb.targetComponent, cb.failureCount)
		}
	case CircuitBreakerHalfOpen:
		// Any failure in half-open state opens the circuit
		cb.state = CircuitBreakerOpen
		cb.failureCount = 1 // Reset counter
		log.Printf("CircuitBreaker %s: Reopening circuit due to failure in HALF_OPEN state", 
			cb.targetComponent)
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.successCount++
	
	switch cb.state {
	case CircuitBreakerHalfOpen:
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = CircuitBreakerClosed
			cb.failureCount = 0
			cb.successCount = 0
			log.Printf("CircuitBreaker %s: Closing circuit after %d successes", 
				cb.targetComponent, cb.successCount)
		}
	case CircuitBreakerClosed:
		// Reset failure count on success
		cb.failureCount = 0
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns current statistics of the circuit breaker
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	return map[string]interface{}{
		"state":           cb.state.String(),
		"failure_count":   cb.failureCount,
		"success_count":   cb.successCount,
		"last_failure":    cb.lastFailureTime,
		"target_component": cb.targetComponent,
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.state = CircuitBreakerClosed
	cb.failureCount = 0
	cb.successCount = 0
	log.Printf("CircuitBreaker %s: Reset to CLOSED state", cb.targetComponent)
}

// CircuitBreakerManager manages multiple circuit breakers for different components
type CircuitBreakerManager struct {
	circuitBreakers map[string]*CircuitBreaker
	defaultConfig   *CircuitBreakerConfig
	mutex           sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(defaultConfig *CircuitBreakerConfig) *CircuitBreakerManager {
	if defaultConfig == nil {
		defaultConfig = DefaultCircuitBreakerConfig()
	}
	
	return &CircuitBreakerManager{
		circuitBreakers: make(map[string]*CircuitBreaker),
		defaultConfig:   defaultConfig,
	}
}

// GetCircuitBreaker gets or creates a circuit breaker for a component
func (cbm *CircuitBreakerManager) GetCircuitBreaker(componentID string) *CircuitBreaker {
	cbm.mutex.RLock()
	if cb, exists := cbm.circuitBreakers[componentID]; exists {
		cbm.mutex.RUnlock()
		return cb
	}
	cbm.mutex.RUnlock()
	
	// Create new circuit breaker
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()
	
	// Double-check after acquiring write lock
	if cb, exists := cbm.circuitBreakers[componentID]; exists {
		return cb
	}
	
	cb := NewCircuitBreaker(componentID, cbm.defaultConfig)
	cbm.circuitBreakers[componentID] = cb
	log.Printf("CircuitBreakerManager: Created circuit breaker for component %s", componentID)
	return cb
}

// ExecuteWithCircuitBreaker executes an operation with circuit breaker protection
func (cbm *CircuitBreakerManager) ExecuteWithCircuitBreaker(
	componentID string, 
	operation func() error,
) error {
	cb := cbm.GetCircuitBreaker(componentID)
	return cb.Execute(operation)
}

// GetAllStats returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetAllStats() map[string]interface{} {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	for componentID, cb := range cbm.circuitBreakers {
		stats[componentID] = cb.GetStats()
	}
	return stats
}

// ResetAll resets all circuit breakers
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	for _, cb := range cbm.circuitBreakers {
		cb.Reset()
	}
	log.Printf("CircuitBreakerManager: Reset all circuit breakers")
}
