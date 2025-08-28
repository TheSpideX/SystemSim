package components

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// ComponentType represents the type of component
type ComponentType string

const (
	ComponentTypeDatabase    ComponentType = "database"
	ComponentTypeWebServer   ComponentType = "web_server"
	ComponentTypeCache       ComponentType = "cache"
	ComponentTypeLoadBalancer ComponentType = "load_balancer"
	ComponentTypeCPU         ComponentType = "cpu"
	ComponentTypeMemory      ComponentType = "memory"
	ComponentTypeStorage     ComponentType = "storage"
	ComponentTypeNetwork     ComponentType = "network"
	ComponentTypeCustom      ComponentType = "custom"
)

// LoadBalancingAlgorithm represents different load balancing algorithms
type LoadBalancingAlgorithm string

const (
	LoadBalancingNone            LoadBalancingAlgorithm = "none"             // No load balancing (single instance)
	LoadBalancingRoundRobin      LoadBalancingAlgorithm = "round_robin"      // Simple round-robin
	LoadBalancingLeastConnections LoadBalancingAlgorithm = "least_connections" // Route to least loaded instance
	LoadBalancingWeighted        LoadBalancingAlgorithm = "weighted"         // Weighted round-robin
	LoadBalancingHealthAware     LoadBalancingAlgorithm = "health_aware"     // Health-based routing
)

// LoadBalancingConfig represents load balancer configuration
type LoadBalancingConfig struct {
	Algorithm        LoadBalancingAlgorithm `json:"algorithm"`
	MinInstances     int                    `json:"min_instances"`
	MaxInstances     int                    `json:"max_instances"`
	AlgorithmPenalty time.Duration          `json:"algorithm_penalty"` // Time penalty for algorithm complexity

	// Auto-scaling configuration
	AutoScaling      bool                   `json:"auto_scaling"`
	ScaleUpThreshold float64               `json:"scale_up_threshold"`   // Buffer utilization threshold (0.0-1.0)
	ScaleDownThreshold float64             `json:"scale_down_threshold"` // Buffer utilization threshold (0.0-1.0)
	ScaleUpCooldown  time.Duration         `json:"scale_up_cooldown"`    // Minimum time between scale-up operations
	ScaleDownCooldown time.Duration        `json:"scale_down_cooldown"`  // Minimum time between scale-down operations

	// Weighted load balancing configuration
	InstanceWeights  map[string]int         `json:"instance_weights"`  // instanceID -> weight
	DefaultWeight    int                    `json:"default_weight"`    // Default weight for new instances
}

// LoadBalancer manages multiple instances of a component with load balancing
type LoadBalancer struct {
	// Component identification
	ComponentID     string                 `json:"component_id"`
	ComponentType   ComponentType          `json:"component_type"`
	Config          *LoadBalancingConfig   `json:"config"`
	ComponentConfig *ComponentConfig       `json:"component_config"` // Full component configuration

	// Instance management
	Instances     []*ComponentInstance   `json:"instances"`
	NextInstanceID int                   `json:"next_instance_id"`

	// Atomic coordination flags per instance
	InstanceReady    map[string]*atomic.Bool `json:"-"` // instanceID -> ready flag
	InstanceShutdown map[string]*atomic.Bool `json:"-"` // instanceID -> shutdown flag
	InstanceHealth   map[string]float64      `json:"instance_health"`

	// Load balancing state
	RoundRobinIndex  int                    `json:"round_robin_index"`
	LastScaleUp      time.Time              `json:"last_scale_up"`
	LastScaleDown    time.Time              `json:"last_scale_down"`

	// Weighted load balancing state
	WeightedSelections map[string]int       `json:"weighted_selections"` // instanceID -> current selections count
	TotalWeight        int                  `json:"total_weight"`        // Sum of all instance weights

	// Communication channels
	InputChannel     chan *engines.Operation     `json:"-"` // Receives operations for the component
	OutputChannel    chan *engines.OperationResult `json:"-"` // Aggregated output from all instances

	// Global registry integration
	GlobalRegistry   GlobalRegistryInterface     `json:"-"`

	// Metrics and health
	Metrics          *ComponentMetrics           `json:"metrics"`
	Health           *ComponentHealth            `json:"health"`

	// Lifecycle management
	ctx              context.Context             `json:"-"`
	cancel           context.CancelFunc          `json:"-"`
	mutex            sync.RWMutex                `json:"-"`
	running          bool                        `json:"-"`
}

// ComponentInstance represents a single instance within a component (managed by LoadBalancer)
// This replaces the old ComponentInstance which is now the LoadBalancer
type ComponentInstance struct {
	// Identity
	ID                string                 `json:"id"`
	ComponentID       string                 `json:"component_id"`
	ComponentType     ComponentType          `json:"component_type"`
	Config            *ComponentConfig       `json:"config"`

	// Atomic coordination flags
	ReadyFlag         *atomic.Bool           `json:"-"` // Ready to receive operations
	ShutdownFlag      *atomic.Bool           `json:"-"` // Should shutdown gracefully
	ProcessingFlag    *atomic.Bool           `json:"-"` // Currently processing operations

	// Independent engine goroutines
	InputNetworkEngine  *engines.EngineWrapper `json:"input_network_engine"`
	CPUEngine          *engines.EngineWrapper `json:"cpu_engine"`
	MemoryEngine       *engines.EngineWrapper `json:"memory_engine"`
	StorageEngine      *engines.EngineWrapper `json:"storage_engine"`
	OutputNetworkEngine *engines.EngineWrapper `json:"output_network_engine"`

	// Engine management
	Engines           map[engines.EngineType]*engines.EngineWrapper `json:"engines"`
	EngineGoroutines  map[engines.EngineType]GoroutineTracker       `json:"engine_goroutines"`

	// Engine router for dynamic routing decisions
	EngineRouter      *EngineRouter          `json:"-"`

	// Decision graph for intra-instance routing
	DecisionGraph     *DecisionGraph         `json:"decision_graph"`

	// Centralized output manager
	CentralizedOutput *CentralizedOutputManager `json:"-"`

	// Communication channels
	InputChannel      chan *engines.Operation     `json:"-"` // Input operations for this instance
	OutputChannel     chan *engines.OperationResult `json:"-"` // Output results from this instance
	InternalChannels  map[engines.EngineType]chan *engines.Operation `json:"-"` // Engine-to-engine communication

	// Health and monitoring
	Health            *ComponentHealth       `json:"health"`
	Metrics           *ComponentMetrics      `json:"metrics"`
	StartTime         time.Time              `json:"start_time"`
	LastTickTime      time.Time              `json:"last_tick_time"`

	// Error handling
	ErrorHandler      *ErrorHandler          `json:"-"`

	// Lifecycle management
	ctx               context.Context        `json:"-"`
	cancel            context.CancelFunc     `json:"-"`
	mutex             sync.RWMutex           `json:"-"`
	running           bool                   `json:"-"`
}

// CentralizedOutputManager handles inter-component routing for each instance
type CentralizedOutputManager struct {
	// Identity
	InstanceID        string                 `json:"instance_id"`
	ComponentID       string                 `json:"component_id"`

	// Communication channels
	InputChannel      chan *engines.OperationResult `json:"-"` // Receives results from last engines
	OutputChannel     chan *engines.OperationResult `json:"-"` // Sends to next components

	// Global registry integration
	GlobalRegistry    GlobalRegistryInterface        `json:"-"`

	// User flow configuration
	UserFlowConfig    *UserFlowConfig               `json:"-"`

	// Routing configuration
	RoutingRules      map[string]string             `json:"routing_rules"` // operation_type -> next_component
	DefaultRouting    string                        `json:"default_routing"`

	// Backpressure management
	BackpressureConfig *BackpressureConfig          `json:"backpressure_config"`

	// Circuit breaker management
	CircuitBreakerManager *CircuitBreakerManager    `json:"-"`

	// Fallback routing configuration
	FallbackRouting       *FallbackRoutingConfig    `json:"fallback_routing"`

	// Lifecycle management
	ctx               context.Context               `json:"-"`
	cancel            context.CancelFunc            `json:"-"`
	mutex             sync.RWMutex                  `json:"-"`
	running           bool                          `json:"-"`
}

// UserFlowConfig defines system-level routing between components
type UserFlowConfig struct {
	Flows             map[string]*UserFlow          `json:"flows"` // flow_name -> flow definition
}

// UserFlow represents a sequence of component operations
type UserFlow struct {
	Name              string                        `json:"name"`
	Description       string                        `json:"description"`
	Steps             []*UserFlowStep               `json:"steps"`
}

// UserFlowStep represents a single step in a user flow
type UserFlowStep struct {
	ComponentID       string                        `json:"component_id"`
	Operation         string                        `json:"operation"`
	Conditions        map[string]string             `json:"conditions"` // condition -> next_component
}

// BackpressureConfig defines backpressure handling policies
type BackpressureConfig struct {
	MaxRetries        int                           `json:"max_retries"`
	RetryDelay        time.Duration                 `json:"retry_delay"`
	CircuitBreakerThreshold int                     `json:"circuit_breaker_threshold"`
	HealthCheckInterval time.Duration               `json:"health_check_interval"`
}

// FallbackRoutingConfig defines fallback routing behavior when primary targets fail
type FallbackRoutingConfig struct {
	Enabled                bool                           `json:"enabled"`
	FallbackTargets        map[string][]string           `json:"fallback_targets"`        // primary -> [fallback1, fallback2, ...]
	OperationTypeFallbacks map[string][]string           `json:"operation_type_fallbacks"` // operation_type -> [fallback1, fallback2, ...]
	ConditionFallbacks     map[string][]string           `json:"condition_fallbacks"`     // condition -> [fallback1, fallback2, ...]
	MaxFallbackAttempts    int                           `json:"max_fallback_attempts"`
	FallbackDelay          time.Duration                 `json:"fallback_delay"`
	FallbackStrategy       FallbackStrategy              `json:"fallback_strategy"`
}

// FallbackStrategy defines how fallback targets are selected
type FallbackStrategy string

const (
	FallbackStrategySequential FallbackStrategy = "sequential" // Try fallbacks in order
	FallbackStrategyRoundRobin FallbackStrategy = "round_robin" // Round-robin through fallbacks
	FallbackStrategyHealthBased FallbackStrategy = "health_based" // Choose healthiest fallback
	FallbackStrategyLoadBased   FallbackStrategy = "load_based"   // Choose least loaded fallback
)

// ComponentState represents the current state of a component
type ComponentState string

const (
	ComponentStateStarting ComponentState = "starting"
	ComponentStateRunning  ComponentState = "running"
	ComponentStatePaused   ComponentState = "paused"
	ComponentStateStopping ComponentState = "stopping"
	ComponentStateStopped  ComponentState = "stopped"
	ComponentStateError    ComponentState = "error"
)

// ComponentHealth represents the health status of a component
type ComponentHealth struct {
	Status            string                 `json:"status"`             // GREEN, YELLOW, RED, CRITICAL
	IsAcceptingLoad   bool                   `json:"is_accepting_load"`  // Whether component can accept new operations
	CurrentCPU        float64                `json:"current_cpu"`        // CPU utilization (0.0-1.0)
	CurrentMemory     float64                `json:"current_memory"`     // Memory utilization (0.0-1.0)
	AvailableCapacity float64                `json:"available_capacity"` // Available processing capacity
	CurrentLatency    time.Duration          `json:"current_latency"`    // Average operation latency
	LastHealthCheck   time.Time              `json:"last_health_check"`  // When health was last checked
	EngineHealth      map[string]interface{} `json:"engine_health"`      // Health of individual engines
}

// ComponentMetrics represents metrics collected from a component
type ComponentMetrics struct {
	ComponentID       string                 `json:"component_id"`
	ComponentType     ComponentType          `json:"component_type"`
	State             ComponentState         `json:"state"`
	Uptime            time.Duration          `json:"uptime"`
	TotalOperations   int64                  `json:"total_operations"`
	CompletedOps      int64                  `json:"completed_operations"`
	FailedOps         int64                  `json:"failed_operations"`
	AverageLatency    time.Duration          `json:"average_latency"`
	CurrentUtilization float64               `json:"current_utilization"`
	EngineMetrics     map[string]interface{} `json:"engine_metrics"`
	LastUpdated       time.Time              `json:"last_updated"`
}

// Component interface defines the contract for all components
type Component interface {
	// Lifecycle management
	Start(ctx context.Context) error
	Stop() error
	Pause() error
	Resume() error

	// Component identification
	GetID() string
	GetType() ComponentType
	GetState() ComponentState

	// Health and monitoring
	IsHealthy() bool
	GetHealth() *ComponentHealth
	GetMetrics() *ComponentMetrics

	// Operation processing
	ProcessOperation(op *engines.Operation) error
	ProcessTick(currentTick int64) error

	// State management
	SaveState() error
	LoadState(componentID string) error

	// Engine management
	GetEngines() map[engines.EngineType]*engines.EngineWrapper
	GetEngine(engineType engines.EngineType) *engines.EngineWrapper

	// Communication channels
	GetInputChannel() chan *engines.Operation
	GetOutputChannel() chan *engines.OperationResult
	GetTickChannel() chan int64
}

// ComponentConfig represents the configuration for a component
type ComponentConfig struct {
	ID                string                            `json:"id"`
	Type              ComponentType                     `json:"type"`
	Name              string                            `json:"name"`
	Description       string                            `json:"description"`

	// Load balancer configuration
	LoadBalancer      *LoadBalancingConfig              `json:"load_balancer"`

	// Engine configuration
	RequiredEngines   []engines.EngineType              `json:"required_engines"`
	EngineProfiles    map[engines.EngineType]string     `json:"engine_profiles"`
	ComplexityLevels  map[engines.EngineType]int        `json:"complexity_levels"`

	// Decision graph configuration
	DecisionGraph     *DecisionGraphConfig              `json:"decision_graph"`

	// Performance configuration
	MaxConcurrentOps  int                               `json:"max_concurrent_ops"`
	QueueCapacity     int                               `json:"queue_capacity"`
	TickTimeout       time.Duration                     `json:"tick_timeout"`

	// User flow and routing configuration
	UserFlow          *UserFlowConfig                   `json:"user_flow"`
	RoutingRules      map[string]string                 `json:"routing_rules"` // operation_type -> next_component
}

// DecisionGraphConfig represents the configuration for a decision graph
type DecisionGraphConfig struct {
	StartNode string                    `json:"start_node"`
	EndNodes  []string                  `json:"end_nodes"`
	Nodes     map[string]*DecisionNode  `json:"nodes"`
}

// DecisionNode represents a node in the decision graph
type DecisionNode struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`       // "engine", "decision", "end"
	EngineType engines.EngineType `json:"engine_type,omitempty"` // For engine nodes
	Conditions map[string]string `json:"conditions"` // For decision nodes
}

// GoroutineTracker tracks information about engine goroutines
type GoroutineTracker struct {
	GoroutineID         string    `json:"goroutine_id"`
	StartTime           time.Time `json:"start_time"`
	LastActivity        time.Time `json:"last_activity"`
	OperationsProcessed int64     `json:"operations_processed"`
	CurrentLoad         float64   `json:"current_load"`
	TicksProcessed      int64     `json:"ticks_processed"`
}

// EngineChannels represents the channels for an engine
type EngineChannels struct {
	Input  chan *engines.Operation      `json:"-"`
	Output chan *engines.OperationResult `json:"-"`
	Tick   chan int64                    `json:"-"`
	Stop   chan struct{}                 `json:"-"`
}

// OLD ComponentInstance definition removed - now replaced by LoadBalancer + ComponentInstance architecture

// BufferStatus represents the buffer utilization status for registry integration
type BufferStatus int

const (
	BufferStatusNormal    BufferStatus = iota // 0-20% buffer full
	BufferStatusWarning                       // 20-40% buffer full
	BufferStatusHigh                          // 40-60% buffer full
	BufferStatusOverflow                      // 60-80% buffer full
	BufferStatusCritical                      // 80-90% buffer full
	BufferStatusEmergency                     // 90%+ buffer full
)

// String returns the string representation of BufferStatus
func (bs BufferStatus) String() string {
	switch bs {
	case BufferStatusNormal:
		return "normal"
	case BufferStatusWarning:
		return "warning"
	case BufferStatusHigh:
		return "high"
	case BufferStatusOverflow:
		return "overflow"
	case BufferStatusCritical:
		return "critical"
	case BufferStatusEmergency:
		return "emergency"
	default:
		return "unknown"
	}
}

// GlobalRegistryInterface defines the interface for global registry integration
type GlobalRegistryInterface interface {
	// Component registration
	Register(componentID string, inputChannel chan *engines.Operation)
	Unregister(componentID string)

	// Component discovery
	GetChannel(componentID string) chan *engines.Operation
	GetAllComponents() map[string]chan *engines.Operation

	// Health and load monitoring
	GetHealth(componentID string) float64
	UpdateHealth(componentID string, health float64)
	GetLoad(componentID string) BufferStatus
	UpdateLoad(componentID string, status BufferStatus)

	// Registry lifecycle
	Start() error
	Stop() error
}
