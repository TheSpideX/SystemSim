package components

import (
	"testing"
)

func TestUserFlowManager_BasicOperations(t *testing.T) {
	manager := NewUserFlowManager()

	// Test adding a flow
	flow := CreateSimpleWebFlow()
	err := manager.AddFlow("simple-web", flow)
	if err != nil {
		t.Fatalf("Failed to add flow: %v", err)
	}

	// Test getting a flow
	retrievedFlow, err := manager.GetFlow("simple-web")
	if err != nil {
		t.Fatalf("Failed to get flow: %v", err)
	}

	if retrievedFlow.Name != "Simple Web Application Flow" {
		t.Errorf("Expected flow name 'Simple Web Application Flow', got '%s'", retrievedFlow.Name)
	}

	// Test listing flows
	flows := manager.ListFlows()
	if len(flows) != 1 {
		t.Errorf("Expected 1 flow, got %d", len(flows))
	}

	if flows[0] != "simple-web" {
		t.Errorf("Expected flow name 'simple-web', got '%s'", flows[0])
	}
}

func TestUserFlowManager_NextComponentRouting(t *testing.T) {
	manager := NewUserFlowManager()

	// Add simple web flow
	flow := CreateSimpleWebFlow()
	err := manager.AddFlow("simple-web", flow)
	if err != nil {
		t.Fatalf("Failed to add flow: %v", err)
	}

	// Test routing from load balancer
	nextComponent, err := manager.GetNextComponent("load-balancer", "http_request")
	if err != nil {
		t.Fatalf("Failed to get next component: %v", err)
	}

	if nextComponent != "web-server" {
		t.Errorf("Expected next component 'web-server', got '%s'", nextComponent)
	}

	// Test routing from web server to database
	nextComponent, err = manager.GetNextComponent("web-server", "http_request")
	if err != nil {
		t.Fatalf("Failed to get next component: %v", err)
	}

	// Should return one of the configured targets (database or cache)
	if nextComponent != "database" && nextComponent != "cache" {
		t.Errorf("Expected next component 'database' or 'cache', got '%s'", nextComponent)
	}

	// Test routing for non-existent component
	_, err = manager.GetNextComponent("nonexistent", "http_request")
	if err == nil {
		t.Error("Expected error for non-existent component routing")
	}
}

func TestUserFlowManager_MicroservicesFlow(t *testing.T) {
	manager := NewUserFlowManager()

	// Add microservices flow
	flow := CreateMicroservicesFlow()
	err := manager.AddFlow("microservices", flow)
	if err != nil {
		t.Fatalf("Failed to add flow: %v", err)
	}

	// Test routing from API gateway
	nextComponent, err := manager.GetNextComponent("api-gateway", "route_request")
	if err != nil {
		t.Fatalf("Failed to get next component: %v", err)
	}

	// Should return one of the configured services
	expectedServices := []string{"user-service", "order-service", "auth-service"}
	found := false
	for _, service := range expectedServices {
		if nextComponent == service {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected next component to be one of %v, got '%s'", expectedServices, nextComponent)
	}

	// Test routing from auth service
	nextComponent, err = manager.GetNextComponent("auth-service", "authenticate")
	if err != nil {
		t.Fatalf("Failed to get next component: %v", err)
	}

	if nextComponent != "user-service" && nextComponent != "error-handler" {
		t.Errorf("Expected next component 'user-service' or 'error-handler', got '%s'", nextComponent)
	}
}

func TestUserFlowManager_FlowValidation(t *testing.T) {
	// Test valid flow
	validFlow := CreateSimpleWebFlow()
	err := ValidateFlow(validFlow)
	if err != nil {
		t.Errorf("Valid flow should pass validation: %v", err)
	}

	// Test nil flow
	err = ValidateFlow(nil)
	if err == nil {
		t.Error("Nil flow should fail validation")
	}

	// Test flow without name
	invalidFlow := &UserFlow{
		Description: "Flow without name",
		Steps:       []*UserFlowStep{},
	}
	err = ValidateFlow(invalidFlow)
	if err == nil {
		t.Error("Flow without name should fail validation")
	}

	// Test flow without steps
	invalidFlow2 := &UserFlow{
		Name:        "Flow without steps",
		Description: "This flow has no steps",
		Steps:       []*UserFlowStep{},
	}
	err = ValidateFlow(invalidFlow2)
	if err == nil {
		t.Error("Flow without steps should fail validation")
	}

	// Test step without component ID
	invalidFlow3 := &UserFlow{
		Name:        "Flow with invalid step",
		Description: "This flow has an invalid step",
		Steps: []*UserFlowStep{
			{
				Operation: "test_operation",
				Conditions: map[string]string{
					"default": "next-component",
				},
			},
		},
	}
	err = ValidateFlow(invalidFlow3)
	if err == nil {
		t.Error("Flow with step missing component ID should fail validation")
	}

	// Test step without operation
	invalidFlow4 := &UserFlow{
		Name:        "Flow with invalid step",
		Description: "This flow has an invalid step",
		Steps: []*UserFlowStep{
			{
				ComponentID: "test-component",
				Conditions: map[string]string{
					"default": "next-component",
				},
			},
		},
	}
	err = ValidateFlow(invalidFlow4)
	if err == nil {
		t.Error("Flow with step missing operation should fail validation")
	}
}

func TestUserFlowManager_RemoveFlow(t *testing.T) {
	manager := NewUserFlowManager()

	// Add a flow
	flow := CreateSimpleWebFlow()
	err := manager.AddFlow("test-flow", flow)
	if err != nil {
		t.Fatalf("Failed to add flow: %v", err)
	}

	// Verify flow exists
	flows := manager.ListFlows()
	if len(flows) != 1 {
		t.Errorf("Expected 1 flow, got %d", len(flows))
	}

	// Remove the flow
	err = manager.RemoveFlow("test-flow")
	if err != nil {
		t.Fatalf("Failed to remove flow: %v", err)
	}

	// Verify flow is removed
	flows = manager.ListFlows()
	if len(flows) != 0 {
		t.Errorf("Expected 0 flows after removal, got %d", len(flows))
	}

	// Try to remove non-existent flow
	err = manager.RemoveFlow("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent flow")
	}
}

func TestUserFlowManager_SetGetConfig(t *testing.T) {
	manager := NewUserFlowManager()

	// Create a config with flows
	config := &UserFlowConfig{
		Flows: map[string]*UserFlow{
			"flow1": CreateSimpleWebFlow(),
			"flow2": CreateMicroservicesFlow(),
		},
	}

	// Set the config
	manager.SetConfig(config)

	// Get the config
	retrievedConfig := manager.GetConfig()
	if len(retrievedConfig.Flows) != 2 {
		t.Errorf("Expected 2 flows in config, got %d", len(retrievedConfig.Flows))
	}

	// Test setting nil config
	manager.SetConfig(nil)
	retrievedConfig = manager.GetConfig()
	if retrievedConfig == nil || len(retrievedConfig.Flows) != 0 {
		t.Error("Setting nil config should result in empty flows map")
	}
}
