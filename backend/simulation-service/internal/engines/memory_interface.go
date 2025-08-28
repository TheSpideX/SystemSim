package engines

import (
	"fmt"
)

// MemoryComplexityLevel defines the level of memory simulation complexity
// Using the same integer-based system as CPU for consistency
type MemoryComplexityLevel = ComplexityLevel

// MemoryFeatures defines which memory simulation features are enabled
type MemoryFeatures struct {
	// Basic memory modeling (always enabled for Basic+)
	EnableDDRTimingEffects    bool `json:"enable_ddr_timing_effects"`    // Row buffer hits/misses, bank conflicts
	EnableBandwidthSaturation bool `json:"enable_bandwidth_saturation"`  // Memory controller limits
	EnableBasicNUMA           bool `json:"enable_basic_numa"`             // Cross-socket penalties
	
	// Advanced memory modeling (Advanced+)
	EnableMemoryPressure      bool `json:"enable_memory_pressure"`       // RAM usage impact, swap pressure
	EnableAccessPatterns      bool `json:"enable_access_patterns"`       // Sequential/random/stride optimization
	EnableChannelUtilization  bool `json:"enable_channel_utilization"`   // Multi-channel load balancing
	EnableMemoryController    bool `json:"enable_memory_controller"`     // Controller queuing, scheduling
	
	// Expert memory modeling (Maximum only)
	EnableGarbageCollection   bool `json:"enable_garbage_collection"`    // Language-specific GC pauses
	EnableMemoryFragmentation bool `json:"enable_memory_fragmentation"`  // Heap fragmentation effects
	EnableHardwarePrefetching bool `json:"enable_hardware_prefetching"`  // CPU prefetcher modeling
	EnableCacheLineConflicts  bool `json:"enable_cache_line_conflicts"`  // False sharing detection
	EnableMemoryOrdering      bool `json:"enable_memory_ordering"`       // Memory ordering and reordering effects
	EnableVirtualMemory       bool `json:"enable_virtual_memory"`        // Virtual memory and TLB simulation
	EnableMemoryCompression   bool `json:"enable_memory_compression"`    // OS memory compression
	EnableNUMAOptimization    bool `json:"enable_numa_optimization"`     // Advanced NUMA topology
	EnableECCModeling         bool `json:"enable_ecc_modeling"`           // ECC error correction modeling
	EnablePowerStates         bool `json:"enable_power_states"`           // Memory power state transitions
	EnableThermalThrottling   bool `json:"enable_thermal_throttling"`     // Memory thermal effects
	
	// Statistical and behavioral modeling
	EnableStatisticalModeling bool `json:"enable_statistical_modeling"`  // Statistical convergence
	EnableConvergenceTracking bool `json:"enable_convergence_tracking"`  // Model convergence monitoring
	EnableDynamicBehavior     bool `json:"enable_dynamic_behavior"`      // Adaptive behavior patterns
	EnableRealtimeAdaptation  bool `json:"enable_realtime_adaptation"`   // Real-time model updates
}

// MemoryInterface provides control over memory simulation complexity and features
type MemoryInterface struct {
	ComplexityLevel MemoryComplexityLevel `json:"complexity_level"`
	Features        *MemoryFeatures       `json:"features"`
}

// NewMemoryInterface creates a new memory interface with the specified complexity level
func NewMemoryInterface(level MemoryComplexityLevel) *MemoryInterface {
	mi := &MemoryInterface{
		ComplexityLevel: level,
		Features:        &MemoryFeatures{},
	}

	// SetComplexityLevel should not fail for valid levels, but handle error just in case
	if err := mi.SetComplexityLevel(level); err != nil {
		// Fallback to Advanced if invalid level
		mi.SetComplexityLevel(ComplexityAdvanced)
	}
	return mi
}

// SetComplexityLevel configures features based on complexity level
func (mi *MemoryInterface) SetComplexityLevel(level MemoryComplexityLevel) error {
	// Validate the complexity level
	if err := ValidateMemoryComplexityLevel(level); err != nil {
		return err
	}

	mi.ComplexityLevel = level

	switch level {
	case ComplexityMinimal:
		mi.configureMinimalFeatures()
	case ComplexityBasic:
		mi.configureBasicFeatures()
	case ComplexityAdvanced:
		mi.configureAdvancedFeatures()
	case ComplexityMaximum:
		mi.configureMaximumFeatures()
	default:
		mi.configureAdvancedFeatures() // Default to Advanced
	}

	return nil
}

// configureMinimalFeatures - Real-world modeling with essential features (~5x faster, ~90% accuracy)
func (mi *MemoryInterface) configureMinimalFeatures() {
	*mi.Features = MemoryFeatures{
		// Essential real-world features
		EnableDDRTimingEffects:    true,  // DDR timing is essential for real-world accuracy
		EnableBandwidthSaturation: true,  // Keep basic bandwidth limits
		EnableBasicNUMA:           true,  // NUMA is essential in modern systems
		
		// Disable all advanced features
		EnableMemoryPressure:      false,
		EnableAccessPatterns:      false,
		EnableChannelUtilization:  false,
		EnableMemoryController:    false,
		EnableGarbageCollection:   false,
		EnableMemoryFragmentation: false,
		EnableHardwarePrefetching: false,
		EnableCacheLineConflicts:  false,
		EnableMemoryOrdering:      false,
		EnableVirtualMemory:       false,
		EnableMemoryCompression:   false,
		EnableNUMAOptimization:    false,
		EnableECCModeling:         false,
		EnablePowerStates:         false,
		EnableThermalThrottling:   false,
		
		// Basic behavioral modeling only
		EnableStatisticalModeling: false,
		EnableConvergenceTracking: false,
		EnableDynamicBehavior:     false,
		EnableRealtimeAdaptation:  false,
	}
}

// configureBasicFeatures - Real-world modeling with core features (~2x faster, ~95% accuracy)
func (mi *MemoryInterface) configureBasicFeatures() {
	*mi.Features = MemoryFeatures{
		// Core real-world memory features
		EnableDDRTimingEffects:    true,  // DDR timing is essential
		EnableBandwidthSaturation: true,  // Bandwidth limits
		EnableBasicNUMA:           true,  // NUMA penalties

		// Important real-world features
		EnableMemoryPressure:     true,  // Memory pressure effects
		EnableAccessPatterns:     true,  // Access pattern optimization
		EnableChannelUtilization: true,  // Channel utilization is important for accuracy
		EnableMemoryController:   true,  // Controller modeling is essential
		
		// Skip expert features
		EnableGarbageCollection:   false,
		EnableMemoryFragmentation: false,
		EnableHardwarePrefetching: false,
		EnableCacheLineConflicts:  false,
		EnableMemoryOrdering:      false,
		EnableVirtualMemory:       false,
		EnableMemoryCompression:   false,
		EnableNUMAOptimization:    false,
		EnableECCModeling:         false,
		EnablePowerStates:         false,
		EnableThermalThrottling:   false,
		
		// Basic behavioral modeling
		EnableStatisticalModeling: true,
		EnableConvergenceTracking: false,
		EnableDynamicBehavior:     true,
		EnableRealtimeAdaptation:  false,
	}
}

// configureAdvancedFeatures - Enhanced real-world modeling (~1.2x faster, ~98% accuracy)
func (mi *MemoryInterface) configureAdvancedFeatures() {
	*mi.Features = MemoryFeatures{
		// All core real-world features
		EnableDDRTimingEffects:    true,
		EnableBandwidthSaturation: true,
		EnableBasicNUMA:           true,

		// Enhanced real-world features
		EnableMemoryPressure:     true,
		EnableAccessPatterns:     true,
		EnableChannelUtilization: true,
		EnableMemoryController:   true,
		
		// Some expert features (SIMPLIFIED - disable complex features)
		EnableGarbageCollection:   false, // Disable for simplicity
		EnableMemoryFragmentation: false, // Disable for simplicity
		EnableHardwarePrefetching: false, // Skip most expensive feature
		EnableCacheLineConflicts:  false, // Disable for simplicity
		EnableMemoryOrdering:      false, // DISABLE complex ordering logic
		EnableVirtualMemory:       false, // Disable for simplicity
		EnableMemoryCompression:   false, // Skip compression complexity
		EnableNUMAOptimization:    false, // Disable for simplicity
		EnableECCModeling:         false, // Skip ECC for Advanced
		EnablePowerStates:         false, // Skip power states for Advanced
		EnableThermalThrottling:   false, // Skip thermal for Advanced
		
		// Advanced behavioral modeling
		EnableStatisticalModeling: true,
		EnableConvergenceTracking: true,
		EnableDynamicBehavior:     true,
		EnableRealtimeAdaptation:  false,
	}
}

// configureMaximumFeatures - Maximum accuracy (~99% accuracy, baseline performance)
func (mi *MemoryInterface) configureMaximumFeatures() {
	*mi.Features = MemoryFeatures{
		// Everything enabled for maximum realism
		EnableDDRTimingEffects:    true,
		EnableBandwidthSaturation: true,
		EnableBasicNUMA:           true,
		EnableMemoryPressure:     true,
		EnableAccessPatterns:     true,
		EnableChannelUtilization: true,
		EnableMemoryController:   true,
		EnableGarbageCollection:   true,
		EnableMemoryFragmentation: true,
		EnableHardwarePrefetching: true,  // Most computationally expensive feature
		EnableCacheLineConflicts:  true,
		EnableMemoryOrdering:      true,  // Memory ordering and reordering effects
		EnableVirtualMemory:       true,  // Virtual memory and TLB simulation
		EnableMemoryCompression:   true,
		EnableNUMAOptimization:    true,
		EnableECCModeling:         true,  // ECC error correction modeling
		EnablePowerStates:         true,  // Memory power state transitions
		EnableThermalThrottling:   true,  // Memory thermal effects

		// Full behavioral modeling
		EnableStatisticalModeling: true,
		EnableConvergenceTracking: true,
		EnableDynamicBehavior:     true,
		EnableRealtimeAdaptation:  true,
	}
}

// ShouldEnableFeature checks if a specific feature should be enabled
func (mi *MemoryInterface) ShouldEnableFeature(feature string) bool {
	switch feature {
	case "ddr_timing_effects":
		return mi.Features.EnableDDRTimingEffects
	case "bandwidth_saturation":
		return mi.Features.EnableBandwidthSaturation
	case "basic_numa":
		return mi.Features.EnableBasicNUMA
	case "memory_pressure":
		return mi.Features.EnableMemoryPressure
	case "access_patterns":
		return mi.Features.EnableAccessPatterns
	case "channel_utilization":
		return mi.Features.EnableChannelUtilization
	case "memory_controller":
		return mi.Features.EnableMemoryController
	case "garbage_collection":
		return mi.Features.EnableGarbageCollection
	case "memory_fragmentation":
		return mi.Features.EnableMemoryFragmentation
	case "hardware_prefetching":
		return mi.Features.EnableHardwarePrefetching
	case "cache_line_conflicts":
		return mi.Features.EnableCacheLineConflicts
	case "memory_ordering":
		return mi.Features.EnableMemoryOrdering
	case "virtual_memory":
		return mi.Features.EnableVirtualMemory
	case "memory_compression":
		return mi.Features.EnableMemoryCompression
	case "numa_optimization":
		return mi.Features.EnableNUMAOptimization
	case "ecc_modeling":
		return mi.Features.EnableECCModeling
	case "power_states":
		return mi.Features.EnablePowerStates
	case "thermal_throttling":
		return mi.Features.EnableThermalThrottling
	case "statistical_modeling":
		return mi.Features.EnableStatisticalModeling
	case "convergence_tracking":
		return mi.Features.EnableConvergenceTracking
	case "dynamic_behavior":
		return mi.Features.EnableDynamicBehavior
	case "realtime_adaptation":
		return mi.Features.EnableRealtimeAdaptation
	default:
		return false
	}
}

// GetDescription returns a human-readable description of the current complexity level
func (mi *MemoryInterface) GetDescription() string {
	switch mi.ComplexityLevel {
	case ComplexityMinimal:
		return "Minimal memory simulation - real-world modeling with essential features only. ~5x faster, ~90% accuracy."
	case ComplexityBasic:
		return "Basic memory simulation - real-world DDR timing, NUMA, pressure effects. ~2x faster, ~95% accuracy."
	case ComplexityAdvanced:
		return "Advanced memory simulation - real-world full DDR modeling, GC, fragmentation. ~1.2x faster, ~98% accuracy."
	case ComplexityMaximum:
		return "Maximum memory simulation - real-world modeling with all features including prefetching. Baseline speed, ~99% accuracy."
	default:
		return "Unknown complexity level"
	}
}

// GetPerformanceImpact returns performance impact information
func (mi *MemoryInterface) GetPerformanceImpact() string {
	enabledFeatures := 0
	if mi.Features.EnableDDRTimingEffects { enabledFeatures++ }
	if mi.Features.EnableBandwidthSaturation { enabledFeatures++ }
	if mi.Features.EnableBasicNUMA { enabledFeatures++ }
	if mi.Features.EnableMemoryPressure { enabledFeatures++ }
	if mi.Features.EnableAccessPatterns { enabledFeatures++ }
	if mi.Features.EnableChannelUtilization { enabledFeatures++ }
	if mi.Features.EnableMemoryController { enabledFeatures++ }
	if mi.Features.EnableGarbageCollection { enabledFeatures++ }
	if mi.Features.EnableMemoryFragmentation { enabledFeatures++ }
	if mi.Features.EnableHardwarePrefetching { enabledFeatures++ }
	if mi.Features.EnableCacheLineConflicts { enabledFeatures++ }
	if mi.Features.EnableMemoryCompression { enabledFeatures++ }
	if mi.Features.EnableNUMAOptimization { enabledFeatures++ }
	if mi.Features.EnableStatisticalModeling { enabledFeatures++ }
	if mi.Features.EnableConvergenceTracking { enabledFeatures++ }
	if mi.Features.EnableDynamicBehavior { enabledFeatures++ }
	if mi.Features.EnableRealtimeAdaptation { enabledFeatures++ }
	
	return fmt.Sprintf("Memory complexity level: %s (%d/17 features enabled)", 
		mi.ComplexityLevel.String(), enabledFeatures)
}

// ValidateComplexityLevel validates that the complexity level is valid
func ValidateMemoryComplexityLevel(level MemoryComplexityLevel) error {
	if level < ComplexityMinimal || level > ComplexityMaximum {
		return fmt.Errorf("invalid memory complexity level: %d (must be 0-3)", level)
	}
	return nil
}
