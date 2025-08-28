package mesh

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MeshClient provides a high-level interface for making gRPC calls through connection pools
type MeshClient struct {
	poolManager *PoolManager
}

// NewMeshClient creates a new mesh client
func NewMeshClient(poolManager *PoolManager) *MeshClient {
	return &MeshClient{
		poolManager: poolManager,
	}
}

// CallWithRetry makes a gRPC call with automatic retry and load balancing
func (mc *MeshClient) CallWithRetry(ctx context.Context, serviceName string, method func(*grpc.ClientConn) error) error {
	const maxRetries = 3
	const baseDelay = 100 * time.Millisecond
	
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Get connection from pool
		conn, err := mc.poolManager.GetConnection(serviceName)
		if err != nil {
			lastErr = fmt.Errorf("failed to get connection to %s: %v", serviceName, err)
			log.Printf("Attempt %d/%d failed to get connection to %s: %v", 
				attempt+1, maxRetries, serviceName, err)
			
			// Wait before retry with exponential backoff
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<attempt) // 100ms, 200ms, 400ms
				time.Sleep(delay)
			}
			continue
		}
		
		// Make the gRPC call
		err = method(conn)
		if err == nil {
			return nil // Success
		}
		
		// Check if error is retryable
		if !isRetryableError(err) {
			return err // Don't retry non-retryable errors
		}
		
		lastErr = err
		log.Printf("Attempt %d/%d failed for %s: %v", attempt+1, maxRetries, serviceName, err)
		
		// Wait before retry
		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<attempt)
			time.Sleep(delay)
		}
	}
	
	return fmt.Errorf("all %d attempts failed for %s, last error: %v", maxRetries, serviceName, lastErr)
}

// Call makes a single gRPC call without retry
func (mc *MeshClient) Call(ctx context.Context, serviceName string, method func(*grpc.ClientConn) error) error {
	conn, err := mc.poolManager.GetConnection(serviceName)
	if err != nil {
		return fmt.Errorf("failed to get connection to %s: %v", serviceName, err)
	}
	
	return method(conn)
}

// CallWithTimeout makes a gRPC call with a specific timeout
func (mc *MeshClient) CallWithTimeout(serviceName string, timeout time.Duration, method func(*grpc.ClientConn) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	return mc.CallWithRetry(ctx, serviceName, method)
}

// isRetryableError determines if a gRPC error should be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check gRPC status codes
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable,     // Service unavailable
			 codes.DeadlineExceeded, // Timeout
			 codes.ResourceExhausted, // Rate limited
			 codes.Aborted:          // Transaction aborted
			return true
		case codes.InvalidArgument, // Bad request
			 codes.NotFound,        // Not found
			 codes.PermissionDenied, // Permission denied
			 codes.Unauthenticated:  // Authentication failed
			return false
		default:
			return false
		}
	}
	
	// Retry on connection errors
	return true
}

// CircuitBreaker implements circuit breaker pattern for service calls
type CircuitBreaker struct {
	serviceName     string
	failureThreshold int
	resetTimeout    time.Duration
	
	// State tracking
	state           CircuitState
	failureCount    int
	lastFailureTime time.Time
	successCount    int
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota // Normal operation
	CircuitOpen                       // Failing, reject calls
	CircuitHalfOpen                   // Testing if service recovered
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(serviceName string, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		serviceName:      serviceName,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:           CircuitClosed,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	switch cb.state {
	case CircuitOpen:
		// Check if we should try to reset
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
			log.Printf("Circuit breaker for %s moved to HALF_OPEN", cb.serviceName)
		} else {
			return fmt.Errorf("circuit breaker OPEN for %s", cb.serviceName)
		}
	case CircuitHalfOpen:
		// In half-open state, allow limited calls
	case CircuitClosed:
		// Normal operation
	}
	
	// Execute the function
	err := fn()
	
	if err != nil {
		cb.onFailure()
		return err
	}
	
	cb.onSuccess()
	return nil
}

// onSuccess handles successful calls
func (cb *CircuitBreaker) onSuccess() {
	cb.failureCount = 0
	
	if cb.state == CircuitHalfOpen {
		cb.successCount++
		// After a few successful calls in half-open, close the circuit
		if cb.successCount >= 3 {
			cb.state = CircuitClosed
			log.Printf("Circuit breaker for %s moved to CLOSED", cb.serviceName)
		}
	}
}

// onFailure handles failed calls
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	
	if cb.failureCount >= cb.failureThreshold {
		cb.state = CircuitOpen
		log.Printf("Circuit breaker for %s moved to OPEN after %d failures", 
			cb.serviceName, cb.failureCount)
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.state
}

// MeshClientWithCircuitBreaker combines mesh client with circuit breaker
type MeshClientWithCircuitBreaker struct {
	meshClient      *MeshClient
	circuitBreakers map[string]*CircuitBreaker
}

// NewMeshClientWithCircuitBreaker creates a mesh client with circuit breaker protection
func NewMeshClientWithCircuitBreaker(poolManager *PoolManager) *MeshClientWithCircuitBreaker {
	return &MeshClientWithCircuitBreaker{
		meshClient:      NewMeshClient(poolManager),
		circuitBreakers: make(map[string]*CircuitBreaker),
	}
}

// CallWithCircuitBreaker makes a call protected by circuit breaker
func (mcb *MeshClientWithCircuitBreaker) CallWithCircuitBreaker(ctx context.Context, serviceName string, method func(*grpc.ClientConn) error) error {
	// Get or create circuit breaker for this service
	cb, exists := mcb.circuitBreakers[serviceName]
	if !exists {
		cb = NewCircuitBreaker(serviceName, 5, 30*time.Second) // 5 failures, 30s reset
		mcb.circuitBreakers[serviceName] = cb
	}
	
	// Execute with circuit breaker protection
	return cb.Call(func() error {
		return mcb.meshClient.CallWithRetry(ctx, serviceName, method)
	})
}

// GetCircuitBreakerStates returns the state of all circuit breakers
func (mcb *MeshClientWithCircuitBreaker) GetCircuitBreakerStates() map[string]CircuitState {
	states := make(map[string]CircuitState)
	for serviceName, cb := range mcb.circuitBreakers {
		states[serviceName] = cb.GetState()
	}
	return states
}

// HealthCheck returns health information about the mesh client
func (mc *MeshClient) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"pool_manager": mc.poolManager.HealthCheck(),
		"services":     mc.poolManager.GetServiceNames(),
	}
}

// HealthCheck returns health information about the mesh client with circuit breaker
func (mcb *MeshClientWithCircuitBreaker) HealthCheck() map[string]interface{} {
	health := mcb.meshClient.HealthCheck()

	// Add circuit breaker states
	circuitStates := make(map[string]string)
	for serviceName, state := range mcb.GetCircuitBreakerStates() {
		switch state {
		case CircuitClosed:
			circuitStates[serviceName] = "CLOSED"
		case CircuitOpen:
			circuitStates[serviceName] = "OPEN"
		case CircuitHalfOpen:
			circuitStates[serviceName] = "HALF_OPEN"
		}
	}

	health["circuit_breakers"] = circuitStates
	return health
}
