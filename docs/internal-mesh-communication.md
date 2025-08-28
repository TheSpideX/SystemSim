# Internal Mesh Communication Protocols

## 🔗 **Service Mesh Communication Architecture**

### **Overview**
The internal service mesh uses **three types of communication** between all services:

1. **gRPC (Synchronous Internal Mesh)**: For immediate response operations requiring strong consistency between services
2. **HTTP/2 (Client Communication)**: For client-facing operations and development/debugging
3. **Redis Pub/Sub + Streams (Asynchronous & Real-time)**: For fire-and-forget operations, real-time data streaming, and heavy processing

All services are **equal mesh participants**, including the API Gateway, with each service maintaining all three communication channels.

### **Mesh Participants**
```
Full Mesh Network (Three-Layer Communication):

API Gateway (Port 8000):
├── HTTP/2 Server: Client interface (web apps, mobile apps)
├── gRPC Client Pool: Internal mesh communication
└── Redis Pub/Sub: System events and real-time data

Auth Service:
├── HTTP/2 Server (Port 9001): Client auth operations (register, login, profile)
├── gRPC Server (Port 9000): Internal mesh (token validation, permissions)
└── Redis Pub/Sub: Login events, background email processing

Project Service:
├── HTTP/2 Server (Port 10001): Client project operations (CRUD, management)
├── gRPC Server (Port 10000): Internal mesh (project access, validation)
└── Redis Pub/Sub: Project events, collaboration notifications

Simulation Service:
├── HTTP/2 Server (Port 11001): Client simulation operations (run, configure)
├── gRPC Server (Port 11000): Internal mesh (status, execution control)
└── Redis Streams: Real-time simulation data streaming, results broadcasting

Each service maintains ALL THREE communication channels simultaneously
```

## 🏗️ **Three-Layer Communication Architecture**

### **Layer 1: gRPC Internal Mesh (Synchronous)**
- **Purpose**: High-performance service-to-service communication
- **Protocol**: gRPC over HTTP/2 with Protocol Buffers
- **Port Pattern**: X000 (Auth: 9000, Project: 10000, Simulation: 11000)
- **Use Cases**:
  - Token validation between services
  - Permission checking for authorization
  - User context retrieval
  - Service health checks
  - Real-time queries requiring immediate response
- **Connection Management**: Dynamic connection pools (5-20 connections per service pair)
- **Performance**: 2-10x faster than HTTP/JSON for internal calls

### **Layer 2: HTTP/2 Client Interface (Client Communication)**
- **Purpose**: Client-facing operations and development/debugging
- **Protocol**: HTTP/2 + JSON
- **Port Pattern**: X001 (Auth: 9001, Project: 10001, Simulation: 11001)
- **Use Cases**:
  - User registration and authentication
  - Profile and account management
  - Project CRUD operations
  - Simulation configuration and control
  - Admin operations and management
- **Features**: CORS, rate limiting, security headers, request validation
- **Development**: Always enabled for debugging; can be disabled in production

### **Layer 3: Redis Pub/Sub + Streams (Asynchronous & Real-time)**
- **Purpose**: Event publishing, background tasks, and real-time data streaming
- **Protocol**: Redis Pub/Sub for events, Redis Streams for real-time data
- **Use Cases**:
  - **Events**: Login/logout, permission changes, system announcements
  - **Background Tasks**: Email sending, data processing, cleanup jobs
  - **Real-time Streaming**: Live simulation data, progress updates, metrics
  - **Notifications**: User notifications, system alerts, collaboration updates
- **Patterns**: Publisher/Subscriber for events, Producer/Consumer for streams
- **Scalability**: Horizontal scaling with Redis Cluster

## 🎯 **gRPC Service Contracts (Layer 1: Internal Mesh)**

### **API Gateway gRPC Interface (Port 8000)**

#### **Service Definition**
```protobuf
syntax = "proto3";

package gateway;

service GatewayService {
  // Client session management
  rpc RegisterClient(RegisterClientRequest) returns (RegisterClientResponse);
  rpc UnregisterClient(UnregisterClientRequest) returns (UnregisterClientResponse);

  // WebSocket message routing
  rpc RouteMessage(RouteMessageRequest) returns (RouteMessageResponse);
  rpc BroadcastMessage(BroadcastMessageRequest) returns (BroadcastMessageResponse);

  // Client connection info
  rpc GetConnectedClients(GetConnectedClientsRequest) returns (GetConnectedClientsResponse);
  rpc GetClientInfo(GetClientInfoRequest) returns (GetClientInfoResponse);

  // Health check
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

#### **Message Definitions**
```protobuf
message RegisterClientRequest {
  string client_id = 1;
  string user_id = 2;
  repeated string subscriptions = 3;
  string connection_type = 4; // "websocket" or "http"
}

message RegisterClientResponse {
  bool success = 1;
  string session_id = 2;
  string message = 3;
}

message RouteMessageRequest {
  string target_user_id = 1;
  string message_type = 2;
  bytes payload = 3;
  string source_service = 4;
}

message RouteMessageResponse {
  bool delivered = 1;
  int32 client_count = 2;
  string message = 3;
}
```

### **Auth Service gRPC Interface (Port 9000)**

#### **Service Definition**
```protobuf
syntax = "proto3";

package auth;

service AuthService {
  // Token validation for other services
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  
  // Get user context with permissions
  rpc GetUserContext(GetUserContextRequest) returns (GetUserContextResponse);
  
  // Check specific permissions
  rpc CheckPermission(CheckPermissionRequest) returns (CheckPermissionResponse);
  
  // Validate user session
  rpc ValidateSession(ValidateSessionRequest) returns (ValidateSessionResponse);
  
  // Get user roles and permissions
  rpc GetUserPermissions(GetUserPermissionsRequest) returns (GetUserPermissionsResponse);
  
  // Health check
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

#### **Message Definitions**
```protobuf
message ValidateTokenRequest {
  string token = 1;
  string calling_service = 2;
  string request_id = 3;
}

message ValidateTokenResponse {
  bool valid = 1;
  string user_id = 2;
  string email = 3;
  bool is_admin = 4;
  string session_id = 5;
  repeated string permissions = 6;
  int64 expires_at = 7;
}

message GetUserContextRequest {
  string user_id = 1;
  string calling_service = 2;
}

message GetUserContextResponse {
  string user_id = 1;
  string email = 2;
  string name = 3;
  repeated string roles = 4;
  repeated string permissions = 5;
  bool is_active = 6;
  int64 last_login = 7;
}

message CheckPermissionRequest {
  string user_id = 1;
  string permission = 2;
  string resource_id = 3;
  string calling_service = 4;
}

message CheckPermissionResponse {
  bool allowed = 1;
  string reason = 2;
}
```

### **Project Service gRPC Interface (Port 10000)**

#### **Service Definition**
```protobuf
syntax = "proto3";

package project;

service ProjectService {
  // Project CRUD operations
  rpc CreateProject(CreateProjectRequest) returns (CreateProjectResponse);
  rpc GetProject(GetProjectRequest) returns (GetProjectResponse);
  rpc UpdateProject(UpdateProjectRequest) returns (UpdateProjectResponse);
  rpc DeleteProject(DeleteProjectRequest) returns (DeleteProjectResponse);
  rpc ListProjects(ListProjectsRequest) returns (ListProjectsResponse);
  
  // Project sharing and permissions
  rpc ShareProject(ShareProjectRequest) returns (ShareProjectResponse);
  rpc GetProjectPermissions(GetProjectPermissionsRequest) returns (GetProjectPermissionsResponse);
  
  // Template management
  rpc GetTemplate(GetTemplateRequest) returns (GetTemplateResponse);
  rpc ListTemplates(ListTemplatesRequest) returns (ListTemplatesResponse);
  
  // Collaboration features
  rpc JoinCollaboration(JoinCollaborationRequest) returns (JoinCollaborationResponse);
  rpc LeaveCollaboration(LeaveCollaborationRequest) returns (LeaveCollaborationResponse);
  
  // Health check
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

#### **Message Definitions**
```protobuf
message CreateProjectRequest {
  string user_id = 1;
  string name = 2;
  string description = 3;
  string template_id = 4;
  map<string, string> configuration = 5;
  bool is_public = 6;
}

message CreateProjectResponse {
  string project_id = 1;
  string name = 2;
  string status = 3;
  int64 created_at = 4;
  string template_used = 5;
}

message GetProjectRequest {
  string project_id = 1;
  string user_id = 2;
  bool include_configuration = 3;
}

message GetProjectResponse {
  string project_id = 1;
  string name = 2;
  string description = 3;
  string owner_id = 4;
  map<string, string> configuration = 5;
  repeated string collaborators = 6;
  int64 created_at = 7;
  int64 updated_at = 8;
  string status = 9;
}
```

### **Simulation Service gRPC Interface (Port 11000)**

#### **Service Definition**
```protobuf
syntax = "proto3";

package simulation;

service SimulationService {
  // Simulation lifecycle
  rpc StartSimulation(StartSimulationRequest) returns (StartSimulationResponse);
  rpc StopSimulation(StopSimulationRequest) returns (StopSimulationResponse);
  rpc GetSimulationStatus(GetSimulationStatusRequest) returns (GetSimulationStatusResponse);
  rpc GetSimulationResults(GetSimulationResultsRequest) returns (GetSimulationResultsResponse);
  
  // Real-time simulation data
  rpc GetLiveMetrics(GetLiveMetricsRequest) returns (GetLiveMetricsResponse);
  rpc GetPerformanceAnalysis(GetPerformanceAnalysisRequest) returns (GetPerformanceAnalysisResponse);
  
  // Simulation configuration
  rpc ValidateConfiguration(ValidateConfigurationRequest) returns (ValidateConfigurationResponse);
  rpc GetSimulationHistory(GetSimulationHistoryRequest) returns (GetSimulationHistoryResponse);
  
  // Health check
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

#### **Message Definitions**
```protobuf
message StartSimulationRequest {
  string user_id = 1;
  string project_id = 2;
  map<string, string> configuration = 3;
  repeated string components = 4;
  int32 duration_seconds = 5;
  bool enable_real_time = 6;
}

message StartSimulationResponse {
  string simulation_id = 1;
  string status = 2;
  int64 started_at = 3;
  string websocket_channel = 4;
  int32 estimated_duration = 5;
}

message GetSimulationStatusRequest {
  string simulation_id = 1;
  string user_id = 2;
}

message GetSimulationStatusResponse {
  string simulation_id = 1;
  string status = 2;
  int32 progress_percentage = 3;
  int64 started_at = 4;
  int64 estimated_completion = 5;
  repeated string active_components = 6;
  map<string, double> current_metrics = 7;
}
```

## 🔄 **Redis Pub/Sub + Streams Communication (Layer 3: Asynchronous & Real-time)**

### **Dual Redis Architecture**
```
Redis Cluster Configuration:

1. Redis Pub/Sub (Events & Notifications):
   - Purpose: Event broadcasting, notifications, background tasks
   - Pattern: Publisher/Subscriber
   - Persistence: Non-persistent (fire-and-forget)
   - Use Cases: Login events, system announcements, background jobs

2. Redis Streams (Real-time Data Streaming):
   - Purpose: High-throughput real-time data streaming
   - Pattern: Producer/Consumer with consumer groups
   - Persistence: Persistent with configurable retention
   - Use Cases: Live simulation data, metrics streaming, real-time collaboration
```

### **Channel Organization**

#### **Event Channels (Redis Pub/Sub - Low-Medium Frequency)**
```
Event Broadcasting Channels:
├── auth:events:login → User login events (100-1K/hour)
├── auth:events:logout → User logout events (100-1K/hour)
├── auth:events:permission_changed → Permission updates (10-100/hour)
├── project:events:created → Project creation events (10-100/hour)
├── project:events:shared → Project sharing events (10-100/hour)
├── project:events:updated → Project modification events (100-1K/hour)
├── simulation:events:started → Simulation start events (10-100/hour)
├── simulation:events:completed → Simulation completion events (10-100/hour)
└── system:announcements → System-wide messages (1-10/hour)
```

#### **Real-time Streaming Channels (Redis Streams - High Frequency)**
```
High-Throughput Data Streams:
├── simulation:stream:{simulation_id} → Live simulation metrics (1K-100K/sec)
├── collaboration:stream:{project_id} → Real-time editing updates (100-10K/sec)
├── monitoring:stream:metrics → System performance data (1K-10K/sec)
├── analytics:stream:events → User interaction tracking (100-10K/sec)
├── notifications:stream:{user_id} → User-specific real-time notifications (10-1K/sec)
└── system:stream:health → Service health monitoring (100-1K/sec)
```

#### **Background Processing Channels (Redis Pub/Sub - Medium Frequency)**
```
Async Work Queues:
├── async:email:queue → Background email sending (100-1K/hour)
├── async:reports:queue → Report generation (10-100/hour)
├── async:exports:queue → Data export processing (10-100/hour)
├── async:cleanup:queue → Background cleanup tasks (10-100/hour)
├── async:analytics:queue → Analytics processing (100-1K/hour)
└── async:notifications:queue → Notification delivery (100-10K/hour)
```

### **Service Subscription Patterns**

#### **API Gateway Subscriptions**
```go
// API Gateway subscribes to ALL channels for WebSocket broadcasting
func (ag *APIGateway) initializeRedisSubscriptions() {
    ag.pubsub = ag.redis.PSubscribe(
        "auth:events:*",           // All auth events
        "project:events:*",        // All project events
        "simulation:events:*",     // All simulation events
        "simulation:data:*",       // Real-time simulation data
        "notifications:user:*",    // All user notifications
        "system:*",                // System-wide messages
    )

    // Handle messages and route to WebSocket clients
    go ag.handleRedisMessages()
}
```

#### **Auth Service Subscriptions**
```go
func (auth *AuthService) initializeRedisSubscriptions() {
    auth.pubsub = auth.redis.Subscribe(
        "async:email:queue",       // Process email sending
        "system:announcements",    // System messages
    )

    // Publish auth events
    auth.publisher = auth.redis
}

// Publishing auth events
func (auth *AuthService) publishLoginEvent(userID string) {
    event := AuthEvent{
        Type:      "login",
        UserID:    userID,
        Timestamp: time.Now(),
    }
    auth.publisher.Publish("auth:events:login", event)
}
```

#### **Project Service Subscriptions**
```go
func (project *ProjectService) initializeRedisSubscriptions() {
    project.pubsub = project.redis.Subscribe(
        "auth:events:permission_changed", // User permission updates
        "async:reports:queue",             // Report generation
        "async:exports:queue",             // Data export processing
        "system:announcements",            // System messages
    )
}

// Publishing project events
func (project *ProjectService) publishProjectCreated(projectID, userID string) {
    event := ProjectEvent{
        Type:      "created",
        ProjectID: projectID,
        UserID:    userID,
        Timestamp: time.Now(),
    }
    project.publisher.Publish("project:events:created", event)
}
```

#### **Simulation Service Subscriptions**
```go
func (sim *SimulationService) initializeRedisSubscriptions() {
    sim.pubsub = sim.redis.Subscribe(
        "project:events:updated",    // Project configuration changes
        "async:cleanup:queue",       // Cleanup old simulation data
        "async:analytics:queue",     // Analytics processing
        "system:announcements",      // System messages
    )
}

// Publishing high-frequency simulation data using Redis Streams
func (sim *SimulationService) publishSimulationData(simulationID string, metrics SimulationMetrics) {
    streamKey := fmt.Sprintf("simulation:stream:%s", simulationID)

    // Use Redis Streams for high-throughput real-time data
    sim.redisClient.XAdd(ctx, &redis.XAddArgs{
        Stream: streamKey,
        Values: map[string]interface{}{
            "timestamp":     time.Now().UnixNano(),
            "cpu_usage":     metrics.CPUUsage,
            "memory_usage":  metrics.MemoryUsage,
            "network_io":    metrics.NetworkIO,
            "active_nodes":  metrics.ActiveNodes,
            "throughput":    metrics.Throughput,
            "latency":       metrics.Latency,
            "error_rate":    metrics.ErrorRate,
        },
        MaxLen: 10000, // Keep last 10K entries
    })
}
```

### **Real-time Simulation Data Streaming**

#### **Redis Streams Architecture for Simulation**
```go
// Simulation Service: High-frequency data producer
type SimulationStreamer struct {
    redisClient *redis.Client
    streamKey   string
    batchSize   int
    flushInterval time.Duration
}

// Stream real-time simulation metrics (1K-100K messages/second)
func (s *SimulationStreamer) StreamMetrics(simulationID string) {
    streamKey := fmt.Sprintf("simulation:stream:%s", simulationID)

    ticker := time.NewTicker(100 * time.Millisecond) // 10 updates/second
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            metrics := s.collectCurrentMetrics()

            // Batch multiple metrics for efficiency
            s.redisClient.XAdd(ctx, &redis.XAddArgs{
                Stream: streamKey,
                Values: map[string]interface{}{
                    "timestamp":        time.Now().UnixNano(),
                    "nodes_active":     metrics.NodesActive,
                    "nodes_failed":     metrics.NodesFailed,
                    "pods_running":     metrics.PodsRunning,
                    "services_healthy": metrics.ServicesHealthy,
                    "cpu_total":        metrics.CPUTotal,
                    "memory_total":     metrics.MemoryTotal,
                    "network_in":       metrics.NetworkIn,
                    "network_out":      metrics.NetworkOut,
                    "requests_per_sec": metrics.RequestsPerSec,
                    "avg_latency":      metrics.AvgLatency,
                    "error_rate":       metrics.ErrorRate,
                    "simulation_step":  metrics.SimulationStep,
                },
                MaxLen: 50000, // Keep last 50K entries (~1.4 hours at 10/sec)
            })
        }
    }
}
```

#### **API Gateway: Real-time Data Consumer & WebSocket Broadcaster**
```go
// API Gateway consumes simulation streams and broadcasts to WebSocket clients
type SimulationStreamConsumer struct {
    redisClient   *redis.Client
    wsManager     *WebSocketManager
    consumerGroup string
}

func (c *SimulationStreamConsumer) ConsumeSimulationStream(simulationID string) {
    streamKey := fmt.Sprintf("simulation:stream:%s", simulationID)
    consumerGroup := "api-gateway-consumers"
    consumerName := fmt.Sprintf("gateway-%s", os.Getenv("INSTANCE_ID"))

    // Create consumer group if not exists
    c.redisClient.XGroupCreate(ctx, streamKey, consumerGroup, "0")

    for {
        // Read from stream with consumer group
        streams := c.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
            Group:    consumerGroup,
            Consumer: consumerName,
            Streams:  []string{streamKey, ">"},
            Count:    100,  // Process up to 100 messages at once
            Block:    100 * time.Millisecond,
        }).Val()

        for _, stream := range streams {
            for _, message := range stream.Messages {
                // Convert Redis stream message to WebSocket format
                wsMessage := c.formatSimulationUpdate(message.Values)

                // Broadcast to all connected clients watching this simulation
                c.wsManager.BroadcastToSimulation(simulationID, wsMessage)

                // Acknowledge message processing
                c.redisClient.XAck(ctx, streamKey, consumerGroup, message.ID)
            }
        }
    }
}
```

#### **Client WebSocket Integration**
```javascript
// Frontend receives real-time simulation data via WebSocket
const simulationSocket = new WebSocket('ws://localhost:8000/simulation/stream');

simulationSocket.onmessage = (event) => {
    const data = JSON.parse(event.data);

    if (data.type === 'simulation_metrics') {
        // Update real-time charts and dashboards
        updateCPUChart(data.cpu_total);
        updateMemoryChart(data.memory_total);
        updateNetworkChart(data.network_in, data.network_out);
        updateLatencyChart(data.avg_latency);
        updateErrorRateChart(data.error_rate);

        // Update simulation step counter
        updateSimulationProgress(data.simulation_step);
    }
};
```

#### **Performance Characteristics**
```
Real-time Simulation Streaming Performance:

Data Production (Simulation Service):
├── Frequency: 10-100 updates/second per simulation
├── Batch Size: 1-100 metrics per update
├── Throughput: 1K-100K messages/second (multiple simulations)
├── Latency: <10ms from generation to Redis
└── Persistence: 50K messages (~1-5 hours of data)

Data Consumption (API Gateway):
├── Consumer Groups: Horizontal scaling support
├── Processing: 100 messages per batch
├── WebSocket Broadcast: <5ms to connected clients
├── Client Updates: Real-time dashboard updates
└── Acknowledgment: Guaranteed message processing

End-to-End Latency:
├── Simulation → Redis: <10ms
├── Redis → API Gateway: <5ms
├── API Gateway → WebSocket: <5ms
├── WebSocket → Client: <10ms
└── Total: <30ms for real-time updates
```

## 🔄 **Connection Pool Management (gRPC)**

### **Connection Pool Architecture**
```go
type ServiceMeshClient struct {
    serviceName    string
    connections    map[string]*ConnectionPool
    discovery      ServiceDiscovery
    healthChecker  HealthChecker
}

type ConnectionPool struct {
    targetService  string
    targetAddress  string
    minConnections int    // 5 (always warm)
    maxConnections int    // 20 (scale up under load)
    currentConns   []*grpc.ClientConn
    loadBalancer   LoadBalancer
    circuitBreaker CircuitBreaker
    metrics        PoolMetrics
}
```

### **Multiple Instance Communication**

#### **Service Instance Discovery**
```
Example Scenario: 3 API Gateway + 2 Auth Service + 1 Project + 1 Simulation

Service Registry (Redis):
├── services:api-gateway:instance-1 → {"grpc_port": 8000, "http_port": 8001, "status": "healthy"}
├── services:api-gateway:instance-2 → {"grpc_port": 8000, "http_port": 8001, "status": "healthy"}
├── services:api-gateway:instance-3 → {"grpc_port": 8000, "http_port": 8001, "status": "healthy"}
├── services:auth-service:instance-1 → {"grpc_port": 9000, "http_port": 9001, "status": "healthy"}
├── services:auth-service:instance-2 → {"grpc_port": 9000, "http_port": 9001, "status": "healthy"}
├── services:project-service:instance-1 → {"grpc_port": 10000, "http_port": 10001, "status": "healthy"}
└── services:simulation-service:instance-1 → {"grpc_port": 11000, "http_port": 11001, "status": "healthy"}
```

#### **Connection Distribution Strategy**
```
Each service instance connects to ALL instances of ALL other services:

Auth Service Instance 1 gRPC Connections:
├── api-gateway:instance-1 (20 connections)
├── api-gateway:instance-2 (20 connections)
├── api-gateway:instance-3 (20 connections)
├── project-service:instance-1 (20 connections)
├── simulation-service:instance-1 (20 connections)
└── Total: 100 gRPC connections

API Gateway Instance 1 gRPC Connections:
├── auth-service:instance-1 (20 connections)
├── auth-service:instance-2 (20 connections)
├── project-service:instance-1 (20 connections)
├── simulation-service:instance-1 (20 connections)
└── Total: 80 gRPC connections

System Total: ~540 gRPC connections (7 instances × average 77 connections each)
```

#### **Load Balancing Across Instances**
```go
type MultiInstanceClient struct {
    serviceName     string
    instances       map[string]*ConnectionPool  // instance-id -> connection pool
    loadBalancer    *RoundRobinLB
    discovery       ServiceDiscovery
    healthChecker   HealthChecker
}

// Make gRPC call with automatic load balancing
func (mic *MultiInstanceClient) Call(ctx context.Context, method string, req interface{}) (interface{}, error) {
    // Get healthy instances
    healthyInstances := mic.getHealthyInstances()
    if len(healthyInstances) == 0 {
        return nil, errors.New("no healthy instances available")
    }

    // Round-robin selection
    selectedInstance := mic.loadBalancer.Next(healthyInstances)

    // Get connection from selected instance pool
    conn, err := selectedInstance.GetConnection()
    if err != nil {
        // Try next instance on failure
        return mic.retryWithNextInstance(ctx, method, req, selectedInstance)
    }

    // Make gRPC call
    return mic.makeGRPCCall(ctx, conn, method, req)
}
```

### **Dynamic Connection Scaling Logic**

#### **Traffic-Based Auto-Scaling Algorithm**
```go
func (pool *DynamicConnectionPool) autoScale() {
    metrics := pool.getMetrics()

    // Scale Up Decision
    if pool.shouldScaleUp(metrics) && pool.canScaleUp() {
        if pool.globalScaler.requestConnectionIncrease(pool) {
            pool.addConnection()
            pool.metrics.LastScaleAction = time.Now()
            log.Printf("Scaled up: %s now has %d connections",
                       pool.instanceID, len(pool.currentConns))
        }
    }

    // Scale Down Decision (with delay to prevent thrashing)
    if pool.shouldScaleDown(metrics) && pool.canScaleDown() {
        pool.removeConnection()
        pool.globalScaler.releaseConnection()
        pool.metrics.LastScaleAction = time.Now()
        log.Printf("Scaled down: %s now has %d connections",
                   pool.instanceID, len(pool.currentConns))
    }
}

func (pool *DynamicConnectionPool) shouldScaleUp(metrics *PoolMetrics) bool {
    return (metrics.ConnectionUtilization > 0.8 ||     // >80% utilization
            metrics.QueueDepth > 10 ||                 // Queue backing up
            metrics.RequestsPerSecond > 100 ||         // High traffic
            metrics.AverageLatency > 50*time.Millisecond) && // High latency
           metrics.ErrorRate < 0.05                    // Service is healthy
}

func (pool *DynamicConnectionPool) shouldScaleDown(metrics *PoolMetrics) bool {
    timeSinceLastScale := time.Since(metrics.LastScaleAction)
    return metrics.ConnectionUtilization < 0.3 &&      // <30% utilization
           metrics.QueueDepth < 2 &&                   // Low queue depth
           timeSinceLastScale > 5*time.Minute &&       // 5 min delay
           len(pool.currentConns) > pool.minConnections // Above minimum
}
```

#### **Global Connection Management**
```go
type GlobalConnectionScaler struct {
    maxTotalConnections int                    // 1000 per service instance
    currentTotal        int                    // Current total across all pools
    priorityServices    []string               // ["auth-service", "project-service", ...]
    servicePools        map[string][]*DynamicConnectionPool
    rebalanceThreshold  float64                // 0.9 (90% of max)
    mutex              sync.RWMutex
}

func (gcs *GlobalConnectionScaler) requestConnectionIncrease(requestingPool *DynamicConnectionPool) bool {
    gcs.mutex.Lock()
    defer gcs.mutex.Unlock()

    // Simple case: under global limit
    if gcs.currentTotal < gcs.maxTotalConnections {
        gcs.currentTotal++
        return true
    }

    // Complex case: need to rebalance
    return gcs.rebalanceConnections(requestingPool)
}

func (gcs *GlobalConnectionScaler) rebalanceConnections(requestingPool *DynamicConnectionPool) bool {
    requestingService := requestingPool.serviceName
    requestingPriority := gcs.getServicePriority(requestingService)

    // Find lower priority services with excess connections
    for priority := len(gcs.priorityServices) - 1; priority > requestingPriority; priority-- {
        serviceName := gcs.priorityServices[priority]

        if pool := gcs.findPoolWithExcessConnections(serviceName); pool != nil {
            pool.forceScaleDown()
            gcs.currentTotal-- // Released one connection
            gcs.currentTotal++ // Allocated to requesting pool

            log.Printf("Rebalanced: Moved connection from %s to %s",
                       serviceName, requestingService)
            return true
        }
    }

    log.Printf("Connection request denied: No available connections for %s",
               requestingService)
    return false
}

// Service priority order (0 = highest priority)
func (gcs *GlobalConnectionScaler) getServicePriority(serviceName string) int {
    for i, service := range gcs.priorityServices {
        if service == serviceName {
            return i
        }
    }
    return len(gcs.priorityServices) // Lowest priority for unknown services
}
```

#### **Connection Efficiency Metrics**
```
Real-World Scaling Example:

Normal Traffic (9 AM - 5 PM):
├── Auth Service: 8-12 connections per target (medium priority calls)
├── Project Service: 5-8 connections per target (steady CRUD operations)
├── Simulation Service: 5-6 connections per target (low simulation activity)
├── API Gateway: 6-10 connections per target (steady client requests)
└── Total System: ~200-300 connections (vs 540 static)

Peak Traffic (Product Launch):
├── Auth Service: 18-20 connections per target (high authentication load)
├── Project Service: 15-18 connections per target (many new projects)
├── Simulation Service: 12-15 connections per target (demo simulations)
├── API Gateway: 16-20 connections per target (high client load)
└── Total System: ~400-500 connections (scales automatically)

Low Traffic (Nights/Weekends):
├── Auth Service: 5 connections per target (minimum warm connections)
├── Project Service: 5 connections per target (minimum warm connections)
├── Simulation Service: 5 connections per target (minimum warm connections)
├── API Gateway: 5 connections per target (minimum warm connections)
└── Total System: ~105 connections (80% reduction from static)

Scaling Performance:
├── Scale Up Time: <500ms (add new connection)
├── Scale Down Time: 5 minutes (prevent thrashing)
├── Rebalancing Time: <1 second (move connection between services)
├── Health Detection: <30 seconds (remove unhealthy connections)
└── Memory Efficiency: 60-80% improvement over static pools
```

## 🎯 **Service Communication Patterns**

### **Two-Type Communication Examples**

#### **Authentication Flow (gRPC + Redis)**
```
Synchronous (gRPC):
1. API Gateway receives client request with JWT token
2. API Gateway → Auth Service (gRPC): ValidateToken(token, "api-gateway")
3. Auth Service validates token and returns user context immediately
4. API Gateway → Target Service (gRPC): Request with user context
5. Target Service processes request with validated user info

Asynchronous (Redis):
6. Auth Service → Redis: Publish("auth:events:login", {user_id, timestamp})
7. API Gateway subscribes to "auth:events:*" and broadcasts to WebSocket clients
8. Other services can subscribe to auth events for analytics, logging, etc.
```

#### **Project Creation Flow (gRPC + Redis)**
```
Synchronous (gRPC):
1. API Gateway → Auth Service (gRPC): ValidateToken(token)
2. API Gateway → Project Service (gRPC): CreateProject(user_id, project_data)
3. Project Service → Auth Service (gRPC): CheckPermission(user_id, "project:create")
4. Project Service creates project and returns project_id immediately

Asynchronous (Redis):
5. Project Service → Redis: Publish("project:events:created", {project_id, user_id})
6. API Gateway receives event and broadcasts to WebSocket clients
7. Analytics Service processes project creation for metrics
8. Notification Service sends welcome email via "async:email:queue"
```

#### **Real-time Simulation Flow (gRPC + Redis)**
```
Synchronous (gRPC):
1. API Gateway → Project Service (gRPC): GetProject(project_id)
2. API Gateway → Simulation Service (gRPC): StartSimulation(project_id, config)
3. Simulation Service → Auth Service (gRPC): CheckPermission(user_id, "simulation:start")
4. Simulation Service returns simulation_id immediately

Asynchronous (Redis):
5. Simulation Service → Redis: Publish("simulation:events:started", {simulation_id})
6. Simulation Service → Redis: Publish("simulation:data:{id}", metrics) [100+ times/second]
7. API Gateway subscribes to "simulation:data:*" and streams to WebSocket clients
8. Heavy analytics processing via "async:analytics:queue"
```

#### **Multiple Instance Load Balancing**
```
Example: 3 API Gateway instances calling 2 Auth Service instances

API Gateway Instance 1:
├── Receives client request
├── Round-robin selects Auth Service Instance 2
├── Makes gRPC call: ValidateToken()
├── Gets immediate response
└── Continues processing

If Auth Service Instance 2 fails:
├── Circuit breaker detects failure
├── Automatically routes to Auth Service Instance 1
├── Client request continues without interruption
└── Failed instance is marked unhealthy in service registry
```

## 📊 **Performance Specifications**

### **gRPC Performance with Dynamic Scaling (Synchronous)**
```
Performance Targets:
├── API Gateway Calls: <10ms (client-facing operations)
├── Auth Service Calls: <5ms (most critical - token validation)
├── Project Service Calls: <10ms (CRUD operations)
├── Simulation Service Calls: <15ms (complex operations)
├── Connection Pool Utilization: 60-80% optimal (dynamic scaling maintains this)
├── Circuit Breaker Threshold: 50% error rate over 10 seconds
├── Health Check Interval: 30 seconds
└── Connection Scaling: <500ms to add connections, 5min delay to remove

Dynamic Scaling Performance:
├── Scale Up Trigger Latency: <100ms (detect high utilization)
├── New Connection Establishment: <500ms
├── Load Rebalancing: <1 second across instances
├── Global Connection Rebalancing: <1 second between services
├── Scale Down Evaluation: Every 30 seconds
├── Scale Down Execution: After 5 minute delay
└── Connection Pool Efficiency: 60-80% utilization maintained
```

### **Redis Pub/Sub Performance (Asynchronous)**
```
Performance Targets:
├── Message Publishing: <1ms latency
├── Message Delivery: <1ms to subscribers
├── High-Frequency Channels: 100,000+ messages/second (simulation data)
├── Event Channels: 1,000+ messages/second (auth, project events)
├── Background Queues: 10,000+ messages/second (async processing)
├── Channel Subscription: <10ms to establish
└── Memory Usage: <1KB per channel, <1KB per subscriber
```

### **Multiple Instance Performance**
```
Scaling Performance:
├── Service Discovery: <100ms to detect new instances
├── Connection Establishment: <500ms to new instances
├── Load Balancing: <1ms overhead per request
├── Failover Time: <2 seconds to detect and route around failures
├── Instance Health Checks: Every 30 seconds
└── Connection Pool Rebalancing: <5 seconds after instance changes
```

### **Monitoring Metrics with Dynamic Scaling**
```
gRPC Dynamic Pool Metrics:
├── Active Connections: Current connection count per instance (5-20 range)
├── Connection Utilization: Percentage of connections actively serving requests
├── Request Queue Depth: Pending requests per connection pool
├── Response Latency: P50, P95, P99 response times per service
├── Error Rate: Failed requests percentage per service
├── Circuit Breaker Status: Open/closed state per service instance
├── Connection Health: Healthy/unhealthy connection count
├── Scaling Events: Scale up/down events per time period
├── Scaling Latency: Time to add/remove connections
├── Global Connection Usage: Total connections vs limit (1000)
├── Priority Rebalancing: Connection moves between services
└── Pool Efficiency: Utilization improvement over static pools

Connection Scaling Metrics:
├── Scale Up Triggers: Count of scale up events by trigger type
│   ├── High Utilization (>80%): Count per service
│   ├── Queue Depth (>10): Count per service
│   ├── High RPS (>100): Count per service
│   └── High Latency (>50ms): Count per service
├── Scale Down Events: Count of scale down events (after 5min delay)
├── Rebalancing Events: Connections moved between services
├── Connection Efficiency: Utilization before/after scaling
├── Memory Savings: Memory usage vs static pools
└── Resource Contention: Times connection requests were denied

Redis Pub/Sub Metrics:
├── Message Throughput: Messages/second per channel
├── Subscriber Count: Active subscribers per channel
├── Message Latency: Publish-to-delivery time
├── Channel Memory Usage: Memory per channel
├── Connection Count: Active Redis connections
└── Failed Deliveries: Messages that failed to deliver

Service Discovery Metrics:
├── Instance Count: Active instances per service
├── Discovery Latency: Time to detect instance changes
├── Health Check Success Rate: Percentage of successful health checks
├── Dynamic Pool Size: Current connections per service instance
├── Load Balancing Distribution: Request distribution across instances
├── Connection Pool Rebalancing: Time to adjust to instance changes
└── Global Resource Utilization: System-wide connection usage
```

---

*Last Updated: January 2025*  
*Version: 1.0 - Internal Mesh Communication*  
*Status: Protocol Definitions Complete*
