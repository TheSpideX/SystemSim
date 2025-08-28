package components

import (
	"github.com/systemsim/simulation-service/internal/engines"
)

// ComponentLoadBalancerInterface defines the interface for component load balancers
type ComponentLoadBalancerInterface interface {
	// Graph access for Engine Output Queues
	GetComponentGraph() *DecisionGraph
	
	// Instance management
	GetInstances() map[string]*ComponentInstance
	SelectInstance(*engines.Operation) (*ComponentInstance, error)
	
	// Health and metrics
	GetHealth() float64
	GetMetrics() *LoadBalancerMetrics
	
	// Configuration
	GetComponentID() string
	GetComponentType() ComponentType
}

// GlobalRegistryInterface defines the interface for global registry
type GlobalRegistryInterface interface {
	// Component registration and discovery
	RegisterComponent(componentID string, channel chan *engines.Operation) error
	UnregisterComponent(componentID string) error
	GetChannel(componentID string) chan *engines.Operation
	
	// System graph access
	GetSystemGraph(flowID string) *DecisionGraph
	UpdateSystemGraph(flowID string, graph *DecisionGraph) error
	
	// Request context management
	GetRequestContext(requestID string) *RequestContext
	UpdateRequestContext(requestID string, context *RequestContext) error
	
	// Health and discovery
	GetComponentHealth(componentID string) float64
	ListComponents() []string
}

// DecisionGraph represents a routing decision graph (static data structure)
type DecisionGraph struct {
	Name      string                  `json:"name"`
	StartNode string                  `json:"start_node"`
	Nodes     map[string]*DecisionNode `json:"nodes"`
	Level     GraphLevel              `json:"level"`
}

// DecisionNode represents a node in the decision graph
type DecisionNode struct {
	// Basic routing
	Target     string            `json:"target"`      // Engine or component target
	Operation  string            `json:"operation"`   // Operation to perform
	Next       string            `json:"next"`        // Default next node
	Conditions map[string]string `json:"conditions"`  // condition -> destination mapping
	
	// Advanced routing types
	RoutingType        string              `json:"routing_type"`         // "standard", "probability_based", "dynamic_state_based"
	ProbabilityConfig  *ProbabilityConfig  `json:"probability_config"`   // For probability-based routing
	StateConfig        *StateConfig        `json:"state_config"`         // For state-based routing
}

// ProbabilityConfig defines probability-based routing configuration
type ProbabilityConfig struct {
	CacheHitRate  float64           `json:"cache_hit_rate"`   // For cache operations
	SuccessRate   float64           `json:"success_rate"`     // For general operations
	Conditions    map[string]string `json:"conditions"`       // condition -> destination mapping
}

// StateConfig defines state-based routing configuration
type StateConfig struct {
	StateChecks map[string]string `json:"state_checks"`     // state_condition -> destination mapping
	Fallback    string            `json:"fallback"`         // Fallback destination
}

// GraphLevel indicates the level of the decision graph
type GraphLevel string

const (
	ComponentLevel GraphLevel = "component"
	SystemLevel    GraphLevel = "system"
)

// RequestContext holds context information for a request
type RequestContext struct {
	RequestID         string `json:"request_id"`
	SystemFlowID      string `json:"system_flow_id"`
	CurrentSystemNode string `json:"current_system_node"`
	StartTime         string `json:"start_time"`
	LastUpdate        string `json:"last_update"`
}

// ComponentInstance represents a component instance
type ComponentInstance struct {
	ID          string                        `json:"id"`
	ComponentID string                        `json:"component_id"`
	Health      float64                       `json:"health"`
	Engines     map[engines.EngineType]*Engine `json:"engines"`
	
	// Communication
	InputChannel  chan *engines.Operation `json:"-"`
	OutputChannel chan *engines.Operation `json:"-"`
	
	// Metrics
	ActiveConnections int     `json:"active_connections"`
	MaxConnections    int     `json:"max_connections"`
	CurrentLoad       float64 `json:"current_load"`
}

// Engine represents an engine within a component instance
type Engine struct {
	ID     string             `json:"id"`
	Type   engines.EngineType `json:"type"`
	Health float64            `json:"health"`
	
	// Processing
	InputQueue  chan *engines.Operation `json:"-"`
	OutputQueue chan *engines.Operation `json:"-"`
	
	// Metrics
	CurrentLoad float64 `json:"current_load"`
	Utilization float64 `json:"utilization"`
}

// ComponentType represents the type of component
type ComponentType string

const (
	WebServerComponent    ComponentType = "web_server"
	DatabaseComponent     ComponentType = "database"
	CacheComponent        ComponentType = "cache"
	LoadBalancerComponent ComponentType = "load_balancer"
	APIGatewayComponent   ComponentType = "api_gateway"
	FileServerComponent   ComponentType = "file_server"
	CustomComponent       ComponentType = "custom"
)

// LoadBalancerMetrics tracks load balancer performance
type LoadBalancerMetrics struct {
	TotalRequests       int64   `json:"total_requests"`
	SuccessfulRequests  int64   `json:"successful_requests"`
	FailedRequests      int64   `json:"failed_requests"`
	AverageResponseTime float64 `json:"average_response_time"`
	
	// Instance metrics
	InstanceCount       int     `json:"instance_count"`
	HealthyInstances    int     `json:"healthy_instances"`
	UnhealthyInstances  int     `json:"unhealthy_instances"`
	
	// Load balancing metrics
	RoundRobinIndex     int     `json:"round_robin_index"`
	LastScaleUp         string  `json:"last_scale_up"`
	LastScaleDown       string  `json:"last_scale_down"`
}

// AutoScalingConfig defines auto-scaling configuration
type AutoScalingConfig struct {
	Enabled            bool                `json:"enabled"`
	Mode               AutoScalingMode     `json:"mode"`
	
	// Fixed instance mode
	FixedInstances     int                 `json:"fixed_instances"`
	
	// Auto-scaling mode
	MinInstances       int                 `json:"min_instances"`
	MaxInstances       int                 `json:"max_instances"`
	ScaleUpThreshold   float64             `json:"scale_up_threshold"`
	ScaleDownThreshold float64             `json:"scale_down_threshold"`
	CooldownPeriod     string              `json:"cooldown_period"`
}

// AutoScalingMode defines the auto-scaling mode
type AutoScalingMode string

const (
	FixedInstances AutoScalingMode = "fixed"
	AutoScaling    AutoScalingMode = "auto"
)

// LoadBalancingAlgorithm defines the load balancing algorithm
type LoadBalancingAlgorithm string

const (
	RoundRobin       LoadBalancingAlgorithm = "round_robin"
	Weighted         LoadBalancingAlgorithm = "weighted"
	LeastConnections LoadBalancingAlgorithm = "least_connections"
	HealthBased      LoadBalancingAlgorithm = "health_based"
	Hybrid           LoadBalancingAlgorithm = "hybrid"
)

// RoutingType defines the type of routing logic
type RoutingType string

const (
	StandardRouting      RoutingType = "standard"
	ProbabilityBased     RoutingType = "probability_based"
	DynamicStateBased    RoutingType = "dynamic_state_based"
	CustomLogic          RoutingType = "custom_logic"
)

// EndNodeInterface defines the interface for end nodes
type EndNodeInterface interface {
	ProcessCompletedRequest(*Request) error
	ProcessFailedRequest(*Request) error
	GetMetrics() *EndNodeMetrics
}

// EndNodeMetrics tracks end node performance
type EndNodeMetrics struct {
	CompletedRequests int64   `json:"completed_requests"`
	FailedRequests    int64   `json:"failed_requests"`
	AverageLatency    float64 `json:"average_latency"`
	TotalLatency      float64 `json:"total_latency"`
}
