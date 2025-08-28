# Decision Graphs and Routing - Pure Data Structure Architecture

## Overview

**Decision graphs are pure data structures that provide routing lookup information only.** They contain no execution logic, no processing capability, and no intelligence - just simple lookup rules that tell the system where to route requests.

Decision graphs operate at **two distinct levels**: system-level (component-to-component) and component-level (engine-to-engine). Each level uses the same fundamental data structure pattern but stores different types of routing targets.

**Key Principle**: Graphs provide routing information, execution systems read the graphs and perform the actual routing.

## Decision Graph Fundamentals

### Core Principle: Static Data Structures
- **Decision graphs are static lookup tables** - they contain routing rules only
- **No execution logic** - graphs don't process or route anything
- **No intelligence** - graphs don't make decisions, they store decision rules
- **Simple data storage** - graphs are just configuration data
- **Routing decisions made by output systems** - Engine Output Queues and Centralized Output Managers read the static graph and make routing decisions based on conditions

### What Static Graphs Actually Contain
- **Nodes**: Static routing rules with conditions and targets
- **Conditions**: Static boolean logic definitions (cache_hit, authenticated, etc.)
- **Targets**: Static engine names (component-level) or component names (system-level)
- **Next pointers**: Static next destinations for linear flows without decisions

**Critical Understanding**: The graph itself is **completely static** - it's just data. The **Engine Output Queues and Centralized Output Managers** read this static data and make the actual routing decisions based on runtime conditions.

### Example: Static Graph with Dynamic Routing Decisions

#### **Static Graph Definition (Just Data)**
```json
{
    "cache_lookup": {
        "engine": "memory",
        "operation": "cache_lookup",
        "conditions": {
            "cache_hit": "return_result",
            "cache_miss": "database_query"
        }
    },
    "database_query": {
        "engine": "storage",
        "operation": "index_scan",
        "conditions": {
            "data_found": "cache_store",
            "data_not_found": "return_empty"
        }
    }
}
```

#### **Dynamic Routing Decision (Runtime Logic)**
```go
func (eoq *EngineOutputQueue) routeRequest(req *Request, result *EngineResult) {
    // 1. Read STATIC graph from Component LB
    staticGraph := eoq.ComponentLB.getComponentGraph()
    currentNode := staticGraph.Nodes[req.CurrentNode]

    // 2. Make DYNAMIC routing decision based on runtime result
    var nextDestination string
    if result.Status == "cache_hit" {
        nextDestination = currentNode.Conditions["cache_hit"]  // "return_result"
    } else if result.Status == "cache_miss" {
        nextDestination = currentNode.Conditions["cache_miss"] // "database_query"
    }

    // 3. Route to destination determined by static graph + runtime condition
    eoq.routeToDestination(nextDestination, req)
}
```

**Key Points:**
- ✅ **Graph is static** - conditions and destinations never change
- ✅ **Routing is dynamic** - decisions made based on runtime results
- ✅ **Clean separation** - configuration (graph) vs execution (output queues)
- ✅ **Simple logic** - just lookup in static data structure

### Dynamic Routing Based on Request Characteristics

**Engine Output Queues make dynamic routing decisions** based on incoming request characteristics and engine penalties:

```go
func (eoq *EngineOutputQueue) routeBasedOnRequestCharacteristics(req *Request, result *EngineResult) string {
    staticGraph := eoq.ComponentLB.getComponentGraph()
    currentNode := staticGraph.Nodes[req.CurrentNode]

    // Dynamic routing based on request characteristics
    switch req.Operation {
    case "cache_lookup":
        if result.Status == "cache_hit" {
            return currentNode.Conditions["cache_hit"]  // Direct to network output
        } else {
            return currentNode.Conditions["cache_miss"] // Route to storage/database
        }

    case "cpu_intensive_task":
        // Route based on CPU load and request priority
        if req.Priority == "high" && eoq.getCPULoad() > 0.8 {
            return currentNode.Conditions["offload_to_gpu"] // Route to specialized engine
        }
        return currentNode.Conditions["continue_cpu"]

    case "data_processing":
        // Route based on data size
        if req.DataSize > 1024*1024 { // > 1MB
            return currentNode.Conditions["large_data_path"] // Route to storage engine
        }
        return currentNode.Conditions["small_data_path"]    // Route to memory engine
    }

    return currentNode.Next // Default routing
}
```

### Branching Decision Graphs Within Components

**Components can have branching decision graphs** where CPU engine can route to either database (storage) or cache (memory) engines:

```json
CPU Engine Branching Example:
{
    "cpu_processing": {
        "engine": "cpu",
        "operation": "process_request",
        "conditions": {
            "cache_penalty_low": "memory_cache",     // Route to memory if cache penalty low
            "storage_penalty_low": "storage_db",     // Route to storage if storage penalty low
            "both_high_penalty": "network_output"    // Skip both if penalties too high
        }
    },
    "memory_cache": {
        "engine": "memory",
        "operation": "cache_lookup",
        "conditions": {
            "cache_hit": "network_output",
            "cache_miss": "storage_db"
        }
    },
    "storage_db": {
        "engine": "storage",
        "operation": "database_query",
        "conditions": {
            "data_found": "memory_cache_store",
            "data_not_found": "network_error"
        }
    }
}
```

### Dynamic vs Fixed Routing Paths

**Decision graphs provide lookup information** for routing decisions - paths can be dynamic based on current state:

```go
func (eoq *EngineOutputQueue) evaluateRoutingPath(node *DecisionNode, req *Request, result *EngineResult) string {
    // Check for dynamic routing conditions
    for condition, destination := range node.Conditions {
        if eoq.evaluateDynamicCondition(condition, req, result) {
            return destination
        }
    }

    // Fallback to fixed path
    return node.Next
}

func (eoq *EngineOutputQueue) evaluateDynamicCondition(condition string, req *Request, result *EngineResult) bool {
    switch condition {
    case "cache_hit":
        return result.Status == "cache_hit"
    case "cache_miss":
        return result.Status == "cache_miss"
    case "cpu_penalty_high":
        return eoq.getCPUPenalty() > 2.0  // Dynamic penalty check
    case "memory_available":
        return eoq.getMemoryUtilization() < 0.8  // Dynamic memory check
    case "storage_fast":
        return eoq.getStorageLatency() < 10*time.Millisecond  // Dynamic storage check
    }
    return false
}

### Probability-Based Decision Routing

**Decision graphs support custom routing logic** with probability-based decisions where users can define the processing logic and routing rules:

```json
Cache Component with Probability-Based Routing:
{
    "cache_lookup": {
        "engine": "memory",
        "operation": "cache_lookup",
        "routing_type": "probability_based",
        "probability_config": {
            "cache_hit_rate": 0.8,  // 80% cache hit rate
            "conditions": {
                "cache_hit": "return_result",
                "cache_miss": "database_query"
            }
        }
    },
    "database_query": {
        "engine": "storage",
        "operation": "database_lookup",
        "routing_type": "probability_based",
        "probability_config": {
            "success_rate": 0.95,  // 95% success rate
            "conditions": {
                "data_found": "cache_store_result",
                "data_not_found": "return_empty"
            }
        }
    }
}
```

#### **Probability-Based Routing Implementation**

```go
func (eoq *EngineOutputQueue) evaluateProbabilityBasedRouting(node *DecisionNode, req *Request) string {
    if node.RoutingType != "probability_based" {
        return eoq.evaluateStandardRouting(node, req)
    }

    config := node.ProbabilityConfig

    // Generate random number for probability decision
    randomValue := rand.Float64()

    switch node.Operation {
    case "cache_lookup":
        if randomValue < config.CacheHitRate {
            return config.Conditions["cache_hit"]
        }
        return config.Conditions["cache_miss"]

    case "database_lookup":
        if randomValue < config.SuccessRate {
            return config.Conditions["data_found"]
        }
        return config.Conditions["data_not_found"]

    case "network_request":
        if randomValue < config.SuccessRate {
            return config.Conditions["request_success"]
        }
        return config.Conditions["request_timeout"]
    }

    return node.Next // Fallback
}
```

#### **Custom User-Defined Routing Logic**

**Users can define their own processing logic and routing rules**:

```json
Custom API Gateway Component:
{
    "request_validation": {
        "engine": "cpu",
        "operation": "validate_request",
        "routing_type": "custom_logic",
        "custom_config": {
            "validation_rules": [
                {"condition": "authenticated", "route": "process_request"},
                {"condition": "rate_limited", "route": "rate_limit_error"},
                {"condition": "invalid_token", "route": "auth_error"}
            ],
            "default_route": "validation_error"
        }
    },
    "process_request": {
        "engine": "cpu",
        "operation": "business_logic",
        "routing_type": "load_based",
        "load_config": {
            "high_load_threshold": 0.8,
            "conditions": {
                "high_load": "queue_request",
                "normal_load": "process_immediately"
            }
        }
    }
}
```

#### **Educational Benefits of Probability-Based Routing**

```
Student Learning Opportunities:
├── Realistic system behavior - cache hit/miss patterns
├── Performance impact understanding - probability affects latency
├── System design trade-offs - cache size vs hit rate
├── Failure scenario modeling - network timeouts, database failures
└── Custom logic implementation - business rule routing

### Fixed Paths vs Dynamic Paths Based on Current State

**Decision graphs support both fixed and dynamic routing paths** - the choice depends on the use case:

#### **Fixed Paths (Predictable Routing)**
```json
Fixed Path Example - Web Server:
{
    "receive_request": {
        "engine": "network",
        "operation": "receive_http",
        "next": "parse_request"  // Always goes to parse_request
    },
    "parse_request": {
        "engine": "cpu",
        "operation": "parse_http",
        "next": "send_response"  // Always goes to send_response
    },
    "send_response": {
        "engine": "network",
        "operation": "send_http",
        "next": "end"  // Always ends here
    }
}
```

#### **Dynamic Paths (State-Based Routing)**
```json
Dynamic Path Example - Database with Cache:
{
    "query_request": {
        "engine": "cpu",
        "operation": "parse_query",
        "routing_type": "dynamic_state_based",
        "conditions": {
            "cache_enabled": "check_cache",
            "cache_disabled": "query_storage",
            "read_only": "check_cache",
            "write_query": "query_storage"
        }
    },
    "check_cache": {
        "engine": "memory",
        "operation": "cache_lookup",
        "routing_type": "dynamic_state_based",
        "conditions": {
            "cache_hit": "return_result",
            "cache_miss": "query_storage",
            "cache_expired": "query_storage"
        }
    }
}
```

#### **Dynamic State-Based Routing Implementation**
```go
func (eoq *EngineOutputQueue) evaluateStateBasedRouting(node *DecisionNode, req *Request, result *EngineResult) string {
    if node.RoutingType != "dynamic_state_based" {
        return eoq.evaluateFixedRouting(node, req)
    }

    // Dynamic routing based on current system state
    for condition, destination := range node.Conditions {
        if eoq.evaluateCurrentStateCondition(condition, req, result) {
            return destination
        }
    }

    return node.Next // Fallback to fixed path
}

func (eoq *EngineOutputQueue) evaluateCurrentStateCondition(condition string, req *Request, result *EngineResult) bool {
    switch condition {
    case "cache_enabled":
        return eoq.isCacheEnabled()  // Check current system configuration
    case "cache_disabled":
        return !eoq.isCacheEnabled()
    case "high_load":
        return eoq.getCurrentSystemLoad() > 0.8  // Check current load
    case "low_memory":
        return eoq.getAvailableMemory() < 0.2   // Check current memory
    case "storage_fast":
        return eoq.getStorageLatency() < 10*time.Millisecond  // Check current storage performance
    case "read_only":
        return req.Operation == "read" || req.Operation == "select"
    case "write_query":
        return req.Operation == "write" || req.Operation == "insert" || req.Operation == "update"
    }

    return false
}
```

#### **When to Use Fixed vs Dynamic Paths**

##### **Use Fixed Paths When:**
```
Scenarios:
├── Simple linear processing (web servers, file servers)
├── Predictable workflows (authentication flows)
├── Educational simplicity needed (beginner students)
├── Performance is critical (minimal routing overhead)
└── Debugging simplicity required (predictable behavior)

Benefits:
├── Easy to understand and debug
├── Minimal routing overhead
├── Predictable performance characteristics
├── Simple to visualize and edit
└── Good for educational progression
```

##### **Use Dynamic Paths When:**
```
Scenarios:
├── Complex business logic (e-commerce, banking)
├── Performance optimization needed (cache routing)
├── System adaptation required (load-based routing)
├── Advanced educational scenarios (system optimization)
└── Real-world system modeling (production-like behavior)

Benefits:
├── Realistic system behavior
├── Performance optimization opportunities
├── Advanced educational value
├── System adaptation capabilities
└── Production-grade complexity
```

#### **Hybrid Approach (Best of Both)**
```json
Hybrid Example - Smart Cache:
{
    "request_processing": {
        "engine": "cpu",
        "operation": "analyze_request",
        "routing_type": "hybrid",
        "fixed_conditions": {
            "invalid_request": "error_response"  // Always route errors
        },
        "dynamic_conditions": {
            "high_priority": "priority_cache",   // Dynamic based on load
            "normal_priority": "standard_cache"  // Dynamic based on availability
        },
        "fallback": "direct_storage"  // Fixed fallback
    }
}
```

### Educational Benefits of Path Type Choice

**Students learn when to use each approach**:

```
Learning Progression:
├── Week 1-2: Fixed paths only (simple, predictable)
├── Week 3-4: Introduction to dynamic conditions
├── Week 5-6: Complex dynamic routing scenarios
├── Week 7-8: Hybrid approaches for optimization
└── Week 9-10: Production-grade dynamic systems

Decision Criteria Students Learn:
├── Performance requirements vs complexity
├── Predictability vs adaptability trade-offs
├── Debugging complexity vs system intelligence
├── Educational clarity vs real-world alignment
└── Maintenance overhead vs system capabilities
```

Implementation Benefits:
├── Configurable behavior - adjust probabilities for different scenarios
├── Educational scenarios - create controlled failure conditions
├── Realistic modeling - matches real-world system behavior
└── User customization - students define their own routing logic
```
```

### Two-Level Decision Graph Architecture
- **System Level**: Component-to-component routing rules (stored in Global Registry)
- **Component Level**: Engine-to-engine routing rules (stored in Component Load Balancer)
- **No Engine Level**: Engines are atomic processing units with no internal routing

**System Level Examples**:
- **Auth Flow**: Load Balancer → Auth Service → User Database → Response
- **Purchase Flow**: Load Balancer → Product Service → Inventory Database → Payment Service → Response
- **Search Flow**: Load Balancer → Search Service → Search Index → Cache → Response

**Component Level Examples**:
- **Cache Template**: Network(Input) → CPU(hash) ↔ Memory(lookup) → Network(Output)
- **Database Template**: Network(Input) → CPU(parse) → Memory(cache check) → Storage(if miss) → Network(Output)
- **Custom Component**: User-defined engine sequence for specialized components

### Graph Data Structure
```json
Decision Graph = {
    "level": "COMPONENT_LEVEL" | "SYSTEM_LEVEL",
    "name": "cache_component" | "purchase_flow",
    "start_node": "initial_node_id",
    "end_nodes": ["terminal_node_1", "terminal_node_2"],
    "nodes": {
        "node_id": {
            "id": "node_id",
            "type": "engine" | "component" | "decision" | "end",
            "target": "cpu" | "memory" | "auth_service" | "database",
            "operation": "cache_lookup" | "validate_user" | "parse_sql",
            "conditions": {
                "cache_hit": "respond_node",
                "cache_miss": "storage_node",
                "authenticated": "inventory_node",
                "failed": "error_node"
            },
            "next": "linear_next_node"  // For simple linear flows
        }
    }
}
```

### How Graphs Are Used
- **Component Load Balancer**: Stores component-level graphs, provides lookup for Engine Output Queues
- **Global Registry**: Stores system-level graphs, provides lookup for Centralized Output Managers
- **Engine Output Queues**: Read component graphs from LB, evaluate conditions, route to next engine
- **Centralized Output Managers**: Read system graphs from Global Registry, evaluate conditions, route to next component
```

### Algorithm Performance Context Model (90-95% Accuracy)
Decision graphs provide precise algorithm context for accurate performance modeling:

**Cross-Reference**: See `simulation-engine-v2-architecture.md` (Lines 370-406) for complete mathematical formulas and `base-engines-specification.md` (Lines 46-72) for implementation details.

```
Context-Aware Algorithm Performance:
├── Decision graph specifies exact algorithm and complexity
├── Variables are defined and measurable
├── Mathematical precision eliminates guesswork
├── Language performance impact is calculable

Example ML Training Decision Graph Node:
{
    "operation": "train_model",
    "engine": "cpu",
    "time_complexity": "O(n³)",
    "variables": {
        "n": "feature_count",
        "m": "sample_count",
        "k": "iteration_count"
    },
    "base_time": "1ns"
}

Performance Calculation:
complexity_operations = feature_count × sample_count × iteration_count
language_multiplier = cpu_engine.getLanguageMultiplier() // Python: 3.0x, C: 1.0x
load_factor = cpu_engine.getCurrentLoadFactor()
actual_time = base_time × complexity_operations × language_multiplier × load_factor

Algorithm Context Examples:
├── Recommendation engine: O(n²) with users × items
├── Search ranking: O(n log n) with results × ranking_factors
├── Image processing: O(n) with pixel_count
├── Database join: O(n × m) with table1_rows × table2_rows
├── Sorting algorithm: O(n log n) with data_points
```

## System-Level Decision Graphs

### Purpose
Route messages between components in a system design, handling load balancing, failover, and system-wide health management.

### Simplified Flow Chaining with Shared References

**Simplified Approach**: Instead of complex sub-flows, use **simple flow chaining with shared message references** for much cleaner implementation.

#### Flow Chaining Concept
```
Flow Chain: ["auth_flow", "purchase_flow", "payment_flow"]

Shared Request Structure:
├── Request Data: *RequestData (shared pointer - all flows see same data)
├── Flow Chain: *FlowChain (shared chain progression)
├── Current Index: 0 → 1 → 2 (simple progression)
└── Results: map[string]interface{} (shared results between flows)

Benefits:
✅ Automatic data sharing via pointers
✅ No complex synchronization needed
✅ Simple flow progression (just increment index)
✅ Natural data accumulation across flows
```

#### Shared Reference Implementation
```go
type Request struct {
    ID          string
    Data        *RequestData          // POINTER - shared reference
    History     []RequestHistoryEntry

    // Flow chaining (much simpler than sub-flows)
    FlowChain   *FlowChain           // POINTER - shared chain

    // Optional tracking
    TrackHistory bool
}

type RequestData struct {
    UserID      string
    ProductID   string
    AuthResult  *AuthResult          // Auth flow populates this
    InventoryResult *InventoryResult // Inventory flow populates this
    PaymentResult   *PaymentResult   // Payment flow populates this
}

type FlowChain struct {
    Flows        []string // ["auth_flow", "purchase_flow", "payment_flow"]
    CurrentIndex int      // 0, 1, 2...
    Results      map[string]interface{} // Shared results
}
```

#### How Shared References Work
```go
// Start with auth flow
authRequest := &Request{
    ID: "req_123",
    Data: &RequestData{UserID: "user_456"}, // Shared pointer
    FlowChain: &FlowChain{
        Flows: ["auth_flow", "purchase_flow", "payment_flow"],
        CurrentIndex: 0,
    },
}

// Auth flow modifies shared data
func (authService *AuthService) process(req *Request) {
    // Validate user
    authResult := validateUser(req.Data.UserID)

    // Modify shared data - changes visible everywhere
    req.Data.AuthResult = authResult
    req.FlowChain.Results["auth_flow"] = authResult

    // Move to next flow
    req.FlowChain.CurrentIndex++
}

// Purchase flow sees auth results automatically
func (purchaseService *PurchaseService) process(req *Request) {
    // Access auth results from shared data
    if req.Data.AuthResult.IsAuthenticated {
        // Continue with purchase logic
        inventoryResult := checkInventory(req.Data.ProductID)
        req.Data.InventoryResult = inventoryResult // Shared modification
    }
}
```

#### Flow Completion Detection via Registry Marking
```go
// Flow completion via registry marking (much simpler than complex tracking)
func (com *CentralizedOutputManager) completeFlow(req *Request) {
    // Mark flow as completed in registries
    com.componentRegistry.MarkCompleted(req.FlowChain.Flows[req.FlowChain.CurrentIndex], req.ID)
    com.globalRegistry.MarkFlowCompleted(req.FlowChain.Flows[req.FlowChain.CurrentIndex], req.ID)
}

// Event cycle checks completion status
func (com *CentralizedOutputManager) processRequest(req *Request) {
    currentFlow := req.FlowChain.Flows[req.FlowChain.CurrentIndex]

    // Check if current flow is completed
    if com.globalRegistry.IsFlowCompleted(currentFlow, req.ID) {
        // Move to next flow in chain
        req.FlowChain.CurrentIndex++

        if req.FlowChain.CurrentIndex < len(req.FlowChain.Flows) {
            // Start next flow
            nextFlow := req.FlowChain.Flows[req.FlowChain.CurrentIndex]
            com.startFlow(nextFlow, req)
        } else {
            // All flows complete - route to end node
            com.routeToEndNode(req)
        }
    }
    // If not completed, wait for next event cycle
}
```

#### Flow Chain Examples
```json
E-commerce Purchase Flow Chain:
{
    "flows": ["auth_flow", "inventory_flow", "payment_flow"],
    "current_index": 0,
    "results": {
        "auth_flow": {"authenticated": true, "user_id": "123"},
        "inventory_flow": {"in_stock": true, "reserved": true},
        "payment_flow": {"processed": true, "transaction_id": "tx_456"}
    }
}

Social Media Post Flow Chain:
{
    "flows": ["content_moderation_flow", "publish_flow", "notification_flow"],
    "current_index": 1,
    "results": {
        "content_moderation_flow": {"approved": true, "score": 0.95}
    }
}
```

Auth Check Sub-Graph:
{
    "nodes": {
        "auth_service": {
            "type": "processing",
            "next": "user_database"
        },
        "user_database": {
            "type": "processing",
            "next": "permission_check"
        },
        "permission_check": {
            "type": "decision",
            "conditions": {
                "authorized": "auth_success",
                "unauthorized": "auth_failure"
            }
        }
    }
}
```

#### Sub-Graph Execution Flow
```
Tick 1-10: Main flow processes at Load Balancer
Tick 11: Message reaches Product Service
Tick 12: Product Service triggers Auth Sub-Graph
Tick 13-25: Auth Sub-Graph executes independently
    ├── Auth Service processing (5 ticks)
    ├── User Database lookup (8 ticks)
    └── Permission Check decision (2 ticks)
Tick 26: Sub-graph completes with result
Tick 27: Main flow resumes based on sub-graph result
    ├── Success → Continue to Payment Service
    └── Failure → Route to Auth Error Response
```

#### Real-World Sub-Graph Examples
```
E-commerce Purchase Flow:
├── Main: Browse → Add to Cart → Checkout → Payment
├── Auth Sub-Graph: Login → Validate → Permissions
├── Inventory Sub-Graph: Check Stock → Reserve Items → Update Inventory
├── Payment Sub-Graph: Validate Card → Process Payment → Send Receipt

Social Media Post Flow:
├── Main: Create Post → Publish → Notify Followers
├── Content Moderation Sub-Graph: Scan Text → Check Images → Apply Filters
├── Recommendation Sub-Graph: Analyze Content → Update User Profile → Trigger Recommendations

Microservices API Flow:
├── Main: API Gateway → Service Router → Response Aggregator
├── Rate Limiting Sub-Graph: Check Limits → Update Counters → Apply Throttling
├── Circuit Breaker Sub-Graph: Health Check → Failure Detection → Fallback Logic
```

### Example: E-commerce System Routing
```
System Decision Graph = {
    "load_balancer": {
        "healthy_backend": "web_server",
        "backend_overloaded": "queue_request",
        "all_backends_down": "error_response"
    },
    "web_server": {
        "static_content": "cdn_server",
        "api_request": "api_server",
        "user_auth": "auth_service",
        "server_overloaded": "load_balancer"
    },
    "api_server": {
        "database_query": "database_server",
        "cache_lookup": "cache_server",
        "external_api": "api_gateway",
        "processing_error": "error_handler"
    },
    "cache_server": {
        "cache_hit": "web_server",
        "cache_miss": "database_server",
        "cache_error": "database_server"
    },
    "database_server": {
        "query_success": "cache_server",
        "query_timeout": "read_replica",
        "database_error": "error_handler"
    }
}
```

### Health-Aware Routing
System-level graphs incorporate component health status:
- **Healthy components**: Normal routing
- **Stressed components**: Reduced traffic allocation
- **Overloaded components**: Traffic redirection
- **Failed components**: Complete avoidance

### Load Balancing Strategies
- **Round Robin**: Distribute requests evenly across healthy backends
- **Least Connections**: Route to backend with fewest active connections
- **Weighted Round Robin**: Distribute based on component capacity
- **Health-Based**: Route only to healthy components

## Component-Level Decision Graphs

### Purpose
**Pure data structures** that provide engine-to-engine routing lookup information within components.

### Storage and Usage
- **Stored in**: Component Load Balancer (not in messages or global registry)
- **Used by**: Engine Output Queues for routing decisions
- **Scope**: Engine-to-engine routing within single component only
- **Access**: Engine Output Queues lookup routing rules from LB's stored graph

### Graph Complexity Levels

#### **Standard Components (90% of use cases) - Linear Graphs**
Most components use simple linear flows with no decision points:

```json
Cache Component Graph = {
    "level": "COMPONENT_LEVEL",
    "name": "cache_component",
    "nodes": {
        "start": {
            "target": "network_input",
            "operation": "receive_request",
            "next": "lookup"
        },
        "lookup": {
            "target": "memory",
            "operation": "cache_lookup",
            "next": "respond"
        },
        "respond": {
            "target": "network_output",
            "operation": "send_response",
            "next": "end"
        }
    }
}
```

#### **Custom Components (10% of use cases) - Static Decision-Based Graphs**
Advanced users can create components with **static conditional routing definitions**:

**Important**: These are still **static graphs** - users define the conditions and destinations, but the actual routing decisions are made dynamically by Engine Output Queues based on runtime results.

```json
Database Component Graph = {
    "level": "COMPONENT_LEVEL",
    "name": "database_component",
    "nodes": {
        "parse": {
            "target": "cpu",
            "operation": "parse_sql",
            "conditions": {
                "select_query": "check_cache",
                "write_query": "write_storage"
            }
        },
        "check_cache": {
            "target": "memory",
            "operation": "buffer_lookup",
            "conditions": {
                "cache_hit": "respond",
                "cache_miss": "read_storage"
            }
        },
        "read_storage": {
            "target": "storage",
            "operation": "disk_read",
            "next": "respond"
        },
        "respond": {
            "target": "network_output",
            "operation": "send_results",
            "next": "end"
        }
    }
}
```

### How Component Graphs Work

#### **Component Load Balancer Role - Detailed Mechanics**

**Component Load Balancer handles instance management and graph storage** - it does NOT execute routing:

```go
type ComponentLoadBalancer struct {
    // Identity
    ComponentID     string
    ComponentType   ComponentType

    // Instance management
    instances       map[string]*ComponentInstance

    // Component-level graph (moved from global registry)
    componentGraph  *DecisionGraph  // COMPONENT_LEVEL graph

    // Global registry access (for system-level routing only)
    globalRegistry  *GlobalRegistry

    // Load balancing
    algorithm       LoadBalancingAlgorithm
    roundRobinIndex int
}

func (clb *ComponentLoadBalancer) processOperation(op *Operation) error {
    // 1. Select instance using load balancing
    instance := clb.selectInstance(op)

    // 2. Use component graph to determine engine routing within instance
    engineSequence := clb.determineEngineSequence(op)

    // 3. Send operation to instance with engine routing instructions
    op.EngineSequence = engineSequence  // LB tells instance how to route internally

    // 4. Route to selected instance
    return instance.processOperation(op)
}

func (clb *ComponentLoadBalancer) processOperation(op *Operation) error {
    // 1. Select instance using load balancing
    instance := clb.selectInstance(op)

    // 2. Component LB just stores graph - does NOT determine sequences upfront
    // Engine Output Queues will read graph dynamically during processing

    // 3. Route to selected instance (instance starts with first engine)
    return instance.processOperation(op)
}

// Component LB provides graph access to Engine Output Queues
func (clb *ComponentLoadBalancer) getComponentGraph() *DecisionGraph {
    return clb.componentGraph  // Engine Output Queues read this dynamically
}
```

**Key Responsibilities:**
- ✅ **Store component-level decision graph** (just data storage)
- ✅ **Select instance** using load balancing algorithm
- ✅ **Provide graph access** to Engine Output Queues
- ✅ **Determine engine sequences** for instances
- ❌ **No routing execution** - just graph storage and instance selection

#### **Centralized Output Manager Role - Detailed Mechanics**

**Centralized Output Managers handle system-level routing** between components:

```go
type CentralizedOutputManager struct {
    InstanceID     string
    ComponentID    string

    // Global registry access (no local graph storage)
    GlobalRegistry *GlobalRegistry

    InputChannel   chan *engines.OperationResult
    OutputChannel  chan *engines.OperationResult
}

func (com *CentralizedOutputManager) handleOperationResult(result *engines.OperationResult) error {
    // 1. Get request context from global registry
    requestCtx := com.GlobalRegistry.GetRequestContext(result.RequestID)
    if requestCtx == nil {
        return fmt.Errorf("request context not found for %s", result.RequestID)
    }

    // 2. Get system graph from global registry
    systemGraph := com.GlobalRegistry.GetSystemGraph(requestCtx.SystemFlowID)
    if systemGraph == nil {
        return fmt.Errorf("system graph not found for flow %s", requestCtx.SystemFlowID)
    }

    // 3. Determine next component using system graph
    nextComponent, subFlowRequired := com.evaluateSystemGraph(systemGraph, requestCtx, result)

    // 4. Handle sub-flow execution if needed
    if subFlowRequired != "" {
        return com.executeSubFlow(subFlowRequired, requestCtx, result)
    }

    // 5. Route to next component
    return com.routeToNextComponent(nextComponent, result)
}

func (com *CentralizedOutputManager) evaluateSystemGraph(graph *DecisionGraph, ctx *RequestContext, result *engines.OperationResult) (string, string) {
    currentNode := graph.Nodes[ctx.CurrentSystemNode]

    // Evaluate business logic conditions
    for condition, nextNode := range currentNode.Conditions {
        if com.evaluateBusinessCondition(condition, result, ctx) {
            if graph.Nodes[nextNode].Type == "sub_flow" {
                return "", graph.Nodes[nextNode].Target // Return sub-flow ID
            }
            return graph.Nodes[nextNode].Target, "" // Return component ID
        }
    }

    return currentNode.Next, "" // Default progression
}
```

**Key Responsibilities:**
- ✅ **Read system graphs** from Global Registry
- ✅ **Evaluate business logic conditions** (authenticated, in_stock, payment_success)
- ✅ **Route to next component** via Global Registry
- ✅ **Handle flow chaining** and sub-flow execution
- ✅ **Manage request context** throughout system journey
- ❌ **No graph storage** - just reads from Global Registry

### Centralized Output Goroutine Per Instance Architecture

**Each component instance has its own Centralized Output Manager** that handles system-level routing:

```go
type ComponentInstance struct {
    ID            string
    ComponentID   string

    // Engines (no graph knowledge needed)
    engines       map[EngineType]*EngineWrapper

    // Centralized output (one per instance)
    outputManager *CentralizedOutputManager
}

func (ci *ComponentInstance) startCentralizedOutput() {
    // Each instance gets its own centralized output goroutine
    go ci.outputManager.run()
}

func (com *CentralizedOutputManager) run() {
    for {
        select {
        case result := <-com.InputChannel:
            // Handle system-level routing for this instance
            com.handleOperationResult(result)
        case <-com.ctx.Done():
            return
        }
    }
}
```

### Centralized Output vs Output Network Engine

**We chose Centralized Output Manager approach** over letting output network engine handle inter-component routing:

#### **Centralized Output Manager Approach (Chosen)**
```
Benefits:
├── Clean separation - engines focus on processing, not routing
├── System-level routing expertise - dedicated component for inter-component routing
├── Flow chaining support - handles complex multi-flow scenarios
├── Business logic evaluation - can evaluate complex conditions
└── Scalable - one per instance, handles all system routing

Implementation:
├── Engine Output Queue routes to Centralized Output when component complete
├── Centralized Output reads system graphs from Global Registry
├── Centralized Output evaluates business conditions and routes to next component
└── Clean handoff between component-level and system-level routing
```

#### **Output Network Engine Approach (Not Chosen)**
```
Drawbacks:
├── Mixed responsibilities - network engine doing both networking and routing
├── Complex engine logic - engines would need system-level knowledge
├── Harder to maintain - routing logic scattered across engines
└── Less scalable - network engines not designed for complex routing decisions
```

### Benefits of Centralized Output Per Instance

```go
// Each instance handles its own system-level routing
func (com *CentralizedOutputManager) handleOperationResult(result *engines.OperationResult) error {
    // 1. This instance's centralized output handles system routing
    nextComponent := com.determineNextComponent(result)

    // 2. Route to next component via Global Registry
    if nextComponent != "" {
        return com.routeToNextComponent(nextComponent, result)
    }

    // 3. End of flow - route to end node
    return com.routeToEndNode(result)
}
```

#### **Component Instance Role - Engine Sequence Execution**

**Component Instances just execute engine sequences** provided by Component LB:

```go
type ComponentInstance struct {
    // Identity
    ID            string
    ComponentID   string

    // Engines (no graph knowledge needed)
    engines       map[EngineType]*EngineWrapper

    // Centralized output (only handles system-level routing)
    outputManager *CentralizedOutputManager
}

func (ci *ComponentInstance) processOperation(op *Operation) error {
    // Instance just follows the engine sequence provided by LB
    for _, engineStep := range op.EngineSequence {
        engine := ci.engines[engineStep.Engine]

        // Process through engine
        result, err := engine.ProcessOperation(op, engineStep.Operation)
        if err != nil {
            return err
        }

        // Update operation with result
        op.Data = result.Data
    }

    // After all engines processed, send to centralized output for system-level routing
    return ci.outputManager.handleOperationResult(op)
}
```

**Key Responsibilities:**
- ✅ **Execute engine sequence** provided by Component LB
- ✅ **Process through engines** in specified order
- ✅ **Update operation data** with engine results
- ✅ **Send to Centralized Output** when component processing complete
- ❌ **No routing decisions** - just follows provided sequence
- ❌ **No graph knowledge** - LB handles all routing logic

### Complete Request Journey - All Components Working Together

**The complete flow showing how all components work together:**

```
1. Request Creation:
   ├── User creates request with optional tracking
   ├── Request contains shared data pointers
   ├── Flow chain specifies sequence of flows
   └── Global Registry stores request context

2. System-Level Routing (Centralized Output Manager):
   ├── Reads system graph from Global Registry
   ├── Evaluates business logic conditions
   ├── Determines next component in flow
   └── Routes request to Component LB

3. Component-Level Processing (Component Load Balancer):
   ├── Stores component graph (data only)
   ├── Selects instance using algorithm
   ├── Determines engine sequence from graph
   └── Routes request to Component Instance

4. Engine Sequence Execution (Component Instance):
   ├── Receives engine sequence from LB
   ├── Processes through engines in order
   ├── Updates shared request data
   └── Sends to Centralized Output when complete

5. Engine-Level Routing (Engine Output Queue):
   ├── Reads component graph from LB
   ├── Evaluates engine conditions
   ├── Routes to next engine or Centralized Output
   └── Simple lookup and push operations

6. Flow Completion:
   ├── Centralized Output checks flow completion
   ├── Moves to next flow in chain if needed
   ├── Routes to End Node when all flows complete
   └── End Node marks complete and drains request
```

**Key Architectural Benefits:**
- ✅ **Clean separation of concerns** - each component has one clear job
- ✅ **No complex coordination** - simple data passing
- ✅ **Shared data automatically** - via pointers across flows
- ✅ **Natural backpressure** - through channel capacity
- ✅ **Optional tracking** - performance optimized
- ✅ **Scalable architecture** - same patterns at all scales

### Error Handling and Failure Injection

#### **Natural Backpressure (No Complex Coordination)**

**Requests cannot get stuck** if implemented properly - natural backpressure handles overload:

```go
// Natural backpressure through channel capacity
func (clb *ComponentLoadBalancer) processRequest(req *Request) error {
    instance := clb.selectInstance(req)

    select {
    case instance.InputChannel <- req:
        // Request accepted
        return nil
    default:
        // Channel full - natural backpressure
        // Can queue, create new instance, or return error
        return clb.handleBackpressure(req)
    }
}

func (clb *ComponentLoadBalancer) handleBackpressure(req *Request) error {
    // Option 1: Create new instance (auto-scaling)
    if clb.canScale() {
        newInstance := clb.createNewInstance()
        return newInstance.processRequest(req)
    }

    // Option 2: Queue with timeout
    return clb.queueWithTimeout(req, 5*time.Second)
}
```

#### **Failure Injection for Educational Value**

**Use failure injection to create artificial problems** and see how the system responds:

```go
type FailureInjector struct {
    componentFailures map[string]float64  // component_id -> failure_rate
    engineFailures    map[string]float64  // engine_type -> failure_rate
    networkLatency    time.Duration       // artificial network delay
    enabled           bool
}

func (fi *FailureInjector) shouldInjectFailure(componentID string) bool {
    if !fi.enabled {
        return false
    }

    failureRate := fi.componentFailures[componentID]
    return rand.Float64() < failureRate
}

// Educational scenarios
func (fi *FailureInjector) createEducationalScenarios() {
    // Scenario 1: Database overload
    fi.componentFailures["database"] = 0.3  // 30% failure rate

    // Scenario 2: Network partition
    fi.networkLatency = 5 * time.Second

    // Scenario 3: Memory pressure
    fi.engineFailures["memory"] = 0.2  // 20% memory failures
}
```

#### **Error Routing to End Nodes**

**On error, route requests to end nodes** where status is updated as failure:

```go
func (com *CentralizedOutputManager) handleError(req *Request, err error) {
    // Update request status
    req.Status = RequestStatusFailed
    req.ErrorMessage = err.Error()
    req.EndTime = time.Now()

    // Route to error end node
    com.routeToEndNode(req, "error_end_node")
}

func (en *EndNode) processFailedRequest(req *Request) {
    // Log failure for analysis
    en.logFailure(req)

    // Update failure statistics
    en.GlobalRegistry.UpdateFailureStats(req)

    // Clean up and drain
    en.GlobalRegistry.CleanupRequest(req.ID)
    en.drainRequest(req)
}
```

#### **Benefits of Natural Error Handling**
- ✅ **Realistic system behavior** - matches production failure patterns
- ✅ **Educational value** - students see how failures propagate
- ✅ **No complex coordination** - simple error routing
- ✅ **Failure injection** - create controlled learning scenarios
- ✅ **Natural recovery** - system self-heals through health scores

### Performance Monitoring and Educational Metrics

#### **Comprehensive Performance Tracking**

**Track performance at all levels** for educational insights:

```go
type PerformanceMonitor struct {
    // Request-level metrics
    requestLatencies    map[string]time.Duration  // request_id -> total_latency
    requestPaths        map[string][]string       // request_id -> component_path
    requestFailures     map[string]string         // request_id -> failure_reason

    // Component-level metrics
    componentThroughput map[string]float64        // component_id -> ops_per_second
    componentLatencies  map[string]time.Duration  // component_id -> avg_latency
    componentHealth     map[string]float64        // component_id -> health_score

    // Engine-level metrics
    engineUtilization   map[string]float64        // engine_type -> utilization
    engineQueueDepth    map[string]int            // engine_type -> queue_depth
    engineFailureRate   map[string]float64        // engine_type -> failure_rate

    // System-level metrics
    totalThroughput     float64                   // system ops_per_second
    averageLatency      time.Duration             // system avg_latency
    systemHealth        float64                   // overall system health
}

func (pm *PerformanceMonitor) trackRequestJourney(req *Request) {
    if !req.TrackHistory {
        return // Skip detailed tracking for performance
    }

    // Calculate request latency
    latency := req.EndTime.Sub(req.StartTime)
    pm.requestLatencies[req.ID] = latency

    // Extract component path
    path := []string{}
    for _, entry := range req.History {
        path = append(path, entry.ComponentID)
    }
    pm.requestPaths[req.ID] = path

    // Update component metrics
    pm.updateComponentMetrics(req)
}
```

#### **Educational Insights Dashboard**

**Provide real-time insights** for learning:

```go
type EducationalDashboard struct {
    performanceMonitor *PerformanceMonitor

    // Real-time metrics
    currentThroughput   float64
    currentLatency      time.Duration
    bottleneckComponent string

    // Educational insights
    designRecommendations []string
    performanceIssues     []string
    optimizationTips      []string
}

func (ed *EducationalDashboard) generateInsights() {
    // Identify bottlenecks
    bottleneck := ed.identifyBottleneck()
    if bottleneck != "" {
        ed.designRecommendations = append(ed.designRecommendations,
            fmt.Sprintf("Consider scaling %s - it's your bottleneck", bottleneck))
    }

    // Analyze failure patterns
    if ed.performanceMonitor.systemHealth < 0.8 {
        ed.performanceIssues = append(ed.performanceIssues,
            "System health is degraded - check component failures")
    }

    // Suggest optimizations
    ed.suggestOptimizations()
}

func (ed *EducationalDashboard) identifyBottleneck() string {
    maxLatency := time.Duration(0)
    bottleneck := ""

    for componentID, latency := range ed.performanceMonitor.componentLatencies {
        if latency > maxLatency {
            maxLatency = latency
            bottleneck = componentID
        }
    }

    return bottleneck
}
```

#### **A/B Testing for Educational Experiments**

**Enable students to test different architectures**:

```go
type ArchitectureExperiment struct {
    name            string
    description     string

    // Architecture variants
    variantA        *SystemArchitecture
    variantB        *SystemArchitecture

    // Traffic split
    trafficSplit    float64  // 0.5 = 50/50 split

    // Results
    resultsA        *ExperimentResults
    resultsB        *ExperimentResults
}

func (ae *ArchitectureExperiment) routeRequest(req *Request) *SystemArchitecture {
    if rand.Float64() < ae.trafficSplit {
        return ae.variantA
    }
    return ae.variantB
}

func (ae *ArchitectureExperiment) compareResults() *ExperimentComparison {
    return &ExperimentComparison{
        ThroughputDiff: ae.resultsB.Throughput - ae.resultsA.Throughput,
        LatencyDiff:    ae.resultsB.AvgLatency - ae.resultsA.AvgLatency,
        CostDiff:       ae.resultsB.Cost - ae.resultsA.Cost,
        Recommendation: ae.generateRecommendation(),
    }
}
```

#### **Benefits of Performance Monitoring**
- ✅ **Real-time insights** - immediate feedback on design decisions
- ✅ **Bottleneck identification** - clear visibility into performance issues
- ✅ **Educational recommendations** - guided learning through suggestions
- ✅ **A/B testing capability** - compare different architectures
- ✅ **Historical analysis** - track performance over time
- ✅ **Failure pattern analysis** - understand how failures propagate

#### **Engine Output Queue Role - Detailed Mechanics**

**Engine Output Queues are the core routing executors** - they read graphs and perform all component-level routing:

```go
type EngineOutputQueue struct {
    EngineType     engines.EngineType
    ComponentLB    *ComponentLoadBalancer  // Reference to LB for graph lookup
    OutputChannel  chan *engines.Operation
}

func (eoq *EngineOutputQueue) routeRequest(req *Request, result *engines.OperationResult) {
    // 1. Simple lookup in LB graph
    currentNode := eoq.ComponentLB.ComponentGraph.Nodes[req.CurrentNode]

    // 2. Evaluate condition to find next engine
    nextEngine := eoq.evaluateCondition(currentNode, result)

    // 3. Simple push to next engine's input queue
    nextEngineQueue := eoq.getEngineInputQueue(nextEngine)
    nextEngineQueue <- req
}

func (eoq *EngineOutputQueue) evaluateCondition(node *DecisionNode, result *engines.OperationResult) string {
    // Simple condition evaluation
    for condition, nextEngine := range node.Conditions {
        if eoq.checkCondition(condition, result) {
            return nextEngine
        }
    }
    return node.Conditions["default"] // Fallback
}
```

**Key Responsibilities:**
- ✅ **Handle ALL routing decisions** (both internal and external)
- ✅ **Dynamic routing** by looking up component/global graphs
- ✅ **Read routing rules** from Component LB's stored graph
- ✅ **Evaluate simple conditions** (cache_hit, cache_miss, parse_success, etc.)
- ✅ **Route to next engine** within component
- ✅ **Route to Centralized Output** when component processing complete
- ❌ **No complex logic** - just lookup and push

### Engine Output Queue Handles ALL Routing Decisions

**Engine Output Queue is the single point** for ALL routing decisions - both internal (engine-to-engine) and external (component-to-component):

```go
func (eoq *EngineOutputQueue) handleAllRouting(req *Request, result *EngineResult) error {
    // Engine Output Queue handles ALL routing decisions

    // 1. Look up component-level graph for internal routing
    componentGraph := eoq.ComponentLB.getComponentGraph()
    currentNode := componentGraph.Nodes[req.CurrentNode]

    // 2. Determine next destination (internal or external)
    nextDestination := eoq.evaluateRoutingConditions(currentNode, req, result)

    // 3. Route based on destination type
    if eoq.isInternalEngine(nextDestination) {
        // Internal routing - route to next engine within component
        return eoq.routeToInternalEngine(nextDestination, req)
    } else if eoq.isExternalComponent(nextDestination) {
        // External routing - route to different component
        return eoq.routeToExternalComponent(nextDestination, req)
    } else if eoq.isEndNode(nextDestination) {
        // End routing - route to end node
        return eoq.routeToEndNode(req)
    }

    return fmt.Errorf("unknown destination: %s", nextDestination)
}

func (eoq *EngineOutputQueue) evaluateRoutingConditions(node *DecisionNode, req *Request, result *EngineResult) string {
    // Dynamic routing by looking up graphs

    // Check component-level conditions first
    for condition, destination := range node.Conditions {
        if eoq.evaluateComponentCondition(condition, req, result) {
            return destination
        }
    }

    // Check global-level conditions if needed
    if node.RequiresGlobalLookup {
        globalGraph := eoq.GlobalRegistry.getSystemGraph(req.SystemFlowID)
        return eoq.evaluateGlobalConditions(globalGraph, req, result)
    }

    return node.Next // Default routing
}
```

### Dynamic Routing by Looking Up Component/Global Graphs

**Engine Output Queue dynamically looks up graphs** rather than using upfront sequences:

```go
func (eoq *EngineOutputQueue) dynamicGraphLookup(req *Request, result *EngineResult) string {
    // 1. Dynamic lookup in component graph
    componentGraph := eoq.ComponentLB.getComponentGraph()

    // 2. Evaluate current node conditions
    currentNode := componentGraph.Nodes[req.CurrentNode]

    // 3. Dynamic condition evaluation
    for condition, destination := range currentNode.Conditions {
        if eoq.evaluateDynamicCondition(condition, req, result) {
            // 4. Check if destination requires global graph lookup
            if eoq.isGlobalDestination(destination) {
                return eoq.lookupGlobalGraph(destination, req)
            }
            return destination
        }
    }

    return currentNode.Next
}

func (eoq *EngineOutputQueue) lookupGlobalGraph(destination string, req *Request) string {
    // Dynamic lookup in global registry
    globalGraph := eoq.GlobalRegistry.getSystemGraph(req.SystemFlowID)

    // Find the actual component to route to
    if globalNode, exists := globalGraph.Nodes[destination]; exists {
        return globalNode.Target // Actual component ID
    }

    return destination // Fallback
}
```

**Benefits of Engine Output Queue Handling ALL Routing:**
- ✅ **Single routing point** - all decisions in one place
- ✅ **Dynamic graph lookup** - no upfront sequence calculation needed
- ✅ **Flexible routing** - can handle both internal and external routing
- ✅ **Simple architecture** - engines just process, Engine Output Queue routes
- ✅ **Easy debugging** - all routing logic in one component

#### **Clarified Routing Flow - ALL Engines Route to Engine Output Queue**

**ALWAYS route to Engine Output Queue** - which then decides internal vs external routing:

```go
func (engine *Engine) completeOperation(req *Request, result *EngineResult) {
    // ALL engines route to Engine Output Queue (no exceptions)
    engine.outputQueue.routeRequest(req, result)
}

func (eoq *EngineOutputQueue) routeRequest(req *Request, result *EngineResult) {
    // Engine Output Queue makes ALL routing decisions

    // 1. Look at component-level graph to determine next step
    componentGraph := eoq.ComponentLB.ComponentGraph
    currentNode := componentGraph.Nodes[req.CurrentNode]

    // 2. Evaluate conditions to find next destination
    nextDestination := eoq.evaluateConditions(currentNode, result)

    // 3. Route based on destination type
    if eoq.isInternalEngine(nextDestination) {
        // Route to next engine within component
        eoq.routeToEngine(nextDestination, req)
    } else {
        // Route to Centralized Output for system-level routing
        eoq.routeToCentralizedOutput(req, result)
    }
}
```

#### **Timeout System for Stuck Requests**

**If requests get stuck, automatically route to error end node**:

```go
type RequestTimeoutManager struct {
    activeRequests map[string]*RequestTimeout
    timeoutDuration time.Duration // e.g., 30 seconds
}

type RequestTimeout struct {
    requestID   string
    startTime   time.Time
    lastUpdate  time.Time
    component   string
    engine      string
}

func (rtm *RequestTimeoutManager) checkTimeouts() {
    for requestID, timeout := range rtm.activeRequests {
        if time.Since(timeout.lastUpdate) > rtm.timeoutDuration {
            // Request is stuck - route to error end node
            rtm.routeToErrorEndNode(requestID, "request_timeout")
        }
    }
}

func (rtm *RequestTimeoutManager) routeToErrorEndNode(requestID, reason string) {
    req := rtm.globalRegistry.GetRequest(requestID)
    if req != nil {
        req.Status = RequestStatusFailed
        req.ErrorMessage = reason
        req.EndTime = time.Now()

        // Route directly to error end node
        rtm.globalRegistry.RouteToEndNode(req, "error_end_node")
    }
}
```
    "storage_read": {
        Engine: "storage",
        Operation: "index_scan",
        TimeComplexity: "O(log n)",
        Conditions: {
            "data_found": "cache_results",
            "data_not_found": "return_empty",
            "storage_error": "error_response"
        }
    },
    "cache_results": {
        Engine: "memory",
        Operation: "buffer_pool_store",
        Conditions: {
            "cache_success": "return_results",
            "cache_full": "evict_and_store"
        }
    },
    "return_results": {
        Engine: "network",
        Operation: "send_response",
        Conditions: {
            "response_sent": "complete"
        }
    }
}
```

### Engine Coordination Patterns
- **Sequential Processing**: Engines process in order (Network → CPU → Memory → Storage)
- **Parallel Processing**: Multiple engines work simultaneously
- **Conditional Routing**: Different paths based on data or state
- **Error Handling**: Fallback paths for engine failures

### Performance Optimization
- **Engine Affinity**: Keep related operations on same engine
- **Cache Locality**: Minimize memory engine round trips
- **I/O Batching**: Group storage operations for efficiency
- **Network Optimization**: Minimize serialization overhead

## Engine-Level Decision Graphs

### Purpose
Handle engine-specific routing and optimization within individual engines.

### Example: Memory Engine Internal Routing
```
Memory Engine Graph = {
    "memory_request": {
        Operation: "determine_access_type",
        Conditions: {
            "cache_lookup": "l1_cache_check",
            "data_store": "memory_allocation",
            "bulk_operation": "memory_mapping"
        }
    },
    "l1_cache_check": {
        Operation: "l1_cache_lookup",
        AccessTime: "1_cycle",
        Conditions: {
            "l1_hit": "return_data",
            "l1_miss": "l2_cache_check"
        }
    },
    "l2_cache_check": {
        Operation: "l2_cache_lookup",
        AccessTime: "10_cycles",
        Conditions: {
            "l2_hit": "return_data",
            "l2_miss": "l3_cache_check"
        }
    },
    "l3_cache_check": {
        Operation: "l3_cache_lookup",
        AccessTime: "50_cycles",
        Conditions: {
            "l3_hit": "return_data",
            "l3_miss": "main_memory_access"
        }
    },
    "main_memory_access": {
        Operation: "ram_access",
        AccessTime: "300_cycles",
        Conditions: {
            "memory_available": "return_data",
            "memory_pressure": "swap_access"
        }
    }
}
```

### Engine Optimization Strategies
- **Cache Hierarchy**: Model realistic memory hierarchies
- **Prefetching**: Anticipate future memory accesses
- **Write Buffering**: Optimize write operations
- **Resource Scheduling**: Manage concurrent engine operations

## Message-Carried Routing Context

### Self-Routing Architecture
Components are autonomous and make routing decisions based on context carried in messages:

### Request Structure - Simple Data Structure

**Key Principle**: **Request is just a data structure** passed through the system with optional tracking.

```go
type Request struct {
    // Core identification
    ID          string                 `json:"id"`

    // Shared data (pointer for automatic sharing across flows)
    Data        *RequestData          `json:"data"`

    // Flow chaining (much simpler than complex sub-flows)
    FlowChain   *FlowChain           `json:"flow_chain"`

    // Optional tracking (configurable per request)
    TrackHistory bool                  `json:"track_history"`
    History     []RequestHistoryEntry `json:"history,omitempty"`

    // Simple counters (lightweight when tracking disabled)
    ComponentCount int                `json:"component_count"`
    EngineCount    int                `json:"engine_count"`
    StartTime      time.Time          `json:"start_time"`
    EndTime        time.Time          `json:"end_time"`
}

type RequestData struct {
    // Core request data
    UserID      string                `json:"user_id"`
    ProductID   string                `json:"product_id"`
    Operation   string                `json:"operation"`

    // Flow results (automatically shared via pointer)
    AuthResult      *AuthResult      `json:"auth_result,omitempty"`
    InventoryResult *InventoryResult `json:"inventory_result,omitempty"`
    PaymentResult   *PaymentResult   `json:"payment_result,omitempty"`
}

type RequestHistoryEntry struct {
    ComponentID   string      `json:"component_id"`
    EngineType    string      `json:"engine_type"`
    Operation     string      `json:"operation"`
    Timestamp     time.Time   `json:"timestamp"`
    Result        interface{} `json:"result"`
}
```
    
    // Decision context
    RequestType: "user_lookup",
    UserID: "user_123",
    SessionData: {...},
    PerformanceRequirements: {
        MaxLatency: "100ms",
        RequiredThroughput: "1000_rps"
    },
    
    // Health and load context
    ComponentHealthStatus: {
        "database_server": "healthy",
        "cache_server": "stressed",
        "web_server": "overloaded"
    }
}
```

### Request Processing with Optional Tracking
```go
// Each engine checks tracking flag and adds history if enabled
func (engine *Engine) processOperation(req *Request) *engines.OperationResult {
    // Do the actual work
    result := engine.doWork(req)

    // Optional history tracking (no performance overhead when disabled)
    if req.TrackHistory {
        req.History = append(req.History, RequestHistoryEntry{
            ComponentID: engine.ComponentID,
            EngineType:  engine.Type,
            Operation:   req.Operation,
            Timestamp:   time.Now(),
            Result:      result.Status,
        })
    }

    // Always update lightweight counters
    req.EngineCount++

    return result
}
```

### End Node Pattern for Request Completion

**All completed requests are forwarded to end nodes** where they are marked complete and drained from the system.

```go
type EndNode struct {
    GlobalRegistry *GlobalRegistry
}

func (en *EndNode) processCompletedRequest(req *Request) {
    // Mark request as complete
    req.Status = RequestStatusCompleted
    req.EndTime = time.Now()

    // Update global statistics
    en.GlobalRegistry.UpdateRequestStats(req)

    // Clean up request context
    en.GlobalRegistry.CleanupRequest(req.ID)

    // Drain from system (request journey complete)
    en.drainRequest(req)
}
```

### Natural Backpressure (No Complex Coordination)

**Requests cannot get stuck** if implemented properly - natural backpressure handles overload:

```go
// Natural backpressure through channel capacity
func (clb *ComponentLoadBalancer) processRequest(req *Request) error {
    instance := clb.selectInstance(req)

    select {
    case instance.InputChannel <- req:
        // Request accepted
        return nil
    default:
        // Channel full - natural backpressure
        // Can queue, create new instance, or return error
        return clb.handleBackpressure(req)
    }
}

func (clb *ComponentLoadBalancer) handleBackpressure(req *Request) error {
    // Option 1: Create new instance (auto-scaling)
    if clb.canScale() {
        newInstance := clb.createNewInstance()
        return newInstance.processRequest(req)
    }

    // Option 2: Queue with timeout
    return clb.queueWithTimeout(req, 5*time.Second)
}
```

## Advanced Routing Features

### Circuit Breaker Pattern
```
Circuit Breaker Decision Logic:
- **Closed**: Normal routing to component
- **Open**: Route around failed component
- **Half-Open**: Test component recovery

Circuit Breaker States:
"database_circuit": {
    State: "closed" | "open" | "half_open",
    FailureCount: 0,
    FailureThreshold: 5,
    TimeoutDuration: "30s",
    LastFailureTime: "timestamp"
}
```

### Retry and Backoff Logic
```
Retry Decision Graph:
"operation_with_retry": {
    Operation: "database_query",
    Conditions: {
        "success": "return_result",
        "timeout": "retry_with_backoff",
        "permanent_error": "error_response"
    },
    RetryPolicy: {
        MaxRetries: 3,
        BackoffStrategy: "exponential",
        BaseDelay: "100ms"
    }
}
```

### Adaptive Routing
Decision graphs can adapt based on performance metrics:
- **Load-based routing**: Adjust paths based on component load
- **Latency-based routing**: Choose fastest available path
- **Cost-based routing**: Optimize for resource efficiency
- **SLA-based routing**: Ensure service level requirements

## Graph Execution Engine

### Concurrent Decision Graph Execution

#### How Multiple Graphs Run Simultaneously
- **System-level graphs**: Route messages between components (different goroutines)
- **Component-level graphs**: Route messages within components (same goroutine)
- **No coordination needed**: Different scopes prevent conflicts
- **Natural resource contention**: Engines are shared, creating realistic performance impact

#### Queue-Based Message Routing
```
Message with Route Queue:
{
    request_data: {...},
    route_queue: ["load_balancer", "auth_service", "cache", "database"],
    current_component: "auth_service",
    completed_route: ["load_balancer"],
    component_graph: {auth_service_internal_routing},
    performance_data: {...}
}
```

#### Component Processing Logic
```
1. Receive message from input channel
2. Check route_queue front: Is it my identifier?
3. If YES: Pop my identifier from front of queue
4. Process through internal engines using component-level graph
5. Look at new front of route_queue for next destination
6. Add my identifier to completed_route
7. Send message to next component via ComponentRegistry lookup
8. If route_queue empty: End of journey, send response back
```

#### Concurrent Execution Example
```
3 Simultaneous Requests Processing:

Request 1: Auth login check
├── System Graph: ["load_balancer", "auth_service", "cache", "database"]
├── Current: auth_service processing internally
├── Component Graph: Network → CPU → Memory → Network
└── Engines: Competing for shared CPU/Memory resources

Request 2: Search result lookup
├── System Graph: ["load_balancer", "search_service", "cache", "search_engine"]
├── Current: cache processing internally
├── Component Graph: Network → CPU → Memory (MISS) → Network
└── Engines: Same cache engines, creating realistic contention

Request 3: Product data check
├── System Graph: ["load_balancer", "product_service", "cache", "database"]
├── Current: database processing internally
├── Component Graph: Network → CPU → Storage → Network
└── Engines: Database engines under load, performance degrades
```

#### Why No Conflicts Occur
1. **Different Scopes**: System graphs route between components, component graphs route between engines
2. **Queue-based routing**: Simple pop-and-forward logic, no complex coordination
3. **Independent messages**: Each request carries its own routing context
4. **Direct communication**: Components send directly via channels, no central router
5. **Shared engine resources**: Only engines are shared, creating realistic contention

### Component Communication and Thread Safety

#### Any-to-Any Component Communication
```
Component Connection Architecture:
├── ComponentRegistry: Global directory of all components
├── Direct Channels: Each component has input channel
├── Universal Access: All components can access registry
├── Simple Lookup: registry[target_name].input_channel
└── Direct Send: target.input_channel <- message
```

#### Thread-Safe Communication
```
Multiple Components Sending Simultaneously:
├── Web Server → database.input_channel <- message1
├── Cache → database.input_channel <- message2
├── API Gateway → database.input_channel <- message3
└── All at EXACTLY the same time

Result: NO CORRUPTION! Go channels handle this automatically.
```

#### Natural Backpressure
```
When Component Channel Gets Full:
├── Channel capacity: 100 messages
├── Current queue: 100 messages (FULL)
├── Senders block until space available
├── Automatic rate limiting occurs
└── System slows down gracefully (realistic!)
```

#### Component Independence
```
Each Component:
├── Runs in own goroutine
├── Has own input/output channels
├── Accesses ComponentRegistry for routing
├── Makes independent routing decisions
├── No shared mutable state
└── Perfect isolation, no race conditions
```

### State Management
- Each message carries its own routing state
- Components maintain local state for optimization
- System-wide state for health and load monitoring
- Historical data for performance analysis

### Performance Monitoring
- Track routing decisions and their outcomes
- Measure path performance and bottlenecks
- Identify optimal routing strategies
- Detect and resolve routing loops

### Error Handling
- Graceful degradation when components fail
- Automatic rerouting around failed components
- Error propagation and recovery strategies
- Fallback paths for critical operations

## Implementation Considerations

### Graph Definition Format
- JSON/YAML configuration files
- Programmatic graph construction
- Dynamic graph modification
- Version control for graph changes

### Validation and Testing
- Graph connectivity validation
- Deadlock detection
- Performance simulation
- A/B testing for routing strategies

### Monitoring and Debugging
- Real-time routing visualization
- Performance metrics collection
- Error tracking and analysis
- Routing decision audit trails

## Variable Resolution and State Management

### The Variable Resolution Problem

Decision graphs often reference variables in their operations that need actual numeric values for calculations:

```json
{
    "operation": "user_lookup",
    "time_complexity": "O(log n)",
    "variables": {
        "n": "total_users"
    },
    "base_time": "0.001ms"
}
```

**Problem**: To calculate processing time, we need the actual value of `total_users` (e.g., 1,000,000), not the string `"total_users"`.

### Variable Scoping Strategy

#### Global Scope Variables (Shared State)
Variables that represent system-wide state, shared across all component instances:

```
Global Variables (Read by all instances, Written by end nodes):
├── total_users: 10,000
├── total_records: 50,000
├── total_orders: 25,000
├── active_sessions: 1,500
└── system_uptime: 3600s
```

#### Instance Scope Variables (Local State)
Variables that are specific to individual component instances:

```
Instance Variables (Local to each instance):
├── current_connections: 15
├── queue_length: 3
├── cpu_utilization: 0.75
├── memory_usage: 0.60
└── processing_requests: 8
```

### Read-Only Instances + Write-Only End Nodes Architecture

#### Core Principle
> **"Instances only READ variables for calculations, End Nodes only WRITE variables for updates"**

This design eliminates race conditions and provides clean separation of concerns.

#### Instance Processing (Read-Only)
```
Database Instance receives "user_register" request:
1. Reads global total_users = 10,000
2. Calculates: O(log 10,000) = 13.3 operations
3. Processes registration (takes 13.3 * base_time)
4. Routes message to END NODE (no variable updates)
```

#### End Node Processing (Write-Only)
```
End Node receives completed "user_register" operation:
1. Applies side effects: total_users++, total_records++
2. Updates global variables: total_users = 10,001
3. Terminates request flow (no further routing)
```

### Variable Update Lifecycle

#### 1. Request Processing with Current State
```
T=0ms: Registration request starts
├── Instance reads: total_users = 10,000
├── Calculates processing time: O(log 10,000)
├── Processes request using current values
└── Routes to end node when complete
```

#### 2. State Update on Completion
```
T=50ms: Request completes and reaches end node
├── End node applies side effects
├── total_users becomes 10,001
├── total_records becomes 50,001
└── Updates available for next request
```

#### 3. Next Request Uses Updated State
```
T=51ms: Next request starts
├── Instance reads: total_users = 10,001 (updated)
├── Calculates: O(log 10,001) = slightly higher
└── System behavior evolves realistically
```

### Concurrent Request Handling

#### Multiple Instances, Same Base Values
```
T=0ms: 3 registration requests arrive simultaneously
├── Instance 1: Reads total_users = 10,000, processes registration A
├── Instance 2: Reads total_users = 10,000, processes registration B
├── Instance 3: Reads total_users = 10,000, processes registration C
└── All use same baseline (consistent during processing)

T=50-60ms: Requests complete sequentially
├── End node processes A: total_users = 10,001
├── End node processes B: total_users = 10,002
├── End node processes C: total_users = 10,003
└── Sequential updates prevent race conditions
```

### Implementation Strategy

#### End Node with Go Routine Buffer
```go
type EndNode struct {
    updateBuffer chan VariableUpdate
    globalVariables map[string]int
    mutex sync.RWMutex
}

// Sequential processing eliminates race conditions
func (en *EndNode) processUpdates() {
    for update := range en.updateBuffer {
        en.applyUpdate(update)  // Atomic updates
    }
}
```

#### Variable Update Operations
```json
{
  "end_node_operations": {
    "user_register": {
      "side_effects": {
        "increment": ["total_users", "total_records"],
        "update": ["last_user_id"],
        "timestamp": ["last_registration_time"]
      }
    },
    "user_delete": {
      "side_effects": {
        "decrement": ["total_users", "total_records"],
        "update": ["last_deletion_time"]
      }
    }
  }
}
```

## Dynamic Queue Scaling for Load Balancers and Centralized Output

### Overview
Load Balancer queues and Centralized Output queues must **scale dynamically** with the number of component instances to maintain realistic system behavior and prevent bottlenecks.

### Dynamic Load Balancer Queue Scaling

#### **Queue Size Scaling Formula**
```
LB Queue Size = base_queue_size × instance_count × scaling_factor

Where:
- base_queue_size = 1000 (minimum capacity)
- instance_count = current number of instances
- scaling_factor = 1.2-2.0 (buffer for load distribution)

Example: 3 instances = 1000 × 3 × 1.5 = 4500 queue capacity
```

#### **Operations Per Event Cycle Scaling**
```
LB Operations Per Cycle = base_ops_per_cycle × instance_count

Where:
- base_ops_per_cycle = 10 (per instance)
- Total operations = 10 × 3 instances = 30 operations per cycle
```

### Dynamic Centralized Output Queue Scaling

#### **Output Queue Size Scaling Formula**
```
Output Queue Size = base_output_size × instance_count × throughput_factor

Where:
- base_output_size = 500 (per instance)
- throughput_factor = 1.5 (output typically higher than input)

Example: 3 instances = 500 × 3 × 1.5 = 2250 output queue capacity
```

#### **Output Operations Per Cycle Scaling**
```
Output Operations Per Cycle = base_output_ops × instance_count

Where:
- base_output_ops = 5 (per instance)
- Total output operations = 5 × 3 instances = 15 operations per cycle
```

### Dynamic Scaling Implementation

#### **When Instance Added**
1. Calculate new queue sizes based on new instance count
2. Create new channels with larger capacity
3. Migrate existing operations to new channels
4. Update operations per cycle limits

#### **When Instance Removed**
1. Calculate reduced queue sizes
2. Drain operations from old channels
3. Create smaller channels
4. Reduce operations per cycle limits

### Benefits of Dynamic Queue Scaling

#### **Realistic Load Distribution**
- **More instances = larger queues** - handles increased throughput
- **Proportional processing** - operations per cycle scales with capacity
- **Natural backpressure** - queues fill when instances can't keep up

#### **Educational Value**
- **Students see scaling effects** - queue sizes change with instance count
- **Realistic system behavior** - matches production load balancer patterns
- **Performance implications** - understand queue sizing impact

### Benefits of This Approach

#### 1. Race Condition Free
- **Instances only read** - no concurrent writes to shared state
- **End nodes only write** - single writer per variable
- **Sequential updates** through buffered go routine

#### 2. Realistic System Behavior
- **Variable lag** matches real-world eventual consistency
- **Batch updates** at operation completion (like database transactions)
- **Consistent read values** during request processing

#### 3. Educational Value
- Students learn about **eventual consistency** in distributed systems
- Understand **state propagation delays** in real systems
- See how **concurrent operations** handle shared state

#### 4. Implementation Simplicity
- **Clear data flow**: Read → Process → Route → Update
- **No complex synchronization** mechanisms needed
- **Easy to debug** and reason about

### Practical Example: E-commerce Registration Flow

#### Initial State
```
Global Variables:
├── total_users: 50,000
├── total_orders: 125,000
├── active_sessions: 2,500
```

#### Registration Processing
```
T=0ms: User registration request
├── Database instance reads: total_users = 50,000
├── Calculates lookup time: O(log 50,000) = 15.6 operations
├── Processes registration: 15.6 * 0.001ms = 0.0156ms
├── Routes to end node

T=25ms: Registration completes
├── End node receives completion
├── Updates: total_users = 50,001
├── Next request will see updated value
```

#### System Evolution
```
After 1000 registrations:
├── total_users = 51,000
├── User lookups now take: O(log 51,000) = 15.6 operations
├── System performance evolves realistically
├── Students see impact of data growth on performance
```

This variable resolution system provides the foundation for realistic, evolving system behavior while maintaining implementation simplicity and correctness.
