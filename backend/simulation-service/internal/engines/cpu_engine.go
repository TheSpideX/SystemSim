package engines

import (
	"container/heap"
	"fmt"
	"math"
	"time"
)

// ProcessingOperation represents an operation currently being processed
type ProcessingOperation struct {
	Operation      *Operation `json:"operation"`
	StartTick      int64      `json:"start_tick"`
	CompletionTick int64      `json:"completion_tick"`
	CoresUsed      int        `json:"cores_used"`
}

// ProcessingHeap implements heap.Interface for ProcessingOperation
type ProcessingHeap []ProcessingOperation

func (h ProcessingHeap) Len() int           { return len(h) }
func (h ProcessingHeap) Less(i, j int) bool { return h[i].CompletionTick < h[j].CompletionTick }
func (h ProcessingHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *ProcessingHeap) Push(x interface{}) {
	*h = append(*h, x.(ProcessingOperation))
}

func (h *ProcessingHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// CPUEngine implements the BaseEngine interface for CPU operations
type CPUEngine struct {
	*CommonEngine

	// Complexity control interface
	ComplexityInterface *CPUInterface `json:"complexity_interface"`

	// CPU-specific properties from profile
	CoreCount       int     `json:"core_count"`
	BaseClockGHz    float64 `json:"base_clock_ghz"`
	BoostClockGHz   float64 `json:"boost_clock_ghz"`
	ThermalLimitC   float64 `json:"thermal_limit_c"`
	TDP             float64 `json:"tdp"`
	CacheL1KB       int     `json:"cache_l1_kb"`
	CacheL2KB       int     `json:"cache_l2_kb"`
	CacheL3MB       int     `json:"cache_l3_mb"`
	
	// Dynamic thermal state (real-time adaptation)
	ThermalState struct {
		CurrentTemperatureC  float64 `json:"current_temperature_c"`
		HeatAccumulation     float64 `json:"heat_accumulation"`
		ThrottleActive       bool    `json:"throttle_active"`
		ThrottleFactor       float64 `json:"throttle_factor"`
		CoolingCapacity      float64 `json:"cooling_capacity"`
		AmbientTemperatureC  float64 `json:"ambient_temperature_c"`
		LastThermalUpdate    int64   `json:"last_thermal_update"`
		AccumulatedWorkHeat  float64 `json:"accumulated_work_heat"` // Heat from actual work performed
	} `json:"thermal_state"`
	
	// Dynamic cache behavior (statistical convergence with full hierarchy)
	CacheState struct {
		L1HitRatio          float64 `json:"l1_hit_ratio"`
		L2HitRatio          float64 `json:"l2_hit_ratio"`
		L3HitRatio          float64 `json:"l3_hit_ratio"`
		WorkingSetSize      int64   `json:"working_set_size"`
		CacheWarming        bool    `json:"cache_warming"`
		WarmupOperations    int64   `json:"warmup_operations"`
		ConvergedHitRatio   float64 `json:"converged_hit_ratio"`
		AccessPatternHistory []int64 `json:"access_pattern_history"`

		// Cache hierarchy configuration (from profile)
		L1HitRatioTarget        float64   `json:"l1_hit_ratio_target"`
		L2HitRatioTarget        float64   `json:"l2_hit_ratio_target"`
		L3HitRatioTarget        float64   `json:"l3_hit_ratio_target"`
		CacheLineSize           int       `json:"cache_line_size"`
		PrefetchEfficiency      float64   `json:"prefetch_efficiency"`



		// Hardware-dependent cache multipliers (from profile)
		L1HitMultiplier         float64   `json:"l1_hit_multiplier"`
		L2HitMultiplier         float64   `json:"l2_hit_multiplier"`
		L3HitMultiplier         float64   `json:"l3_hit_multiplier"`
		MemoryAccessMultiplier  float64   `json:"memory_access_multiplier"`
	} `json:"cache_state"`
	
	// Core utilization tracking
	ActiveCores     int     `json:"active_cores"`
	CoreUtilization []float64 `json:"core_utilization"`

	// NEW: Processing state tracking for realistic utilization
	ActiveOperations *ProcessingHeap `json:"active_operations"`
	BusyCores        int             `json:"busy_cores"`

	// Boost clock state (dynamic frequency scaling)
	BoostState struct {
		CurrentClockGHz    float64 `json:"current_clock_ghz"`
		BoostActive        bool    `json:"boost_active"`
		BoostStartTick     int64   `json:"boost_start_tick"`
		SingleCoreBoostGHz float64 `json:"single_core_boost_ghz"`
		AllCoreBoostGHz    float64 `json:"all_core_boost_ghz"`
		BoostDurationTicks int64   `json:"boost_duration_ticks"`
		ThermalDependent   bool    `json:"thermal_dependent"`
	} `json:"boost_state"`

	// NUMA topology state
	NUMAState struct {
		NumaNodes          int     `json:"numa_nodes"`
		CrossSocketPenalty float64 `json:"cross_socket_penalty"`
		LocalMemoryRatio   float64 `json:"local_memory_ratio"`
		MemoryBandwidthMBs int64   `json:"memory_bandwidth_mbs"`
	} `json:"numa_state"`

	// Hyperthreading state
	HyperthreadingState struct {
		Enabled           bool    `json:"enabled"`
		ThreadsPerCore    int     `json:"threads_per_core"`
		EfficiencyFactor  float64 `json:"efficiency_factor"`
		EffectiveCores    int     `json:"effective_cores"`
	} `json:"hyperthreading_state"`

	// SIMD/Vectorization state (statistical modeling)
	VectorizationState struct {
		SupportedInstructions []string `json:"supported_instructions"` // ["SSE4.2", "AVX2", "AVX512"]
		VectorWidth          int      `json:"vector_width"`           // 128, 256, 512 bits
		SIMDEfficiency       float64  `json:"simd_efficiency"`        // 0.8-0.95
		VectorizationRatio   float64  `json:"vectorization_ratio"`    // Current operation vectorization ratio

		// Statistical tracking
		VectorOperationsCount int64   `json:"vector_operations_count"`
		ScalarOperationsCount int64   `json:"scalar_operations_count"`
		AverageSpeedup       float64  `json:"average_speedup"`

		// Profile-based vectorization probabilities
		OperationVectorizability map[string]float64 `json:"operation_vectorizability"`
	} `json:"vectorization_state"`

	// Language performance multipliers (from real benchmarks)
	LanguageMultipliers map[string]float64 `json:"language_multipliers"`
	
	// Algorithm complexity factors
	ComplexityFactors map[string]float64 `json:"complexity_factors"`

	// Branch prediction state (statistical modeling)
	BranchPredictionState struct {
		BaseAccuracy           float64 `json:"base_accuracy"`
		RandomPatternAccuracy  float64 `json:"random_pattern_accuracy"`
		LoopPatternAccuracy    float64 `json:"loop_pattern_accuracy"`
		CallReturnAccuracy     float64 `json:"call_return_accuracy"`
		MispredictionPenalty   float64 `json:"misprediction_penalty"`
		PipelineDepth          int     `json:"pipeline_depth"`
		TotalBranches          int64   `json:"total_branches"`
		TotalMispredictions    int64   `json:"total_mispredictions"`
	} `json:"branch_prediction_state"`

	// Memory bandwidth contention state (multi-core modeling)
	MemoryBandwidthState struct {
		TotalBandwidthGBps         float64 `json:"total_bandwidth_gbps"`
		PerCoreDegradation         float64 `json:"per_core_degradation"`
		ContentionThreshold        int     `json:"contention_threshold"`
		SevereContentionProbability float64 `json:"severe_contention_probability"`
		SevereContentionPenalty    float64 `json:"severe_contention_penalty"`
		CurrentBandwidthUtilization float64 `json:"current_bandwidth_utilization"`
	} `json:"memory_bandwidth_state"`

	// Advanced prefetch state (hardware prefetcher modeling)
	AdvancedPrefetchState struct {
		HardwarePrefetchers  int     `json:"hardware_prefetchers"`
		SequentialAccuracy   float64 `json:"sequential_accuracy"`
		StrideAccuracy       float64 `json:"stride_accuracy"`
		PatternAccuracy      float64 `json:"pattern_accuracy"`
		PrefetchDistance     int     `json:"prefetch_distance"`
		BandwidthUsage       float64 `json:"bandwidth_usage"`
		AccessPatternHistory []int64 `json:"access_pattern_history"`
	} `json:"advanced_prefetch_state"`

	// Parallel processing state (multi-core speedup modeling)
	ParallelProcessingState struct {
		Enabled                bool                `json:"enabled"`
		MaxParallelizableRatio float64             `json:"max_parallelizable_ratio"`
		ParallelizabilityMap   map[string]float64  `json:"parallelizability_map"`
		EfficiencyCurve        map[string]float64  `json:"efficiency_curve"`
		OverheadPerCore        float64             `json:"overhead_per_core"`
		SynchronizationOverhead float64            `json:"synchronization_overhead"`
	} `json:"parallel_processing_state"`
}

// NewCPUEngine creates a new CPU engine with default values (will be overridden by profile)
func NewCPUEngine(queueCapacity int) *CPUEngine {
	common := NewCommonEngine(CPUEngineType, queueCapacity)

	cpu := &CPUEngine{
		CommonEngine:        common,
		ComplexityInterface: NewCPUInterface(CPUAdvanced), // Default to advanced complexity
		CoreCount:           1,    // Default, will be set by profile
		BaseClockGHz:        2.0,  // Default, will be set by profile
		BoostClockGHz:       2.5,  // Default, will be set by profile
		ThermalLimitC:       70.0, // Default, will be set by profile
		TDP:                 65.0, // Default, will be set by profile
		CacheL1KB:           32,   // Default, will be set by profile
		CacheL2KB:           256,  // Default, will be set by profile
		CacheL3MB:           8,    // Default, will be set by profile
		ActiveCores:         0,
		CoreUtilization:     make([]float64, 1), // Will be resized by profile
		BusyCores:          0,
		LanguageMultipliers: make(map[string]float64),
		ComplexityFactors:   make(map[string]float64),
	}

	// Initialize processing heap
	cpu.ActiveOperations = &ProcessingHeap{}
	heap.Init(cpu.ActiveOperations)

	// Initialize thermal state with safe defaults (profile will override these)
	cpu.ThermalState.CurrentTemperatureC = 22.0 // Safe default ambient temperature
	cpu.ThermalState.HeatAccumulation = 0.0
	cpu.ThermalState.ThrottleActive = false
	cpu.ThermalState.ThrottleFactor = 1.0
	cpu.ThermalState.CoolingCapacity = 0.0 // Will be set from profile during LoadProfile()
	cpu.ThermalState.AmbientTemperatureC = 22.0 // Safe default
	cpu.ThermalState.LastThermalUpdate = 0

	// Initialize cache state with minimal defaults (profile will set proper values)
	cpu.CacheState.WorkingSetSize = 0
	cpu.CacheState.CacheWarming = true
	cpu.CacheState.WarmupOperations = 0
	cpu.CacheState.AccessPatternHistory = make([]int64, 0, 1000)
	// Cache hit ratios and converged ratio will be set by profile loading

	// NOTE: Profile-dependent initialization moved to LoadProfile() method
	// This ensures proper initialization order and prevents accessing nil profile
	cpu.initializeHyperthreadingState()

	// Initialize cache hierarchy from profile
	cpu.initializeCacheHierarchy()

	// Initialize advanced features with Intel Xeon defaults
	cpu.initializeAdvancedFeatures()

	return cpu
}

// ProcessOperation processes a single CPU operation with dynamic behavior
func (cpu *CPUEngine) ProcessOperation(op *Operation, currentTick int64) *OperationResult {
	cpu.CurrentTick = currentTick
	
	// Calculate base processing time from profile and operation
	baseTime := cpu.calculateBaseProcessingTime(op)
	
	// Apply language performance multiplier (if enabled)
	languageTime := baseTime
	if cpu.ComplexityInterface.ShouldEnableFeature("language_multipliers") {
		languageTime = cpu.applyLanguageMultiplier(baseTime, op.Language)
	}

	// Apply algorithm complexity factor (if enabled)
	complexityTime := languageTime
	if cpu.ComplexityInterface.ShouldEnableFeature("complexity_factors") {
		complexityTime = cpu.applyComplexityFactor(languageTime, op.Complexity, op.DataSize)
	}

	// Apply SIMD/Vectorization effects (if enabled)
	vectorizedTime := complexityTime
	if cpu.ComplexityInterface.ShouldEnableFeature("simd_vectorization") {
		vectorizedTime = cpu.applyVectorizationEffects(complexityTime, op)
	}

	// Apply cache behavior (basic or advanced based on settings)
	cacheAdjustedTime := vectorizedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("basic_caching") ||
	   cpu.ComplexityInterface.ShouldEnableFeature("advanced_caching") {
		cacheAdjustedTime = cpu.applyCacheBehavior(vectorizedTime, op)
	}

	// Apply advanced prefetching effects (if enabled)
	prefetchAdjustedTime := cacheAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("advanced_prefetching") {
		prefetchAdjustedTime = cpu.applyAdvancedPrefetching(cacheAdjustedTime, op)
	}

	// Apply branch prediction effects (if enabled)
	branchAdjustedTime := prefetchAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("branch_prediction") {
		branchAdjustedTime = cpu.applyBranchPrediction(prefetchAdjustedTime, op)
	}

	// Calculate optimal cores and apply parallel processing (if enabled)
	coresAllocated := 1 // Default to single core
	parallelAdjustedTime := branchAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("parallel_processing") {
		optimalCores := cpu.calculateCoresNeeded(op)
		coresAllocated = int(math.Min(float64(optimalCores), float64(cpu.CoreCount)))
		parallelAdjustedTime = cpu.applyParallelProcessingSpeedup(branchAdjustedTime, op, coresAllocated)
	}

	// Update ActiveCores to reflect current operation (for monitoring/metrics)
	cpu.ActiveCores = coresAllocated

	// Apply boost clock effects (if enabled)
	boostAdjustedTime := parallelAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("boost_clocks") {
		boostAdjustedTime = cpu.applyBoostClockEffects(parallelAdjustedTime)
	}

	// Apply NUMA topology effects (if enabled)
	numaAdjustedTime := boostAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("numa_topology") {
		numaAdjustedTime = cpu.applyNUMAEffects(boostAdjustedTime, op)
	}

	// Apply memory bandwidth contention (if enabled)
	bandwidthAdjustedTime := numaAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("memory_bandwidth") {
		bandwidthAdjustedTime = cpu.applyMemoryBandwidthContention(numaAdjustedTime, op)
	}

	// Apply thermal effects (if enabled)
	thermalAdjustedTime := bandwidthAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("thermal_modeling") {
		thermalAdjustedTime = cpu.applyThermalEffects(bandwidthAdjustedTime)
	}

	// Add heat based on actual work performed (realistic thermal modeling)
	cpu.addWorkBasedHeat(op, thermalAdjustedTime)

	// Update thermal state based on accumulated work heat
	cpu.updateThermalState()

	// Apply common performance factors (load, queue, health, variance)
	utilization := cpu.calculateCurrentUtilization()
	finalTime := cpu.ApplyCommonPerformanceFactors(thermalAdjustedTime, utilization)

	// Update remaining dynamic state
	cpu.updateRemainingDynamicState(op, finalTime)
	
	// Calculate penalty factors for routing decisions
	loadPenalty := 1.0 + (utilization * 0.5) // Higher utilization = higher penalty
	queuePenalty := 1.0 + (float64(cpu.GetQueueLength()) / float64(cpu.GetQueueCapacity()) * 0.3)
	thermalPenalty := 1.0 / cpu.ThermalState.ThrottleFactor // Throttling increases penalty
	contentionPenalty := cpu.getMemoryContentionFactor(1) // Memory contention
	healthPenalty := 1.0 + (1.0 - cpu.GetHealth().Score) * 0.2 // Health issues increase penalty

	totalPenaltyFactor := loadPenalty * queuePenalty * thermalPenalty * contentionPenalty * healthPenalty

	// Determine performance grade
	performanceGrade := "A"
	recommendedAction := "continue"
	if totalPenaltyFactor > 2.0 {
		performanceGrade = "F"
		recommendedAction = "redirect"
	} else if totalPenaltyFactor > 1.5 {
		performanceGrade = "D"
		recommendedAction = "throttle"
	} else if totalPenaltyFactor > 1.3 {
		performanceGrade = "C"
		recommendedAction = "throttle"
	} else if totalPenaltyFactor > 1.1 {
		performanceGrade = "B"
	}

	// Create result with penalty information
	result := &OperationResult{
		OperationID:    op.ID,
		OperationType:  op.Type, // For routing decisions
		ProcessingTime: finalTime,
		CompletedTick:  currentTick + cpu.DurationToTicks(finalTime),
		Success:        true,
		PenaltyInfo: &PenaltyInformation{
			EngineType:           CPUEngineType,
			EngineID:            cpu.ID,
			BaseProcessingTime:   baseTime,
			ActualProcessingTime: finalTime,
			LoadPenalty:         loadPenalty,
			QueuePenalty:        queuePenalty,
			ThermalPenalty:      thermalPenalty,
			ContentionPenalty:   contentionPenalty,
			HealthPenalty:       healthPenalty,
			TotalPenaltyFactor:  totalPenaltyFactor,
			PerformanceGrade:    performanceGrade,
			RecommendedAction:   recommendedAction,
			CPUPenalties: &CPUPenaltyDetails{
				CacheHitRatio:      cpu.CacheState.L1HitRatio,
				VectorizationRatio: cpu.getOperationVectorizationRatio(op),
				ThermalThrottling:  cpu.ThermalState.ThrottleFactor,
				CoreUtilization:    utilization,
				MemoryContention:   contentionPenalty,
			},
		},
		Metrics: map[string]interface{}{
			"base_time_ms":         float64(baseTime) / float64(time.Millisecond),
			"language_factor":      cpu.getLanguageMultiplier(op.Language),
			"complexity_factor":    cpu.getComplexityFactor(op.Complexity),
			"vectorization_ratio":  cpu.getOperationVectorizationRatio(op),
			"vector_speedup":       cpu.VectorizationState.AverageSpeedup,
			"vector_width":         cpu.VectorizationState.VectorWidth,
			"cache_hit_ratio":      cpu.CacheState.L1HitRatio,
			"thermal_factor":       cpu.ThermalState.ThrottleFactor,
			"utilization":          utilization,
			"active_cores":         cpu.ActiveCores,
			"temperature_c":        cpu.ThermalState.CurrentTemperatureC,
		},
	}
	
	// Update operation history for convergence
	cpu.AddOperationToHistory(finalTime)
	cpu.CompletedOps++
	
	return result
}

// ProcessTick processes one simulation tick with proper state tracking
func (cpu *CPUEngine) ProcessTick(currentTick int64) []OperationResult {
	cpu.CurrentTick = currentTick
	results := make([]OperationResult, 0)

	// STEP 1: Check for completed operations and move to output
	completedOps := cpu.checkCompletedOperations(currentTick)
	results = append(results, completedOps...)

	// STEP 2: Start new operations from input queue
	cpu.startNewOperationsFromQueue(currentTick)

	// STEP 3: Update metrics based on actual busy state
	cpu.updateThermalState()
	cpu.UpdateHealth()
	cpu.UpdateDynamicBehavior()

	return results
}

// checkCompletedOperations checks for operations that completed this tick
func (cpu *CPUEngine) checkCompletedOperations(currentTick int64) []OperationResult {
	completed := make([]OperationResult, 0)

	// Check heap root for completed operations
	for cpu.ActiveOperations.Len() > 0 {
		// Peek at the next completion without removing it
		nextOp := (*cpu.ActiveOperations)[0]

		if nextOp.CompletionTick <= currentTick {
			// INTRA-ENGINE FLOW: Operation completed, try to move to output
			// For now, we'll always move to output (wrapper handles output queue full)
			// In future, we could implement output queue checking here

			// Operation completed - remove from heap
			completedOp := heap.Pop(cpu.ActiveOperations).(ProcessingOperation)

			// Create result
			result := OperationResult{
				OperationID:    completedOp.Operation.ID,
				OperationType:  completedOp.Operation.Type, // For routing decisions
				CompletedAt:    currentTick,
				CompletedTick:  currentTick,
				ProcessingTime: time.Duration(completedOp.CompletionTick - completedOp.StartTick) * cpu.TickDuration,
				Success:        true,
				NextComponent:  completedOp.Operation.NextComponent, // For routing
			}
			completed = append(completed, result)

			// Free the cores (cores are released when operation completes)
			cpu.BusyCores -= completedOp.CoresUsed
		} else {
			// No more completed operations this tick
			break
		}
	}

	return completed
}

// startNewOperationsFromQueue starts new operations from input queue
func (cpu *CPUEngine) startNewOperationsFromQueue(currentTick int64) {
	maxQueuedOpsPerTick := cpu.getProfileInt("queue_processing", "max_ops_per_tick", 3)
	opsStartedThisTick := 0

	for cpu.GetQueueLength() > 0 && opsStartedThisTick < maxQueuedOpsPerTick {
		// INTRA-ENGINE FLOW: Check heap length before accepting operations
		if cpu.ActiveOperations.Len() >= cpu.getMaxInternalQueueSize() {
			// Heap is full - cannot accept new operations (realistic constraint)
			break
		}

		// Check if we have available cores
		availableCores := cpu.CoreCount - cpu.BusyCores
		if availableCores <= 0 {
			break // No cores available
		}

		queuedOp := cpu.DequeueOperation()
		if queuedOp == nil {
			break
		}

		// Calculate processing requirements
		coresNeeded := cpu.calculateCoresNeeded(queuedOp.Operation)

		if coresNeeded <= availableCores {
			// Start processing
			cpu.startProcessing(queuedOp, currentTick)
			opsStartedThisTick++
		} else {
			// Not enough cores - put back in queue
			cpu.QueueOperation(queuedOp.Operation)
			break
		}
	}
}

// startProcessing begins processing an operation
func (cpu *CPUEngine) startProcessing(queuedOp *QueuedOperation, currentTick int64) {
	// Calculate processing time using existing logic
	processingTime := cpu.calculateProcessingTimeForOperation(queuedOp.Operation)
	coresNeeded := cpu.calculateCoresNeeded(queuedOp.Operation)
	completionTick := currentTick + cpu.DurationToTicks(processingTime)

	// Create processing operation
	activeOp := ProcessingOperation{
		Operation:      queuedOp.Operation,
		StartTick:      currentTick,
		CompletionTick: completionTick,
		CoresUsed:      coresNeeded,
	}

	// Add to processing heap
	heap.Push(cpu.ActiveOperations, activeOp)
	cpu.BusyCores += coresNeeded

	// Generate heat for starting this operation
	cpu.addWorkBasedHeat(queuedOp.Operation, processingTime)
}

// Note: freeCoresBasedOnLoad function removed - cores are now allocated per operation
// This eliminates the conflicting allocation/freeing systems that caused BUG-004

// calculateBaseProcessingTime calculates base processing time from profile
func (cpu *CPUEngine) calculateBaseProcessingTime(op *Operation) time.Duration {
	// Get base time from profile with realistic fallback
	baseTimeMs := cpu.getProfileFloat("baseline_performance", "base_processing_time", 1.0) // 1ms realistic fallback

	// Validate base time is reasonable (0.01ms to 100ms)
	if baseTimeMs < 0.01 || baseTimeMs > 100.0 {
		baseTimeMs = 1.0 // Reset to safe default if unrealistic
	}
	
	// Apply clock speed normalization (higher clock = faster processing = lower time)
	// Get normalization baseline from profile with realistic fallback
	normalizationBaseline := cpu.getProfileFloat("baseline_performance", "clock_normalization_baseline", 3.0)

	// Validate normalization baseline is reasonable (1.0 to 6.0 GHz)
	if normalizationBaseline < 1.0 || normalizationBaseline > 6.0 {
		normalizationBaseline = 3.0 // Reset to safe default
	}

	// Validate BaseClockGHz is reasonable to prevent division by zero and unrealistic values
	if cpu.BaseClockGHz <= 0.5 || cpu.BaseClockGHz > 6.0 {
		cpu.BaseClockGHz = 2.5 // Realistic modern CPU base clock
	}

	// Clock normalization: normalize to baseline, then adjust for actual clock
	// Formula: time = baseTime * (baseline / actualClock)
	// Higher clock speed → lower clockFactor → faster processing (lower time)
	clockFactor := normalizationBaseline / cpu.BaseClockGHz
	adjustedTimeMs := baseTimeMs * clockFactor

	// Final validation: ensure result is reasonable (0.001ms to 1000ms)
	if adjustedTimeMs < 0.001 || adjustedTimeMs > 1000.0 {
		adjustedTimeMs = 1.0 // Reset to safe default if calculation produced unrealistic result
	}

	return time.Duration(adjustedTimeMs * float64(time.Millisecond))
}



// applyLanguageMultiplier applies language-specific performance multiplier
// Note: Profile values represent "performance factors" where higher values = faster execution
// Examples: C++ = 1.3 (30% faster), Go = 1.0 (baseline), Python = 0.3 (3.3x slower)
// Formula: time = baseTime / performanceFactor
func (cpu *CPUEngine) applyLanguageMultiplier(baseTime time.Duration, language string) time.Duration {
	performanceFactor := cpu.getLanguageMultiplier(language)
	return time.Duration(float64(baseTime) / performanceFactor)
}

// applyComplexityFactor applies algorithm complexity factor with realistic scaling
func (cpu *CPUEngine) applyComplexityFactor(baseTime time.Duration, complexity string, dataSize int64) time.Duration {
	complexityFactor := cpu.getComplexityFactor(complexity)

	// Validate data size is reasonable (0 to 1TB)
	if dataSize < 0 || dataSize > 1024*1024*1024*1024 {
		fmt.Printf("Warning: Extreme data size %d bytes for complexity %s, capping scaling\n", dataSize, complexity)
		dataSize = int64(math.Min(float64(dataSize), 1024*1024*1024*1024)) // Cap at 1TB
		dataSize = int64(math.Max(0, float64(dataSize)))
	}

	// For O(n) and higher complexities, scale with data size but cap at realistic levels
	if complexity == ComplexityON || complexity == ComplexityONLogN || complexity == ComplexityON2 {
		// Use logarithmic scaling instead of linear to prevent extreme values
		// Real-world algorithms have optimizations and practical limits
		sizeKB := math.Max(1.0, float64(dataSize)/1024.0)

		switch complexity {
		case ComplexityON:
			// O(n): Linear scaling from profile
			logFactor := cpu.getProfileFloat("algorithm_complexity", "size_scaling.O(n).log_factor", 1.0)
			maxFactor := cpu.getProfileFloat("algorithm_complexity", "size_scaling.O(n).max_factor", 10.0)
			sizeFactor := 1.0 + math.Log10(sizeKB) * logFactor
			complexityFactor *= math.Min(sizeFactor, maxFactor)
		case ComplexityONLogN:
			// O(n log n): From profile
			logFactor := cpu.getProfileFloat("algorithm_complexity", "size_scaling.O(n log n).log_factor", 1.2)
			maxFactor := cpu.getProfileFloat("algorithm_complexity", "size_scaling.O(n log n).max_factor", 15.0)
			sizeFactor := 1.0 + math.Log10(sizeKB) * logFactor
			complexityFactor *= math.Min(sizeFactor, maxFactor)
		case ComplexityON2:
			// O(n²): From profile
			logFactor := cpu.getProfileFloat("algorithm_complexity", "size_scaling.O(n²).log_factor", 2.0)
			maxFactor := cpu.getProfileFloat("algorithm_complexity", "size_scaling.O(n²).max_factor", 50.0)
			sizeFactor := 1.0 + math.Log10(sizeKB) * logFactor
			complexityFactor *= math.Min(sizeFactor, maxFactor)
		}
	}

	return time.Duration(float64(baseTime) * complexityFactor)
}

// getLanguageMultiplier returns the performance multiplier for a language
func (cpu *CPUEngine) getLanguageMultiplier(language string) float64 {
	if multiplier, ok := cpu.LanguageMultipliers[language]; ok {
		// Validate multiplier is reasonable
		return cpu.validateMultiplier("language", language, multiplier)
	}
	// Log warning for unknown language
	if language != "" && language != "unknown" {
		// Note: Using fmt.Printf instead of log to avoid import issues
		// In production, this should use proper logging
		fmt.Printf("Warning: Unknown language '%s', using baseline performance (1.0)\n", language)
	}
	return 1.0 // Default to Go baseline
}

// getComplexityFactor returns the complexity factor for an algorithm
func (cpu *CPUEngine) getComplexityFactor(complexity string) float64 {
	if factor, ok := cpu.ComplexityFactors[complexity]; ok {
		// Validate factor is reasonable
		return cpu.validateMultiplier("complexity", complexity, factor)
	}
	// Log warning for unknown complexity
	if complexity != "" && complexity != "unknown" {
		fmt.Printf("Warning: Unknown complexity '%s', using O(1) baseline (1.0)\n", complexity)
	}
	return 1.0 // Default to O(1)
}

// validateMultiplier validates that a multiplier value is reasonable and safe
func (cpu *CPUEngine) validateMultiplier(multiplierType, name string, value float64) float64 {
	// Check for invalid values that could cause crashes or unrealistic behavior
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		fmt.Printf("Warning: Invalid %s multiplier for '%s': %.3f, using 1.0\n", multiplierType, name, value)
		return 1.0
	}

	// Check for extreme values that are unrealistic
	if value > 100.0 || value < 0.001 {
		fmt.Printf("Warning: Extreme %s multiplier for '%s': %.3f (outside realistic range 0.001-100.0)\n",
			multiplierType, name, value)
		// Don't change extreme values, just warn - they might be intentional for testing
	}

	return value
}

// calculateCurrentUtilization calculates current CPU utilization based on busy cores
func (cpu *CPUEngine) calculateCurrentUtilization() float64 {
	if cpu.CoreCount == 0 {
		return 0.0
	}

	// Calculate utilization based on cores currently processing operations
	coreUtilization := float64(cpu.BusyCores) / float64(cpu.CoreCount)

	// Ensure utilization is between 0 and 1
	return math.Max(0.0, math.Min(1.0, coreUtilization))
}

// calculateProcessingTimeForOperation calculates processing time for an operation
// FIXED: Now uses the COMPLETE pipeline from ProcessOperation to ensure all CPU modeling effects are applied
func (cpu *CPUEngine) calculateProcessingTimeForOperation(op *Operation) time.Duration {
	// Use the COMPLETE pipeline from ProcessOperation (not simplified version)
	// This ensures cache, boost, thermal, and all other effects are properly applied

	// Calculate base processing time from profile and operation
	baseTime := cpu.calculateBaseProcessingTime(op)

	// Apply language performance multiplier (if enabled)
	languageTime := baseTime
	if cpu.ComplexityInterface.ShouldEnableFeature("language_multipliers") {
		languageTime = cpu.applyLanguageMultiplier(baseTime, op.Language)
	}

	// Apply algorithm complexity factor (if enabled)
	complexityTime := languageTime
	if cpu.ComplexityInterface.ShouldEnableFeature("complexity_factors") {
		complexityTime = cpu.applyComplexityFactor(languageTime, op.Complexity, op.DataSize)
	}

	// Apply SIMD/Vectorization effects (if enabled)
	vectorizedTime := complexityTime
	if cpu.ComplexityInterface.ShouldEnableFeature("simd_vectorization") {
		vectorizedTime = cpu.applyVectorizationEffects(complexityTime, op)
	}

	// Apply cache behavior (basic or advanced based on settings)
	cacheAdjustedTime := vectorizedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("basic_caching") ||
	   cpu.ComplexityInterface.ShouldEnableFeature("advanced_caching") {
		cacheAdjustedTime = cpu.applyCacheBehavior(vectorizedTime, op)
	}

	// Apply advanced prefetching effects (if enabled)
	prefetchAdjustedTime := cacheAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("advanced_prefetching") {
		prefetchAdjustedTime = cpu.applyAdvancedPrefetching(cacheAdjustedTime, op)
	}

	// Apply branch prediction effects (if enabled)
	branchAdjustedTime := prefetchAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("branch_prediction") {
		branchAdjustedTime = cpu.applyBranchPrediction(prefetchAdjustedTime, op)
	}

	// Calculate optimal cores and apply parallel processing (if enabled)
	parallelAdjustedTime := branchAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("parallel_processing") {
		optimalCores := cpu.calculateCoresNeeded(op)
		coresAllocated := int(math.Min(float64(optimalCores), float64(cpu.CoreCount)))
		parallelAdjustedTime = cpu.applyParallelProcessingSpeedup(branchAdjustedTime, op, coresAllocated)
	}

	// Apply boost clock effects (if enabled)
	boostAdjustedTime := parallelAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("boost_clocks") {
		boostAdjustedTime = cpu.applyBoostClockEffects(parallelAdjustedTime)
	}

	// Apply NUMA topology effects (if enabled)
	numaAdjustedTime := boostAdjustedTime
	if cpu.ComplexityInterface.ShouldEnableFeature("numa_topology") {
		numaAdjustedTime = cpu.applyNUMAEffects(boostAdjustedTime, op)
	}

	// Apply thermal effects (thermal throttling)
	thermalAdjustedTime := cpu.applyThermalEffects(numaAdjustedTime)

	// Apply common performance factors (final step)
	utilization := cpu.calculateCurrentUtilization()
	finalTime := cpu.ApplyCommonPerformanceFactors(thermalAdjustedTime, utilization)

	return finalTime
}



// DurationToTicks converts a duration to number of ticks
func (cpu *CPUEngine) DurationToTicks(duration time.Duration) int64 {
	if cpu.TickDuration == 0 {
		return 1 // Fallback to prevent division by zero
	}

	// Round UP to ensure operations take at least 1 tick
	ticks := int64(math.Ceil(float64(duration) / float64(cpu.TickDuration)))

	// Ensure minimum of 1 tick
	if ticks < 1 {
		return 1
	}

	return ticks
}

// applyCacheBehavior applies cache hierarchy (L1 → L2 → L3 → Memory) effects
func (cpu *CPUEngine) applyCacheBehavior(baseTime time.Duration, op *Operation) time.Duration {
	// Update working set size with realistic cache management
	cpu.updateWorkingSetSize(op)

	// Simulate cache hierarchy: L1 → L2 → L3 → Memory
	cacheLevel, multiplier := cpu.simulateCacheHierarchy(op)

	// Apply hardware-dependent cache multiplier (not fixed penalty)
	adjustedTime := time.Duration(float64(baseTime) * multiplier)



	// Update cache statistics
	cpu.updateCacheStatistics(op, cacheLevel)

	return adjustedTime
}

// simulateCacheHierarchy simulates L1 → L2 → L3 → Memory hierarchy with hardware-dependent multipliers
func (cpu *CPUEngine) simulateCacheHierarchy(op *Operation) (int, float64) {
	// Calculate hit ratios for each level based on current state
	l1HitRatio := cpu.calculateLevelHitRatio(op, 1)
	l2HitRatio := cpu.calculateLevelHitRatio(op, 2)
	l3HitRatio := cpu.calculateLevelHitRatio(op, 3)

	// L1 cache check - absolute hit ratio
	if cpu.determineCacheHitForLevel(op, l1HitRatio, 1) {
		return 1, cpu.CacheState.L1HitMultiplier // L1 hit - fastest
	}

	// L1 missed - check L2 cache (conditional hit ratio of L1 misses)
	if cpu.determineCacheHitForLevel(op, l2HitRatio, 2) {
		return 2, cpu.CacheState.L2HitMultiplier // L2 hit - slower than L1
	}

	// L1 and L2 missed - check L3 cache (conditional hit ratio of L2 misses)
	if cpu.determineCacheHitForLevel(op, l3HitRatio, 3) {
		return 3, cpu.CacheState.L3HitMultiplier // L3 hit - slower than L2
	}

	// All caches missed - memory access (slowest)
	return 0, cpu.CacheState.MemoryAccessMultiplier
}

// calculateLevelHitRatio calculates hit ratio for specific cache level
func (cpu *CPUEngine) calculateLevelHitRatio(op *Operation, level int) float64 {
	var targetRatio, coldStartRatio float64

	switch level {
	case 1:
		targetRatio = cpu.getProfileFloat("cache_behavior", "l1_hit_ratio_target", 0.95)
		coldStartRatio = cpu.getProfileFloat("cache_behavior", "cold_start_l1_ratio", 0.3)
	case 2:
		targetRatio = cpu.getProfileFloat("cache_behavior", "l2_hit_ratio_target", 0.85)
		coldStartRatio = cpu.getProfileFloat("cache_behavior", "cold_start_l2_ratio", 0.2)
	case 3:
		targetRatio = cpu.getProfileFloat("cache_behavior", "l3_hit_ratio_target", 0.70)
		coldStartRatio = cpu.getProfileFloat("cache_behavior", "cold_start_l3_ratio", 0.1)
	default:
		return 0.0
	}

	// Get warmup threshold from profile
	warmupThreshold := int64(cpu.getProfileInt("cache_behavior", "warmup_operations", 100))

	// Apply cache warming and working set effects
	if cpu.CacheState.CacheWarming && cpu.CacheState.WarmupOperations < warmupThreshold {
		// Cold cache: start low and warm up to target (profile-driven)
		warmupProgress := float64(cpu.CacheState.WarmupOperations) / float64(warmupThreshold)
		currentRatio := coldStartRatio + warmupProgress*(targetRatio-coldStartRatio)

		// Return calculated ratio without updating state (pure calculation)
		return math.Max(0.1, math.Min(0.98, currentRatio))
	}

	// Converged behavior: adjust based on working set pressure
	workingSetRatio := math.Min(1.0, float64(cpu.CacheState.WorkingSetSize)/float64(cpu.getCacheSizeForLevel(level)))
	workingSetPressure := cpu.getProfileFloat("cache_behavior", "working_set_pressure_factor", 0.3)

	// Higher working set pressure reduces hit ratio
	pressureAdjustment := 1.0 - (workingSetRatio * workingSetPressure)
	adjustedRatio := targetRatio * pressureAdjustment

	// Return calculated ratio without updating state (pure calculation)
	return math.Max(0.1, math.Min(0.98, adjustedRatio))
}

// getCacheSizeForLevel returns cache size in bytes for specific level
func (cpu *CPUEngine) getCacheSizeForLevel(level int) int64 {
	switch level {
	case 1:
		return int64(cpu.CacheL1KB) * 1024
	case 2:
		return int64(cpu.CacheL2KB) * 1024
	case 3:
		return int64(cpu.CacheL3MB) * 1024 * 1024
	default:
		return 0
	}
}

// updateCacheStatistics tracks cache access patterns for convergence
func (cpu *CPUEngine) updateCacheStatistics(op *Operation, hitLevel int) {
	// Increment warmup operations for cache warming
	if cpu.CacheState.CacheWarming {
		cpu.CacheState.WarmupOperations++
		warmupThreshold := int64(cpu.getProfileInt("cache_behavior", "warmup_operations", 100))
		if cpu.CacheState.WarmupOperations >= warmupThreshold {
			cpu.CacheState.CacheWarming = false
		}
	}

	// Update cache hit ratios based on current calculations
	cpu.CacheState.L1HitRatio = cpu.calculateLevelHitRatio(op, 1)
	cpu.CacheState.L2HitRatio = cpu.calculateLevelHitRatio(op, 2)
	cpu.CacheState.L3HitRatio = cpu.calculateLevelHitRatio(op, 3)

	// Track access patterns for working set analysis
	if len(cpu.CacheState.AccessPatternHistory) >= 1000 {
		cpu.CacheState.AccessPatternHistory = cpu.CacheState.AccessPatternHistory[1:]
	}
	cpu.CacheState.AccessPatternHistory = append(cpu.CacheState.AccessPatternHistory, cpu.CurrentTick)
}

// Note: calculateCacheHitRatio() function removed - was conflicting with calculateLevelHitRatio()
// Cache hit ratios are now calculated only by calculateLevelHitRatio() as single source of truth

// applyThermalEffects applies physics-based thermal modeling
func (cpu *CPUEngine) applyThermalEffects(baseTime time.Duration) time.Duration {
	// No thermal throttling if below limit
	if !cpu.ThermalState.ThrottleActive {
		return baseTime
	}

	// Apply Intel-documented thermal throttling
	return time.Duration(float64(baseTime) / cpu.ThermalState.ThrottleFactor)
}

// updateThermalState updates thermal state based on actual CPU work performed
func (cpu *CPUEngine) updateThermalState() {
	// Real-world thermal modeling: Heat generation and cooling based on actual work time
	// Both heat generation and cooling must use the same time scale for realistic physics

	// Get thermal behavior from profile - NO hardcoded defaults
	// If profile doesn't specify, use TDP-based fallback calculations
	var heatGenerationRate, coolingCapacity, coolingEfficiency float64

	if cpu.Profile != nil {
		if specs, ok := cpu.Profile.EngineSpecific["thermal_behavior"]; ok {
			if thermal, ok := specs.(map[string]interface{}); ok {
				if rate, ok := thermal["heat_generation_rate"].(float64); ok {
					heatGenerationRate = rate
				}
				if capacity, ok := thermal["cooling_capacity"].(float64); ok {
					coolingCapacity = capacity
				}
				if efficiency, ok := thermal["cooling_efficiency"].(float64); ok {
					coolingEfficiency = efficiency
				}
			}
		}
	}

	// Fallback: derive from TDP if profile doesn't specify
	if heatGenerationRate == 0 {
		heatGenerationRate = cpu.TDP / 100.0 // TDP spread across 100% load
	}
	if coolingCapacity == 0 {
		coolingCapacity = cpu.TDP * 1.2 // 20% headroom over TDP
	}
	if coolingEfficiency == 0 {
		coolingEfficiency = 0.9 // Conservative 90% if not specified
	}

	// Work-based heat generation: Heat generated from actual operations processed
	// Get the accumulated work since last thermal update
	workBasedHeat := cpu.calculateWorkBasedHeat()

	// Calculate the actual time period for thermal effects
	// Use tick time for baseline idle heat, but work time for cooling
	tickTimeSeconds := float64(cpu.TickDuration) / float64(time.Second)

	// Calculate heat based on actual core utilization
	currentUtilization := float64(cpu.BusyCores) / float64(cpu.CoreCount)
	activeHeatRate := heatGenerationRate * currentUtilization
	idleHeatRate := heatGenerationRate * 0.1 * (1.0 - currentUtilization)

	totalHeatGeneration := (activeHeatRate + idleHeatRate) * tickTimeSeconds

	// Physics-based heat dissipation: Newton's Law of Cooling
	// Cooling must operate for the same time period as heat generation
	tempDifference := math.Max(0, cpu.ThermalState.CurrentTemperatureC - cpu.ThermalState.AmbientTemperatureC)

	// Calculate cooling time based on accumulated work heat
	// If significant work heat was generated, cooling operates for longer period
	coolingTimeSeconds := tickTimeSeconds
	idleHeatThreshold := heatGenerationRate * 0.1 * tickTimeSeconds * 10
	if workBasedHeat > idleHeatThreshold { // If work heat dominates
		// Estimate the time period that generated this work heat
		// This is approximate but more realistic than tick time
		estimatedWorkTime := workBasedHeat / (cpu.TDP * 0.8) // Assume 80% TDP average
		coolingTimeSeconds = math.Max(tickTimeSeconds, estimatedWorkTime)
	}

	// Real cooling physics: Higher temperature difference = more effective cooling
	// Newton's Law: dQ/dt = h × A × (T_hot - T_ambient)
	coolingPower := coolingCapacity * coolingEfficiency * (tempDifference / 63.0) // 63°C = typical design point

	// Convert to Joules: Power × Time (using appropriate time scale)
	heatDissipation := coolingPower * coolingTimeSeconds

	// Net heat accumulation
	netHeat := totalHeatGeneration - heatDissipation
	cpu.ThermalState.HeatAccumulation += netHeat

	// Reset accumulated work heat after processing
	cpu.ThermalState.AccumulatedWorkHeat = 0.0

	// Prevent negative heat accumulation
	cpu.ThermalState.HeatAccumulation = math.Max(0.0, cpu.ThermalState.HeatAccumulation)

	// Temperature calculation (simplified thermal model)
	// Heat capacity: J/°C (thermal mass of CPU + heatsink) - from profile or derived
	thermalCapacity := cpu.getThermalCapacityFromProfile()
	cpu.ThermalState.CurrentTemperatureC = cpu.ThermalState.AmbientTemperatureC +
		(cpu.ThermalState.HeatAccumulation / thermalCapacity) // Convert to temperature rise

	// Check for thermal throttling - use profile values
	throttleThresholdFactor := cpu.getProfileFloat("thermal_behavior", "throttle_threshold_factor", 0.95)
	throttleThreshold := cpu.ThermalLimitC * throttleThresholdFactor
	if cpu.ThermalState.CurrentTemperatureC > throttleThreshold {
		cpu.ThermalState.ThrottleActive = true
		// Throttle factor reduces linearly with excess temperature
		excessHeat := cpu.ThermalState.CurrentTemperatureC - cpu.ThermalLimitC
		maxExcess := cpu.getProfileFloat("thermal_behavior", "max_excess_temp", 15.0)
		maxThrottleReduction := cpu.getProfileFloat("thermal_behavior", "max_throttle_reduction", 0.5)
		throttleReduction := math.Min(excessHeat/maxExcess, maxThrottleReduction)
		cpu.ThermalState.ThrottleFactor = 1.0 - throttleReduction
	} else {
		cpu.ThermalState.ThrottleActive = false
		cpu.ThermalState.ThrottleFactor = 1.0
	}

	cpu.ThermalState.LastThermalUpdate = cpu.CurrentTick
}

// Note: updateCacheConvergence() function removed to eliminate conflicting cache hit ratio updates
// Cache convergence is now handled by calculateLevelHitRatio() as single source of truth

// updateRemainingDynamicState updates remaining dynamic state after processing an operation
func (cpu *CPUEngine) updateRemainingDynamicState(op *Operation, processingTime time.Duration) {
	// Update utilization in health metrics (cores already allocated)
	cpu.Health.Utilization = cpu.calculateCurrentUtilization()

	// Update convergence state
	cpu.ConvergenceState.OperationCount++
	cpu.ConvergenceState.DataProcessed += op.DataSize
}

// calculateCoresNeeded determines how many cores an operation can effectively use
func (cpu *CPUEngine) calculateCoresNeeded(op *Operation) int {
	// If parallel processing is disabled, use only 1 core
	if !cpu.ParallelProcessingState.Enabled {
		return 1
	}

	// Special handling for I/O operations - they are typically single-threaded or limited parallelization
	if op.Type == "io" {
		maxIOCores := cpu.getProfileFloat("core_allocation", "max_io_cores", 2.0)
		return int(math.Min(maxIOCores, float64(cpu.CoreCount)))
	}

	// Special handling for small data - use limited cores for small datasets
	dataSizeKB := float64(op.DataSize) / 1024.0
	tinyDataThreshold := cpu.getProfileFloat("core_allocation", "tiny_data_threshold_kb", 4.0)
	smallDataThreshold := cpu.getProfileFloat("core_allocation", "small_data_threshold_kb", 128.0)

	if dataSizeKB < tinyDataThreshold {
		return 1 // Force single core for tiny data
	} else if dataSizeKB < smallDataThreshold {
		// Small data: limit to 2-8 cores based on size
		if dataSizeKB < 16.0 {
			return 2 // 4KB-16KB: max 2 cores
		} else if dataSizeKB < 32.0 {
			return 4 // 16KB-32KB: max 4 cores
		} else {
			return 8 // 32KB-64KB: max 8 cores
		}
	}

	// Get parallelizability for this operation type
	parallelizability := cpu.getOperationParallelizability(op)
	if parallelizability <= 0.1 {
		return 1 // Not worth parallelizing
	}

	// Use Amdahl's Law to find optimal core count
	// Find the point where adding more cores gives diminishing returns (< 90% efficiency)
	optimalCores := cpu.calculateOptimalCoresAmdahl(parallelizability)

	// Consider data size for scaling (larger datasets can benefit from more cores)
	dataSizeAdjustment := cpu.calculateDataSizeAdjustment(op.DataSize)
	adjustedCores := int(float64(optimalCores) * dataSizeAdjustment)

	// Apply profile-driven limits, but data size adjustment takes precedence for small data
	maxCoresForComplexity := cpu.getMaxCoresForComplexity(op.Complexity)

	var finalCores int
	// If data size adjustment significantly reduces cores (for small data), respect that
	if dataSizeAdjustment <= 0.25 { // Small data factor
		finalCores = adjustedCores
	} else {
		finalCores = int(math.Min(float64(adjustedCores), float64(maxCoresForComplexity)))
	}

	// Ensure we don't exceed available cores and always use at least 1
	finalCores = int(math.Min(float64(finalCores), float64(cpu.CoreCount)))
	return int(math.Max(1, float64(finalCores)))
}

// calculateOptimalCoresAmdahl uses Amdahl's Law to find optimal core count
func (cpu *CPUEngine) calculateOptimalCoresAmdahl(parallelizability float64) int {
	// Find the point where adding more cores gives diminishing returns
	// Look for where speedup per core drops below a threshold

	bestCores := 1
	bestEfficiencyPerCore := 1.0 // 1 core always has 100% efficiency per core

	for cores := 2; cores <= cpu.CoreCount; cores++ {
		// Amdahl's Law: speedup = 1 / ((1-P) + P/N)
		speedup := 1.0 / ((1.0 - parallelizability) + parallelizability/float64(cores))
		efficiencyPerCore := speedup / float64(cores) // How much speedup per core used

		// If efficiency per core drops significantly, stop adding cores
		if efficiencyPerCore < bestEfficiencyPerCore * 0.7 { // 70% of best efficiency per core
			break
		}

		bestCores = cores
		bestEfficiencyPerCore = efficiencyPerCore
	}

	return bestCores
}

// calculateDataSizeAdjustment adjusts core count based on data size
func (cpu *CPUEngine) calculateDataSizeAdjustment(dataSize int64) float64 {
	if dataSize <= 0 {
		return 0.1 // Zero or negative data should use minimal cores
	}

	// Get profile-driven data size scaling factors with more realistic thresholds
	tinyDataThreshold := cpu.getProfileFloat("core_allocation", "tiny_data_threshold_kb", 4.0)    // 4KB
	smallDataThreshold := cpu.getProfileFloat("core_allocation", "small_data_threshold_kb", 64.0)  // 64KB
	largeDataThreshold := cpu.getProfileFloat("core_allocation", "large_data_threshold_kb", 1024.0) // 1MB

	dataSizeKB := float64(dataSize) / 1024.0

	if dataSizeKB < tinyDataThreshold {
		// Tiny data: use single core only (overhead dominates completely)
		return cpu.getProfileFloat("core_allocation", "tiny_data_factor", 0.05) // Even smaller factor
	} else if dataSizeKB < smallDataThreshold {
		// Small data: reduce core count significantly (overhead dominates)
		return cpu.getProfileFloat("core_allocation", "small_data_factor", 0.25)
	} else if dataSizeKB > largeDataThreshold {
		// Large data: can use more cores effectively
		return cpu.getProfileFloat("core_allocation", "large_data_factor", 1.5)
	}

	// Medium data: no adjustment
	return 1.0
}

// getMaxCoresForComplexity returns maximum cores that make sense for a given complexity
func (cpu *CPUEngine) getMaxCoresForComplexity(complexity string) int {
	// Get profile-driven complexity limits
	if cpu.Profile != nil {
		if specs, ok := cpu.Profile.EngineSpecific["core_allocation"]; ok {
			if coreAlloc, ok := specs.(map[string]interface{}); ok {
				if complexityLimits, ok := coreAlloc["max_cores_by_complexity"].(map[string]interface{}); ok {
					if limit, ok := complexityLimits[complexity].(float64); ok {
						return int(limit)
					}
				}
			}
		}
	}

	// Fallback: reasonable defaults based on complexity
	switch complexity {
	case "O(1)":
		return 2 // Simple operations don't benefit from many cores
	case "O(log n)":
		return 4
	case "O(n)":
		return cpu.CoreCount // Linear operations can use all cores
	case "O(n log n)":
		return cpu.CoreCount
	case "O(n²)":
		return cpu.CoreCount // Complex operations benefit from parallelization
	default:
		return cpu.CoreCount / 2 // Conservative default
	}
}

// initializeConvergenceModels initializes statistical convergence models
func (cpu *CPUEngine) initializeConvergenceModels() {
	cpu.ConvergenceState.Models["cache_behavior"] = &StatisticalModel{
		Name:             "cache_behavior",
		ConvergencePoint: 0.88, // Intel Xeon typical cache hit ratio
		BaseVariance:     0.05,
		MinOperations:    10000,
		CurrentValue:     0.5, // Start low
		IsConverged:      false,
	}

	cpu.ConvergenceState.Models["thermal_behavior"] = &StatisticalModel{
		Name:             "thermal_behavior",
		ConvergencePoint: 1.0, // No throttling under normal load
		BaseVariance:     0.02,
		MinOperations:    5000,
		CurrentValue:     1.0,
		IsConverged:      false,
	}

	cpu.ConvergenceState.Models["load_behavior"] = &StatisticalModel{
		Name:             "load_behavior",
		ConvergencePoint: 1.0, // Optimal performance baseline
		BaseVariance:     0.03,
		MinOperations:    1000,
		CurrentValue:     1.0,
		IsConverged:      false,
	}
}

// determineCacheHit determines if an operation results in a cache hit using deterministic logic
func (cpu *CPUEngine) determineCacheHit(op *Operation, hitRatio float64) bool {
	// Use operation characteristics to determine cache behavior deterministically
	// This ensures identical operations always produce identical results

	// Create a deterministic hash based on operation characteristics
	opHash := cpu.hashOperationForCacheDecision(op)

	// Convert hash to a value between 0 and 1
	hashValue := float64(opHash%10000) / 10000.0

	// Cache hit if hash value is less than hit ratio
	return hashValue < hitRatio
}

// determineCacheHitForLevel determines cache hit for specific level with physical constraints
func (cpu *CPUEngine) determineCacheHitForLevel(op *Operation, hitRatio float64, level int) bool {
	// First check: Does the data size fit in this cache level?
	var cacheSizeBytes int64
	switch level {
	case 1:
		cacheSizeBytes = int64(cpu.CacheL1KB) * 1024
	case 2:
		cacheSizeBytes = int64(cpu.CacheL2KB) * 1024
	case 3:
		cacheSizeBytes = int64(cpu.CacheL3MB) * 1024 * 1024
	default:
		return false // Invalid level
	}

	// If operation data size exceeds cache size, it's an automatic miss
	if op.DataSize > cacheSizeBytes {
		return false
	}

	// If data fits, use probabilistic hit ratio for cache behavior
	opHash := cpu.hashOperationForCacheDecision(op)
	levelHash := opHash + uint32(level*12345) // Different seed per level
	hashValue := float64(levelHash%10000) / 10000.0

	return hashValue < hitRatio
}

// hashOperationForCacheDecision creates a deterministic hash for cache decisions
// Uses a combination approach to ensure good distribution across all operation characteristics
func (cpu *CPUEngine) hashOperationForCacheDecision(op *Operation) uint32 {
	// Create separate hashes for each characteristic to ensure equal influence

	// Hash 1: Data size (use multiple bits for better distribution)
	dataHash := uint32(op.DataSize) ^ uint32(op.DataSize>>16) ^ uint32(op.DataSize>>32)
	dataHash = dataHash*2654435761 + 1013904223 // Knuth's multiplicative hash

	// Hash 2: Operation type
	typeHash := uint32(0)
	for i, b := range []byte(op.Type) {
		typeHash = typeHash*31 + uint32(b) + uint32(i)*17
	}

	// Hash 3: Complexity
	complexityHash := uint32(0)
	for i, b := range []byte(op.Complexity) {
		complexityHash = complexityHash*37 + uint32(b) + uint32(i)*19
	}

	// Hash 4: Language
	languageHash := uint32(0)
	for i, b := range []byte(op.Language) {
		languageHash = languageHash*41 + uint32(b) + uint32(i)*23
	}

	// Hash 5: Operation ID (using FNV-1a for good UUID distribution)
	const fnvOffsetBasis uint32 = 2166136261
	const fnvPrime uint32 = 16777619
	idHash := fnvOffsetBasis
	for _, b := range []byte(op.ID) {
		idHash ^= uint32(b)
		idHash *= fnvPrime
	}

	// Combine all hashes with different weights and rotations to avoid correlation
	combined := dataHash
	combined ^= (typeHash << 7) | (typeHash >> 25)     // Rotate left 7
	combined ^= (complexityHash << 13) | (complexityHash >> 19) // Rotate left 13
	combined ^= (languageHash << 19) | (languageHash >> 13)     // Rotate left 19
	combined ^= (idHash << 3) | (idHash >> 29)         // Rotate left 3

	// Final mixing to eliminate any remaining patterns
	combined ^= combined >> 16
	combined *= 0x85ebca6b
	combined ^= combined >> 13
	combined *= 0xc2b2ae35
	combined ^= combined >> 16

	return combined
}

// getCacheEfficiencyFromProfile gets cache efficiency from profile, with fallback
func (cpu *CPUEngine) getCacheEfficiencyFromProfile() float64 {
	// Try to get L3 hit ratio from profile
	if cpu.Profile != nil {
		if specs, ok := cpu.Profile.EngineSpecific["cache_behavior"]; ok {
			if cache, ok := specs.(map[string]interface{}); ok {
				if l3HitRatio, ok := cache["l3_hit_ratio"].(float64); ok {
					return l3HitRatio
				}
			}
		}
	}

	// Fallback: derive from CPU architecture if not specified
	// Higher-end CPUs typically have better cache efficiency
	if cpu.CacheL3MB >= 32 {
		return 0.75 // High-end server CPU
	} else if cpu.CacheL3MB >= 16 {
		return 0.70 // Mid-range CPU
	} else {
		return 0.65 // Entry-level CPU
	}
}

// getThermalCapacityFromProfile gets thermal capacity from profile, with fallback
func (cpu *CPUEngine) getThermalCapacityFromProfile() float64 {
	// Get thermal mass from profile - use real-world values
	if cpu.Profile != nil {
		if specs, ok := cpu.Profile.EngineSpecific["thermal_behavior"]; ok {
			if thermal, ok := specs.(map[string]interface{}); ok {
				if mass, ok := thermal["thermal_mass"].(float64); ok {
					return mass // Use profile value as-is (real-world thermal capacity)
				}
			}
		}
	}

	// Fallback: derive from TDP and CPU size using REALISTIC thermal capacity
	// Real CPU + heatsink thermal capacity (Joules per °C) - realistic values for observable temperature changes
	// These values are calibrated to show realistic temperature rises (10-40°C) under load
	if cpu.TDP >= 200 {
		return 150.0 // High-end server CPU with large heatsink (realistic thermal mass)
	} else if cpu.TDP >= 100 {
		return 100.0 // Mid-range CPU with standard heatsink
	} else {
		return 75.0  // Low-power CPU with small heatsink
	}
}

// calculateWorkBasedHeat calculates heat generated from actual work performed
func (cpu *CPUEngine) calculateWorkBasedHeat() float64 {
	// Return and reset the accumulated work heat
	workHeat := cpu.ThermalState.AccumulatedWorkHeat
	return workHeat
}

// addWorkBasedHeat adds heat based on realistic CPU power consumption and actual processing time
func (cpu *CPUEngine) addWorkBasedHeat(op *Operation, processingTime time.Duration) {
	// Real-world thermal physics: Heat = Power × Time
	// Heat generation is directly proportional to actual simulated processing time

	// Calculate power consumption based on operation complexity and current utilization
	currentUtilization := float64(cpu.ActiveCores) / float64(cpu.CoreCount)

	// Base power consumption (Watts) - realistic CPU power draw
	idlePower := cpu.TDP * 0.1  // 10% TDP when idle
	maxPower := cpu.TDP         // 100% TDP at full load

	// Power scales with utilization and operation complexity
	complexityMultiplier := 1.0
	switch op.Complexity {
	case ComplexityO1:
		complexityMultiplier = 0.3 // Light computational load
	case ComplexityOLogN:
		complexityMultiplier = 0.5 // Medium computational load
	case ComplexityON:
		complexityMultiplier = 0.8 // Heavy computational load
	case ComplexityON2:
		complexityMultiplier = 1.0 // Maximum computational load
	default:
		complexityMultiplier = 0.3
	}

	// Calculate actual power consumption for this operation
	operationPower := idlePower + (maxPower - idlePower) * currentUtilization * complexityMultiplier

	// Convert processing time to seconds for heat calculation
	processingTimeSeconds := float64(processingTime) / float64(time.Second)

	// Real physics: Heat = Power × Time (Joules = Watts × Seconds)
	operationHeat := operationPower * processingTimeSeconds

	// No artificial scaling - heat is directly proportional to actual simulated time
	// If operation takes 10ms, generate 10ms worth of heat
	// If operation takes 1s, generate 1s worth of heat

	// Accumulate heat for thermal update
	cpu.ThermalState.AccumulatedWorkHeat += operationHeat
}

// Reset resets the CPU engine to initial state
func (cpu *CPUEngine) Reset() {
	// Reset common engine state first
	cpu.CommonEngine.Reset()

	// Reset CPU-specific state
	cpu.ActiveCores = 0

	// Reset thermal state to initial values
	ambientTemp := 22.0 // Default
	if cpu.Profile != nil {
		if specs, ok := cpu.Profile.EngineSpecific["thermal_behavior"]; ok {
			if thermal, ok := specs.(map[string]interface{}); ok {
				if temp, ok := thermal["ambient_temp"].(float64); ok {
					ambientTemp = temp
				}
			}
		}
	}

	cpu.ThermalState.CurrentTemperatureC = ambientTemp
	cpu.ThermalState.HeatAccumulation = 0.0
	cpu.ThermalState.ThrottleActive = false
	cpu.ThermalState.ThrottleFactor = 1.0
	cpu.ThermalState.CoolingCapacity = 0.0
	cpu.ThermalState.AmbientTemperatureC = ambientTemp
	cpu.ThermalState.LastThermalUpdate = 0
	cpu.ThermalState.AccumulatedWorkHeat = 0.0

	// Reset cache state to cold start (using profile values)
	cpu.initializeCacheState() // This will load from profile
	cpu.CacheState.WorkingSetSize = 0
	cpu.CacheState.CacheWarming = true
	cpu.CacheState.WarmupOperations = 0
	cpu.CacheState.AccessPatternHistory = make([]int64, 0, 1000)

	// Reset convergence state
	cpu.ConvergenceState.OperationCount = 0
	cpu.ConvergenceState.DataProcessed = 0
	cpu.ConvergenceState.Models = make(map[string]*StatisticalModel)

	// Reset boost clock state
	cpu.initializeBoostClockState()

	// Reset NUMA state
	cpu.initializeNUMAState()

	// Reset hyperthreading state
	cpu.initializeHyperthreadingState()

	// Reset cache hierarchy
	cpu.initializeCacheHierarchy()

	// Reset branch prediction state
	cpu.BranchPredictionState.TotalBranches = 0
	cpu.BranchPredictionState.TotalMispredictions = 0
}

// initializeBoostClockState initializes boost clock behavior from profile
func (cpu *CPUEngine) initializeBoostClockState() {
	// Set defaults
	cpu.BoostState.CurrentClockGHz = cpu.BaseClockGHz
	cpu.BoostState.BoostActive = false
	cpu.BoostState.BoostStartTick = 0
	cpu.BoostState.SingleCoreBoostGHz = cpu.BaseClockGHz
	cpu.BoostState.AllCoreBoostGHz = cpu.BaseClockGHz
	cpu.BoostState.BoostDurationTicks = 10000 // Default 10 seconds at 1ms ticks
	cpu.BoostState.ThermalDependent = true

	// Load from profile if available
	if cpu.Profile != nil {
		if specs, ok := cpu.Profile.EngineSpecific["boost_behavior"]; ok {
			if boost, ok := specs.(map[string]interface{}); ok {
				if singleBoost, ok := boost["single_core_boost"].(float64); ok {
					cpu.BoostState.SingleCoreBoostGHz = singleBoost
				}
				if allBoost, ok := boost["all_core_boost"].(float64); ok {
					cpu.BoostState.AllCoreBoostGHz = allBoost
				}
				if duration, ok := boost["boost_duration"].(float64); ok {
					cpu.BoostState.BoostDurationTicks = int64(duration * 1000) // Convert seconds to ticks
				}
				if thermal, ok := boost["thermal_dependent"].(bool); ok {
					cpu.BoostState.ThermalDependent = thermal
				}
			}
		}
	}

	// Fallback: derive from base clock if not specified
	if cpu.BoostState.SingleCoreBoostGHz == cpu.BaseClockGHz {
		cpu.BoostState.SingleCoreBoostGHz = cpu.BaseClockGHz * 1.3 // 30% boost
	}
	if cpu.BoostState.AllCoreBoostGHz == cpu.BaseClockGHz {
		cpu.BoostState.AllCoreBoostGHz = cpu.BaseClockGHz * 1.1 // 10% boost
	}
}

// initializeNUMAState initializes NUMA topology from profile
func (cpu *CPUEngine) initializeNUMAState() {
	// Set defaults
	cpu.NUMAState.NumaNodes = 1
	cpu.NUMAState.CrossSocketPenalty = 1.0 // No penalty for single socket
	cpu.NUMAState.LocalMemoryRatio = 1.0   // All memory is local
	cpu.NUMAState.MemoryBandwidthMBs = 100000 // 100 GB/s default

	// Load from profile if available
	if cpu.Profile != nil {
		if specs, ok := cpu.Profile.EngineSpecific["numa_behavior"]; ok {
			if numa, ok := specs.(map[string]interface{}); ok {
				if nodes, ok := numa["numa_nodes"].(float64); ok {
					cpu.NUMAState.NumaNodes = int(nodes)
				}
				if penalty, ok := numa["cross_socket_penalty"].(float64); ok {
					cpu.NUMAState.CrossSocketPenalty = penalty
				}
				if ratio, ok := numa["local_memory_ratio"].(float64); ok {
					cpu.NUMAState.LocalMemoryRatio = ratio
				}
				if bandwidth, ok := numa["memory_bandwidth"].(float64); ok {
					cpu.NUMAState.MemoryBandwidthMBs = int64(bandwidth)
				}
			}
		}
	}

	// Fallback: derive from core count if not specified
	if cpu.NUMAState.NumaNodes == 1 && cpu.CoreCount >= 16 {
		cpu.NUMAState.NumaNodes = 2 // Dual socket for high core count
		cpu.NUMAState.CrossSocketPenalty = 1.8 // 80% penalty
		cpu.NUMAState.LocalMemoryRatio = 0.8   // 80% local access
	}
}

// initializeHyperthreadingState initializes hyperthreading from profile
func (cpu *CPUEngine) initializeHyperthreadingState() {
	// Set defaults
	cpu.HyperthreadingState.Enabled = false
	cpu.HyperthreadingState.ThreadsPerCore = 1
	cpu.HyperthreadingState.EfficiencyFactor = 1.0
	cpu.HyperthreadingState.EffectiveCores = cpu.CoreCount

	// Check if hyperthreading is specified in profile
	if cpu.Profile != nil {
		// Check baseline performance for thread count
		if baseline, ok := cpu.Profile.EngineSpecific["baseline_performance"]; ok {
			if perf, ok := baseline.(map[string]interface{}); ok {
				if threads, ok := perf["threads"].(float64); ok {
					threadsInt := int(threads)
					if threadsInt > cpu.CoreCount {
						cpu.HyperthreadingState.Enabled = true
						cpu.HyperthreadingState.ThreadsPerCore = threadsInt / cpu.CoreCount
						cpu.HyperthreadingState.EfficiencyFactor = 0.65 // 65% efficiency per thread
						cpu.HyperthreadingState.EffectiveCores = int(float64(threadsInt) * cpu.HyperthreadingState.EfficiencyFactor)
					}
				}
			}
		}
	}
}

// initializeCacheHierarchy initializes cache hierarchy from profile
func (cpu *CPUEngine) initializeCacheHierarchy() {
	// Load cache hierarchy from profile
	cpu.CacheState.L1HitRatioTarget = cpu.getProfileFloat("cache_behavior", "l1_hit_ratio_target", 0.95)
	cpu.CacheState.L2HitRatioTarget = cpu.getProfileFloat("cache_behavior", "l2_hit_ratio_target", 0.85)
	cpu.CacheState.L3HitRatioTarget = cpu.getProfileFloat("cache_behavior", "l3_hit_ratio_target", 0.70)
	cpu.CacheState.CacheLineSize = cpu.getProfileInt("cache_behavior", "cache_line_size", 64)
	cpu.CacheState.PrefetchEfficiency = cpu.getProfileFloat("cache_behavior", "prefetch_efficiency", 0.85)

	// Cache penalties are now handled by multipliers, not cycle counts

	// Load hardware-dependent multipliers from profile
	cpu.CacheState.L1HitMultiplier = cpu.getProfileFloat("cache_behavior", "l1_hit_multiplier", 1.0)
	cpu.CacheState.L2HitMultiplier = cpu.getProfileFloat("cache_behavior", "l2_hit_multiplier", 1.2)
	cpu.CacheState.L3HitMultiplier = cpu.getProfileFloat("cache_behavior", "l3_hit_multiplier", 2.0)
	cpu.CacheState.MemoryAccessMultiplier = cpu.getProfileFloat("cache_behavior", "memory_access_multiplier", 8.0)

	// Load from profile if available
	if cpu.Profile != nil {
		if specs, ok := cpu.Profile.EngineSpecific["cache_behavior"]; ok {
			if cache, ok := specs.(map[string]interface{}); ok {
				if l1Hit, ok := cache["l1_hit_ratio"].(float64); ok {
					cpu.CacheState.L1HitRatioTarget = l1Hit
				}
				if l2Hit, ok := cache["l2_hit_ratio"].(float64); ok {
					cpu.CacheState.L2HitRatioTarget = l2Hit
				}
				if l3Hit, ok := cache["l3_hit_ratio"].(float64); ok {
					cpu.CacheState.L3HitRatioTarget = l3Hit
				}
				if lineSize, ok := cache["cache_line_size"].(float64); ok {
					cpu.CacheState.CacheLineSize = int(lineSize)
				}
				if prefetch, ok := cache["prefetch_efficiency"].(float64); ok {
					cpu.CacheState.PrefetchEfficiency = prefetch
				}
				// miss_penalty is deprecated, use memory_access_multiplier instead

				// Load hardware-dependent cache multipliers from profile
				if l1Mult, ok := cache["l1_hit_multiplier"].(float64); ok {
					cpu.CacheState.L1HitMultiplier = l1Mult
				}
				if l2Mult, ok := cache["l2_hit_multiplier"].(float64); ok {
					cpu.CacheState.L2HitMultiplier = l2Mult
				}
				if l3Mult, ok := cache["l3_hit_multiplier"].(float64); ok {
					cpu.CacheState.L3HitMultiplier = l3Mult
				}
				if memMult, ok := cache["memory_access_multiplier"].(float64); ok {
					cpu.CacheState.MemoryAccessMultiplier = memMult
				}
			}
		}
	}

	// Set converged hit ratio to L3 target
	cpu.CacheState.ConvergedHitRatio = cpu.CacheState.L3HitRatioTarget
}

// applyBoostClockEffects applies boost clock frequency scaling to processing time
func (cpu *CPUEngine) applyBoostClockEffects(baseTime time.Duration) time.Duration {
	// Calculate target clock speed based on current load
	targetClock := cpu.calculateTargetClockSpeed()

	// Update boost state
	cpu.updateBoostState(targetClock)

	// Apply frequency scaling: higher frequency = faster processing
	clockMultiplier := cpu.BoostState.CurrentClockGHz / cpu.BaseClockGHz
	adjustedTime := time.Duration(float64(baseTime) / clockMultiplier)

	return adjustedTime
}

// calculateTargetClockSpeed determines target clock speed based on load and thermal state
func (cpu *CPUEngine) calculateTargetClockSpeed() float64 {
	// FIXED: Use BusyCores instead of ActiveCores for sustained load detection
	// BusyCores reflects actual sustained load, unlike ActiveCores which resets after operations
	var targetClock float64

	if cpu.BusyCores == 1 {
		// Single core active: maximum boost (real-world Intel/AMD behavior)
		targetClock = cpu.BoostState.SingleCoreBoostGHz
	} else if cpu.BusyCores <= cpu.CoreCount/4 {
		// Light load (≤25% cores): high boost (realistic boost behavior)
		targetClock = cpu.BoostState.SingleCoreBoostGHz * 0.95 // 95% of single core boost
	} else if cpu.BusyCores <= cpu.CoreCount/2 {
		// Medium load (≤50% cores): moderate boost (thermal/power constraints)
		targetClock = cpu.BoostState.AllCoreBoostGHz * 1.1 // 110% of all core boost
	} else {
		// Heavy load (>50% cores): all-core boost (sustained multi-core workload)
		targetClock = cpu.BoostState.AllCoreBoostGHz
	}

	// Apply thermal throttling if enabled
	if cpu.BoostState.ThermalDependent {
		thermalLimit := cpu.ThermalLimitC * 0.9 // Start boost reduction at 90% of limit (76.5°C - realistic)
		if cpu.ThermalState.CurrentTemperatureC > thermalLimit {
			// Linear throttling based on temperature
			throttleFactor := 1.0 - (cpu.ThermalState.CurrentTemperatureC-thermalLimit)/(cpu.ThermalLimitC-thermalLimit)
			throttleFactor = math.Max(0.5, throttleFactor) // Minimum 50% clock speed
			targetClock *= throttleFactor
		}
	}

	return targetClock
}

// updateBoostState updates the boost clock state based on target frequency
func (cpu *CPUEngine) updateBoostState(targetClock float64) {
	// Check if we should start boost
	if targetClock > cpu.BaseClockGHz && !cpu.BoostState.BoostActive {
		cpu.BoostState.BoostActive = true
		cpu.BoostState.BoostStartTick = cpu.CurrentTick
	}

	// Check if boost duration expired
	if cpu.BoostState.BoostActive {
		boostDuration := cpu.CurrentTick - cpu.BoostState.BoostStartTick
		if boostDuration > cpu.BoostState.BoostDurationTicks {
			// Boost expired, return to base clock
			targetClock = cpu.BaseClockGHz
			cpu.BoostState.BoostActive = false
		}
	}

	// Update current clock (smooth transition)
	if targetClock > cpu.BoostState.CurrentClockGHz {
		// Boost up quickly (1 tick)
		cpu.BoostState.CurrentClockGHz = targetClock
	} else if targetClock < cpu.BoostState.CurrentClockGHz {
		// Boost down gradually (realistic behavior)
		step := (cpu.BoostState.CurrentClockGHz - targetClock) * 0.1
		cpu.BoostState.CurrentClockGHz = math.Max(targetClock, cpu.BoostState.CurrentClockGHz-step)
	}
}



// updateWorkingSetSize updates working set size with realistic cache management
func (cpu *CPUEngine) updateWorkingSetSize(op *Operation) {
	// Calculate total cache size (L1 + L2 + L3)
	totalCacheSize := int64(cpu.CacheL1KB*1024 + cpu.CacheL2KB*1024 + cpu.CacheL3MB*1024*1024)

	// Add new data to working set
	cpu.CacheState.WorkingSetSize += op.DataSize

	// Implement cache replacement policy (LRU-like behavior)
	if cpu.CacheState.WorkingSetSize > totalCacheSize {
		// Cache overflow - implement realistic replacement
		// Keep 80% of cache, evict 20% (simulates LRU replacement)
		cpu.CacheState.WorkingSetSize = int64(float64(totalCacheSize) * 0.8)

		// Add the new operation data
		cpu.CacheState.WorkingSetSize += op.DataSize

		// Ensure we don't exceed cache size again
		if cpu.CacheState.WorkingSetSize > totalCacheSize {
			cpu.CacheState.WorkingSetSize = totalCacheSize
		}
	}

	// Track access patterns for temporal locality
	cpu.CacheState.AccessPatternHistory = append(cpu.CacheState.AccessPatternHistory, op.DataSize)

	// Keep only recent access patterns (sliding window)
	maxHistorySize := 100
	if len(cpu.CacheState.AccessPatternHistory) > maxHistorySize {
		cpu.CacheState.AccessPatternHistory = cpu.CacheState.AccessPatternHistory[len(cpu.CacheState.AccessPatternHistory)-maxHistorySize:]
	}
}

// applyNUMAEffects applies NUMA topology penalties for cross-socket memory access
func (cpu *CPUEngine) applyNUMAEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Skip NUMA effects for single-socket systems
	if cpu.NUMAState.NumaNodes <= 1 {
		return baseTime
	}

	// Calculate NUMA penalty based on memory access pattern
	numaPenalty := cpu.calculateNUMAPenalty(op)

	return time.Duration(float64(baseTime) * numaPenalty)
}

// calculateNUMAPenalty determines NUMA penalty for memory access
func (cpu *CPUEngine) calculateNUMAPenalty(op *Operation) float64 {
	// Simulate memory access locality
	// Use operation hash to determine if access is local or remote
	opHash := cpu.hashOperationForCacheDecision(op)

	// Determine if this is a local or cross-socket access
	localAccessThreshold := uint32(cpu.NUMAState.LocalMemoryRatio * 100)
	isLocalAccess := (opHash % 100) < localAccessThreshold

	if isLocalAccess {
		return 1.0 // No penalty for local access
	} else {
		return cpu.NUMAState.CrossSocketPenalty // Cross-socket penalty
	}
}

// applyHyperthreadingEffects calculates effective cores considering hyperthreading
func (cpu *CPUEngine) applyHyperthreadingEffects() {
	if !cpu.HyperthreadingState.Enabled {
		cpu.HyperthreadingState.EffectiveCores = cpu.ActiveCores
		return
	}

	// Hyperthreading provides diminishing returns
	// 2 threads per core, but only ~65% efficiency per thread
	maxThreads := cpu.CoreCount * cpu.HyperthreadingState.ThreadsPerCore
	currentThreads := cpu.ActiveCores * cpu.HyperthreadingState.ThreadsPerCore

	// Calculate effective performance
	effectivePerformance := float64(currentThreads) * cpu.HyperthreadingState.EfficiencyFactor
	cpu.HyperthreadingState.EffectiveCores = int(math.Min(effectivePerformance, float64(maxThreads)))
}

// applyVectorizationEffects applies SIMD/vectorization speedup using statistical modeling
func (cpu *CPUEngine) applyVectorizationEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Get vectorization probability for this operation type
	vectorizationRatio := cpu.getOperationVectorizationRatio(op)

	// If operation is not vectorizable, return original time
	if vectorizationRatio <= 0.1 {
		cpu.VectorizationState.ScalarOperationsCount++
		return baseTime
	}

	// Calculate theoretical SIMD speedup based on vector width
	scalarWidth := 64 // 64-bit scalar operations
	vectorWidth := float64(cpu.VectorizationState.VectorWidth)
	theoreticalSpeedup := vectorWidth / float64(scalarWidth)

	// Apply SIMD efficiency (real hardware doesn't achieve theoretical speedup)
	efficiency := cpu.VectorizationState.SIMDEfficiency

	// Apply vectorization ratio (not all parts of operation can be vectorized)
	actualSpeedup := theoreticalSpeedup * efficiency * vectorizationRatio

	// Clamp speedup to realistic bounds (max 16x for AVX-512)
	actualSpeedup = math.Min(actualSpeedup, 16.0)
	actualSpeedup = math.Max(actualSpeedup, 1.0)

	// Calculate vectorized time
	vectorizedTime := time.Duration(float64(baseTime) / actualSpeedup)

	// Update statistics
	cpu.VectorizationState.VectorOperationsCount++
	cpu.updateVectorizationStatistics(actualSpeedup)

	return vectorizedTime
}

// GetOperationVectorizationRatio returns the vectorization ratio for an operation type (public for testing)
func (cpu *CPUEngine) GetOperationVectorizationRatio(op *Operation) float64 {
	return cpu.getOperationVectorizationRatio(op)
}

// getOperationVectorizationRatio returns the vectorization ratio for an operation type
func (cpu *CPUEngine) getOperationVectorizationRatio(op *Operation) float64 {
	// Check profile-based vectorizability first
	if ratio, exists := cpu.VectorizationState.OperationVectorizability[op.Type]; exists {
		return ratio
	}

	// Fallback to operation type-based detection
	switch op.Type {
	case "matrix_multiply", "matrix_operation":
		return 0.90 // 90% vectorizable
	case "image_process", "image_filter":
		return 0.85 // 85% vectorizable
	case "ml_inference", "neural_network":
		return 0.80 // 80% vectorizable
	case "array_sum", "vector_add", "vector_multiply":
		return 0.95 // 95% vectorizable
	case "fft", "signal_process":
		return 0.85 // 85% vectorizable
	case "crypto_hash", "encryption":
		return 0.70 // 70% vectorizable (some crypto operations)
	case "string_process", "text_parse":
		return 0.20 // 20% vectorizable
	case "database_query", "sql_parse":
		return 0.15 // 15% vectorizable
	case "network_process", "protocol_parse":
		return 0.10 // 10% vectorizable
	default:
		// Default based on complexity for unknown operations
		switch op.Complexity {
		case "O(n²)", "O(n³)":
			return 0.60 // Matrix-like operations often vectorizable
		case "O(n log n)":
			return 0.40 // Some sorting/search algorithms
		case "O(n)":
			return 0.30 // Linear operations sometimes vectorizable
		default:
			return 0.20 // Conservative default
		}
	}
}

// updateVectorizationStatistics updates running statistics for vectorization
func (cpu *CPUEngine) updateVectorizationStatistics(speedup float64) {
	totalOps := cpu.VectorizationState.VectorOperationsCount + cpu.VectorizationState.ScalarOperationsCount
	if totalOps == 0 {
		return
	}

	// Update running average speedup
	alpha := 0.1 // Smoothing factor
	cpu.VectorizationState.AverageSpeedup = (1-alpha)*cpu.VectorizationState.AverageSpeedup + alpha*speedup

	// Update current vectorization ratio for this operation
	vectorRatio := float64(cpu.VectorizationState.VectorOperationsCount) / float64(totalOps)
	cpu.VectorizationState.VectorizationRatio = vectorRatio
}

// CPU-specific BaseEngine interface overrides

// GetUtilization returns the current CPU utilization
func (cpu *CPUEngine) GetUtilization() float64 {
	return cpu.calculateCurrentUtilization()
}

// GetDynamicState returns the current dynamic state with CPU-specific data
func (cpu *CPUEngine) GetDynamicState() *DynamicState {
	return &DynamicState{
		CurrentUtilization:  cpu.calculateCurrentUtilization(),
		PerformanceFactor:   cpu.ThermalState.ThrottleFactor,
		ConvergenceProgress: cpu.CommonEngine.calculateConvergenceProgress(),
		HardwareSpecific: map[string]interface{}{
			"temperature_c":        cpu.ThermalState.CurrentTemperatureC,
			"active_cores":         cpu.ActiveCores,
			"boost_active":         cpu.BoostState.BoostActive,
			"current_clock_ghz":    cpu.BoostState.CurrentClockGHz,
			"cache_hit_ratio":      cpu.CacheState.L1HitRatio,
			"thermal_throttle":     cpu.ThermalState.ThrottleActive,
			"vector_operations":    cpu.VectorizationState.VectorOperationsCount,
			"scalar_operations":    cpu.VectorizationState.ScalarOperationsCount,
			"average_vector_speedup": cpu.VectorizationState.AverageSpeedup,
			"vector_width":         cpu.VectorizationState.VectorWidth,
		},
		LastUpdated: cpu.CurrentTick,
	}
}

// GetConvergenceMetrics returns CPU-specific convergence metrics
func (cpu *CPUEngine) GetConvergenceMetrics() *ConvergenceMetrics {
	return &ConvergenceMetrics{
		OperationCount:      cpu.ConvergenceState.OperationCount,
		ConvergencePoint:    cpu.CacheState.ConvergedHitRatio,
		CurrentVariance:     cpu.CommonEngine.calculateCurrentVariance(),
		IsConverged:         cpu.ConvergenceState.OperationCount > 10000,
		TimeToConvergence:   cpu.ConvergenceState.ConvergedTick - cpu.ConvergenceState.StartTick,
		ConvergenceFactors: map[string]float64{
			"cache_convergence":   cpu.calculateCacheConvergence(),
			"thermal_convergence": cpu.calculateThermalConvergence(),
			"load_convergence":    cpu.calculateLoadConvergence(),
		},
	}
}

// LoadProfile loads a CPU engine profile
func (cpu *CPUEngine) LoadProfile(profile *EngineProfile) error {
	if profile.Type != CPUEngineType {
		return fmt.Errorf("profile type mismatch: expected CPU, got %v", profile.Type)
	}

	// Set the profile
	cpu.Profile = profile

	// Load CPU-specific profile data
	if err := cpu.loadCPUSpecificProfile(); err != nil {
		return fmt.Errorf("failed to load CPU-specific profile: %w", err)
	}

	// Reinitialize with profile data
	cpu.initializeFromProfile()

	return nil
}

// GetCurrentState returns the complete current state with CPU-specific data
func (cpu *CPUEngine) GetCurrentState() map[string]interface{} {
	baseState := cpu.CommonEngine.GetCurrentState()

	// Add CPU-specific state
	baseState["active_cores"] = cpu.ActiveCores
	baseState["core_count"] = cpu.CoreCount
	baseState["thermal_state"] = cpu.ThermalState
	baseState["cache_state"] = cpu.CacheState
	baseState["boost_state"] = cpu.BoostState
	baseState["numa_state"] = cpu.NUMAState
	baseState["hyperthreading"] = cpu.HyperthreadingState
	baseState["vectorization_state"] = cpu.VectorizationState
	baseState["branch_prediction_state"] = cpu.BranchPredictionState
	baseState["memory_bandwidth_state"] = cpu.MemoryBandwidthState
	baseState["advanced_prefetch_state"] = cpu.AdvancedPrefetchState

	return baseState
}

// COMPLEXITY CONTROL METHODS

// SetComplexityLevel sets the CPU simulation complexity level (legacy method)
func (cpu *CPUEngine) SetComplexityLevelDirect(level CPUComplexityLevel) {
	cpu.ComplexityInterface.SetComplexityLevel(level)
}

// GetComplexityLevelDirect returns the current CPU simulation complexity level (legacy method)
func (cpu *CPUEngine) GetComplexityLevelDirect() CPUComplexityLevel {
	return cpu.ComplexityInterface.ComplexityLevel
}

// GetComplexityDescription returns a description of the current complexity level
func (cpu *CPUEngine) GetComplexityDescription() string {
	return cpu.ComplexityInterface.GetDescription()
}

// GetComplexityPerformanceImpact returns the performance impact of current complexity
func (cpu *CPUEngine) GetComplexityPerformanceImpact() string {
	return cpu.ComplexityInterface.GetPerformanceImpact()
}

// GetEnabledFeatures returns a list of currently enabled CPU features
func (cpu *CPUEngine) GetEnabledFeatures() []string {
	return cpu.ComplexityInterface.GetEnabledFeatures()
}

// EnableFeature manually enables a specific CPU feature
func (cpu *CPUEngine) EnableFeature(feature string) error {
	return cpu.ComplexityInterface.EnableFeature(feature)
}

// DisableFeature manually disables a specific CPU feature
func (cpu *CPUEngine) DisableFeature(feature string) error {
	return cpu.ComplexityInterface.DisableFeature(feature)
}

// SetComplexityLevel sets the complexity level for the CPU engine (BaseEngine interface)
func (cpu *CPUEngine) SetComplexityLevel(level int) error {
	if err := ValidateComplexityLevel(ComplexityLevel(level)); err != nil {
		return err
	}
	cpu.ComplexityInterface.SetComplexityLevel(ComplexityLevel(level))
	return nil
}

// GetComplexityLevel returns the current complexity level (BaseEngine interface)
func (cpu *CPUEngine) GetComplexityLevel() int {
	return int(cpu.ComplexityInterface.ComplexityLevel)
}

// getMaxInternalQueueSize returns the maximum size for the internal processing heap
func (cpu *CPUEngine) getMaxInternalQueueSize() int {
	// INTRA-ENGINE FLOW: Limit heap size based on realistic CPU constraints
	// Heap should hold operations for: cores × average_operation_duration_ticks

	// Get average operation duration from profile
	avgOpDurationMs := cpu.getProfileFloat("queue_processing", "avg_operation_duration_ms", 2.0)
	tickDurationMs := float64(cpu.TickDuration) / float64(time.Millisecond)
	avgOpDurationTicks := int(avgOpDurationMs / tickDurationMs)

	// Calculate max heap size: cores × operation_duration × safety_factor
	safetyFactor := 2.0 // Allow 2x buffer for operation variance
	maxHeapSize := int(float64(cpu.CoreCount * avgOpDurationTicks) * safetyFactor)

	// Ensure reasonable bounds
	if maxHeapSize < 100 {
		maxHeapSize = 100 // Minimum heap size
	}
	if maxHeapSize > 10000 {
		maxHeapSize = 10000 // Maximum heap size to prevent memory issues
	}

	return maxHeapSize
}

// IsFeatureEnabled checks if a specific CPU feature is enabled
func (cpu *CPUEngine) IsFeatureEnabled(feature string) bool {
	return cpu.ComplexityInterface.ShouldEnableFeature(feature)
}

// CPU-specific helper methods

// calculateCacheConvergence calculates cache behavior convergence
func (cpu *CPUEngine) calculateCacheConvergence() float64 {
	if cpu.CacheState.WarmupOperations < 50 {
		return float64(cpu.CacheState.WarmupOperations) / 50.0
	}
	return 1.0
}

// calculateThermalConvergence calculates thermal behavior convergence
func (cpu *CPUEngine) calculateThermalConvergence() float64 {
	// Thermal behavior converges based on sustained load history
	if len(cpu.LoadHistory) < 100 {
		return float64(len(cpu.LoadHistory)) / 100.0
	}
	return 1.0
}

// calculateLoadConvergence calculates load behavior convergence
func (cpu *CPUEngine) calculateLoadConvergence() float64 {
	if cpu.ConvergenceState.OperationCount < 1000 {
		return float64(cpu.ConvergenceState.OperationCount) / 1000.0
	}
	return 1.0
}

// loadCPUSpecificProfile loads CPU-specific profile data
func (cpu *CPUEngine) loadCPUSpecificProfile() error {
	if cpu.Profile == nil {
		return fmt.Errorf("profile is nil")
	}

	// Load baseline performance
	if cpu.Profile.BaselinePerformance != nil {
		if cores, ok := cpu.Profile.BaselinePerformance["cores"]; ok {
			cpu.CoreCount = int(cores)
			cpu.CoreUtilization = make([]float64, cpu.CoreCount)
		}
		if baseClock, ok := cpu.Profile.BaselinePerformance["base_clock"]; ok {
			cpu.BaseClockGHz = baseClock
		}
		if boostClock, ok := cpu.Profile.BaselinePerformance["boost_clock"]; ok {
			cpu.BoostClockGHz = boostClock
		}
	}

	// Load technology specs
	if cpu.Profile.TechnologySpecs != nil {
		if l1Cache, ok := cpu.Profile.TechnologySpecs["cache_l1_kb"].(float64); ok {
			cpu.CacheL1KB = int(l1Cache)
		}
		if l2Cache, ok := cpu.Profile.TechnologySpecs["cache_l2_kb"].(float64); ok {
			cpu.CacheL2KB = int(l2Cache)
		}
		if l3Cache, ok := cpu.Profile.TechnologySpecs["cache_l3_mb"].(float64); ok {
			cpu.CacheL3MB = int(l3Cache)
		}
		if tdp, ok := cpu.Profile.TechnologySpecs["tdp"].(float64); ok {
			cpu.TDP = tdp
		}
		if thermalLimit, ok := cpu.Profile.TechnologySpecs["thermal_limit"].(float64); ok {
			cpu.ThermalLimitC = thermalLimit
		}
	}

	// Load engine-specific configurations
	if cpu.Profile.EngineSpecific != nil {
		cpu.loadEngineSpecificConfigs()
	}

	return nil
}

// loadEngineSpecificConfigs loads all engine-specific configurations
func (cpu *CPUEngine) loadEngineSpecificConfigs() {
	// Load cache behavior
	if cacheConfig, ok := cpu.Profile.EngineSpecific["cache_behavior"].(map[string]interface{}); ok {
		cpu.loadCacheConfig(cacheConfig)
	}

	// Load thermal behavior
	if thermalConfig, ok := cpu.Profile.EngineSpecific["thermal_behavior"].(map[string]interface{}); ok {
		cpu.loadThermalConfig(thermalConfig)
	}

	// Load NUMA behavior
	if numaConfig, ok := cpu.Profile.EngineSpecific["numa_behavior"].(map[string]interface{}); ok {
		cpu.loadNUMAConfig(numaConfig)
	}

	// Load boost behavior
	if boostConfig, ok := cpu.Profile.EngineSpecific["boost_behavior"].(map[string]interface{}); ok {
		cpu.loadBoostConfig(boostConfig)
	}

	// Load hyperthreading configuration
	if htConfig, ok := cpu.Profile.EngineSpecific["hyperthreading"].(map[string]interface{}); ok {
		cpu.loadHyperthreadingConfig(htConfig)
	}

	// Load parallel processing configuration
	if parallelConfig, ok := cpu.Profile.EngineSpecific["parallel_processing"].(map[string]interface{}); ok {
		cpu.loadParallelProcessingConfig(parallelConfig)
	}

	// Load language multipliers
	if langConfig, ok := cpu.Profile.EngineSpecific["language_multipliers"].(map[string]interface{}); ok {
		cpu.loadLanguageMultipliers(langConfig)
	}

	// Load complexity factors
	if complexityConfig, ok := cpu.Profile.EngineSpecific["complexity_factors"].(map[string]interface{}); ok {
		cpu.loadComplexityFactors(complexityConfig)
	}

	// Load memory bandwidth configuration
	if memBandwidthConfig, ok := cpu.Profile.EngineSpecific["memory_bandwidth"].(map[string]interface{}); ok {
		cpu.loadMemoryBandwidthConfig(memBandwidthConfig)
	}

	// Load branch prediction configuration
	if branchConfig, ok := cpu.Profile.EngineSpecific["branch_prediction"].(map[string]interface{}); ok {
		cpu.loadBranchPredictionConfig(branchConfig)
	}
}

// loadCacheConfig loads cache configuration from profile
func (cpu *CPUEngine) loadCacheConfig(cache map[string]interface{}) {
	if l1Hit, ok := cache["l1_hit_ratio"].(float64); ok {
		cpu.CacheState.L1HitRatioTarget = l1Hit
	}
	if l2Hit, ok := cache["l2_hit_ratio"].(float64); ok {
		cpu.CacheState.L2HitRatioTarget = l2Hit
	}
	if l3Hit, ok := cache["l3_hit_ratio"].(float64); ok {
		cpu.CacheState.L3HitRatioTarget = l3Hit
		cpu.CacheState.ConvergedHitRatio = l3Hit
	}
	// miss_penalty is deprecated, use memory_access_multiplier instead
	if lineSize, ok := cache["cache_line_size"].(float64); ok {
		cpu.CacheState.CacheLineSize = int(lineSize)
	}
	if prefetch, ok := cache["prefetch_efficiency"].(float64); ok {
		cpu.CacheState.PrefetchEfficiency = prefetch
	}

	// Load hardware-dependent cache multipliers from profile
	if l1Mult, ok := cache["l1_hit_multiplier"].(float64); ok {
		cpu.CacheState.L1HitMultiplier = l1Mult
	}
	if l2Mult, ok := cache["l2_hit_multiplier"].(float64); ok {
		cpu.CacheState.L2HitMultiplier = l2Mult
	}
	if l3Mult, ok := cache["l3_hit_multiplier"].(float64); ok {
		cpu.CacheState.L3HitMultiplier = l3Mult
	}
	if memMult, ok := cache["memory_access_multiplier"].(float64); ok {
		cpu.CacheState.MemoryAccessMultiplier = memMult
	}
}

// loadThermalConfig loads thermal configuration from profile
func (cpu *CPUEngine) loadThermalConfig(thermal map[string]interface{}) {
	if ambient, ok := thermal["ambient_temp"].(float64); ok {
		cpu.ThermalState.AmbientTemperatureC = ambient
		cpu.ThermalState.CurrentTemperatureC = ambient
	}
	if capacity, ok := thermal["cooling_capacity"].(float64); ok {
		cpu.ThermalState.CoolingCapacity = capacity
	}
}

// loadNUMAConfig loads NUMA configuration from profile
func (cpu *CPUEngine) loadNUMAConfig(numa map[string]interface{}) {
	if nodes, ok := numa["numa_nodes"].(float64); ok {
		cpu.NUMAState.NumaNodes = int(nodes)
	}
	if penalty, ok := numa["cross_socket_penalty"].(float64); ok {
		cpu.NUMAState.CrossSocketPenalty = penalty
	}
	if ratio, ok := numa["local_memory_ratio"].(float64); ok {
		cpu.NUMAState.LocalMemoryRatio = ratio
	}
	if bandwidth, ok := numa["memory_bandwidth"].(float64); ok {
		cpu.NUMAState.MemoryBandwidthMBs = int64(bandwidth)
	}
}

// loadBoostConfig loads boost configuration from profile
func (cpu *CPUEngine) loadBoostConfig(boost map[string]interface{}) {
	if singleBoost, ok := boost["single_core_boost"].(float64); ok {
		cpu.BoostState.SingleCoreBoostGHz = singleBoost
	}
	if allBoost, ok := boost["all_core_boost"].(float64); ok {
		cpu.BoostState.AllCoreBoostGHz = allBoost
	}
	if duration, ok := boost["boost_duration"].(float64); ok {
		cpu.BoostState.BoostDurationTicks = int64(duration * 1000) // Convert seconds to ticks
	}
	if thermal, ok := boost["thermal_dependent"].(bool); ok {
		cpu.BoostState.ThermalDependent = thermal
	}
}

// loadHyperthreadingConfig loads hyperthreading configuration from profile
func (cpu *CPUEngine) loadHyperthreadingConfig(ht map[string]interface{}) {
	if enabled, ok := ht["enabled"].(bool); ok {
		cpu.HyperthreadingState.Enabled = enabled
	}
	if threads, ok := ht["threads_per_core"].(float64); ok {
		cpu.HyperthreadingState.ThreadsPerCore = int(threads)
	}
	if efficiency, ok := ht["efficiency_factor"].(float64); ok {
		cpu.HyperthreadingState.EfficiencyFactor = efficiency
	}

	// Calculate effective cores using the existing method
	cpu.applyHyperthreadingEffects()
}

// initializeFromProfile reinitializes engine state from loaded profile
func (cpu *CPUEngine) initializeFromProfile() {
	// Reinitialize convergence models with profile data
	cpu.initializeConvergenceModels()

	// Reinitialize boost clock state
	cpu.initializeBoostClockState()

	// Reinitialize NUMA state
	cpu.initializeNUMAState()

	// Reinitialize hyperthreading state
	cpu.initializeHyperthreadingState()

	// Reinitialize cache hierarchy
	cpu.initializeCacheHierarchy()

	// Initialize thermal state from profile
	cpu.initializeThermalState()

	// Initialize cache state from profile
	cpu.initializeCacheState()

	// Reset dynamic state to initial values
	cpu.ThermalState.HeatAccumulation = 0.0
	cpu.ThermalState.ThrottleActive = false
	cpu.ThermalState.ThrottleFactor = 1.0
	cpu.CacheState.CacheWarming = true
	cpu.CacheState.WarmupOperations = 0
	cpu.ActiveCores = 0
}

// initializeThermalState initializes thermal state from profile
func (cpu *CPUEngine) initializeThermalState() {
	// Get ambient temperature from profile
	ambientTemp := cpu.getProfileFloat("system_defaults", "ambient_temperature", 22.0)
	if ambientTemp == 22.0 { // Try thermal_behavior section as fallback
		ambientTemp = cpu.getProfileFloat("thermal_behavior", "ambient_temp", 22.0)
	}

	cpu.ThermalState.CurrentTemperatureC = ambientTemp
	cpu.ThermalState.AmbientTemperatureC = ambientTemp
}

// initializeCacheState initializes cache state from profile
func (cpu *CPUEngine) initializeCacheState() {
	// Initialize cache hit ratios from profile (use actual profile values, not separate cold start values)
	cpu.CacheState.L1HitRatio = cpu.getProfileFloat("cache_behavior", "l1_hit_ratio", 0.95)
	cpu.CacheState.L2HitRatio = cpu.getProfileFloat("cache_behavior", "l2_hit_ratio", 0.85)
	cpu.CacheState.L3HitRatio = cpu.getProfileFloat("cache_behavior", "l3_hit_ratio", 0.70)
	cpu.CacheState.ConvergedHitRatio = cpu.CacheState.L3HitRatio // Use L3 as converged target

	// For cache warming behavior, start with reduced ratios if warming is enabled
	if cpu.CacheState.CacheWarming {
		warmingFactor := cpu.getProfileFloat("cache_behavior", "cold_start_factor", 0.3)
		cpu.CacheState.L1HitRatio *= warmingFactor
		cpu.CacheState.L2HitRatio *= warmingFactor
		cpu.CacheState.L3HitRatio *= warmingFactor
	}
}

// initializeAdvancedFeatures initializes advanced CPU features with Intel Xeon defaults
func (cpu *CPUEngine) initializeAdvancedFeatures() {
	// Branch prediction defaults (Intel Xeon specifications)
	cpu.BranchPredictionState.BaseAccuracy = 0.96           // 96% base accuracy
	cpu.BranchPredictionState.RandomPatternAccuracy = 0.85  // 85% for random patterns
	cpu.BranchPredictionState.LoopPatternAccuracy = 0.98    // 98% for loops
	cpu.BranchPredictionState.CallReturnAccuracy = 0.99     // 99% for call/return
	cpu.BranchPredictionState.MispredictionPenalty = 0.15   // 15% penalty
	cpu.BranchPredictionState.PipelineDepth = 14            // Intel Xeon pipeline depth

	// Memory bandwidth defaults (Intel Xeon specifications - REALISTIC VALUES)
	// Real Intel Xeon Gold 6248R scales well up to 12-16 cores before significant degradation
	cpu.MemoryBandwidthState.TotalBandwidthGBps = 131.0           // 131 GB/s
	cpu.MemoryBandwidthState.PerCoreDegradation = 0.03           // 3% per additional core (realistic)
	cpu.MemoryBandwidthState.ContentionThreshold = 8             // 8+ cores cause noticeable contention
	cpu.MemoryBandwidthState.SevereContentionProbability = 0.15  // 15% chance (realistic)
	cpu.MemoryBandwidthState.SevereContentionPenalty = 0.15      // 15% penalty (realistic)

	// Advanced prefetch defaults (Intel Xeon specifications)
	cpu.AdvancedPrefetchState.HardwarePrefetchers = 4        // 4 hardware prefetchers
	cpu.AdvancedPrefetchState.SequentialAccuracy = 0.90     // 90% for sequential
	cpu.AdvancedPrefetchState.StrideAccuracy = 0.85         // 85% for stride
	cpu.AdvancedPrefetchState.PatternAccuracy = 0.75        // 75% for complex patterns
	cpu.AdvancedPrefetchState.PrefetchDistance = 8          // 8 cache lines ahead
	cpu.AdvancedPrefetchState.BandwidthUsage = 0.15         // 15% bandwidth usage
	cpu.AdvancedPrefetchState.AccessPatternHistory = make([]int64, 16) // 16-entry history

	// SIMD/Vectorization defaults (Intel Xeon specifications)
	cpu.VectorizationState.SupportedInstructions = []string{"SSE4.2", "AVX2", "AVX512"}
	cpu.VectorizationState.VectorWidth = 512                // AVX-512 (512-bit vectors)
	cpu.VectorizationState.SIMDEfficiency = 0.85            // 85% efficiency (realistic)
	cpu.VectorizationState.VectorizationRatio = 0.0         // Will be calculated dynamically
	cpu.VectorizationState.VectorOperationsCount = 0
	cpu.VectorizationState.ScalarOperationsCount = 0
	cpu.VectorizationState.AverageSpeedup = 1.0
	cpu.VectorizationState.OperationVectorizability = make(map[string]float64)

	// Initialize default vectorization ratios for common operations
	cpu.VectorizationState.OperationVectorizability["matrix_multiply"] = 0.90
	cpu.VectorizationState.OperationVectorizability["image_process"] = 0.85
	cpu.VectorizationState.OperationVectorizability["ml_inference"] = 0.80
	cpu.VectorizationState.OperationVectorizability["array_sum"] = 0.95
	cpu.VectorizationState.OperationVectorizability["vector_add"] = 0.95
	cpu.VectorizationState.OperationVectorizability["fft"] = 0.85
	cpu.VectorizationState.OperationVectorizability["crypto_hash"] = 0.70

	// Load from profile if available
	cpu.loadAdvancedFeaturesFromProfile()
}

// loadAdvancedFeaturesFromProfile loads advanced features configuration from profile
func (cpu *CPUEngine) loadAdvancedFeaturesFromProfile() {
	if cpu.Profile == nil {
		return
	}

	// Load branch prediction configuration
	if specs, ok := cpu.Profile.EngineSpecific["branch_prediction"]; ok {
		if branchPred, ok := specs.(map[string]interface{}); ok {
			if baseAcc, ok := branchPred["base_accuracy"].(float64); ok {
				cpu.BranchPredictionState.BaseAccuracy = baseAcc
			}
			if randomAcc, ok := branchPred["random_pattern_accuracy"].(float64); ok {
				cpu.BranchPredictionState.RandomPatternAccuracy = randomAcc
			}
			if loopAcc, ok := branchPred["loop_pattern_accuracy"].(float64); ok {
				cpu.BranchPredictionState.LoopPatternAccuracy = loopAcc
			}
			if callAcc, ok := branchPred["call_return_accuracy"].(float64); ok {
				cpu.BranchPredictionState.CallReturnAccuracy = callAcc
			}
			if penalty, ok := branchPred["misprediction_penalty"].(float64); ok {
				cpu.BranchPredictionState.MispredictionPenalty = penalty
			}
			if depth, ok := branchPred["pipeline_depth"].(float64); ok {
				cpu.BranchPredictionState.PipelineDepth = int(depth)
			}
		}
	}

	// Load memory bandwidth configuration
	if specs, ok := cpu.Profile.EngineSpecific["memory_bandwidth"]; ok {
		if memBand, ok := specs.(map[string]interface{}); ok {
			if totalBW, ok := memBand["total_bandwidth_gbps"].(float64); ok {
				cpu.MemoryBandwidthState.TotalBandwidthGBps = totalBW
			}
			if degradation, ok := memBand["per_core_degradation"].(float64); ok {
				cpu.MemoryBandwidthState.PerCoreDegradation = degradation
			}
			if threshold, ok := memBand["contention_threshold"].(float64); ok {
				cpu.MemoryBandwidthState.ContentionThreshold = int(threshold)
			}
			if prob, ok := memBand["severe_contention_probability"].(float64); ok {
				cpu.MemoryBandwidthState.SevereContentionProbability = prob
			}
			if penalty, ok := memBand["severe_contention_penalty"].(float64); ok {
				cpu.MemoryBandwidthState.SevereContentionPenalty = penalty
			}
		}
	}

	// Load advanced prefetch configuration
	if specs, ok := cpu.Profile.EngineSpecific["advanced_prefetch"]; ok {
		if prefetch, ok := specs.(map[string]interface{}); ok {
			if prefetchers, ok := prefetch["hardware_prefetchers"].(float64); ok {
				cpu.AdvancedPrefetchState.HardwarePrefetchers = int(prefetchers)
			}
			if seqAcc, ok := prefetch["sequential_accuracy"].(float64); ok {
				cpu.AdvancedPrefetchState.SequentialAccuracy = seqAcc
			}
			if strideAcc, ok := prefetch["stride_accuracy"].(float64); ok {
				cpu.AdvancedPrefetchState.StrideAccuracy = strideAcc
			}
			if patternAcc, ok := prefetch["pattern_accuracy"].(float64); ok {
				cpu.AdvancedPrefetchState.PatternAccuracy = patternAcc
			}
			if distance, ok := prefetch["prefetch_distance"].(float64); ok {
				cpu.AdvancedPrefetchState.PrefetchDistance = int(distance)
			}
			if usage, ok := prefetch["bandwidth_usage"].(float64); ok {
				cpu.AdvancedPrefetchState.BandwidthUsage = usage
			}
		}
	}

	// Load parallel processing configuration
	if specs, ok := cpu.Profile.EngineSpecific["parallel_processing"]; ok {
		if parallel, ok := specs.(map[string]interface{}); ok {
			if enabled, ok := parallel["enabled"].(bool); ok {
				cpu.ParallelProcessingState.Enabled = enabled
			}
			if maxRatio, ok := parallel["max_parallelizable_ratio"].(float64); ok {
				cpu.ParallelProcessingState.MaxParallelizableRatio = maxRatio
			}
			if overhead, ok := parallel["overhead_per_core"].(float64); ok {
				cpu.ParallelProcessingState.OverheadPerCore = overhead
			}
			if syncOverhead, ok := parallel["synchronization_overhead"].(float64); ok {
				cpu.ParallelProcessingState.SynchronizationOverhead = syncOverhead
			}

			// Load parallelizability map
			if parallelMap, ok := parallel["parallelizability_by_complexity"].(map[string]interface{}); ok {
				cpu.ParallelProcessingState.ParallelizabilityMap = make(map[string]float64)
				for complexity, value := range parallelMap {
					if val, ok := value.(float64); ok {
						cpu.ParallelProcessingState.ParallelizabilityMap[complexity] = val
					}
				}
			}

			// Load efficiency curve
			if efficiencyCurve, ok := parallel["efficiency_curve"].(map[string]interface{}); ok {
				cpu.ParallelProcessingState.EfficiencyCurve = make(map[string]float64)
				for cores, value := range efficiencyCurve {
					if val, ok := value.(float64); ok {
						cpu.ParallelProcessingState.EfficiencyCurve[cores] = val
					}
				}
			}
		}
	}

	// Load SIMD/Vectorization configuration
	if specs, ok := cpu.Profile.EngineSpecific["vectorization"]; ok {
		if vectorization, ok := specs.(map[string]interface{}); ok {
			if instructions, ok := vectorization["supported_instructions"].([]interface{}); ok {
				cpu.VectorizationState.SupportedInstructions = make([]string, len(instructions))
				for i, instr := range instructions {
					if instrStr, ok := instr.(string); ok {
						cpu.VectorizationState.SupportedInstructions[i] = instrStr
					}
				}
			}
			if vectorWidth, ok := vectorization["vector_width"].(float64); ok {
				cpu.VectorizationState.VectorWidth = int(vectorWidth)
			}
			if efficiency, ok := vectorization["simd_efficiency"].(float64); ok {
				cpu.VectorizationState.SIMDEfficiency = efficiency
			}

			// Load operation-specific vectorizability
			if opVectorMap, ok := vectorization["operation_vectorizability"].(map[string]interface{}); ok {
				for opType, value := range opVectorMap {
					if val, ok := value.(float64); ok {
						cpu.VectorizationState.OperationVectorizability[opType] = val
					}
				}
			}
		}
	}
}

// applyAdvancedPrefetching applies hardware prefetcher effects using statistical modeling
func (cpu *CPUEngine) applyAdvancedPrefetching(baseTime time.Duration, op *Operation) time.Duration {
	// Determine access pattern type using statistical analysis
	accessPattern := cpu.analyzeAccessPattern(op)

	// Calculate prefetch effectiveness based on pattern
	var prefetchAccuracy float64
	switch accessPattern {
	case "sequential":
		prefetchAccuracy = cpu.AdvancedPrefetchState.SequentialAccuracy
	case "stride":
		prefetchAccuracy = cpu.AdvancedPrefetchState.StrideAccuracy
	case "pattern":
		prefetchAccuracy = cpu.AdvancedPrefetchState.PatternAccuracy
	default:
		prefetchAccuracy = 0.0 // Random access - no prefetching benefit
	}

	// Use deterministic hash for consistent prefetch behavior
	opHash := cpu.hashOperationForCacheDecision(op)
	prefetchHash := opHash + 54321 // Different seed for prefetch decisions
	hashValue := float64(prefetchHash%10000) / 10000.0

	// Prefetch hit reduces memory access time
	if hashValue < prefetchAccuracy {
		// Successful prefetch: reduce memory access penalty
		prefetchBenefit := 0.3 // 30% reduction in memory access time
		return time.Duration(float64(baseTime) * (1.0 - prefetchBenefit))
	}

	// No prefetch benefit
	return baseTime
}

// applyBranchPrediction applies branch prediction effects using statistical modeling
func (cpu *CPUEngine) applyBranchPrediction(baseTime time.Duration, op *Operation) time.Duration {
	// Determine if operation has branches (based on complexity and language)
	hasBranches := cpu.operationHasBranches(op)
	if !hasBranches {
		return baseTime // No branches, no branch prediction effects
	}

	// Determine branch pattern type
	branchPattern := cpu.analyzeBranchPattern(op)

	// Get prediction accuracy based on pattern
	var predictionAccuracy float64
	switch branchPattern {
	case "loop":
		predictionAccuracy = cpu.BranchPredictionState.LoopPatternAccuracy
	case "call_return":
		predictionAccuracy = cpu.BranchPredictionState.CallReturnAccuracy
	case "random":
		predictionAccuracy = cpu.BranchPredictionState.RandomPatternAccuracy
	default:
		predictionAccuracy = cpu.BranchPredictionState.BaseAccuracy
	}

	// Use deterministic hash for consistent branch prediction behavior
	opHash := cpu.hashOperationForCacheDecision(op)
	branchHash := opHash + 98765 // Different seed for branch decisions
	hashValue := float64(branchHash%10000) / 10000.0

	// Count total branches (always)
	cpu.BranchPredictionState.TotalBranches++

	// Branch misprediction penalty
	if hashValue >= predictionAccuracy {
		// Misprediction: apply pipeline flush penalty
		cpu.BranchPredictionState.TotalMispredictions++
		penalty := cpu.BranchPredictionState.MispredictionPenalty
		return time.Duration(float64(baseTime) * (1.0 + penalty))
	}

	// Correct prediction: no penalty
	return baseTime
}

// applyMemoryBandwidthContention applies memory bandwidth contention using profile-driven scaling
func (cpu *CPUEngine) applyMemoryBandwidthContention(baseTime time.Duration, op *Operation) time.Duration {
	// Calculate cores needed for THIS operation (don't use static ActiveCores)
	coresNeeded := cpu.calculateCoresNeeded(op)

	// Skip if single core
	if coresNeeded <= 1 {
		return baseTime
	}

	// Get contention factor from profile-driven curve
	contentionFactor := cpu.getMemoryContentionFactor(coresNeeded)

	// Consider operation memory intensity (different operations stress memory differently)
	memoryIntensity := cpu.getOperationMemoryIntensity(op)

	// Apply intensity scaling: CPU-bound operations have less memory contention
	adjustedFactor := 1.0 + (contentionFactor - 1.0) * memoryIntensity

	return time.Duration(float64(baseTime) * adjustedFactor)
}

// getOperationMemoryIntensity returns how memory-intensive an operation is (0.0 to 1.0)
func (cpu *CPUEngine) getOperationMemoryIntensity(op *Operation) float64 {
	// Get from profile if available
	if cpu.Profile != nil {
		if specs, ok := cpu.Profile.EngineSpecific["memory_bandwidth"]; ok {
			if memBand, ok := specs.(map[string]interface{}); ok {
				if intensity, ok := memBand["operation_intensity"].(map[string]interface{}); ok {
					if factor, ok := intensity[op.Type].(float64); ok {
						return cpu.validateMemoryIntensity(op.Type, factor)
					}
				}
			}
		}
	}

	// Fallback based on operation type and complexity
	switch op.Type {
	case "compute":
		// CPU-bound operations have lower memory intensity
		switch op.Complexity {
		case "O(1)":
			return 0.1 // Very low memory usage
		case "O(n)", "O(n log n)":
			return 0.3 // Moderate memory usage
		case "O(n²)":
			return 0.5 // Higher memory usage for large datasets
		default:
			return 0.2 // Conservative default
		}
	case "memory":
		return 1.0 // Memory operations are fully memory-bound
	case "io":
		return 0.2 // I/O operations have moderate memory usage
	default:
		return 0.3 // Conservative default for unknown types
	}
}

// validateMemoryIntensity validates that memory intensity is in valid range
func (cpu *CPUEngine) validateMemoryIntensity(opType string, intensity float64) float64 {
	if intensity < 0.0 || intensity > 1.0 || math.IsNaN(intensity) {
		fmt.Printf("Warning: Invalid memory intensity for operation type '%s': %.3f, using 0.3\n", opType, intensity)
		return 0.3
	}
	return intensity
}

// getMemoryContentionFactor gets the memory contention factor for a given number of cores
func (cpu *CPUEngine) getMemoryContentionFactor(cores int) float64 {
	// Validate input
	if cores <= 0 {
		return 1.0 // No contention for invalid core counts
	}
	if cores == 1 {
		return 1.0 // No contention for single core
	}

	// Try to get exact match from contention curve
	key := fmt.Sprintf("%d_cores", cores)
	if factor, ok := cpu.getContentionCurveValue(key); ok {
		return cpu.validateContentionFactor(cores, factor)
	}

	// Find closest matches for interpolation
	lowerCores, lowerFactor := cpu.findLowerContentionPoint(cores)
	upperCores, upperFactor := cpu.findUpperContentionPoint(cores)

	// If we have exact bounds, interpolate
	if lowerCores != upperCores {
		// Linear interpolation
		ratio := float64(cores-lowerCores) / float64(upperCores-lowerCores)
		interpolatedFactor := lowerFactor + ratio*(upperFactor-lowerFactor)
		return cpu.validateContentionFactor(cores, interpolatedFactor)
	}

	// Fallback: extrapolate from the closest point
	if lowerCores > 0 {
		// Extrapolate beyond the curve using profile values
		maxDegradation := cpu.getProfileFloat("memory_bandwidth", "max_degradation_factor", 1.5)
		extrapolationRate := cpu.getProfileFloat("memory_bandwidth", "extrapolation_degradation_per_core", 0.02)
		extraCores := cores - lowerCores
		extraDegradation := float64(extraCores) * extrapolationRate
		extrapolatedFactor := math.Min(lowerFactor+extraDegradation, maxDegradation)
		return cpu.validateContentionFactor(cores, extrapolatedFactor)
	}

	// Ultimate fallback: minimal degradation from profile
	fallbackRate := cpu.getProfileFloat("memory_bandwidth", "fallback_degradation_per_core", 0.01)
	fallbackFactor := 1.0 + float64(cores-1)*fallbackRate
	return cpu.validateContentionFactor(cores, fallbackFactor)
}

// validateContentionFactor validates that a contention factor is reasonable
func (cpu *CPUEngine) validateContentionFactor(cores int, factor float64) float64 {
	// Contention factors must be >= 1.0 (can't be faster with more cores)
	if factor < 1.0 || math.IsNaN(factor) || math.IsInf(factor, 0) {
		fmt.Printf("Warning: Invalid memory contention factor for %d cores: %.3f, using 1.0\n", cores, factor)
		return 1.0
	}

	// Check for unrealistic contention (more than 10x slowdown is suspicious)
	if factor > 10.0 {
		fmt.Printf("Warning: Extreme memory contention factor for %d cores: %.3f (>10x slowdown)\n", cores, factor)
		// Don't change it, just warn - might be intentional for testing
	}

	return factor
}

// getContentionCurveValue gets a value from the contention curve
func (cpu *CPUEngine) getContentionCurveValue(key string) (float64, bool) {
	if cpu.Profile == nil {
		return 0, false
	}

	if specs, ok := cpu.Profile.EngineSpecific["memory_bandwidth"]; ok {
		if bandwidth, ok := specs.(map[string]interface{}); ok {
			if curve, ok := bandwidth["contention_curve"].(map[string]interface{}); ok {
				if factor, ok := curve[key].(float64); ok {
					return factor, true
				}
			}
		}
	}

	return 0, false
}

// findLowerContentionPoint finds the highest core count <= target with a defined factor
func (cpu *CPUEngine) findLowerContentionPoint(targetCores int) (int, float64) {
	bestCores := 1
	bestFactor := 1.0

	if cpu.Profile == nil {
		return bestCores, bestFactor
	}

	if specs, ok := cpu.Profile.EngineSpecific["memory_bandwidth"]; ok {
		if bandwidth, ok := specs.(map[string]interface{}); ok {
			if curve, ok := bandwidth["contention_curve"].(map[string]interface{}); ok {
				for coreKey, factorVal := range curve {
					var cores int
					if _, err := fmt.Sscanf(coreKey, "%d_cores", &cores); err == nil {
						if factor, ok := factorVal.(float64); ok {
							if cores <= targetCores && cores > bestCores {
								bestCores = cores
								bestFactor = factor
							}
						}
					}
				}
			}
		}
	}

	return bestCores, bestFactor
}

// findUpperContentionPoint finds the lowest core count >= target with a defined factor
func (cpu *CPUEngine) findUpperContentionPoint(targetCores int) (int, float64) {
	bestCores := targetCores
	bestFactor := 1.0
	found := false

	if cpu.Profile == nil {
		return bestCores, bestFactor
	}

	if specs, ok := cpu.Profile.EngineSpecific["memory_bandwidth"]; ok {
		if bandwidth, ok := specs.(map[string]interface{}); ok {
			if curve, ok := bandwidth["contention_curve"].(map[string]interface{}); ok {
				for coreKey, factorVal := range curve {
					var cores int
					if _, err := fmt.Sscanf(coreKey, "%d_cores", &cores); err == nil {
						if factor, ok := factorVal.(float64); ok {
							if cores >= targetCores && (!found || cores < bestCores) {
								bestCores = cores
								bestFactor = factor
								found = true
							}
						}
					}
				}
			}
		}
	}

	if !found {
		// No upper bound found, return the target with extrapolated factor
		lowerCores, lowerFactor := cpu.findLowerContentionPoint(targetCores)
		extraCores := targetCores - lowerCores
		extraDegradation := float64(extraCores) * 0.02 // 2% per extra core
		return targetCores, lowerFactor + extraDegradation
	}

	return bestCores, bestFactor
}



// analyzeAccessPattern determines memory access pattern using statistical analysis
func (cpu *CPUEngine) analyzeAccessPattern(op *Operation) string {
	// Update access pattern history
	cpu.AdvancedPrefetchState.AccessPatternHistory = append(
		cpu.AdvancedPrefetchState.AccessPatternHistory[1:],
		op.DataSize,
	)

	// Analyze pattern based on operation characteristics
	opHash := cpu.hashOperationForCacheDecision(op)
	patternHash := opHash + 24680 // Different seed for pattern analysis

	// Sequential access: large data sizes, linear algorithms
	if op.DataSize > 64*1024 && (op.Complexity == ComplexityON || op.Complexity == ComplexityONLogN) {
		// 80% chance of sequential access for large linear operations
		if (patternHash % 100) < 80 {
			return "sequential"
		}
	}

	// Stride access: medium data sizes, structured algorithms
	if op.DataSize > 4*1024 && op.DataSize <= 64*1024 {
		// 60% chance of stride access for medium structured operations
		if (patternHash % 100) < 60 {
			return "stride"
		}
	}

	// Pattern access: complex algorithms with predictable patterns
	if op.Complexity == ComplexityONLogN || op.Complexity == ComplexityON2 {
		// 40% chance of pattern access for complex algorithms
		if (patternHash % 100) < 40 {
			return "pattern"
		}
	}

	// Default: random access (no prefetching benefit)
	return "random"
}

// operationHasBranches determines if operation has branches based on complexity and language
func (cpu *CPUEngine) operationHasBranches(op *Operation) bool {
	// Simple operations (O(1)) typically have fewer branches
	if op.Complexity == ComplexityO1 {
		return false
	}

	// Complex operations typically have more branches
	if op.Complexity == ComplexityON2 || op.Complexity == ComplexityONLogN {
		return true
	}

	// Language-dependent: interpreted languages have more runtime branches
	if op.Language == LangPython || op.Language == LangJS {
		return true
	}

	// Medium complexity operations may have branches
	return op.Complexity == ComplexityON || op.Complexity == ComplexityOLogN
}

// analyzeBranchPattern determines branch pattern type using statistical analysis
func (cpu *CPUEngine) analyzeBranchPattern(op *Operation) string {
	opHash := cpu.hashOperationForCacheDecision(op)
	branchPatternHash := opHash + 97531 // Different seed for branch pattern analysis

	// Loop patterns: common in O(n) and O(n²) algorithms
	if op.Complexity == ComplexityON || op.Complexity == ComplexityON2 {
		// 70% chance of loop patterns for linear/quadratic algorithms
		if (branchPatternHash % 100) < 70 {
			return "loop"
		}
	}

	// Call/return patterns: common in O(log n) and recursive algorithms
	if op.Complexity == ComplexityOLogN || op.Complexity == ComplexityONLogN {
		// 60% chance of call/return patterns for logarithmic algorithms
		if (branchPatternHash % 100) < 60 {
			return "call_return"
		}
	}

	// Random patterns: unpredictable branches (worst for prediction)
	if op.Language == LangPython || op.Language == LangJS {
		// 40% chance of random patterns for interpreted languages
		if (branchPatternHash % 100) < 40 {
			return "random"
		}
	}

	// Default: predictable patterns
	return "predictable"
}

// getProfileFloat gets a float value from the profile with fallback to default
func (cpu *CPUEngine) getProfileFloat(section, key string, defaultValue float64) float64 {
	if cpu.Profile == nil {
		return defaultValue
	}

	if specs, ok := cpu.Profile.EngineSpecific[section]; ok {
		if sectionMap, ok := specs.(map[string]interface{}); ok {
			if value, ok := sectionMap[key].(float64); ok {
				return value
			}
		}
	}

	return defaultValue
}

// getProfileInt gets an int value from the profile with fallback to default
func (cpu *CPUEngine) getProfileInt(section, key string, defaultValue int) int {
	if cpu.Profile == nil {
		return defaultValue
	}

	if specs, ok := cpu.Profile.EngineSpecific[section]; ok {
		if sectionMap, ok := specs.(map[string]interface{}); ok {
			if value, ok := sectionMap[key].(float64); ok {
				return int(value)
			}
		}
	}

	return defaultValue
}

// getProfileBool gets a bool value from the profile with fallback to default
func (cpu *CPUEngine) getProfileBool(section, key string, defaultValue bool) bool {
	if cpu.Profile == nil {
		return defaultValue
	}

	if specs, ok := cpu.Profile.EngineSpecific[section]; ok {
		if sectionMap, ok := specs.(map[string]interface{}); ok {
			if value, ok := sectionMap[key].(bool); ok {
				return value
			}
		}
	}

	return defaultValue
}

// applyParallelProcessingSpeedup applies parallel processing speedup benefits
func (cpu *CPUEngine) applyParallelProcessingSpeedup(baseTime time.Duration, op *Operation, coresUsed int) time.Duration {
	if !cpu.ParallelProcessingState.Enabled || coresUsed <= 1 {
		return baseTime
	}

	// Get parallelizability for this operation type
	parallelizability := cpu.getOperationParallelizability(op)
	if parallelizability <= 0 {
		return baseTime
	}

	// Calculate theoretical speedup using correct Amdahl's Law
	// Correct formula: speedup = 1 / ((1-P) + P/N) where P=parallelizable, N=cores
	serialPortion := 1.0 - parallelizability
	parallelPortion := parallelizability / float64(coresUsed)
	theoreticalSpeedup := 1.0 / (serialPortion + parallelPortion)

	// Apply efficiency factor AFTER calculating theoretical speedup
	efficiency := cpu.getCoreEfficiency(coresUsed)
	speedup := theoreticalSpeedup * efficiency

	// Note: Synchronization overhead is now included in the efficiency calculation
	// to avoid double-penalizing multi-core performance

	// Ensure speedup is realistic - Amdahl's Law naturally limits speedup
	// Only cap at theoretical maximum (no arbitrary 95% factor)
	maxTheoreticalSpeedup := 1.0 / (1.0 - parallelizability)
	speedup = math.Min(speedup, maxTheoreticalSpeedup)

	// Additional validation: speedup cannot exceed number of cores (super-linear impossible)
	maxPossibleSpeedup := float64(coresUsed)
	speedup = math.Min(speedup, maxPossibleSpeedup)

	speedup = math.Max(speedup, 1.0) // Never slower than single-core

	return time.Duration(float64(baseTime) / speedup)
}

// getOperationParallelizability returns how parallelizable an operation is
func (cpu *CPUEngine) getOperationParallelizability(op *Operation) float64 {
	if parallelizability, ok := cpu.ParallelProcessingState.ParallelizabilityMap[op.Complexity]; ok {
		return parallelizability
	}
	return cpu.getProfileFloat("parallel_processing", "parallelizability_by_complexity."+op.Complexity, 0.5)
}

// getCoreEfficiency returns efficiency factor for given number of cores
func (cpu *CPUEngine) getCoreEfficiency(cores int) float64 {
	// Try to get exact match from efficiency curve
	key := fmt.Sprintf("%d_cores", cores)
	if efficiency, ok := cpu.ParallelProcessingState.EfficiencyCurve[key]; ok {
		// Cap efficiency at 100% - cannot be more efficient than perfect scaling
		return math.Min(efficiency, 1.0)
	}

	// Find closest match in efficiency curve and interpolate
	bestMatch := 1
	bestEfficiency := 1.0

	for coreKey, efficiency := range cpu.ParallelProcessingState.EfficiencyCurve {
		var coreCount int
		if _, err := fmt.Sscanf(coreKey, "%d_cores", &coreCount); err == nil {
			if coreCount <= cores && coreCount > bestMatch {
				bestMatch = coreCount
				bestEfficiency = efficiency
			}
		}
	}

	// Apply additional degradation for cores beyond the curve
	if cores > bestMatch {
		extraCores := cores - bestMatch
		extraDegradation := float64(extraCores) * cpu.ParallelProcessingState.OverheadPerCore
		bestEfficiency = math.Max(0.1, bestEfficiency - extraDegradation)
	}

	// Cap efficiency at 100% - cannot be more efficient than perfect scaling
	return math.Min(bestEfficiency, 1.0)
}

// loadParallelProcessingConfig loads parallel processing configuration from profile
func (cpu *CPUEngine) loadParallelProcessingConfig(config map[string]interface{}) {
	if enabled, ok := config["enabled"].(bool); ok {
		cpu.ParallelProcessingState.Enabled = enabled
	}

	if maxRatio, ok := config["max_parallelizable_ratio"].(float64); ok {
		cpu.ParallelProcessingState.MaxParallelizableRatio = maxRatio
	}

	if parallelMap, ok := config["parallelizability_by_complexity"].(map[string]interface{}); ok {
		cpu.ParallelProcessingState.ParallelizabilityMap = make(map[string]float64)
		for complexity, ratio := range parallelMap {
			if ratioFloat, ok := ratio.(float64); ok {
				cpu.ParallelProcessingState.ParallelizabilityMap[complexity] = ratioFloat
			}
		}
	}

	if efficiencyMap, ok := config["efficiency_curve"].(map[string]interface{}); ok {
		cpu.ParallelProcessingState.EfficiencyCurve = make(map[string]float64)
		for cores, efficiency := range efficiencyMap {
			if efficiencyFloat, ok := efficiency.(float64); ok {
				cpu.ParallelProcessingState.EfficiencyCurve[cores] = efficiencyFloat
			}
		}
	}

	if overhead, ok := config["overhead_per_core"].(float64); ok {
		cpu.ParallelProcessingState.OverheadPerCore = overhead
	}

	if syncOverhead, ok := config["synchronization_overhead"].(float64); ok {
		cpu.ParallelProcessingState.SynchronizationOverhead = syncOverhead
	}
}

// loadLanguageMultipliers loads language performance multipliers from profile
func (cpu *CPUEngine) loadLanguageMultipliers(config map[string]interface{}) {
	cpu.LanguageMultipliers = make(map[string]float64)
	for lang, multiplier := range config {
		if multiplierFloat, ok := multiplier.(float64); ok {
			// Validate and store the multiplier
			validatedMultiplier := cpu.validateMultiplier("language", lang, multiplierFloat)
			cpu.LanguageMultipliers[lang] = validatedMultiplier
		} else {
			fmt.Printf("Warning: Invalid language multiplier type for '%s', skipping\n", lang)
		}
	}
}

// loadComplexityFactors loads algorithm complexity factors from profile
func (cpu *CPUEngine) loadComplexityFactors(config map[string]interface{}) {
	cpu.ComplexityFactors = make(map[string]float64)
	for complexity, factor := range config {
		if factorFloat, ok := factor.(float64); ok {
			// Validate and store the factor
			validatedFactor := cpu.validateMultiplier("complexity", complexity, factorFloat)
			cpu.ComplexityFactors[complexity] = validatedFactor
		} else {
			fmt.Printf("Warning: Invalid complexity factor type for '%s', skipping\n", complexity)
		}
	}
}

// loadMemoryBandwidthConfig loads memory bandwidth configuration from profile
func (cpu *CPUEngine) loadMemoryBandwidthConfig(config map[string]interface{}) {
	if bandwidth, ok := config["total_bandwidth_gbps"].(float64); ok {
		cpu.MemoryBandwidthState.TotalBandwidthGBps = bandwidth
	}

	// The contention curve is loaded directly from profile when needed
	// No need to pre-load it into the state structure
}

// loadBranchPredictionConfig loads branch prediction configuration from profile
func (cpu *CPUEngine) loadBranchPredictionConfig(config map[string]interface{}) {
	if accuracy, ok := config["base_accuracy"].(float64); ok {
		cpu.BranchPredictionState.BaseAccuracy = accuracy
	}
	if randomAccuracy, ok := config["random_pattern_accuracy"].(float64); ok {
		cpu.BranchPredictionState.RandomPatternAccuracy = randomAccuracy
	}
	if loopAccuracy, ok := config["loop_pattern_accuracy"].(float64); ok {
		cpu.BranchPredictionState.LoopPatternAccuracy = loopAccuracy
	}
	if callAccuracy, ok := config["call_return_accuracy"].(float64); ok {
		cpu.BranchPredictionState.CallReturnAccuracy = callAccuracy
	}
	if penalty, ok := config["misprediction_penalty"].(float64); ok {
		cpu.BranchPredictionState.MispredictionPenalty = penalty
	}
	if depth, ok := config["pipeline_depth"].(float64); ok {
		cpu.BranchPredictionState.PipelineDepth = int(depth)
	}
}
