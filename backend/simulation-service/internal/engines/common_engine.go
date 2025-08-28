package engines

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// CommonEngine provides shared functionality for all engines
type CommonEngine struct {
	// Identity and configuration
	ID           string        `json:"id"`
	Type         EngineType    `json:"type"`
	TickDuration time.Duration `json:"tick_duration"`
	Profile      *EngineProfile `json:"profile"`

	// Complexity level for processing decisions
	ComplexityLevel CPUComplexityLevel `json:"complexity_level"`

	// Queue management (engines handle their own queues)
	Queue    []*QueuedOperation `json:"queue"`
	QueueCap int               `json:"queue_capacity"`
	mutex    sync.RWMutex      // Protects queue operations

	// Health and monitoring
	Health *HealthMetrics `json:"health"`

	// State tracking
	CurrentTick     int64 `json:"current_tick"`
	TotalOperations int64 `json:"total_operations"`
	CompletedOps    int64 `json:"completed_operations"`
	FailedOps       int64 `json:"failed_operations"`

	// Performance modeling
	LoadDegradation LoadDegradationCurve `json:"load_degradation"`
	Variance        PerformanceVariance  `json:"variance"`

	// Statistical convergence modeling
	ConvergenceState *ConvergenceState `json:"convergence_state"`

	// Performance history for convergence
	OperationHistory []time.Duration `json:"operation_history"`
	LoadHistory      []float64       `json:"load_history"`
}

// NewCommonEngine creates a new common engine foundation
func NewCommonEngine(engineType EngineType, queueCapacity int) *CommonEngine {
	return &CommonEngine{
		ID:           fmt.Sprintf("%s-%d", engineType.String(), time.Now().UnixNano()),
		Type:         engineType,
		TickDuration: 1 * time.Millisecond, // Realistic 1ms tick for scalability
		QueueCap:     queueCapacity,
		Queue:        make([]*QueuedOperation, 0, queueCapacity),
		Health: &HealthMetrics{
			Score:            1.0,
			Utilization:      0.0,
			QueueUtilization: 0.0,
			ErrorRate:        0.0,
			AverageLatency:   0.0,
			ThroughputOps:    0.0,
			LastUpdated:      0,
		},
		LoadDegradation: LoadDegradationCurve{
			OptimalThreshold:  0.70,
			WarningThreshold:  0.85,
			CriticalThreshold: 0.95,
			OptimalFactor:     1.0,
			WarningFactor:     2.0,
			CriticalFactor:    5.0, // Default fallback - should be overridden by profile
		},
		Variance: PerformanceVariance{
			BaseVariance:   0.15,  // Increased from 0.05 to 0.15 (15% base variance)
			LoadMultiplier: 3.0,   // Increased from 1.5 to 3.0 (more load sensitivity)
			ScaleReduction: 0.05,  // Reduced from 0.1 to 0.05 (slower convergence)
		},
		ConvergenceState: &ConvergenceState{
			Models:         make(map[string]*StatisticalModel),
			OperationCount: 0,
			DataProcessed:  0,
			StartTick:      0,
			ConvergedTick:  -1,
		},
		OperationHistory: make([]time.Duration, 0, 1000),
		LoadHistory:      make([]float64, 0, 1000),
	}
}

// QueueOperation adds an operation to the queue
func (ce *CommonEngine) QueueOperation(op *Operation) error {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	if len(ce.Queue) >= ce.QueueCap {
		return fmt.Errorf("queue is full (capacity: %d)", ce.QueueCap)
	}

	queuedOp := &QueuedOperation{
		Operation: op,
		QueuedAt:  ce.CurrentTick,
	}

	ce.Queue = append(ce.Queue, queuedOp)
	return nil
}

// DequeueOperation removes and returns the next operation from the queue
func (ce *CommonEngine) DequeueOperation() *QueuedOperation {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	if len(ce.Queue) == 0 {
		return nil
	}

	op := ce.Queue[0]
	ce.Queue = ce.Queue[1:]
	return op
}

// GetQueueLength returns the current queue length
func (ce *CommonEngine) GetQueueLength() int {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()
	return len(ce.Queue)
}

// GetQueueCapacity returns the queue capacity
func (ce *CommonEngine) GetQueueCapacity() int {
	return ce.QueueCap
}

// GetEngineType returns the engine type
func (ce *CommonEngine) GetEngineType() EngineType {
	return ce.Type
}

// GetEngineID returns the engine ID
func (ce *CommonEngine) GetEngineID() string {
	return ce.ID
}

// SetTickDuration sets the tick duration
func (ce *CommonEngine) SetTickDuration(duration time.Duration) {
	ce.TickDuration = duration
}

// GetTickDuration returns the tick duration
func (ce *CommonEngine) GetTickDuration() time.Duration {
	return ce.TickDuration
}

// LoadProfile loads an engine profile
func (ce *CommonEngine) LoadProfile(profile *EngineProfile) error {
	if profile.Type != ce.Type {
		return fmt.Errorf("profile type %s does not match engine type %s", profile.Type, ce.Type)
	}
	ce.Profile = profile
	ce.initializeFromProfile()
	return nil
}

// GetProfile returns the current profile
func (ce *CommonEngine) GetProfile() *EngineProfile {
	return ce.Profile
}

// SetComplexityLevel sets the complexity level for processing decisions
func (ce *CommonEngine) SetComplexityLevel(level CPUComplexityLevel) error {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()
	ce.ComplexityLevel = level
	return nil
}

// GetComplexityLevel returns the current complexity level
func (ce *CommonEngine) GetComplexityLevel() CPUComplexityLevel {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()
	return ce.ComplexityLevel
}

// GetHealth returns current health metrics
func (ce *CommonEngine) GetHealth() *HealthMetrics {
	return ce.Health
}

// GetUtilization returns current utilization (0.0 to 1.0)
func (ce *CommonEngine) GetUtilization() float64 {
	return ce.Health.Utilization
}

// Reset resets the engine to initial state
func (ce *CommonEngine) Reset() {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	ce.Queue = ce.Queue[:0]
	ce.CurrentTick = 0
	ce.TotalOperations = 0
	ce.CompletedOps = 0
	ce.FailedOps = 0
	ce.OperationHistory = ce.OperationHistory[:0]
	ce.LoadHistory = ce.LoadHistory[:0]

	// Reset health
	ce.Health.Score = 1.0
	ce.Health.Utilization = 0.0
	ce.Health.QueueUtilization = 0.0
	ce.Health.ErrorRate = 0.0
	ce.Health.AverageLatency = 0.0
	ce.Health.ThroughputOps = 0.0

	// Reset convergence state
	ce.ConvergenceState.OperationCount = 0
	ce.ConvergenceState.DataProcessed = 0
	ce.ConvergenceState.StartTick = 0
	ce.ConvergenceState.ConvergedTick = -1
	for _, model := range ce.ConvergenceState.Models {
		model.CurrentValue = model.ConvergencePoint
		model.IsConverged = false
	}
}

// GetCurrentState returns current engine state
func (ce *CommonEngine) GetCurrentState() map[string]interface{} {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()

	return map[string]interface{}{
		"id":                ce.ID,
		"type":              ce.Type.String(),
		"current_tick":      ce.CurrentTick,
		"queue_length":      len(ce.Queue),
		"queue_capacity":    ce.QueueCap,
		"total_operations":  ce.TotalOperations,
		"completed_ops":     ce.CompletedOps,
		"failed_ops":        ce.FailedOps,
		"health":            ce.Health,
		"convergence_state": ce.ConvergenceState,
	}
}

// DurationToTicks converts a duration to number of ticks
func (ce *CommonEngine) DurationToTicks(duration time.Duration) int64 {
	return int64(duration / ce.TickDuration)
}

// TicksToDuration converts number of ticks to duration
func (ce *CommonEngine) TicksToDuration(ticks int64) time.Duration {
	return time.Duration(ticks) * ce.TickDuration
}

// initializeFromProfile initializes engine settings from profile
func (ce *CommonEngine) initializeFromProfile() {
	if ce.Profile == nil {
		return
	}

	// Initialize load degradation curve from profile
	if loadCurves, ok := ce.Profile.LoadCurves["default"]; ok {
		if curve, ok := loadCurves.(map[string]interface{}); ok {
			if val, ok := curve["optimal_threshold"].(float64); ok {
				ce.LoadDegradation.OptimalThreshold = val
			}
			if val, ok := curve["warning_threshold"].(float64); ok {
				ce.LoadDegradation.WarningThreshold = val
			}
			if val, ok := curve["critical_threshold"].(float64); ok {
				ce.LoadDegradation.CriticalThreshold = val
			}
			if val, ok := curve["optimal_factor"].(float64); ok {
				ce.LoadDegradation.OptimalFactor = val
			}
			if val, ok := curve["warning_factor"].(float64); ok {
				ce.LoadDegradation.WarningFactor = val
			}
			if val, ok := curve["critical_factor"].(float64); ok {
				ce.LoadDegradation.CriticalFactor = val
			}
		}
	}
}

// ApplyCommonPerformanceFactors applies shared performance factors
func (ce *CommonEngine) ApplyCommonPerformanceFactors(baseTime time.Duration, utilization float64) time.Duration {
	// Apply load degradation
	loadFactor := ce.calculateLoadDegradationFactor(utilization)
	
	// Apply queue penalty
	queueFactor := ce.calculateQueuePenaltyFactor()
	
	// Apply health penalty
	healthFactor := ce.calculateHealthPenaltyFactor()
	
	// Apply realistic variance
	varianceFactor := ce.calculateVarianceFactor(utilization)
	
	// Combine all factors
	totalFactor := loadFactor * queueFactor * healthFactor * varianceFactor
	
	return time.Duration(float64(baseTime) * totalFactor)
}

// calculateLoadDegradationFactor calculates performance degradation based on load
func (ce *CommonEngine) calculateLoadDegradationFactor(utilization float64) float64 {
	switch {
	case utilization <= ce.LoadDegradation.OptimalThreshold:
		return ce.LoadDegradation.OptimalFactor
	case utilization <= ce.LoadDegradation.WarningThreshold:
		// Linear interpolation between optimal and warning
		ratio := (utilization - ce.LoadDegradation.OptimalThreshold) / 
			(ce.LoadDegradation.WarningThreshold - ce.LoadDegradation.OptimalThreshold)
		return ce.LoadDegradation.OptimalFactor + 
			ratio*(ce.LoadDegradation.WarningFactor-ce.LoadDegradation.OptimalFactor)
	case utilization <= ce.LoadDegradation.CriticalThreshold:
		// Linear interpolation between warning and critical
		ratio := (utilization - ce.LoadDegradation.WarningThreshold) / 
			(ce.LoadDegradation.CriticalThreshold - ce.LoadDegradation.WarningThreshold)
		return ce.LoadDegradation.WarningFactor + 
			ratio*(ce.LoadDegradation.CriticalFactor-ce.LoadDegradation.WarningFactor)
	default:
		// Exponential degradation beyond critical
		excess := utilization - ce.LoadDegradation.CriticalThreshold
		return ce.LoadDegradation.CriticalFactor * (1.0 + excess*10.0)
	}
}

// calculateQueuePenaltyFactor calculates penalty based on queue utilization
func (ce *CommonEngine) calculateQueuePenaltyFactor() float64 {
	queueUtil := float64(len(ce.Queue)) / float64(ce.QueueCap)
	
	if queueUtil < 0.5 {
		return 1.0 // No penalty
	} else if queueUtil < 0.8 {
		return 1.0 + (queueUtil-0.5)*0.4 // Up to 1.12x penalty
	} else {
		return 1.12 + (queueUtil-0.8)*2.0 // Up to 1.52x penalty
	}
}

// calculateHealthPenaltyFactor calculates penalty based on health score
func (ce *CommonEngine) calculateHealthPenaltyFactor() float64 {
	if ce.Health.Score >= 0.8 {
		return 1.0 // No penalty for healthy engines
	} else if ce.Health.Score >= 0.5 {
		return 1.0 + (0.8-ce.Health.Score)*0.5 // Up to 1.15x penalty
	} else {
		return 1.15 + (0.5-ce.Health.Score)*2.0 // Up to 2.15x penalty
	}
}

// calculateVarianceFactor adds realistic performance variance
func (ce *CommonEngine) calculateVarianceFactor(utilization float64) float64 {
	// Base variance increases with load
	variance := ce.Variance.BaseVariance * (1.0 + utilization*ce.Variance.LoadMultiplier)
	
	// Variance reduces with scale (more operations = more predictable)
	if ce.ConvergenceState.OperationCount > 100 {
		scaleReduction := ce.Variance.ScaleReduction * math.Log(float64(ce.ConvergenceState.OperationCount)/100.0)
		variance = variance * (1.0 - math.Min(scaleReduction, 0.8))
	}
	
	// For deterministic behavior, return base factor without random variance
	// But still apply the calculated variance for load-dependent behavior
	return 1.0 + variance
}

// UpdateHealth updates the health metrics
func (ce *CommonEngine) UpdateHealth() {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	// Calculate utilization (to be overridden by specific engines)
	queueUtil := float64(len(ce.Queue)) / float64(ce.QueueCap)

	// Calculate error rate
	errorRate := 0.0
	if ce.TotalOperations > 0 {
		errorRate = float64(ce.FailedOps) / float64(ce.TotalOperations)
	}

	// Calculate average latency from recent operations
	avgLatency := 0.0
	if len(ce.OperationHistory) > 0 {
		total := time.Duration(0)
		for _, duration := range ce.OperationHistory {
			total += duration
		}
		avgLatency = float64(total/time.Duration(len(ce.OperationHistory))) / float64(time.Millisecond)
	}

	// Calculate throughput (operations per second)
	throughput := 0.0
	if ce.CurrentTick > 0 {
		timeSeconds := float64(ce.CurrentTick) * ce.TickDuration.Seconds()
		throughput = float64(ce.CompletedOps) / timeSeconds
	}

	// Update health metrics
	ce.Health.QueueUtilization = queueUtil
	ce.Health.ErrorRate = errorRate
	ce.Health.AverageLatency = avgLatency
	ce.Health.ThroughputOps = throughput
	ce.Health.LastUpdated = ce.CurrentTick

	// Calculate overall health score
	ce.Health.Score = ce.calculateHealthScore()
}

// calculateHealthScore calculates overall health score (0.0 to 1.0)
func (ce *CommonEngine) calculateHealthScore() float64 {
	score := 1.0

	// Penalize high utilization
	if ce.Health.Utilization > 0.8 {
		score -= (ce.Health.Utilization - 0.8) * 0.5
	}

	// Penalize high queue utilization
	if ce.Health.QueueUtilization > 0.7 {
		score -= (ce.Health.QueueUtilization - 0.7) * 0.3
	}

	// Penalize high error rate
	score -= ce.Health.ErrorRate * 0.5

	// Penalize high latency (relative to baseline)
	if ce.Health.AverageLatency > 10.0 { // 10ms baseline
		latencyPenalty := (ce.Health.AverageLatency - 10.0) / 100.0
		score -= math.Min(latencyPenalty, 0.3)
	}

	return math.Max(0.0, math.Min(1.0, score))
}

// GetDynamicState returns current dynamic state
func (ce *CommonEngine) GetDynamicState() *DynamicState {
	return &DynamicState{
		CurrentUtilization:  ce.Health.Utilization,
		PerformanceFactor:   ce.calculateCurrentPerformanceFactor(),
		ConvergenceProgress: ce.calculateConvergenceProgress(),
		HardwareSpecific:    make(map[string]interface{}),
		LastUpdated:         ce.CurrentTick,
	}
}

// UpdateDynamicBehavior updates dynamic behavior state
func (ce *CommonEngine) UpdateDynamicBehavior() {
	// Update convergence state
	ce.updateConvergenceState()

	// Update load history
	ce.updateLoadHistory()

	// Update operation history (keep last 1000)
	if len(ce.OperationHistory) > 1000 {
		ce.OperationHistory = ce.OperationHistory[len(ce.OperationHistory)-1000:]
	}

	// Update load history (keep last 1000)
	if len(ce.LoadHistory) > 1000 {
		ce.LoadHistory = ce.LoadHistory[len(ce.LoadHistory)-1000:]
	}
}

// GetConvergenceMetrics returns convergence metrics
func (ce *CommonEngine) GetConvergenceMetrics() *ConvergenceMetrics {
	factors := make(map[string]float64)
	for name, model := range ce.ConvergenceState.Models {
		factors[name] = model.CurrentValue
	}

	return &ConvergenceMetrics{
		OperationCount:     ce.ConvergenceState.OperationCount,
		ConvergencePoint:   ce.calculateOverallConvergencePoint(),
		CurrentVariance:    ce.calculateCurrentVariance(),
		IsConverged:        ce.isFullyConverged(),
		TimeToConvergence:  ce.ConvergenceState.ConvergedTick - ce.ConvergenceState.StartTick,
		ConvergenceFactors: factors,
	}
}

// Helper methods for convergence calculations
func (ce *CommonEngine) calculateCurrentPerformanceFactor() float64 {
	utilization := ce.Health.Utilization
	return ce.calculateLoadDegradationFactor(utilization) *
		   ce.calculateQueuePenaltyFactor() *
		   ce.calculateHealthPenaltyFactor()
}

func (ce *CommonEngine) calculateConvergenceProgress() float64 {
	if len(ce.ConvergenceState.Models) == 0 {
		return 0.0
	}

	convergedCount := 0
	for _, model := range ce.ConvergenceState.Models {
		if model.IsConverged {
			convergedCount++
		}
	}

	return float64(convergedCount) / float64(len(ce.ConvergenceState.Models))
}

func (ce *CommonEngine) updateConvergenceState() {
	ce.ConvergenceState.OperationCount = ce.TotalOperations

	// Check if we've reached convergence
	if !ce.isFullyConverged() && ce.calculateConvergenceProgress() >= 1.0 {
		ce.ConvergenceState.ConvergedTick = ce.CurrentTick
	}
}

func (ce *CommonEngine) updateLoadHistory() {
	ce.LoadHistory = append(ce.LoadHistory, ce.Health.Utilization)
}

func (ce *CommonEngine) calculateOverallConvergencePoint() float64 {
	if len(ce.ConvergenceState.Models) == 0 {
		return 1.0
	}

	total := 0.0
	for _, model := range ce.ConvergenceState.Models {
		total += model.ConvergencePoint
	}

	return total / float64(len(ce.ConvergenceState.Models))
}

func (ce *CommonEngine) calculateCurrentVariance() float64 {
	if ce.ConvergenceState.OperationCount < 100 {
		return ce.Variance.BaseVariance
	}

	// Variance reduces with more operations
	scaleReduction := math.Log(float64(ce.ConvergenceState.OperationCount)/100.0) * ce.Variance.ScaleReduction
	return ce.Variance.BaseVariance * (1.0 - math.Min(scaleReduction, 0.8))
}

func (ce *CommonEngine) isFullyConverged() bool {
	for _, model := range ce.ConvergenceState.Models {
		if !model.IsConverged {
			return false
		}
	}
	return len(ce.ConvergenceState.Models) > 0
}

// AddOperationToHistory adds an operation to the history for convergence tracking
func (ce *CommonEngine) AddOperationToHistory(duration time.Duration) {
	ce.OperationHistory = append(ce.OperationHistory, duration)
	ce.TotalOperations++
}

// Additional helper methods for common functionality

// loadLoadCurve loads load degradation curve from profile
func (ce *CommonEngine) loadLoadCurve(curveMap map[string]interface{}) {
	if optimal, ok := curveMap["optimal_threshold"].(float64); ok {
		ce.LoadDegradation.OptimalThreshold = optimal
	}
	if warning, ok := curveMap["warning_threshold"].(float64); ok {
		ce.LoadDegradation.WarningThreshold = warning
	}
	if critical, ok := curveMap["critical_threshold"].(float64); ok {
		ce.LoadDegradation.CriticalThreshold = critical
	}
	if optimalFactor, ok := curveMap["optimal_factor"].(float64); ok {
		ce.LoadDegradation.OptimalFactor = optimalFactor
	}
	if warningFactor, ok := curveMap["warning_factor"].(float64); ok {
		ce.LoadDegradation.WarningFactor = warningFactor
	}
	if criticalFactor, ok := curveMap["critical_factor"].(float64); ok {
		ce.LoadDegradation.CriticalFactor = criticalFactor
	}
}
