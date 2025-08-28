package components

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// GlobalRegistry implements the GlobalRegistryInterface
// Enhanced to store system graphs, request contexts, and eliminate local cache complexity
type GlobalRegistry struct {
	// Component registration (single source of truth)
	components map[string]chan *engines.Operation

	// System graphs storage (key enhancement)
	systemGraphs map[string]*DecisionGraph

	// Request context management (key enhancement)
	requestContexts map[string]*RequestContext

	// Health and load monitoring
	health map[string]float64
	load   map[string]BufferStatus

	// Lifecycle management
	running bool
	mutex   sync.RWMutex

	// Health monitoring
	healthTicker *time.Ticker
	stopChan     chan struct{}

	// Context cleanup
	contextCleanupTicker *time.Ticker
	contextTTL           time.Duration
}

// NewGlobalRegistry creates a new enhanced global registry
func NewGlobalRegistry() *GlobalRegistry {
	return &GlobalRegistry{
		components:           make(map[string]chan *engines.Operation),
		systemGraphs:         make(map[string]*DecisionGraph),
		requestContexts:      make(map[string]*RequestContext),
		health:               make(map[string]float64),
		load:                 make(map[string]BufferStatus),
		running:              false,
		stopChan:             make(chan struct{}),
		contextCleanupTicker: time.NewTicker(30 * time.Second), // Cleanup every 30 seconds
		contextTTL:           5 * time.Minute,                   // Context TTL of 5 minutes
	}
}

// NewGlobalRegistryWithConfig creates a new global registry with custom configuration
func NewGlobalRegistryWithConfig(contextTTL time.Duration, cleanupInterval time.Duration) *GlobalRegistry {
	return &GlobalRegistry{
		components:           make(map[string]chan *engines.Operation),
		systemGraphs:         make(map[string]*DecisionGraph),
		requestContexts:      make(map[string]*RequestContext),
		health:               make(map[string]float64),
		load:                 make(map[string]BufferStatus),
		running:              false,
		stopChan:             make(chan struct{}),
		contextCleanupTicker: time.NewTicker(cleanupInterval),
		contextTTL:           contextTTL,
	}
}

// Register registers a component with the global registry
func (gr *GlobalRegistry) Register(componentID string, inputChannel chan *engines.Operation) {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if componentID == "" {
		log.Printf("GlobalRegistry: Cannot register component with empty ID")
		return
	}

	if inputChannel == nil {
		log.Printf("GlobalRegistry: Cannot register component %s with nil input channel", componentID)
		return
	}

	gr.components[componentID] = inputChannel
	gr.health[componentID] = 1.0 // Start with full health
	gr.load[componentID] = BufferStatusNormal // Start with normal load

	log.Printf("GlobalRegistry: Registered component %s", componentID)
}

// Unregister removes a component from the global registry
func (gr *GlobalRegistry) Unregister(componentID string) {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if _, exists := gr.components[componentID]; !exists {
		log.Printf("GlobalRegistry: Component %s not found for unregistration", componentID)
		return
	}

	delete(gr.components, componentID)
	delete(gr.health, componentID)
	delete(gr.load, componentID)

	log.Printf("GlobalRegistry: Unregistered component %s", componentID)
}

// GetChannel returns the input channel for a component
func (gr *GlobalRegistry) GetChannel(componentID string) chan *engines.Operation {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	channel, exists := gr.components[componentID]
	if !exists {
		log.Printf("GlobalRegistry: Component %s not found", componentID)
		return nil
	}

	return channel
}

// GetAllComponents returns all registered components and their channels
func (gr *GlobalRegistry) GetAllComponents() map[string]chan *engines.Operation {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	// Return a copy to prevent external modification
	components := make(map[string]chan *engines.Operation)
	for id, channel := range gr.components {
		components[id] = channel
	}

	return components
}

// System Graph Management Methods

// GetSystemGraph returns the system graph for a given flow ID
func (gr *GlobalRegistry) GetSystemGraph(flowID string) *DecisionGraph {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	graph, exists := gr.systemGraphs[flowID]
	if !exists {
		log.Printf("GlobalRegistry: System graph %s not found", flowID)
		return nil
	}

	return graph
}

// UpdateSystemGraph updates or creates a system graph for a given flow ID
func (gr *GlobalRegistry) UpdateSystemGraph(flowID string, graph *DecisionGraph) error {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if flowID == "" {
		return fmt.Errorf("flow ID cannot be empty")
	}

	if graph == nil {
		return fmt.Errorf("graph cannot be nil")
	}

	// Set graph level to system level
	graph.Level = SystemLevel

	gr.systemGraphs[flowID] = graph
	log.Printf("GlobalRegistry: Updated system graph for flow %s", flowID)

	return nil
}

// GetRequestContext returns the request context for a given request ID
func (gr *GlobalRegistry) GetRequestContext(requestID string) *RequestContext {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	context, exists := gr.requestContexts[requestID]
	if !exists {
		log.Printf("GlobalRegistry: Request context %s not found", requestID)
		return nil
	}

	return context
}

// UpdateRequestContext updates or creates a request context
func (gr *GlobalRegistry) UpdateRequestContext(requestID string, context *RequestContext) error {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if requestID == "" {
		return fmt.Errorf("request ID cannot be empty")
	}

	if context == nil {
		return fmt.Errorf("context cannot be nil")
	}

	// Update last update timestamp
	context.LastUpdate = time.Now().Format(time.RFC3339)

	gr.requestContexts[requestID] = context
	log.Printf("GlobalRegistry: Updated request context for request %s", requestID)

	return nil
}

// System Graph Management Methods

// GetSystemGraph returns the system graph for a given flow ID
func (gr *GlobalRegistry) GetSystemGraph(flowID string) *DecisionGraph {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	graph, exists := gr.systemGraphs[flowID]
	if !exists {
		log.Printf("GlobalRegistry: System graph %s not found", flowID)
		return nil
	}

	return graph
}

// UpdateSystemGraph updates or creates a system graph for a given flow ID
func (gr *GlobalRegistry) UpdateSystemGraph(flowID string, graph *DecisionGraph) error {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if flowID == "" {
		return fmt.Errorf("flow ID cannot be empty")
	}

	if graph == nil {
		return fmt.Errorf("graph cannot be nil")
	}

	// Set graph level to system level
	graph.Level = SystemLevel

	gr.systemGraphs[flowID] = graph
	log.Printf("GlobalRegistry: Updated system graph for flow %s", flowID)

	return nil
}

// ListSystemGraphs returns all available system graph flow IDs
func (gr *GlobalRegistry) ListSystemGraphs() []string {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	flowIDs := make([]string, 0, len(gr.systemGraphs))
	for flowID := range gr.systemGraphs {
		flowIDs = append(flowIDs, flowID)
	}

	return flowIDs
}

// DeleteSystemGraph removes a system graph
func (gr *GlobalRegistry) DeleteSystemGraph(flowID string) error {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if _, exists := gr.systemGraphs[flowID]; !exists {
		return fmt.Errorf("system graph %s not found", flowID)
	}

	delete(gr.systemGraphs, flowID)
	log.Printf("GlobalRegistry: Deleted system graph for flow %s", flowID)

	return nil
}

// Request Context Management Methods

// GetRequestContext returns the request context for a given request ID
func (gr *GlobalRegistry) GetRequestContext(requestID string) *RequestContext {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	context, exists := gr.requestContexts[requestID]
	if !exists {
		log.Printf("GlobalRegistry: Request context %s not found", requestID)
		return nil
	}

	return context
}

// UpdateRequestContext updates or creates a request context
func (gr *GlobalRegistry) UpdateRequestContext(requestID string, context *RequestContext) error {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if requestID == "" {
		return fmt.Errorf("request ID cannot be empty")
	}

	if context == nil {
		return fmt.Errorf("context cannot be nil")
	}

	// Update last update timestamp
	context.LastUpdate = time.Now().Format(time.RFC3339)

	gr.requestContexts[requestID] = context
	log.Printf("GlobalRegistry: Updated request context for request %s", requestID)

	return nil
}

// CreateRequestContext creates a new request context
func (gr *GlobalRegistry) CreateRequestContext(requestID, systemFlowID, startNode string) *RequestContext {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	now := time.Now().Format(time.RFC3339)
	context := &RequestContext{
		RequestID:         requestID,
		SystemFlowID:      systemFlowID,
		CurrentSystemNode: startNode,
		StartTime:         now,
		LastUpdate:        now,
	}

	gr.requestContexts[requestID] = context
	log.Printf("GlobalRegistry: Created request context for request %s", requestID)

	return context
}

// DeleteRequestContext removes a request context
func (gr *GlobalRegistry) DeleteRequestContext(requestID string) error {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if _, exists := gr.requestContexts[requestID]; !exists {
		return fmt.Errorf("request context %s not found", requestID)
	}

	delete(gr.requestContexts, requestID)
	log.Printf("GlobalRegistry: Deleted request context for request %s", requestID)

	return nil
}

// ListActiveRequests returns all active request IDs
func (gr *GlobalRegistry) ListActiveRequests() []string {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	requestIDs := make([]string, 0, len(gr.requestContexts))
	for requestID := range gr.requestContexts {
		requestIDs = append(requestIDs, requestID)
	}

	return requestIDs
}

// GetHealth returns the health score for a component
func (gr *GlobalRegistry) GetHealth(componentID string) float64 {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	health, exists := gr.health[componentID]
	if !exists {
		log.Printf("GlobalRegistry: Health not found for component %s", componentID)
		return 0.0
	}

	return health
}

// UpdateHealth updates the health score for a component
func (gr *GlobalRegistry) UpdateHealth(componentID string, health float64) {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if _, exists := gr.components[componentID]; !exists {
		log.Printf("GlobalRegistry: Cannot update health for unregistered component %s", componentID)
		return
	}

	// Clamp health between 0.0 and 1.0
	if health < 0.0 {
		health = 0.0
	} else if health > 1.0 {
		health = 1.0
	}

	gr.health[componentID] = health
	log.Printf("GlobalRegistry: Updated health for component %s to %.2f", componentID, health)
}

// GetLoad returns the load status for a component
func (gr *GlobalRegistry) GetLoad(componentID string) BufferStatus {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	load, exists := gr.load[componentID]
	if !exists {
		log.Printf("GlobalRegistry: Load not found for component %s", componentID)
		return BufferStatusEmergency // Assume worst case
	}

	return load
}

// UpdateLoad updates the load status for a component
func (gr *GlobalRegistry) UpdateLoad(componentID string, status BufferStatus) {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if _, exists := gr.components[componentID]; !exists {
		log.Printf("GlobalRegistry: Cannot update load for unregistered component %s", componentID)
		return
	}

	gr.load[componentID] = status
	log.Printf("GlobalRegistry: Updated load for component %s to %s", componentID, status.String())
}

// Start starts the global registry and health monitoring
func (gr *GlobalRegistry) Start() error {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if gr.running {
		return fmt.Errorf("global registry is already running")
	}

	gr.running = true
	gr.healthTicker = time.NewTicker(time.Second * 5) // Health check every 5 seconds

	// Start health monitoring goroutine
	go gr.healthMonitoringLoop()

	log.Printf("GlobalRegistry: Started with health monitoring")
	return nil
}

// Stop stops the global registry
func (gr *GlobalRegistry) Stop() error {
	gr.mutex.Lock()
	defer gr.mutex.Unlock()

	if !gr.running {
		return fmt.Errorf("global registry is not running")
	}

	gr.running = false

	// Stop health monitoring
	if gr.healthTicker != nil {
		gr.healthTicker.Stop()
	}

	close(gr.stopChan)

	log.Printf("GlobalRegistry: Stopped")
	return nil
}

// healthMonitoringLoop runs the health monitoring in a separate goroutine
func (gr *GlobalRegistry) healthMonitoringLoop() {
	for {
		select {
		case <-gr.healthTicker.C:
			gr.performHealthCheck()
		case <-gr.stopChan:
			log.Printf("GlobalRegistry: Health monitoring stopped")
			return
		}
	}
}

// performHealthCheck performs periodic health checks on all components
func (gr *GlobalRegistry) performHealthCheck() {
	gr.mutex.RLock()
	componentIDs := make([]string, 0, len(gr.components))
	for id := range gr.components {
		componentIDs = append(componentIDs, id)
	}
	gr.mutex.RUnlock()

	for _, componentID := range componentIDs {
		// Simulate health check by checking channel capacity
		channel := gr.GetChannel(componentID)
		if channel != nil {
			// Calculate health based on channel utilization
			capacity := cap(channel)
			length := len(channel)
			utilization := float64(length) / float64(capacity)
			
			// Health decreases as utilization increases
			health := 1.0 - utilization
			if health < 0.0 {
				health = 0.0
			}

			// Update health
			gr.UpdateHealth(componentID, health)

			// Update load status based on utilization
			var loadStatus BufferStatus
			switch {
			case utilization < 0.2:
				loadStatus = BufferStatusNormal
			case utilization < 0.4:
				loadStatus = BufferStatusWarning
			case utilization < 0.6:
				loadStatus = BufferStatusHigh
			case utilization < 0.8:
				loadStatus = BufferStatusOverflow
			case utilization < 0.9:
				loadStatus = BufferStatusCritical
			default:
				loadStatus = BufferStatusEmergency
			}

			gr.UpdateLoad(componentID, loadStatus)
		}
	}
}

// GetRegisteredComponents returns a list of all registered component IDs
func (gr *GlobalRegistry) GetRegisteredComponents() []string {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	components := make([]string, 0, len(gr.components))
	for id := range gr.components {
		components = append(components, id)
	}

	return components
}

// GetComponentStats returns health and load statistics for all components
func (gr *GlobalRegistry) GetComponentStats() map[string]ComponentStats {
	gr.mutex.RLock()
	defer gr.mutex.RUnlock()

	stats := make(map[string]ComponentStats)
	for id := range gr.components {
		stats[id] = ComponentStats{
			ComponentID: id,
			Health:      gr.health[id],
			Load:        gr.load[id],
			Registered:  true,
		}
	}

	return stats
}

// ComponentStats represents statistics for a component
type ComponentStats struct {
	ComponentID string       `json:"component_id"`
	Health      float64      `json:"health"`
	Load        BufferStatus `json:"load"`
	Registered  bool         `json:"registered"`
}
