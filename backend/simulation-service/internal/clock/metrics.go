package clock

import (
	"time"
)

// PerformanceMetrics holds simulation performance data
type PerformanceMetrics struct {
	CurrentTick       int64         `json:"current_tick"`
	TotalTicks        int64         `json:"total_ticks"`
	TicksPerSecond    float64       `json:"ticks_per_second"`
	AverageTickTime   time.Duration `json:"average_tick_time"`
	MaxTickTime       time.Duration `json:"max_tick_time"`
	SimulationTime    time.Duration `json:"simulation_time"`
	RealTimeElapsed   time.Duration `json:"real_time_elapsed"`
	ScalingFactor     float64       `json:"scaling_factor"`
	ComponentCount    int           `json:"component_count"`
	EfficiencyRatio   float64       `json:"efficiency_ratio"`   // Simulation time / Real time
	TickUtilization   float64       `json:"tick_utilization"`   // Average tick time / Tick duration
}

// GetPerformanceMetrics returns current performance statistics
func (gtc *GlobalTickCoordinator) GetPerformanceMetrics() PerformanceMetrics {
	gtc.mutex.RLock()
	defer gtc.mutex.RUnlock()
	
	simulationTime := time.Duration(gtc.CurrentTick) * gtc.TickDuration
	realTimeElapsed := time.Since(gtc.StartTime)
	
	// Calculate efficiency ratio (how much simulation time vs real time)
	efficiencyRatio := 0.0
	if realTimeElapsed > 0 {
		efficiencyRatio = float64(simulationTime) / float64(realTimeElapsed)
	}
	
	// Calculate tick utilization (how much of each tick duration is used)
	tickUtilization := 0.0
	if gtc.TickDuration > 0 {
		tickUtilization = float64(gtc.AverageTickTime) / float64(gtc.TickDuration)
	}
	
	return PerformanceMetrics{
		CurrentTick:       gtc.CurrentTick,
		TotalTicks:        gtc.TotalTicks,
		TicksPerSecond:    gtc.TicksPerSecond,
		AverageTickTime:   gtc.AverageTickTime,
		MaxTickTime:       gtc.MaxTickTime,
		SimulationTime:    simulationTime,
		RealTimeElapsed:   realTimeElapsed,
		ScalingFactor:     gtc.ScalingFactor,
		ComponentCount:    len(gtc.Components),
		EfficiencyRatio:   efficiencyRatio,
		TickUtilization:   tickUtilization,
	}
}

// GetHealthStatus returns the overall health status of the simulation
func (gtc *GlobalTickCoordinator) GetHealthStatus() HealthStatus {
	metrics := gtc.GetPerformanceMetrics()
	
	// Determine health based on performance metrics
	health := HealthyStatus
	
	// Check tick utilization (if > 80%, we're struggling)
	if metrics.TickUtilization > 0.8 {
		health = DegradedStatus
	}
	
	// Check if we're falling behind real-time significantly
	if metrics.EfficiencyRatio < 0.1 && metrics.ScalingFactor >= 1.0 {
		health = UnhealthyStatus
	}
	
	// Check component health
	gtc.ComponentsMux.RLock()
	unhealthyComponents := 0
	for _, component := range gtc.Components {
		if !component.IsHealthy() {
			unhealthyComponents++
		}
	}
	gtc.ComponentsMux.RUnlock()
	
	// If more than 50% of components are unhealthy
	if len(gtc.Components) > 0 && float64(unhealthyComponents)/float64(len(gtc.Components)) > 0.5 {
		health = UnhealthyStatus
	}
	
	return HealthStatus{
		Status:              health,
		TickUtilization:     metrics.TickUtilization,
		EfficiencyRatio:     metrics.EfficiencyRatio,
		UnhealthyComponents: unhealthyComponents,
		TotalComponents:     len(gtc.Components),
		LastUpdated:         time.Now(),
	}
}

// HealthStatusType represents the health status of the simulation
type HealthStatusType string

const (
	HealthyStatus   HealthStatusType = "healthy"
	DegradedStatus  HealthStatusType = "degraded"
	UnhealthyStatus HealthStatusType = "unhealthy"
)

// HealthStatus represents the overall health of the simulation
type HealthStatus struct {
	Status              HealthStatusType `json:"status"`
	TickUtilization     float64          `json:"tick_utilization"`
	EfficiencyRatio     float64          `json:"efficiency_ratio"`
	UnhealthyComponents int              `json:"unhealthy_components"`
	TotalComponents     int              `json:"total_components"`
	LastUpdated         time.Time        `json:"last_updated"`
}

// ScalingRecommendation provides recommendations for scaling adjustments
type ScalingRecommendation struct {
	RecommendedScaling float64 `json:"recommended_scaling"`
	Reason             string  `json:"reason"`
	CurrentUtilization float64 `json:"current_utilization"`
	TargetUtilization  float64 `json:"target_utilization"`
}

// GetScalingRecommendation analyzes performance and suggests scaling adjustments
func (gtc *GlobalTickCoordinator) GetScalingRecommendation() ScalingRecommendation {
	metrics := gtc.GetPerformanceMetrics()
	
	targetUtilization := 0.7 // Target 70% tick utilization
	currentUtilization := metrics.TickUtilization
	
	// If utilization is too high, recommend slower scaling
	if currentUtilization > 0.9 {
		recommendedScaling := gtc.ScalingFactor * 0.5 // Slow down significantly
		return ScalingRecommendation{
			RecommendedScaling: recommendedScaling,
			Reason:             "High tick utilization detected - reducing simulation speed",
			CurrentUtilization: currentUtilization,
			TargetUtilization:  targetUtilization,
		}
	}
	
	if currentUtilization > 0.8 {
		recommendedScaling := gtc.ScalingFactor * 0.8 // Slow down moderately
		return ScalingRecommendation{
			RecommendedScaling: recommendedScaling,
			Reason:             "Moderate tick utilization - slightly reducing simulation speed",
			CurrentUtilization: currentUtilization,
			TargetUtilization:  targetUtilization,
		}
	}
	
	// If utilization is too low, recommend faster scaling
	if currentUtilization < 0.3 {
		recommendedScaling := gtc.ScalingFactor * 1.5 // Speed up
		return ScalingRecommendation{
			RecommendedScaling: recommendedScaling,
			Reason:             "Low tick utilization - can increase simulation speed",
			CurrentUtilization: currentUtilization,
			TargetUtilization:  targetUtilization,
		}
	}
	
	// Current scaling is fine
	return ScalingRecommendation{
		RecommendedScaling: gtc.ScalingFactor,
		Reason:             "Current scaling is optimal",
		CurrentUtilization: currentUtilization,
		TargetUtilization:  targetUtilization,
	}
}

// ApplyScalingRecommendation automatically applies the recommended scaling
func (gtc *GlobalTickCoordinator) ApplyScalingRecommendation() error {
	recommendation := gtc.GetScalingRecommendation()
	
	// Only apply if the change is significant (> 10% difference)
	currentScaling := gtc.ScalingFactor
	if abs(recommendation.RecommendedScaling-currentScaling)/currentScaling < 0.1 {
		return nil // No significant change needed
	}
	
	return gtc.SetScalingFactor(recommendation.RecommendedScaling)
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetDetailedStatus returns comprehensive status information
func (gtc *GlobalTickCoordinator) GetDetailedStatus() DetailedStatus {
	return DetailedStatus{
		Performance: gtc.GetPerformanceMetrics(),
		Health:      gtc.GetHealthStatus(),
		Scaling:     gtc.GetScalingRecommendation(),
		Running:     gtc.IsRunning(),
		Paused:      gtc.IsPaused(),
	}
}

// DetailedStatus combines all status information
type DetailedStatus struct {
	Performance PerformanceMetrics    `json:"performance"`
	Health      HealthStatus          `json:"health"`
	Scaling     ScalingRecommendation `json:"scaling"`
	Running     bool                  `json:"running"`
	Paused      bool                  `json:"paused"`
}
