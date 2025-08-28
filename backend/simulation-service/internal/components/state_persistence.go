package components

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// StatePersistenceManager manages state saving and loading for components
type StatePersistenceManager struct {
	StateDirectory string
	mutex          sync.RWMutex
}

// NewStatePersistenceManager creates a new state persistence manager
func NewStatePersistenceManager(stateDirectory string) *StatePersistenceManager {
	return &StatePersistenceManager{
		StateDirectory: stateDirectory,
	}
}

// ComponentInstanceState represents the complete state of a component instance
type ComponentInstanceState struct {
	// Identity
	ID            string        `json:"id"`
	ComponentID   string        `json:"component_id"`
	ComponentType ComponentType `json:"component_type"`

	// Runtime state
	Running       bool          `json:"running"`
	Ready         bool          `json:"ready"`
	Processing    bool          `json:"processing"`
	StartTime     time.Time     `json:"start_time"`
	LastTick      time.Time     `json:"last_tick"`

	// Health and metrics
	Health        *ComponentHealth  `json:"health"`
	Metrics       *ComponentMetrics `json:"metrics"`

	// Engine states
	EngineStates  map[engines.EngineType]*engines.WrapperState `json:"engine_states"`

	// Configuration
	Config        *ComponentConfig `json:"config"`

	// Persistence metadata
	SavedAt       time.Time `json:"saved_at"`
	Version       string    `json:"version"`
}

// LoadBalancerState represents the complete state of a load balancer
type LoadBalancerState struct {
	// Identity
	ComponentID     string                 `json:"component_id"`
	ComponentType   ComponentType          `json:"component_type"`

	// Configuration
	Config          *LoadBalancingConfig   `json:"config"`
	ComponentConfig *ComponentConfig       `json:"component_config"`

	// Instance management
	NextInstanceID  int                    `json:"next_instance_id"`
	InstanceHealth  map[string]float64     `json:"instance_health"`

	// Load balancing state
	RoundRobinIndex    int                 `json:"round_robin_index"`
	WeightedSelections map[string]int      `json:"weighted_selections"`
	TotalWeight        int                 `json:"total_weight"`
	LastScaleUp        time.Time           `json:"last_scale_up"`
	LastScaleDown      time.Time           `json:"last_scale_down"`

	// Instance states
	InstanceStates  []*ComponentInstanceState `json:"instance_states"`

	// Runtime state
	Running         bool                      `json:"running"`
	Metrics         *ComponentMetrics         `json:"metrics"`

	// Persistence metadata
	SavedAt         time.Time                 `json:"saved_at"`
	Version         string                    `json:"version"`
}

// SaveComponentInstanceState saves the state of a component instance
func (spm *StatePersistenceManager) SaveComponentInstanceState(instance *ComponentInstance) error {
	spm.mutex.Lock()
	defer spm.mutex.Unlock()

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(spm.StateDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Collect engine states
	engineStates := make(map[engines.EngineType]*engines.WrapperState)
	for engineType, wrapper := range instance.Engines {
		if wrapper != nil {
			if err := wrapper.SaveState(); err != nil {
				return fmt.Errorf("failed to save engine %s state: %w", engineType, err)
			}
			// Note: SaveState saves to file, we would need a different method to get the state object
			// For now, we'll create a placeholder state
			engineStates[engineType] = &engines.WrapperState{
				EngineID:   wrapper.GetID(),
				EngineType: engineType,
			}
		}
	}

	// Create instance state
	state := &ComponentInstanceState{
		ID:            instance.ID,
		ComponentID:   instance.ComponentID,
		ComponentType: instance.ComponentType,
		Running:       instance.running,
		Ready:         instance.ReadyFlag.Load(),
		Processing:    instance.ProcessingFlag.Load(),
		StartTime:     instance.StartTime,
		LastTick:      instance.LastTickTime,
		Health:        instance.Health,
		Metrics:       instance.Metrics,
		EngineStates:  engineStates,
		Config:        instance.Config,
		SavedAt:       time.Now(),
		Version:       "1.0",
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instance state: %w", err)
	}

	// Write to file
	filename := filepath.Join(spm.StateDirectory, fmt.Sprintf("instance_%s.json", instance.ID))
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write instance state file: %w", err)
	}

	return nil
}

// LoadComponentInstanceState loads the state of a component instance
func (spm *StatePersistenceManager) LoadComponentInstanceState(instanceID string) (*ComponentInstanceState, error) {
	spm.mutex.RLock()
	defer spm.mutex.RUnlock()

	filename := filepath.Join(spm.StateDirectory, fmt.Sprintf("instance_%s.json", instanceID))
	
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("state file not found for instance %s", instanceID)
	}

	// Read file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read instance state file: %w", err)
	}

	// Deserialize from JSON
	var state ComponentInstanceState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance state: %w", err)
	}

	return &state, nil
}

// SaveLoadBalancerState saves the state of a load balancer
func (spm *StatePersistenceManager) SaveLoadBalancerState(lb *LoadBalancer) error {
	spm.mutex.Lock()
	defer spm.mutex.Unlock()

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(spm.StateDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Collect instance states
	instanceStates := make([]*ComponentInstanceState, len(lb.Instances))
	for i, instance := range lb.Instances {
		// Collect engine states for this instance
		engineStates := make(map[engines.EngineType]*engines.WrapperState)
		for engineType, wrapper := range instance.Engines {
			if wrapper != nil {
				if err := wrapper.SaveState(); err != nil {
					return fmt.Errorf("failed to save engine %s state for instance %s: %w", engineType, instance.ID, err)
				}
				// Note: SaveState saves to file, we would need a different method to get the state object
				// For now, we'll create a placeholder state
				engineStates[engineType] = &engines.WrapperState{
					EngineID:   wrapper.GetID(),
					EngineType: engineType,
				}
			}
		}

		instanceStates[i] = &ComponentInstanceState{
			ID:            instance.ID,
			ComponentID:   instance.ComponentID,
			ComponentType: instance.ComponentType,
			Running:       instance.running,
			Ready:         instance.ReadyFlag.Load(),
			Processing:    instance.ProcessingFlag.Load(),
			StartTime:     instance.StartTime,
			LastTick:      instance.LastTickTime,
			Health:        instance.Health,
			Metrics:       instance.Metrics,
			EngineStates:  engineStates,
			Config:        instance.Config,
			SavedAt:       time.Now(),
			Version:       "1.0",
		}
	}

	// Create load balancer state
	state := &LoadBalancerState{
		ComponentID:        lb.ComponentID,
		ComponentType:      lb.ComponentType,
		Config:             lb.Config,
		ComponentConfig:    lb.ComponentConfig,
		NextInstanceID:     lb.NextInstanceID,
		InstanceHealth:     lb.InstanceHealth,
		RoundRobinIndex:    lb.RoundRobinIndex,
		WeightedSelections: lb.WeightedSelections,
		TotalWeight:        lb.TotalWeight,
		LastScaleUp:        lb.LastScaleUp,
		LastScaleDown:      lb.LastScaleDown,
		InstanceStates:     instanceStates,
		Running:            lb.running,
		Metrics:            lb.Metrics,
		SavedAt:            time.Now(),
		Version:            "1.0",
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal load balancer state: %w", err)
	}

	// Write to file
	filename := filepath.Join(spm.StateDirectory, fmt.Sprintf("loadbalancer_%s.json", lb.ComponentID))
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write load balancer state file: %w", err)
	}

	return nil
}

// LoadLoadBalancerState loads the state of a load balancer
func (spm *StatePersistenceManager) LoadLoadBalancerState(componentID string) (*LoadBalancerState, error) {
	spm.mutex.RLock()
	defer spm.mutex.RUnlock()

	filename := filepath.Join(spm.StateDirectory, fmt.Sprintf("loadbalancer_%s.json", componentID))
	
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("state file not found for load balancer %s", componentID)
	}

	// Read file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read load balancer state file: %w", err)
	}

	// Deserialize from JSON
	var state LoadBalancerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal load balancer state: %w", err)
	}

	return &state, nil
}

// DeleteComponentInstanceState removes the state file for a component instance
func (spm *StatePersistenceManager) DeleteComponentInstanceState(instanceID string) error {
	spm.mutex.Lock()
	defer spm.mutex.Unlock()

	filename := filepath.Join(spm.StateDirectory, fmt.Sprintf("instance_%s.json", instanceID))
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete instance state file: %w", err)
	}

	return nil
}

// DeleteLoadBalancerState removes the state file for a load balancer
func (spm *StatePersistenceManager) DeleteLoadBalancerState(componentID string) error {
	spm.mutex.Lock()
	defer spm.mutex.Unlock()

	filename := filepath.Join(spm.StateDirectory, fmt.Sprintf("loadbalancer_%s.json", componentID))
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete load balancer state file: %w", err)
	}

	return nil
}

// ListSavedStates returns a list of all saved component states
func (spm *StatePersistenceManager) ListSavedStates() ([]string, error) {
	spm.mutex.RLock()
	defer spm.mutex.RUnlock()

	files, err := ioutil.ReadDir(spm.StateDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read state directory: %w", err)
	}

	var states []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			states = append(states, file.Name())
		}
	}

	return states, nil
}

// Global state persistence manager instance
var GlobalStatePersistenceManager *StatePersistenceManager

// InitializeStatePersistence initializes the global state persistence manager
func InitializeStatePersistence(stateDirectory string) {
	GlobalStatePersistenceManager = NewStatePersistenceManager(stateDirectory)
}
