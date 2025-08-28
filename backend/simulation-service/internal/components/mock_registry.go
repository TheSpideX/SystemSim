package components

import (
	"sync"

	"github.com/systemsim/simulation-service/internal/engines"
)

// MockGlobalRegistry is a mock implementation of GlobalRegistry for testing
type MockGlobalRegistry struct {
	components map[string]chan *engines.Operation
	health     map[string]float64
	load       map[string]BufferStatus
	mutex      sync.RWMutex
}

// NewMockGlobalRegistry creates a new mock global registry
func NewMockGlobalRegistry() *MockGlobalRegistry {
	return &MockGlobalRegistry{
		components: make(map[string]chan *engines.Operation),
		health:     make(map[string]float64),
		load:       make(map[string]BufferStatus),
	}
}

// RegisterComponent registers a component with the mock registry
func (mgr *MockGlobalRegistry) RegisterComponent(componentID string, channel chan *engines.Operation) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	mgr.components[componentID] = channel
	mgr.health[componentID] = 1.0 // Default to healthy
	mgr.load[componentID] = BufferStatusNormal // Default to normal load
}

// Register registers a component (alias for RegisterComponent to match interface)
func (mgr *MockGlobalRegistry) Register(componentID string, inputChannel chan *engines.Operation) {
	mgr.RegisterComponent(componentID, inputChannel)
}

// GetChannel returns the input channel for a component
func (mgr *MockGlobalRegistry) GetChannel(componentID string) chan *engines.Operation {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()

	return mgr.components[componentID]
}

// GetAllComponents returns all registered components and their channels
func (mgr *MockGlobalRegistry) GetAllComponents() map[string]chan *engines.Operation {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]chan *engines.Operation)
	for id, channel := range mgr.components {
		result[id] = channel
	}
	return result
}

// GetHealth returns the health score for a component
func (mgr *MockGlobalRegistry) GetHealth(componentID string) float64 {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()
	
	if health, exists := mgr.health[componentID]; exists {
		return health
	}
	return 0.0 // Unknown component is unhealthy
}

// GetLoad returns the load status for a component
func (mgr *MockGlobalRegistry) GetLoad(componentID string) BufferStatus {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()
	
	if load, exists := mgr.load[componentID]; exists {
		return load
	}
	return BufferStatusEmergency // Unknown component is overloaded
}

// SetHealth sets the health score for a component (for testing)
func (mgr *MockGlobalRegistry) SetHealth(componentID string, health float64) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	
	mgr.health[componentID] = health
}

// SetLoad sets the load status for a component (for testing)
func (mgr *MockGlobalRegistry) SetLoad(componentID string, load BufferStatus) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	
	mgr.load[componentID] = load
}

// GetComponentIDs returns all registered component IDs
func (mgr *MockGlobalRegistry) GetComponentIDs() []string {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()
	
	ids := make([]string, 0, len(mgr.components))
	for id := range mgr.components {
		ids = append(ids, id)
	}
	return ids
}

// UnregisterComponent removes a component from the registry
func (mgr *MockGlobalRegistry) UnregisterComponent(componentID string) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	delete(mgr.components, componentID)
	delete(mgr.health, componentID)
	delete(mgr.load, componentID)
}

// Unregister removes a component (alias for UnregisterComponent to match interface)
func (mgr *MockGlobalRegistry) Unregister(componentID string) {
	mgr.UnregisterComponent(componentID)
}

// IsComponentRegistered checks if a component is registered
func (mgr *MockGlobalRegistry) IsComponentRegistered(componentID string) bool {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()

	_, exists := mgr.components[componentID]
	return exists
}

// UpdateHealth updates the health score for a component
func (mgr *MockGlobalRegistry) UpdateHealth(componentID string, health float64) {
	mgr.SetHealth(componentID, health)
}

// UpdateLoad updates the load status for a component
func (mgr *MockGlobalRegistry) UpdateLoad(componentID string, status BufferStatus) {
	mgr.SetLoad(componentID, status)
}

// Start starts the mock registry (no-op for testing)
func (mgr *MockGlobalRegistry) Start() error {
	return nil
}

// Stop stops the mock registry (no-op for testing)
func (mgr *MockGlobalRegistry) Stop() error {
	return nil
}
