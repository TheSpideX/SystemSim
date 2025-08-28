package components

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"
)

// HealthMonitor monitors component health for auto-scaling decisions
type HealthMonitor struct {
	componentHealth map[string]float64
	mutex           sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	ticker          *time.Ticker
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor() *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthMonitor{
		componentHealth: make(map[string]float64),
		ctx:             ctx,
		cancel:          cancel,
		ticker:          time.NewTicker(5 * time.Second),
	}
}

// Start starts the health monitor
func (hm *HealthMonitor) Start() {
	go hm.run()
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop() {
	hm.cancel()
	hm.ticker.Stop()
}

// run monitors component health
func (hm *HealthMonitor) run() {
	for {
		select {
		case <-hm.ticker.C:
			hm.updateHealthMetrics()
		case <-hm.ctx.Done():
			return
		}
	}
}

// updateHealthMetrics simulates health metric updates
func (hm *HealthMonitor) updateHealthMetrics() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	
	// Simulate health fluctuations for educational purposes
	for componentID, currentHealth := range hm.componentHealth {
		// Small random changes (±5%)
		change := (rand.Float64() - 0.5) * 0.1
		newHealth := currentHealth + change
		
		// Clamp to valid range
		if newHealth < 0.0 {
			newHealth = 0.0
		}
		if newHealth > 1.0 {
			newHealth = 1.0
		}
		
		hm.componentHealth[componentID] = newHealth
	}
}

// GetComponentHealth returns the health of a component
func (hm *HealthMonitor) GetComponentHealth(componentID string) float64 {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()
	
	health, exists := hm.componentHealth[componentID]
	if !exists {
		// Initialize with healthy state
		hm.componentHealth[componentID] = 0.9
		return 0.9
	}
	
	return health
}

// SetComponentHealth sets the health of a component (for testing/scenarios)
func (hm *HealthMonitor) SetComponentHealth(componentID string, health float64) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	
	if health < 0.0 {
		health = 0.0
	}
	if health > 1.0 {
		health = 1.0
	}
	
	hm.componentHealth[componentID] = health
}

// MetricsCollector collects load and performance metrics
type MetricsCollector struct {
	componentLoad map[string]float64
	mutex         sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	ticker        *time.Ticker
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())
	return &MetricsCollector{
		componentLoad: make(map[string]float64),
		ctx:           ctx,
		cancel:        cancel,
		ticker:        time.NewTicker(3 * time.Second),
	}
}

// Start starts the metrics collector
func (mc *MetricsCollector) Start() {
	go mc.run()
}

// Stop stops the metrics collector
func (mc *MetricsCollector) Stop() {
	mc.cancel()
	mc.ticker.Stop()
}

// run collects metrics
func (mc *MetricsCollector) run() {
	for {
		select {
		case <-mc.ticker.C:
			mc.updateLoadMetrics()
		case <-mc.ctx.Done():
			return
		}
	}
}

// updateLoadMetrics simulates load metric updates
func (mc *MetricsCollector) updateLoadMetrics() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	// Simulate load fluctuations based on time of day
	hour := time.Now().Hour()
	var baseLoad float64
	
	if hour >= 9 && hour <= 17 {
		baseLoad = 0.7 // Business hours
	} else if hour >= 18 && hour <= 22 {
		baseLoad = 0.5 // Evening
	} else {
		baseLoad = 0.2 // Night
	}
	
	for componentID := range mc.componentLoad {
		// Add random variation
		variation := (rand.Float64() - 0.5) * 0.3
		newLoad := baseLoad + variation
		
		// Clamp to valid range
		if newLoad < 0.0 {
			newLoad = 0.0
		}
		if newLoad > 1.0 {
			newLoad = 1.0
		}
		
		mc.componentLoad[componentID] = newLoad
	}
}

// GetComponentLoad returns the load of a component
func (mc *MetricsCollector) GetComponentLoad(componentID string) float64 {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	load, exists := mc.componentLoad[componentID]
	if !exists {
		// Initialize with moderate load
		mc.componentLoad[componentID] = 0.5
		return 0.5
	}
	
	return load
}

// SetComponentLoad sets the load of a component (for testing/scenarios)
func (mc *MetricsCollector) SetComponentLoad(componentID string, load float64) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	if load < 0.0 {
		load = 0.0
	}
	if load > 1.0 {
		load = 1.0
	}
	
	mc.componentLoad[componentID] = load
}

// ScenarioManager manages educational auto-scaling scenarios
type ScenarioManager struct {
	educationalMode bool
	scenarios       []AutoScalingScenario
	currentScenario int
	ctx             context.Context
	cancel          context.CancelFunc
	ticker          *time.Ticker
}

// AutoScalingScenario represents an educational scenario
type AutoScalingScenario struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Duration    time.Duration `json:"duration"`
	LoadPattern LoadPattern   `json:"load_pattern"`
	HealthEvents []HealthEvent `json:"health_events"`
}

// LoadPattern defines load changes over time
type LoadPattern struct {
	Type      string    `json:"type"` // "spike", "gradual", "oscillating"
	StartLoad float64   `json:"start_load"`
	PeakLoad  float64   `json:"peak_load"`
	Duration  time.Duration `json:"duration"`
}

// HealthEvent defines health changes during scenarios
type HealthEvent struct {
	Timestamp   time.Duration `json:"timestamp"` // Relative to scenario start
	ComponentID string        `json:"component_id"`
	NewHealth   float64       `json:"new_health"`
	Reason      string        `json:"reason"`
}

// NewScenarioManager creates a new scenario manager
func NewScenarioManager(educationalMode bool) *ScenarioManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	sm := &ScenarioManager{
		educationalMode: educationalMode,
		scenarios:       createDefaultScenarios(),
		currentScenario: 0,
		ctx:             ctx,
		cancel:          cancel,
		ticker:          time.NewTicker(30 * time.Second),
	}
	
	return sm
}

// Start starts the scenario manager
func (sm *ScenarioManager) Start() {
	if !sm.educationalMode {
		return
	}
	
	log.Printf("ScenarioManager: Starting educational scenarios")
	go sm.run()
}

// Stop stops the scenario manager
func (sm *ScenarioManager) Stop() {
	sm.cancel()
	sm.ticker.Stop()
}

// run executes educational scenarios
func (sm *ScenarioManager) run() {
	for {
		select {
		case <-sm.ticker.C:
			sm.executeNextScenario()
		case <-sm.ctx.Done():
			return
		}
	}
}

// executeNextScenario executes the next educational scenario
func (sm *ScenarioManager) executeNextScenario() {
	if len(sm.scenarios) == 0 {
		return
	}
	
	scenario := sm.scenarios[sm.currentScenario]
	log.Printf("ScenarioManager: Executing scenario '%s': %s", scenario.Name, scenario.Description)
	
	// Execute scenario (simplified implementation)
	// In a full implementation, this would simulate the load pattern and health events
	
	sm.currentScenario = (sm.currentScenario + 1) % len(sm.scenarios)
}

// createDefaultScenarios creates default educational scenarios
func createDefaultScenarios() []AutoScalingScenario {
	return []AutoScalingScenario{
		{
			Name:        "Traffic Spike",
			Description: "Sudden increase in traffic requiring scale-up",
			Duration:    5 * time.Minute,
			LoadPattern: LoadPattern{
				Type:      "spike",
				StartLoad: 0.3,
				PeakLoad:  0.9,
				Duration:  2 * time.Minute,
			},
		},
		{
			Name:        "Gradual Load Increase",
			Description: "Gradual increase in load over time",
			Duration:    10 * time.Minute,
			LoadPattern: LoadPattern{
				Type:      "gradual",
				StartLoad: 0.2,
				PeakLoad:  0.8,
				Duration:  8 * time.Minute,
			},
		},
		{
			Name:        "Component Failure",
			Description: "Component health degradation requiring scaling",
			Duration:    7 * time.Minute,
			HealthEvents: []HealthEvent{
				{
					Timestamp:   1 * time.Minute,
					ComponentID: "web_component",
					NewHealth:   0.3,
					Reason:      "simulated_failure",
				},
			},
		},
	}
}
