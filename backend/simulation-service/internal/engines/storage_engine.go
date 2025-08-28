package engines

import (
	"container/heap"
	"fmt"
	"math"
	"time"
)

// StorageProcessingOperation represents an active storage operation
type StorageProcessingOperation struct {
	Operation      *QueuedOperation `json:"operation"`
	StartTick      int64            `json:"start_tick"`
	CompletionTick int64            `json:"completion_tick"`
	IOPSUsed       int              `json:"iops_used"`
	AccessPattern  string           `json:"access_pattern"` // "sequential", "random", "mixed"
	QueuePosition  int              `json:"queue_position"` // For NCQ/TCQ modeling
}

// StorageProcessingHeap implements heap.Interface for storage operations
type StorageProcessingHeap []*StorageProcessingOperation

func (h StorageProcessingHeap) Len() int           { return len(h) }
func (h StorageProcessingHeap) Less(i, j int) bool { return h[i].CompletionTick < h[j].CompletionTick }
func (h StorageProcessingHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *StorageProcessingHeap) Push(x interface{}) {
	*h = append(*h, x.(*StorageProcessingOperation))
}

func (h *StorageProcessingHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// StorageEngine implements the BaseEngine interface for storage operations
type StorageEngine struct {
	*CommonEngine

	// Complexity control interface (like CPU/Memory engines)
	ComplexityInterface *StorageInterface `json:"complexity_interface"`

	// Storage-specific properties from profile (NO HARDCODED VALUES)
	StorageType     string  `json:"storage_type"`      // SSD, HDD, NVMe, etc.
	CapacityGB      int64   `json:"capacity_gb"`
	IOPSRead        int     `json:"iops_read"`
	IOPSWrite       int     `json:"iops_write"`
	LatencyReadUs   float64 `json:"latency_read_us"`   // Microseconds
	LatencyWriteUs  float64 `json:"latency_write_us"`  // Microseconds
	BandwidthMBps   float64 `json:"bandwidth_mbps"`
	QueueDepth      int     `json:"queue_depth"`       // NCQ/TCQ depth
	BlockSizeBytes  int     `json:"block_size_bytes"`  // Minimum I/O unit

	// Active operation tracking (like CPU/Memory engines)
	ActiveOperations *StorageProcessingHeap `json:"-"`
	BusyIOPS         int                    `json:"busy_iops"`

	// Storage-specific state tracking
	IOPSState struct {
		CurrentReadIOPS  int     `json:"current_read_iops"`
		CurrentWriteIOPS int     `json:"current_write_iops"`
		TotalIOPS        int     `json:"total_iops"`
		IOPSUtilization  float64 `json:"iops_utilization"`
	} `json:"iops_state"`

	// Queue depth management (NCQ/TCQ modeling)
	QueueState struct {
		ActiveCommands   int     `json:"active_commands"`
		QueueUtilization float64 `json:"queue_utilization"`
		AverageQueueTime float64 `json:"average_queue_time_us"`
	} `json:"queue_state"`

	// Access pattern tracking
	AccessPatternState struct {
		SequentialRatio    float64            `json:"sequential_ratio"`
		RandomRatio        float64            `json:"random_ratio"`
		LastAccessLBA      int64              `json:"last_access_lba"`
		AccessHistory      []int64            `json:"access_history"`
		PatternEfficiency  map[string]float64 `json:"pattern_efficiency"`
	} `json:"access_pattern_state"`

	// Wear leveling state (SSD-specific)
	WearLevelingState struct {
		Enabled           bool    `json:"enabled"`
		WearLevel         float64 `json:"wear_level"`         // 0.0 to 1.0
		EraseCount        int64   `json:"erase_count"`
		BadBlockCount     int     `json:"bad_block_count"`
		OverProvisioningGB int    `json:"over_provisioning_gb"`
	} `json:"wear_leveling_state"`

	// Controller cache state
	ControllerCacheState struct {
		Enabled       bool    `json:"enabled"`
		CacheSizeMB   int     `json:"cache_size_mb"`
		HitRatio      float64 `json:"hit_ratio"`
		WritePolicy   string  `json:"write_policy"`   // "write-through", "write-back"
		FlushPending  bool    `json:"flush_pending"`
	} `json:"controller_cache_state"`

	// Thermal state (like CPU engine)
	ThermalState struct {
		CurrentTemperatureC float64 `json:"current_temperature_c"`
		ThermalLimitC       float64 `json:"thermal_limit_c"`
		ThrottlingActive    bool    `json:"throttling_active"`
		ThrottlingFactor    float64 `json:"throttling_factor"`
	} `json:"thermal_state"`

	// Power state (HDD-specific)
	PowerState struct {
		CurrentState    string  `json:"current_state"`    // "active", "idle", "standby", "sleep"
		SpinUpTimeMs    float64 `json:"spin_up_time_ms"`
		SpinDownTimeMs  float64 `json:"spin_down_time_ms"`
		IdleTimeoutMs   float64 `json:"idle_timeout_ms"`
		LastActivityTick int64  `json:"last_activity_tick"`
	} `json:"power_state"`

	// Fragmentation state (HDD-specific)
	FragmentationState struct {
		FragmentationLevel float64 `json:"fragmentation_level"` // 0.0 to 1.0
		SeekPenaltyFactor  float64 `json:"seek_penalty_factor"`
		DefragBenefit      float64 `json:"defrag_benefit"`
	} `json:"fragmentation_state"`
}

// NewStorageEngine creates a new Storage engine with profile-driven configuration (NO HARDCODED VALUES)
func NewStorageEngine(queueCapacity int) *StorageEngine {
	common := NewCommonEngine(StorageEngineType, queueCapacity)

	// Initialize processing heap (like CPU/Memory engines)
	activeOps := &StorageProcessingHeap{}
	heap.Init(activeOps)

	storage := &StorageEngine{
		CommonEngine:        common,
		ComplexityInterface: NewStorageInterface(ComplexityAdvanced), // Default to advanced complexity
		ActiveOperations:    activeOps,
		BusyIOPS:           0,

		// All values will be loaded from profile - NO DEFAULTS
		StorageType:     "",
		CapacityGB:      0,
		IOPSRead:        0,
		IOPSWrite:       0,
		LatencyReadUs:   0.0,
		LatencyWriteUs:  0.0,
		BandwidthMBps:   0.0,
		QueueDepth:      0,
		BlockSizeBytes:  0,
	}
	
	// Initialize state structures with minimal defaults (profile will set proper values)
	storage.IOPSState.CurrentReadIOPS = 0
	storage.IOPSState.CurrentWriteIOPS = 0
	storage.IOPSState.TotalIOPS = 0
	storage.IOPSState.IOPSUtilization = 0.0

	storage.QueueState.ActiveCommands = 0
	storage.QueueState.QueueUtilization = 0.0
	storage.QueueState.AverageQueueTime = 0.0

	storage.AccessPatternState.SequentialRatio = 0.0
	storage.AccessPatternState.RandomRatio = 0.0
	storage.AccessPatternState.LastAccessLBA = 0
	storage.AccessPatternState.AccessHistory = make([]int64, 0, 1000)
	storage.AccessPatternState.PatternEfficiency = make(map[string]float64)

	// Initialize wear leveling with defaults (will be overridden by profile)
	storage.WearLevelingState.Enabled = false
	storage.WearLevelingState.WearLevel = 0.0
	storage.WearLevelingState.EraseCount = 0
	storage.WearLevelingState.BadBlockCount = 0
	storage.WearLevelingState.OverProvisioningGB = 0

	// Initialize controller cache with defaults
	storage.ControllerCacheState.Enabled = false
	storage.ControllerCacheState.CacheSizeMB = 0
	storage.ControllerCacheState.HitRatio = 0.0
	storage.ControllerCacheState.WritePolicy = "write-through"
	storage.ControllerCacheState.FlushPending = false

	// Initialize thermal state with defaults
	storage.ThermalState.CurrentTemperatureC = 25.0 // Room temperature
	storage.ThermalState.ThermalLimitC = 70.0       // Will be set by profile
	storage.ThermalState.ThrottlingActive = false
	storage.ThermalState.ThrottlingFactor = 1.0

	// Initialize power state with defaults
	storage.PowerState.CurrentState = "active"
	storage.PowerState.SpinUpTimeMs = 0.0    // SSD default (no spin-up)
	storage.PowerState.SpinDownTimeMs = 0.0  // SSD default (no spin-down)
	storage.PowerState.IdleTimeoutMs = 0.0
	storage.PowerState.LastActivityTick = 0

	// Initialize fragmentation state with defaults
	storage.FragmentationState.FragmentationLevel = 0.0
	storage.FragmentationState.SeekPenaltyFactor = 1.0
	storage.FragmentationState.DefragBenefit = 0.0

	return storage
}

// ProcessOperation processes a single storage operation with realistic storage modeling
func (storage *StorageEngine) ProcessOperation(op *Operation, currentTick int64) *OperationResult {
	storage.CurrentTick = currentTick

	// Calculate base storage access time from profile (IOPS and latency)
	baseTime := storage.calculateBaseStorageAccessTime(op)

	// Apply access pattern optimization (if enabled)
	patternAdjustedTime := baseTime
	if storage.ComplexityInterface.ShouldEnableFeature("sequential_optimization") {
		patternAdjustedTime = storage.applyAccessPatternOptimization(baseTime, op)
	}

	// Apply queue depth management effects (if enabled)
	queueAdjustedTime := patternAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("queue_depth_management") {
		queueAdjustedTime = storage.applyQueueDepthEffects(patternAdjustedTime, op)
	}

	// Apply controller cache effects (if enabled)
	cacheAdjustedTime := queueAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("controller_cache") {
		cacheAdjustedTime = storage.applyControllerCacheEffects(queueAdjustedTime, op)
	}

	// Apply filesystem overhead (if enabled)
	filesystemAdjustedTime := cacheAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("filesystem_overhead") {
		filesystemAdjustedTime = storage.applyFilesystemOverhead(cacheAdjustedTime, op)
	}

	// Apply fragmentation effects (if enabled - HDD specific)
	fragmentationAdjustedTime := filesystemAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("fragmentation_effects") {
		fragmentationAdjustedTime = storage.applyFragmentationEffects(filesystemAdjustedTime, op)
	}

	// Apply wear leveling overhead (if enabled - SSD specific)
	wearAdjustedTime := fragmentationAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("basic_wear_leveling") ||
	   storage.ComplexityInterface.ShouldEnableFeature("advanced_wear_leveling") {
		wearAdjustedTime = storage.applyWearLevelingEffects(fragmentationAdjustedTime, op)
	}

	// Apply power state effects (if enabled - HDD specific)
	powerAdjustedTime := wearAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("power_state_transitions") {
		powerAdjustedTime = storage.applyPowerStateEffects(wearAdjustedTime, op)
	}

	// Apply thermal throttling effects (if enabled)
	thermalAdjustedTime := powerAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("thermal_throttling") {
		thermalAdjustedTime = storage.applyThermalThrottlingEffects(powerAdjustedTime, op)
	}

	// Apply TRIM/garbage collection effects (if enabled - SSD specific)
	trimAdjustedTime := thermalAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("trim_garbage_collection") {
		trimAdjustedTime = storage.applyTrimGarbageCollectionEffects(thermalAdjustedTime, op)
	}

	// Apply advanced prefetching effects (if enabled)
	prefetchAdjustedTime := trimAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("advanced_prefetching") {
		prefetchAdjustedTime = storage.applyAdvancedPrefetchingEffects(trimAdjustedTime, op)
	}

	// Apply compression effects (if enabled)
	compressionAdjustedTime := prefetchAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("compression_effects") {
		compressionAdjustedTime = storage.applyCompressionEffects(prefetchAdjustedTime, op)
	}

	// Apply encryption overhead (if enabled)
	encryptionAdjustedTime := compressionAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("encryption_overhead") {
		encryptionAdjustedTime = storage.applyEncryptionOverhead(compressionAdjustedTime, op)
	}

	// Apply multi-stream I/O effects (if enabled - SSD specific)
	multiStreamAdjustedTime := encryptionAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("multi_stream_io") {
		multiStreamAdjustedTime = storage.applyMultiStreamIOEffects(encryptionAdjustedTime, op)
	}

	// Apply zoned storage effects (if enabled - ZNS SSD specific)
	zonedAdjustedTime := multiStreamAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("zoned_storage") {
		zonedAdjustedTime = storage.applyZonedStorageEffects(multiStreamAdjustedTime, op)
	}

	// Apply error correction effects (if enabled)
	eccAdjustedTime := zonedAdjustedTime
	if storage.ComplexityInterface.ShouldEnableFeature("error_correction") {
		eccAdjustedTime = storage.applyErrorCorrectionEffects(zonedAdjustedTime, op)
	}

	// Apply common performance factors (load, queue, health, variance)
	utilization := storage.calculateCurrentUtilization()
	finalTime := storage.ApplyCommonPerformanceFactors(eccAdjustedTime, utilization)

	// Update dynamic state tracking (if enabled)
	if storage.ComplexityInterface.ShouldEnableFeature("dynamic_behavior") {
		storage.updateStorageState(op, finalTime)
	}

	// Ensure operations take at least 1 tick to complete
	ticksToComplete := storage.DurationToTicks(finalTime)
	if ticksToComplete < 1 {
		ticksToComplete = 1
	}

	// Calculate penalty factors for routing decisions
	loadPenalty := 1.0 + (storage.IOPSState.IOPSUtilization * 0.6) // IOPS utilization penalty
	queuePenalty := 1.0 + (storage.QueueState.QueueUtilization * 0.4) // Queue depth penalty
	thermalPenalty := 1.0
	if storage.ThermalState.ThrottlingActive {
		thermalPenalty = 1.5 // Significant penalty for thermal throttling
	}
	contentionPenalty := 1.0 + (storage.QueueState.QueueUtilization * 0.2) // Queue contention
	healthPenalty := 1.0 + (1.0 - storage.GetHealth().Score) * 0.3 // Storage health is critical

	// Power state impact
	powerPenalty := 1.0
	switch storage.PowerState.CurrentState {
	case "active":
		powerPenalty = 1.0
	case "idle":
		powerPenalty = 1.1 // Slight penalty for spin-up
	case "standby":
		powerPenalty = 1.3 // Moderate penalty for wake-up
	case "sleep":
		powerPenalty = 2.0 // Significant penalty for full wake-up
	}

	totalPenaltyFactor := loadPenalty * queuePenalty * thermalPenalty * contentionPenalty * healthPenalty * powerPenalty

	// Ensure total penalty factor is valid
	if totalPenaltyFactor != totalPenaltyFactor || totalPenaltyFactor <= 0 { // Check for NaN or invalid values
		totalPenaltyFactor = 1.0
	}

	// Determine performance grade
	performanceGrade := "A"
	recommendedAction := "continue"
	if totalPenaltyFactor > 3.0 {
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

	// Create operation result with penalty information
	result := &OperationResult{
		OperationID:    op.ID,
		OperationType:  op.Type,
		ProcessingTime: finalTime,
		CompletedTick:  currentTick + ticksToComplete,
		CompletedAt:    currentTick + ticksToComplete,
		Success:        true,
		NextComponent:  op.NextComponent,
		PenaltyInfo: &PenaltyInformation{
			EngineType:           StorageEngineType,
			EngineID:            storage.ID,
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
			StoragePenalties: &StoragePenaltyDetails{
				IOPSUtilization:   storage.IOPSState.IOPSUtilization,
				QueueDepth:        storage.QueueState.QueueUtilization,
				AccessPattern:     storage.determineAccessPattern(op),
				ThermalThrottling: func() float64 { if storage.ThermalState.ThrottlingActive { return 1.0 } else { return 0.0 } }(),
				PowerStateImpact:  powerPenalty,
			},
		},
		Metrics: map[string]interface{}{
			"base_time_us":           baseTime.Microseconds(),
			"final_time_us":          finalTime.Microseconds(),
			"access_pattern":         storage.determineAccessPattern(op),
			"iops_utilization":       storage.IOPSState.IOPSUtilization,
			"queue_utilization":      storage.QueueState.QueueUtilization,
			"thermal_throttling":     storage.ThermalState.ThrottlingActive,
			"power_state":           storage.PowerState.CurrentState,
		},
	}

	return result
}

// ProcessTick processes one simulation tick following CPU/Memory engine pattern
func (storage *StorageEngine) ProcessTick(currentTick int64) []OperationResult {
	storage.CurrentTick = currentTick
	results := make([]OperationResult, 0)

	// Step 1: Complete finished operations from active heap
	for storage.ActiveOperations.Len() > 0 {
		nextOp := (*storage.ActiveOperations)[0]
		if nextOp.CompletionTick <= currentTick {
			// Operation completed
			completedOp := heap.Pop(storage.ActiveOperations).(*StorageProcessingOperation)

			// Update busy IOPS count
			storage.BusyIOPS -= completedOp.IOPSUsed

			// Create completion result
			result := OperationResult{
				OperationID:    completedOp.Operation.Operation.ID,
				OperationType:  completedOp.Operation.Operation.Type,
				ProcessingTime: time.Duration(completedOp.CompletionTick-completedOp.StartTick) * storage.TickDuration,
				CompletedTick:  completedOp.CompletionTick,
				CompletedAt:    completedOp.CompletionTick,
				Success:        true,
				NextComponent:  completedOp.Operation.Operation.NextComponent,
				Metrics: map[string]interface{}{
					"access_pattern":     completedOp.AccessPattern,
					"queue_position":     completedOp.QueuePosition,
					"iops_used":         completedOp.IOPSUsed,
					"completion_tick":   completedOp.CompletionTick,
				},
			}
			results = append(results, result)

			// Update operation history for convergence
			storage.AddOperationToHistory(result.ProcessingTime)
			storage.CompletedOps++
		} else {
			break // No more completed operations
		}
	}

	// Step 2: Start new operations from queue (max 3 per tick like CPU/Memory)
	operationsStarted := 0
	maxOperationsPerTick := 3 // Same limit as CPU/Memory engines

	for operationsStarted < maxOperationsPerTick && storage.GetQueueLength() > 0 {
		// Check if we have available IOPS capacity
		if storage.BusyIOPS >= storage.getMaxConcurrentIOPS() {
			break // No IOPS capacity available
		}

		queuedOp := storage.DequeueOperation()
		if queuedOp == nil {
			break
		}

		// Process the operation to get timing
		result := storage.ProcessOperation(queuedOp.Operation, currentTick)

		// Create processing operation for active heap
		processingOp := &StorageProcessingOperation{
			Operation:      queuedOp,
			StartTick:      currentTick,
			CompletionTick: result.CompletedTick,
			IOPSUsed:       storage.calculateIOPSUsage(queuedOp.Operation),
			AccessPattern:  storage.determineAccessPattern(queuedOp.Operation),
			QueuePosition:  storage.QueueState.ActiveCommands,
		}

		// Add to active operations heap
		heap.Push(storage.ActiveOperations, processingOp)

		// Update busy IOPS count
		storage.BusyIOPS += processingOp.IOPSUsed

		operationsStarted++
	}

	// Step 3: Update storage state
	storage.updateStorageStatePerTick()

	// Update health metrics
	storage.UpdateHealth()

	// Update dynamic behavior (if enabled)
	if storage.ComplexityInterface.ShouldEnableFeature("dynamic_behavior") {
		storage.UpdateDynamicBehavior()
	}

	return results
}

// calculateBaseStorageAccessTime calculates base storage access time from profile (NO HARDCODED VALUES)
func (storage *StorageEngine) calculateBaseStorageAccessTime(op *Operation) time.Duration {
	// Get base latency from profile based on operation type
	var baseLatencyUs float64

	switch op.Type {
	case OpStorageRead:
		baseLatencyUs = storage.LatencyReadUs
	case OpStorageWrite:
		baseLatencyUs = storage.LatencyWriteUs
	case OpStorageSeek:
		// Seek operations use read latency as baseline
		baseLatencyUs = storage.LatencyReadUs * 2.0 // Seek typically takes longer
	default:
		baseLatencyUs = storage.LatencyReadUs // Default to read latency
	}

	// Fallback to reasonable defaults if profile not loaded (for testing)
	if baseLatencyUs <= 0 {
		switch op.Type {
		case OpStorageRead:
			baseLatencyUs = 50.0 // 50 microseconds default
		case OpStorageWrite:
			baseLatencyUs = 60.0 // 60 microseconds default
		case OpStorageSeek:
			baseLatencyUs = 100.0 // 100 microseconds default
		default:
			baseLatencyUs = 50.0
		}
	}

	// Apply size dependency for large operations (bandwidth-limited)
	if op.DataSize > int64(storage.BlockSizeBytes) && storage.BandwidthMBps > 0 {
		// Calculate transfer time based on bandwidth
		transferTimeUs := float64(op.DataSize) / (storage.BandwidthMBps * 1024 * 1024) * 1000000
		baseLatencyUs += transferTimeUs
	}

	return time.Duration(baseLatencyUs * float64(time.Microsecond))
}

// calculateCurrentUtilization calculates current storage utilization
func (storage *StorageEngine) calculateCurrentUtilization() float64 {
	maxIOPS := storage.getMaxConcurrentIOPS()
	if maxIOPS == 0 {
		return 0.0
	}
	return float64(storage.BusyIOPS) / float64(maxIOPS)
}

// getMaxConcurrentIOPS returns the maximum concurrent IOPS based on storage type and queue depth
func (storage *StorageEngine) getMaxConcurrentIOPS() int {
	// Use the lower of IOPS limits and queue depth
	maxReadWrite := storage.IOPSRead + storage.IOPSWrite
	if storage.QueueDepth > 0 && storage.QueueDepth < maxReadWrite {
		return storage.QueueDepth
	}
	return maxReadWrite
}

// calculateIOPSUsage calculates how many IOPS this operation will consume
func (storage *StorageEngine) calculateIOPSUsage(op *Operation) int {
	// Most operations use 1 IOPS, but large operations may use more
	if op.DataSize > int64(storage.BlockSizeBytes)*16 { // Large operations (>16 blocks)
		return 2 // Use 2 IOPS for large operations
	}
	return 1 // Standard 1 IOPS per operation
}

// determineAccessPattern determines the access pattern for an operation
func (storage *StorageEngine) determineAccessPattern(op *Operation) string {
	// Use deterministic pattern based on operation characteristics
	hash := uint32(storage.CompletedOps + int64(op.DataSize))

	// 70% sequential, 30% random (typical workload)
	if hash%100 < 70 {
		return "sequential"
	}
	return "random"
}

// applyAccessPatternOptimization applies sequential vs random access optimization
func (storage *StorageEngine) applyAccessPatternOptimization(baseTime time.Duration, op *Operation) time.Duration {
	pattern := storage.determineAccessPattern(op)

	// Update access pattern state
	storage.AccessPatternState.LastAccessLBA = op.DataSize // Simplified LBA tracking

	// Sequential access is faster, especially for HDDs
	switch pattern {
	case "sequential":
		storage.AccessPatternState.SequentialRatio += 0.01
		storage.AccessPatternState.RandomRatio = math.Max(0, storage.AccessPatternState.RandomRatio-0.01)

		// Sequential optimization benefit depends on storage type
		if storage.StorageType == "HDD" {
			return time.Duration(float64(baseTime) * 0.6) // 40% faster for HDD sequential
		} else {
			return time.Duration(float64(baseTime) * 0.9) // 10% faster for SSD sequential
		}

	case "random":
		storage.AccessPatternState.RandomRatio += 0.01
		storage.AccessPatternState.SequentialRatio = math.Max(0, storage.AccessPatternState.SequentialRatio-0.01)

		// Random access penalty, especially for HDDs
		if storage.StorageType == "HDD" {
			return time.Duration(float64(baseTime) * 1.8) // 80% slower for HDD random
		} else {
			return time.Duration(float64(baseTime) * 1.1) // 10% slower for SSD random
		}

	default:
		return baseTime
	}
}

// applyQueueDepthEffects applies NCQ/TCQ queue depth management effects
func (storage *StorageEngine) applyQueueDepthEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Update queue state
	storage.QueueState.ActiveCommands = storage.ActiveOperations.Len()
	storage.QueueState.QueueUtilization = float64(storage.QueueState.ActiveCommands) / float64(storage.QueueDepth)

	// Queue depth benefits (parallel processing) vs overhead
	utilization := storage.QueueState.QueueUtilization

	var queueFactor float64
	switch {
	case utilization < 0.5:
		// Low utilization - good parallelism benefit
		queueFactor = 0.8 // 20% faster due to parallelism

	case utilization < 0.8:
		// Moderate utilization - some benefit
		queueFactor = 0.9 // 10% faster

	case utilization < 0.95:
		// High utilization - queue management overhead
		queueFactor = 1.2 // 20% slower due to queue management

	default:
		// Queue saturation - significant overhead
		queueFactor = 1.5 // 50% slower due to queue contention
	}

	// Update average queue time for metrics
	storage.QueueState.AverageQueueTime = float64(baseTime.Microseconds()) * (queueFactor - 1.0)

	return time.Duration(float64(baseTime) * queueFactor)
}

// applyControllerCacheEffects applies storage controller cache effects
func (storage *StorageEngine) applyControllerCacheEffects(baseTime time.Duration, op *Operation) time.Duration {
	if !storage.ControllerCacheState.Enabled {
		return baseTime
	}

	// Determine cache hit based on operation type and access pattern
	var hitRatio float64
	pattern := storage.determineAccessPattern(op)

	// Cache hit ratios depend on access pattern and operation type
	switch op.Type {
	case OpStorageWrite:
		if pattern == "sequential" {
			hitRatio = storage.ControllerCacheState.HitRatio * 1.2 // Better hit ratio for sequential writes
		} else {
			hitRatio = storage.ControllerCacheState.HitRatio * 0.8 // Lower hit ratio for random writes
		}
	case OpStorageRead:
		if pattern == "sequential" {
			hitRatio = storage.ControllerCacheState.HitRatio * 1.1 // Slightly better for sequential reads
		} else {
			hitRatio = storage.ControllerCacheState.HitRatio
		}
	default:
		hitRatio = 0.0 // No cache benefit for other operations
	}

	// Deterministic cache hit check
	hash := uint32(storage.CompletedOps + int64(op.DataSize))
	isCacheHit := float64(hash%10000)/10000.0 < hitRatio

	if isCacheHit {
		// Cache hit - much faster
		if op.Type == OpStorageWrite && storage.ControllerCacheState.WritePolicy == "write-back" {
			return time.Duration(float64(baseTime) * 0.05) // Write-back cache: very fast
		} else if op.Type == OpStorageWrite {
			return time.Duration(float64(baseTime) * 0.1) // Write-through cache: faster
		} else {
			return time.Duration(float64(baseTime) * 0.2) // Read cache: 5x faster
		}
	}

	// Cache miss - full storage access
	return baseTime
}

// applyFilesystemOverhead applies filesystem metadata operation overhead
func (storage *StorageEngine) applyFilesystemOverhead(baseTime time.Duration, op *Operation) time.Duration {
	// Filesystem operations have metadata overhead
	var overheadFactor float64

	switch op.Type {
	case OpStorageWrite:
		// Writes require metadata updates (inode, allocation tables, etc.)
		overheadFactor = 1.15 // 15% overhead for metadata updates

	case OpStorageRead:
		// Reads require metadata lookups
		overheadFactor = 1.05 // 5% overhead for metadata lookups

	case OpStorageSeek:
		// Seeks require directory traversal
		overheadFactor = 1.10 // 10% overhead for directory operations

	default:
		overheadFactor = 1.0
	}

	// Small files have proportionally higher overhead
	if op.DataSize < 4096 { // Less than 4KB
		overheadFactor += 0.1 // Additional 10% overhead for small files
	}

	return time.Duration(float64(baseTime) * overheadFactor)
}

// applyFragmentationEffects applies HDD fragmentation effects
func (storage *StorageEngine) applyFragmentationEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Only applies to HDDs
	if storage.StorageType != "HDD" {
		return baseTime
	}

	// Fragmentation increases seek time
	fragmentationPenalty := 1.0 + storage.FragmentationState.FragmentationLevel*0.5 // Up to 50% penalty
	seekPenalty := storage.FragmentationState.SeekPenaltyFactor

	// Random access suffers more from fragmentation
	pattern := storage.determineAccessPattern(op)
	if pattern == "random" {
		fragmentationPenalty *= 1.3 // 30% additional penalty for random access
	}

	// Update fragmentation level based on write operations
	if op.Type == OpStorageWrite {
		storage.FragmentationState.FragmentationLevel += 0.0001 // Gradual fragmentation increase
		storage.FragmentationState.FragmentationLevel = math.Min(storage.FragmentationState.FragmentationLevel, 1.0)
	}

	return time.Duration(float64(baseTime) * fragmentationPenalty * seekPenalty)
}

// applyWearLevelingEffects applies SSD wear leveling overhead
func (storage *StorageEngine) applyWearLevelingEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Only applies to SSDs
	if storage.StorageType == "HDD" {
		return baseTime
	}

	if !storage.WearLevelingState.Enabled {
		return baseTime
	}

	// Wear leveling adds overhead, especially for writes
	var wearOverhead float64

	if op.Type == OpStorageWrite {
		// Write operations trigger wear leveling
		wearOverhead = 1.05 + storage.WearLevelingState.WearLevel*0.1 // 5-15% overhead

		// Update wear level
		storage.WearLevelingState.EraseCount++
		storage.WearLevelingState.WearLevel += 0.000001 // Very gradual wear increase
		storage.WearLevelingState.WearLevel = math.Min(storage.WearLevelingState.WearLevel, 1.0)
	} else {
		// Read operations have minimal wear leveling overhead
		wearOverhead = 1.01 // 1% overhead
	}

	return time.Duration(float64(baseTime) * wearOverhead)
}

// applyPowerStateEffects applies HDD power state effects
func (storage *StorageEngine) applyPowerStateEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Only applies to HDDs
	if storage.StorageType != "HDD" {
		return baseTime
	}

	// Check if drive needs to spin up
	idleTime := storage.CurrentTick - storage.PowerState.LastActivityTick

	var powerPenalty float64 = 1.0

	switch storage.PowerState.CurrentState {
	case "sleep":
		// Drive is asleep, needs full spin-up
		powerPenalty = 1.0 + storage.PowerState.SpinUpTimeMs/float64(baseTime.Milliseconds())
		storage.PowerState.CurrentState = "active"

	case "standby":
		// Drive is in standby, needs partial spin-up
		powerPenalty = 1.0 + (storage.PowerState.SpinUpTimeMs*0.5)/float64(baseTime.Milliseconds())
		storage.PowerState.CurrentState = "active"

	case "idle":
		// Drive is idle, minimal spin-up needed
		if idleTime > int64(storage.PowerState.IdleTimeoutMs) {
			powerPenalty = 1.1 // 10% penalty for idle recovery
		}
		storage.PowerState.CurrentState = "active"

	case "active":
		// Drive is active, no penalty
		powerPenalty = 1.0
	}

	// Update last activity
	storage.PowerState.LastActivityTick = storage.CurrentTick

	return time.Duration(float64(baseTime) * powerPenalty)
}

// applyThermalThrottlingEffects applies thermal throttling effects
func (storage *StorageEngine) applyThermalThrottlingEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Update temperature based on activity
	utilization := storage.calculateCurrentUtilization()
	targetTemp := 25.0 + utilization*45.0 // 25°C to 70°C based on utilization

	// Thermal inertia (gradual temperature change)
	tempDiff := targetTemp - storage.ThermalState.CurrentTemperatureC
	storage.ThermalState.CurrentTemperatureC += tempDiff * 0.1 // 10% change per tick

	// Check for thermal throttling
	if storage.ThermalState.CurrentTemperatureC > storage.ThermalState.ThermalLimitC {
		storage.ThermalState.ThrottlingActive = true

		// Calculate throttling factor based on excess temperature
		excessTemp := storage.ThermalState.CurrentTemperatureC - storage.ThermalState.ThermalLimitC
		maxExcess := 20.0 // 20°C above limit before severe throttling
		throttleReduction := math.Min(excessTemp/maxExcess, 0.7) // Max 70% reduction
		storage.ThermalState.ThrottlingFactor = 1.0 - throttleReduction

		// Apply throttling
		return time.Duration(float64(baseTime) / storage.ThermalState.ThrottlingFactor)

	} else if storage.ThermalState.CurrentTemperatureC < storage.ThermalState.ThermalLimitC-5.0 {
		// Temperature dropped sufficiently, disable throttling
		storage.ThermalState.ThrottlingActive = false
		storage.ThermalState.ThrottlingFactor = 1.0
	}

	return baseTime
}

// updateStorageState updates storage-specific state after processing an operation
func (storage *StorageEngine) updateStorageState(op *Operation, processingTime time.Duration) {
	// Update IOPS state
	storage.IOPSState.TotalIOPS = storage.IOPSState.CurrentReadIOPS + storage.IOPSState.CurrentWriteIOPS
	storage.IOPSState.IOPSUtilization = storage.calculateCurrentUtilization()

	// Update access pattern history
	if len(storage.AccessPatternState.AccessHistory) >= 1000 {
		storage.AccessPatternState.AccessHistory = storage.AccessPatternState.AccessHistory[1:]
	}
	storage.AccessPatternState.AccessHistory = append(storage.AccessPatternState.AccessHistory, op.DataSize)

	// Update controller cache hit ratio based on recent patterns
	if storage.ControllerCacheState.Enabled {
		pattern := storage.determineAccessPattern(op)
		if pattern == "sequential" {
			storage.ControllerCacheState.HitRatio += 0.001 // Gradual improvement
		} else {
			storage.ControllerCacheState.HitRatio -= 0.0005 // Gradual degradation
		}
		storage.ControllerCacheState.HitRatio = math.Max(0.1, math.Min(0.95, storage.ControllerCacheState.HitRatio))
	}
}

// updateStorageStatePerTick updates storage state each tick
func (storage *StorageEngine) updateStorageStatePerTick() {
	// Update IOPS utilization
	storage.IOPSState.TotalIOPS = storage.BusyIOPS
	storage.IOPSState.IOPSUtilization = storage.calculateCurrentUtilization()

	// Update queue utilization
	storage.QueueState.ActiveCommands = storage.ActiveOperations.Len()
	storage.QueueState.QueueUtilization = float64(storage.QueueState.ActiveCommands) / float64(storage.QueueDepth)

	// Update power state for HDDs (idle timeout)
	if storage.StorageType == "HDD" {
		idleTime := storage.CurrentTick - storage.PowerState.LastActivityTick

		if storage.PowerState.CurrentState == "active" && idleTime > int64(storage.PowerState.IdleTimeoutMs) {
			storage.PowerState.CurrentState = "idle"
		} else if storage.PowerState.CurrentState == "idle" && idleTime > int64(storage.PowerState.IdleTimeoutMs*2) {
			storage.PowerState.CurrentState = "standby"
		} else if storage.PowerState.CurrentState == "standby" && idleTime > int64(storage.PowerState.IdleTimeoutMs*10) {
			storage.PowerState.CurrentState = "sleep"
		}
	}

	// Update fragmentation level for HDDs (gradual defragmentation during idle)
	if storage.StorageType == "HDD" && storage.PowerState.CurrentState == "idle" {
		storage.FragmentationState.FragmentationLevel -= 0.00001 // Very slow defragmentation
		storage.FragmentationState.FragmentationLevel = math.Max(0.0, storage.FragmentationState.FragmentationLevel)
	}
}

// SetComplexityLevel sets the storage simulation complexity level
func (storage *StorageEngine) SetComplexityLevel(level int) error {
	complexityLevel := StorageComplexityLevel(level)
	return storage.ComplexityInterface.SetComplexityLevel(complexityLevel)
}

// GetComplexityLevel returns the current storage simulation complexity level
func (storage *StorageEngine) GetComplexityLevel() int {
	return int(storage.ComplexityInterface.ComplexityLevel)
}

// LoadProfile loads storage configuration from profile (NO HARDCODED VALUES)
func (storage *StorageEngine) LoadProfile(profile *EngineProfile) error {
	if profile == nil {
		return fmt.Errorf("profile cannot be nil")
	}

	// Validate profile type like CPU/Memory engines
	if profile.Type != StorageEngineType {
		return fmt.Errorf("profile type mismatch: expected Storage, got %v", profile.Type)
	}

	// Call common profile loading first (like Memory engine)
	if err := storage.CommonEngine.LoadProfile(profile); err != nil {
		return err
	}

	// Load storage-specific profile data
	return storage.loadStorageSpecificProfile()
}

// loadStorageSpecificProfile loads storage-specific profile data (like CPU/Memory engines)
func (storage *StorageEngine) loadStorageSpecificProfile() error {
	if storage.Profile == nil {
		return fmt.Errorf("profile is nil")
	}

	// Load baseline performance
	if storage.Profile.BaselinePerformance != nil {
		if capacity, ok := storage.Profile.BaselinePerformance["capacity_gb"]; ok {
			storage.CapacityGB = int64(capacity)
		}
		if iopsRead, ok := storage.Profile.BaselinePerformance["iops_read"]; ok {
			storage.IOPSRead = int(iopsRead)
		}
		if iopsWrite, ok := storage.Profile.BaselinePerformance["iops_write"]; ok {
			storage.IOPSWrite = int(iopsWrite)
		}
		if latencyRead, ok := storage.Profile.BaselinePerformance["latency_read_us"]; ok {
			storage.LatencyReadUs = latencyRead
		}
		if latencyWrite, ok := storage.Profile.BaselinePerformance["latency_write_us"]; ok {
			storage.LatencyWriteUs = latencyWrite
		}
		if bandwidth, ok := storage.Profile.BaselinePerformance["bandwidth_mbps"]; ok {
			storage.BandwidthMBps = bandwidth
		}
		if queueDepth, ok := storage.Profile.BaselinePerformance["queue_depth"]; ok {
			storage.QueueDepth = int(queueDepth)
		}
		if blockSize, ok := storage.Profile.BaselinePerformance["block_size_bytes"]; ok {
			storage.BlockSizeBytes = int(blockSize)
		}
	}

	// Load technology specs
	if storage.Profile.TechnologySpecs != nil {
		if storageType, ok := storage.Profile.TechnologySpecs["storage_type"].(string); ok {
			storage.StorageType = storageType
		} else {
			// Default to NVMe if not specified
			storage.StorageType = "NVMe"
		}
		if thermalLimit, ok := storage.Profile.TechnologySpecs["thermal_limit_c"].(float64); ok {
			storage.ThermalState.ThermalLimitC = thermalLimit
		}
	}

	// Load engine-specific configurations
	if storage.Profile.EngineSpecific != nil {
		storage.loadEngineSpecificConfigs()
	}

	// Reinitialize with profile data (like CPU engine)
	storage.initializeFromProfile()

	return nil
}

// loadEngineSpecificConfigs loads all engine-specific configurations (like CPU engine)
func (storage *StorageEngine) loadEngineSpecificConfigs() {
	// Load controller cache configuration
	if cacheConfig, ok := storage.Profile.EngineSpecific["controller_cache"].(map[string]interface{}); ok {
		storage.loadControllerCacheConfig(cacheConfig)
	}

	// Load wear leveling configuration
	if wearConfig, ok := storage.Profile.EngineSpecific["wear_leveling"].(map[string]interface{}); ok {
		storage.loadWearLevelingConfig(wearConfig)
	}

	// Load power management configuration
	if powerConfig, ok := storage.Profile.EngineSpecific["power_management"].(map[string]interface{}); ok {
		storage.loadPowerManagementConfig(powerConfig)
	}

	// Load thermal configuration
	if thermalConfig, ok := storage.Profile.EngineSpecific["thermal_behavior"].(map[string]interface{}); ok {
		storage.loadThermalConfig(thermalConfig)
	}
}

// initializeFromProfile reinitializes engine state from loaded profile (like CPU engine)
func (storage *StorageEngine) initializeFromProfile() {
	// Initialize controller cache state from profile
	storage.initializeControllerCacheState()

	// Initialize wear leveling state from profile
	storage.initializeWearLevelingState()

	// Initialize thermal state from profile
	storage.initializeThermalState()

	// Initialize power state from profile
	storage.initializePowerState()

	// Initialize access pattern state
	storage.initializeAccessPatternState()
}

// loadControllerCacheConfig loads controller cache configuration from profile
func (storage *StorageEngine) loadControllerCacheConfig(config map[string]interface{}) {
	if cacheSize, ok := config["cache_size_mb"].(float64); ok {
		storage.ControllerCacheState.CacheSizeMB = int(cacheSize)
		storage.ControllerCacheState.Enabled = storage.ControllerCacheState.CacheSizeMB > 0
	}
	if hitRatio, ok := config["hit_ratio"].(float64); ok {
		storage.ControllerCacheState.HitRatio = hitRatio
	}
	if writePolicy, ok := config["write_policy"].(string); ok {
		storage.ControllerCacheState.WritePolicy = writePolicy
	}
}

// loadWearLevelingConfig loads wear leveling configuration from profile
func (storage *StorageEngine) loadWearLevelingConfig(config map[string]interface{}) {
	if enabled, ok := config["enabled"].(bool); ok {
		storage.WearLevelingState.Enabled = enabled
	}
	if overProvisioning, ok := config["over_provisioning_gb"].(float64); ok {
		storage.WearLevelingState.OverProvisioningGB = int(overProvisioning)
	}
}

// loadPowerManagementConfig loads power management configuration from profile
func (storage *StorageEngine) loadPowerManagementConfig(config map[string]interface{}) {
	if spinUpTime, ok := config["spin_up_time_ms"].(float64); ok {
		storage.PowerState.SpinUpTimeMs = spinUpTime
	}
	if spinDownTime, ok := config["spin_down_time_ms"].(float64); ok {
		storage.PowerState.SpinDownTimeMs = spinDownTime
	}
	if idleTimeout, ok := config["idle_timeout_ms"].(float64); ok {
		storage.PowerState.IdleTimeoutMs = idleTimeout
	}
}

// loadThermalConfig loads thermal configuration from profile
func (storage *StorageEngine) loadThermalConfig(config map[string]interface{}) {
	if thermalLimit, ok := config["thermal_limit_c"].(float64); ok {
		storage.ThermalState.ThermalLimitC = thermalLimit
	}
}

// initializeControllerCacheState initializes controller cache state from profile
func (storage *StorageEngine) initializeControllerCacheState() {
	// Set reasonable defaults if not configured
	if storage.ControllerCacheState.CacheSizeMB == 0 {
		// Default cache size based on storage type
		switch storage.StorageType {
		case "NVMe":
			storage.ControllerCacheState.CacheSizeMB = 1024 // 1GB for NVMe
		case "SSD":
			storage.ControllerCacheState.CacheSizeMB = 512  // 512MB for SATA SSD
		case "HDD":
			storage.ControllerCacheState.CacheSizeMB = 256  // 256MB for HDD
		default:
			storage.ControllerCacheState.CacheSizeMB = 512
		}
		storage.ControllerCacheState.Enabled = true
	}

	if storage.ControllerCacheState.HitRatio == 0.0 {
		storage.ControllerCacheState.HitRatio = 0.7 // 70% default hit ratio
	}

	if storage.ControllerCacheState.WritePolicy == "" {
		storage.ControllerCacheState.WritePolicy = "write-through"
	}
}

// initializeWearLevelingState initializes wear leveling state from profile
func (storage *StorageEngine) initializeWearLevelingState() {
	// Enable wear leveling for SSDs by default
	if storage.StorageType == "SSD" || storage.StorageType == "NVMe" {
		storage.WearLevelingState.Enabled = true
		if storage.WearLevelingState.OverProvisioningGB == 0 {
			// Default over-provisioning based on capacity
			storage.WearLevelingState.OverProvisioningGB = int(storage.CapacityGB / 10) // 10% over-provisioning
		}
	}
}

// initializeThermalState initializes thermal state from profile
func (storage *StorageEngine) initializeThermalState() {
	if storage.ThermalState.ThermalLimitC == 0.0 {
		// Default thermal limits based on storage type
		switch storage.StorageType {
		case "NVMe":
			storage.ThermalState.ThermalLimitC = 70.0 // NVMe SSDs run hotter
		case "SSD":
			storage.ThermalState.ThermalLimitC = 70.0 // SATA SSDs
		case "HDD":
			storage.ThermalState.ThermalLimitC = 60.0 // HDDs are more temperature sensitive
		default:
			storage.ThermalState.ThermalLimitC = 70.0
		}
	}
}

// initializePowerState initializes power state from profile
func (storage *StorageEngine) initializePowerState() {
	if storage.StorageType == "HDD" {
		// Set HDD-specific power state defaults
		if storage.PowerState.SpinUpTimeMs == 0.0 {
			storage.PowerState.SpinUpTimeMs = 8000.0 // 8 seconds typical HDD spin-up
		}
		if storage.PowerState.SpinDownTimeMs == 0.0 {
			storage.PowerState.SpinDownTimeMs = 5000.0 // 5 seconds spin-down
		}
		if storage.PowerState.IdleTimeoutMs == 0.0 {
			storage.PowerState.IdleTimeoutMs = 300000.0 // 5 minutes idle timeout
		}
	} else {
		// SSDs don't have spin-up/down
		storage.PowerState.SpinUpTimeMs = 0.0
		storage.PowerState.SpinDownTimeMs = 0.0
		storage.PowerState.IdleTimeoutMs = 0.0
	}
}

// initializeAccessPatternState initializes access pattern state
func (storage *StorageEngine) initializeAccessPatternState() {
	// Initialize pattern efficiency map
	storage.AccessPatternState.PatternEfficiency = map[string]float64{
		"sequential": 1.0, // Will be adjusted based on storage type
		"random":     1.0, // Will be adjusted based on storage type
		"mixed":      1.0,
	}

	// Set pattern efficiency based on storage type
	switch storage.StorageType {
	case "HDD":
		// HDDs have significant sequential advantage
		storage.AccessPatternState.PatternEfficiency["sequential"] = 0.6  // 40% faster
		storage.AccessPatternState.PatternEfficiency["random"] = 1.8      // 80% slower
	case "SSD", "NVMe":
		// SSDs have less pattern sensitivity
		storage.AccessPatternState.PatternEfficiency["sequential"] = 0.9  // 10% faster
		storage.AccessPatternState.PatternEfficiency["random"] = 1.1      // 10% slower
	}
}

// ========================================
// ADVANCED STORAGE FEATURES IMPLEMENTATION
// ========================================

// applyTrimGarbageCollectionEffects applies SSD TRIM/garbage collection effects
func (storage *StorageEngine) applyTrimGarbageCollectionEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Only applies to SSDs
	if storage.StorageType == "HDD" {
		return baseTime
	}

	// TRIM/GC operations have overhead, especially during high write activity
	var gcOverhead float64 = 1.0

	if op.Type == OpStorageWrite {
		// Write operations may trigger garbage collection
		writeRatio := storage.AccessPatternState.RandomRatio + 0.1 // Higher GC overhead for random writes

		// GC overhead increases with wear level and write activity
		gcTriggerThreshold := 0.8 - storage.WearLevelingState.WearLevel*0.3 // Lower threshold as wear increases

		if writeRatio > gcTriggerThreshold {
			// Garbage collection triggered
			gcOverhead = 1.1 + storage.WearLevelingState.WearLevel*0.2 // 10-30% overhead

			// Update wear leveling state
			storage.WearLevelingState.EraseCount += 10 // GC causes additional erases
		} else {
			// Background GC overhead
			gcOverhead = 1.02 // 2% background overhead
		}
	} else {
		// Read operations have minimal GC impact
		gcOverhead = 1.005 // 0.5% overhead
	}

	return time.Duration(float64(baseTime) * gcOverhead)
}

// applyAdvancedPrefetchingEffects applies storage controller prefetching effects
func (storage *StorageEngine) applyAdvancedPrefetchingEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Determine access pattern for prefetching effectiveness
	pattern := storage.determineAccessPattern(op)

	var prefetchBenefit float64 = 1.0

	switch pattern {
	case "sequential":
		// Sequential access benefits significantly from prefetching
		if storage.StorageType == "HDD" {
			prefetchBenefit = 0.7 // 30% faster due to read-ahead
		} else {
			prefetchBenefit = 0.85 // 15% faster for SSDs
		}

	case "random":
		// Random access has minimal prefetching benefit
		prefetchBenefit = 0.98 // 2% faster due to some pattern detection

	default:
		prefetchBenefit = 1.0
	}

	// Prefetching effectiveness decreases with high queue utilization
	queueUtilization := storage.QueueState.QueueUtilization
	if queueUtilization > 0.7 {
		// High queue pressure reduces prefetching effectiveness
		prefetchPenalty := 1.0 + (queueUtilization-0.7)*0.5 // Up to 15% penalty
		prefetchBenefit *= prefetchPenalty
	}

	return time.Duration(float64(baseTime) * prefetchBenefit)
}

// applyCompressionEffects applies data compression impact on performance
func (storage *StorageEngine) applyCompressionEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Compression effects depend on operation type and data characteristics
	var compressionFactor float64 = 1.0

	switch op.Type {
	case OpStorageWrite:
		// Write operations: compression overhead vs space savings
		if op.DataSize > int64(storage.BlockSizeBytes)*4 { // Large writes benefit more
			// Assume ~2:1 compression ratio for typical data
			compressionOverhead := 1.15 // 15% CPU overhead for compression
			bandwidthBenefit := 0.6     // 40% less data to write
			compressionFactor = compressionOverhead * bandwidthBenefit // Net ~31% faster
		} else {
			// Small writes: overhead dominates
			compressionFactor = 1.1 // 10% slower due to compression overhead
		}

	case OpStorageRead:
		// Read operations: decompression overhead vs bandwidth savings
		if op.DataSize > int64(storage.BlockSizeBytes)*4 { // Large reads
			decompressionOverhead := 1.08 // 8% CPU overhead for decompression
			bandwidthBenefit := 0.6       // 40% less data to read
			compressionFactor = decompressionOverhead * bandwidthBenefit // Net ~35% faster
		} else {
			// Small reads: minimal benefit
			compressionFactor = 1.05 // 5% slower due to decompression overhead
		}

	default:
		compressionFactor = 1.0
	}

	return time.Duration(float64(baseTime) * compressionFactor)
}

// applyEncryptionOverhead applies hardware encryption costs
func (storage *StorageEngine) applyEncryptionOverhead(baseTime time.Duration, op *Operation) time.Duration {
	// Hardware encryption has minimal overhead for modern storage controllers
	var encryptionOverhead float64 = 1.0

	switch op.Type {
	case OpStorageWrite:
		// Write operations: encryption overhead
		if storage.StorageType == "NVMe" {
			encryptionOverhead = 1.02 // 2% overhead for NVMe hardware encryption
		} else {
			encryptionOverhead = 1.05 // 5% overhead for SATA/SAS encryption
		}

	case OpStorageRead:
		// Read operations: decryption overhead
		if storage.StorageType == "NVMe" {
			encryptionOverhead = 1.015 // 1.5% overhead for NVMe hardware decryption
		} else {
			encryptionOverhead = 1.03 // 3% overhead for SATA/SAS decryption
		}

	default:
		encryptionOverhead = 1.0
	}

	// Larger operations have relatively less encryption overhead
	if op.DataSize > int64(storage.BlockSizeBytes)*16 {
		encryptionOverhead = 1.0 + (encryptionOverhead-1.0)*0.7 // 30% reduction in relative overhead
	}

	return time.Duration(float64(baseTime) * encryptionOverhead)
}

// applyMultiStreamIOEffects applies multi-stream SSD optimization effects
func (storage *StorageEngine) applyMultiStreamIOEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Only applies to modern SSDs with multi-stream support
	if storage.StorageType == "HDD" {
		return baseTime
	}

	// Multi-stream I/O reduces write amplification and improves performance
	var multiStreamBenefit float64 = 1.0

	if op.Type == OpStorageWrite {
		// Write operations benefit from stream separation
		queueUtilization := storage.QueueState.QueueUtilization

		if queueUtilization > 0.3 { // Multi-stream benefits appear under load
			// Stream separation reduces write amplification
			multiStreamBenefit = 0.9 - queueUtilization*0.1 // 10-20% improvement under load

			// Additional benefit for large writes (better stream utilization)
			if op.DataSize > int64(storage.BlockSizeBytes)*8 {
				multiStreamBenefit *= 0.95 // Additional 5% benefit
			}
		}
	} else {
		// Read operations have minimal multi-stream benefit
		multiStreamBenefit = 0.99 // 1% improvement due to better organization
	}

	return time.Duration(float64(baseTime) * multiStreamBenefit)
}

// applyZonedStorageEffects applies ZNS (Zoned Namespace) SSD effects
func (storage *StorageEngine) applyZonedStorageEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Only applies to ZNS SSDs (advanced NVMe feature)
	if storage.StorageType != "NVMe" {
		return baseTime
	}

	// ZNS SSDs require zone-aware operations
	var zonedBenefit float64 = 1.0

	switch op.Type {
	case OpStorageWrite:
		// Sequential writes within zones are highly optimized
		pattern := storage.determineAccessPattern(op)

		if pattern == "sequential" {
			// Sequential writes align with zone structure
			zonedBenefit = 0.8 // 20% faster due to zone optimization
		} else {
			// Random writes may require zone management overhead
			zonedBenefit = 1.15 // 15% slower due to zone allocation/management
		}

	case OpStorageRead:
		// Reads benefit from zone organization
		zonedBenefit = 0.95 // 5% faster due to better data locality

	default:
		zonedBenefit = 1.0
	}

	// Zone utilization affects performance
	utilization := storage.calculateCurrentUtilization()
	if utilization > 0.8 {
		// High utilization may require zone compaction
		zonedBenefit *= 1.1 // 10% penalty for zone management under high utilization
	}

	return time.Duration(float64(baseTime) * zonedBenefit)
}

// applyErrorCorrectionEffects applies ECC and bad block management effects
func (storage *StorageEngine) applyErrorCorrectionEffects(baseTime time.Duration, op *Operation) time.Duration {
	// Error correction overhead depends on storage type and health
	var eccOverhead float64 = 1.0

	// Base ECC overhead
	switch storage.StorageType {
	case "HDD":
		eccOverhead = 1.01 // 1% overhead for HDD ECC
	case "SSD":
		eccOverhead = 1.005 // 0.5% overhead for SSD ECC
	case "NVMe":
		eccOverhead = 1.002 // 0.2% overhead for NVMe hardware ECC
	}

	// Additional overhead based on storage health
	healthFactor := storage.Health.Score
	if healthFactor < 0.9 {
		// Degraded storage requires more error correction
		healthPenalty := 1.0 + (0.9-healthFactor)*0.5 // Up to 5% additional overhead
		eccOverhead *= healthPenalty
	}

	// Wear level affects error correction requirements (SSDs)
	if storage.StorageType != "HDD" {
		wearLevel := storage.WearLevelingState.WearLevel
		if wearLevel > 0.7 {
			// High wear increases error correction overhead
			wearPenalty := 1.0 + (wearLevel-0.7)*0.2 // Up to 6% additional overhead
			eccOverhead *= wearPenalty
		}
	}

	// Large operations have relatively less ECC overhead
	if op.DataSize > int64(storage.BlockSizeBytes)*32 {
		eccOverhead = 1.0 + (eccOverhead-1.0)*0.8 // 20% reduction in relative overhead
	}

	return time.Duration(float64(baseTime) * eccOverhead)
}
