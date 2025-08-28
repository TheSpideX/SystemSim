package components

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// ErrorCategory represents different categories of errors
type ErrorCategory string

const (
	ErrorCategoryTransient    ErrorCategory = "transient"    // Temporary errors that may resolve
	ErrorCategoryPermanent    ErrorCategory = "permanent"    // Errors that won't resolve without intervention
	ErrorCategoryConfiguration ErrorCategory = "configuration" // Configuration-related errors
	ErrorCategoryResource     ErrorCategory = "resource"     // Resource exhaustion errors
	ErrorCategoryNetwork      ErrorCategory = "network"      // Network-related errors
	ErrorCategoryTimeout      ErrorCategory = "timeout"      // Timeout errors
	ErrorCategoryValidation   ErrorCategory = "validation"   // Input validation errors
	ErrorCategoryInternal     ErrorCategory = "internal"     // Internal system errors
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"      // Minor issues, system continues normally
	ErrorSeverityMedium   ErrorSeverity = "medium"   // Moderate issues, some degradation
	ErrorSeverityHigh     ErrorSeverity = "high"     // Serious issues, significant impact
	ErrorSeverityCritical ErrorSeverity = "critical" // Critical issues, system failure imminent
)

// RecoveryStrategy represents different recovery strategies
type RecoveryStrategy string

const (
	RecoveryStrategyRetry     RecoveryStrategy = "retry"     // Retry the operation
	RecoveryStrategyFallback  RecoveryStrategy = "fallback"  // Use fallback mechanism
	RecoveryStrategyCircuit   RecoveryStrategy = "circuit"   // Use circuit breaker
	RecoveryStrategyDegrade   RecoveryStrategy = "degrade"   // Graceful degradation
	RecoveryStrategyFail      RecoveryStrategy = "fail"      // Fail fast
	RecoveryStrategyIgnore    RecoveryStrategy = "ignore"    // Ignore and continue
)

// ComponentError represents a comprehensive error with context and recovery information
type ComponentError struct {
	// Error identification
	ID          string        `json:"id"`
	Category    ErrorCategory `json:"category"`
	Severity    ErrorSeverity `json:"severity"`
	Code        string        `json:"code"`
	Message     string        `json:"message"`
	
	// Context information
	ComponentID string        `json:"component_id"`
	InstanceID  string        `json:"instance_id,omitempty"`
	EngineType  engines.EngineType `json:"engine_type,omitempty"`
	OperationID string        `json:"operation_id,omitempty"`
	
	// Error details
	OriginalError error         `json:"-"`
	StackTrace    string        `json:"stack_trace,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
	
	// Recovery information
	Strategy      RecoveryStrategy `json:"strategy"`
	RetryCount    int             `json:"retry_count"`
	MaxRetries    int             `json:"max_retries"`
	RetryDelay    time.Duration   `json:"retry_delay"`
	
	// Additional context
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Error implements the error interface
func (ce *ComponentError) Error() string {
	return fmt.Sprintf("[%s:%s] %s: %s (component: %s)", 
		ce.Category, ce.Severity, ce.Code, ce.Message, ce.ComponentID)
}

// IsRetryable returns true if the error can be retried
func (ce *ComponentError) IsRetryable() bool {
	return ce.Category == ErrorCategoryTransient || 
		   ce.Category == ErrorCategoryNetwork ||
		   ce.Category == ErrorCategoryTimeout
}

// ShouldCircuitBreak returns true if this error should trigger circuit breaker
func (ce *ComponentError) ShouldCircuitBreak() bool {
	return ce.Severity == ErrorSeverityHigh || 
		   ce.Severity == ErrorSeverityCritical ||
		   ce.Category == ErrorCategoryResource
}

// ErrorHandler manages comprehensive error handling for components
type ErrorHandler struct {
	// Configuration
	MaxRetries      int
	BaseRetryDelay  time.Duration
	MaxRetryDelay   time.Duration
	
	// Error tracking
	errorCounts     map[string]int
	lastErrors      map[string]*ComponentError
	mutex           sync.RWMutex
	
	// Recovery strategies
	strategies      map[ErrorCategory]RecoveryStrategy
	
	// Callbacks
	onError         func(*ComponentError)
	onRecovery      func(*ComponentError)
}

// NewErrorHandler creates a new error handler with default configuration
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		MaxRetries:     3,
		BaseRetryDelay: 100 * time.Millisecond,
		MaxRetryDelay:  5 * time.Second,
		errorCounts:    make(map[string]int),
		lastErrors:     make(map[string]*ComponentError),
		strategies: map[ErrorCategory]RecoveryStrategy{
			ErrorCategoryTransient:     RecoveryStrategyRetry,
			ErrorCategoryPermanent:     RecoveryStrategyFail,
			ErrorCategoryConfiguration: RecoveryStrategyFail,
			ErrorCategoryResource:      RecoveryStrategyCircuit,
			ErrorCategoryNetwork:       RecoveryStrategyRetry,
			ErrorCategoryTimeout:       RecoveryStrategyRetry,
			ErrorCategoryValidation:    RecoveryStrategyFail,
			ErrorCategoryInternal:     RecoveryStrategyCircuit,
		},
	}
}

// HandleError processes an error and determines the appropriate recovery strategy
func (eh *ErrorHandler) HandleError(ctx context.Context, err error, componentID string) *ComponentError {
	// Convert to ComponentError if not already
	var compErr *ComponentError
	if ce, ok := err.(*ComponentError); ok {
		compErr = ce
	} else {
		compErr = eh.categorizeError(err, componentID)
	}
	
	// Add stack trace if not present
	if compErr.StackTrace == "" {
		compErr.StackTrace = eh.getStackTrace()
	}
	
	// Update error tracking
	eh.updateErrorTracking(compErr)
	
	// Determine recovery strategy
	compErr.Strategy = eh.determineRecoveryStrategy(compErr)
	
	// Execute recovery strategy
	eh.executeRecoveryStrategy(ctx, compErr)
	
	// Call error callback
	if eh.onError != nil {
		eh.onError(compErr)
	}
	
	return compErr
}

// categorizeError categorizes a generic error into a ComponentError
func (eh *ErrorHandler) categorizeError(err error, componentID string) *ComponentError {
	errorMsg := err.Error()
	
	// Generate unique error ID
	errorID := fmt.Sprintf("%s-%d", componentID, time.Now().UnixNano())
	
	// Categorize based on error message patterns
	category := ErrorCategoryInternal
	severity := ErrorSeverityMedium
	code := "unknown_error"
	
	// Pattern matching for categorization
	switch {
	case containsAny(errorMsg, []string{"timeout", "deadline exceeded", "context deadline exceeded"}):
		category = ErrorCategoryTimeout
		severity = ErrorSeverityMedium
		code = "timeout_error"
		
	case containsAny(errorMsg, []string{"connection refused", "network", "dial", "no route"}):
		category = ErrorCategoryNetwork
		severity = ErrorSeverityMedium
		code = "network_error"
		
	case containsAny(errorMsg, []string{"resource", "memory", "disk", "cpu", "overloaded", "capacity"}):
		category = ErrorCategoryResource
		severity = ErrorSeverityHigh
		code = "resource_error"
		
	case containsAny(errorMsg, []string{"config", "configuration", "invalid", "missing"}):
		category = ErrorCategoryConfiguration
		severity = ErrorSeverityHigh
		code = "configuration_error"
		
	case containsAny(errorMsg, []string{"validation", "invalid input", "bad request"}):
		category = ErrorCategoryValidation
		severity = ErrorSeverityLow
		code = "validation_error"
		
	case containsAny(errorMsg, []string{"temporary", "retry", "transient"}):
		category = ErrorCategoryTransient
		severity = ErrorSeverityLow
		code = "transient_error"
		
	case containsAny(errorMsg, []string{"panic", "fatal", "critical", "emergency"}):
		category = ErrorCategoryInternal
		severity = ErrorSeverityCritical
		code = "critical_error"
	}
	
	return &ComponentError{
		ID:            errorID,
		Category:      category,
		Severity:      severity,
		Code:          code,
		Message:       errorMsg,
		ComponentID:   componentID,
		OriginalError: err,
		Timestamp:     time.Now(),
		RetryCount:    0,
		MaxRetries:    eh.MaxRetries,
		RetryDelay:    eh.BaseRetryDelay,
		Metadata:      make(map[string]interface{}),
	}
}

// updateErrorTracking updates error tracking statistics
func (eh *ErrorHandler) updateErrorTracking(compErr *ComponentError) {
	eh.mutex.Lock()
	defer eh.mutex.Unlock()
	
	key := fmt.Sprintf("%s:%s", compErr.ComponentID, compErr.Code)
	eh.errorCounts[key]++
	eh.lastErrors[key] = compErr
}

// determineRecoveryStrategy determines the appropriate recovery strategy
func (eh *ErrorHandler) determineRecoveryStrategy(compErr *ComponentError) RecoveryStrategy {
	// Check if we have a configured strategy for this category
	if strategy, exists := eh.strategies[compErr.Category]; exists {
		// Modify strategy based on retry count and severity
		if compErr.RetryCount >= compErr.MaxRetries {
			if strategy == RecoveryStrategyRetry {
				return RecoveryStrategyFallback
			}
		}
		
		// Escalate strategy for critical errors
		if compErr.Severity == ErrorSeverityCritical {
			return RecoveryStrategyCircuit
		}
		
		return strategy
	}
	
	// Default strategy based on severity
	switch compErr.Severity {
	case ErrorSeverityLow:
		return RecoveryStrategyIgnore
	case ErrorSeverityMedium:
		return RecoveryStrategyRetry
	case ErrorSeverityHigh:
		return RecoveryStrategyFallback
	case ErrorSeverityCritical:
		return RecoveryStrategyCircuit
	default:
		return RecoveryStrategyFail
	}
}

// executeRecoveryStrategy executes the determined recovery strategy
func (eh *ErrorHandler) executeRecoveryStrategy(ctx context.Context, compErr *ComponentError) {
	switch compErr.Strategy {
	case RecoveryStrategyRetry:
		eh.executeRetryStrategy(compErr)
	case RecoveryStrategyFallback:
		eh.executeFallbackStrategy(compErr)
	case RecoveryStrategyCircuit:
		eh.executeCircuitStrategy(compErr)
	case RecoveryStrategyDegrade:
		eh.executeDegradeStrategy(compErr)
	case RecoveryStrategyIgnore:
		eh.executeIgnoreStrategy(compErr)
	case RecoveryStrategyFail:
		eh.executeFailStrategy(compErr)
	}
}

// executeRetryStrategy implements retry logic with exponential backoff
func (eh *ErrorHandler) executeRetryStrategy(compErr *ComponentError) {
	if compErr.RetryCount < compErr.MaxRetries {
		// Calculate exponential backoff delay
		delay := time.Duration(float64(eh.BaseRetryDelay) * (1.5 * float64(compErr.RetryCount+1)))
		if delay > eh.MaxRetryDelay {
			delay = eh.MaxRetryDelay
		}
		compErr.RetryDelay = delay
		
		log.Printf("ErrorHandler: Scheduling retry %d/%d for error %s in %v", 
			compErr.RetryCount+1, compErr.MaxRetries, compErr.ID, delay)
	}
}

// executeFallbackStrategy implements fallback logic
func (eh *ErrorHandler) executeFallbackStrategy(compErr *ComponentError) {
	log.Printf("ErrorHandler: Executing fallback strategy for error %s", compErr.ID)
	// Fallback logic would be implemented by the calling component
}

// executeCircuitStrategy implements circuit breaker logic
func (eh *ErrorHandler) executeCircuitStrategy(compErr *ComponentError) {
	log.Printf("ErrorHandler: Triggering circuit breaker for component %s due to error %s", 
		compErr.ComponentID, compErr.ID)
	// Circuit breaker logic would be handled by CircuitBreakerManager
}

// executeDegradeStrategy implements graceful degradation
func (eh *ErrorHandler) executeDegradeStrategy(compErr *ComponentError) {
	log.Printf("ErrorHandler: Executing graceful degradation for error %s", compErr.ID)
	// Degradation logic would be implemented by the calling component
}

// executeIgnoreStrategy logs and ignores the error
func (eh *ErrorHandler) executeIgnoreStrategy(compErr *ComponentError) {
	log.Printf("ErrorHandler: Ignoring low-severity error %s: %s", compErr.ID, compErr.Message)
}

// executeFailStrategy logs the error and marks it as failed
func (eh *ErrorHandler) executeFailStrategy(compErr *ComponentError) {
	log.Printf("ErrorHandler: Failing fast for error %s: %s", compErr.ID, compErr.Message)
}

// getStackTrace captures the current stack trace
func (eh *ErrorHandler) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// GetErrorStats returns error statistics
func (eh *ErrorHandler) GetErrorStats() map[string]interface{} {
	eh.mutex.RLock()
	defer eh.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_errors"] = len(eh.errorCounts)
	stats["error_counts"] = make(map[string]int)
	
	for key, count := range eh.errorCounts {
		stats["error_counts"].(map[string]int)[key] = count
	}
	
	return stats
}

// SetErrorCallback sets a callback function to be called when errors occur
func (eh *ErrorHandler) SetErrorCallback(callback func(*ComponentError)) {
	eh.onError = callback
}

// SetRecoveryCallback sets a callback function to be called when recovery occurs
func (eh *ErrorHandler) SetRecoveryCallback(callback func(*ComponentError)) {
	eh.onRecovery = callback
}

// Helper function to check if a string contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// CreateComponentError creates a new ComponentError with the given parameters
func CreateComponentError(category ErrorCategory, severity ErrorSeverity, code, message, componentID string) *ComponentError {
	return &ComponentError{
		ID:          fmt.Sprintf("%s-%d", componentID, time.Now().UnixNano()),
		Category:    category,
		Severity:    severity,
		Code:        code,
		Message:     message,
		ComponentID: componentID,
		Timestamp:   time.Now(),
		RetryCount:  0,
		Metadata:    make(map[string]interface{}),
	}
}

// WrapError wraps a generic error with component context
func WrapError(err error, componentID, operationID string) *ComponentError {
	if err == nil {
		return nil
	}

	if ce, ok := err.(*ComponentError); ok {
		// Already a ComponentError, just update context if needed
		if ce.ComponentID == "" {
			ce.ComponentID = componentID
		}
		if ce.OperationID == "" {
			ce.OperationID = operationID
		}
		return ce
	}

	// Create new ComponentError from generic error
	compErr := &ComponentError{
		ID:            fmt.Sprintf("%s-%d", componentID, time.Now().UnixNano()),
		Category:      ErrorCategoryInternal,
		Severity:      ErrorSeverityMedium,
		Code:          "wrapped_error",
		Message:       err.Error(),
		ComponentID:   componentID,
		OperationID:   operationID,
		OriginalError: err,
		Timestamp:     time.Now(),
		RetryCount:    0,
		Metadata:      make(map[string]interface{}),
	}

	return compErr
}

// Global error handler instance
var GlobalErrorHandler *ErrorHandler

// InitializeErrorHandler initializes the global error handler
func InitializeErrorHandler() {
	GlobalErrorHandler = NewErrorHandler()
}
