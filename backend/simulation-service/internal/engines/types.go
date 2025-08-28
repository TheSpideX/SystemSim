package engines

import (
	"time"
)

// EngineType represents the type of engine
type EngineType int

const (
	CPUEngineType EngineType = iota
	MemoryEngineType
	StorageEngineType
	NetworkEngineType
)

func (et EngineType) String() string {
	switch et {
	case CPUEngineType:
		return "CPU"
	case MemoryEngineType:
		return "Memory"
	case StorageEngineType:
		return "Storage"
	case NetworkEngineType:
		return "Network"
	default:
		return "Unknown"
	}
}

// BaseEngine interface that all engines must implement
type BaseEngine interface {
	// Core processing methods
	ProcessOperation(op *Operation, currentTick int64) *OperationResult
	ProcessTick(currentTick int64) []OperationResult

	// Queue management
	QueueOperation(op *Operation) error
	GetQueueLength() int
	GetQueueCapacity() int

	// Health and metrics
	GetHealth() *HealthMetrics
	UpdateHealth()
	GetUtilization() float64

	// Engine identification
	GetEngineType() EngineType
	GetEngineID() string

	// Configuration and profiles
	SetTickDuration(duration time.Duration)
	GetTickDuration() time.Duration
	LoadProfile(profile *EngineProfile) error
	GetProfile() *EngineProfile

	// Complexity management (integer-based, universal)
	SetComplexityLevel(level int) error  // 0=Minimal, 1=Basic, 2=Advanced, 3=Maximum
	GetComplexityLevel() int             // Returns current complexity level

	// Dynamic behavior methods
	GetDynamicState() *DynamicState
	UpdateDynamicBehavior()
	GetConvergenceMetrics() *ConvergenceMetrics

	// State management
	Reset()
	GetCurrentState() map[string]interface{}
}

// Operation represents a single operation to be processed by an engine
type Operation struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	DataSize      int64                  `json:"data_size"`
	Complexity    string                 `json:"complexity"`
	Language      string                 `json:"language"`
	Priority      int                    `json:"priority"`
	Metadata      map[string]interface{} `json:"metadata"`
	StartTick     int64                  `json:"start_tick"`
	Deadline      int64                  `json:"deadline"`
	NextComponent string                 `json:"next_component"` // For routing to next engine
}

// OperationResult represents the result of processing an operation
type OperationResult struct {
	OperationID    string                 `json:"operation_id"`
	OperationType  string                 `json:"operation_type"`  // For routing decisions
	ProcessingTime time.Duration          `json:"processing_time"`
	CompletedTick  int64                  `json:"completed_tick"`
	CompletedAt    int64                  `json:"completed_at"`    // Tick when completed
	Success        bool                   `json:"success"`
	ErrorMessage   string                 `json:"error_message"`
	NextComponent  string                 `json:"next_component"`  // Where to route next

	// Performance penalty information for routing decisions
	PenaltyInfo    *PenaltyInformation    `json:"penalty_info,omitempty"`

	// Detailed metrics (legacy field, use PenaltyInfo for routing decisions)
	Metrics        map[string]interface{} `json:"metrics"`
}

// PenaltyInformation contains structured penalty data for performance-aware routing
type PenaltyInformation struct {
	// Engine identification
	EngineType     EngineType `json:"engine_type"`
	EngineID       string     `json:"engine_id"`

	// Base performance metrics
	BaseProcessingTime time.Duration `json:"base_processing_time"` // Theoretical minimum time
	ActualProcessingTime time.Duration `json:"actual_processing_time"` // Time actually taken

	// Penalty factors (1.0 = no penalty, >1.0 = penalty applied)
	LoadPenalty        float64 `json:"load_penalty"`        // Due to engine utilization
	QueuePenalty       float64 `json:"queue_penalty"`       // Due to queue backlog
	ThermalPenalty     float64 `json:"thermal_penalty"`     // Due to thermal throttling
	ContentionPenalty  float64 `json:"contention_penalty"`  // Due to resource contention
	HealthPenalty      float64 `json:"health_penalty"`      // Due to engine health issues

	// Engine-specific penalties
	CPUPenalties       *CPUPenaltyDetails       `json:"cpu_penalties,omitempty"`
	MemoryPenalties    *MemoryPenaltyDetails    `json:"memory_penalties,omitempty"`
	StoragePenalties   *StoragePenaltyDetails   `json:"storage_penalties,omitempty"`
	NetworkPenalties   *NetworkPenaltyDetails   `json:"network_penalties,omitempty"`

	// Overall performance assessment
	TotalPenaltyFactor float64 `json:"total_penalty_factor"` // Combined penalty multiplier
	PerformanceGrade   string  `json:"performance_grade"`    // A, B, C, D, F
	RecommendedAction  string  `json:"recommended_action"`   // "continue", "throttle", "redirect"
}

// CPUPenaltyDetails contains CPU-specific penalty information
type CPUPenaltyDetails struct {
	CacheHitRatio      float64 `json:"cache_hit_ratio"`      // L1/L2/L3 cache performance
	VectorizationRatio float64 `json:"vectorization_ratio"`  // SIMD utilization
	ThermalThrottling  float64 `json:"thermal_throttling"`   // Thermal throttling factor
	CoreUtilization    float64 `json:"core_utilization"`     // CPU core usage
	MemoryContention   float64 `json:"memory_contention"`    // Memory bandwidth contention
}

// MemoryPenaltyDetails contains memory-specific penalty information
type MemoryPenaltyDetails struct {
	BandwidthUtilization float64 `json:"bandwidth_utilization"` // Memory bandwidth usage
	NUMAPenalty         float64 `json:"numa_penalty"`          // NUMA cross-socket penalty
	RowBufferHitRate    float64 `json:"row_buffer_hit_rate"`   // DRAM row buffer efficiency
	MemoryPressure      float64 `json:"memory_pressure"`       // Memory pressure factor
	ChannelContention   float64 `json:"channel_contention"`    // Memory channel contention
}

// StoragePenaltyDetails contains storage-specific penalty information
type StoragePenaltyDetails struct {
	IOPSUtilization    float64 `json:"iops_utilization"`     // IOPS usage percentage
	QueueDepth         float64 `json:"queue_depth"`          // Storage queue utilization
	AccessPattern      string  `json:"access_pattern"`       // "sequential", "random"
	ThermalThrottling  float64 `json:"thermal_throttling"`   // Storage thermal throttling
	PowerStateImpact   float64 `json:"power_state_impact"`   // Power management impact
}

// NetworkPenaltyDetails contains network-specific penalty information
type NetworkPenaltyDetails struct {
	BandwidthUtilization float64 `json:"bandwidth_utilization"` // Network bandwidth usage
	CongestionFactor     float64 `json:"congestion_factor"`     // Network congestion
	PacketLossRate       float64 `json:"packet_loss_rate"`      // Packet loss percentage
	LatencyPenalty       float64 `json:"latency_penalty"`       // Geographic/routing latency
	ProtocolEfficiency   float64 `json:"protocol_efficiency"`   // Protocol overhead impact
}

// QueuedOperation represents an operation in the queue
type QueuedOperation struct {
	Operation *Operation `json:"operation"`
	QueuedAt  int64      `json:"queued_at"`
}

// HealthMetrics represents the health state of an engine
type HealthMetrics struct {
	Score              float64 `json:"score"`              // 0.0 to 1.0
	Utilization        float64 `json:"utilization"`        // 0.0 to 1.0
	QueueUtilization   float64 `json:"queue_utilization"`  // 0.0 to 1.0
	ErrorRate          float64 `json:"error_rate"`         // 0.0 to 1.0
	AverageLatency     float64 `json:"average_latency"`    // milliseconds
	ThroughputOps      float64 `json:"throughput_ops"`     // operations per second
	LastUpdated        int64   `json:"last_updated"`       // tick
}

// DynamicState represents the current dynamic state of an engine
type DynamicState struct {
	CurrentUtilization   float64                `json:"current_utilization"`
	PerformanceFactor    float64                `json:"performance_factor"`
	ConvergenceProgress  float64                `json:"convergence_progress"`
	HardwareSpecific     map[string]interface{} `json:"hardware_specific"`
	LastUpdated          int64                  `json:"last_updated"`
}

// ConvergenceMetrics tracks statistical convergence progress
type ConvergenceMetrics struct {
	OperationCount       int64   `json:"operation_count"`
	ConvergencePoint     float64 `json:"convergence_point"`
	CurrentVariance      float64 `json:"current_variance"`
	IsConverged          bool    `json:"is_converged"`
	TimeToConvergence    int64   `json:"time_to_convergence"`
	ConvergenceFactors   map[string]float64 `json:"convergence_factors"`
}

// EngineProfile represents the configuration profile for an engine
type EngineProfile struct {
	Name                string                 `json:"name"`
	Type                EngineType             `json:"type"`
	Description         string                 `json:"description"`
	Version             string                 `json:"version"`
	BaselinePerformance map[string]float64     `json:"baseline_performance"`
	TechnologySpecs     map[string]interface{} `json:"technology_specs"`
	LoadCurves          map[string]interface{} `json:"load_curves"`
	EngineSpecific      map[string]interface{} `json:"engine_specific"`
}

// LoadDegradationCurve represents performance degradation under load
type LoadDegradationCurve struct {
	OptimalThreshold  float64 `json:"optimal_threshold"`   // 0.0 to 1.0
	WarningThreshold  float64 `json:"warning_threshold"`   // 0.0 to 1.0
	CriticalThreshold float64 `json:"critical_threshold"`  // 0.0 to 1.0
	OptimalFactor     float64 `json:"optimal_factor"`      // 1.0 = no degradation
	WarningFactor     float64 `json:"warning_factor"`      // > 1.0 = degradation
	CriticalFactor    float64 `json:"critical_factor"`     // >> 1.0 = severe degradation
}

// PerformanceVariance represents realistic performance variance
type PerformanceVariance struct {
	BaseVariance    float64 `json:"base_variance"`
	LoadMultiplier  float64 `json:"load_multiplier"`
	ScaleReduction  float64 `json:"scale_reduction"`
}

// StatisticalModel represents a convergence model for a specific behavior
type StatisticalModel struct {
	Name             string  `json:"name"`
	ConvergencePoint float64 `json:"convergence_point"`
	BaseVariance     float64 `json:"base_variance"`
	MinOperations    int64   `json:"min_operations"`
	CurrentValue     float64 `json:"current_value"`
	IsConverged      bool    `json:"is_converged"`
}

// ConvergenceState tracks the overall convergence state of an engine
type ConvergenceState struct {
	Models          map[string]*StatisticalModel `json:"models"`
	OperationCount  int64                        `json:"operation_count"`
	DataProcessed   int64                        `json:"data_processed"`
	StartTick       int64                        `json:"start_tick"`
	ConvergedTick   int64                        `json:"converged_tick"`
}

// Constants for common operation types
const (
	// CPU Operations
	OpCPUCompute    = "cpu_compute"
	OpCPUAlgorithm  = "cpu_algorithm"
	OpCPUBranch     = "cpu_branch"
	
	// Memory Operations
	OpMemoryRead    = "memory_read"
	OpMemoryWrite   = "memory_write"
	OpMemoryAlloc   = "memory_alloc"
	OpMemoryFree    = "memory_free"
	
	// Storage Operations
	OpStorageRead   = "storage_read"
	OpStorageWrite  = "storage_write"
	OpStorageSeek   = "storage_seek"
	OpStorageSync   = "storage_sync"
	OpStorageTrim   = "storage_trim"
	
	// Network Operations
	OpNetworkSend   = "network_send"
	OpNetworkRecv   = "network_recv"
	OpNetworkConn   = "network_connect"
)

// Constants for complexity levels
const (
	ComplexityO1    = "O(1)"
	ComplexityOLogN = "O(log n)"
	ComplexityON    = "O(n)"
	ComplexityON2   = "O(n²)"
	ComplexityONLogN = "O(n log n)"
)

// Constants for programming languages
const (
	LangGo     = "go"
	LangPython = "python"
	LangJava   = "java"
	LangCPP    = "cpp"
	LangRust   = "rust"
	LangJS     = "javascript"
)
