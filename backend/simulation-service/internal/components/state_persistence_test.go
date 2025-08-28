package components

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestStatePersistenceManager_ComponentInstance(t *testing.T) {
	// Create temporary directory for state files
	tempDir, err := os.MkdirTemp("", "state_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize state persistence manager
	spm := NewStatePersistenceManager(tempDir)

	// Create a test component instance
	config := &ComponentConfig{
		ID:   "test-instance-1",
		Type: ComponentTypeCPU,
		RequiredEngines: []engines.EngineType{
			engines.CPUEngineType,
		},
		EngineProfiles: map[engines.EngineType]string{
			engines.CPUEngineType: "default_cpu",
		},
	}

	instance, err := NewComponentInstance(config)
	if err != nil {
		t.Fatalf("Failed to create component instance: %v", err)
	}

	// Set some test state
	instance.Health.Status = "YELLOW"
	instance.Health.CurrentCPU = 0.75
	instance.Metrics.TotalOperations = 100
	instance.Metrics.CompletedOps = 95
	instance.Metrics.FailedOps = 5

	// Save state
	err = spm.SaveComponentInstanceState(instance)
	if err != nil {
		t.Fatalf("Failed to save component instance state: %v", err)
	}

	// Verify state file exists
	stateFile := filepath.Join(tempDir, "instance_test-instance-1.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatalf("State file was not created: %s", stateFile)
	}

	// Load state
	loadedState, err := spm.LoadComponentInstanceState("test-instance-1")
	if err != nil {
		t.Fatalf("Failed to load component instance state: %v", err)
	}

	// Verify loaded state
	if loadedState.ID != "test-instance-1" {
		t.Errorf("Expected ID 'test-instance-1', got %s", loadedState.ID)
	}

	if loadedState.ComponentType != ComponentTypeCPU {
		t.Errorf("Expected component type %v, got %v", ComponentTypeCPU, loadedState.ComponentType)
	}

	if loadedState.Health.Status != "YELLOW" {
		t.Errorf("Expected health status 'YELLOW', got %s", loadedState.Health.Status)
	}

	if loadedState.Health.CurrentCPU != 0.75 {
		t.Errorf("Expected CPU utilization 0.75, got %f", loadedState.Health.CurrentCPU)
	}

	if loadedState.Metrics.TotalOperations != 100 {
		t.Errorf("Expected total operations 100, got %d", loadedState.Metrics.TotalOperations)
	}

	if loadedState.Metrics.CompletedOps != 95 {
		t.Errorf("Expected completed operations 95, got %d", loadedState.Metrics.CompletedOps)
	}

	if loadedState.Metrics.FailedOps != 5 {
		t.Errorf("Expected failed operations 5, got %d", loadedState.Metrics.FailedOps)
	}

	// Verify version and timestamp
	if loadedState.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", loadedState.Version)
	}

	if loadedState.SavedAt.IsZero() {
		t.Error("Expected non-zero saved timestamp")
	}
}

func TestStatePersistenceManager_LoadBalancer(t *testing.T) {
	// Create temporary directory for state files
	tempDir, err := os.MkdirTemp("", "state_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize state persistence manager
	spm := NewStatePersistenceManager(tempDir)

	// Create a test load balancer
	config := &ComponentConfig{
		ID:   "test-loadbalancer-1",
		Type: ComponentTypeCPU,
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:     "round_robin",
			MaxInstances:  5,
			MinInstances:  1,
			AutoScaling:   true,
		},
		RequiredEngines: []engines.EngineType{
			engines.CPUEngineType,
		},
		EngineProfiles: map[engines.EngineType]string{
			engines.CPUEngineType: "default_cpu",
		},
	}

	lb, err := NewLoadBalancer(config)
	if err != nil {
		t.Fatalf("Failed to create load balancer: %v", err)
	}

	// Set some test state
	lb.RoundRobinIndex = 1
	lb.NextInstanceID = 3
	lb.WeightedSelections = map[string]int{
		"instance-1": 10,
		"instance-2": 15,
	}
	lb.TotalWeight = 25
	lb.InstanceHealth = map[string]float64{
		"instance-1": 0.9,
		"instance-2": 0.8,
	}
	lb.Metrics.TotalOperations = 200
	lb.Metrics.CompletedOps = 190
	lb.Metrics.FailedOps = 10

	// Save state
	err = spm.SaveLoadBalancerState(lb)
	if err != nil {
		t.Fatalf("Failed to save load balancer state: %v", err)
	}

	// Verify state file exists
	stateFile := filepath.Join(tempDir, "loadbalancer_test-loadbalancer-1.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatalf("State file was not created: %s", stateFile)
	}

	// Load state
	loadedState, err := spm.LoadLoadBalancerState("test-loadbalancer-1")
	if err != nil {
		t.Fatalf("Failed to load load balancer state: %v", err)
	}

	// Verify loaded state
	if loadedState.ComponentID != "test-loadbalancer-1" {
		t.Errorf("Expected component ID 'test-loadbalancer-1', got %s", loadedState.ComponentID)
	}

	if loadedState.ComponentType != ComponentTypeCPU {
		t.Errorf("Expected component type %v, got %v", ComponentTypeCPU, loadedState.ComponentType)
	}

	if loadedState.RoundRobinIndex != 1 {
		t.Errorf("Expected round robin index 1, got %d", loadedState.RoundRobinIndex)
	}

	if loadedState.NextInstanceID != 3 {
		t.Errorf("Expected next instance ID 3, got %d", loadedState.NextInstanceID)
	}

	if loadedState.TotalWeight != 25 {
		t.Errorf("Expected total weight 25, got %d", loadedState.TotalWeight)
	}

	// Verify weighted selections
	if len(loadedState.WeightedSelections) != 2 {
		t.Errorf("Expected 2 weighted selections, got %d", len(loadedState.WeightedSelections))
	}

	if loadedState.WeightedSelections["instance-1"] != 10 {
		t.Errorf("Expected instance-1 selections 10, got %d", loadedState.WeightedSelections["instance-1"])
	}

	if loadedState.WeightedSelections["instance-2"] != 15 {
		t.Errorf("Expected instance-2 selections 15, got %d", loadedState.WeightedSelections["instance-2"])
	}

	// Verify instance health
	if len(loadedState.InstanceHealth) != 2 {
		t.Errorf("Expected 2 instance health entries, got %d", len(loadedState.InstanceHealth))
	}

	if loadedState.InstanceHealth["instance-1"] != 0.9 {
		t.Errorf("Expected instance-1 health 0.9, got %f", loadedState.InstanceHealth["instance-1"])
	}

	if loadedState.InstanceHealth["instance-2"] != 0.8 {
		t.Errorf("Expected instance-2 health 0.8, got %f", loadedState.InstanceHealth["instance-2"])
	}

	// Verify metrics
	if loadedState.Metrics.TotalOperations != 200 {
		t.Errorf("Expected total operations 200, got %d", loadedState.Metrics.TotalOperations)
	}

	if loadedState.Metrics.CompletedOps != 190 {
		t.Errorf("Expected completed operations 190, got %d", loadedState.Metrics.CompletedOps)
	}

	if loadedState.Metrics.FailedOps != 10 {
		t.Errorf("Expected failed operations 10, got %d", loadedState.Metrics.FailedOps)
	}

	// Verify version and timestamp
	if loadedState.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", loadedState.Version)
	}

	if loadedState.SavedAt.IsZero() {
		t.Error("Expected non-zero saved timestamp")
	}
}

func TestStatePersistenceManager_DeleteStates(t *testing.T) {
	// Create temporary directory for state files
	tempDir, err := os.MkdirTemp("", "state_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize state persistence manager
	spm := NewStatePersistenceManager(tempDir)

	// Create test files
	instanceFile := filepath.Join(tempDir, "instance_test-instance.json")
	lbFile := filepath.Join(tempDir, "loadbalancer_test-lb.json")

	// Create dummy files
	if err := os.WriteFile(instanceFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test instance file: %v", err)
	}

	if err := os.WriteFile(lbFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test load balancer file: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
		t.Fatal("Instance file should exist")
	}

	if _, err := os.Stat(lbFile); os.IsNotExist(err) {
		t.Fatal("Load balancer file should exist")
	}

	// Delete instance state
	err = spm.DeleteComponentInstanceState("test-instance")
	if err != nil {
		t.Fatalf("Failed to delete instance state: %v", err)
	}

	// Verify instance file is deleted
	if _, err := os.Stat(instanceFile); !os.IsNotExist(err) {
		t.Error("Instance file should be deleted")
	}

	// Delete load balancer state
	err = spm.DeleteLoadBalancerState("test-lb")
	if err != nil {
		t.Fatalf("Failed to delete load balancer state: %v", err)
	}

	// Verify load balancer file is deleted
	if _, err := os.Stat(lbFile); !os.IsNotExist(err) {
		t.Error("Load balancer file should be deleted")
	}
}

func TestStatePersistenceManager_ListSavedStates(t *testing.T) {
	// Create temporary directory for state files
	tempDir, err := os.MkdirTemp("", "state_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize state persistence manager
	spm := NewStatePersistenceManager(tempDir)

	// Initially should be empty
	states, err := spm.ListSavedStates()
	if err != nil {
		t.Fatalf("Failed to list saved states: %v", err)
	}

	if len(states) != 0 {
		t.Errorf("Expected 0 saved states, got %d", len(states))
	}

	// Create test files
	testFiles := []string{
		"instance_test1.json",
		"loadbalancer_test2.json",
		"instance_test3.json",
		"other_file.txt", // Should be ignored
	}

	for _, filename := range testFiles {
		filepath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filepath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// List saved states
	states, err = spm.ListSavedStates()
	if err != nil {
		t.Fatalf("Failed to list saved states: %v", err)
	}

	// Should only include JSON files
	expectedCount := 3 // Only the .json files
	if len(states) != expectedCount {
		t.Errorf("Expected %d saved states, got %d", expectedCount, len(states))
	}

	// Verify all returned files are JSON files
	for _, state := range states {
		if filepath.Ext(state) != ".json" {
			t.Errorf("Expected JSON file, got %s", state)
		}
	}
}

func TestComponentInstance_SaveLoadState_Integration(t *testing.T) {
	// Create temporary directory for state files
	tempDir, err := os.MkdirTemp("", "state_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize global state persistence manager
	InitializeStatePersistence(tempDir)

	// Create a test component instance
	config := &ComponentConfig{
		ID:   "integration-test-instance",
		Type: ComponentTypeCPU,
		RequiredEngines: []engines.EngineType{
			engines.CPUEngineType,
		},
		EngineProfiles: map[engines.EngineType]string{
			engines.CPUEngineType: "default_cpu",
		},
	}

	instance, err := NewComponentInstance(config)
	if err != nil {
		t.Fatalf("Failed to create component instance: %v", err)
	}

	// Set some test state
	instance.Health.Status = "YELLOW"
	instance.Health.CurrentCPU = 0.65
	instance.Metrics.TotalOperations = 150
	instance.Metrics.CompletedOps = 140
	instance.Metrics.FailedOps = 10

	// Save state using component method
	err = instance.SaveState()
	if err != nil {
		t.Fatalf("Failed to save component instance state: %v", err)
	}

	// Create a new instance to load the state into
	newInstance, err := NewComponentInstance(config)
	if err != nil {
		t.Fatalf("Failed to create new component instance: %v", err)
	}

	// Load state using component method
	err = newInstance.LoadState("integration-test-instance")
	if err != nil {
		t.Fatalf("Failed to load component instance state: %v", err)
	}

	// Verify state was restored
	if newInstance.ID != "integration-test-instance" {
		t.Errorf("Expected ID 'integration-test-instance', got %s", newInstance.ID)
	}

	if newInstance.Health.Status != "YELLOW" {
		t.Errorf("Expected health status 'YELLOW', got %s", newInstance.Health.Status)
	}

	if newInstance.Health.CurrentCPU != 0.65 {
		t.Errorf("Expected CPU utilization 0.65, got %f", newInstance.Health.CurrentCPU)
	}

	if newInstance.Metrics.TotalOperations != 150 {
		t.Errorf("Expected total operations 150, got %d", newInstance.Metrics.TotalOperations)
	}

	if newInstance.Metrics.CompletedOps != 140 {
		t.Errorf("Expected completed operations 140, got %d", newInstance.Metrics.CompletedOps)
	}

	if newInstance.Metrics.FailedOps != 10 {
		t.Errorf("Expected failed operations 10, got %d", newInstance.Metrics.FailedOps)
	}
}

func TestLoadBalancer_SaveLoadState_Integration(t *testing.T) {
	// Create temporary directory for state files
	tempDir, err := os.MkdirTemp("", "state_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize global state persistence manager
	InitializeStatePersistence(tempDir)

	// Create a test load balancer
	config := &ComponentConfig{
		ID:   "integration-test-lb",
		Type: ComponentTypeCPU,
		LoadBalancer: &LoadBalancingConfig{
			Algorithm:     "weighted",
			MaxInstances:  5,
			MinInstances:  1,
			AutoScaling:   true,
		},
		RequiredEngines: []engines.EngineType{
			engines.CPUEngineType,
		},
		EngineProfiles: map[engines.EngineType]string{
			engines.CPUEngineType: "default_cpu",
		},
	}

	lb, err := NewLoadBalancer(config)
	if err != nil {
		t.Fatalf("Failed to create load balancer: %v", err)
	}

	// Set some test state
	lb.RoundRobinIndex = 1
	lb.NextInstanceID = 4
	lb.WeightedSelections = map[string]int{
		"instance-1": 25,
		"instance-2": 35,
	}
	lb.TotalWeight = 60
	lb.InstanceHealth = map[string]float64{
		"instance-1": 0.85,
		"instance-2": 0.92,
	}
	lb.Metrics.TotalOperations = 300
	lb.Metrics.CompletedOps = 285
	lb.Metrics.FailedOps = 15

	// Save state using component method
	err = lb.SaveState()
	if err != nil {
		t.Fatalf("Failed to save load balancer state: %v", err)
	}

	// Create a new load balancer to load the state into
	newLB, err := NewLoadBalancer(config)
	if err != nil {
		t.Fatalf("Failed to create new load balancer: %v", err)
	}

	// Load state using component method
	err = newLB.LoadState("integration-test-lb")
	if err != nil {
		t.Fatalf("Failed to load load balancer state: %v", err)
	}

	// Verify state was restored
	if newLB.ComponentID != "integration-test-lb" {
		t.Errorf("Expected component ID 'integration-test-lb', got %s", newLB.ComponentID)
	}

	if newLB.RoundRobinIndex != 1 {
		t.Errorf("Expected round robin index 1, got %d", newLB.RoundRobinIndex)
	}

	if newLB.NextInstanceID != 4 {
		t.Errorf("Expected next instance ID 4, got %d", newLB.NextInstanceID)
	}

	if newLB.TotalWeight != 60 {
		t.Errorf("Expected total weight 60, got %d", newLB.TotalWeight)
	}

	// Verify weighted selections
	if len(newLB.WeightedSelections) != 2 {
		t.Errorf("Expected 2 weighted selections, got %d", len(newLB.WeightedSelections))
	}

	if newLB.WeightedSelections["instance-1"] != 25 {
		t.Errorf("Expected instance-1 selections 25, got %d", newLB.WeightedSelections["instance-1"])
	}

	if newLB.WeightedSelections["instance-2"] != 35 {
		t.Errorf("Expected instance-2 selections 35, got %d", newLB.WeightedSelections["instance-2"])
	}

	// Verify instance health
	if len(newLB.InstanceHealth) != 2 {
		t.Errorf("Expected 2 instance health entries, got %d", len(newLB.InstanceHealth))
	}

	if newLB.InstanceHealth["instance-1"] != 0.85 {
		t.Errorf("Expected instance-1 health 0.85, got %f", newLB.InstanceHealth["instance-1"])
	}

	if newLB.InstanceHealth["instance-2"] != 0.92 {
		t.Errorf("Expected instance-2 health 0.92, got %f", newLB.InstanceHealth["instance-2"])
	}

	// Verify metrics
	if newLB.Metrics.TotalOperations != 300 {
		t.Errorf("Expected total operations 300, got %d", newLB.Metrics.TotalOperations)
	}

	if newLB.Metrics.CompletedOps != 285 {
		t.Errorf("Expected completed operations 285, got %d", newLB.Metrics.CompletedOps)
	}

	if newLB.Metrics.FailedOps != 15 {
		t.Errorf("Expected failed operations 15, got %d", newLB.Metrics.FailedOps)
	}
}
