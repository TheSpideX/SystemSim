# Component Design Patterns - Hardware-Adaptive Architecture

## Overview

Components are mini system designs created by combining the 6 base engines (CPU, Memory, Storage, Network Input, Network Output, Coordination) with specific profiles. Each component represents a specialized server optimized for particular workloads, with **hardware-adaptive tick processing** that automatically calculates optimal tick duration based on system scale and hardware capabilities.

## Component-Level Graph Architecture

### Key Principle: Graphs as Pure Data Structures
**Component-level decision graphs are pure data structures stored in Component Load Balancers.** They provide routing lookup information only - no execution logic, no processing capability, just simple routing rules.

### How Component Routing Actually Works

#### **Component Load Balancer Role**
- **Stores** the component-level decision graph (just data storage)
- **Selects** which instance to use (load balancing algorithm)
- **Does NOT execute** any routing logic
- **Does NOT evaluate** any graph conditions

#### **Engine Output Queue Role**
- **Looks up** routing rules from LB's stored graph
- **Evaluates** simple conditions (cache_hit, cache_miss, authenticated, etc.)
- **Pushes** request to next engine's input queue
- **Nothing more complex** than lookup and push

### Component Pattern Categories

#### **1. Linear Flow Patterns (90% of components)**
Simple components with **no decision points** - just linear engine sequences:

```json
Cache Component Pattern = {
    "level": "COMPONENT_LEVEL",
    "complexity": "LINEAR",
    "engines": ["network_input", "memory", "network_output"],
    "graph": {
        "start": {"target": "network_input", "next": "lookup"},
        "lookup": {"target": "memory", "next": "respond"},
        "respond": {"target": "network_output", "next": "end"}
    }
}
```

#### **2. Decision-Based Patterns (10% of components)**
Advanced components with **simple conditional routing**:

```json
Database Component Pattern = {
    "level": "COMPONENT_LEVEL",
    "complexity": "DECISION_BASED",
    "engines": ["network_input", "cpu", "memory", "storage", "network_output"],
    "graph": {
        "parse": {
            "target": "cpu",
            "conditions": {
                "select_query": "check_cache",
                "write_query": "write_storage"
            }
        },
        "check_cache": {
            "target": "memory",
            "conditions": {
                "cache_hit": "respond",
                "cache_miss": "read_storage"
            }
        }
    }
}
```

## Algorithm-Based Load Balancer Instance Selection

### Load Balancing Algorithm Flexibility

**Component Load Balancers** use configurable algorithms for instance selection, with **health as a major factor** in routing decisions.

#### **Supported Load Balancing Algorithms**
```go
type ComponentLoadBalancer struct {
    ComponentID   string
    Algorithm     LoadBalancingAlgorithm // Configurable algorithm
    instances     map[string]*ComponentInstance
}

type LoadBalancingAlgorithm int
const (
    ROUND_ROBIN LoadBalancingAlgorithm = iota
    WEIGHTED
    LEAST_CONNECTIONS
    HEALTH_BASED
    HYBRID
)

func (clb *ComponentLoadBalancer) selectInstance(req *Request) *ComponentInstance {
    switch clb.Algorithm {
    case ROUND_ROBIN:
        return clb.roundRobinSelect()
    case WEIGHTED:
        return clb.weightedSelect()
    case LEAST_CONNECTIONS:
        return clb.leastConnectionsSelect()
    case HEALTH_BASED:
        return clb.healthBasedSelect() // Uses health as major factor
    case HYBRID:
        return clb.hybridSelect() // Health + connections + weights
    }
}
```

#### **Health-Based Selection (Recommended)**
```go
func (clb *ComponentLoadBalancer) healthBasedSelect() *ComponentInstance {
    var bestInstance *ComponentInstance
    var bestHealth float64 = 0

    for _, instance := range clb.instances {
        if instance.Health > bestHealth && instance.Health > 0.5 {
            bestHealth = instance.Health
            bestInstance = instance
        }
    }

    if bestInstance == nil {
        // All instances unhealthy - create new one or use least unhealthy
        return clb.createNewInstanceOrFallback()
    }

    return bestInstance
}
```

#### **Simplified Health-Based Routing (No Circuit Breakers)**
Instead of complex circuit breaker state machines, use **simple health scores**:

```go
// Engine failure just updates health score
func (engine *Engine) recordFailure() {
    engine.Health *= 0.8 // Degrade health on failure
}

func (engine *Engine) recordSuccess() {
    engine.Health = math.Min(1.0, engine.Health + 0.1) // Recover health
}

// Instance selection based on health threshold
func (clb *ComponentLoadBalancer) selectHealthyInstance() *ComponentInstance {
    for _, instance := range clb.instances {
        if instance.Health > 0.5 { // Simple threshold
            return instance
        }
    }

    // If no healthy instances, create new one (auto-scaling)
    return clb.createNewInstance()
}
```

### Auto-Scaling and Dynamic Instance Management

#### **Configurable Auto-Scaling (Modeling Reality)**

**Auto-scaling can be enabled/disabled** to model different deployment scenarios:

```go
type AutoScalingConfig struct {
    Enabled         bool     // Can be disabled for fixed instance scenarios
    Mode            AutoScalingMode

    // Fixed instance mode
    FixedInstances  int      // When auto-scaling disabled

    // Auto-scaling mode
    MinInstances    int      // Never scale below this
    MaxInstances    int      // Never scale above this
    ScaleUpThreshold    float64  // Scale up when avg health < this
    ScaleDownThreshold  float64  // Scale down when avg health > this
    CooldownPeriod      time.Duration
}

type AutoScalingMode int
const (
    FIXED_INSTANCES AutoScalingMode = iota  // 1 or more fixed instances
    AUTO_SCALING                            // Dynamic scaling enabled
)

func (clb *ComponentLoadBalancer) configureAutoScaling(config *AutoScalingConfig) {
    clb.autoScalingConfig = config

    if config.Mode == FIXED_INSTANCES {
        // Create fixed number of instances
        for i := 0; i < config.FixedInstances; i++ {
            instance := clb.createNewInstance()
            clb.addInstance(instance)
        }

        // Disable auto-scaler
        clb.autoScaler = nil

        log.Printf("Component %s configured with %d fixed instances",
            clb.ComponentID, config.FixedInstances)
    } else {
        // Enable auto-scaling
        clb.autoScaler = &AutoScaler{
            componentLB:        clb,
            scaleUpThreshold:   config.ScaleUpThreshold,
            scaleDownThreshold: config.ScaleDownThreshold,
            minInstances:       config.MinInstances,
            maxInstances:       config.MaxInstances,
            cooldownPeriod:     config.CooldownPeriod,
        }

        // Start with minimum instances
        for i := 0; i < config.MinInstances; i++ {
            instance := clb.createNewInstance()
            clb.addInstance(instance)
        }

        log.Printf("Component %s configured with auto-scaling (min: %d, max: %d)",
            clb.ComponentID, config.MinInstances, config.MaxInstances)
    }
}
```

### Educational Scenarios with Different Scaling Configurations

#### **Fixed Instance Scenarios**
```go
// Scenario 1: Single instance (simple learning)
singleInstanceConfig := &AutoScalingConfig{
    Enabled:         false,
    Mode:            FIXED_INSTANCES,
    FixedInstances:  1,
}

// Scenario 2: Multiple fixed instances (load balancing learning)
multiInstanceConfig := &AutoScalingConfig{
    Enabled:         false,
    Mode:            FIXED_INSTANCES,
    FixedInstances:  3,
}

// Scenario 3: High availability setup (reliability learning)
haConfig := &AutoScalingConfig{
    Enabled:         false,
    Mode:            FIXED_INSTANCES,
    FixedInstances:  5,
}
```

#### **Auto-Scaling Scenarios**
```go
// Scenario 4: Basic auto-scaling (elasticity learning)
basicAutoScaling := &AutoScalingConfig{
    Enabled:            true,
    Mode:               AUTO_SCALING,
    MinInstances:       2,
    MaxInstances:       10,
    ScaleUpThreshold:   0.7,   // Scale up when health < 70%
    ScaleDownThreshold: 0.9,   // Scale down when health > 90%
    CooldownPeriod:     30 * time.Second,
}

// Scenario 5: Aggressive auto-scaling (performance learning)
aggressiveAutoScaling := &AutoScalingConfig{
    Enabled:            true,
    Mode:               AUTO_SCALING,
    MinInstances:       1,
    MaxInstances:       50,
    ScaleUpThreshold:   0.8,   // Scale up when health < 80%
    ScaleDownThreshold: 0.95,  // Scale down when health > 95%
    CooldownPeriod:     10 * time.Second,
}
```

#### **Health-Based Auto-Scaling (When Enabled)**

**Automatically scale instances** based on health and load metrics:

```go
type AutoScaler struct {
    componentLB     *ComponentLoadBalancer

    // Scaling thresholds
    scaleUpThreshold    float64  // Scale up when avg health < this
    scaleDownThreshold  float64  // Scale down when avg health > this
    minInstances        int      // Never scale below this
    maxInstances        int      // Never scale above this

    // Scaling cooldown
    lastScaleUp         time.Time
    lastScaleDown       time.Time
    cooldownPeriod      time.Duration
}

func (as *AutoScaler) evaluateScaling() {
    avgHealth := as.calculateAverageHealth()
    currentLoad := as.calculateCurrentLoad()
    instanceCount := len(as.componentLB.instances)

    // Scale up conditions
    if as.shouldScaleUp(avgHealth, currentLoad, instanceCount) {
        as.scaleUp()
    }

    // Scale down conditions
    if as.shouldScaleDown(avgHealth, currentLoad, instanceCount) {
        as.scaleDown()
    }
}

func (as *AutoScaler) shouldScaleUp(avgHealth, currentLoad float64, instanceCount int) bool {
    return avgHealth < as.scaleUpThreshold ||
           currentLoad > 0.8 ||
           (instanceCount < as.maxInstances &&
            time.Since(as.lastScaleUp) > as.cooldownPeriod)
}

func (as *AutoScaler) scaleUp() {
    newInstance := as.componentLB.createNewInstance()
    as.componentLB.addInstance(newInstance)
    as.lastScaleUp = time.Now()

    log.Printf("AutoScaler: Scaled up %s to %d instances",
        as.componentLB.ComponentID, len(as.componentLB.instances))
}
```

#### **Load-Based Instance Selection with Auto-Scaling**

**Combine load balancing with auto-scaling**:

```go
func (clb *ComponentLoadBalancer) selectInstanceWithAutoScaling() *ComponentInstance {
    // Check if auto-scaling is needed
    if clb.autoScaler != nil {
        clb.autoScaler.evaluateScaling()
    }

    // Use algorithm-based selection
    return clb.selectInstance()
}

func (clb *ComponentLoadBalancer) createNewInstance() *ComponentInstance {
    instanceID := fmt.Sprintf("%s-instance-%d", clb.ComponentID, clb.nextInstanceID)
    clb.nextInstanceID++

    instance := &ComponentInstance{
        ID:           instanceID,
        ComponentID:  clb.ComponentID,
        Health:       1.0,  // Start with full health
        InputChannel: make(chan *engines.Operation, 1000),
        engines:      clb.createEngines(),
    }

    // Start instance goroutine
    go instance.run()

    return instance
}
```

#### **Dynamic Queue Scaling with Instance Count**

**Queues automatically scale** with instance count:

```go
func (clb *ComponentLoadBalancer) updateQueueSizes() {
    instanceCount := len(clb.instances)

    // Scale load balancer queue
    newLBQueueSize := clb.baseLBQueueSize * instanceCount * clb.scalingFactor
    clb.resizeLBQueue(newLBQueueSize)

    // Scale centralized output queue
    newOutputQueueSize := clb.baseOutputQueueSize * instanceCount * clb.throughputFactor
    clb.resizeOutputQueue(newOutputQueueSize)

    // Update operations per cycle
    clb.opsPerCycle = clb.baseOpsPerCycle * instanceCount
}

func (clb *ComponentLoadBalancer) onInstanceAdded() {
    clb.updateQueueSizes()
    clb.updateLoadBalancingWeights()
}

func (clb *ComponentLoadBalancer) onInstanceRemoved() {
    clb.updateQueueSizes()
    clb.updateLoadBalancingWeights()
}
```

## Unified Component Architecture

### Engine Goroutine Coordination
Each component instance tracks its engine goroutines at all scales:

```go
type ComponentInstance struct {
    ID                string
    Type              ComponentType

    // Engine tracking (unified architecture)
    EngineCoordination map[EngineType]EngineCoordinator
    TickDuration       time.Duration   // Hardware-adaptive timing

    // Instance-to-engine mapping (similar to LB-to-instance)
    EngineInstances   map[EngineType]*EngineInstance
    EngineHealth      map[EngineType]float64
    EngineMetrics     map[EngineType]EngineMetrics
}
```

## Component Creation Methodology

### Engine Profile Selection
Components are defined by selecting appropriate engine profiles:

1. **Identify primary function** (database, cache, web server, etc.)
2. **Determine resource requirements** (CPU-heavy, memory-intensive, I/O-bound)
3. **Select engine profiles** that match real-world server specifications
4. **Define decision graph** for internal message routing
5. **Configure health monitoring** and backpressure thresholds
6. **Specify coordination requirements** for engine orchestration

### Profile-Based Component Assembly
Each component specifies:
- **Engine types needed** (flexible combination based on function)
- **Required**: Network Input + Network Output + Coordination engines
- **Optional**: CPU, Memory, Storage engines as needed
- **Engine profiles** (specific configurations for each engine)
- **Decision graph** (template-based or custom engine flow)
- **Health thresholds** (when component becomes stressed/overloaded)
- **Goroutine coordination** (engine-to-engine communication patterns)

**Component Composition Rules**:
- **All components**: Must have Network(Input) + Network(Output) + Coordination
- **Stateless components**: May only need CPU (e.g., simple load balancer)
- **Memory-only components**: Exclude Storage engine (e.g., cache)
- **Processing components**: Include CPU + Memory for computation
- **Persistent components**: Include Storage engine (e.g., database)
- **Complex components**: Use all 6 engines for maximum functionality

## Standard Component Patterns

### Database Server Component

**Purpose**: Persistent data storage with ACID properties and complex query processing.

**Hybrid Architecture**: Uses all 6 engines for maximum functionality and realistic database behavior.

**Engine Configuration** (6 engines total):
- **Network Input Engine**: 10Gbps, connection pooling profile
  - Max connections: 200-500 concurrent
  - Connection pooling for efficiency
  - Protocol: Database-specific (PostgreSQL, MySQL)
  - **Goroutine**: Dedicated goroutine for connection handling
- **CPU Engine**: 8 cores, 3.2GHz, heavy processing profile
  - Language: Go/Java (1.0-1.1x multiplier)
  - Operations: SQL parsing (O(n)), query optimization (O(n log n)), joins (O(n²))
  - **Goroutine**: Dedicated goroutine for query processing
- **Memory Engine**: 64GB DDR4, database buffer pool profile
  - Buffer pool: 80% of RAM for data caching
  - Query cache: 10% of RAM for result caching
  - Working memory: 10% of RAM for operations
  - **Goroutine**: Dedicated goroutine for memory management
- **Storage Engine**: NVMe SSD, 50K IOPS, database storage profile
  - Random access optimized for index lookups
  - Sequential access for table scans
  - Write-ahead logging for durability
  - **Goroutine**: Dedicated goroutine for I/O operations
- **Network Output Engine**: Response delivery profile
  - Optimized for result set transmission
  - Connection reuse for efficiency
  - **Goroutine**: Dedicated goroutine for response handling
- **Coordination Engine**: Transaction and consistency management
  - ACID transaction coordination
  - Lock management and deadlock detection
  - Backup and replication coordination
  - **Goroutine**: Dedicated goroutine for orchestration

**Component-Level Decision Graph Template**:
1. Network Input Engine: Receive SQL query
2. Coordination Engine: Begin transaction context
3. CPU Engine: Parse and optimize query
4. Memory Engine: Check buffer pool cache
5. If cache hit: Skip to step 8
6. If cache miss: Storage Engine reads data
7. CPU Engine: Process results
8. Memory Engine: Cache results (if applicable)
9. Coordination Engine: Commit/rollback transaction
10. Network Output Engine: Send response

**Goroutine Coordination**: Each engine runs in its own goroutine, with the Coordination Engine orchestrating the flow between engines using channels and context management.

**Health Thresholds**:
- Healthy: <70% CPU, <80% Memory, <85% Storage IOPS
- Stressed: 70-85% CPU, 80-90% Memory, 85-95% Storage IOPS
- Overloaded: >85% CPU, >90% Memory, >95% Storage IOPS

### Cache Server Component

**Purpose**: High-speed data caching with sub-millisecond response times.

**Engine Configuration** (4 engines total):
- **Network Engine (Input)**: 10Gbps, high-throughput profile
  - Max connections: 10,000+ concurrent
  - Keep-alive connections
  - Protocol: Redis/Memcached
- **CPU Engine**: 4 cores, 4.0GHz, light processing profile
  - Language: C/Go (1.0-1.3x multiplier)
  - Operations: Hash calculations (O(1)), key lookups (O(1))
- **Memory Engine**: 32GB DDR4, high-speed RAM profile
  - 100% RAM allocation (no persistent storage)
  - LRU eviction policy
  - Memory-mapped data structures
- **Network Engine (Output)**: Fast response delivery profile
  - Optimized for small, frequent responses
  - Connection reuse for efficiency
- **Storage Engine**: **None** (volatile memory only)

**Component-Level Decision Graph Template**:
1. Network Engine (Input): Receive cache request
2. CPU Engine: Hash key for lookup
3. Memory Engine: Check cache
4. If hit: Memory Engine retrieves data
5. If miss: Return cache miss signal
6. CPU Engine: Serialize response
7. Network Engine (Output): Send response

**Health Thresholds**:
- Healthy: <60% CPU, <85% Memory, <70% Network
- Stressed: 60-80% CPU, 85-95% Memory, 70-90% Network
- Overloaded: >80% CPU, >95% Memory, >90% Network

### Web Server Component

**Purpose**: HTTP request processing and business logic execution.

**Engine Configuration**:
- **CPU Engine**: 4 cores, 3.5GHz, balanced processing profile
  - Language: Go/Node.js/Java (0.8-1.1x multiplier)
  - Operations: HTTP parsing (O(n)), business logic (O(1) to O(n))
- **Memory Engine**: 16GB DDR4, application memory profile
  - Heap space for application data
  - Session storage
  - Template caching
- **Storage Engine**: SSD, 10K IOPS, application storage profile
  - Static file serving
  - Log file writing
  - Temporary file storage
- **Network Engine**: 1Gbps, HTTP profile
  - Max connections: 1,000-5,000 concurrent
  - HTTP/1.1 and HTTP/2 support
  - Keep-alive connections

**Decision Graph**:
1. Network Engine: Receive HTTP request
2. CPU Engine: Parse HTTP and route request
3. CPU Engine: Execute business logic
4. Memory Engine: Access session/cache data
5. Storage Engine: Read static files (if needed)
6. CPU Engine: Generate response
7. Network Engine: Send HTTP response

**Health Thresholds**:
- Healthy: <70% CPU, <75% Memory, <60% Storage, <80% Network
- Stressed: 70-85% CPU, 75-90% Memory, 60-85% Storage, 80-95% Network
- Overloaded: >85% CPU, >90% Memory, >85% Storage, >95% Network

### Load Balancer Component

**Purpose**: Traffic distribution and health checking for backend services.

**Engine Configuration**:
- **CPU Engine**: 2 cores, 3.0GHz, routing profile
  - Language: C/Go (1.0-1.3x multiplier)
  - Operations: Routing decisions (O(1)), health checks (O(n))
- **Memory Engine**: 8GB DDR4, minimal profile
  - Connection state tracking
  - Health check results
  - Routing tables
- **Storage Engine**: None or minimal (configuration only)
- **Network Engine**: 40Gbps, high-bandwidth profile
  - Max connections: 50,000+ concurrent
  - Connection multiplexing
  - Protocol: HTTP/TCP proxy

**Decision Graph**:
1. Network Engine: Receive incoming request
2. CPU Engine: Select backend server (round-robin, least-connections)
3. CPU Engine: Check backend health status
4. Network Engine: Forward to healthy backend
5. Network Engine: Receive backend response
6. Network Engine: Forward response to client

**Health Thresholds**:
- Healthy: <50% CPU, <60% Memory, <80% Network
- Stressed: 50-70% CPU, 60-80% Memory, 80-95% Network
- Overloaded: >70% CPU, >80% Memory, >95% Network

### Message Queue Component

**Purpose**: Asynchronous message processing and reliable delivery.

**Engine Configuration**:
- **CPU Engine**: 6 cores, 3.2GHz, message processing profile
  - Language: Go/Java (1.0-1.1x multiplier)
  - Operations: Message routing (O(1)), serialization (O(n))
- **Memory Engine**: 16GB DDR4, buffer management profile
  - Message buffers
  - Topic/partition metadata
  - Consumer group state
- **Storage Engine**: SSD, 20K IOPS, sequential write profile
  - Message persistence
  - Commit log storage
  - Offset tracking
- **Network Engine**: 10Gbps, pub/sub profile
  - Max connections: 5,000+ concurrent
  - Producer/consumer protocols
  - Cluster communication

**Decision Graph**:
1. Network Engine: Receive message from producer
2. CPU Engine: Route message to appropriate topic/partition
3. Memory Engine: Buffer message temporarily
4. Storage Engine: Persist message to disk
5. CPU Engine: Update consumer offsets
6. Network Engine: Acknowledge to producer
7. Network Engine: Deliver to consumers

### Search Engine Component

**Purpose**: Full-text search and complex query processing.

**Engine Configuration**:
- **CPU Engine**: 12 cores, 3.2GHz, search processing profile
  - Language: Java/Go (1.0-1.1x multiplier)
  - Operations: Index building (O(n log n)), search queries (O(log n))
- **Memory Engine**: 64GB DDR4, index caching profile
  - Index caching for fast searches
  - Query result caching
  - Aggregation buffers
- **Storage Engine**: NVMe SSD, 100K IOPS, index storage profile
  - Inverted index storage
  - Document storage
  - Shard management
- **Network Engine**: 10Gbps, search API profile
  - RESTful API endpoints
  - Cluster communication
  - Result streaming

**Decision Graph**:
1. Network Engine: Receive search query
2. CPU Engine: Parse and analyze query
3. Memory Engine: Check query cache
4. If cache miss: Storage Engine reads index
5. CPU Engine: Execute search algorithm
6. CPU Engine: Rank and aggregate results
7. Memory Engine: Cache results
8. Network Engine: Return search results

## Custom Component Creation

### Component Factory Pattern
Users can create custom components by:

1. **Selecting base engines** needed for the component
2. **Configuring engine profiles** based on workload requirements
3. **Defining decision graph** for message flow
4. **Setting health thresholds** for monitoring
5. **Testing component behavior** under various loads

### Example: ML Model Server Component

**Engine Configuration**:
- **CPU Engine**: 16 cores + GPU, ML processing profile
  - Language: Python/C++ (0.3x-1.3x multiplier)
  - Operations: Matrix operations (O(n³)), inference (O(n))
- **Memory Engine**: 128GB DDR4, model storage profile
  - Model weights storage
  - Batch processing buffers
  - Result caching
- **Storage Engine**: NVMe SSD, 50K IOPS, model storage profile
  - Model file storage
  - Training data access
  - Checkpoint storage
- **Network Engine**: 10Gbps, API serving profile
  - REST/gRPC API endpoints
  - Model serving protocols
  - Batch request handling

### Component Validation
Each component should be validated against:
- **Real-world benchmarks** for similar systems
- **Resource utilization patterns** under various loads
- **Performance characteristics** compared to actual servers
- **Failure modes** and recovery behavior

## Component Composition Guidelines

### Resource Allocation Principles
- **CPU-bound components**: Emphasize CPU engine capacity
- **Memory-intensive components**: Prioritize Memory engine size
- **I/O-heavy components**: Focus on Storage engine IOPS
- **Network-intensive components**: Maximize Network engine bandwidth

### Performance Optimization
- **Profile-based tuning**: Adjust engine profiles based on workload
- **Health threshold tuning**: Set appropriate stress/overload levels
- **Decision graph optimization**: Minimize unnecessary engine hops
- **Resource balancing**: Ensure no single engine becomes bottleneck

### Scalability Considerations
- **Horizontal scaling**: Multiple component instances
- **Vertical scaling**: Larger engine capacities
- **Load distribution**: Effective load balancing strategies
- **Resource sharing**: Efficient utilization of shared resources

## Educational Progression and Complexity Levels

### Level 1: Linear Components (Beginner)

**90% of components use simple linear flows** - perfect for beginners:

```json
Linear Web Server Component:
{
    "component_type": "web_server",
    "complexity_level": "linear",
    "engine_sequence": [
        {"engine": "network", "operation": "receive_request"},
        {"engine": "cpu", "operation": "parse_request"},
        {"engine": "cpu", "operation": "process_logic"},
        {"engine": "network", "operation": "send_response"}
    ],
    "educational_focus": [
        "Basic request processing",
        "Engine coordination",
        "Performance impact of each step"
    ]
}
```

**Benefits for Beginners:**
- ✅ **Simple to understand** - clear linear progression
- ✅ **Predictable behavior** - no complex routing decisions
- ✅ **Clear performance impact** - see effect of each engine
- ✅ **Easy debugging** - straightforward request path

### Level 2: Decision-Based Components (Intermediate)

**10% of components use decision-based routing** - for intermediate students:

```json
Decision-Based Database Component:
{
    "component_type": "database",
    "complexity_level": "decision_based",
    "decision_graph": {
        "start_node": "receive_query",
        "nodes": {
            "receive_query": {
                "engine": "network",
                "operation": "receive_request",
                "conditions": {
                    "read_query": "check_cache",
                    "write_query": "write_storage",
                    "invalid_query": "error_response"
                }
            },
            "check_cache": {
                "engine": "memory",
                "operation": "cache_lookup",
                "conditions": {
                    "cache_hit": "return_cached",
                    "cache_miss": "read_storage"
                }
            }
        }
    },
    "educational_focus": [
        "Conditional routing",
        "Cache hit/miss patterns",
        "Performance optimization decisions"
    ]
}
```

### Progressive Learning Path

**Students naturally progress** through complexity levels:

```
Learning Progression:
├── Week 1-2: Linear Components
│   ├── Web servers, file servers, simple caches
│   ├── Focus: Basic engine coordination
│   └── Goal: Understand request processing
├── Week 3-4: Decision-Based Components
│   ├── Databases with cache layers
│   ├── API gateways with routing
│   └── Goal: Understand conditional logic
├── Week 5-6: System-Level Flows
│   ├── Multi-component user flows
│   ├── Authentication and authorization
│   └── Goal: Understand system integration
├── Week 7-8: Custom Components
│   ├── Student-designed components
│   ├── Advanced routing algorithms
│   └── Goal: Apply architectural principles
└── Week 9-10: Production Systems
    ├── Large-scale system design
    ├── Performance optimization
    └── Goal: Real-world application
```

### Educational Benefits of Progressive Complexity
- ✅ **Natural learning curve** - students build confidence gradually
- ✅ **Clear milestones** - visible progress through complexity levels
- ✅ **Practical application** - each level builds on previous knowledge
- ✅ **Individual pacing** - students can progress at their own speed
- ✅ **Real-world relevance** - complexity matches industry patterns

## Load Balancer Architecture - Invisible When Not Needed

### Load Balancer Visibility Model

**Load balancers are invisible when only one instance exists** but become useful for auto-scaling or when fixed number of instances > 1:

**This should be documented in detail** as a key architectural principle we discussed.

### Load Balancer Invisible/Visible Implementation

**Load balancers become invisible when only one instance exists** - this is a key architectural optimization:

```go
func (clb *ComponentLoadBalancer) processOperation(op *Operation) error {
    clb.updateVisibility()

    if !clb.isVisible {
        // Load balancer is invisible - direct routing to single instance
        singleInstance := clb.getSingleInstance()
        return singleInstance.processOperation(op)
    }

    // Load balancer is visible - use load balancing algorithm
    instance := clb.selectInstance(op)
    return instance.processOperation(op)
}

func (clb *ComponentLoadBalancer) updateVisibility() {
    instanceCount := len(clb.instances)

    if clb.autoScalingConfig.Enabled {
        // Always visible when auto-scaling enabled (even with 1 instance)
        clb.isVisible = true
        clb.visibilityReason = "auto_scaling_enabled"
    } else if instanceCount > 1 {
        // Visible when multiple fixed instances
        clb.isVisible = true
        clb.visibilityReason = "multiple_instances"
    } else {
        // Invisible when single fixed instance
        clb.isVisible = false
        clb.visibilityReason = "single_instance"
    }
}
```

### Educational Benefits of Invisible Load Balancers

#### **Single Instance Scenario (Load Balancer Invisible)**

**Students focus on component logic, not load balancing complexity**:

```go
// When load balancer is invisible
func (clb *ComponentLoadBalancer) processOperationInvisible(op *Operation) error {
    // No load balancing overhead
    // No algorithm selection
    // No health checking
    // Direct routing to single instance

    return clb.instances[0].processOperation(op)
}

Educational Benefits:
├── Simple direct routing - no complexity
├── Focus on component logic - not load balancing
├── Clear performance characteristics - no LB overhead
├── Natural progression to multi-instance scenarios
└── Students understand when LB is actually needed
```

#### **Multi-Instance Scenario (Load Balancer Visible)**

**Load balancing becomes relevant and educational**:

```go
// When load balancer is visible
func (clb *ComponentLoadBalancer) processOperationVisible(op *Operation) error {
    // Load balancing algorithm becomes relevant
    instance := clb.selectInstanceUsingAlgorithm(op)

    // Health checking becomes important
    if instance.Health < 0.5 {
        instance = clb.findHealthyInstance()
    }

    // Auto-scaling decisions become active
    if clb.autoScaler != nil {
        clb.autoScaler.evaluateScaling()
    }

    return instance.processOperation(op)
}

Educational Benefits:
├── Load balancing algorithms become relevant
├── Health-based routing decisions matter
├── Auto-scaling behavior is observable
├── Production-like complexity emerges naturally
└── Students see when and why LB is needed
```

### Documentation Requirement Implementation

**This architectural principle must be clearly documented**:

```markdown
## Load Balancer Architecture Principle

**Load balancers are invisible when only one instance exists but become useful for auto-scaling or when fixed number of instances > 1.**

### Visibility Rules:
1. **Invisible**: Single fixed instance + no auto-scaling
   - Direct routing to instance
   - No load balancing overhead
   - Educational focus on component logic

2. **Visible**: Multiple instances OR auto-scaling enabled
   - Algorithm-based instance selection
   - Health monitoring and scoring
   - Auto-scaling decisions
   - Production-like behavior

### Implementation:
- Load balancer checks visibility on each operation
- Invisible: Direct routing (O(1) operation)
- Visible: Algorithm-based selection (O(n) operation)
- Visibility can change dynamically based on scaling

### Educational Value:
- Students see when load balancing is actually needed
- Natural progression from simple to complex scenarios
- Clear understanding of load balancer overhead
- Realistic modeling of production deployments
```

### Load Balancer Overhead Analysis

**Students can observe the performance impact** of load balancer visibility:

```go
type LoadBalancerMetrics struct {
    InvisibleOperations int64     // Direct routing count
    VisibleOperations   int64     // Load balanced routing count

    InvisibleLatency    time.Duration  // Average latency when invisible
    VisibleLatency      time.Duration  // Average latency when visible

    OverheadPerOperation time.Duration // LB overhead per operation
}

func (clb *ComponentLoadBalancer) recordOperationMetrics(startTime time.Time, wasVisible bool) {
    latency := time.Since(startTime)

    if wasVisible {
        clb.metrics.VisibleOperations++
        clb.metrics.VisibleLatency = clb.updateAverage(clb.metrics.VisibleLatency, latency)
    } else {
        clb.metrics.InvisibleOperations++
        clb.metrics.InvisibleLatency = clb.updateAverage(clb.metrics.InvisibleLatency, latency)
    }

    // Calculate overhead
    clb.metrics.OverheadPerOperation = clb.metrics.VisibleLatency - clb.metrics.InvisibleLatency
}
```

```go
type ComponentLoadBalancer struct {
    ComponentID     string
    instances       map[string]*ComponentInstance

    // Visibility logic
    isVisible       bool    // Calculated based on instance count and config

    // Configuration
    autoScalingEnabled  bool
    fixedInstanceCount  int
}

func (clb *ComponentLoadBalancer) updateVisibility() {
    instanceCount := len(clb.instances)

    if clb.autoScalingEnabled {
        // Always visible when auto-scaling enabled
        clb.isVisible = true
    } else if clb.fixedInstanceCount > 1 {
        // Visible when multiple fixed instances
        clb.isVisible = true
    } else {
        // Invisible when single fixed instance
        clb.isVisible = false
    }
}

func (clb *ComponentLoadBalancer) processOperation(op *Operation) error {
    if !clb.isVisible {
        // Direct routing to single instance (no load balancing overhead)
        return clb.instances[0].processOperation(op)
    }

    // Use load balancing algorithm
    instance := clb.selectInstance(op)
    return instance.processOperation(op)
}
```

### Educational Benefits of Invisible Load Balancers

#### **Single Instance Scenario (Load Balancer Invisible)**
```
Student Learning:
├── Simple direct routing - no complexity
├── Focus on component logic - not load balancing
├── Clear performance characteristics - no LB overhead
└── Natural progression to multi-instance scenarios

Implementation:
├── No load balancing algorithm needed
├── No health checking overhead
├── Direct instance access
└── Minimal resource usage
```

#### **Multi-Instance Scenario (Load Balancer Visible)**
```
Student Learning:
├── Load balancing algorithms become relevant
├── Health-based routing decisions
├── Auto-scaling behavior
└── Production-like complexity

Implementation:
├── Algorithm-based instance selection
├── Health monitoring and scoring
├── Dynamic scaling decisions
└── Realistic load distribution
```

### Documentation Requirement

**This should be documented in detail** as a key architectural principle:

```markdown
## Load Balancer Architecture Principle

**Load balancers are invisible when only one instance exists but become useful for auto-scaling or when fixed number of instances > 1.**

### Single Instance (LB Invisible):
- Direct routing to instance
- No load balancing overhead
- Simple educational model
- Focus on component logic

### Multiple Instances (LB Visible):
- Algorithm-based selection
- Health monitoring
- Auto-scaling decisions
- Production-like behavior

### Auto-Scaling Enabled (LB Always Visible):
- Dynamic instance management
- Health-based scaling decisions
- Realistic production modeling
- Advanced educational scenarios
```
