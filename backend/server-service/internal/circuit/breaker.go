package circuit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	// StateClosed - circuit breaker is closed, requests pass through
	StateClosed State = iota
	// StateOpen - circuit breaker is open, requests fail fast
	StateOpen
	// StateHalfOpen - circuit breaker is half-open, testing if service recovered
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	// MaxRequests is the maximum number of requests allowed to pass through
	// when the circuit breaker is half-open
	MaxRequests uint32

	// Interval is the cyclic period of the closed state
	// for the circuit breaker to clear the internal counts
	Interval time.Duration

	// Timeout is the period of the open state,
	// after which the state of the circuit breaker becomes half-open
	Timeout time.Duration

	// ReadyToTrip is called with a copy of Counts whenever a request fails
	// in the closed state. If ReadyToTrip returns true, the circuit breaker will be placed into the open state
	ReadyToTrip func(counts Counts) bool

	// OnStateChange is called whenever the state of the circuit breaker changes
	OnStateChange func(name string, from State, to State)
}

// Counts holds the numbers of requests and their successes/failures
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// CircuitBreaker is a state machine to prevent sending requests that are likely to fail
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from State, to State)

	mutex      sync.Mutex
	state      State
	generation uint64
	counts     Counts
	expiry     time.Time
}

// NewCircuitBreaker returns a new CircuitBreaker configured with the given Config
func NewCircuitBreaker(name string, config Config) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          name,
		maxRequests:   config.MaxRequests,
		interval:      config.Interval,
		timeout:       config.Timeout,
		readyToTrip:   config.ReadyToTrip,
		onStateChange: config.OnStateChange,
	}

	if cb.maxRequests == 0 {
		cb.maxRequests = 1
	}

	if cb.interval <= 0 {
		cb.interval = time.Duration(0)
	}

	if cb.timeout <= 0 {
		cb.timeout = 60 * time.Second
	}

	if cb.readyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	}

	cb.toNewGeneration(time.Now())

	return cb
}

// Name returns the name of the CircuitBreaker
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// State returns the current state of the CircuitBreaker
func (cb *CircuitBreaker) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts returns a copy of the internal counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

// Execute runs the given request if the CircuitBreaker accepts it
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, err == nil)
	return result, err
}

// ExecuteWithContext runs the given request with context if the CircuitBreaker accepts it
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, req func(context.Context) (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req(ctx)
	cb.afterRequest(generation, err == nil)
	return result, err
}

var (
	// ErrTooManyRequests is returned when the CB state is half open and the requests count is over the cb maxRequests
	ErrTooManyRequests = errors.New("circuit breaker: too many requests")
	// ErrOpenState is returned when the CB state is open
	ErrOpenState = errors.New("circuit breaker: open state")
)

func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrOpenState
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}

	cb.counts.Requests++
	return generation, nil
}

func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0

	if state == StateHalfOpen && cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
		cb.setState(StateClosed, now)
	}
}

func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0

	if state == StateClosed {
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	} else if state == StateHalfOpen {
		cb.setState(StateOpen, now)
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// DefaultConfig returns a default circuit breaker configuration
func DefaultConfig() Config {
	return Config{
		MaxRequests: 1,
		Interval:    0,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	}
}

// ServiceConfig returns a service-specific circuit breaker configuration
func ServiceConfig(serviceName string) Config {
	config := DefaultConfig()
	config.OnStateChange = func(name string, from State, to State) {
		fmt.Printf("Circuit breaker [%s] state changed from %s to %s\n", name, from, to)
	}

	// Service-specific configurations
	switch serviceName {
	case "auth":
		config.MaxRequests = 3
		config.Timeout = 30 * time.Second
		config.ReadyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 3
		}
	case "project":
		config.MaxRequests = 5
		config.Timeout = 45 * time.Second
		config.ReadyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	case "simulation":
		config.MaxRequests = 2
		config.Timeout = 60 * time.Second
		config.ReadyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 2
		}
	}

	return config
}

// Manager manages multiple circuit breakers
type Manager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

// NewManager creates a new circuit breaker manager
func NewManager() *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetBreaker returns a circuit breaker for the given service
func (m *Manager) GetBreaker(serviceName string) *CircuitBreaker {
	m.mutex.RLock()
	breaker, exists := m.breakers[serviceName]
	m.mutex.RUnlock()

	if exists {
		return breaker
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := m.breakers[serviceName]; exists {
		return breaker
	}

	// Create new circuit breaker
	config := ServiceConfig(serviceName)
	breaker = NewCircuitBreaker(serviceName, config)
	m.breakers[serviceName] = breaker

	return breaker
}

// GetAllBreakers returns all circuit breakers
func (m *Manager) GetAllBreakers() map[string]*CircuitBreaker {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*CircuitBreaker)
	for name, breaker := range m.breakers {
		result[name] = breaker
	}

	return result
}

// GetStats returns statistics for all circuit breakers
func (m *Manager) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]interface{})
	for name, breaker := range m.breakers {
		counts := breaker.Counts()
		stats[name] = map[string]interface{}{
			"state":                 breaker.State().String(),
			"requests":              counts.Requests,
			"total_successes":       counts.TotalSuccesses,
			"total_failures":        counts.TotalFailures,
			"consecutive_successes": counts.ConsecutiveSuccesses,
			"consecutive_failures":  counts.ConsecutiveFailures,
		}
	}

	return stats
}
