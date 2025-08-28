package components

import (
	"fmt"
	"log"
	"sync"
)

// UserFlowManager manages user flow configurations and routing
type UserFlowManager struct {
	config *UserFlowConfig
	mutex  sync.RWMutex
}

// NewUserFlowManager creates a new user flow manager
func NewUserFlowManager() *UserFlowManager {
	return &UserFlowManager{
		config: &UserFlowConfig{
			Flows: make(map[string]*UserFlow),
		},
	}
}

// AddFlow adds a new user flow
func (ufm *UserFlowManager) AddFlow(name string, flow *UserFlow) error {
	ufm.mutex.Lock()
	defer ufm.mutex.Unlock()

	if name == "" {
		return fmt.Errorf("flow name cannot be empty")
	}

	if flow == nil {
		return fmt.Errorf("flow cannot be nil")
	}

	ufm.config.Flows[name] = flow
	log.Printf("UserFlowManager: Added flow '%s' with %d steps", name, len(flow.Steps))

	return nil
}

// GetFlow retrieves a user flow by name
func (ufm *UserFlowManager) GetFlow(name string) (*UserFlow, error) {
	ufm.mutex.RLock()
	defer ufm.mutex.RUnlock()

	flow, exists := ufm.config.Flows[name]
	if !exists {
		return nil, fmt.Errorf("flow not found: %s", name)
	}

	return flow, nil
}

// GetNextComponent determines the next component based on user flow
func (ufm *UserFlowManager) GetNextComponent(currentComponent, operationType string) (string, error) {
	ufm.mutex.RLock()
	defer ufm.mutex.RUnlock()

	// Search through all flows to find routing for current component
	for flowName, flow := range ufm.config.Flows {
		for _, step := range flow.Steps {
			if step.ComponentID == currentComponent && step.Operation == operationType {
				// Check conditions for routing
				if len(step.Conditions) > 0 {
					// Use default routing for now - can be enhanced later with operation result data
					if defaultComponent, exists := step.Conditions["default"]; exists {
						log.Printf("UserFlowManager: Routing %s -> %s via flow '%s' (default)",
							currentComponent, defaultComponent, flowName)
						return defaultComponent, nil
					}

					// If no default, return first available condition
					for _, nextComponent := range step.Conditions {
						log.Printf("UserFlowManager: Routing %s -> %s via flow '%s' (first available)",
							currentComponent, nextComponent, flowName)
						return nextComponent, nil
					}
				}
			}
		}
	}

	// No routing found
	return "", fmt.Errorf("no routing found for component %s with operation %s", currentComponent, operationType)
}

// GetConfig returns the current user flow configuration
func (ufm *UserFlowManager) GetConfig() *UserFlowConfig {
	ufm.mutex.RLock()
	defer ufm.mutex.RUnlock()

	return ufm.config
}

// SetConfig sets the user flow configuration
func (ufm *UserFlowManager) SetConfig(config *UserFlowConfig) {
	ufm.mutex.Lock()
	defer ufm.mutex.Unlock()

	if config == nil {
		config = &UserFlowConfig{
			Flows: make(map[string]*UserFlow),
		}
	}

	ufm.config = config
	log.Printf("UserFlowManager: Updated configuration with %d flows", len(config.Flows))
}

// ListFlows returns all available flow names
func (ufm *UserFlowManager) ListFlows() []string {
	ufm.mutex.RLock()
	defer ufm.mutex.RUnlock()

	flows := make([]string, 0, len(ufm.config.Flows))
	for name := range ufm.config.Flows {
		flows = append(flows, name)
	}

	return flows
}

// RemoveFlow removes a user flow
func (ufm *UserFlowManager) RemoveFlow(name string) error {
	ufm.mutex.Lock()
	defer ufm.mutex.Unlock()

	if _, exists := ufm.config.Flows[name]; !exists {
		return fmt.Errorf("flow not found: %s", name)
	}

	delete(ufm.config.Flows, name)
	log.Printf("UserFlowManager: Removed flow '%s'", name)

	return nil
}

// CreateSimpleWebFlow creates a simple web application flow
func CreateSimpleWebFlow() *UserFlow {
	return &UserFlow{
		Name:        "Simple Web Application Flow",
		Description: "Basic web app flow: Load Balancer -> Web Server -> Database",
		Steps: []*UserFlowStep{
			{
				ComponentID: "load-balancer",
				Operation:   "http_request",
				Conditions: map[string]string{
					"default": "web-server",
				},
			},
			{
				ComponentID: "web-server",
				Operation:   "http_request",
				Conditions: map[string]string{
					"database_query": "database",
					"cache_lookup":   "cache",
				},
			},
			{
				ComponentID: "cache",
				Operation:   "cache_lookup",
				Conditions: map[string]string{
					"cache_miss": "database",
				},
			},
			{
				ComponentID: "database",
				Operation:   "database_query",
				Conditions: map[string]string{
					"response": "web-server",
				},
			},
		},
	}
}

// CreateMicroservicesFlow creates a microservices architecture flow
func CreateMicroservicesFlow() *UserFlow {
	return &UserFlow{
		Name:        "Microservices Architecture Flow",
		Description: "Complex microservices flow with API gateway and multiple services",
		Steps: []*UserFlowStep{
			{
				ComponentID: "api-gateway",
				Operation:   "route_request",
				Conditions: map[string]string{
					"user_service":  "user-service",
					"order_service": "order-service",
					"auth_service":  "auth-service",
				},
			},
			{
				ComponentID: "auth-service",
				Operation:   "authenticate",
				Conditions: map[string]string{
					"success": "user-service",
					"failure": "error-handler",
				},
			},
			{
				ComponentID: "user-service",
				Operation:   "get_user",
				Conditions: map[string]string{
					"database_query": "user-database",
					"cache_lookup":   "user-cache",
				},
			},
			{
				ComponentID: "order-service",
				Operation:   "process_order",
				Conditions: map[string]string{
					"payment":        "payment-service",
					"inventory":      "inventory-service",
					"database_query": "order-database",
				},
			},
			{
				ComponentID: "payment-service",
				Operation:   "process_payment",
				Conditions: map[string]string{
					"success": "notification-service",
					"failure": "error-handler",
				},
			},
		},
	}
}

// ValidateFlow validates a user flow configuration
func ValidateFlow(flow *UserFlow) error {
	if flow == nil {
		return fmt.Errorf("flow cannot be nil")
	}

	if flow.Name == "" {
		return fmt.Errorf("flow name is required")
	}

	if len(flow.Steps) == 0 {
		return fmt.Errorf("flow must have at least one step")
	}

	// Validate each step
	componentIDs := make(map[string]bool)
	for i, step := range flow.Steps {
		if step.ComponentID == "" {
			return fmt.Errorf("step %d: component ID is required", i)
		}

		if step.Operation == "" {
			return fmt.Errorf("step %d: operation is required", i)
		}

		componentIDs[step.ComponentID] = true
	}

	// Validate that condition targets reference valid components or external services
	for i, step := range flow.Steps {
		for condition, target := range step.Conditions {
			if condition == "" {
				return fmt.Errorf("step %d: condition cannot be empty", i)
			}

			if target == "" {
				return fmt.Errorf("step %d: condition target cannot be empty", i)
			}

			// Note: We don't validate that targets exist in the flow
			// because they might reference external components
		}
	}

	return nil
}
