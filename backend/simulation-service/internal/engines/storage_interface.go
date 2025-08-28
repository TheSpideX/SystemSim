package engines

import (
	"fmt"
)

// StorageComplexityLevel defines the level of storage simulation complexity
// Using the same integer-based system as CPU/Memory for consistency
type StorageComplexityLevel = ComplexityLevel

// StorageFeatures defines which storage simulation features are enabled
type StorageFeatures struct {
	// Basic storage modeling (always enabled for Basic+)
	EnableIOPSLimits          bool `json:"enable_iops_limits"`           // Read/Write IOPS constraints
	EnableSequentialOptimization bool `json:"enable_sequential_optimization"` // Sequential vs random I/O patterns
	EnableQueueDepthManagement bool `json:"enable_queue_depth_management"` // NCQ/TCQ modeling
	EnableBasicWearLeveling   bool `json:"enable_basic_wear_leveling"`   // SSD wear leveling basics
	
	// Advanced storage modeling (Advanced+)
	EnableFileSystemOverhead  bool `json:"enable_filesystem_overhead"`   // Metadata operation costs
	EnableFragmentationEffects bool `json:"enable_fragmentation_effects"` // HDD fragmentation impact
	EnableControllerCache     bool `json:"enable_controller_cache"`      // Storage controller cache
	EnableTrimGarbageCollection bool `json:"enable_trim_garbage_collection"` // SSD maintenance operations
	EnableAdvancedWearLeveling bool `json:"enable_advanced_wear_leveling"` // Advanced SSD wear algorithms
	
	// Expert storage modeling (Maximum only)
	EnablePowerStateTransitions bool `json:"enable_power_state_transitions"` // HDD spin-up/down modeling
	EnableThermalThrottling    bool `json:"enable_thermal_throttling"`     // Temperature effects on performance
	EnableErrorCorrection      bool `json:"enable_error_correction"`       // ECC and bad block management
	EnableAdvancedPrefetching  bool `json:"enable_advanced_prefetching"`   // Storage controller prefetching
	EnableCompressionEffects   bool `json:"enable_compression_effects"`    // Data compression impact
	EnableEncryptionOverhead   bool `json:"enable_encryption_overhead"`    // Hardware encryption costs
	EnableMultiStreamIO       bool `json:"enable_multi_stream_io"`        // Multi-stream SSD optimization
	EnableZonedStorage        bool `json:"enable_zoned_storage"`          // ZNS SSD modeling
	
	// Statistical and behavioral modeling
	EnableStatisticalModeling bool `json:"enable_statistical_modeling"`   // Statistical convergence
	EnableConvergenceTracking bool `json:"enable_convergence_tracking"`   // Model convergence monitoring
	EnableDynamicBehavior     bool `json:"enable_dynamic_behavior"`       // Adaptive behavior patterns
	EnableRealtimeAdaptation  bool `json:"enable_realtime_adaptation"`    // Real-time model updates
}

// StorageInterface provides control over storage simulation complexity and features
type StorageInterface struct {
	ComplexityLevel StorageComplexityLevel `json:"complexity_level"`
	Features        *StorageFeatures       `json:"features"`
}

// NewStorageInterface creates a new storage interface with the specified complexity level
func NewStorageInterface(level StorageComplexityLevel) *StorageInterface {
	si := &StorageInterface{
		ComplexityLevel: level,
		Features:        &StorageFeatures{},
	}

	// SetComplexityLevel should not fail for valid levels, but handle error just in case
	if err := si.SetComplexityLevel(level); err != nil {
		// Fallback to Advanced if invalid level
		si.SetComplexityLevel(ComplexityAdvanced)
	}
	return si
}

// SetComplexityLevel changes the storage simulation complexity level
func (si *StorageInterface) SetComplexityLevel(level StorageComplexityLevel) error {
	if level < ComplexityMinimal || level > ComplexityMaximum {
		return fmt.Errorf("invalid complexity level: %d", level)
	}
	
	si.ComplexityLevel = level
	si.configureFeatures()
	return nil
}

// configureFeatures sets up features based on complexity level
func (si *StorageInterface) configureFeatures() {
	switch si.ComplexityLevel {
	case ComplexityMinimal:
		si.configureMinimalFeatures()
	case ComplexityBasic:
		si.configureBasicFeatures()
	case ComplexityAdvanced:
		si.configureAdvancedFeatures()
	case ComplexityMaximum:
		si.configureMaximumFeatures()
	}
}

// configureMinimalFeatures - Real-world modeling with essential features (~5x faster, ~90% accuracy)
func (si *StorageInterface) configureMinimalFeatures() {
	*si.Features = StorageFeatures{
		// Essential real-world features
		EnableIOPSLimits:          true,  // Essential for any storage simulation
		EnableSequentialOptimization: true,  // Pattern optimization is essential for real-world accuracy
		EnableQueueDepthManagement: true,  // Queue management is critical for modern storage
		EnableBasicWearLeveling:   true,  // Wear leveling affects real-world performance
		
		// Skip all advanced features
		EnableFileSystemOverhead:  false,
		EnableFragmentationEffects: false,
		EnableControllerCache:     false,
		EnableTrimGarbageCollection: false,
		EnableAdvancedWearLeveling: false,
		
		// Skip all expert features
		EnablePowerStateTransitions: false,
		EnableThermalThrottling:    false,
		EnableErrorCorrection:      false,
		EnableAdvancedPrefetching:  false,
		EnableCompressionEffects:   false,
		EnableEncryptionOverhead:   false,
		EnableMultiStreamIO:       false,
		EnableZonedStorage:        false,
		
		// Minimal behavioral modeling
		EnableStatisticalModeling: false,
		EnableConvergenceTracking: false,
		EnableDynamicBehavior:     false,
		EnableRealtimeAdaptation:  false,
	}
}

// configureBasicFeatures - Real-world modeling with core features (~2x faster, ~95% accuracy)
func (si *StorageInterface) configureBasicFeatures() {
	*si.Features = StorageFeatures{
		// Core real-world storage features
		EnableIOPSLimits:          true,  // IOPS constraints
		EnableSequentialOptimization: true,  // Sequential vs random patterns
		EnableQueueDepthManagement: true,  // Queue management is essential
		EnableBasicWearLeveling:   true,  // SSD wear leveling

		// Important real-world features
		EnableFileSystemOverhead:  true,  // Filesystem metadata costs
		EnableFragmentationEffects: true,  // Fragmentation affects real-world performance
		EnableControllerCache:     true,  // Controller cache benefits
		EnableTrimGarbageCollection: true,  // TRIM/GC is essential for SSD accuracy
		EnableAdvancedWearLeveling: true,  // Advanced wear algorithms are standard
		
		// Skip expert features
		EnablePowerStateTransitions: false,
		EnableThermalThrottling:    false,
		EnableErrorCorrection:      false,
		EnableAdvancedPrefetching:  false,
		EnableCompressionEffects:   false,
		EnableEncryptionOverhead:   false,
		EnableMultiStreamIO:       false,
		EnableZonedStorage:        false,
		
		// Basic behavioral modeling
		EnableStatisticalModeling: true,
		EnableConvergenceTracking: false,
		EnableDynamicBehavior:     true,
		EnableRealtimeAdaptation:  false,
	}
}

// configureAdvancedFeatures - Enhanced real-world modeling (~1.2x faster, ~98% accuracy)
func (si *StorageInterface) configureAdvancedFeatures() {
	*si.Features = StorageFeatures{
		// All basic real-world features
		EnableIOPSLimits:          true,
		EnableSequentialOptimization: true,
		EnableQueueDepthManagement: true,
		EnableBasicWearLeveling:   true,

		// Enhanced real-world features
		EnableFileSystemOverhead:  true,
		EnableFragmentationEffects: true,  // HDD fragmentation modeling
		EnableControllerCache:     true,
		EnableTrimGarbageCollection: true,  // SSD maintenance
		EnableAdvancedWearLeveling: true,  // Advanced wear algorithms

		// Enhanced expert features
		EnablePowerStateTransitions: true,  // Power state modeling
		EnableThermalThrottling:    true,  // Thermal effects
		EnableErrorCorrection:      false, // Skip complex ECC modeling
		EnableAdvancedPrefetching:  false, // Skip most expensive feature
		EnableCompressionEffects:   true,  // Data compression impact
		EnableEncryptionOverhead:   false, // Skip encryption complexity
		EnableMultiStreamIO:       false, // Skip multi-stream complexity
		EnableZonedStorage:        false, // Skip ZNS complexity
		
		// Advanced behavioral modeling
		EnableStatisticalModeling: true,
		EnableConvergenceTracking: true,
		EnableDynamicBehavior:     true,
		EnableRealtimeAdaptation:  false,
	}
}

// configureMaximumFeatures - Maximum accuracy (~99% accuracy, baseline performance)
func (si *StorageInterface) configureMaximumFeatures() {
	*si.Features = StorageFeatures{
		// Everything enabled for maximum realism
		EnableIOPSLimits:          true,
		EnableSequentialOptimization: true,
		EnableQueueDepthManagement: true,
		EnableBasicWearLeveling:   true,
		EnableFileSystemOverhead:  true,
		EnableFragmentationEffects: true,
		EnableControllerCache:     true,
		EnableTrimGarbageCollection: true,
		EnableAdvancedWearLeveling: true,
		EnablePowerStateTransitions: true,
		EnableThermalThrottling:    true,
		EnableErrorCorrection:      true,  // Full ECC modeling
		EnableAdvancedPrefetching:  true,  // Most computationally expensive feature
		EnableCompressionEffects:   true,
		EnableEncryptionOverhead:   true,  // Hardware encryption costs
		EnableMultiStreamIO:       true,  // Multi-stream SSD optimization
		EnableZonedStorage:        true,  // ZNS SSD modeling

		// Full behavioral modeling
		EnableStatisticalModeling: true,
		EnableConvergenceTracking: true,
		EnableDynamicBehavior:     true,
		EnableRealtimeAdaptation:  true,
	}
}

// ShouldEnableFeature checks if a specific feature should be enabled
func (si *StorageInterface) ShouldEnableFeature(featureName string) bool {
	switch featureName {
	case "iops_limits":
		return si.Features.EnableIOPSLimits
	case "sequential_optimization":
		return si.Features.EnableSequentialOptimization
	case "queue_depth_management":
		return si.Features.EnableQueueDepthManagement
	case "basic_wear_leveling":
		return si.Features.EnableBasicWearLeveling
	case "filesystem_overhead":
		return si.Features.EnableFileSystemOverhead
	case "fragmentation_effects":
		return si.Features.EnableFragmentationEffects
	case "controller_cache":
		return si.Features.EnableControllerCache
	case "trim_garbage_collection":
		return si.Features.EnableTrimGarbageCollection
	case "advanced_wear_leveling":
		return si.Features.EnableAdvancedWearLeveling
	case "power_state_transitions":
		return si.Features.EnablePowerStateTransitions
	case "thermal_throttling":
		return si.Features.EnableThermalThrottling
	case "error_correction":
		return si.Features.EnableErrorCorrection
	case "advanced_prefetching":
		return si.Features.EnableAdvancedPrefetching
	case "compression_effects":
		return si.Features.EnableCompressionEffects
	case "encryption_overhead":
		return si.Features.EnableEncryptionOverhead
	case "multi_stream_io":
		return si.Features.EnableMultiStreamIO
	case "zoned_storage":
		return si.Features.EnableZonedStorage
	case "statistical_modeling":
		return si.Features.EnableStatisticalModeling
	case "convergence_tracking":
		return si.Features.EnableConvergenceTracking
	case "dynamic_behavior":
		return si.Features.EnableDynamicBehavior
	case "realtime_adaptation":
		return si.Features.EnableRealtimeAdaptation
	default:
		return false
	}
}

// GetDescription returns a description of the current complexity level
func (si *StorageInterface) GetDescription() string {
	switch si.ComplexityLevel {
	case ComplexityMinimal:
		return "Minimal storage simulation: Real-world modeling with essential IOPS limits only. Fast performance (~5x), high accuracy (~90%)."
	case ComplexityBasic:
		return "Basic storage simulation: Real-world modeling with core features (IOPS, patterns, queue depth, wear). Good balance (~2x faster, ~95% accuracy)."
	case ComplexityAdvanced:
		return "Advanced storage simulation: Real-world modeling with most features (fragmentation, thermal, power states). Enhanced accuracy (~1.2x faster, ~98% accuracy)."
	case ComplexityMaximum:
		return "Maximum storage simulation: Real-world modeling with all features including advanced prefetching and ZNS. Highest accuracy (~99%), baseline performance."
	default:
		return "Unknown complexity level"
	}
}

// GetEnabledFeatures returns a list of currently enabled features
func (si *StorageInterface) GetEnabledFeatures() []string {
	features := make([]string, 0)
	
	if si.Features.EnableIOPSLimits {
		features = append(features, "IOPS Limits")
	}
	if si.Features.EnableSequentialOptimization {
		features = append(features, "Sequential Optimization")
	}
	if si.Features.EnableQueueDepthManagement {
		features = append(features, "Queue Depth Management")
	}
	if si.Features.EnableBasicWearLeveling {
		features = append(features, "Basic Wear Leveling")
	}
	if si.Features.EnableFileSystemOverhead {
		features = append(features, "Filesystem Overhead")
	}
	if si.Features.EnableFragmentationEffects {
		features = append(features, "Fragmentation Effects")
	}
	if si.Features.EnableControllerCache {
		features = append(features, "Controller Cache")
	}
	if si.Features.EnableTrimGarbageCollection {
		features = append(features, "TRIM/Garbage Collection")
	}
	if si.Features.EnableAdvancedWearLeveling {
		features = append(features, "Advanced Wear Leveling")
	}
	if si.Features.EnablePowerStateTransitions {
		features = append(features, "Power State Transitions")
	}
	if si.Features.EnableThermalThrottling {
		features = append(features, "Thermal Throttling")
	}
	if si.Features.EnableErrorCorrection {
		features = append(features, "Error Correction")
	}
	if si.Features.EnableAdvancedPrefetching {
		features = append(features, "Advanced Prefetching")
	}
	if si.Features.EnableCompressionEffects {
		features = append(features, "Compression Effects")
	}
	if si.Features.EnableEncryptionOverhead {
		features = append(features, "Encryption Overhead")
	}
	if si.Features.EnableMultiStreamIO {
		features = append(features, "Multi-Stream I/O")
	}
	if si.Features.EnableZonedStorage {
		features = append(features, "Zoned Storage")
	}
	
	return features
}
