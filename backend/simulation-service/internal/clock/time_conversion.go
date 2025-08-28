package clock

import (
	"fmt"
	"time"
)

// Global time conversion utilities
// These work with the fixed TICK_DURATION (0.01ms = 10 microseconds)

// DurationToTicks converts a duration to number of ticks
// Uses the global TICK_DURATION constant
func DurationToTicks(duration time.Duration) int64 {
	return int64(duration / TICK_DURATION)
}

// TicksToDuration converts number of ticks to duration
// Uses the global TICK_DURATION constant
func TicksToDuration(ticks int64) time.Duration {
	return time.Duration(ticks) * TICK_DURATION
}

// CalculateProcessingTicks calculates the number of ticks needed for an operation
// considering base time and load factor
func CalculateProcessingTicks(baseTime time.Duration, loadFactor float64) int64 {
	adjustedTime := time.Duration(float64(baseTime) * loadFactor)
	return DurationToTicks(adjustedTime)
}

// CalculateCompletionTick calculates when an operation will complete
func CalculateCompletionTick(currentTick int64, processingTime time.Duration) int64 {
	processingTicks := DurationToTicks(processingTime)
	return currentTick + processingTicks
}

// TimeToTicksWithValidation converts time to ticks with validation
func TimeToTicksWithValidation(duration time.Duration) (int64, error) {
	if duration < 0 {
		return 0, fmt.Errorf("duration cannot be negative: %v", duration)
	}
	
	ticks := DurationToTicks(duration)
	
	// Ensure minimum of 1 tick for any positive duration
	if duration > 0 && ticks == 0 {
		ticks = 1
	}
	
	return ticks, nil
}

// TicksToTimeWithValidation converts ticks to time with validation
func TicksToTimeWithValidation(ticks int64) (time.Duration, error) {
	if ticks < 0 {
		return 0, fmt.Errorf("ticks cannot be negative: %d", ticks)
	}
	
	return TicksToDuration(ticks), nil
}

// ScaleTimeForSimulation applies scaling factor to convert real time to simulation time
func ScaleTimeForSimulation(realTime time.Duration, scalingFactor float64) time.Duration {
	return time.Duration(float64(realTime) / scalingFactor)
}

// ScaleTimeForReal applies scaling factor to convert simulation time to real time
func ScaleTimeForReal(simulationTime time.Duration, scalingFactor float64) time.Duration {
	return time.Duration(float64(simulationTime) * scalingFactor)
}

// TimeConversionInfo provides detailed information about time conversions
type TimeConversionInfo struct {
	OriginalDuration time.Duration `json:"original_duration"`
	Ticks           int64         `json:"ticks"`
	ConvertedBack   time.Duration `json:"converted_back"`
	LossNanoseconds int64         `json:"loss_nanoseconds"`
	LossPercentage  float64       `json:"loss_percentage"`
}

// AnalyzeTimeConversion provides detailed analysis of time conversion accuracy
func AnalyzeTimeConversion(duration time.Duration) TimeConversionInfo {
	ticks := DurationToTicks(duration)
	convertedBack := TicksToDuration(ticks)
	
	loss := duration - convertedBack
	lossPercentage := 0.0
	if duration > 0 {
		lossPercentage = float64(loss) / float64(duration) * 100
	}
	
	return TimeConversionInfo{
		OriginalDuration: duration,
		Ticks:           ticks,
		ConvertedBack:   convertedBack,
		LossNanoseconds: loss.Nanoseconds(),
		LossPercentage:  lossPercentage,
	}
}

// ValidateTickDuration checks if the current tick duration is appropriate
func ValidateTickDuration(smallestOperation time.Duration) ValidationResult {
	// Check if smallest operation is at least 1 tick
	ticks := DurationToTicks(smallestOperation)
	
	if ticks == 0 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Smallest operation (%v) is smaller than tick duration (%v)", smallestOperation, TICK_DURATION),
			Recommendation: fmt.Sprintf("Consider reducing tick duration to %v", smallestOperation/10),
		}
	}
	
	if ticks == 1 {
		return ValidationResult{
			Valid:   true,
			Message: fmt.Sprintf("Tick duration (%v) is optimal for smallest operation (%v)", TICK_DURATION, smallestOperation),
			Recommendation: "Current tick duration is appropriate",
		}
	}
	
	// If smallest operation is much larger than tick duration
	if ticks > 100 {
		return ValidationResult{
			Valid:   true,
			Message: fmt.Sprintf("Tick duration (%v) provides high granularity for smallest operation (%v = %d ticks)", TICK_DURATION, smallestOperation, ticks),
			Recommendation: fmt.Sprintf("Could increase tick duration to %v for better performance", smallestOperation/10),
		}
	}
	
	return ValidationResult{
		Valid:   true,
		Message: fmt.Sprintf("Tick duration (%v) is appropriate for smallest operation (%v = %d ticks)", TICK_DURATION, smallestOperation, ticks),
		Recommendation: "Current tick duration is well-suited",
	}
}

// ValidationResult holds the result of tick duration validation
type ValidationResult struct {
	Valid          bool   `json:"valid"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation"`
}

// OperationTimingInfo provides timing information for operations
type OperationTimingInfo struct {
	BaseTime        time.Duration `json:"base_time"`
	LoadFactor      float64       `json:"load_factor"`
	AdjustedTime    time.Duration `json:"adjusted_time"`
	ProcessingTicks int64         `json:"processing_ticks"`
	StartTick       int64         `json:"start_tick"`
	CompletionTick  int64         `json:"completion_tick"`
}

// CalculateOperationTiming provides comprehensive timing calculation for an operation
func CalculateOperationTiming(baseTime time.Duration, loadFactor float64, currentTick int64) OperationTimingInfo {
	adjustedTime := time.Duration(float64(baseTime) * loadFactor)
	processingTicks := DurationToTicks(adjustedTime)
	completionTick := currentTick + processingTicks
	
	return OperationTimingInfo{
		BaseTime:        baseTime,
		LoadFactor:      loadFactor,
		AdjustedTime:    adjustedTime,
		ProcessingTicks: processingTicks,
		StartTick:       currentTick,
		CompletionTick:  completionTick,
	}
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000)
	}
	return d.String()
}

// FormatTicks formats ticks in a human-readable way with equivalent time
func FormatTicks(ticks int64) string {
	duration := TicksToDuration(ticks)
	return fmt.Sprintf("%d ticks (%s)", ticks, FormatDuration(duration))
}

// GetTickDurationInfo returns information about the current tick duration
func GetTickDurationInfo() TickDurationInfo {
	return TickDurationInfo{
		Duration:      TICK_DURATION,
		Nanoseconds:   TICK_DURATION.Nanoseconds(),
		Microseconds:  float64(TICK_DURATION.Nanoseconds()) / 1000,
		Milliseconds:  float64(TICK_DURATION.Nanoseconds()) / 1000000,
		TicksPerSecond: int64(time.Second / TICK_DURATION),
	}
}

// TickDurationInfo provides detailed information about tick duration
type TickDurationInfo struct {
	Duration       time.Duration `json:"duration"`
	Nanoseconds    int64         `json:"nanoseconds"`
	Microseconds   float64       `json:"microseconds"`
	Milliseconds   float64       `json:"milliseconds"`
	TicksPerSecond int64         `json:"ticks_per_second"`
}
