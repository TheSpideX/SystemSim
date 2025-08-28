package components

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// ComponentFactory creates components from profiles
type ComponentFactory struct {
	profilesPath  string
	engineFactory *engines.EngineFactory
	registry      GlobalRegistryInterface
}

// NewComponentFactory creates a new component factory
func NewComponentFactory(profilesPath string, engineFactory *engines.EngineFactory) *ComponentFactory {
	return &ComponentFactory{
		profilesPath:  profilesPath,
		engineFactory: engineFactory,
		registry:      nil, // Registry can be set later
	}
}

// SetRegistry sets the global registry for components created by this factory
func (cf *ComponentFactory) SetRegistry(registry GlobalRegistryInterface) {
	cf.registry = registry
}

// CreateComponent creates a component from a profile
func (cf *ComponentFactory) CreateComponent(componentType ComponentType, componentID string) (*LoadBalancer, error) {
	// Load component profile
	config, err := cf.loadComponentProfile(componentType)
	if err != nil {
		return nil, fmt.Errorf("failed to load component profile: %w", err)
	}

	// Set component ID
	config.ID = componentID

	// Add default load balancer configuration if not present
	if config.LoadBalancer == nil {
		config.LoadBalancer = cf.createDefaultLoadBalancerConfig(componentType)
	}

	// Create load balancer with component configuration
	loadBalancer, err := NewLoadBalancer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	// Set registry if available
	if cf.registry != nil {
		loadBalancer.SetRegistry(cf.registry)
	}

	return loadBalancer, nil
}

// CreateComponentFromConfig creates a component from a configuration
func (cf *ComponentFactory) CreateComponentFromConfig(config *ComponentConfig) (*LoadBalancer, error) {
	// Add default load balancer configuration if not present
	if config.LoadBalancer == nil {
		config.LoadBalancer = cf.createDefaultLoadBalancerConfig(config.Type)
	}

	// Create load balancer with component configuration
	loadBalancer, err := NewLoadBalancer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	// Set registry if available
	if cf.registry != nil {
		loadBalancer.SetRegistry(cf.registry)
	}

	return loadBalancer, nil
}

// loadComponentProfile loads a component profile from file
func (cf *ComponentFactory) loadComponentProfile(componentType ComponentType) (*ComponentConfig, error) {
	// Determine profile filename
	var filename string
	switch componentType {
	case ComponentTypeDatabase:
		filename = "database_server.json"
	case ComponentTypeWebServer:
		filename = "web_server.json"
	case ComponentTypeCache:
		filename = "cache_server.json"
	case ComponentTypeLoadBalancer:
		filename = "load_balancer.json"
	default:
		return nil, fmt.Errorf("unknown component type: %s", componentType)
	}

	// Read profile file
	profilePath := filepath.Join(cf.profilesPath, filename)
	data, err := ioutil.ReadFile(profilePath)
	if err != nil {
		// If file doesn't exist, create a default profile
		return cf.createDefaultProfile(componentType), nil
	}

	// Parse JSON
	var config ComponentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse profile JSON: %w", err)
	}

	return &config, nil
}

// createDefaultProfile creates a default profile for a component type
func (cf *ComponentFactory) createDefaultProfile(componentType ComponentType) *ComponentConfig {
	baseConfig := &ComponentConfig{
		Type:             componentType,
		Name:             string(componentType),
		Description:      fmt.Sprintf("Default %s component", componentType),
		MaxConcurrentOps: 10,
		QueueCapacity:    100,
		TickTimeout:      time.Millisecond * 10,
		EngineProfiles:   make(map[engines.EngineType]string),
		ComplexityLevels: make(map[engines.EngineType]int),
		LoadBalancer:     cf.createDefaultLoadBalancerConfig(componentType),
	}

	switch componentType {
	case ComponentTypeDatabase:
		baseConfig.RequiredEngines = []engines.EngineType{
			engines.NetworkEngineType, // Input
			engines.CPUEngineType,
			engines.MemoryEngineType,
			engines.StorageEngineType,
			engines.NetworkEngineType, // Output
		}
		baseConfig.EngineProfiles[engines.CPUEngineType] = "database_cpu"
		baseConfig.EngineProfiles[engines.MemoryEngineType] = "database_memory"
		baseConfig.EngineProfiles[engines.StorageEngineType] = "database_storage"
		baseConfig.EngineProfiles[engines.NetworkEngineType] = "database_network"
		baseConfig.ComplexityLevels[engines.CPUEngineType] = 3
		baseConfig.ComplexityLevels[engines.MemoryEngineType] = 2
		baseConfig.ComplexityLevels[engines.StorageEngineType] = 4
		baseConfig.ComplexityLevels[engines.NetworkEngineType] = 2

	case ComponentTypeWebServer:
		baseConfig.RequiredEngines = []engines.EngineType{
			engines.NetworkEngineType, // Input
			engines.CPUEngineType,
			engines.MemoryEngineType,
			engines.NetworkEngineType, // Output
		}
		baseConfig.EngineProfiles[engines.CPUEngineType] = "webserver_cpu"
		baseConfig.EngineProfiles[engines.MemoryEngineType] = "webserver_memory"
		baseConfig.EngineProfiles[engines.NetworkEngineType] = "webserver_network"
		baseConfig.ComplexityLevels[engines.CPUEngineType] = 2
		baseConfig.ComplexityLevels[engines.MemoryEngineType] = 1
		baseConfig.ComplexityLevels[engines.NetworkEngineType] = 3

	case ComponentTypeCache:
		baseConfig.RequiredEngines = []engines.EngineType{
			engines.NetworkEngineType, // Input
			engines.CPUEngineType,
			engines.MemoryEngineType,
			engines.NetworkEngineType, // Output
		}
		baseConfig.EngineProfiles[engines.CPUEngineType] = "cache_cpu"
		baseConfig.EngineProfiles[engines.MemoryEngineType] = "cache_memory"
		baseConfig.EngineProfiles[engines.NetworkEngineType] = "cache_network"
		baseConfig.ComplexityLevels[engines.CPUEngineType] = 1
		baseConfig.ComplexityLevels[engines.MemoryEngineType] = 3
		baseConfig.ComplexityLevels[engines.NetworkEngineType] = 2

	case ComponentTypeLoadBalancer:
		baseConfig.RequiredEngines = []engines.EngineType{
			engines.NetworkEngineType, // Input
			engines.CPUEngineType,
			engines.NetworkEngineType, // Output
		}
		baseConfig.EngineProfiles[engines.CPUEngineType] = "loadbalancer_cpu"
		baseConfig.EngineProfiles[engines.NetworkEngineType] = "loadbalancer_network"
		baseConfig.ComplexityLevels[engines.CPUEngineType] = 1
		baseConfig.ComplexityLevels[engines.NetworkEngineType] = 4
	}

	// Create default decision graph
	baseConfig.DecisionGraph = cf.createDefaultDecisionGraph(componentType)

	return baseConfig
}

// createDefaultDecisionGraph creates a default decision graph for a component type
func (cf *ComponentFactory) createDefaultDecisionGraph(componentType ComponentType) *DecisionGraphConfig {
	switch componentType {
	case ComponentTypeDatabase:
		return &DecisionGraphConfig{
			StartNode: "network_input",
			EndNodes:  []string{"network_output"},
			Nodes: map[string]*DecisionNode{
				"network_input": {
					ID:         "network_input",
					Type:       "engine",
					EngineType: engines.NetworkEngineType,
					Conditions: map[string]string{"default": "cpu_process"},
				},
				"cpu_process": {
					ID:         "cpu_process",
					Type:       "engine",
					EngineType: engines.CPUEngineType,
					Conditions: map[string]string{"default": "memory_check"},
				},
				"memory_check": {
					ID:         "memory_check",
					Type:       "engine",
					EngineType: engines.MemoryEngineType,
					Conditions: map[string]string{"default": "storage_access"},
				},
				"storage_access": {
					ID:         "storage_access",
					Type:       "engine",
					EngineType: engines.StorageEngineType,
					Conditions: map[string]string{"default": "network_output"},
				},
				"network_output": {
					ID:   "network_output",
					Type: "engine",
					EngineType: engines.NetworkEngineType,
					Conditions: map[string]string{"default": "end"},
				},
				"end": {
					ID:   "end",
					Type: "end",
				},
			},
		}

	case ComponentTypeCache:
		return &DecisionGraphConfig{
			StartNode: "network_input",
			EndNodes:  []string{"network_output"},
			Nodes: map[string]*DecisionNode{
				"network_input": {
					ID:         "network_input",
					Type:       "engine",
					EngineType: engines.NetworkEngineType,
					Conditions: map[string]string{"default": "cpu_hash"},
				},
				"cpu_hash": {
					ID:         "cpu_hash",
					Type:       "engine",
					EngineType: engines.CPUEngineType,
					Conditions: map[string]string{"default": "memory_lookup"},
				},
				"memory_lookup": {
					ID:         "memory_lookup",
					Type:       "engine",
					EngineType: engines.MemoryEngineType,
					Conditions: map[string]string{"default": "network_output"},
				},
				"network_output": {
					ID:         "network_output",
					Type:       "engine",
					EngineType: engines.NetworkEngineType,
					Conditions: map[string]string{"default": "end"},
				},
				"end": {
					ID:   "end",
					Type: "end",
				},
			},
		}

	default:
		// Simple default graph
		return &DecisionGraphConfig{
			StartNode: "start",
			EndNodes:  []string{"end"},
			Nodes: map[string]*DecisionNode{
				"start": {
					ID:         "start",
					Type:       "decision",
					Conditions: map[string]string{"default": "end"},
				},
				"end": {
					ID:   "end",
					Type: "end",
				},
			},
		}
	}
}

// createDefaultLoadBalancerConfig creates a default load balancer configuration for a component type
func (cf *ComponentFactory) createDefaultLoadBalancerConfig(componentType ComponentType) *LoadBalancingConfig {
	switch componentType {
	case ComponentTypeDatabase:
		return &LoadBalancingConfig{
			Algorithm:    LoadBalancingLeastConnections, // Databases benefit from least connections
			MinInstances: 1,
			MaxInstances: 3,
			AutoScaling:  true,
		}
	case ComponentTypeWebServer:
		return &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin, // Web servers work well with round robin
			MinInstances: 2,
			MaxInstances: 5,
			AutoScaling:  true,
		}
	case ComponentTypeCache:
		return &LoadBalancingConfig{
			Algorithm:    LoadBalancingNone, // Cache typically single instance
			MinInstances: 1,
			MaxInstances: 1,
			AutoScaling:  false,
		}
	case ComponentTypeLoadBalancer:
		return &LoadBalancingConfig{
			Algorithm:    LoadBalancingWeighted, // Load balancers can use weighted
			MinInstances: 1,
			MaxInstances: 2,
			AutoScaling:  true,
		}
	default:
		return &LoadBalancingConfig{
			Algorithm:    LoadBalancingRoundRobin,
			MinInstances: 1,
			MaxInstances: 3,
			AutoScaling:  true,
		}
	}
}
