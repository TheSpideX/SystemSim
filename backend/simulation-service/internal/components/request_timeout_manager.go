package components

import (
	"context"
	"log"
	"sync"
	"time"
)

// RequestTimeoutManager manages request timeouts to prevent stuck requests
type RequestTimeoutManager struct {
	// Configuration
	TimeoutDuration time.Duration `json:"timeout_duration"`

	// Active request tracking
	activeRequests map[string]*RequestTimeout `json:"-"`
	mutex          sync.RWMutex               `json:"-"`

	// Lifecycle management
	ctx    context.Context    `json:"-"`
	cancel context.CancelFunc `json:"-"`
	ticker *time.Ticker       `json:"-"`

	// Metrics
	TimeoutCount int64 `json:"timeout_count"`
}

// RequestTimeout tracks timeout information for a request
type RequestTimeout struct {
	RequestID   string    `json:"request_id"`
	StartTime   time.Time `json:"start_time"`
	LastUpdate  time.Time `json:"last_update"`
	ComponentID string    `json:"component_id"`
	EngineType  string    `json:"engine_type"`
}

// NewRequestTimeoutManager creates a new request timeout manager
func NewRequestTimeoutManager(timeoutDuration time.Duration) *RequestTimeoutManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &RequestTimeoutManager{
		TimeoutDuration: timeoutDuration,
		activeRequests:  make(map[string]*RequestTimeout),
		ctx:             ctx,
		cancel:          cancel,
		ticker:          time.NewTicker(5 * time.Second), // Check every 5 seconds
	}
}

// Start starts the timeout manager
func (rtm *RequestTimeoutManager) Start() {
	log.Printf("RequestTimeoutManager: Starting timeout manager with %v timeout", rtm.TimeoutDuration)
	go rtm.run()
}

// Stop stops the timeout manager
func (rtm *RequestTimeoutManager) Stop() {
	log.Printf("RequestTimeoutManager: Stopping timeout manager")
	rtm.cancel()
	rtm.ticker.Stop()
}

// run is the main timeout checking loop
func (rtm *RequestTimeoutManager) run() {
	for {
		select {
		case <-rtm.ticker.C:
			rtm.checkTimeouts()

		case <-rtm.ctx.Done():
			log.Printf("RequestTimeoutManager: Timeout manager stopping")
			return
		}
	}
}

// RegisterRequest registers a request for timeout monitoring
func (rtm *RequestTimeoutManager) RegisterRequest(requestID, componentID, engineType string) {
	rtm.mutex.Lock()
	defer rtm.mutex.Unlock()

	now := time.Now()
	rtm.activeRequests[requestID] = &RequestTimeout{
		RequestID:   requestID,
		StartTime:   now,
		LastUpdate:  now,
		ComponentID: componentID,
		EngineType:  engineType,
	}

	log.Printf("RequestTimeoutManager: Registered request %s for timeout monitoring", requestID)
}

// UpdateRequest updates the last activity time for a request
func (rtm *RequestTimeoutManager) UpdateRequest(requestID string) {
	rtm.mutex.Lock()
	defer rtm.mutex.Unlock()

	if timeout, exists := rtm.activeRequests[requestID]; exists {
		timeout.LastUpdate = time.Now()
	}
}

// UnregisterRequest removes a request from timeout monitoring
func (rtm *RequestTimeoutManager) UnregisterRequest(requestID string) {
	rtm.mutex.Lock()
	defer rtm.mutex.Unlock()

	if _, exists := rtm.activeRequests[requestID]; exists {
		delete(rtm.activeRequests, requestID)
		log.Printf("RequestTimeoutManager: Unregistered request %s from timeout monitoring", requestID)
	}
}

// checkTimeouts checks for timed out requests and handles them
func (rtm *RequestTimeoutManager) checkTimeouts() {
	rtm.mutex.Lock()
	timedOutRequests := make([]*RequestTimeout, 0)
	
	for requestID, timeout := range rtm.activeRequests {
		if time.Since(timeout.LastUpdate) > rtm.TimeoutDuration {
			timedOutRequests = append(timedOutRequests, timeout)
			delete(rtm.activeRequests, requestID)
		}
	}
	rtm.mutex.Unlock()

	// Handle timed out requests
	for _, timeout := range timedOutRequests {
		rtm.handleTimeout(timeout)
	}
}

// handleTimeout handles a timed out request
func (rtm *RequestTimeoutManager) handleTimeout(timeout *RequestTimeout) {
	rtm.TimeoutCount++
	
	log.Printf("RequestTimeoutManager: Request %s timed out after %v (component: %s, engine: %s)", 
		timeout.RequestID, time.Since(timeout.StartTime), timeout.ComponentID, timeout.EngineType)

	// Route to error end node
	rtm.routeToErrorEndNode(timeout.RequestID, "request_timeout")
}

// routeToErrorEndNode routes a timed out request to error end node
func (rtm *RequestTimeoutManager) routeToErrorEndNode(requestID, reason string) {
	// This would typically interact with the global registry to route the request
	// For now, we'll just log the action
	log.Printf("RequestTimeoutManager: Routing request %s to error end node (reason: %s)", requestID, reason)
	
	// In a full implementation, this would:
	// 1. Get the request from global registry
	// 2. Mark it as failed with timeout reason
	// 3. Route it to an error end node for cleanup
}

// GetActiveRequestCount returns the number of active requests being monitored
func (rtm *RequestTimeoutManager) GetActiveRequestCount() int {
	rtm.mutex.RLock()
	defer rtm.mutex.RUnlock()
	return len(rtm.activeRequests)
}

// GetTimeoutCount returns the total number of timeouts that have occurred
func (rtm *RequestTimeoutManager) GetTimeoutCount() int64 {
	return rtm.TimeoutCount
}

// GetActiveRequests returns a copy of all active request timeouts
func (rtm *RequestTimeoutManager) GetActiveRequests() map[string]*RequestTimeout {
	rtm.mutex.RLock()
	defer rtm.mutex.RUnlock()

	// Return a copy to prevent external modification
	activeRequests := make(map[string]*RequestTimeout)
	for id, timeout := range rtm.activeRequests {
		activeRequests[id] = &RequestTimeout{
			RequestID:   timeout.RequestID,
			StartTime:   timeout.StartTime,
			LastUpdate:  timeout.LastUpdate,
			ComponentID: timeout.ComponentID,
			EngineType:  timeout.EngineType,
		}
	}

	return activeRequests
}

// SetTimeoutDuration updates the timeout duration
func (rtm *RequestTimeoutManager) SetTimeoutDuration(duration time.Duration) {
	rtm.TimeoutDuration = duration
	log.Printf("RequestTimeoutManager: Updated timeout duration to %v", duration)
}

// GetOldestRequest returns information about the oldest active request
func (rtm *RequestTimeoutManager) GetOldestRequest() *RequestTimeout {
	rtm.mutex.RLock()
	defer rtm.mutex.RUnlock()

	var oldest *RequestTimeout
	var oldestTime time.Time

	for _, timeout := range rtm.activeRequests {
		if oldest == nil || timeout.StartTime.Before(oldestTime) {
			oldest = timeout
			oldestTime = timeout.StartTime
		}
	}

	if oldest != nil {
		// Return a copy
		return &RequestTimeout{
			RequestID:   oldest.RequestID,
			StartTime:   oldest.StartTime,
			LastUpdate:  oldest.LastUpdate,
			ComponentID: oldest.ComponentID,
			EngineType:  oldest.EngineType,
		}
	}

	return nil
}

// GetRequestsByComponent returns all active requests for a specific component
func (rtm *RequestTimeoutManager) GetRequestsByComponent(componentID string) []*RequestTimeout {
	rtm.mutex.RLock()
	defer rtm.mutex.RUnlock()

	var requests []*RequestTimeout
	for _, timeout := range rtm.activeRequests {
		if timeout.ComponentID == componentID {
			// Add a copy
			requests = append(requests, &RequestTimeout{
				RequestID:   timeout.RequestID,
				StartTime:   timeout.StartTime,
				LastUpdate:  timeout.LastUpdate,
				ComponentID: timeout.ComponentID,
				EngineType:  timeout.EngineType,
			})
		}
	}

	return requests
}
