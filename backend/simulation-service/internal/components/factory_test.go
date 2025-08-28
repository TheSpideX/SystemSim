package components

import (
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// MockRegistry implements GlobalRegistryInterface for testing
type MockRegistry struct {
	components map[string]chan *engines.Operation
	health     map[string]float64
	load       map[string]BufferStatus
}

func (mr *MockRegistry) Register(componentID string, inputChannel chan *engines.Operation) {
	if mr.components == nil {
		mr.components = make(map[string]chan *engines.Operation)
		mr.health = make(map[string]float64)
		mr.load = make(map[string]BufferStatus)
	}
	mr.components[componentID] = inputChannel
	mr.health[componentID] = 1.0
	mr.load[componentID] = BufferStatusNormal
}

func (mr *MockRegistry) Unregister(componentID string) {
	delete(mr.components, componentID)
	delete(mr.health, componentID)
	delete(mr.load, componentID)
}

func (mr *MockRegistry) GetChannel(componentID string) chan *engines.Operation {
	return mr.components[componentID]
}

func (mr *MockRegistry) GetAllComponents() map[string]chan *engines.Operation {
	return mr.components
}

func (mr *MockRegistry) GetHealth(componentID string) float64 {
	if health, exists := mr.health[componentID]; exists {
		return health
	}
	return 0.0
}

func (mr *MockRegistry) UpdateHealth(componentID string, health float64) {
	if mr.health == nil {
		mr.health = make(map[string]float64)
	}
	mr.health[componentID] = health
}

func (mr *MockRegistry) GetLoad(componentID string) BufferStatus {
	if load, exists := mr.load[componentID]; exists {
		return load
	}
	return BufferStatusEmergency
}

func (mr *MockRegistry) UpdateLoad(componentID string, status BufferStatus) {
	if mr.load == nil {
		mr.load = make(map[string]BufferStatus)
	}
	mr.load[componentID] = status
}

func (mr *MockRegistry) Start() error {
	return nil
}

func (mr *MockRegistry) Stop() error {
	return nil
}

func TestComponentFactory_CreateComponent(t *testing.T) {
	// Create factory with test profiles path and nil engine factory
	factory := NewComponentFactory("test-profiles", nil)

	// Test creating different component types
	componentTypes := []ComponentType{
		ComponentTypeWebServer,
		ComponentTypeDatabase,
		ComponentTypeCache,
		ComponentTypeLoadBalancer,
	}

	for _, componentType := range componentTypes {
		t.Run(string(componentType), func(t *testing.T) {
			componentID := "test-" + string(componentType)
			
			// Create component
			loadBalancer, err := factory.CreateComponent(componentType, componentID)
			if err != nil {
				t.Fatalf("Failed to create component: %v", err)
			}

			// Verify component properties
			if loadBalancer.ComponentID != componentID {
				t.Errorf("Expected component ID %s, got %s", componentID, loadBalancer.ComponentID)
			}

			if loadBalancer.ComponentType != componentType {
				t.Errorf("Expected component type %v, got %v", componentType, loadBalancer.ComponentType)
			}

			// Verify load balancer configuration exists
			if loadBalancer.Config == nil {
				t.Fatal("Expected load balancer configuration")
			}

			// Verify default load balancer settings are appropriate for component type
			switch componentType {
			case ComponentTypeDatabase:
				if loadBalancer.Config.Algorithm != LoadBalancingLeastConnections {
					t.Errorf("Expected least connections algorithm for database, got %v", loadBalancer.Config.Algorithm)
				}
			case ComponentTypeWebServer:
				if loadBalancer.Config.Algorithm != LoadBalancingRoundRobin {
					t.Errorf("Expected round robin algorithm for web server, got %v", loadBalancer.Config.Algorithm)
				}
			case ComponentTypeCache:
				if loadBalancer.Config.Algorithm != LoadBalancingNone {
					t.Errorf("Expected no load balancing for cache, got %v", loadBalancer.Config.Algorithm)
				}
				if loadBalancer.Config.MinInstances != 1 || loadBalancer.Config.MaxInstances != 1 {
					t.Errorf("Expected single instance for cache, got min=%d max=%d", 
						loadBalancer.Config.MinInstances, loadBalancer.Config.MaxInstances)
				}
			case ComponentTypeLoadBalancer:
				if loadBalancer.Config.Algorithm != LoadBalancingWeighted {
					t.Errorf("Expected weighted algorithm for load balancer, got %v", loadBalancer.Config.Algorithm)
				}
			}

			// Verify required engines are set
			if len(loadBalancer.ComponentConfig.RequiredEngines) == 0 {
				t.Error("Expected required engines to be set")
			}
		})
	}
}

func TestComponentFactory_CreateComponentFromConfig(t *testing.T) {
	// Create factory with test profiles path and nil engine factory
	factory := NewComponentFactory("test-profiles", nil)

	// Create custom configuration
	config := &ComponentConfig{
		ID:               "custom-component",
		Type:             ComponentTypeWebServer,
		Name:             "Custom Web Server",
		Description:      "Custom web server component",
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType},
		MaxConcurrentOps: 15,
		QueueCapacity:    50,
		TickTimeout:      time.Millisecond * 20,
		EngineProfiles:   make(map[engines.EngineType]string),
		ComplexityLevels: make(map[engines.EngineType]int),
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:    LoadBalancingWeighted,
			MinInstances: 3,
			MaxInstances: 6,
			AutoScaling:  true,
		},
	}

	// Create component from config
	loadBalancer, err := factory.CreateComponentFromConfig(config)
	if err != nil {
		t.Fatalf("Failed to create component from config: %v", err)
	}

	// Verify component properties
	if loadBalancer.ComponentID != config.ID {
		t.Errorf("Expected component ID %s, got %s", config.ID, loadBalancer.ComponentID)
	}

	if loadBalancer.ComponentType != config.Type {
		t.Errorf("Expected component type %v, got %v", config.Type, loadBalancer.ComponentType)
	}

	// Verify load balancer configuration
	if loadBalancer.Config.Algorithm != LoadBalancingWeighted {
		t.Errorf("Expected weighted algorithm, got %v", loadBalancer.Config.Algorithm)
	}

	if loadBalancer.Config.MinInstances != 3 {
		t.Errorf("Expected min instances 3, got %d", loadBalancer.Config.MinInstances)
	}

	if loadBalancer.Config.MaxInstances != 6 {
		t.Errorf("Expected max instances 6, got %d", loadBalancer.Config.MaxInstances)
	}

	// Verify component configuration
	if loadBalancer.ComponentConfig.MaxConcurrentOps != 15 {
		t.Errorf("Expected max concurrent ops 15, got %d", loadBalancer.ComponentConfig.MaxConcurrentOps)
	}

	if loadBalancer.ComponentConfig.QueueCapacity != 50 {
		t.Errorf("Expected queue capacity 50, got %d", loadBalancer.ComponentConfig.QueueCapacity)
	}
}

func TestComponentFactory_CreateComponentWithoutLoadBalancerConfig(t *testing.T) {
	// Create factory with test profiles path and nil engine factory
	factory := NewComponentFactory("test-profiles", nil)

	// Create configuration without load balancer config
	config := &ComponentConfig{
		ID:               "no-lb-component",
		Type:             ComponentTypeDatabase,
		Name:             "Database Without LB Config",
		Description:      "Database component without load balancer config",
		RequiredEngines:  []engines.EngineType{engines.NetworkEngineType, engines.CPUEngineType, engines.StorageEngineType},
		MaxConcurrentOps: 10,
		QueueCapacity:    20,
		TickTimeout:      time.Millisecond * 10,
		EngineProfiles:   make(map[engines.EngineType]string),
		ComplexityLevels: make(map[engines.EngineType]int),
		// LoadBalancer is nil - should get default
	}

	// Create component from config
	loadBalancer, err := factory.CreateComponentFromConfig(config)
	if err != nil {
		t.Fatalf("Failed to create component from config: %v", err)
	}

	// Verify default load balancer configuration was applied
	if loadBalancer.Config == nil {
		t.Fatal("Expected load balancer configuration to be created")
	}

	// Should have database-appropriate defaults
	if loadBalancer.Config.Algorithm != LoadBalancingLeastConnections {
		t.Errorf("Expected least connections algorithm for database, got %v", loadBalancer.Config.Algorithm)
	}

	if !loadBalancer.Config.AutoScaling {
		t.Error("Expected auto-scaling to be enabled for database")
	}
}

func TestComponentFactory_DefaultLoadBalancerConfigs(t *testing.T) {
	// Create factory with test profiles path and nil engine factory
	factory := NewComponentFactory("test-profiles", nil)

	// Test default configurations for each component type
	testCases := []struct {
		componentType    ComponentType
		expectedAlgorithm LoadBalancingAlgorithm
		expectedMinInstances int
		expectedMaxInstances int
		expectedAutoScaling  bool
	}{
		{ComponentTypeWebServer, LoadBalancingRoundRobin, 2, 5, true},
		{ComponentTypeDatabase, LoadBalancingLeastConnections, 1, 3, true},
		{ComponentTypeCache, LoadBalancingNone, 1, 1, false},
		{ComponentTypeLoadBalancer, LoadBalancingWeighted, 1, 2, true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.componentType), func(t *testing.T) {
			config := factory.createDefaultLoadBalancerConfig(tc.componentType)

			if config.Algorithm != tc.expectedAlgorithm {
				t.Errorf("Expected algorithm %v, got %v", tc.expectedAlgorithm, config.Algorithm)
			}

			if config.MinInstances != tc.expectedMinInstances {
				t.Errorf("Expected min instances %d, got %d", tc.expectedMinInstances, config.MinInstances)
			}

			if config.MaxInstances != tc.expectedMaxInstances {
				t.Errorf("Expected max instances %d, got %d", tc.expectedMaxInstances, config.MaxInstances)
			}

			if config.AutoScaling != tc.expectedAutoScaling {
				t.Errorf("Expected auto-scaling %v, got %v", tc.expectedAutoScaling, config.AutoScaling)
			}
		})
	}
}

func TestComponentFactory_WithRegistry(t *testing.T) {
	// Create factory with test profiles path and nil engine factory
	factory := NewComponentFactory("test-profiles", nil)

	// Create a mock registry
	registry := &MockRegistry{
		components: make(map[string]chan *engines.Operation),
		health:     make(map[string]float64),
		load:       make(map[string]BufferStatus),
	}
	factory.SetRegistry(registry)

	// Create component
	loadBalancer, err := factory.CreateComponent(ComponentTypeWebServer, "test-with-registry")
	if err != nil {
		t.Fatalf("Failed to create component: %v", err)
	}

	// Verify registry was set (we can't easily test this without exposing internal state,
	// but we can verify the component was created successfully)
	if loadBalancer == nil {
		t.Fatal("Expected component to be created")
	}

	if loadBalancer.ComponentID != "test-with-registry" {
		t.Errorf("Expected component ID test-with-registry, got %s", loadBalancer.ComponentID)
	}
}

func TestComponentFactory_ProfileLoading(t *testing.T) {
	// Create factory with test profiles path and nil engine factory
	factory := NewComponentFactory("test-profiles", nil)

	// Test that profiles are loaded correctly for different component types
	componentTypes := []ComponentType{
		ComponentTypeWebServer,
		ComponentTypeDatabase,
		ComponentTypeCache,
		ComponentTypeLoadBalancer,
	}

	for _, componentType := range componentTypes {
		t.Run(string(componentType), func(t *testing.T) {
			// Load profile
			config, err := factory.loadComponentProfile(componentType)
			if err != nil {
				t.Fatalf("Failed to load profile for %s: %v", componentType, err)
			}

			// Verify basic profile properties
			if config.Type != componentType {
				t.Errorf("Expected type %v, got %v", componentType, config.Type)
			}

			if config.Name == "" {
				t.Error("Expected profile to have a name")
			}

			if len(config.RequiredEngines) == 0 {
				t.Error("Expected profile to have required engines")
			}

			if config.LoadBalancer == nil {
				t.Error("Expected profile to have load balancer configuration")
			}
		})
	}
}
