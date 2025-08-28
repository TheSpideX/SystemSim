package components

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// AutoScalingSystem manages configurable auto-scaling for components
type AutoScalingSystem struct {
	// Configuration
	config *AutoScalingSystemConfig

	// Component management
	components map[string]*AutoScalingComponent
	mutex      sync.RWMutex

	// Monitoring
	healthMonitor *HealthMonitor
	metricsCollector *MetricsCollector

	// Educational scenarios
	scenarioManager *ScenarioManager

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
}

// AutoScalingSystemConfig defines system-wide auto-scaling configuration
type AutoScalingSystemConfig struct {
	// Global settings
	CheckInterval       time.Duration `json:"check_interval"`
	DefaultCooldown     time.Duration `json:"default_cooldown"`
	MaxComponentsTotal  int           `json:"max_components_total"`
	
	// Health thresholds
	HealthyThreshold    float64       `json:"healthy_threshold"`
	UnhealthyThreshold  float64       `json:"unhealthy_threshold"`
	
	// Educational mode
	EducationalMode     bool          `json:"educational_mode"`
	ScenarioEnabled     bool          `json:"scenario_enabled"`
}

// AutoScalingComponent represents a component with auto-scaling configuration
type AutoScalingComponent struct {
	ComponentID   string                    `json:"component_id"`
	Config        *AutoScalingConfig        `json:"config"`
	LoadBalancer  ComponentLoadBalancerInterface `json:"-"`
	
	// State tracking
	CurrentInstances  int           `json:"current_instances"`
	LastScaleAction   time.Time     `json:"last_scale_action"`
	ScaleHistory      []ScaleEvent  `json:"scale_history"`
	
	// Metrics
	Metrics          *AutoScalingMetrics `json:"metrics"`
}

// ScaleEvent records scaling actions for educational purposes
type ScaleEvent struct {
	Timestamp   time.Time     `json:"timestamp"`
	Action      ScaleAction   `json:"action"`
	Reason      string        `json:"reason"`
	FromCount   int           `json:"from_count"`
	ToCount     int           `json:"to_count"`
	Trigger     ScaleTrigger  `json:"trigger"`
}

// ScaleAction defines the type of scaling action
type ScaleAction string

const (
	ScaleUp   ScaleAction = "scale_up"
	ScaleDown ScaleAction = "scale_down"
	NoAction  ScaleAction = "no_action"
)

// ScaleTrigger defines what triggered the scaling action
type ScaleTrigger string

const (
	TriggerLoad     ScaleTrigger = "load"
	TriggerHealth   ScaleTrigger = "health"
	TriggerManual   ScaleTrigger = "manual"
	TriggerScenario ScaleTrigger = "scenario"
)

// AutoScalingMetrics tracks auto-scaling performance
type AutoScalingMetrics struct {
	TotalScaleUps     int64         `json:"total_scale_ups"`
	TotalScaleDowns   int64         `json:"total_scale_downs"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	EfficiencyScore   float64       `json:"efficiency_score"`
	CostSavings       float64       `json:"cost_savings"`
}

// NewAutoScalingSystem creates a new auto-scaling system
func NewAutoScalingSystem(config *AutoScalingSystemConfig) *AutoScalingSystem {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &AutoScalingSystem{
		config:           config,
		components:       make(map[string]*AutoScalingComponent),
		healthMonitor:    NewHealthMonitor(),
		metricsCollector: NewMetricsCollector(),
		scenarioManager:  NewScenarioManager(config.EducationalMode),
		ctx:              ctx,
		cancel:           cancel,
		ticker:           time.NewTicker(config.CheckInterval),
	}
}

// Start starts the auto-scaling system
func (ass *AutoScalingSystem) Start() error {
	log.Printf("AutoScalingSystem: Starting with check interval %v", ass.config.CheckInterval)
	
	go ass.run()
	go ass.healthMonitor.Start()
	go ass.metricsCollector.Start()
	
	if ass.config.ScenarioEnabled {
		go ass.scenarioManager.Start()
	}
	
	return nil
}

// Stop stops the auto-scaling system
func (ass *AutoScalingSystem) Stop() error {
	log.Printf("AutoScalingSystem: Stopping")
	
	ass.cancel()
	ass.ticker.Stop()
	ass.healthMonitor.Stop()
	ass.metricsCollector.Stop()
	ass.scenarioManager.Stop()
	
	return nil
}

// RegisterComponent registers a component for auto-scaling
func (ass *AutoScalingSystem) RegisterComponent(componentID string, config *AutoScalingConfig, lb ComponentLoadBalancerInterface) error {
	ass.mutex.Lock()
	defer ass.mutex.Unlock()
	
	if componentID == "" {
		return fmt.Errorf("component ID cannot be empty")
	}
	
	if config == nil {
		return fmt.Errorf("auto-scaling config cannot be nil")
	}
	
	component := &AutoScalingComponent{
		ComponentID:      componentID,
		Config:           config,
		LoadBalancer:     lb,
		CurrentInstances: config.MinInstances,
		LastScaleAction:  time.Now(),
		ScaleHistory:     make([]ScaleEvent, 0),
		Metrics: &AutoScalingMetrics{
			TotalScaleUps:   0,
			TotalScaleDowns: 0,
			EfficiencyScore: 1.0,
			CostSavings:     0.0,
		},
	}
	
	ass.components[componentID] = component
	log.Printf("AutoScalingSystem: Registered component %s for auto-scaling", componentID)
	
	return nil
}

// run is the main auto-scaling loop
func (ass *AutoScalingSystem) run() {
	for {
		select {
		case <-ass.ticker.C:
			ass.performScalingCheck()
			
		case <-ass.ctx.Done():
			log.Printf("AutoScalingSystem: Main loop stopping")
			return
		}
	}
}

// performScalingCheck checks all components and performs scaling actions
func (ass *AutoScalingSystem) performScalingCheck() {
	ass.mutex.RLock()
	components := make([]*AutoScalingComponent, 0, len(ass.components))
	for _, component := range ass.components {
		components = append(components, component)
	}
	ass.mutex.RUnlock()
	
	for _, component := range components {
		if component.Config.Enabled {
			ass.checkComponentScaling(component)
		}
	}
}

// checkComponentScaling checks if a component needs scaling
func (ass *AutoScalingSystem) checkComponentScaling(component *AutoScalingComponent) {
	// Check cooldown period
	if time.Since(component.LastScaleAction) < ass.getCooldownPeriod(component) {
		return
	}
	
	// Get current metrics
	health := ass.healthMonitor.GetComponentHealth(component.ComponentID)
	load := ass.metricsCollector.GetComponentLoad(component.ComponentID)
	
	// Determine scaling action
	action, reason := ass.determineScalingAction(component, health, load)
	
	if action != NoAction {
		ass.executeScalingAction(component, action, reason, TriggerLoad)
	}
}

// determineScalingAction determines what scaling action to take
func (ass *AutoScalingSystem) determineScalingAction(component *AutoScalingComponent, health, load float64) (ScaleAction, string) {
	config := component.Config
	
	// Health-based scaling (priority)
	if health < ass.config.UnhealthyThreshold && component.CurrentInstances < config.MaxInstances {
		return ScaleUp, fmt.Sprintf("health_low_%.2f", health)
	}
	
	// Load-based scaling
	if load > config.ScaleUpThreshold && component.CurrentInstances < config.MaxInstances {
		return ScaleUp, fmt.Sprintf("load_high_%.2f", load)
	}
	
	if load < config.ScaleDownThreshold && component.CurrentInstances > config.MinInstances {
		return ScaleDown, fmt.Sprintf("load_low_%.2f", load)
	}
	
	return NoAction, "no_action_needed"
}

// executeScalingAction executes a scaling action
func (ass *AutoScalingSystem) executeScalingAction(component *AutoScalingComponent, action ScaleAction, reason string, trigger ScaleTrigger) {
	oldCount := component.CurrentInstances
	var newCount int
	
	switch action {
	case ScaleUp:
		newCount = oldCount + 1
		if newCount > component.Config.MaxInstances {
			newCount = component.Config.MaxInstances
		}
		component.Metrics.TotalScaleUps++
		
	case ScaleDown:
		newCount = oldCount - 1
		if newCount < component.Config.MinInstances {
			newCount = component.Config.MinInstances
		}
		component.Metrics.TotalScaleDowns++
	}
	
	if newCount != oldCount {
		// Record scale event
		event := ScaleEvent{
			Timestamp: time.Now(),
			Action:    action,
			Reason:    reason,
			FromCount: oldCount,
			ToCount:   newCount,
			Trigger:   trigger,
		}
		
		component.ScaleHistory = append(component.ScaleHistory, event)
		component.CurrentInstances = newCount
		component.LastScaleAction = time.Now()
		
		log.Printf("AutoScalingSystem: %s component %s from %d to %d instances (reason: %s)", 
			action, component.ComponentID, oldCount, newCount, reason)
		
		// Update efficiency metrics
		ass.updateEfficiencyMetrics(component, action)
	}
}

// getCooldownPeriod returns the cooldown period for a component
func (ass *AutoScalingSystem) getCooldownPeriod(component *AutoScalingComponent) time.Duration {
	if component.Config.CooldownPeriod != "" {
		if duration, err := time.ParseDuration(component.Config.CooldownPeriod); err == nil {
			return duration
		}
	}
	return ass.config.DefaultCooldown
}

// updateEfficiencyMetrics updates efficiency metrics for educational purposes
func (ass *AutoScalingSystem) updateEfficiencyMetrics(component *AutoScalingComponent, action ScaleAction) {
	// Calculate efficiency score based on scaling frequency and resource utilization
	totalActions := component.Metrics.TotalScaleUps + component.Metrics.TotalScaleDowns
	
	if totalActions > 0 {
		// Lower score for frequent scaling (indicates instability)
		frequencyPenalty := float64(totalActions) * 0.01
		component.Metrics.EfficiencyScore = 1.0 - frequencyPenalty
		
		if component.Metrics.EfficiencyScore < 0.1 {
			component.Metrics.EfficiencyScore = 0.1
		}
	}
	
	// Calculate cost savings (educational metric)
	if action == ScaleDown {
		component.Metrics.CostSavings += 10.0 // $10 per instance hour saved
	}
}
