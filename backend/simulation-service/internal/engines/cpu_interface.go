package engines

import (
	"fmt"
)

// ComplexityLevel defines the level of simulation complexity (integer-based)
type ComplexityLevel int

const (
	// ComplexityMinimal - Basic simulation with minimal overhead
	ComplexityMinimal ComplexityLevel = 0

	// ComplexityBasic - Standard simulation with core features
	ComplexityBasic ComplexityLevel = 1

	// ComplexityAdvanced - Full simulation with advanced features
	ComplexityAdvanced ComplexityLevel = 2

	// ComplexityMaximum - Maximum fidelity with all features enabled
	ComplexityMaximum ComplexityLevel = 3
)

// Legacy aliases for backward compatibility
const (
	CPUMinimal  = ComplexityMinimal
	CPUBasic    = ComplexityBasic
	CPUAdvanced = ComplexityAdvanced
	CPUMaximum  = ComplexityMaximum
)

type CPUComplexityLevel = ComplexityLevel

func (cl ComplexityLevel) String() string {
	switch cl {
	case ComplexityMinimal:
		return "Minimal"
	case ComplexityBasic:
		return "Basic"
	case ComplexityAdvanced:
		return "Advanced"
	case ComplexityMaximum:
		return "Maximum"
	default:
		return "Unknown"
	}
}

// CPUInterface provides control over CPU simulation complexity
type CPUInterface struct {
	ComplexityLevel ComplexityLevel `json:"complexity_level"`
	Features        *CPUFeatures    `json:"features"`
}

// CPUFeatures defines which CPU features are enabled
type CPUFeatures struct {
	// Core Features
	EnableLanguageMultipliers bool `json:"enable_language_multipliers"`
	EnableComplexityFactors   bool `json:"enable_complexity_factors"`
	EnableBasicCaching        bool `json:"enable_basic_caching"`
	
	// Advanced Features
	EnableSIMDVectorization   bool `json:"enable_simd_vectorization"`
	EnableAdvancedCaching     bool `json:"enable_advanced_cache_hierarchy"`
	EnableThermalModeling     bool `json:"enable_thermal_modeling"`
	EnableBoostClocks         bool `json:"enable_boost_clocks"`
	
	// Expert Features
	EnableNUMATopology        bool `json:"enable_numa_topology"`
	EnableHyperthreading      bool `json:"enable_hyperthreading"`
	EnableBranchPrediction    bool `json:"enable_branch_prediction"`
	EnableAdvancedPrefetching bool `json:"enable_advanced_prefetching"`
	EnableMemoryBandwidth     bool `json:"enable_memory_bandwidth_contention"`
	EnableParallelProcessing  bool `json:"enable_parallel_processing"`
	
	// Behavioral Features
	EnableStatisticalModeling bool `json:"enable_statistical_modeling"`
	EnableConvergenceTracking bool `json:"enable_convergence_tracking"`
	EnableDynamicBehavior     bool `json:"enable_dynamic_behavior"`
}

// NewCPUInterface creates a new CPU interface with specified complexity level
func NewCPUInterface(level CPUComplexityLevel) *CPUInterface {
	iface := &CPUInterface{
		ComplexityLevel: level,
		Features:        &CPUFeatures{},
	}
	
	iface.configureFeatures()
	return iface
}

// SetComplexityLevel changes the CPU simulation complexity level
func (ci *CPUInterface) SetComplexityLevel(level CPUComplexityLevel) {
	ci.ComplexityLevel = level
	ci.configureFeatures()
}

// configureFeatures sets up features based on complexity level
func (ci *CPUInterface) configureFeatures() {
	switch ci.ComplexityLevel {
	case CPUMinimal:
		ci.configureMinimalFeatures()
	case CPUBasic:
		ci.configureBasicFeatures()
	case CPUAdvanced:
		ci.configureAdvancedFeatures()
	case CPUMaximum:
		ci.configureMaximumFeatures()
	}
}

// configureMinimalFeatures - Real-world modeling with essential features only (~5x faster, ~90% accuracy)
func (ci *CPUInterface) configureMinimalFeatures() {
	*ci.Features = CPUFeatures{
		// Essential real-world features
		EnableLanguageMultipliers: true,  // Keep language differences (real-world impact)
		EnableComplexityFactors:   true,  // Keep basic complexity scaling
		EnableBasicCaching:        true,  // Keep basic cache modeling (essential for accuracy)
		
		// Disable all advanced features
		EnableSIMDVectorization:   false,
		EnableAdvancedCaching:     false,
		EnableThermalModeling:     false,
		EnableBoostClocks:         false,
		EnableNUMATopology:        false,
		EnableHyperthreading:      false,
		EnableBranchPrediction:    false,
		EnableAdvancedPrefetching: false,
		EnableMemoryBandwidth:     false,
		EnableParallelProcessing:  false,
		
		// Minimal behavioral modeling
		EnableStatisticalModeling: false,
		EnableConvergenceTracking: false,
		EnableDynamicBehavior:     false,
	}
}

// configureBasicFeatures - Real-world modeling with core features (~2x faster, ~95% accuracy)
func (ci *CPUInterface) configureBasicFeatures() {
	*ci.Features = CPUFeatures{
		// Core real-world features
		EnableLanguageMultipliers: true,  // Language differences are important
		EnableComplexityFactors:   true,  // Algorithm complexity scaling
		EnableBasicCaching:        true,  // Basic cache hit/miss modeling

		// Common real-world features
		EnableSIMDVectorization:   true,  // SIMD is widely used
		EnableAdvancedCaching:     true,  // Cache hierarchy is essential for real-world accuracy
		EnableThermalModeling:     true,  // Thermal effects are real-world critical
		EnableBoostClocks:         true,  // Boost clocks are standard
		
		// Skip expert-level features
		EnableNUMATopology:        false,
		EnableHyperthreading:      false,
		EnableBranchPrediction:    false,
		EnableAdvancedPrefetching: false,
		EnableMemoryBandwidth:     false,
		EnableParallelProcessing:  true,  // Basic parallel processing
		
		// Basic behavioral modeling
		EnableStatisticalModeling: true,
		EnableConvergenceTracking: false,
		EnableDynamicBehavior:     true,
	}
}

// configureAdvancedFeatures - Enhanced real-world modeling (~1.2x faster, ~98% accuracy)
func (ci *CPUInterface) configureAdvancedFeatures() {
	*ci.Features = CPUFeatures{
		// All core real-world features
		EnableLanguageMultipliers: true,
		EnableComplexityFactors:   true,
		EnableBasicCaching:        true,

		// Enhanced real-world features
		EnableSIMDVectorization:   true,
		EnableAdvancedCaching:     true,  // Full L1/L2/L3 cache hierarchy
		EnableThermalModeling:     true,
		EnableBoostClocks:         true,
		EnableNUMATopology:        true,  // NUMA topology effects
		EnableHyperthreading:      true,  // Hyperthreading modeling
		EnableBranchPrediction:    true,  // Branch prediction accuracy
		EnableAdvancedPrefetching: true,  // Include for enhanced accuracy
		EnableMemoryBandwidth:     true,  // Memory bandwidth contention
		EnableParallelProcessing:  true,
		
		// Advanced behavioral modeling
		EnableStatisticalModeling: true,
		EnableConvergenceTracking: true,
		EnableDynamicBehavior:     true,
	}
}

// configureMaximumFeatures - Maximum accuracy (~99% accuracy, baseline performance)
func (ci *CPUInterface) configureMaximumFeatures() {
	*ci.Features = CPUFeatures{
		// Everything enabled for maximum realism
		EnableLanguageMultipliers: true,
		EnableComplexityFactors:   true,
		EnableBasicCaching:        true,
		EnableSIMDVectorization:   true,
		EnableAdvancedCaching:     true,
		EnableThermalModeling:     true,
		EnableBoostClocks:         true,
		EnableNUMATopology:        true,
		EnableHyperthreading:      true,
		EnableBranchPrediction:    true,
		EnableAdvancedPrefetching: true,  // Most computationally expensive feature
		EnableMemoryBandwidth:     true,
		EnableParallelProcessing:  true,
		EnableStatisticalModeling: true,
		EnableConvergenceTracking: true,
		EnableDynamicBehavior:     true,
	}
}

// ShouldEnableFeature checks if a specific CPU feature should be enabled
func (ci *CPUInterface) ShouldEnableFeature(feature string) bool {
	switch feature {
	case "language_multipliers":
		return ci.Features.EnableLanguageMultipliers
	case "complexity_factors":
		return ci.Features.EnableComplexityFactors
	case "basic_caching":
		return ci.Features.EnableBasicCaching
	case "simd_vectorization":
		return ci.Features.EnableSIMDVectorization
	case "advanced_caching":
		return ci.Features.EnableAdvancedCaching
	case "thermal_modeling":
		return ci.Features.EnableThermalModeling
	case "boost_clocks":
		return ci.Features.EnableBoostClocks
	case "numa_topology":
		return ci.Features.EnableNUMATopology
	case "hyperthreading":
		return ci.Features.EnableHyperthreading
	case "branch_prediction":
		return ci.Features.EnableBranchPrediction
	case "advanced_prefetching":
		return ci.Features.EnableAdvancedPrefetching
	case "memory_bandwidth":
		return ci.Features.EnableMemoryBandwidth
	case "parallel_processing":
		return ci.Features.EnableParallelProcessing
	case "statistical_modeling":
		return ci.Features.EnableStatisticalModeling
	case "convergence_tracking":
		return ci.Features.EnableConvergenceTracking
	case "dynamic_behavior":
		return ci.Features.EnableDynamicBehavior
	default:
		return false
	}
}

// GetDescription returns a description of the current complexity level
func (ci *CPUInterface) GetDescription() string {
	switch ci.ComplexityLevel {
	case CPUMinimal:
		return "Minimal CPU simulation: Real-world modeling with essential features only. Fast performance (~5x), high accuracy (~90%)."
	case CPUBasic:
		return "Basic CPU simulation: Real-world modeling with core features (language, SIMD, thermal, boost). Good balance (~2x faster, ~95% accuracy)."
	case CPUAdvanced:
		return "Advanced CPU simulation: Real-world modeling with most features (NUMA, hyperthreading, branch prediction). Enhanced accuracy (~1.2x faster, ~98% accuracy)."
	case CPUMaximum:
		return "Maximum CPU simulation: Real-world modeling with all features including advanced prefetching. Highest accuracy (~99%), baseline performance."
	default:
		return "Unknown CPU complexity level"
	}
}

// GetEnabledFeatures returns a list of currently enabled CPU features
func (ci *CPUInterface) GetEnabledFeatures() []string {
	var enabled []string
	
	features := map[string]bool{
		"language_multipliers":   ci.Features.EnableLanguageMultipliers,
		"complexity_factors":     ci.Features.EnableComplexityFactors,
		"basic_caching":          ci.Features.EnableBasicCaching,
		"simd_vectorization":     ci.Features.EnableSIMDVectorization,
		"advanced_caching":       ci.Features.EnableAdvancedCaching,
		"thermal_modeling":       ci.Features.EnableThermalModeling,
		"boost_clocks":           ci.Features.EnableBoostClocks,
		"numa_topology":          ci.Features.EnableNUMATopology,
		"hyperthreading":         ci.Features.EnableHyperthreading,
		"branch_prediction":      ci.Features.EnableBranchPrediction,
		"advanced_prefetching":   ci.Features.EnableAdvancedPrefetching,
		"memory_bandwidth":       ci.Features.EnableMemoryBandwidth,
		"parallel_processing":    ci.Features.EnableParallelProcessing,
		"statistical_modeling":   ci.Features.EnableStatisticalModeling,
		"convergence_tracking":   ci.Features.EnableConvergenceTracking,
		"dynamic_behavior":       ci.Features.EnableDynamicBehavior,
	}
	
	for feature, isEnabled := range features {
		if isEnabled {
			enabled = append(enabled, feature)
		}
	}
	
	return enabled
}

// GetPerformanceImpact returns estimated performance impact
func (ci *CPUInterface) GetPerformanceImpact() string {
	switch ci.ComplexityLevel {
	case CPUMinimal:
		return "~5x faster than maximum, ~90% accuracy"
	case CPUBasic:
		return "~2x faster than maximum, ~95% accuracy"
	case CPUAdvanced:
		return "~1.2x faster than maximum, ~98% accuracy"
	case CPUMaximum:
		return "Baseline performance, ~99% accuracy"
	default:
		return "Unknown performance impact"
	}
}

// ValidateComplexityLevel checks if the complexity level is valid
func ValidateComplexityLevel(level CPUComplexityLevel) error {
	if level < CPUMinimal || level > CPUMaximum {
		return fmt.Errorf("invalid CPU complexity level: %d (must be between %d and %d)", 
			level, CPUMinimal, CPUMaximum)
	}
	return nil
}

// EnableFeature manually enables a specific feature (overrides complexity level setting)
func (ci *CPUInterface) EnableFeature(feature string) error {
	switch feature {
	case "language_multipliers":
		ci.Features.EnableLanguageMultipliers = true
	case "complexity_factors":
		ci.Features.EnableComplexityFactors = true
	case "basic_caching":
		ci.Features.EnableBasicCaching = true
	case "simd_vectorization":
		ci.Features.EnableSIMDVectorization = true
	case "advanced_caching":
		ci.Features.EnableAdvancedCaching = true
	case "thermal_modeling":
		ci.Features.EnableThermalModeling = true
	case "boost_clocks":
		ci.Features.EnableBoostClocks = true
	case "numa_topology":
		ci.Features.EnableNUMATopology = true
	case "hyperthreading":
		ci.Features.EnableHyperthreading = true
	case "branch_prediction":
		ci.Features.EnableBranchPrediction = true
	case "advanced_prefetching":
		ci.Features.EnableAdvancedPrefetching = true
	case "memory_bandwidth":
		ci.Features.EnableMemoryBandwidth = true
	case "parallel_processing":
		ci.Features.EnableParallelProcessing = true
	case "statistical_modeling":
		ci.Features.EnableStatisticalModeling = true
	case "convergence_tracking":
		ci.Features.EnableConvergenceTracking = true
	case "dynamic_behavior":
		ci.Features.EnableDynamicBehavior = true
	default:
		return fmt.Errorf("unknown CPU feature: %s", feature)
	}
	return nil
}

// DisableFeature manually disables a specific feature (overrides complexity level setting)
func (ci *CPUInterface) DisableFeature(feature string) error {
	switch feature {
	case "language_multipliers":
		ci.Features.EnableLanguageMultipliers = false
	case "complexity_factors":
		ci.Features.EnableComplexityFactors = false
	case "basic_caching":
		ci.Features.EnableBasicCaching = false
	case "simd_vectorization":
		ci.Features.EnableSIMDVectorization = false
	case "advanced_caching":
		ci.Features.EnableAdvancedCaching = false
	case "thermal_modeling":
		ci.Features.EnableThermalModeling = false
	case "boost_clocks":
		ci.Features.EnableBoostClocks = false
	case "numa_topology":
		ci.Features.EnableNUMATopology = false
	case "hyperthreading":
		ci.Features.EnableHyperthreading = false
	case "branch_prediction":
		ci.Features.EnableBranchPrediction = false
	case "advanced_prefetching":
		ci.Features.EnableAdvancedPrefetching = false
	case "memory_bandwidth":
		ci.Features.EnableMemoryBandwidth = false
	case "parallel_processing":
		ci.Features.EnableParallelProcessing = false
	case "statistical_modeling":
		ci.Features.EnableStatisticalModeling = false
	case "convergence_tracking":
		ci.Features.EnableConvergenceTracking = false
	case "dynamic_behavior":
		ci.Features.EnableDynamicBehavior = false
	default:
		return fmt.Errorf("unknown CPU feature: %s", feature)
	}
	return nil
}
