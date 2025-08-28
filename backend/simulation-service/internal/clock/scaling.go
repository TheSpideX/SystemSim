package clock

import (
	"fmt"
	"math"
	"time"
)

// ScalingStrategy represents different approaches to scaling
type ScalingStrategy string

const (
	// Manual scaling - user controls scaling factor
	ManualScaling ScalingStrategy = "manual"
	
	// Automatic scaling - system adjusts based on performance
	AutomaticScaling ScalingStrategy = "automatic"
	
	// Adaptive scaling - learns optimal settings over time
	AdaptiveScaling ScalingStrategy = "adaptive"
)

// ScalingConfiguration holds scaling-related settings
type ScalingConfiguration struct {
	Strategy           ScalingStrategy `json:"strategy"`
	MinScalingFactor   float64         `json:"min_scaling_factor"`
	MaxScalingFactor   float64         `json:"max_scaling_factor"`
	TargetUtilization  float64         `json:"target_utilization"`
	AdjustmentRate     float64         `json:"adjustment_rate"`
	StabilityThreshold float64         `json:"stability_threshold"`
}

// DefaultScalingConfiguration returns sensible default scaling settings
func DefaultScalingConfiguration() ScalingConfiguration {
	return ScalingConfiguration{
		Strategy:           AutomaticScaling,
		MinScalingFactor:   0.01,  // Minimum 1% of real-time
		MaxScalingFactor:   10.0,  // Maximum 10x real-time
		TargetUtilization:  0.7,   // Target 70% tick utilization
		AdjustmentRate:     0.1,   // Adjust by 10% each time
		StabilityThreshold: 0.05,  // 5% change threshold
	}
}

// CalculateOptimalScaling calculates optimal scaling based on system complexity
func CalculateOptimalScaling(componentCount int, targetPerformance float64) float64 {
	// Base scaling on component count and target performance
	// More components = need slower scaling for stability
	
	baseScaling := 1.0
	
	// Adjust for component count
	switch {
	case componentCount <= 10:
		baseScaling = 2.0 // Can run faster with few components
	case componentCount <= 100:
		baseScaling = 1.0 // Real-time for moderate systems
	case componentCount <= 1000:
		baseScaling = 0.5 // Half-speed for large systems
	default:
		baseScaling = 0.1 // Very slow for massive systems
	}
	
	// Adjust for target performance
	return baseScaling * targetPerformance
}

// CalculateScalingForComplexity determines scaling based on system complexity
func CalculateScalingForComplexity(complexity SystemComplexity) float64 {
	switch complexity {
	case SimpleSystem:
		return 2.0 // Can run 2x real-time
	case ModerateSystem:
		return 1.0 // Real-time
	case ComplexSystem:
		return 0.5 // Half real-time
	case VeryComplexSystem:
		return 0.2 // One-fifth real-time
	case MassiveSystem:
		return 0.1 // One-tenth real-time
	default:
		return 1.0 // Default to real-time
	}
}

// SystemComplexity represents different levels of system complexity
type SystemComplexity string

const (
	SimpleSystem      SystemComplexity = "simple"       // 1-10 components
	ModerateSystem    SystemComplexity = "moderate"     // 11-100 components
	ComplexSystem     SystemComplexity = "complex"      // 101-1000 components
	VeryComplexSystem SystemComplexity = "very_complex" // 1001-10000 components
	MassiveSystem     SystemComplexity = "massive"      // 10000+ components
)

// DetermineSystemComplexity categorizes system complexity based on component count
func DetermineSystemComplexity(componentCount int) SystemComplexity {
	switch {
	case componentCount <= 10:
		return SimpleSystem
	case componentCount <= 100:
		return ModerateSystem
	case componentCount <= 1000:
		return ComplexSystem
	case componentCount <= 10000:
		return VeryComplexSystem
	default:
		return MassiveSystem
	}
}

// AutoScaler manages automatic scaling adjustments
type AutoScaler struct {
	config          ScalingConfiguration
	coordinator     *GlobalTickCoordinator
	lastAdjustment  time.Time
	adjustmentCount int
	stabilityWindow []float64 // Recent utilization values for stability analysis
}

// NewAutoScaler creates a new automatic scaler
func NewAutoScaler(coordinator *GlobalTickCoordinator, config ScalingConfiguration) *AutoScaler {
	return &AutoScaler{
		config:          config,
		coordinator:     coordinator,
		lastAdjustment:  time.Now(),
		adjustmentCount: 0,
		stabilityWindow: make([]float64, 0, 10), // Keep last 10 measurements
	}
}

// Update analyzes current performance and adjusts scaling if needed
func (as *AutoScaler) Update() error {
	if as.config.Strategy != AutomaticScaling {
		return nil // Only process automatic scaling
	}
	
	metrics := as.coordinator.GetPerformanceMetrics()
	
	// Add current utilization to stability window
	as.stabilityWindow = append(as.stabilityWindow, metrics.TickUtilization)
	if len(as.stabilityWindow) > 10 {
		as.stabilityWindow = as.stabilityWindow[1:]
	}
	
	// Check if we should adjust (don't adjust too frequently)
	if time.Since(as.lastAdjustment) < 5*time.Second {
		return nil
	}
	
	// Check if system is stable enough to adjust
	if !as.isSystemStable() {
		return nil
	}
	
	// Calculate desired scaling adjustment
	currentUtilization := metrics.TickUtilization
	targetUtilization := as.config.TargetUtilization
	
	// If utilization is within acceptable range, don't adjust
	if math.Abs(currentUtilization-targetUtilization) < as.config.StabilityThreshold {
		return nil
	}
	
	// Calculate new scaling factor
	currentScaling := as.coordinator.ScalingFactor
	var newScaling float64
	
	if currentUtilization > targetUtilization {
		// System is overloaded, slow down
		adjustment := 1.0 - as.config.AdjustmentRate
		newScaling = currentScaling * adjustment
	} else {
		// System has capacity, speed up
		adjustment := 1.0 + as.config.AdjustmentRate
		newScaling = currentScaling * adjustment
	}
	
	// Clamp to configured limits
	newScaling = math.Max(as.config.MinScalingFactor, math.Min(as.config.MaxScalingFactor, newScaling))
	
	// Apply the new scaling
	err := as.coordinator.SetScalingFactor(newScaling)
	if err != nil {
		return fmt.Errorf("failed to apply scaling adjustment: %w", err)
	}
	
	as.lastAdjustment = time.Now()
	as.adjustmentCount++
	
	return nil
}

// isSystemStable checks if the system utilization is stable enough for adjustment
func (as *AutoScaler) isSystemStable() bool {
	if len(as.stabilityWindow) < 5 {
		return false // Need at least 5 measurements
	}
	
	// Calculate variance in recent utilization
	mean := 0.0
	for _, util := range as.stabilityWindow {
		mean += util
	}
	mean /= float64(len(as.stabilityWindow))
	
	variance := 0.0
	for _, util := range as.stabilityWindow {
		variance += math.Pow(util-mean, 2)
	}
	variance /= float64(len(as.stabilityWindow))
	
	standardDeviation := math.Sqrt(variance)
	
	// System is stable if standard deviation is low
	return standardDeviation < 0.1 // Less than 10% variation
}

// GetScalingStatus returns current scaling status and recommendations
func (as *AutoScaler) GetScalingStatus() ScalingStatus {
	metrics := as.coordinator.GetPerformanceMetrics()
	complexity := DetermineSystemComplexity(metrics.ComponentCount)
	optimalScaling := CalculateScalingForComplexity(complexity)
	
	return ScalingStatus{
		CurrentScaling:    metrics.ScalingFactor,
		OptimalScaling:    optimalScaling,
		SystemComplexity:  complexity,
		TargetUtilization: as.config.TargetUtilization,
		CurrentUtilization: metrics.TickUtilization,
		AdjustmentCount:   as.adjustmentCount,
		LastAdjustment:    as.lastAdjustment,
		IsStable:          as.isSystemStable(),
		Strategy:          as.config.Strategy,
	}
}

// ScalingStatus provides comprehensive scaling information
type ScalingStatus struct {
	CurrentScaling     float64           `json:"current_scaling"`
	OptimalScaling     float64           `json:"optimal_scaling"`
	SystemComplexity   SystemComplexity  `json:"system_complexity"`
	TargetUtilization  float64           `json:"target_utilization"`
	CurrentUtilization float64           `json:"current_utilization"`
	AdjustmentCount    int               `json:"adjustment_count"`
	LastAdjustment     time.Time         `json:"last_adjustment"`
	IsStable           bool              `json:"is_stable"`
	Strategy           ScalingStrategy   `json:"strategy"`
}

// ScalingPreset represents predefined scaling configurations
type ScalingPreset struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Scaling     float64 `json:"scaling"`
	UseCase     string  `json:"use_case"`
}

// GetScalingPresets returns common scaling presets
func GetScalingPresets() []ScalingPreset {
	return []ScalingPreset{
		{
			Name:        "Educational",
			Description: "Slow motion for learning and analysis",
			Scaling:     0.1,
			UseCase:     "Students learning system behavior",
		},
		{
			Name:        "Development",
			Description: "Moderate speed for development and testing",
			Scaling:     0.5,
			UseCase:     "Development and debugging",
		},
		{
			Name:        "Real-time",
			Description: "Real-time simulation speed",
			Scaling:     1.0,
			UseCase:     "Production-like behavior analysis",
		},
		{
			Name:        "Fast Testing",
			Description: "Accelerated for quick testing",
			Scaling:     2.0,
			UseCase:     "Rapid testing and validation",
		},
		{
			Name:        "Stress Testing",
			Description: "Very fast for stress testing",
			Scaling:     5.0,
			UseCase:     "Load testing and capacity planning",
		},
	}
}

// ApplyScalingPreset applies a predefined scaling preset
func (gtc *GlobalTickCoordinator) ApplyScalingPreset(presetName string) error {
	presets := GetScalingPresets()
	
	for _, preset := range presets {
		if preset.Name == presetName {
			return gtc.SetScalingFactor(preset.Scaling)
		}
	}
	
	return fmt.Errorf("scaling preset '%s' not found", presetName)
}
