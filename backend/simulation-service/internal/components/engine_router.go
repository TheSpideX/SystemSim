package components

import (
	"log"
	"strings"

	"github.com/systemsim/simulation-service/internal/engines"
)

// EngineRoutingDecision represents a routing decision made by an engine
type EngineRoutingDecision struct {
	NextEngine    engines.EngineType `json:"next_engine"`    // Which engine to route to next
	RouteToOutput bool               `json:"route_to_output"` // Whether to route to component output
	Reason        string             `json:"reason"`          // Why this routing decision was made
	Metadata      map[string]interface{} `json:"metadata"`    // Additional routing metadata
}

// EngineRouter handles routing decisions between engines within a component instance
type EngineRouter struct {
	// Available engines in this instance
	availableEngines map[engines.EngineType]bool
	
	// Component type for context-aware routing
	componentType ComponentType
	
	// Default routing rules (fallback)
	defaultRouting map[engines.EngineType]engines.EngineType
}

// NewEngineRouter creates a new engine router
func NewEngineRouter(componentType ComponentType, availableEngines []engines.EngineType) *EngineRouter {
	// Build available engines map
	engineMap := make(map[engines.EngineType]bool)
	for _, engineType := range availableEngines {
		engineMap[engineType] = true
	}

	// Create router
	router := &EngineRouter{
		availableEngines: engineMap,
		componentType:    componentType,
		defaultRouting:   make(map[engines.EngineType]engines.EngineType),
	}

	// Set up default routing rules based on component type
	router.setupDefaultRouting()

	return router
}

// setupDefaultRouting sets up default routing rules based on component type
func (er *EngineRouter) setupDefaultRouting() {
	switch er.componentType {
	case ComponentTypeWebServer:
		// Web server: Network → CPU → Memory → Network (output)
		er.defaultRouting[engines.NetworkEngineType] = engines.CPUEngineType
		er.defaultRouting[engines.CPUEngineType] = engines.MemoryEngineType
		er.defaultRouting[engines.MemoryEngineType] = engines.NetworkEngineType

	case ComponentTypeDatabase:
		// Database: Network → CPU → Storage → Memory → Network (output)
		er.defaultRouting[engines.NetworkEngineType] = engines.CPUEngineType
		er.defaultRouting[engines.CPUEngineType] = engines.StorageEngineType
		er.defaultRouting[engines.StorageEngineType] = engines.MemoryEngineType
		er.defaultRouting[engines.MemoryEngineType] = engines.NetworkEngineType

	case ComponentTypeCache:
		// Cache: Network → Memory → Network (output)
		er.defaultRouting[engines.NetworkEngineType] = engines.MemoryEngineType
		er.defaultRouting[engines.MemoryEngineType] = engines.NetworkEngineType

	case ComponentTypeLoadBalancer:
		// Load balancer: Network → CPU → Network (output)
		er.defaultRouting[engines.NetworkEngineType] = engines.CPUEngineType
		er.defaultRouting[engines.CPUEngineType] = engines.NetworkEngineType

	default:
		// Generic: Network → CPU → Memory → Storage → Network (output)
		er.defaultRouting[engines.NetworkEngineType] = engines.CPUEngineType
		er.defaultRouting[engines.CPUEngineType] = engines.MemoryEngineType
		er.defaultRouting[engines.MemoryEngineType] = engines.StorageEngineType
		er.defaultRouting[engines.StorageEngineType] = engines.NetworkEngineType
	}
}

// MakeRoutingDecision determines where an operation should go next after processing by an engine
func (er *EngineRouter) MakeRoutingDecision(
	currentEngine engines.EngineType,
	op *engines.Operation,
	result *engines.OperationResult,
) *EngineRoutingDecision {
	
	// 1. Check if operation explicitly specifies next engine
	if nextEngineStr, exists := op.Metadata["next_engine"]; exists {
		if nextEngineType, ok := er.parseEngineType(nextEngineStr.(string)); ok {
			if er.availableEngines[nextEngineType] {
				return &EngineRoutingDecision{
					NextEngine:    nextEngineType,
					RouteToOutput: false,
					Reason:        "explicit_operation_routing",
					Metadata:      map[string]interface{}{"source": "operation_metadata"},
				}
			}
		}
	}

	// 2. Check if result specifies next engine
	if nextEngineStr, exists := result.Metrics["next_engine"]; exists {
		if nextEngineType, ok := er.parseEngineType(nextEngineStr.(string)); ok {
			if er.availableEngines[nextEngineType] {
				return &EngineRoutingDecision{
					NextEngine:    nextEngineType,
					RouteToOutput: false,
					Reason:        "engine_decision",
					Metadata:      map[string]interface{}{"source": "engine_result"},
				}
			}
		}
	}

	// 3. Apply operation-type based routing logic
	if decision := er.makeOperationTypeDecision(currentEngine, op, result); decision != nil {
		return decision
	}

	// 4. Apply data-size based routing logic
	if decision := er.makeDataSizeDecision(currentEngine, op, result); decision != nil {
		return decision
	}

	// 5. Apply priority-based routing logic
	if decision := er.makePriorityDecision(currentEngine, op, result); decision != nil {
		return decision
	}

	// 6. Use default routing
	if nextEngine, exists := er.defaultRouting[currentEngine]; exists {
		if er.availableEngines[nextEngine] {
			return &EngineRoutingDecision{
				NextEngine:    nextEngine,
				RouteToOutput: false,
				Reason:        "default_routing",
				Metadata:      map[string]interface{}{"source": "default_rules"},
			}
		}
	}

	// 7. Route to output (end of processing)
	return &EngineRoutingDecision{
		NextEngine:    engines.NetworkEngineType, // Always use network for output
		RouteToOutput: true,
		Reason:        "end_of_processing",
		Metadata:      map[string]interface{}{"source": "fallback"},
	}
}

// makeOperationTypeDecision makes routing decisions based on operation type
func (er *EngineRouter) makeOperationTypeDecision(
	currentEngine engines.EngineType,
	op *engines.Operation,
	result *engines.OperationResult,
) *EngineRoutingDecision {
	
	switch op.Type {
	case "read_request":
		// Read requests: try cache first (memory), then storage if cache miss
		if currentEngine == engines.NetworkEngineType {
			if er.availableEngines[engines.MemoryEngineType] {
				return &EngineRoutingDecision{
					NextEngine:    engines.MemoryEngineType,
					RouteToOutput: false,
					Reason:        "read_request_cache_check",
					Metadata:      map[string]interface{}{"operation_type": op.Type},
				}
			}
		}
		if currentEngine == engines.MemoryEngineType {
			// Check if cache hit/miss
			if cacheHit, exists := result.Metrics["cache_hit"]; exists && !cacheHit.(bool) {
				if er.availableEngines[engines.StorageEngineType] {
					return &EngineRoutingDecision{
						NextEngine:    engines.StorageEngineType,
						RouteToOutput: false,
						Reason:        "cache_miss_storage_lookup",
						Metadata:      map[string]interface{}{"cache_hit": false},
					}
				}
			}
		}

	case "write_request":
		// Write requests: CPU processing, then storage, then memory (cache update)
		if currentEngine == engines.NetworkEngineType && er.availableEngines[engines.CPUEngineType] {
			return &EngineRoutingDecision{
				NextEngine:    engines.CPUEngineType,
				RouteToOutput: false,
				Reason:        "write_request_processing",
				Metadata:      map[string]interface{}{"operation_type": op.Type},
			}
		}

	case "compute_request":
		// Compute requests: CPU intensive, may need memory
		if currentEngine == engines.NetworkEngineType && er.availableEngines[engines.CPUEngineType] {
			return &EngineRoutingDecision{
				NextEngine:    engines.CPUEngineType,
				RouteToOutput: false,
				Reason:        "compute_request_processing",
				Metadata:      map[string]interface{}{"operation_type": op.Type},
			}
		}
	}

	return nil // No specific routing for this operation type
}

// makeDataSizeDecision makes routing decisions based on data size
func (er *EngineRouter) makeDataSizeDecision(
	currentEngine engines.EngineType,
	op *engines.Operation,
	result *engines.OperationResult,
) *EngineRoutingDecision {
	
	// Large data operations (> 1MB) should prefer storage over memory
	if op.DataSize > 1024*1024 {
		if currentEngine == engines.CPUEngineType && er.availableEngines[engines.StorageEngineType] {
			return &EngineRoutingDecision{
				NextEngine:    engines.StorageEngineType,
				RouteToOutput: false,
				Reason:        "large_data_storage_preferred",
				Metadata:      map[string]interface{}{"data_size": op.DataSize},
			}
		}
	}

	// Small data operations (< 64KB) can skip storage and go directly to memory
	if op.DataSize < 64*1024 {
		if currentEngine == engines.CPUEngineType && er.availableEngines[engines.MemoryEngineType] {
			return &EngineRoutingDecision{
				NextEngine:    engines.MemoryEngineType,
				RouteToOutput: false,
				Reason:        "small_data_memory_direct",
				Metadata:      map[string]interface{}{"data_size": op.DataSize},
			}
		}
	}

	return nil // No specific routing for this data size
}

// makePriorityDecision makes routing decisions based on operation priority
func (er *EngineRouter) makePriorityDecision(
	currentEngine engines.EngineType,
	op *engines.Operation,
	result *engines.OperationResult,
) *EngineRoutingDecision {
	
	// High priority operations (priority > 7) can skip some processing stages
	if op.Priority > 7 {
		if currentEngine == engines.NetworkEngineType && er.availableEngines[engines.MemoryEngineType] {
			return &EngineRoutingDecision{
				NextEngine:    engines.MemoryEngineType,
				RouteToOutput: false,
				Reason:        "high_priority_fast_path",
				Metadata:      map[string]interface{}{"priority": op.Priority},
			}
		}
	}

	return nil // No specific routing for this priority
}

// parseEngineType parses a string into an EngineType
func (er *EngineRouter) parseEngineType(engineStr string) (engines.EngineType, bool) {
	switch strings.ToLower(engineStr) {
	case "cpu", "cpu_engine":
		return engines.CPUEngineType, true
	case "memory", "memory_engine":
		return engines.MemoryEngineType, true
	case "storage", "storage_engine":
		return engines.StorageEngineType, true
	case "network", "network_engine":
		return engines.NetworkEngineType, true
	default:
		return engines.EngineType(0), false
	}
}

// GetAvailableEngines returns the list of available engines
func (er *EngineRouter) GetAvailableEngines() []engines.EngineType {
	engines := make([]engines.EngineType, 0, len(er.availableEngines))
	for engineType := range er.availableEngines {
		engines = append(engines, engineType)
	}
	return engines
}

// SetDefaultRouting allows overriding default routing rules
func (er *EngineRouter) SetDefaultRouting(from, to engines.EngineType) {
	er.defaultRouting[from] = to
	log.Printf("EngineRouter: Set default routing %s → %s", from, to)
}
