package engines

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// MemoryProcessingHeap implements heap.Interface for MemoryProcessingOperation
type MemoryProcessingHeap []*MemoryProcessingOperation

func (h MemoryProcessingHeap) Len() int           { return len(h) }
func (h MemoryProcessingHeap) Less(i, j int) bool { return h[i].CompletionTick < h[j].CompletionTick }
func (h MemoryProcessingHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *MemoryProcessingHeap) Push(x interface{}) {
	*h = append(*h, x.(*MemoryProcessingOperation))
}

func (h *MemoryProcessingHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// MemoryProcessingOperation represents an operation currently being processed
type MemoryProcessingOperation struct {
	Operation      *QueuedOperation `json:"operation"`
	StartTick      int64            `json:"start_tick"`
	CompletionTick int64            `json:"completion_tick"`
	ChannelsUsed   int              `json:"channels_used"`
	AccessPattern  string           `json:"access_pattern"` // sequential, random, stride
}

// MemoryOrderingOp represents an operation in the memory ordering window
type MemoryOrderingOp struct {
	Operation     *Operation `json:"operation"`      // The memory operation
	OriginalTick  int64      `json:"original_tick"`  // When operation was originally scheduled
	ReorderedTick int64      `json:"reordered_tick"` // When operation will actually execute
	OpType        string     `json:"op_type"`        // load, store, barrier
	Address       uint64     `json:"address"`        // Memory address (for ordering dependencies)
	CanReorder    bool       `json:"can_reorder"`    // Whether this operation can be reordered
}

// MemoryControllerRequest represents a request being processed by a memory controller
type MemoryControllerRequest struct {
	Operation      *Operation `json:"operation"`       // The memory operation
	ControllerID   int        `json:"controller_id"`   // Which controller is handling this
	QueuePosition  int        `json:"queue_position"`  // Position in controller queue
	StartTick      int64      `json:"start_tick"`      // When request started processing
	CompletionTick int64      `json:"completion_tick"` // When request will complete
	Priority       int        `json:"priority"`        // Request priority for arbitration
}

// ThermalZone represents a thermal zone for enhanced thermal modeling
type ThermalZone struct {
	ZoneID          int     `json:"zone_id"`          // Thermal zone identifier
	Temperature     float64 `json:"temperature"`      // Current temperature (°C)
	MaxTemperature  float64 `json:"max_temperature"`  // Maximum safe temperature (°C)
	HeatGeneration  float64 `json:"heat_generation"`  // Heat generation rate (W)
	CoolingCapacity float64 `json:"cooling_capacity"` // Cooling capacity (W)
	ThermalMass     float64 `json:"thermal_mass"`     // Thermal mass (J/°C)
}

// MemoryEngine implements the BaseEngine interface for RAM operations
type MemoryEngine struct {
	*CommonEngine

	// Complexity control interface (like CPU engine)
	ComplexityInterface *MemoryInterface `json:"complexity_interface"`

	// Memory-specific properties from profile (NO HARDCODED VALUES)
	CapacityGB      int64   `json:"capacity_gb"`
	MemoryType      string  `json:"memory_type"`      // DDR4, DDR5, HBM2
	FrequencyMHz    int     `json:"frequency_mhz"`
	CASLatency      int     `json:"cas_latency"`
	Channels        int     `json:"channels"`
	BandwidthGBps   float64 `json:"bandwidth_gbps"`
	AccessTimeNs    float64 `json:"access_time_ns"`
	
	// Memory processing state tracking (like CPU ActiveOperations)
	ActiveOperations *MemoryProcessingHeap `json:"active_operations"`
	BusyChannels     int                   `json:"busy_channels"`

	// Memory timing state (realistic DDR modeling)
	TimingState struct {
		tRCD            int     `json:"trcd"`             // RAS to CAS delay (from profile)
		tRP             int     `json:"trp"`              // Row precharge time (from profile)
		tRAS            int     `json:"tras"`             // Row active time (from profile)
		tREFI           int     `json:"trefi"`            // Refresh interval (from profile)
		BankGroups      int     `json:"bank_groups"`      // Number of bank groups (from profile)
		BanksPerGroup   int     `json:"banks_per_group"`  // Banks per group (from profile)
		RowBufferHitRate float64 `json:"row_buffer_hit_rate"` // Statistical hit rate
		LastTimingUpdate int64   `json:"last_timing_update"`
	} `json:"timing_state"`

	// Memory bandwidth state (realistic saturation modeling)
	BandwidthState struct {
		CurrentUtilization float64 `json:"current_utilization"` // 0.0 to 1.0
		SaturationCurve    float64 `json:"saturation_curve"`    // Performance degradation
		QueueDepth         int     `json:"queue_depth"`         // Operations waiting
		LastBandwidthUpdate int64  `json:"last_bandwidth_update"`
	} `json:"bandwidth_state"`

	// NUMA topology state (realistic cross-socket penalties)
	NUMAState struct {
		CrossSocketPenalty   float64 `json:"cross_socket_penalty"`   // From profile
		LocalAccessRatio     float64 `json:"local_access_ratio"`     // Statistical tracking
		SocketCount          int     `json:"socket_count"`           // From profile
		InterSocketLatencyNs float64 `json:"inter_socket_latency_ns"` // From profile
		LastNUMAUpdate       int64   `json:"last_numa_update"`
	} `json:"numa_state"`

	// Memory pressure state (realistic RAM pressure modeling)
	PressureState struct {
		CurrentUsageGB    float64 `json:"current_usage_gb"`    // Simulated usage
		PressureFactor    float64 `json:"pressure_factor"`    // Performance impact
		SwapPressure      float64 `json:"swap_pressure"`      // OS swap activity
		LastPressureCheck int64   `json:"last_pressure_check"`
	} `json:"pressure_state"`

	// Statistical convergence models (like CPU engine)
	ConvergenceState struct {
		Models map[string]*StatisticalModel `json:"models"`
	} `json:"convergence_state"`

	// Advanced memory features (for Maximum complexity)
	ECCState struct {
		ErrorRate           float64 `json:"error_rate"`           // ECC error rate
		CorrectedErrors     int64   `json:"corrected_errors"`     // Total corrected errors
		UncorrectedErrors   int64   `json:"uncorrected_errors"`   // Total uncorrected errors
		LastECCCheck        int64   `json:"last_ecc_check"`       // Last ECC check tick
		ECCOverhead         float64 `json:"ecc_overhead"`         // Performance overhead
	} `json:"ecc_state"`

	PowerState struct {
		CurrentState        string  `json:"current_state"`        // Active, Standby, Sleep
		PowerConsumption    float64 `json:"power_consumption"`    // Current power usage (W)
		StateTransitions    int64   `json:"state_transitions"`    // Number of state changes
		LastStateChange     int64   `json:"last_state_change"`    // Last state change tick
		PowerEfficiency     float64 `json:"power_efficiency"`     // Performance per watt
	} `json:"power_state"`

	ThermalState struct {
		Temperature         float64 `json:"temperature"`          // Current temperature (°C)
		ThermalThrottling   bool    `json:"thermal_throttling"`   // Is throttling active
		ThrottleLevel       float64 `json:"throttle_level"`       // Throttling intensity (0-1)
		LastThermalUpdate   int64   `json:"last_thermal_update"`  // Last thermal check tick
		CoolingEfficiency   float64 `json:"cooling_efficiency"`   // Cooling system efficiency
	} `json:"thermal_state"`

	// PRIORITY 1 CRITICAL FEATURES - Hardware Prefetching State
	HardwarePrefetchState struct {
		PrefetcherCount      int     `json:"prefetcher_count"`       // Number of hardware prefetchers (from profile)
		SequentialAccuracy   float64 `json:"sequential_accuracy"`    // Sequential access prefetch accuracy (from profile)
		StrideAccuracy       float64 `json:"stride_accuracy"`        // Stride access prefetch accuracy (from profile)
		PatternAccuracy      float64 `json:"pattern_accuracy"`       // Complex pattern prefetch accuracy (from profile)
		PrefetchDistance     int     `json:"prefetch_distance"`      // Lines prefetched ahead (from profile)
		BandwidthUsage       float64 `json:"bandwidth_usage"`        // Prefetch bandwidth overhead (from profile)
		AccessPatternHistory []int64 `json:"access_pattern_history"` // Recent access pattern tracking
		LastPrefetchUpdate   int64   `json:"last_prefetch_update"`   // Last prefetch state update
	} `json:"hardware_prefetch_state"`

	// PRIORITY 1 CRITICAL FEATURES - Cache Line Conflict State
	CacheLineConflictState struct {
		CacheLineSize        int     `json:"cache_line_size"`        // Cache line size in bytes (from profile)
		FalseSharingDetection bool   `json:"false_sharing_detection"` // Enable false sharing detection (from profile)
		ConflictThreshold    float64 `json:"conflict_threshold"`     // Threshold for conflict detection (from profile)
		ConflictPenalty      float64 `json:"conflict_penalty"`       // Performance penalty for conflicts (from profile)
		ActiveConflicts      int     `json:"active_conflicts"`       // Current number of conflicts
		ConflictHistory      map[uint64]int64 `json:"conflict_history"` // Cache line conflict tracking
		LastConflictUpdate   int64   `json:"last_conflict_update"`   // Last conflict state update
	} `json:"cache_line_conflict_state"`

	// PRIORITY 1 CRITICAL FEATURES - Memory Ordering State
	MemoryOrderingState struct {
		OrderingModel        string  `json:"ordering_model"`         // weak, strong, tso, pso (from profile)
		ReorderingWindow     int     `json:"reordering_window"`      // Instructions that can be reordered (from profile)
		MemoryBarrierCost    float64 `json:"memory_barrier_cost"`    // Cost of memory barriers (from profile)
		LoadStoreReordering  bool    `json:"load_store_reordering"`  // Allow load-store reordering (from profile)
		StoreStoreReordering bool    `json:"store_store_reordering"` // Allow store-store reordering (from profile)
		LoadLoadReordering   bool    `json:"load_load_reordering"`   // Allow load-load reordering (from profile)
		PendingOperations    []MemoryOrderingOp `json:"pending_operations"` // Operations in reordering window
		LastOrderingUpdate   int64   `json:"last_ordering_update"`   // Last ordering state update
	} `json:"memory_ordering_state"`

	// PRIORITY 2 IMPORTANT FEATURES - Memory Controller State
	MemoryControllerState struct {
		ControllerCount      int     `json:"controller_count"`       // Number of memory controllers (from profile)
		QueueDepth           int     `json:"queue_depth"`            // Controller queue depth (from profile)
		ArbitrationPolicy    string  `json:"arbitration_policy"`     // round_robin, priority, fair (from profile)
		BandwidthPerController float64 `json:"bandwidth_per_controller"` // Bandwidth per controller (from profile)
		ControllerLatency    float64 `json:"controller_latency"`     // Controller processing latency (from profile)
		ActiveRequests       []MemoryControllerRequest `json:"active_requests"` // Requests being processed
		LastControllerUpdate int64   `json:"last_controller_update"` // Last controller state update
	} `json:"memory_controller_state"`

	// PRIORITY 2 IMPORTANT FEATURES - Advanced NUMA State (extends existing NUMAState)
	AdvancedNUMAState struct {
		TopologyMap          map[int][]int                    `json:"topology_map"`          // NUMA node connectivity (from profile)
		DistanceMatrix       [][]float64                      `json:"distance_matrix"`       // Inter-node distance matrix (from profile)
		BandwidthMatrix      [][]float64                      `json:"bandwidth_matrix"`      // Inter-node bandwidth matrix (from profile)
		NodeMemoryPressure   []float64                        `json:"node_memory_pressure"`  // Per-node memory pressure
		NodeAffinityPolicy   string                           `json:"node_affinity_policy"`  // strict, preferred, interleave (from profile)
		MigrationThreshold   float64                          `json:"migration_threshold"`   // Threshold for page migration (from profile)
		PageAccessPatterns   map[uint64]*PageAccessPattern    `json:"page_access_patterns"`  // Page access pattern tracking
		PageNodeMapping      map[uint64]int                   `json:"page_node_mapping"`     // Current page locations
		MigrationCount       int64                            `json:"migration_count"`       // Total migrations performed
		LastMigrationTick    int64                            `json:"last_migration_tick"`   // Last migration tick
		LastNUMAUpdate       int64                            `json:"last_numa_update"`      // Last NUMA state update
	} `json:"advanced_numa_state"`

	// PRIORITY 2 IMPORTANT FEATURES - Virtual Memory State
	VirtualMemoryState struct {
		PageSize             int     `json:"page_size"`              // Page size in bytes (from profile)
		TLBSize              int     `json:"tlb_size"`               // TLB entries (from profile)
		TLBHitRatio          float64 `json:"tlb_hit_ratio"`          // TLB hit ratio (from profile)
		PageTableLevels      int     `json:"page_table_levels"`      // Page table levels (from profile)
		PageWalkLatency      float64 `json:"page_walk_latency"`      // Page table walk latency (from profile)
		SwapEnabled          bool    `json:"swap_enabled"`           // Swap enabled (from profile)
		SwapLatency          float64 `json:"swap_latency"`           // Swap access latency (from profile)
		TLBMissCount         int64   `json:"tlb_miss_count"`         // TLB miss counter
		PageFaultCount       int64   `json:"page_fault_count"`       // Page fault counter
		LastVMUpdate         int64   `json:"last_vm_update"`         // Last VM state update
	} `json:"virtual_memory_state"`

	// PRIORITY 3 ENHANCEMENT FEATURES - ECC Error Modeling State
	ECCModelingState struct {
		ECCEnabled           bool    `json:"ecc_enabled"`            // ECC enabled (from profile)
		SingleBitErrorRate   float64 `json:"single_bit_error_rate"`  // Single-bit error rate per GB/hour (from profile)
		MultiBitErrorRate    float64 `json:"multi_bit_error_rate"`   // Multi-bit error rate per GB/hour (from profile)
		CorrectionLatency    float64 `json:"correction_latency"`     // ECC correction latency (from profile)
		DetectionLatency     float64 `json:"detection_latency"`      // ECC detection latency (from profile)
		SingleBitErrorCount  int64   `json:"single_bit_error_count"` // Single-bit error counter
		MultiBitErrorCount   int64   `json:"multi_bit_error_count"`  // Multi-bit error counter
		LastECCUpdate        int64   `json:"last_ecc_update"`        // Last ECC state update
	} `json:"ecc_modeling_state"`

	// PRIORITY 3 ENHANCEMENT FEATURES - Power State Transitions
	PowerStateTransitions struct {
		CurrentPowerState    string  `json:"current_power_state"`    // active, standby, sleep, deep_sleep (from profile)
		StateTransitionCost  float64 `json:"state_transition_cost"`  // Cost of state transitions (from profile)
		ActivePowerDraw      float64 `json:"active_power_draw"`      // Power draw in active state (from profile)
		StandbyPowerDraw     float64 `json:"standby_power_draw"`     // Power draw in standby state (from profile)
		SleepPowerDraw       float64 `json:"sleep_power_draw"`       // Power draw in sleep state (from profile)
		WakeupLatency        float64 `json:"wakeup_latency"`         // Wakeup latency from sleep states (from profile)
		IdleThreshold        float64 `json:"idle_threshold"`         // Idle time before state transition (from profile)
		LastPowerUpdate      int64   `json:"last_power_update"`      // Last power state update
		IdleStartTick        int64   `json:"idle_start_tick"`        // When idle period started
	} `json:"power_state_transitions"`

	// PRIORITY 3 ENHANCEMENT FEATURES - Enhanced Thermal State (extends existing ThermalState)
	EnhancedThermalState struct {
		ThermalZones         []ThermalZone `json:"thermal_zones"`         // Multiple thermal zones (from profile)
		ThrottlingThresholds []float64     `json:"throttling_thresholds"` // Temperature thresholds for throttling (from profile)
		ThrottlingLevels     []float64     `json:"throttling_levels"`     // Throttling intensity levels (from profile)
		HeatDissipationRate  float64       `json:"heat_dissipation_rate"` // Heat dissipation rate (from profile)
		ThermalCapacity      float64       `json:"thermal_capacity"`      // Thermal capacity (from profile)
		AmbientTemperature   float64       `json:"ambient_temperature"`   // Ambient temperature (from profile)
		LastThermalUpdate    int64         `json:"last_thermal_update"`   // Last thermal state update
	} `json:"enhanced_thermal_state"`

	// Unique state identity for save/load
	StateIdentity struct {
		EngineID        string `json:"engine_id"`        // Unique engine identifier
		ProfileName     string `json:"profile_name"`     // Profile used
		ComplexityLevel int    `json:"complexity_level"` // Complexity level
		CreatedAt       int64  `json:"created_at"`       // Creation timestamp
		LastSaved       int64  `json:"last_saved"`       // Last save timestamp
	} `json:"state_identity"`
}

// NewMemoryEngine creates a new Memory engine with profile-driven configuration (NO HARDCODED VALUES)
func NewMemoryEngine(queueCapacity int) *MemoryEngine {
	common := NewCommonEngine(MemoryEngineType, queueCapacity)

	// Initialize processing heap (like CPU engine)
	activeOps := &MemoryProcessingHeap{}
	heap.Init(activeOps)

	mem := &MemoryEngine{
		CommonEngine:        common,
		ComplexityInterface: NewMemoryInterface(ComplexityAdvanced), // Default to advanced complexity
		ActiveOperations:    activeOps,
		BusyChannels:        0,

		// All values will be loaded from profile - NO DEFAULTS
		CapacityGB:    0,
		MemoryType:    "",
		FrequencyMHz:  0,
		CASLatency:    0,
		Channels:      0,
		BandwidthGBps: 0.0,
		AccessTimeNs:  0.0,
	}
	
	// Initialize state structures (values loaded from profile)
	mem.TimingState.RowBufferHitRate = 0.0  // Will be calculated statistically
	mem.TimingState.LastTimingUpdate = 0

	mem.BandwidthState.CurrentUtilization = 0.0
	mem.BandwidthState.SaturationCurve = 1.0
	mem.BandwidthState.QueueDepth = 0
	mem.BandwidthState.LastBandwidthUpdate = 0

	mem.NUMAState.LocalAccessRatio = 1.0  // Start with 100% local access
	mem.NUMAState.LastNUMAUpdate = 0

	mem.PressureState.CurrentUsageGB = 0.0
	mem.PressureState.PressureFactor = 1.0
	mem.PressureState.SwapPressure = 0.0
	mem.PressureState.LastPressureCheck = 0

	// Initialize Priority 1 & 2 Features state structures
	mem.initializePriorityFeatures()

	// Initialize statistical convergence models (like CPU engine)
	mem.initializeConvergenceModels()

	// Initialize advanced memory features
	mem.initializeAdvancedFeatures()

	// Initialize state identity
	mem.initializeStateIdentity()

	return mem
}

// ProcessOperation processes a single memory operation with realistic RAM modeling
func (mem *MemoryEngine) ProcessOperation(op *Operation, currentTick int64) *OperationResult {
	mem.CurrentTick = currentTick

	// Calculate base memory access time from profile (DDR timings)
	baseTime := mem.calculateBaseMemoryAccessTime(op)

	// Apply memory timing effects (if enabled)
	timingAdjustedTime := baseTime
	if mem.ComplexityInterface.ShouldEnableFeature("ddr_timing_effects") {
		timingAdjustedTime = mem.applyMemoryTimingEffects(baseTime, op)
	}

	// Apply bandwidth saturation effects (if enabled)
	bandwidthAdjustedTime := timingAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("bandwidth_saturation") {
		bandwidthAdjustedTime = mem.applyBandwidthSaturation(timingAdjustedTime)
	}

	// Apply NUMA effects (if enabled)
	numaAdjustedTime := bandwidthAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("basic_numa") {
		numaAdjustedTime = mem.applyNUMAEffects(bandwidthAdjustedTime, op)
	}

	// Apply memory pressure effects (if enabled)
	pressureAdjustedTime := numaAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("memory_pressure") {
		pressureAdjustedTime = mem.applyMemoryPressure(numaAdjustedTime)
	}

	// Apply garbage collection effects (if enabled)
	gcAdjustedTime := pressureAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("garbage_collection") {
		gcAdjustedTime = mem.applyGarbageCollectionEffects(pressureAdjustedTime, op)
	}

	// Apply memory fragmentation effects (if enabled)
	fragmentationAdjustedTime := gcAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("memory_fragmentation") {
		fragmentationAdjustedTime = mem.applyMemoryFragmentationEffects(gcAdjustedTime, op)
	}

	// PRIORITY 1 CRITICAL FEATURES - Apply hardware prefetching effects (if enabled)
	prefetchAdjustedTime := fragmentationAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("hardware_prefetching") {
		prefetchAdjustedTime = mem.applyHardwarePrefetchingEffects(fragmentationAdjustedTime, op)
	}

	// PRIORITY 1 CRITICAL FEATURES - Apply cache line conflict effects (if enabled)
	conflictAdjustedTime := prefetchAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("cache_line_conflicts") {
		conflictAdjustedTime = mem.applyCacheLineConflictEffects(prefetchAdjustedTime, op)
	}

	// PRIORITY 1 CRITICAL FEATURES - Apply memory ordering effects (if enabled)
	orderingAdjustedTime := conflictAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("memory_ordering") {
		orderingAdjustedTime = mem.applyMemoryOrderingEffects(conflictAdjustedTime, op)
	}

	// PRIORITY 2 IMPORTANT FEATURES - Apply memory controller effects (if enabled)
	controllerAdjustedTime := orderingAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("memory_controller") {
		controllerAdjustedTime = mem.applyMemoryControllerEffects(orderingAdjustedTime, op)
	}

	// PRIORITY 2 IMPORTANT FEATURES - Apply advanced NUMA effects (if enabled)
	advancedNUMAAdjustedTime := controllerAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("numa_optimization") {
		advancedNUMAAdjustedTime = mem.applyAdvancedNUMAEffects(controllerAdjustedTime, op)
	}

	// PRIORITY 2 IMPORTANT FEATURES - Apply virtual memory effects (if enabled)
	virtualMemoryAdjustedTime := advancedNUMAAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("virtual_memory") {
		virtualMemoryAdjustedTime = mem.applyVirtualMemoryEffects(advancedNUMAAdjustedTime, op)
	}

	// PRIORITY 3 ENHANCEMENT FEATURES - Apply ECC error modeling effects (if enabled)
	eccAdjustedTime := virtualMemoryAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("ecc_modeling") {
		eccAdjustedTime = mem.applyECCModelingEffects(virtualMemoryAdjustedTime, op)
	}

	// PRIORITY 3 ENHANCEMENT FEATURES - Apply power state effects (if enabled)
	powerAdjustedTime := eccAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("power_states") {
		powerAdjustedTime = mem.applyPowerStateEffects(eccAdjustedTime, op)
	}

	// PRIORITY 3 ENHANCEMENT FEATURES - Apply enhanced thermal throttling effects (if enabled)
	thermalAdjustedTime := powerAdjustedTime
	if mem.ComplexityInterface.ShouldEnableFeature("thermal_throttling") {
		thermalAdjustedTime = mem.applyEnhancedThermalThrottlingEffects(powerAdjustedTime, op)
	}

	// Apply common performance factors (load, queue, health, variance)
	utilization := mem.calculateCurrentUtilization()
	finalTime := mem.ApplyCommonPerformanceFactors(thermalAdjustedTime, utilization)

	// Update dynamic state tracking (if enabled)
	if mem.ComplexityInterface.ShouldEnableFeature("dynamic_behavior") {
		mem.updateMemoryState(op, finalTime)
	}

	// Calculate penalty factors for routing decisions
	loadPenalty := 1.0 + (utilization * 0.4) // Memory utilization penalty
	queuePenalty := 1.0 + (float64(mem.GetQueueLength()) / float64(mem.GetQueueCapacity()) * 0.3)
	thermalPenalty := 1.0 // Memory engines typically don't have thermal throttling
	// Ensure bandwidth utilization is valid
	bandwidthUtil := mem.BandwidthState.CurrentUtilization
	if bandwidthUtil < 0 || bandwidthUtil > 1 || bandwidthUtil != bandwidthUtil { // Check for NaN
		bandwidthUtil = 0.0
	}
	contentionPenalty := 1.0 + (bandwidthUtil * 0.3) // Bandwidth contention
	healthPenalty := 1.0 + (1.0 - mem.GetHealth().Score) * 0.2

	// NUMA penalty is significant for memory operations
	numaPenalty := mem.NUMAState.CrossSocketPenalty
	if numaPenalty <= 0 {
		numaPenalty = 1.0 // Default to no penalty if not initialized
	}
	pressurePenalty := mem.PressureState.PressureFactor
	if pressurePenalty <= 0 {
		pressurePenalty = 1.0 // Default to no penalty if not initialized
	}

	totalPenaltyFactor := loadPenalty * queuePenalty * contentionPenalty * healthPenalty * numaPenalty * pressurePenalty

	// Ensure total penalty factor is valid
	if totalPenaltyFactor != totalPenaltyFactor || totalPenaltyFactor <= 0 { // Check for NaN or invalid values
		totalPenaltyFactor = 1.0
	}

	// Determine performance grade
	performanceGrade := "A"
	recommendedAction := "continue"
	if totalPenaltyFactor > 2.5 {
		performanceGrade = "F"
		recommendedAction = "redirect"
	} else if totalPenaltyFactor > 2.0 {
		performanceGrade = "D"
		recommendedAction = "throttle"
	} else if totalPenaltyFactor > 1.5 {
		performanceGrade = "C"
		recommendedAction = "throttle"
	} else if totalPenaltyFactor > 1.2 {
		performanceGrade = "B"
	}

	// Create result with penalty information
	result := &OperationResult{
		OperationID:    op.ID,
		ProcessingTime: finalTime,
		CompletedTick:  currentTick + mem.DurationToTicks(finalTime),
		Success:        true,
		PenaltyInfo: &PenaltyInformation{
			EngineType:           MemoryEngineType,
			EngineID:            mem.ID,
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
			MemoryPenalties: &MemoryPenaltyDetails{
				BandwidthUtilization: bandwidthUtil,
				NUMAPenalty:         numaPenalty,
				RowBufferHitRate:    mem.TimingState.RowBufferHitRate,
				MemoryPressure:      pressurePenalty,
				ChannelContention:   func() float64 {
					if mem.Channels > 0 {
						return float64(mem.BusyChannels) / float64(mem.Channels)
					}
					return 0.0
				}(),
			},
		},
		Metrics: map[string]interface{}{
			"base_time_ns":         float64(baseTime) / float64(time.Nanosecond),
			"timing_penalty":       float64(timingAdjustedTime) / float64(baseTime),
			"bandwidth_utilization": mem.BandwidthState.CurrentUtilization,
			"numa_penalty":         mem.NUMAState.CrossSocketPenalty,
			"pressure_factor":      mem.PressureState.PressureFactor,
			"utilization":          utilization,
			"channels_used":        mem.BusyChannels,
			"row_buffer_hit_rate":  mem.TimingState.RowBufferHitRate,
		},
	}

	// Update operation history for convergence
	mem.AddOperationToHistory(finalTime)
	mem.CompletedOps++

	return result
}

// ProcessTick processes one simulation tick (SIMPLIFIED like CPU engine)
func (mem *MemoryEngine) ProcessTick(currentTick int64) []OperationResult {
	mem.CurrentTick = currentTick
	results := make([]OperationResult, 0)

	// STEP 1: Check for completed operations and move to output (like CPU engine)
	completedOps := mem.checkCompletedOperations(currentTick)
	results = append(results, completedOps...)

	// STEP 2: Start new operations from input queue (like CPU engine)
	mem.startNewOperationsFromQueue(currentTick)

	// STEP 3: Update metrics based on actual busy state (like CPU engine)
	mem.UpdateHealth()
	mem.UpdateDynamicBehavior()

	return results
}

// checkCompletedOperations checks for operations that completed this tick (like CPU engine)
func (mem *MemoryEngine) checkCompletedOperations(currentTick int64) []OperationResult {
	completed := make([]OperationResult, 0)

	// Check heap root for completed operations (like CPU engine)
	for mem.ActiveOperations.Len() > 0 {
		// Peek at the next completion without removing it
		nextOp := (*mem.ActiveOperations)[0]

		if nextOp.CompletionTick <= currentTick {
			// Operation completed - remove from heap
			completedOp := heap.Pop(mem.ActiveOperations).(*MemoryProcessingOperation)

			// Create result (like CPU engine)
			result := OperationResult{
				OperationID:    completedOp.Operation.Operation.ID,
				OperationType:  completedOp.Operation.Operation.Type, // For routing decisions
				CompletedAt:    currentTick,
				CompletedTick:  currentTick,
				ProcessingTime: time.Duration(completedOp.CompletionTick - completedOp.StartTick) * mem.TickDuration,
				Success:        true,
				NextComponent:  completedOp.Operation.Operation.NextComponent, // For routing
			}
			completed = append(completed, result)

			// Free the channels (channels are released when operation completes)
			mem.BusyChannels -= completedOp.ChannelsUsed
		} else {
			// No more completed operations this tick
			break
		}
	}

	return completed
}

// startNewOperationsFromQueue starts new operations from input queue (like CPU engine)
func (mem *MemoryEngine) startNewOperationsFromQueue(currentTick int64) {
	maxQueuedOpsPerTick := mem.getProfileInt("queue_processing", "max_ops_per_tick", 3)
	opsStartedThisTick := 0

	for mem.GetQueueLength() > 0 && opsStartedThisTick < maxQueuedOpsPerTick {
		// Check heap length before accepting operations (like CPU engine)
		if mem.ActiveOperations.Len() >= mem.getMaxInternalQueueSize() {
			// Heap is full - cannot accept new operations (realistic constraint)
			break
		}

		// Check if we have available channels
		availableChannels := mem.Channels - mem.BusyChannels
		if availableChannels <= 0 {
			break // No channels available
		}

		queuedOp := mem.DequeueOperation()
		if queuedOp == nil {
			break
		}

		// Calculate channel requirements
		channelsNeeded := mem.calculateChannelsNeeded(queuedOp.Operation)

		if channelsNeeded <= availableChannels {
			// Start processing
			mem.startProcessing(queuedOp, currentTick)
			opsStartedThisTick++
		} else {
			// Not enough channels - put back in queue
			mem.QueueOperation(queuedOp.Operation)
			break
		}
	}
}

// startProcessing begins processing an operation (like CPU engine)
func (mem *MemoryEngine) startProcessing(queuedOp *QueuedOperation, currentTick int64) {
	// Calculate processing time using existing logic
	processingTime := mem.calculateProcessingTimeForOperation(queuedOp.Operation)
	channelsNeeded := mem.calculateChannelsNeeded(queuedOp.Operation)
	completionTick := currentTick + mem.DurationToTicks(processingTime)

	// Create processing operation
	activeOp := &MemoryProcessingOperation{
		Operation:      queuedOp,
		StartTick:      currentTick,
		CompletionTick: completionTick,
		ChannelsUsed:   channelsNeeded,
		AccessPattern:  mem.determineAccessPattern(queuedOp.Operation),
	}

	// Add to processing heap
	heap.Push(mem.ActiveOperations, activeOp)
	mem.BusyChannels += channelsNeeded
}

// DurationToTicks converts a duration to number of ticks (like CPU engine)
func (mem *MemoryEngine) DurationToTicks(duration time.Duration) int64 {
	if mem.TickDuration == 0 {
		return 1 // Fallback to prevent division by zero
	}

	// Round UP to ensure operations take at least 1 tick
	ticks := int64(math.Ceil(float64(duration) / float64(mem.TickDuration)))

	// Ensure minimum of 1 tick
	if ticks < 1 {
		return 1
	}

	return ticks
}

// calculateProcessingTimeForOperation calculates processing time for an operation (EXACTLY like CPU engine)
func (mem *MemoryEngine) calculateProcessingTimeForOperation(op *Operation) time.Duration {
	// Calculate base memory access time from profile (like CPU engine's base time)
	baseTime := mem.calculateBaseMemoryAccessTime(op)

	// Apply common performance factors ONLY (like CPU engine)
	// Skip all complex memory-specific features for now
	utilization := mem.calculateCurrentUtilization()
	finalTime := mem.ApplyCommonPerformanceFactors(baseTime, utilization)

	return finalTime
}

// applySimpleMemoryTimingEffects applies simple DDR timing overhead (like CPU engine applies CPU overhead)
func (mem *MemoryEngine) applySimpleMemoryTimingEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Simple row buffer hit/miss calculation (like CPU cache effects)
	rowBufferHitRate := mem.getProfileFloat("ddr_timings", "row_buffer_hit_rate", 0.8) // 80% hit rate default

	// Use deterministic hash for consistent behavior
	opHash := mem.hashOperationForMemoryDecision(op)
	hashValue := float64(opHash%10000) / 10000.0

	if hashValue < rowBufferHitRate {
		// Row buffer hit - no additional latency
		return baseTime
	} else {
		// Row buffer miss - add precharge + activate latency
		additionalLatency := mem.getProfileFloat("ddr_timings", "row_buffer_miss_penalty", 15.0) // 15ns default
		return baseTime + time.Duration(additionalLatency*float64(time.Nanosecond))
	}
}

// applySimpleBandwidthSaturation applies simple bandwidth saturation overhead (like CPU memory contention)
func (mem *MemoryEngine) applySimpleBandwidthSaturation(baseTime time.Duration) time.Duration {
	utilization := mem.calculateCurrentUtilization()

	// Simple saturation curve (like CPU engine load degradation)
	if utilization > 0.8 {
		// High utilization - apply penalty
		saturationPenalty := 1.0 + (utilization-0.8)*2.0 // Up to 1.4x penalty at 100%
		return time.Duration(float64(baseTime) * saturationPenalty)
	}

	return baseTime
}

// applySimpleNUMAEffects applies simple NUMA overhead (like CPU NUMA effects)
func (mem *MemoryEngine) applySimpleNUMAEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Simple cross-socket penalty calculation
	crossSocketPenalty := mem.getProfileFloat("basic_numa", "cross_socket_penalty", 1.3) // 30% penalty default
	crossSocketProb := mem.getProfileFloat("basic_numa", "cross_socket_probability", 0.2) // 20% cross-socket default

	// Use deterministic hash for consistent behavior
	opHash := mem.hashOperationForMemoryDecision(op)
	hashValue := float64(opHash%10000) / 10000.0

	if hashValue < crossSocketProb {
		// Cross-socket access - apply penalty
		return time.Duration(float64(baseTime) * crossSocketPenalty)
	}

	return baseTime
}

// calculateBaseMemoryAccessTime calculates base RAM access time from profile (SIMPLIFIED like CPU engine)
func (mem *MemoryEngine) calculateBaseMemoryAccessTime(op *Operation) time.Duration {
	// Simple base access time from profile (like CPU engine's base processing time)
	baseAccessTimeNs := mem.getProfileFloat("baseline_performance", "access_time", 10.0) // Fallback: 10ns

	// Simple frequency scaling (like CPU frequency scaling)
	normalizationBaseline := mem.getProfileFloat("baseline_performance", "frequency_normalization_baseline", 3200.0)
	actualFrequency := float64(mem.FrequencyMHz)
	if actualFrequency == 0 {
		actualFrequency = 3200.0 // Fallback
	}

	// Higher frequency = lower latency (simple linear scaling)
	frequencyScaling := normalizationBaseline / actualFrequency
	scaledAccessTime := baseAccessTimeNs * frequencyScaling

	// Convert to duration
	duration := time.Duration(scaledAccessTime * float64(time.Nanosecond))

	// Ensure minimum of 1ns
	if duration < time.Nanosecond {
		duration = time.Nanosecond
	}

	return duration
}

// applyMemoryTimingEffects applies realistic DDR timing effects (row buffer hits, bank conflicts)
func (mem *MemoryEngine) applyMemoryTimingEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Determine if this is a row buffer hit using deterministic hash
	isRowBufferHit := mem.determineRowBufferHit(op)

	if isRowBufferHit {
		// Row buffer hit - optimal case, just CAS latency
		return baseTime
	} else {
		// Row buffer miss - need precharge + activate + CAS
		// tRP (precharge) + tRCD (activate) + CAS latency
		additionalCycles := float64(mem.TimingState.tRP + mem.TimingState.tRCD)
		additionalNs := (additionalCycles / float64(mem.FrequencyMHz)) * 1000.0

		return baseTime + time.Duration(additionalNs*float64(time.Nanosecond))
	}
}

// applyBandwidthSaturation applies realistic memory bandwidth saturation effects
func (mem *MemoryEngine) applyBandwidthSaturation(baseTime time.Duration) time.Duration {
	// Calculate current bandwidth utilization
	utilization := mem.BandwidthState.CurrentUtilization

	// Apply saturation curve (performance degrades as utilization increases)
	// Based on real memory controller behavior
	if utilization > 0.8 {
		// Severe degradation above 80% utilization
		saturationPenalty := 1.0 + (utilization-0.8)*5.0 // Up to 2x penalty at 100%
		return time.Duration(float64(baseTime) * saturationPenalty)
	} else if utilization > 0.6 {
		// Moderate degradation above 60% utilization
		saturationPenalty := 1.0 + (utilization-0.6)*1.5 // Up to 1.3x penalty at 80%
		return time.Duration(float64(baseTime) * saturationPenalty)
	}

	return baseTime
}

// calculateCurrentUtilization calculates current memory utilization
func (mem *MemoryEngine) calculateCurrentUtilization() float64 {
	if mem.CapacityGB == 0 {
		return 0.0
	}
	return mem.PressureState.CurrentUsageGB / float64(mem.CapacityGB)
}

// applyNUMAEffects applies cross-socket memory access penalties from profile
func (mem *MemoryEngine) applyNUMAEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Only apply NUMA effects if we have multiple sockets
	if mem.NUMAState.SocketCount <= 1 {
		return baseTime
	}

	// Determine if this is a cross-socket access using deterministic hash
	isCrossSocket := mem.determineCrossSocketAccess(op)

	if isCrossSocket {
		// Apply cross-socket penalty from profile
		penalty := mem.NUMAState.CrossSocketPenalty
		additionalLatency := time.Duration(mem.NUMAState.InterSocketLatencyNs * float64(time.Nanosecond))
		return time.Duration(float64(baseTime)*penalty) + additionalLatency
	}

	return baseTime
}

// applyMemoryPressure applies RAM pressure effects (realistic OS behavior)
func (mem *MemoryEngine) applyMemoryPressure(baseTime time.Duration) time.Duration {
	utilization := mem.calculateCurrentUtilization()

	// Apply pressure effects based on memory usage
	if utilization > 0.9 {
		// Critical pressure - OS starts aggressive swapping
		mem.PressureState.SwapPressure = (utilization - 0.9) * 10.0 // 0-1.0 scale
		pressurePenalty := 1.0 + mem.PressureState.SwapPressure*3.0 // Up to 4x penalty
		return time.Duration(float64(baseTime) * pressurePenalty)
	} else if utilization > 0.7 {
		// Moderate pressure - page cache eviction
		pressurePenalty := 1.0 + (utilization-0.7)*1.5 // Up to 1.3x penalty
		return time.Duration(float64(baseTime) * pressurePenalty)
	}

	mem.PressureState.SwapPressure = 0.0
	return baseTime
}

// determineRowBufferHit determines if memory access hits row buffer (deterministic)
func (mem *MemoryEngine) determineRowBufferHit(op *Operation) bool {
	// Create deterministic hash based on operation characteristics
	opHash := mem.hashOperationForMemoryDecision(op)

	// Convert hash to probability value
	hashValue := float64(opHash%10000) / 10000.0

	// Row buffer hit rate depends on access pattern (from profile or calculated)
	hitRate := mem.TimingState.RowBufferHitRate
	if hitRate == 0.0 {
		// Calculate based on access pattern
		hitRate = mem.calculateRowBufferHitRate(op)
	}

	return hashValue < hitRate
}

// determineCrossSocketAccess determines if access is cross-socket (deterministic)
func (mem *MemoryEngine) determineCrossSocketAccess(op *Operation) bool {
	// Create deterministic hash
	opHash := mem.hashOperationForMemoryDecision(op)

	// Convert to probability
	hashValue := float64(opHash%10000) / 10000.0

	// Cross-socket probability (inverse of local access ratio)
	crossSocketProb := 1.0 - mem.NUMAState.LocalAccessRatio

	return hashValue < crossSocketProb
}

// hashOperationForMemoryDecision creates deterministic hash for memory decisions
func (mem *MemoryEngine) hashOperationForMemoryDecision(op *Operation) uint32 {
	// Combine operation characteristics for deterministic behavior
	hash := uint32(0)

	// Hash operation ID
	for _, b := range []byte(op.ID) {
		hash = hash*31 + uint32(b)
	}

	// Include data size and type
	hash = hash*31 + uint32(op.DataSize%1000)
	hash = hash*31 + uint32(len(op.Type))

	// Include current tick for temporal variation
	hash = hash*31 + uint32(mem.CurrentTick%1000)

	return hash
}

// calculateChannelsNeeded determines how many memory channels an operation needs (CONSERVATIVE like CPU cores)
func (mem *MemoryEngine) calculateChannelsNeeded(op *Operation) int {
	// Most operations use only 1 channel (like CPU operations use 1 core)
	// This allows multiple operations to run concurrently

	// Only very large operations (>10MB) might use multiple channels
	if op.DataSize > 10*1024*1024 { // > 10MB
		return 2 // Use at most 2 channels for very large transfers
	}

	return 1 // Single channel for all normal operations (like CPU engine)
}

// determineAccessPattern determines the memory access pattern
func (mem *MemoryEngine) determineAccessPattern(op *Operation) string {
	// Determine based on operation characteristics
	if op.DataSize > 1024*1024 {
		return "sequential" // Large transfers are typically sequential
	} else if op.Type == "memory_read" || op.Type == "memory_write" {
		return "random" // Small reads/writes are typically random
	}

	return "stride" // Default to stride pattern
}

// calculateRowBufferHitRate calculates expected row buffer hit rate
func (mem *MemoryEngine) calculateRowBufferHitRate(op *Operation) float64 {
	// Row buffer hit rate depends on access pattern
	switch mem.determineAccessPattern(op) {
	case "sequential":
		return 0.85 // High hit rate for sequential access
	case "random":
		return 0.15 // Low hit rate for random access
	case "stride":
		return 0.45 // Medium hit rate for stride access
	default:
		return 0.30 // Conservative default
	}
}

// updateMemoryState updates memory state tracking
func (mem *MemoryEngine) updateMemoryState(op *Operation, processingTime time.Duration) {
	// Update bandwidth utilization
	dataTransferred := float64(op.DataSize) / (1024 * 1024 * 1024) // GB
	transferTime := processingTime.Seconds()
	if transferTime > 0 {
		currentBandwidth := dataTransferred / transferTime
		mem.BandwidthState.CurrentUtilization = currentBandwidth / mem.BandwidthGBps
	}

	// Update row buffer hit rate (statistical convergence)
	if model, exists := mem.ConvergenceState.Models["row_buffer_hits"]; exists {
		// Simple convergence - gradually move toward convergence point
		if mem.TotalOperations > model.MinOperations {
			convergenceRate := 0.01 // 1% per operation
			mem.TimingState.RowBufferHitRate += (model.ConvergencePoint - mem.TimingState.RowBufferHitRate) * convergenceRate
			model.CurrentValue = mem.TimingState.RowBufferHitRate
		}
	}

	// Update NUMA access ratio
	mem.NUMAState.LocalAccessRatio = mem.calculateLocalAccessRatio()
}

// updateMemoryStatistics updates memory statistics and convergence models
func (mem *MemoryEngine) updateMemoryStatistics(currentTick int64) {
	// Update bandwidth state
	mem.BandwidthState.QueueDepth = mem.GetQueueLength()
	mem.BandwidthState.LastBandwidthUpdate = currentTick

	// Update timing state
	mem.TimingState.LastTimingUpdate = currentTick

	// Update NUMA state
	mem.NUMAState.LastNUMAUpdate = currentTick

	// Update pressure state
	mem.PressureState.LastPressureCheck = currentTick

	// Update convergence models (simple convergence tracking)
	for _, model := range mem.ConvergenceState.Models {
		if mem.TotalOperations > model.MinOperations {
			// Mark as converged if close enough to convergence point
			if math.Abs(model.CurrentValue - model.ConvergencePoint) < 0.01 {
				model.IsConverged = true
			}
		}
	}
}

// calculateLocalAccessRatio calculates the ratio of local vs cross-socket accesses
func (mem *MemoryEngine) calculateLocalAccessRatio() float64 {
	// This would be updated based on actual access patterns
	// For now, return current value (will be updated by statistical models)
	return mem.NUMAState.LocalAccessRatio
}

// initializeConvergenceModels initializes statistical convergence models (like CPU engine)
func (mem *MemoryEngine) initializeConvergenceModels() {
	mem.ConvergenceState.Models = make(map[string]*StatisticalModel)

	// Row buffer hit rate convergence
	mem.ConvergenceState.Models["row_buffer_hits"] = &StatisticalModel{
		Name:             "row_buffer_hits",
		ConvergencePoint: 0.65, // Typical DDR4 row buffer hit rate
		BaseVariance:     0.05,
		MinOperations:    500,
		CurrentValue:     0.30, // Start conservative
		IsConverged:      false,
	}

	// Bandwidth utilization convergence
	mem.ConvergenceState.Models["bandwidth_utilization"] = &StatisticalModel{
		Name:             "bandwidth_utilization",
		ConvergencePoint: 0.75, // Typical sustained bandwidth utilization
		BaseVariance:     0.08,
		MinOperations:    1000,
		CurrentValue:     0.50,
		IsConverged:      false,
	}

	// NUMA locality convergence
	mem.ConvergenceState.Models["numa_locality"] = &StatisticalModel{
		Name:             "numa_locality",
		ConvergencePoint: 0.80, // Typical NUMA locality
		BaseVariance:     0.04,
		MinOperations:    800,
		CurrentValue:     1.00, // Start with 100% local
		IsConverged:      false,
	}
}

// Note: Memory allocation tracking removed - we only simulate RAM access patterns, not actual allocation

// LoadProfile loads memory-specific profile data
func (mem *MemoryEngine) LoadProfile(profile *EngineProfile) error {
	// Call common profile loading first
	if err := mem.CommonEngine.LoadProfile(profile); err != nil {
		return err
	}

	// Load memory-specific profile data
	return mem.loadMemorySpecificProfile()
}

// loadMemorySpecificProfile loads memory-specific profile data
func (mem *MemoryEngine) loadMemorySpecificProfile() error {
	if mem.Profile == nil {
		return fmt.Errorf("profile is nil")
	}

	// Load baseline performance
	if mem.Profile.BaselinePerformance != nil {
		if capacity, ok := mem.Profile.BaselinePerformance["capacity_gb"]; ok {
			mem.CapacityGB = int64(capacity)
		}
		if freq, ok := mem.Profile.BaselinePerformance["frequency_mhz"]; ok {
			mem.FrequencyMHz = int(freq)
		}
		if cas, ok := mem.Profile.BaselinePerformance["cas_latency"]; ok {
			mem.CASLatency = int(cas)
		}
		if channels, ok := mem.Profile.BaselinePerformance["channels"]; ok {
			mem.Channels = int(channels)
		}
		if bandwidth, ok := mem.Profile.BaselinePerformance["bandwidth_gbps"]; ok {
			mem.BandwidthGBps = bandwidth
		}
		if accessTime, ok := mem.Profile.BaselinePerformance["access_time"]; ok {
			mem.AccessTimeNs = accessTime
		}
	}

	// Load technology specs
	if mem.Profile.TechnologySpecs != nil {
		if memType, ok := mem.Profile.TechnologySpecs["memory_type"].(string); ok {
			mem.MemoryType = memType
		}
	}

	// Load engine-specific configurations
	if mem.Profile.EngineSpecific != nil {
		mem.loadEngineSpecificConfigs()
	}

	return nil
}

// loadEngineSpecificConfigs loads engine-specific memory configurations
func (mem *MemoryEngine) loadEngineSpecificConfigs() {
	// Load DDR timings
	if timings, ok := mem.Profile.EngineSpecific["ddr_timings"]; ok {
		if timingMap, ok := timings.(map[string]interface{}); ok {
			if trcd, ok := timingMap["trcd"].(float64); ok {
				mem.TimingState.tRCD = int(trcd)
			}
			if trp, ok := timingMap["trp"].(float64); ok {
				mem.TimingState.tRP = int(trp)
			}
			if tras, ok := timingMap["tras"].(float64); ok {
				mem.TimingState.tRAS = int(tras)
			}
		}
	}

	// Load NUMA configuration
	if numa, ok := mem.Profile.EngineSpecific["numa_configuration"]; ok {
		if numaMap, ok := numa.(map[string]interface{}); ok {
			if socketCount, ok := numaMap["socket_count"].(float64); ok {
				mem.NUMAState.SocketCount = int(socketCount)
			}
			if penalty, ok := numaMap["cross_socket_penalty"].(float64); ok {
				mem.NUMAState.CrossSocketPenalty = penalty
			}
			if latency, ok := numaMap["inter_socket_latency_ns"].(float64); ok {
				mem.NUMAState.InterSocketLatencyNs = latency
			}
			if ratio, ok := numaMap["local_access_ratio"].(float64); ok {
				mem.NUMAState.LocalAccessRatio = ratio
			}
		}
	}

	// Load bandwidth characteristics (store in main fields, not BandwidthState)
	if bandwidth, ok := mem.Profile.EngineSpecific["bandwidth_characteristics"]; ok {
		if bwMap, ok := bandwidth.(map[string]interface{}); ok {
			if peak, ok := bwMap["peak_bandwidth_gbps"].(float64); ok {
				// Store peak bandwidth in main BandwidthGBps field
				if mem.BandwidthGBps == 0.0 {
					mem.BandwidthGBps = peak
				}
			}
			if sustained, ok := bwMap["sustained_bandwidth_gbps"].(float64); ok {
				// Use sustained bandwidth as the main bandwidth value
				mem.BandwidthGBps = sustained
			}
			// Note: saturation_threshold is used in calculations, not stored in state
		}
	}

	// PRIORITY 1 CRITICAL FEATURES - Load hardware prefetch configuration
	if prefetch, ok := mem.Profile.EngineSpecific["hardware_prefetch"]; ok {
		if prefetchMap, ok := prefetch.(map[string]interface{}); ok {
			if count, ok := prefetchMap["prefetcher_count"].(float64); ok {
				mem.HardwarePrefetchState.PrefetcherCount = int(count)
			}
			if seqAcc, ok := prefetchMap["sequential_accuracy"].(float64); ok {
				mem.HardwarePrefetchState.SequentialAccuracy = seqAcc
			}
			if strideAcc, ok := prefetchMap["stride_accuracy"].(float64); ok {
				mem.HardwarePrefetchState.StrideAccuracy = strideAcc
			}
			if patternAcc, ok := prefetchMap["pattern_accuracy"].(float64); ok {
				mem.HardwarePrefetchState.PatternAccuracy = patternAcc
			}
			if distance, ok := prefetchMap["prefetch_distance"].(float64); ok {
				mem.HardwarePrefetchState.PrefetchDistance = int(distance)
			}
			if bandwidth, ok := prefetchMap["bandwidth_usage"].(float64); ok {
				mem.HardwarePrefetchState.BandwidthUsage = bandwidth
			}
		}
	}

	// PRIORITY 1 CRITICAL FEATURES - Load cache line conflict configuration
	if conflicts, ok := mem.Profile.EngineSpecific["cache_line_conflicts"]; ok {
		if conflictMap, ok := conflicts.(map[string]interface{}); ok {
			if lineSize, ok := conflictMap["cache_line_size"].(float64); ok {
				mem.CacheLineConflictState.CacheLineSize = int(lineSize)
			}
			if detection, ok := conflictMap["false_sharing_detection"].(bool); ok {
				mem.CacheLineConflictState.FalseSharingDetection = detection
			}
			if threshold, ok := conflictMap["conflict_threshold"].(float64); ok {
				mem.CacheLineConflictState.ConflictThreshold = threshold
			}
			if penalty, ok := conflictMap["conflict_penalty"].(float64); ok {
				mem.CacheLineConflictState.ConflictPenalty = penalty
			}
		}
	}

	// PRIORITY 1 CRITICAL FEATURES - Load memory ordering configuration (ONLY if enabled)
	// Skip memory ordering if disabled for simplified engine behavior
	if mem.ComplexityInterface.ShouldEnableFeature("memory_ordering") {
		if ordering, ok := mem.Profile.EngineSpecific["memory_ordering"]; ok {
			if orderingMap, ok := ordering.(map[string]interface{}); ok {
				if model, ok := orderingMap["ordering_model"].(string); ok {
					mem.MemoryOrderingState.OrderingModel = model
				}
				if window, ok := orderingMap["reordering_window"].(float64); ok {
					mem.MemoryOrderingState.ReorderingWindow = int(window)
				}
				if barrierCost, ok := orderingMap["memory_barrier_cost"].(float64); ok {
					mem.MemoryOrderingState.MemoryBarrierCost = barrierCost
				}
				if loadStore, ok := orderingMap["load_store_reordering"].(bool); ok {
					mem.MemoryOrderingState.LoadStoreReordering = loadStore
				}
				if storeStore, ok := orderingMap["store_store_reordering"].(bool); ok {
					mem.MemoryOrderingState.StoreStoreReordering = storeStore
				}
				if loadLoad, ok := orderingMap["load_load_reordering"].(bool); ok {
					mem.MemoryOrderingState.LoadLoadReordering = loadLoad
				}
			}
		}
	}
	// If memory ordering is disabled, keep the initialized disabled values

	// PRIORITY 2 IMPORTANT FEATURES - Load memory controller configuration
	if controller, ok := mem.Profile.EngineSpecific["memory_controller"]; ok {
		if controllerMap, ok := controller.(map[string]interface{}); ok {
			if count, ok := controllerMap["controller_count"].(float64); ok {
				mem.MemoryControllerState.ControllerCount = int(count)
			}
			if depth, ok := controllerMap["queue_depth"].(float64); ok {
				mem.MemoryControllerState.QueueDepth = int(depth)
			}
			if policy, ok := controllerMap["arbitration_policy"].(string); ok {
				mem.MemoryControllerState.ArbitrationPolicy = policy
			}
			if bandwidth, ok := controllerMap["bandwidth_per_controller"].(float64); ok {
				mem.MemoryControllerState.BandwidthPerController = bandwidth
			}
			if latency, ok := controllerMap["controller_latency"].(float64); ok {
				mem.MemoryControllerState.ControllerLatency = latency
			}
		}
	}

	// PRIORITY 2 IMPORTANT FEATURES - Load advanced NUMA configuration
	if advancedNuma, ok := mem.Profile.EngineSpecific["advanced_numa"]; ok {
		if numaMap, ok := advancedNuma.(map[string]interface{}); ok {
			if policy, ok := numaMap["node_affinity_policy"].(string); ok {
				mem.AdvancedNUMAState.NodeAffinityPolicy = policy
			}
			if threshold, ok := numaMap["migration_threshold"].(float64); ok {
				mem.AdvancedNUMAState.MigrationThreshold = threshold
			}
			// Load topology map, distance matrix, and bandwidth matrix would be more complex
			// For now, we'll initialize them based on the basic NUMA configuration
			mem.initializeAdvancedNUMAFromBasic()
		}
	}

	// PRIORITY 2 IMPORTANT FEATURES - Load virtual memory configuration
	if vm, ok := mem.Profile.EngineSpecific["virtual_memory"]; ok {
		if vmMap, ok := vm.(map[string]interface{}); ok {
			if pageSize, ok := vmMap["page_size"].(float64); ok {
				mem.VirtualMemoryState.PageSize = int(pageSize)
			}
			if tlbSize, ok := vmMap["tlb_size"].(float64); ok {
				mem.VirtualMemoryState.TLBSize = int(tlbSize)
			}
			if hitRatio, ok := vmMap["tlb_hit_ratio"].(float64); ok {
				mem.VirtualMemoryState.TLBHitRatio = hitRatio
			}
			if levels, ok := vmMap["page_table_levels"].(float64); ok {
				mem.VirtualMemoryState.PageTableLevels = int(levels)
			}
			if walkLatency, ok := vmMap["page_walk_latency"].(float64); ok {
				mem.VirtualMemoryState.PageWalkLatency = walkLatency
			}
			if swapEnabled, ok := vmMap["swap_enabled"].(bool); ok {
				mem.VirtualMemoryState.SwapEnabled = swapEnabled
			}
			if swapLatency, ok := vmMap["swap_latency"].(float64); ok {
				mem.VirtualMemoryState.SwapLatency = swapLatency
			}
		}
	}

	// PRIORITY 3 ENHANCEMENT FEATURES - Load ECC modeling configuration
	if ecc, ok := mem.Profile.EngineSpecific["ecc_modeling"]; ok {
		if eccMap, ok := ecc.(map[string]interface{}); ok {
			if enabled, ok := eccMap["ecc_enabled"].(bool); ok {
				mem.ECCModelingState.ECCEnabled = enabled
			}
			if singleRate, ok := eccMap["single_bit_error_rate"].(float64); ok {
				mem.ECCModelingState.SingleBitErrorRate = singleRate
			}
			if multiRate, ok := eccMap["multi_bit_error_rate"].(float64); ok {
				mem.ECCModelingState.MultiBitErrorRate = multiRate
			}
			if correctionLatency, ok := eccMap["correction_latency"].(float64); ok {
				mem.ECCModelingState.CorrectionLatency = correctionLatency
			}
			if detectionLatency, ok := eccMap["detection_latency"].(float64); ok {
				mem.ECCModelingState.DetectionLatency = detectionLatency
			}
		}
	}

	// PRIORITY 3 ENHANCEMENT FEATURES - Load power state configuration
	if power, ok := mem.Profile.EngineSpecific["power_states"]; ok {
		if powerMap, ok := power.(map[string]interface{}); ok {
			if transitionCost, ok := powerMap["state_transition_cost"].(float64); ok {
				mem.PowerStateTransitions.StateTransitionCost = transitionCost
			}
			if activePower, ok := powerMap["active_power_draw"].(float64); ok {
				mem.PowerStateTransitions.ActivePowerDraw = activePower
			}
			if standbyPower, ok := powerMap["standby_power_draw"].(float64); ok {
				mem.PowerStateTransitions.StandbyPowerDraw = standbyPower
			}
			if sleepPower, ok := powerMap["sleep_power_draw"].(float64); ok {
				mem.PowerStateTransitions.SleepPowerDraw = sleepPower
			}
			if wakeupLatency, ok := powerMap["wakeup_latency"].(float64); ok {
				mem.PowerStateTransitions.WakeupLatency = wakeupLatency
			}
			if idleThreshold, ok := powerMap["idle_threshold"].(float64); ok {
				mem.PowerStateTransitions.IdleThreshold = idleThreshold
			}
		}
	}

	// PRIORITY 3 ENHANCEMENT FEATURES - Load enhanced thermal configuration
	if thermal, ok := mem.Profile.EngineSpecific["enhanced_thermal"]; ok {
		if thermalMap, ok := thermal.(map[string]interface{}); ok {
			if dissipationRate, ok := thermalMap["heat_dissipation_rate"].(float64); ok {
				mem.EnhancedThermalState.HeatDissipationRate = dissipationRate
			}
			if capacity, ok := thermalMap["thermal_capacity"].(float64); ok {
				mem.EnhancedThermalState.ThermalCapacity = capacity
			}
			if ambient, ok := thermalMap["ambient_temperature"].(float64); ok {
				mem.EnhancedThermalState.AmbientTemperature = ambient
			}
			// Load thermal zones, thresholds, and levels would be more complex
			mem.initializeEnhancedThermalFromProfile(thermalMap)
		}
	}
}

// initializePriorityFeatures initializes Priority 1 & 2 Features state structures
func (mem *MemoryEngine) initializePriorityFeatures() {
	// Initialize Hardware Prefetch State (values will be loaded from profile)
	mem.HardwarePrefetchState.PrefetcherCount = 0        // From profile
	mem.HardwarePrefetchState.SequentialAccuracy = 0.0  // From profile
	mem.HardwarePrefetchState.StrideAccuracy = 0.0      // From profile
	mem.HardwarePrefetchState.PatternAccuracy = 0.0     // From profile
	mem.HardwarePrefetchState.PrefetchDistance = 0      // From profile
	mem.HardwarePrefetchState.BandwidthUsage = 0.0      // From profile
	mem.HardwarePrefetchState.AccessPatternHistory = make([]int64, 16) // 16-entry history
	mem.HardwarePrefetchState.LastPrefetchUpdate = 0

	// Initialize Cache Line Conflict State (values will be loaded from profile)
	mem.CacheLineConflictState.CacheLineSize = 0         // From profile
	mem.CacheLineConflictState.FalseSharingDetection = false // From profile
	mem.CacheLineConflictState.ConflictThreshold = 0.0   // From profile
	mem.CacheLineConflictState.ConflictPenalty = 0.0     // From profile
	mem.CacheLineConflictState.ActiveConflicts = 0
	mem.CacheLineConflictState.ConflictHistory = make(map[uint64]int64)
	mem.CacheLineConflictState.LastConflictUpdate = 0

	// Initialize Memory Ordering State (DISABLED for simplified engine like CPU)
	mem.MemoryOrderingState.OrderingModel = "simple"     // Simplified model
	mem.MemoryOrderingState.ReorderingWindow = 0         // DISABLED - no reordering window
	mem.MemoryOrderingState.MemoryBarrierCost = 0.0      // DISABLED
	mem.MemoryOrderingState.LoadStoreReordering = false  // DISABLED
	mem.MemoryOrderingState.StoreStoreReordering = false // DISABLED
	mem.MemoryOrderingState.LoadLoadReordering = false   // DISABLED
	mem.MemoryOrderingState.PendingOperations = make([]MemoryOrderingOp, 0)
	mem.MemoryOrderingState.LastOrderingUpdate = 0

	// PRIORITY 2 IMPORTANT FEATURES - Initialize Memory Controller State
	mem.MemoryControllerState.ControllerCount = 0        // From profile
	mem.MemoryControllerState.QueueDepth = 0             // From profile
	mem.MemoryControllerState.ArbitrationPolicy = ""     // From profile
	mem.MemoryControllerState.BandwidthPerController = 0.0 // From profile
	mem.MemoryControllerState.ControllerLatency = 0.0    // From profile
	mem.MemoryControllerState.ActiveRequests = make([]MemoryControllerRequest, 0)
	mem.MemoryControllerState.LastControllerUpdate = 0

	// PRIORITY 2 IMPORTANT FEATURES - Initialize Advanced NUMA State
	mem.AdvancedNUMAState.TopologyMap = make(map[int][]int)
	mem.AdvancedNUMAState.DistanceMatrix = make([][]float64, 0)
	mem.AdvancedNUMAState.BandwidthMatrix = make([][]float64, 0)
	mem.AdvancedNUMAState.NodeMemoryPressure = make([]float64, 0)
	mem.AdvancedNUMAState.NodeAffinityPolicy = ""                              // From profile
	mem.AdvancedNUMAState.MigrationThreshold = 0.0                             // From profile
	mem.AdvancedNUMAState.PageAccessPatterns = make(map[uint64]*PageAccessPattern)
	mem.AdvancedNUMAState.PageNodeMapping = make(map[uint64]int)
	mem.AdvancedNUMAState.MigrationCount = 0
	mem.AdvancedNUMAState.LastMigrationTick = 0
	mem.AdvancedNUMAState.LastNUMAUpdate = 0

	// PRIORITY 2 IMPORTANT FEATURES - Initialize Virtual Memory State
	mem.VirtualMemoryState.PageSize = 0                  // From profile
	mem.VirtualMemoryState.TLBSize = 0                   // From profile
	mem.VirtualMemoryState.TLBHitRatio = 0.0             // From profile
	mem.VirtualMemoryState.PageTableLevels = 0           // From profile
	mem.VirtualMemoryState.PageWalkLatency = 0.0         // From profile
	mem.VirtualMemoryState.SwapEnabled = false           // From profile
	mem.VirtualMemoryState.SwapLatency = 0.0             // From profile
	mem.VirtualMemoryState.TLBMissCount = 0
	mem.VirtualMemoryState.PageFaultCount = 0
	mem.VirtualMemoryState.LastVMUpdate = 0

	// PRIORITY 3 ENHANCEMENT FEATURES - Initialize ECC Modeling State
	mem.ECCModelingState.ECCEnabled = false              // From profile
	mem.ECCModelingState.SingleBitErrorRate = 0.0       // From profile
	mem.ECCModelingState.MultiBitErrorRate = 0.0        // From profile
	mem.ECCModelingState.CorrectionLatency = 0.0        // From profile
	mem.ECCModelingState.DetectionLatency = 0.0         // From profile
	mem.ECCModelingState.SingleBitErrorCount = 0
	mem.ECCModelingState.MultiBitErrorCount = 0
	mem.ECCModelingState.LastECCUpdate = 0

	// PRIORITY 3 ENHANCEMENT FEATURES - Initialize Power State Transitions
	mem.PowerStateTransitions.CurrentPowerState = "active" // Default to active
	mem.PowerStateTransitions.StateTransitionCost = 0.0  // From profile
	mem.PowerStateTransitions.ActivePowerDraw = 0.0      // From profile
	mem.PowerStateTransitions.StandbyPowerDraw = 0.0     // From profile
	mem.PowerStateTransitions.SleepPowerDraw = 0.0       // From profile
	mem.PowerStateTransitions.WakeupLatency = 0.0        // From profile
	mem.PowerStateTransitions.IdleThreshold = 0.0        // From profile
	mem.PowerStateTransitions.LastPowerUpdate = 0
	mem.PowerStateTransitions.IdleStartTick = 0

	// PRIORITY 3 ENHANCEMENT FEATURES - Initialize Enhanced Thermal State
	mem.EnhancedThermalState.ThermalZones = make([]ThermalZone, 0)
	mem.EnhancedThermalState.ThrottlingThresholds = make([]float64, 0)
	mem.EnhancedThermalState.ThrottlingLevels = make([]float64, 0)
	mem.EnhancedThermalState.HeatDissipationRate = 0.0   // From profile
	mem.EnhancedThermalState.ThermalCapacity = 0.0       // From profile
	mem.EnhancedThermalState.AmbientTemperature = 0.0    // From profile
	mem.EnhancedThermalState.LastThermalUpdate = 0
}

// initializeAdvancedFeatures initializes advanced memory features for Maximum complexity
func (mem *MemoryEngine) initializeAdvancedFeatures() {
	// Initialize ECC state
	mem.ECCState.ErrorRate = 0.0
	mem.ECCState.CorrectedErrors = 0
	mem.ECCState.UncorrectedErrors = 0
	mem.ECCState.LastECCCheck = 0
	mem.ECCState.ECCOverhead = 0.02 // 2% overhead for ECC

	// Initialize power state
	mem.PowerState.CurrentState = "Active"
	mem.PowerState.PowerConsumption = 0.0
	mem.PowerState.StateTransitions = 0
	mem.PowerState.LastStateChange = 0
	mem.PowerState.PowerEfficiency = 1.0

	// Initialize thermal state
	mem.ThermalState.Temperature = 45.0 // Start at 45°C
	mem.ThermalState.ThermalThrottling = false
	mem.ThermalState.ThrottleLevel = 0.0
	mem.ThermalState.LastThermalUpdate = 0
	mem.ThermalState.CoolingEfficiency = 1.0
}

// initializeStateIdentity initializes unique state identity
func (mem *MemoryEngine) initializeStateIdentity() {
	mem.StateIdentity.EngineID = mem.GetEngineID()
	mem.StateIdentity.ProfileName = ""
	mem.StateIdentity.ComplexityLevel = 0
	mem.StateIdentity.CreatedAt = time.Now().UnixNano()
	mem.StateIdentity.LastSaved = 0
}

// SaveEngineState saves complete memory engine state to JSON
func (mem *MemoryEngine) SaveEngineState() ([]byte, error) {
	mem.StateIdentity.LastSaved = time.Now().UnixNano()

	// Create complete state snapshot
	state := struct {
		*MemoryEngine
		SavedAt int64 `json:"saved_at"`
	}{
		MemoryEngine: mem,
		SavedAt:      mem.StateIdentity.LastSaved,
	}

	return json.MarshalIndent(state, "", "  ")
}

// LoadEngineState loads complete memory engine state from JSON
func (mem *MemoryEngine) LoadEngineState(data []byte) error {
	// Create temporary state structure
	var state struct {
		*MemoryEngine
		SavedAt int64 `json:"saved_at"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal memory engine state: %w", err)
	}

	// Restore all state fields
	if state.MemoryEngine != nil {
		// Copy all state fields
		mem.BusyChannels = state.BusyChannels
		mem.TimingState = state.TimingState
		mem.BandwidthState = state.BandwidthState
		mem.NUMAState = state.NUMAState
		mem.PressureState = state.PressureState
		mem.ConvergenceState = state.ConvergenceState
		mem.ECCState = state.ECCState
		mem.PowerState = state.PowerState
		mem.ThermalState = state.ThermalState
		mem.StateIdentity = state.StateIdentity

		// Update common engine fields
		mem.CompletedOps = state.CompletedOps
		mem.CurrentTick = state.CurrentTick
		mem.TotalOperations = state.TotalOperations
	}

	return nil
}

// GetQueueCapacity returns the maximum queue capacity
func (mem *MemoryEngine) GetQueueCapacity() int {
	return mem.CommonEngine.GetQueueCapacity()
}

// getProfileInt gets an integer value from profile with fallback (like CPU engine)
func (mem *MemoryEngine) getProfileInt(category, key string, defaultValue int) int {
	if mem.Profile == nil {
		return defaultValue
	}

	// Look in EngineSpecific first (like CPU engine)
	if categoryMap, ok := mem.Profile.EngineSpecific[category].(map[string]interface{}); ok {
		if value, ok := categoryMap[key].(float64); ok {
			return int(value)
		}
	}

	// Fallback to TechnologySpecs for backward compatibility
	if categoryMap, ok := mem.Profile.TechnologySpecs[category].(map[string]interface{}); ok {
		if value, ok := categoryMap[key].(float64); ok {
			return int(value)
		}
	}

	return defaultValue
}

// getMaxInternalQueueSize returns the maximum size for the internal processing heap (like CPU engine)
func (mem *MemoryEngine) getMaxInternalQueueSize() int {
	// INTRA-ENGINE FLOW: Limit heap size based on realistic memory constraints
	// Heap should hold operations for: channels × average_operation_duration_ticks

	// Get average operation duration from profile
	avgOpDurationMs := mem.getProfileFloat("queue_processing", "avg_operation_duration_ms", 1.0) // Memory ops are faster than CPU
	tickDurationMs := float64(mem.TickDuration) / float64(time.Millisecond)
	avgOpDurationTicks := int(avgOpDurationMs / tickDurationMs)

	// Calculate max heap size: channels × operation_duration × safety_factor
	safetyFactor := 2.0 // Allow 2x buffer for operation variance
	maxHeapSize := int(float64(mem.Channels * avgOpDurationTicks) * safetyFactor)

	// Ensure reasonable bounds
	if maxHeapSize < 50 {
		maxHeapSize = 50 // Minimum heap size for memory operations
	}
	if maxHeapSize > 5000 {
		maxHeapSize = 5000 // Maximum heap size to prevent memory issues
	}

	return maxHeapSize
}

// getProfileFloat gets a float value from profile with fallback (like CPU engine)
func (mem *MemoryEngine) getProfileFloat(category, key string, defaultValue float64) float64 {
	if mem.Profile == nil {
		return defaultValue
	}

	// Look in EngineSpecific first (like CPU engine)
	if categoryMap, ok := mem.Profile.EngineSpecific[category].(map[string]interface{}); ok {
		if value, ok := categoryMap[key].(float64); ok {
			return value
		}
	}

	// Fallback to TechnologySpecs for backward compatibility
	if categoryMap, ok := mem.Profile.TechnologySpecs[category].(map[string]interface{}); ok {
		if value, ok := categoryMap[key].(float64); ok {
			return value
		}
	}

	return defaultValue
}

// SetComplexityLevel sets the complexity level using the memory interface
func (mem *MemoryEngine) SetComplexityLevel(level int) error {
	return mem.ComplexityInterface.SetComplexityLevel(MemoryComplexityLevel(level))
}

// GetComplexityLevel returns the complexity level from the memory interface
func (mem *MemoryEngine) GetComplexityLevel() int {
	return int(mem.ComplexityInterface.ComplexityLevel)
}

// applyGarbageCollectionEffects applies language-specific garbage collection effects
func (mem *MemoryEngine) applyGarbageCollectionEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Language-specific GC modeling based on operation language
	switch op.Language {
	case "java":
		// Java G1GC: ~8ms per GB of heap pressure
		heapPressure := mem.calculateHeapPressure()
		if heapPressure > 0.7 {
			gcPause := time.Duration(heapPressure * 8.0 * float64(time.Millisecond))
			return baseTime + gcPause
		}
	case "go":
		// Go GC: ~0.5ms per GB, more frequent but shorter
		heapPressure := mem.calculateHeapPressure()
		if heapPressure > 0.8 {
			gcPause := time.Duration(heapPressure * 0.5 * float64(time.Millisecond))
			return baseTime + gcPause
		}
	case "csharp":
		// .NET GC: ~3ms per GB
		heapPressure := mem.calculateHeapPressure()
		if heapPressure > 0.75 {
			gcPause := time.Duration(heapPressure * 3.0 * float64(time.Millisecond))
			return baseTime + gcPause
		}
	}
	return baseTime
}

// applyMemoryFragmentationEffects applies heap fragmentation effects
func (mem *MemoryEngine) applyMemoryFragmentationEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Fragmentation increases with memory usage and allocation patterns
	utilization := mem.calculateCurrentUtilization()

	if utilization > 0.6 {
		// Fragmentation penalty increases exponentially with utilization
		fragmentationFactor := 1.0 + (utilization-0.6)*2.0 // Up to 80% penalty at full utilization
		return time.Duration(float64(baseTime) * fragmentationFactor)
	}

	return baseTime
}

// calculateHeapPressure calculates current heap pressure for GC modeling
func (mem *MemoryEngine) calculateHeapPressure() float64 {
	utilization := mem.calculateCurrentUtilization()

	// Heap pressure is higher than general utilization due to allocation patterns
	heapPressure := utilization * 1.2 // 20% higher than general utilization
	if heapPressure > 1.0 {
		heapPressure = 1.0
	}

	return heapPressure
}

// ========================================
// PRIORITY 1 CRITICAL FEATURES IMPLEMENTATION
// ========================================

// applyHardwarePrefetchingEffects applies hardware prefetcher effects using profile-driven configuration
func (mem *MemoryEngine) applyHardwarePrefetchingEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Skip if no prefetchers configured
	if mem.HardwarePrefetchState.PrefetcherCount == 0 {
		return baseTime
	}

	// Determine access pattern type using statistical analysis
	accessPattern := mem.analyzeMemoryAccessPattern(op)

	// Calculate prefetch effectiveness based on pattern (from profile)
	var prefetchAccuracy float64
	switch accessPattern {
	case "sequential":
		prefetchAccuracy = mem.HardwarePrefetchState.SequentialAccuracy
	case "stride":
		prefetchAccuracy = mem.HardwarePrefetchState.StrideAccuracy
	case "pattern":
		prefetchAccuracy = mem.HardwarePrefetchState.PatternAccuracy
	default:
		prefetchAccuracy = 0.0 // Random access - no prefetching benefit
	}

	// Use deterministic hash for consistent prefetch behavior
	opHash := mem.hashOperationForMemoryDecision(op)
	prefetchHash := opHash + 54321 // Different seed for prefetch decisions
	hashValue := float64(prefetchHash%10000) / 10000.0

	// Prefetch hit reduces memory access time
	if hashValue < prefetchAccuracy {
		// Successful prefetch: reduce memory access penalty (benefit from profile)
		prefetchBenefit := mem.getProfileFloat("hardware_prefetch", "prefetch_benefit", 0.3) // Default 30% reduction
		return time.Duration(float64(baseTime) * (1.0 - prefetchBenefit))
	}

	// No prefetch benefit, but add bandwidth overhead if prefetching is active
	bandwidthOverhead := 1.0 + mem.HardwarePrefetchState.BandwidthUsage
	return time.Duration(float64(baseTime) * bandwidthOverhead)
}

// applyCacheLineConflictEffects applies cache line conflict and false sharing detection
func (mem *MemoryEngine) applyCacheLineConflictEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Skip if false sharing detection is disabled
	if !mem.CacheLineConflictState.FalseSharingDetection {
		return baseTime
	}

	// Calculate cache line address for this operation
	cacheLineAddr := mem.calculateCacheLineAddress(op)

	// Check for conflicts with recent operations
	currentTick := mem.CurrentTick
	conflictWindow := mem.getProfileInt("cache_line_conflicts", "conflict_window_ticks", 10)

	// Check conflict history for this cache line
	if lastAccess, exists := mem.CacheLineConflictState.ConflictHistory[cacheLineAddr]; exists {
		ticksSinceLastAccess := currentTick - lastAccess

		// If accessed recently, it's a potential conflict
		if ticksSinceLastAccess < int64(conflictWindow) {
			// Determine if this is a true conflict (different threads/cores accessing same line)
			isConflict := mem.detectFalseSharing(op, cacheLineAddr)

			if isConflict {
				mem.CacheLineConflictState.ActiveConflicts++

				// Apply conflict penalty (from profile)
				conflictPenalty := 1.0 + mem.CacheLineConflictState.ConflictPenalty
				return time.Duration(float64(baseTime) * conflictPenalty)
			}
		}
	}

	// Update conflict history
	mem.CacheLineConflictState.ConflictHistory[cacheLineAddr] = currentTick

	// Clean old entries to prevent memory growth
	mem.cleanConflictHistory(currentTick, int64(conflictWindow*2))

	return baseTime
}

// applyMemoryOrderingEffects applies memory ordering and reordering effects
func (mem *MemoryEngine) applyMemoryOrderingEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Skip if no reordering window configured
	if mem.MemoryOrderingState.ReorderingWindow == 0 {
		return baseTime
	}

	// Determine operation type
	opType := mem.determineMemoryOpType(op)

	// Handle memory barriers immediately (no reordering)
	if opType == "barrier" {
		// Flush pending operations and apply barrier cost
		mem.flushPendingMemoryOperations()
		barrierCost := time.Duration(mem.MemoryOrderingState.MemoryBarrierCost * float64(time.Nanosecond))
		return baseTime + barrierCost
	}

	// Check for memory dependencies with pending operations
	dependencyDelay := mem.checkMemoryDependencies(op, opType)
	if dependencyDelay > 0 {
		// Cannot reorder due to dependencies - apply dependency delay
		return baseTime + dependencyDelay
	}

	// Check if this operation can be reordered based on ordering model
	canReorder := mem.canReorderOperation(op, opType)

	// Only use reordering window if there's significant benefit and low queue pressure
	// This prevents the reordering logic from interfering with normal operation flow
	shouldUseReordering := canReorder &&
		len(mem.MemoryOrderingState.PendingOperations) < (mem.MemoryOrderingState.ReorderingWindow/2) && // Use only half the window
		mem.GetQueueLength() < (mem.GetQueueCapacity()/2) // Only when queue is not under pressure

	if shouldUseReordering {
		// Calculate optimal reordering position (but keep it small)
		reorderingDelay := mem.calculateOptimalReorderingDelay(op, opType)
		if reorderingDelay > 2 { // Limit reordering delay to 2 ticks max
			reorderingDelay = 2
		}

		// Add to reordering window
		reorderedOp := MemoryOrderingOp{
			Operation:     op,
			OriginalTick:  mem.CurrentTick,
			ReorderedTick: mem.CurrentTick + reorderingDelay,
			OpType:        opType,
			Address:       mem.calculateMemoryAddress(op),
			CanReorder:    true,
		}

		mem.MemoryOrderingState.PendingOperations = append(mem.MemoryOrderingState.PendingOperations, reorderedOp)

		// Sort pending operations by reordered tick for optimal execution order
		mem.sortPendingOperationsByTick()

		// Return optimized time due to reordering (but smaller benefit)
		reorderingBenefit := mem.calculateReorderingBenefit(op, opType) * 0.5 // Reduce benefit to be less aggressive
		return time.Duration(float64(baseTime) * (1.0 - reorderingBenefit))
	}

	// Process any ready operations from reordering window
	mem.processReadyReorderedOperations()

	return baseTime
}

// ========================================
// PRIORITY 2 IMPORTANT FEATURES IMPLEMENTATION
// ========================================

// applyMemoryControllerEffects applies memory controller modeling and arbitration
func (mem *MemoryEngine) applyMemoryControllerEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Skip if no controllers configured
	if mem.MemoryControllerState.ControllerCount == 0 {
		return baseTime
	}

	// Determine which controller handles this request
	controllerID := mem.selectMemoryController(op)

	// Check controller queue depth and apply queuing delay
	queueDelay := mem.calculateControllerQueueDelay(controllerID)

	// Apply controller processing latency (from profile)
	controllerLatency := time.Duration(mem.MemoryControllerState.ControllerLatency * float64(time.Nanosecond))

	// Apply bandwidth limitation per controller
	bandwidthDelay := mem.calculateControllerBandwidthDelay(op, controllerID)

	// Total controller overhead
	totalControllerTime := baseTime + controllerLatency + queueDelay + bandwidthDelay

	// Update controller state
	mem.updateControllerState(op, controllerID)

	return totalControllerTime
}

// applyAdvancedNUMAEffects applies advanced NUMA topology and migration effects
func (mem *MemoryEngine) applyAdvancedNUMAEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Skip if no advanced NUMA topology configured
	if len(mem.AdvancedNUMAState.DistanceMatrix) == 0 {
		return baseTime
	}

	// Determine source and target NUMA nodes
	sourceNode := mem.determineSourceNUMANode(op)
	targetNode := mem.determineTargetNUMANode(op)

	// Apply distance-based latency penalty
	distancePenalty := mem.calculateNUMADistancePenalty(sourceNode, targetNode)

	// Apply bandwidth limitations between nodes
	bandwidthPenalty := mem.calculateNUMABandwidthPenalty(sourceNode, targetNode, op)

	// Check for page migration opportunities
	migrationBenefit := mem.evaluatePageMigration(op, sourceNode, targetNode)

	// Apply NUMA affinity policy effects
	affinityPenalty := mem.applyNUMAAffinityPolicy(op, sourceNode, targetNode)

	// Total NUMA overhead
	totalNUMATime := baseTime + distancePenalty + bandwidthPenalty + affinityPenalty - migrationBenefit

	// Update NUMA state
	mem.updateAdvancedNUMAState(sourceNode, targetNode)

	return totalNUMATime
}

// applyVirtualMemoryEffects applies virtual memory, TLB, and page table effects
func (mem *MemoryEngine) applyVirtualMemoryEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Skip if no virtual memory configured
	if mem.VirtualMemoryState.PageSize == 0 {
		return baseTime
	}

	// Check for TLB hit/miss
	tlbHit := mem.checkTLBHit(op)

	if tlbHit {
		// TLB hit - minimal overhead
		tlbHitLatency := mem.getProfileFloat("virtual_memory", "tlb_hit_latency", 1.0) // 1ns default
		return baseTime + time.Duration(tlbHitLatency*float64(time.Nanosecond))
	}

	// TLB miss - need page table walk
	mem.VirtualMemoryState.TLBMissCount++

	// Calculate page table walk latency based on levels
	pageWalkLatency := time.Duration(mem.VirtualMemoryState.PageWalkLatency *
		float64(mem.VirtualMemoryState.PageTableLevels) * float64(time.Nanosecond))

	// Check for page fault (swap)
	if mem.VirtualMemoryState.SwapEnabled && mem.checkPageFault(op) {
		mem.VirtualMemoryState.PageFaultCount++

		// Apply swap latency (much higher than RAM)
		swapLatency := time.Duration(mem.VirtualMemoryState.SwapLatency * float64(time.Nanosecond))
		return baseTime + pageWalkLatency + swapLatency
	}

	// Update TLB with new entry
	mem.updateTLB(op)

	return baseTime + pageWalkLatency
}

// ========================================
// PRIORITY 3 ENHANCEMENT FEATURES IMPLEMENTATION
// ========================================

// applyECCModelingEffects applies ECC error detection and correction modeling
func (mem *MemoryEngine) applyECCModelingEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Skip if ECC is not enabled
	if !mem.ECCModelingState.ECCEnabled {
		return baseTime
	}

	// Calculate probability of ECC events based on data size and error rates
	dataSize := float64(op.DataSize) / (1024 * 1024 * 1024) // Convert to GB

	// Use deterministic hash for consistent ECC behavior
	opHash := mem.hashOperationForMemoryDecision(op)
	eccHash := opHash + 24680 // Different seed for ECC decisions
	hashValue := float64(eccHash%100000) / 100000.0

	// Calculate error probabilities (per operation, scaled by data size)
	singleBitProb := mem.ECCModelingState.SingleBitErrorRate * dataSize / 3600.0 // Scale from per-hour to per-operation
	multiBitProb := mem.ECCModelingState.MultiBitErrorRate * dataSize / 3600.0

	if hashValue < multiBitProb {
		// Multi-bit error - uncorrectable, causes system impact
		mem.ECCModelingState.MultiBitErrorCount++

		// Multi-bit errors cause significant delays (system recovery)
		multiBitPenalty := mem.getProfileFloat("ecc_modeling", "multi_bit_penalty", 1000.0) // 1μs default
		return baseTime + time.Duration(multiBitPenalty*float64(time.Nanosecond))

	} else if hashValue < singleBitProb + multiBitProb {
		// Single-bit error - correctable
		mem.ECCModelingState.SingleBitErrorCount++

		// Apply ECC correction latency
		correctionLatency := time.Duration(mem.ECCModelingState.CorrectionLatency * float64(time.Nanosecond))
		return baseTime + correctionLatency
	}

	// No ECC event, but add detection overhead
	detectionLatency := time.Duration(mem.ECCModelingState.DetectionLatency * float64(time.Nanosecond))
	return baseTime + detectionLatency
}

// applyPowerStateEffects applies memory power state transition effects
func (mem *MemoryEngine) applyPowerStateEffects(baseTime time.Duration, op *Operation) time.Duration {
	currentTick := mem.CurrentTick

	// Check if memory is in low power state and needs to wake up
	if mem.PowerStateTransitions.CurrentPowerState != "active" {
		// Apply wakeup latency
		wakeupLatency := time.Duration(mem.PowerStateTransitions.WakeupLatency * float64(time.Nanosecond))

		// Transition to active state
		mem.PowerStateTransitions.CurrentPowerState = "active"
		mem.PowerStateTransitions.LastPowerUpdate = currentTick
		mem.PowerStateTransitions.IdleStartTick = 0 // Reset idle tracking

		// Apply state transition cost
		transitionCost := time.Duration(mem.PowerStateTransitions.StateTransitionCost * float64(time.Nanosecond))

		return baseTime + wakeupLatency + transitionCost
	}

	// Memory is active - reset idle tracking
	mem.PowerStateTransitions.IdleStartTick = 0
	mem.PowerStateTransitions.LastPowerUpdate = currentTick

	// Check if we should transition to lower power state (simplified - based on queue emptiness)
	if mem.ActiveOperations.Len() == 0 && mem.PowerStateTransitions.IdleThreshold > 0 {
		// Start idle tracking if not already started
		if mem.PowerStateTransitions.IdleStartTick == 0 {
			mem.PowerStateTransitions.IdleStartTick = currentTick
		} else {
			// Check if idle threshold exceeded
			idleDuration := currentTick - mem.PowerStateTransitions.IdleStartTick
			if float64(idleDuration) > mem.PowerStateTransitions.IdleThreshold {
				// Transition to standby state
				mem.PowerStateTransitions.CurrentPowerState = "standby"
			}
		}
	}

	return baseTime
}

// applyEnhancedThermalThrottlingEffects applies enhanced thermal modeling with multiple zones
func (mem *MemoryEngine) applyEnhancedThermalThrottlingEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Skip if no thermal zones configured
	if len(mem.EnhancedThermalState.ThermalZones) == 0 {
		return baseTime
	}

	// Update thermal state based on operation
	mem.updateThermalZones(op)

	// Find the hottest thermal zone
	maxTemp := 0.0
	hottestZone := -1
	for i, zone := range mem.EnhancedThermalState.ThermalZones {
		if zone.Temperature > maxTemp {
			maxTemp = zone.Temperature
			hottestZone = i
		}
	}

	if hottestZone == -1 {
		return baseTime
	}

	// Apply throttling based on temperature thresholds
	throttlingLevel := mem.calculateThrottlingLevel(maxTemp)

	if throttlingLevel > 0 {
		// Apply throttling penalty
		throttlingPenalty := 1.0 + throttlingLevel
		return time.Duration(float64(baseTime) * throttlingPenalty)
	}

	return baseTime
}

// ========================================
// PRIORITY 1 HELPER METHODS
// ========================================

// analyzeMemoryAccessPattern determines the memory access pattern for prefetching
func (mem *MemoryEngine) analyzeMemoryAccessPattern(op *Operation) string {
	// Update access pattern history
	currentAddr := mem.calculateMemoryAddress(op)
	mem.HardwarePrefetchState.AccessPatternHistory = append(mem.HardwarePrefetchState.AccessPatternHistory[1:], int64(currentAddr))

	// Analyze recent access patterns
	history := mem.HardwarePrefetchState.AccessPatternHistory

	// Check for sequential access (addresses increasing by cache line size)
	cacheLineSize := int64(mem.CacheLineConflictState.CacheLineSize)
	if cacheLineSize == 0 {
		cacheLineSize = 64 // Default cache line size
	}

	sequentialCount := 0
	strideCount := 0
	lastStride := int64(0)

	for i := 1; i < len(history); i++ {
		if history[i] == 0 || history[i-1] == 0 {
			continue // Skip uninitialized entries
		}

		stride := history[i] - history[i-1]

		if stride == cacheLineSize {
			sequentialCount++
		} else if stride > 0 && stride == lastStride {
			strideCount++
		}

		lastStride = stride
	}

	// Determine pattern based on analysis
	if sequentialCount >= 3 {
		return "sequential"
	} else if strideCount >= 2 {
		return "stride"
	} else if sequentialCount > 0 || strideCount > 0 {
		return "pattern"
	}

	return "random"
}

// calculateCacheLineAddress calculates the cache line address for conflict detection
func (mem *MemoryEngine) calculateCacheLineAddress(op *Operation) uint64 {
	baseAddr := mem.calculateMemoryAddress(op)
	cacheLineSize := uint64(mem.CacheLineConflictState.CacheLineSize)
	if cacheLineSize == 0 {
		cacheLineSize = 64 // Default cache line size
	}

	// Align to cache line boundary
	return baseAddr & ^(cacheLineSize - 1)
}

// detectFalseSharing detects potential false sharing scenarios
func (mem *MemoryEngine) detectFalseSharing(op *Operation, cacheLineAddr uint64) bool {
	// Use deterministic hash to simulate different threads/cores accessing same cache line
	opHash := mem.hashOperationForMemoryDecision(op)

	// Check if this represents a different thread accessing the same cache line
	// In real systems, this would be based on actual thread/core information
	conflictHash := float64((opHash+uint32(cacheLineAddr))%10000) / 10000.0

	return conflictHash < mem.CacheLineConflictState.ConflictThreshold
}

// cleanConflictHistory removes old entries from conflict history
func (mem *MemoryEngine) cleanConflictHistory(currentTick, maxAge int64) {
	for addr, lastTick := range mem.CacheLineConflictState.ConflictHistory {
		if currentTick-lastTick > maxAge {
			delete(mem.CacheLineConflictState.ConflictHistory, addr)
		}
	}
}

// determineMemoryOpType determines the type of memory operation for ordering
func (mem *MemoryEngine) determineMemoryOpType(op *Operation) string {
	switch op.Type {
	case "memory_read":
		return "load"
	case "memory_write":
		return "store"
	case "memory_barrier", "memory_fence":
		return "barrier"
	default:
		// Determine based on operation characteristics
		if op.DataSize == 0 {
			return "barrier"
		} else if op.Type == "memory_allocate" || op.Type == "memory_deallocate" {
			return "store" // Allocation operations are treated as stores
		}
		return "load" // Default to load
	}
}

// canReorderOperation determines if an operation can be reordered based on memory ordering model
func (mem *MemoryEngine) canReorderOperation(op *Operation, opType string) bool {
	switch mem.MemoryOrderingState.OrderingModel {
	case "strong":
		return false // Strong ordering - no reordering allowed
	case "weak":
		// Weak ordering - allow most reordering
		switch opType {
		case "load":
			return mem.MemoryOrderingState.LoadLoadReordering
		case "store":
			return mem.MemoryOrderingState.StoreStoreReordering
		case "barrier":
			return false // Barriers cannot be reordered
		}
	case "tso": // Total Store Ordering (x86-like)
		// Allow load-load reordering, limited store reordering
		return opType == "load" && mem.MemoryOrderingState.LoadLoadReordering
	case "pso": // Partial Store Ordering (SPARC-like)
		// Allow more reordering than TSO
		switch opType {
		case "load":
			return mem.MemoryOrderingState.LoadLoadReordering
		case "store":
			return mem.MemoryOrderingState.StoreStoreReordering
		}
	}

	return false
}

// calculateReorderingDelay calculates how much an operation can be delayed due to reordering
func (mem *MemoryEngine) calculateReorderingDelay(op *Operation) int64 {
	// Reordering delay is typically small (1-5 ticks) and depends on operation complexity
	baseDelay := mem.getProfileInt("memory_ordering", "base_reordering_delay", 2)

	// Larger operations may have more reordering potential
	if op.DataSize > 1024 {
		return int64(baseDelay * 2)
	}

	return int64(baseDelay)
}

// checkMemoryDependencies checks for memory dependencies that prevent reordering
func (mem *MemoryEngine) checkMemoryDependencies(op *Operation, opType string) time.Duration {
	currentAddr := mem.calculateMemoryAddress(op)

	// Check for dependencies with pending operations
	for _, pendingOp := range mem.MemoryOrderingState.PendingOperations {
		pendingAddr := pendingOp.Address

		// Check for address conflicts (same cache line)
		cacheLineSize := uint64(64) // Default cache line size
		currentCacheLine := currentAddr / cacheLineSize
		pendingCacheLine := pendingAddr / cacheLineSize

		if currentCacheLine == pendingCacheLine {
			// Same cache line - check for hazards
			if mem.hasMemoryHazard(opType, pendingOp.OpType) {
				// Dependency found - must wait for pending operation
				dependencyDelay := mem.getProfileFloat("memory_ordering", "dependency_delay", 5.0) // 5ns default
				return time.Duration(dependencyDelay * float64(time.Nanosecond))
			}
		}
	}

	return 0
}

// hasMemoryHazard checks if two operations have a memory hazard
func (mem *MemoryEngine) hasMemoryHazard(currentOpType, pendingOpType string) bool {
	switch mem.MemoryOrderingState.OrderingModel {
	case "strong":
		// Strong ordering - all operations are ordered
		return true
	case "weak":
		// Weak ordering - only certain combinations create hazards
		return (currentOpType == "store" && pendingOpType == "load") ||
			   (currentOpType == "load" && pendingOpType == "store") ||
			   (currentOpType == "store" && pendingOpType == "store")
	case "tso": // Total Store Ordering
		// TSO allows load-load reordering but not store reordering
		return (currentOpType == "store" || pendingOpType == "store")
	case "pso": // Partial Store Ordering
		// PSO allows more reordering but still has some constraints
		return (currentOpType == "store" && pendingOpType == "load")
	}

	return false
}

// calculateOptimalReorderingDelay calculates optimal delay for reordering
func (mem *MemoryEngine) calculateOptimalReorderingDelay(op *Operation, opType string) int64 {
	baseDelay := mem.calculateReorderingDelay(op)

	// Analyze pending operations to find optimal insertion point
	optimalDelay := baseDelay

	for _, pendingOp := range mem.MemoryOrderingState.PendingOperations {
		// Try to group similar operations together for better cache locality
		if pendingOp.OpType == opType {
			// Same operation type - try to schedule close together
			tickDiff := pendingOp.ReorderedTick - mem.CurrentTick
			if tickDiff > 0 && tickDiff < optimalDelay {
				optimalDelay = tickDiff + 1 // Schedule right after similar operation
			}
		}
	}

	return optimalDelay
}

// calculateReorderingBenefit calculates the performance benefit from reordering
func (mem *MemoryEngine) calculateReorderingBenefit(op *Operation, opType string) float64 {
	baseBenefit := mem.getProfileFloat("memory_ordering", "reordering_benefit", 0.1) // Default 10% benefit

	// Different operation types have different reordering benefits
	switch opType {
	case "load":
		// Loads can often be reordered for better cache utilization
		return baseBenefit * 1.2
	case "store":
		// Stores can be coalesced and buffered
		return baseBenefit * 1.5
	default:
		return baseBenefit
	}
}

// sortPendingOperationsByTick sorts pending operations by their reordered execution tick
func (mem *MemoryEngine) sortPendingOperationsByTick() {
	// Simple bubble sort for small arrays (reordering window is typically small)
	n := len(mem.MemoryOrderingState.PendingOperations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if mem.MemoryOrderingState.PendingOperations[j].ReorderedTick >
			   mem.MemoryOrderingState.PendingOperations[j+1].ReorderedTick {
				// Swap operations
				mem.MemoryOrderingState.PendingOperations[j],
				mem.MemoryOrderingState.PendingOperations[j+1] =
				mem.MemoryOrderingState.PendingOperations[j+1],
				mem.MemoryOrderingState.PendingOperations[j]
			}
		}
	}
}

// calculateMemoryAddress calculates a memory address for an operation (for ordering and conflict detection)
func (mem *MemoryEngine) calculateMemoryAddress(op *Operation) uint64 {
	// Create deterministic address based on operation characteristics
	opHash := mem.hashOperationForMemoryDecision(op)

	// Simulate memory address space (use operation hash as base address)
	baseAddr := uint64(opHash) << 12 // Align to 4KB pages

	// Add data size offset
	return baseAddr + uint64(op.DataSize)
}

// flushPendingMemoryOperations processes all pending operations in the reordering window
func (mem *MemoryEngine) flushPendingMemoryOperations() {
	// Clear all pending operations (they would be processed immediately)
	mem.MemoryOrderingState.PendingOperations = mem.MemoryOrderingState.PendingOperations[:0]
}

// processReadyReorderedOperations processes operations that are ready to execute from reordering window
func (mem *MemoryEngine) processReadyReorderedOperations() {
	currentTick := mem.CurrentTick
	readyOps := make([]MemoryOrderingOp, 0)
	pendingOps := make([]MemoryOrderingOp, 0)

	// Separate ready and pending operations
	for _, op := range mem.MemoryOrderingState.PendingOperations {
		if op.ReorderedTick <= currentTick {
			readyOps = append(readyOps, op)
		} else {
			pendingOps = append(pendingOps, op)
		}
	}

	// Update pending operations list
	mem.MemoryOrderingState.PendingOperations = pendingOps

	// Ready operations would be processed (in a real implementation, they would be sent back to the processing queue)
	// For simulation purposes, we just track that they were processed
}

// ========================================
// PRIORITY 2 HELPER METHODS
// ========================================

// selectMemoryController determines which memory controller handles a request
func (mem *MemoryEngine) selectMemoryController(op *Operation) int {
	// Use deterministic hash to distribute requests across controllers
	opHash := mem.hashOperationForMemoryDecision(op)
	return int(opHash) % mem.MemoryControllerState.ControllerCount
}

// calculateControllerQueueDelay calculates queuing delay for a memory controller
func (mem *MemoryEngine) calculateControllerQueueDelay(controllerID int) time.Duration {
	// Count active requests for this controller
	activeCount := 0
	for _, req := range mem.MemoryControllerState.ActiveRequests {
		if req.ControllerID == controllerID {
			activeCount++
		}
	}

	// Apply queuing delay based on queue depth
	if activeCount >= mem.MemoryControllerState.QueueDepth {
		// Queue is full - apply backpressure delay
		backpressureDelay := mem.getProfileFloat("memory_controller", "backpressure_delay", 50.0) // 50ns default
		return time.Duration(backpressureDelay * float64(time.Nanosecond))
	}

	// Linear queuing delay based on queue occupancy
	queueFactor := float64(activeCount) / float64(mem.MemoryControllerState.QueueDepth)
	baseQueueDelay := mem.getProfileFloat("memory_controller", "base_queue_delay", 10.0) // 10ns default

	return time.Duration(baseQueueDelay * queueFactor * float64(time.Nanosecond))
}

// calculateControllerBandwidthDelay calculates bandwidth-based delay for a controller
func (mem *MemoryEngine) calculateControllerBandwidthDelay(op *Operation, controllerID int) time.Duration {
	// Calculate data transfer time based on controller bandwidth
	dataSize := float64(op.DataSize) // bytes
	bandwidthBytesPerNs := mem.MemoryControllerState.BandwidthPerController * 1e9 / 8 // Convert GB/s to bytes/ns

	if bandwidthBytesPerNs == 0 {
		return 0
	}

	transferTime := dataSize / bandwidthBytesPerNs // nanoseconds
	return time.Duration(transferTime * float64(time.Nanosecond))
}

// updateControllerState updates memory controller state after processing a request
func (mem *MemoryEngine) updateControllerState(op *Operation, controllerID int) {
	// Add request to active requests (simplified - in real implementation would track completion)
	request := MemoryControllerRequest{
		Operation:      op,
		ControllerID:   controllerID,
		QueuePosition:  len(mem.MemoryControllerState.ActiveRequests),
		StartTick:      mem.CurrentTick,
		CompletionTick: mem.CurrentTick + 1, // Simplified completion time
		Priority:       mem.calculateRequestPriority(op),
	}

	mem.MemoryControllerState.ActiveRequests = append(mem.MemoryControllerState.ActiveRequests, request)

	// Clean up completed requests (simplified)
	mem.cleanupCompletedRequests()
}

// calculateRequestPriority calculates priority for memory controller arbitration
func (mem *MemoryEngine) calculateRequestPriority(op *Operation) int {
	// Higher priority for smaller requests (to reduce head-of-line blocking)
	if op.DataSize <= 64 {
		return 3 // High priority
	} else if op.DataSize <= 1024 {
		return 2 // Medium priority
	}
	return 1 // Low priority
}

// cleanupCompletedRequests removes completed requests from active list
func (mem *MemoryEngine) cleanupCompletedRequests() {
	activeRequests := make([]MemoryControllerRequest, 0)

	for _, req := range mem.MemoryControllerState.ActiveRequests {
		if req.CompletionTick > mem.CurrentTick {
			activeRequests = append(activeRequests, req)
		}
	}

	mem.MemoryControllerState.ActiveRequests = activeRequests
}

// initializeAdvancedNUMAFromBasic initializes advanced NUMA state from basic NUMA configuration
func (mem *MemoryEngine) initializeAdvancedNUMAFromBasic() {
	socketCount := mem.NUMAState.SocketCount
	if socketCount <= 1 {
		return // No NUMA topology to initialize
	}

	// Initialize distance matrix (simplified - real systems have complex topologies)
	mem.AdvancedNUMAState.DistanceMatrix = make([][]float64, socketCount)
	mem.AdvancedNUMAState.BandwidthMatrix = make([][]float64, socketCount)
	mem.AdvancedNUMAState.NodeMemoryPressure = make([]float64, socketCount)

	for i := 0; i < socketCount; i++ {
		mem.AdvancedNUMAState.DistanceMatrix[i] = make([]float64, socketCount)
		mem.AdvancedNUMAState.BandwidthMatrix[i] = make([]float64, socketCount)

		for j := 0; j < socketCount; j++ {
			if i == j {
				// Local node
				mem.AdvancedNUMAState.DistanceMatrix[i][j] = 1.0
				mem.AdvancedNUMAState.BandwidthMatrix[i][j] = 1.0 // Full bandwidth
			} else {
				// Remote node - use basic NUMA penalty as distance
				mem.AdvancedNUMAState.DistanceMatrix[i][j] = mem.NUMAState.CrossSocketPenalty
				mem.AdvancedNUMAState.BandwidthMatrix[i][j] = 1.0 / mem.NUMAState.CrossSocketPenalty
			}
		}

		// Initialize memory pressure to neutral
		mem.AdvancedNUMAState.NodeMemoryPressure[i] = 1.0
	}

	// Initialize topology map (simplified - all nodes connected to all)
	mem.AdvancedNUMAState.TopologyMap = make(map[int][]int)
	for i := 0; i < socketCount; i++ {
		connections := make([]int, 0)
		for j := 0; j < socketCount; j++ {
			if i != j {
				connections = append(connections, j)
			}
		}
		mem.AdvancedNUMAState.TopologyMap[i] = connections
	}
}

// determineSourceNUMANode determines the source NUMA node for an operation
func (mem *MemoryEngine) determineSourceNUMANode(op *Operation) int {
	// Use deterministic hash to simulate thread/process NUMA affinity
	opHash := mem.hashOperationForMemoryDecision(op)
	return int(opHash) % len(mem.AdvancedNUMAState.DistanceMatrix)
}

// determineTargetNUMANode determines the target NUMA node for memory access
func (mem *MemoryEngine) determineTargetNUMANode(op *Operation) int {
	// Use different hash seed to simulate memory allocation patterns
	opHash := mem.hashOperationForMemoryDecision(op)
	targetHash := (opHash >> 8) + 12345 // Different seed
	return int(targetHash) % len(mem.AdvancedNUMAState.DistanceMatrix)
}

// calculateNUMADistancePenalty calculates latency penalty based on NUMA distance
func (mem *MemoryEngine) calculateNUMADistancePenalty(sourceNode, targetNode int) time.Duration {
	if sourceNode >= len(mem.AdvancedNUMAState.DistanceMatrix) ||
	   targetNode >= len(mem.AdvancedNUMAState.DistanceMatrix[0]) {
		return 0
	}

	distance := mem.AdvancedNUMAState.DistanceMatrix[sourceNode][targetNode]
	baseLatency := mem.getProfileFloat("advanced_numa", "base_inter_node_latency", 100.0) // 100ns default

	return time.Duration(baseLatency * distance * float64(time.Nanosecond))
}

// calculateNUMABandwidthPenalty calculates bandwidth penalty between NUMA nodes
func (mem *MemoryEngine) calculateNUMABandwidthPenalty(sourceNode, targetNode int, op *Operation) time.Duration {
	if sourceNode >= len(mem.AdvancedNUMAState.BandwidthMatrix) ||
	   targetNode >= len(mem.AdvancedNUMAState.BandwidthMatrix[0]) {
		return 0
	}

	bandwidthFactor := mem.AdvancedNUMAState.BandwidthMatrix[sourceNode][targetNode]
	if bandwidthFactor >= 1.0 {
		return 0 // No penalty for local or high-bandwidth connections
	}

	// Calculate additional time due to reduced bandwidth
	baseBandwidth := mem.getProfileFloat("advanced_numa", "base_bandwidth_gbps", 100.0) // 100 GB/s default
	effectiveBandwidth := baseBandwidth * bandwidthFactor

	dataSize := float64(op.DataSize) // bytes
	bandwidthBytesPerNs := effectiveBandwidth * 1e9 / 8 // Convert GB/s to bytes/ns

	if bandwidthBytesPerNs == 0 {
		return 0
	}

	transferTime := dataSize / bandwidthBytesPerNs // nanoseconds
	baseTransferTime := dataSize / (baseBandwidth * 1e9 / 8)

	penalty := transferTime - baseTransferTime
	if penalty < 0 {
		penalty = 0
	}

	return time.Duration(penalty * float64(time.Nanosecond))
}

// evaluatePageMigration evaluates potential benefits of page migration
func (mem *MemoryEngine) evaluatePageMigration(op *Operation, sourceNode, targetNode int) time.Duration {
	// Skip if migration threshold not configured
	if mem.AdvancedNUMAState.MigrationThreshold == 0 {
		return 0
	}

	// Skip if source and target are the same node
	if sourceNode == targetNode {
		return 0
	}

	// Track access patterns for this memory page
	pageAddr := mem.calculatePageAddress(op)
	accessPattern := mem.trackPageAccessPattern(pageAddr, sourceNode, targetNode)

	// Calculate migration cost vs benefit
	migrationCost := mem.calculatePageMigrationCost(op, sourceNode, targetNode)
	migrationBenefit := mem.calculatePageMigrationBenefit(op, accessPattern, sourceNode, targetNode)

	// Only migrate if benefit exceeds cost and threshold
	if migrationBenefit > migrationCost && accessPattern.MigrationScore > mem.AdvancedNUMAState.MigrationThreshold {
		// Apply migration - return net benefit
		mem.performPageMigration(pageAddr, sourceNode, targetNode)
		return time.Duration((migrationBenefit - migrationCost) * float64(time.Nanosecond))
	}

	return 0
}

// PageAccessPattern tracks access patterns for page migration decisions
type PageAccessPattern struct {
	PageAddress     uint64    `json:"page_address"`
	AccessCount     int       `json:"access_count"`
	SourceNodeHits  int       `json:"source_node_hits"`
	TargetNodeHits  int       `json:"target_node_hits"`
	LastAccessTick  int64     `json:"last_access_tick"`
	MigrationScore  float64   `json:"migration_score"`
	AccessFrequency float64   `json:"access_frequency"`
}

// trackPageAccessPattern tracks and analyzes page access patterns
func (mem *MemoryEngine) trackPageAccessPattern(pageAddr uint64, sourceNode, targetNode int) *PageAccessPattern {
	// Initialize page access tracking if not exists
	if mem.AdvancedNUMAState.PageAccessPatterns == nil {
		mem.AdvancedNUMAState.PageAccessPatterns = make(map[uint64]*PageAccessPattern)
	}

	pattern, exists := mem.AdvancedNUMAState.PageAccessPatterns[pageAddr]
	if !exists {
		pattern = &PageAccessPattern{
			PageAddress:     pageAddr,
			AccessCount:     0,
			SourceNodeHits:  0,
			TargetNodeHits:  0,
			LastAccessTick:  0,
			MigrationScore:  0.0,
			AccessFrequency: 0.0,
		}
		mem.AdvancedNUMAState.PageAccessPatterns[pageAddr] = pattern
	}

	// Update access pattern
	pattern.AccessCount++
	pattern.LastAccessTick = mem.CurrentTick

	// Track node affinity
	if sourceNode == targetNode {
		pattern.SourceNodeHits++
	} else {
		pattern.TargetNodeHits++
	}

	// Calculate access frequency (accesses per tick)
	if pattern.AccessCount > 1 {
		ticksSinceFirst := mem.CurrentTick - (pattern.LastAccessTick - int64(pattern.AccessCount-1))
		if ticksSinceFirst > 0 {
			pattern.AccessFrequency = float64(pattern.AccessCount) / float64(ticksSinceFirst)
		}
	}

	// Calculate migration score based on access patterns
	pattern.MigrationScore = mem.calculateMigrationScore(pattern, sourceNode, targetNode)

	// Cleanup old patterns to prevent memory leak
	mem.cleanupOldAccessPatterns()

	return pattern
}

// calculatePageAddress calculates the page address for an operation
func (mem *MemoryEngine) calculatePageAddress(op *Operation) uint64 {
	memAddr := mem.calculateMemoryAddress(op)
	pageSize := uint64(4096) // Default 4KB pages
	if mem.VirtualMemoryState.PageSize > 0 {
		pageSize = uint64(mem.VirtualMemoryState.PageSize)
	}
	return (memAddr / pageSize) * pageSize // Align to page boundary
}

// calculateMigrationScore calculates a score indicating migration benefit
func (mem *MemoryEngine) calculateMigrationScore(pattern *PageAccessPattern, sourceNode, targetNode int) float64 {
	// Base score on access frequency and node affinity
	baseScore := pattern.AccessFrequency * 0.5

	// Increase score if most accesses are from target node
	if pattern.AccessCount > 0 {
		targetNodeRatio := float64(pattern.TargetNodeHits) / float64(pattern.AccessCount)
		baseScore += targetNodeRatio * 0.4
	}

	// Increase score for recent frequent access
	ticksSinceLastAccess := mem.CurrentTick - pattern.LastAccessTick
	if ticksSinceLastAccess < 100 { // Recent access
		baseScore += 0.1
	}

	// Consider NUMA distance - closer nodes get higher scores
	if len(mem.AdvancedNUMAState.DistanceMatrix) > sourceNode &&
	   len(mem.AdvancedNUMAState.DistanceMatrix[sourceNode]) > targetNode {
		distance := mem.AdvancedNUMAState.DistanceMatrix[sourceNode][targetNode]
		if distance > 0 {
			baseScore += (1.0 / distance) * 0.1 // Closer nodes get higher scores
		}
	}

	return baseScore
}

// calculatePageMigrationCost calculates the cost of migrating a page
func (mem *MemoryEngine) calculatePageMigrationCost(op *Operation, sourceNode, targetNode int) float64 {
	// Base migration cost from profile
	baseCost := mem.getProfileFloat("advanced_numa", "migration_base_cost", 100.0) // 100ns default

	// Cost increases with distance between nodes
	if len(mem.AdvancedNUMAState.DistanceMatrix) > sourceNode &&
	   len(mem.AdvancedNUMAState.DistanceMatrix[sourceNode]) > targetNode {
		distance := mem.AdvancedNUMAState.DistanceMatrix[sourceNode][targetNode]
		baseCost *= distance
	}

	// Cost increases with memory pressure on target node
	if targetNode < len(mem.AdvancedNUMAState.NodeMemoryPressure) {
		pressure := mem.AdvancedNUMAState.NodeMemoryPressure[targetNode]
		baseCost *= (1.0 + pressure) // Higher pressure = higher cost
	}

	// Cost increases with page size
	pageSize := float64(4096) // Default 4KB
	if mem.VirtualMemoryState.PageSize > 0 {
		pageSize = float64(mem.VirtualMemoryState.PageSize)
	}
	sizeFactor := pageSize / 4096.0 // Normalize to 4KB baseline
	baseCost *= sizeFactor

	return baseCost
}

// calculatePageMigrationBenefit calculates the benefit of migrating a page
func (mem *MemoryEngine) calculatePageMigrationBenefit(op *Operation, pattern *PageAccessPattern, sourceNode, targetNode int) float64 {
	// Base benefit from avoiding cross-node access
	baseBenefit := mem.getProfileFloat("advanced_numa", "migration_benefit", 50.0) // 50ns default

	// Benefit scales with access frequency
	baseBenefit *= pattern.AccessFrequency

	// Benefit scales with expected future accesses
	if pattern.AccessCount > 0 {
		targetNodeRatio := float64(pattern.TargetNodeHits) / float64(pattern.AccessCount)
		baseBenefit *= targetNodeRatio // More benefit if target node is frequently accessed
	}

	// Benefit from reduced bandwidth contention
	if len(mem.AdvancedNUMAState.BandwidthMatrix) > sourceNode &&
	   len(mem.AdvancedNUMAState.BandwidthMatrix[sourceNode]) > targetNode {
		// Higher bandwidth between nodes = higher benefit
		bandwidth := mem.AdvancedNUMAState.BandwidthMatrix[sourceNode][targetNode]
		baseBenefit *= (bandwidth / 100.0) // Normalize to 100 GB/s baseline
	}

	return baseBenefit
}

// performPageMigration simulates the page migration process
func (mem *MemoryEngine) performPageMigration(pageAddr uint64, sourceNode, targetNode int) {
	// Update page location tracking
	if mem.AdvancedNUMAState.PageNodeMapping == nil {
		mem.AdvancedNUMAState.PageNodeMapping = make(map[uint64]int)
	}

	// Record the migration
	mem.AdvancedNUMAState.PageNodeMapping[pageAddr] = targetNode

	// Update migration statistics
	mem.AdvancedNUMAState.MigrationCount++
	mem.AdvancedNUMAState.LastMigrationTick = mem.CurrentTick

	// Update memory pressure on both nodes
	if sourceNode < len(mem.AdvancedNUMAState.NodeMemoryPressure) {
		mem.AdvancedNUMAState.NodeMemoryPressure[sourceNode] -= 0.01 // Reduce pressure on source
		if mem.AdvancedNUMAState.NodeMemoryPressure[sourceNode] < 0 {
			mem.AdvancedNUMAState.NodeMemoryPressure[sourceNode] = 0
		}
	}

	if targetNode < len(mem.AdvancedNUMAState.NodeMemoryPressure) {
		mem.AdvancedNUMAState.NodeMemoryPressure[targetNode] += 0.01 // Increase pressure on target
		if mem.AdvancedNUMAState.NodeMemoryPressure[targetNode] > 1.0 {
			mem.AdvancedNUMAState.NodeMemoryPressure[targetNode] = 1.0
		}
	}
}

// cleanupOldAccessPatterns removes old access patterns to prevent memory leaks
func (mem *MemoryEngine) cleanupOldAccessPatterns() {
	if mem.AdvancedNUMAState.PageAccessPatterns == nil {
		return
	}

	// Remove patterns that haven't been accessed recently
	maxAge := int64(10000) // 10,000 ticks
	currentTick := mem.CurrentTick

	for pageAddr, pattern := range mem.AdvancedNUMAState.PageAccessPatterns {
		if currentTick - pattern.LastAccessTick > maxAge {
			delete(mem.AdvancedNUMAState.PageAccessPatterns, pageAddr)
		}
	}

	// Limit total number of tracked patterns
	maxPatterns := 1000
	if len(mem.AdvancedNUMAState.PageAccessPatterns) > maxPatterns {
		// Remove oldest patterns
		oldestTick := currentTick
		var oldestAddr uint64

		for pageAddr, pattern := range mem.AdvancedNUMAState.PageAccessPatterns {
			if pattern.LastAccessTick < oldestTick {
				oldestTick = pattern.LastAccessTick
				oldestAddr = pageAddr
			}
		}

		if oldestAddr != 0 {
			delete(mem.AdvancedNUMAState.PageAccessPatterns, oldestAddr)
		}
	}
}

// applyNUMAAffinityPolicy applies NUMA affinity policy effects
func (mem *MemoryEngine) applyNUMAAffinityPolicy(op *Operation, sourceNode, targetNode int) time.Duration {
	switch mem.AdvancedNUMAState.NodeAffinityPolicy {
	case "strict":
		// Strict affinity - penalty for cross-node access
		if sourceNode != targetNode {
			penalty := mem.getProfileFloat("advanced_numa", "strict_affinity_penalty", 50.0) // 50ns default
			return time.Duration(penalty * float64(time.Nanosecond))
		}
	case "preferred":
		// Preferred affinity - smaller penalty for cross-node access
		if sourceNode != targetNode {
			penalty := mem.getProfileFloat("advanced_numa", "preferred_affinity_penalty", 20.0) // 20ns default
			return time.Duration(penalty * float64(time.Nanosecond))
		}
	case "interleave":
		// Interleave policy - no penalty, but potential benefit from load balancing
		return 0
	}

	return 0
}

// updateAdvancedNUMAState updates advanced NUMA state after an operation
func (mem *MemoryEngine) updateAdvancedNUMAState(sourceNode, targetNode int) {
	// Update memory pressure for target node
	if targetNode < len(mem.AdvancedNUMAState.NodeMemoryPressure) {
		mem.AdvancedNUMAState.NodeMemoryPressure[targetNode] += 0.01 // Small pressure increase

		// Apply pressure decay over time
		for i := range mem.AdvancedNUMAState.NodeMemoryPressure {
			if mem.AdvancedNUMAState.NodeMemoryPressure[i] > 1.0 {
				mem.AdvancedNUMAState.NodeMemoryPressure[i] *= 0.99 // Slow decay
			}
		}
	}

	mem.AdvancedNUMAState.LastNUMAUpdate = mem.CurrentTick
}

// checkTLBHit checks if a memory access results in a TLB hit
func (mem *MemoryEngine) checkTLBHit(op *Operation) bool {
	// Use deterministic hash for consistent TLB behavior
	opHash := mem.hashOperationForMemoryDecision(op)
	tlbHash := opHash + 98765 // Different seed for TLB decisions
	hashValue := float64(tlbHash%10000) / 10000.0

	return hashValue < mem.VirtualMemoryState.TLBHitRatio
}

// checkPageFault checks if a memory access results in a page fault (swap)
func (mem *MemoryEngine) checkPageFault(op *Operation) bool {
	// Use deterministic hash for consistent page fault behavior
	opHash := mem.hashOperationForMemoryDecision(op)
	faultHash := opHash + 13579 // Different seed for page fault decisions
	hashValue := float64(faultHash%10000) / 10000.0

	// Page fault probability based on memory pressure
	faultProbability := mem.getProfileFloat("virtual_memory", "page_fault_probability", 0.01) // 1% default

	return hashValue < faultProbability
}

// updateTLB updates TLB state after a page table walk
func (mem *MemoryEngine) updateTLB(op *Operation) {
	// In a real implementation, this would update TLB entries
	// For simulation purposes, we just track that TLB was updated
	mem.VirtualMemoryState.LastVMUpdate = mem.CurrentTick
}

// ========================================
// PRIORITY 3 HELPER METHODS
// ========================================

// initializeEnhancedThermalFromProfile initializes enhanced thermal state from profile
func (mem *MemoryEngine) initializeEnhancedThermalFromProfile(thermalMap map[string]interface{}) {
	// Load thermal zones configuration
	if zones, ok := thermalMap["thermal_zones"].([]interface{}); ok {
		mem.EnhancedThermalState.ThermalZones = make([]ThermalZone, len(zones))

		for i, zoneInterface := range zones {
			if zoneMap, ok := zoneInterface.(map[string]interface{}); ok {
				zone := ThermalZone{
					ZoneID:     i,
					Temperature: mem.EnhancedThermalState.AmbientTemperature,
				}

				if maxTemp, ok := zoneMap["max_temperature"].(float64); ok {
					zone.MaxTemperature = maxTemp
				}
				if heatGen, ok := zoneMap["heat_generation"].(float64); ok {
					zone.HeatGeneration = heatGen
				}
				if cooling, ok := zoneMap["cooling_capacity"].(float64); ok {
					zone.CoolingCapacity = cooling
				}
				if mass, ok := zoneMap["thermal_mass"].(float64); ok {
					zone.ThermalMass = mass
				}

				mem.EnhancedThermalState.ThermalZones[i] = zone
			}
		}
	}

	// Load throttling thresholds
	if thresholds, ok := thermalMap["throttling_thresholds"].([]interface{}); ok {
		mem.EnhancedThermalState.ThrottlingThresholds = make([]float64, len(thresholds))
		for i, threshold := range thresholds {
			if temp, ok := threshold.(float64); ok {
				mem.EnhancedThermalState.ThrottlingThresholds[i] = temp
			}
		}
	}

	// Load throttling levels
	if levels, ok := thermalMap["throttling_levels"].([]interface{}); ok {
		mem.EnhancedThermalState.ThrottlingLevels = make([]float64, len(levels))
		for i, level := range levels {
			if intensity, ok := level.(float64); ok {
				mem.EnhancedThermalState.ThrottlingLevels[i] = intensity
			}
		}
	}
}

// updateThermalZones updates thermal state for all zones based on operation
func (mem *MemoryEngine) updateThermalZones(op *Operation) {
	// Calculate heat generation from this operation
	operationHeat := mem.calculateOperationHeatGeneration(op)

	// Update each thermal zone
	for i := range mem.EnhancedThermalState.ThermalZones {
		zone := &mem.EnhancedThermalState.ThermalZones[i]

		// Add heat from operation (distributed across zones)
		heatPerZone := operationHeat / float64(len(mem.EnhancedThermalState.ThermalZones))
		zone.HeatGeneration += heatPerZone

		// Calculate temperature change based on thermal model
		// ΔT = (Heat Generated - Heat Dissipated) / Thermal Mass
		heatDissipated := zone.CoolingCapacity * mem.EnhancedThermalState.HeatDissipationRate
		netHeat := zone.HeatGeneration - heatDissipated

		if zone.ThermalMass > 0 {
			temperatureChange := netHeat / zone.ThermalMass
			zone.Temperature += temperatureChange

			// Ensure temperature doesn't go below ambient
			if zone.Temperature < mem.EnhancedThermalState.AmbientTemperature {
				zone.Temperature = mem.EnhancedThermalState.AmbientTemperature
			}
		}

		// Apply heat dissipation (cooling)
		zone.HeatGeneration *= 0.95 // Heat dissipation factor
	}

	mem.EnhancedThermalState.LastThermalUpdate = mem.CurrentTick
}

// calculateOperationHeatGeneration calculates heat generated by a memory operation
func (mem *MemoryEngine) calculateOperationHeatGeneration(op *Operation) float64 {
	// Heat generation based on operation size and type
	baseHeat := mem.getProfileFloat("enhanced_thermal", "base_heat_per_operation", 0.1) // 0.1W default

	// Scale by data size
	sizeMultiplier := 1.0 + (float64(op.DataSize) / 1024.0) // More heat for larger operations

	// Different heat for different operation types
	typeMultiplier := 1.0
	switch op.Type {
	case "memory_write":
		typeMultiplier = 1.2 // Writes generate more heat
	case "memory_read":
		typeMultiplier = 1.0 // Reads generate baseline heat
	default:
		typeMultiplier = 1.1 // Other operations generate slightly more heat
	}

	return baseHeat * sizeMultiplier * typeMultiplier
}

// calculateThrottlingLevel calculates throttling level based on temperature
func (mem *MemoryEngine) calculateThrottlingLevel(temperature float64) float64 {
	// Find appropriate throttling level based on temperature thresholds
	for i, threshold := range mem.EnhancedThermalState.ThrottlingThresholds {
		if temperature >= threshold {
			if i < len(mem.EnhancedThermalState.ThrottlingLevels) {
				return mem.EnhancedThermalState.ThrottlingLevels[i]
			}
		}
	}

	return 0.0 // No throttling
}

// End of Memory Engine implementation - all old methods removed
