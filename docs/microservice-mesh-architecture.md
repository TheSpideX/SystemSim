# System Design Simulator - Final Backend Architecture

## 🏗️ **Complete System Architecture**

```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                           EXTERNAL CLIENTS                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                   │
│  │   Browser   │  │   Mobile    │  │   Desktop   │  │   API       │                   │
│  │   Client    │  │    App      │  │    App      │  │  Client     │                   │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘                   │
└─────────────────────────────────────────────────────────────────────────────────────────┘
                                      │
                        ┌─────────────▼──────────────┐
                        │      LOAD BALANCER         │
                        │    (HAProxy/Nginx)         │
                        │   • Health Checks          │
                        │   • SSL Termination        │
                        │   • Rate Limiting          │
                        └────────────────────────────┘
                                      │
        ┌─────────────────────────────┼─────────────────────────────┐
        │                             │                             │
┌───────▼──────┐            ┌─────────▼──────┐            ┌────────▼──────┐
│ API Gateway  │            │ API Gateway    │            │ API Gateway   │
│ Instance 1   │            │ Instance 2     │            │ Instance 3    │
│ Port: 8000   │            │ Port: 8000     │            │ Port: 8000    │
│              │            │                │            │               │
│ • 100K WS    │            │ • 100K WS      │            │ • 100K WS     │
│ • HTTP API   │            │ • HTTP API     │            │ • HTTP API    │
│ • Auth Layer │            │ • Auth Layer   │            │ • Auth Layer  │
│ • Routing    │            │ • Routing      │            │ • Routing     │
└──────────────┘            └────────────────┘            └───────────────┘
        │                             │                             │
        └─────────────────────────────┼─────────────────────────────┘
                                      │
                        ┌─────────────▼──────────────┐
                        │      REDIS CLUSTER         │
                        │     (Pub/Sub + Cache)      │
                        │                            │
                        │  ┌──────────┐ ┌──────────┐ │
                        │  │ Master   │ │ Replica  │ │
                        │  │ Node 1   │ │ Node 2   │ │
                        │  │ Port:6379│ │ Port:6379│ │
                        │  └──────────┘ └──────────┘ │
                        │                            │
                        │ • Event Distribution       │
                        │ • Session Storage          │
                        │ • Service Discovery        │
                        │ • Application Cache        │
                        └────────────────────────────┘
                                      │
        ┌─────────────────────────────┼─────────────────────────────┐
        │                             │                             │
┌───────▼──────┐            ┌─────────▼──────┐            ┌────────▼──────┐
│ Auth Service │            │ Project Service│            │Simulation Svc │
│              │◄──────────►│                │◄──────────►│               │
│ gRPC: 9000   │   20 conn  │ gRPC: 10000    │   20 conn  │ gRPC: 11000   │
│ HTTP: 9001   │    pool    │ HTTP: 10001    │    pool    │ HTTP: 11001   │
│              │            │                │            │               │
│ • JWT Auth   │            │ • Project CRUD │            │ • Simulations │
│ • User Mgmt  │            │ • Templates    │            │ • Real-time   │
│ • RBAC       │            │ • Sharing      │            │ • Analytics   │
│ • Sessions   │            │ • Collaboration│            │ • Engine API  │
└──────────────┘            └────────────────┘            └───────────────┘
        │                             │                             │
        └─────────────────────────────┼─────────────────────────────┘
                                      │
                        ┌─────────────▼──────────────┐
                        │      POSTGRESQL            │
                        │    (Shared Database)       │
                        │                            │
                        │  ┌──────────────────────┐  │
                        │  │   auth_schema        │  │
                        │  │ • users              │  │
                        │  │ • sessions           │  │
                        │  │ • roles              │  │
                        │  │ • permissions        │  │
                        │  └──────────────────────┘  │
                        │                            │
                        │  ┌──────────────────────┐  │
                        │  │   project_schema     │  │
                        │  │ • projects           │  │
                        │  │ • templates          │  │
                        │  │ • sharing            │  │
                        │  │ • collaboration      │  │
                        │  └──────────────────────┘  │
                        │                            │
                        │  ┌──────────────────────┐  │
                        │  │  simulation_schema   │  │
                        │  │ • simulations        │  │
                        │  │ • results            │  │
                        │  │ • metrics            │  │
                        │  │ • analytics          │  │
                        │  └──────────────────────┘  │
                        └────────────────────────────┘
```

## 🎯 **Core Architecture Principles**

### **1. Service Mesh Network (All Services Are Equal Participants)**
- **API Gateway**: Full mesh participant (not just external proxy)
- **Auth Service**: Full mesh participant with authentication services
- **Project Service**: Full mesh participant with project management
- **Simulation Service**: Full mesh participant with simulation processing
- **Mesh Communication**: All services communicate directly with each other

### **2. Two-Type Communication Architecture**
- **gRPC (Synchronous)**: For immediate response operations
  - Token validation, permission checks, CRUD operations
  - Request-response pattern with strong typing
  - 20 connections between each service pair
- **Redis Pub/Sub (Asynchronous)**: For fire-and-forget operations
  - Real-time data streaming, event notifications
  - Heavy data transfer, background processing
  - Single Redis cluster with multiple channels

### **3. Multiple Instance Communication Strategy**
- **Service Discovery**: Redis-based registry for dynamic instance discovery
- **Connection Distribution**: Services connect to ALL instances of target services
- **Load Balancing**: Round-robin across all available instances
- **Fault Tolerance**: Automatic failover when instances become unavailable

## 🔗 **Service Communication Matrix**

### **Port Allocation Strategy**
```
┌─────────────────┬─────────────────┬─────────────────────────────────────┐
│    Service      │   Port Range    │           Purpose                   │
├─────────────────┼─────────────────┼─────────────────────────────────────┤
│ Load Balancer   │   80/443        │ SSL termination, traffic distribution│
│ API Gateway     │   8000          │ Client connections (HTTP/WebSocket) │
│ Auth Service    │   9000/9001     │ gRPC server / HTTP endpoints        │
│ Project Service │  10000/10001    │ gRPC server / HTTP endpoints        │
│ Simulation Svc  │  11000/11001    │ gRPC server / HTTP endpoints        │
│ Redis Cluster   │   6379          │ Pub/Sub, Cache, Service Discovery   │
│ PostgreSQL      │   5432          │ Shared database with schemas        │
└─────────────────┴─────────────────┴─────────────────────────────────────┘
```

## 🔄 **Internal Mesh Communication Protocols**

### **Service-to-Service gRPC Contracts**

#### **Auth Service gRPC Interface (Port 9000)**
```protobuf
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

message ValidateTokenRequest {
  string token = 1;
  string calling_service = 2;
}

message ValidateTokenResponse {
  bool valid = 1;
  string user_id = 2;
  string email = 3;
  bool is_admin = 4;
  string session_id = 5;
  repeated string permissions = 6;
}
```

#### **Project Service gRPC Interface (Port 10000)**
```protobuf
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

  // Health check
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

message CreateProjectRequest {
  string user_id = 1;
  string name = 2;
  string description = 3;
  string template_id = 4;
  map<string, string> configuration = 5;
}

message CreateProjectResponse {
  string project_id = 1;
  string name = 2;
  string status = 3;
  int64 created_at = 4;
}
```

#### **Simulation Service gRPC Interface (Port 11000)**
```protobuf
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

message StartSimulationRequest {
  string user_id = 1;
  string project_id = 2;
  map<string, string> configuration = 3;
  repeated string components = 4;
}

message StartSimulationResponse {
  string simulation_id = 1;
  string status = 2;
  int64 started_at = 3;
  string websocket_channel = 4;
}
```

### **Two-Type Communication Architecture**

#### **1. gRPC Communication (Synchronous)**
```
Purpose: Immediate response operations
Pattern: Request-Response with strong typing
Use Cases:
├── Token validation (Auth Service)
├── Permission checks (Auth Service)
├── Project CRUD operations (Project Service)
├── Simulation control (Simulation Service)
└── User context retrieval (Auth Service)

Connection Matrix (20 connections per service pair):
┌─────────────────┬─────────────────┬─────────────────┬─────────────────┬─────────────────┐
│ From Service    │ To API Gateway  │ To Auth (9000)  │ To Project      │ To Simulation   │
│                 │ (8000)          │                 │ (10000)         │ (11000)         │
├─────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ API Gateway     │ N/A             │ 20 connections  │ 20 connections  │ 20 connections  │
│ Auth Service    │ 20 connections  │ N/A             │ 20 connections  │ 20 connections  │
│ Project Service │ 20 connections  │ 20 connections  │ N/A             │ 20 connections  │
│ Simulation Svc  │ 20 connections  │ 20 connections  │ 20 connections  │ N/A             │
└─────────────────┴─────────────────┴─────────────────┴─────────────────┴─────────────────┘

Total gRPC Connections: 240 connections across the mesh (4 services × 3 targets × 20)
```

#### **2. Redis Pub/Sub Communication (Asynchronous)**
```
Purpose: Fire-and-forget operations, real-time data, heavy processing
Pattern: Publisher-Subscriber with multiple channels
Use Cases:
├── Real-time simulation data (high-frequency streaming)
├── Event notifications (login, logout, project changes)
├── Background processing (email sending, report generation)
├── System announcements and alerts
└── Heavy data transfer operations

Single Redis Cluster with Multiple Channels:
├── auth:events:* → Authentication events
├── project:events:* → Project lifecycle events
├── simulation:data:* → Real-time simulation metrics
├── simulation:events:* → Simulation lifecycle events
├── notifications:user:* → User-specific notifications
├── async:email:queue → Background email processing
├── async:reports:queue → Report generation queue
└── system:announcements → System-wide messages
```

### **Multiple Instance Communication Strategy**

#### **Service Instance Discovery**
```
Example: 3 API Gateway instances + 2 Auth Service instances

Service Registry (Redis):
├── services:api-gateway:instance-1 → {"grpc_port": 8000, "status": "healthy"}
├── services:api-gateway:instance-2 → {"grpc_port": 8000, "status": "healthy"}
├── services:api-gateway:instance-3 → {"grpc_port": 8000, "status": "healthy"}
├── services:auth-service:instance-1 → {"grpc_port": 9000, "status": "healthy"}
└── services:auth-service:instance-2 → {"grpc_port": 9000, "status": "healthy"}
```

#### **Dynamic Connection Distribution Strategy**
```
Each service connects to ALL instances with DYNAMIC connection pools:

Auth Service Instance 1 Dynamic Connections:
├── api-gateway:instance-1 (5-20 connections, traffic-based)
├── api-gateway:instance-2 (5-20 connections, traffic-based)
├── api-gateway:instance-3 (5-20 connections, traffic-based)
├── project-service:instance-1 (5-20 connections, traffic-based)
└── simulation-service:instance-1 (5-20 connections, traffic-based)

Low Traffic: 25 connections (5 each)
High Traffic: 100 connections (20 each)
Efficiency Gain: 75% reduction in idle connections

API Gateway Instance 1 Dynamic Connections:
├── auth-service:instance-1 (5-20 connections, traffic-based)
├── auth-service:instance-2 (5-20 connections, traffic-based)
├── project-service:instance-1 (5-20 connections, traffic-based)
└── simulation-service:instance-1 (5-20 connections, traffic-based)

Low Traffic: 20 connections (5 each)
High Traffic: 80 connections (20 each)
Efficiency Gain: 75% reduction in idle connections
```

#### **Dynamic Connection Pool Configuration**
```go
type ServiceMeshClient struct {
    serviceName     string
    targetServices  map[string]*MultiInstancePool
    discovery       ServiceDiscovery
    healthChecker   HealthChecker
    globalScaler    *GlobalConnectionScaler
}

type MultiInstancePool struct {
    serviceName     string
    instances       map[string]*DynamicConnectionPool  // instance-id -> dynamic pool
    loadBalancer    *WeightedRoundRobin               // Weight by utilization
    circuitBreaker  *CircuitBreaker                   // Per-service circuit breaker
    totalConnections int                              // Sum across all instances
}

type DynamicConnectionPool struct {
    instanceID       string
    targetAddress    string
    minConnections   int           // 5 (always warm)
    maxConnections   int           // 20 (scale up under load)
    currentConns     []*grpc.ClientConn
    activeConns      int           // Currently active connections
    requestQueue     chan Request  // Pending requests
    metrics          *PoolMetrics
    scaler          *ConnectionScaler
    healthStatus     HealthStatus
}

type PoolMetrics struct {
    RequestsPerSecond     float64
    AverageLatency        time.Duration
    QueueDepth            int
    ConnectionUtilization float64
    ErrorRate             float64
    LastScaleAction       time.Time
}

type ConnectionScaler struct {
    scaleUpThreshold    float64       // 80% utilization
    scaleDownThreshold  float64       // 30% utilization
    scaleUpTrigger      int           // Queue depth > 10
    scaleDownDelay      time.Duration // 5 minutes before scaling down
    maxGlobalConnections int          // 1000 per service instance
}

// Intelligent load balancing with utilization awareness
func (mp *MultiInstancePool) GetConnection() (*grpc.ClientConn, error) {
    // Select instance with lowest utilization
    selectedInstance := mp.loadBalancer.SelectByUtilization()
    if selectedInstance == nil {
        return nil, errors.New("no healthy instances available")
    }

    // Get connection from dynamic pool (may trigger scaling)
    return selectedInstance.GetConnectionWithScaling()
}
```

### **Dynamic Connection Scaling Logic**

#### **Traffic-Based Auto-Scaling**
```
Connection Scaling Matrix:

Traffic Level     | Connections | Trigger Conditions
Low (0-10 RPS)    | 5          | Default minimum (always warm)
Medium (10-50)    | 8-12       | Queue depth > 5 OR utilization > 60%
High (50-100)     | 12-16      | Queue depth > 8 OR utilization > 70%
Peak (100+ RPS)   | 16-20      | Queue depth > 10 OR utilization > 80%

Scale Up Triggers:
├── Connection utilization > 80%
├── Request queue depth > 10
├── Requests per second > 100
├── Average latency > 50ms
└── Error rate < 5% (healthy service)

Scale Down Triggers:
├── Connection utilization < 30%
├── Request queue depth < 2
├── Sustained low traffic for 5+ minutes
├── No scale actions in last 5 minutes
└── Above minimum connection count (5)
```

#### **Global Connection Management**
```go
type GlobalConnectionScaler struct {
    maxTotalConnections int     // 1000 per service instance
    currentTotal        int     // Current total across all pools
    priorityServices    []string // Service priority for connection allocation
    rebalanceThreshold  float64  // 90% of max before rebalancing
}

// Service Priority Order (highest to lowest):
Priority Order:
├── 1. Auth Service (most critical - token validation)
├── 2. Project Service (core business logic)
├── 3. Simulation Service (resource intensive)
└── 4. API Gateway (can handle connection pressure better)

func (gcs *GlobalConnectionScaler) requestConnectionIncrease(pool *DynamicConnectionPool) bool {
    if gcs.currentTotal < gcs.maxTotalConnections {
        gcs.currentTotal++
        return true
    }

    // Rebalance: Scale down lower priority services
    return gcs.rebalanceConnections(pool)
}
```

#### **Connection Efficiency Metrics**
```
System Efficiency Comparison:

Static 20 Connections (Old):
├── Total Connections: 540 (7 instances × ~77 avg)
├── Average Utilization: 15-20%
├── Memory Usage: 4.3MB (540 × 8KB)
├── Idle Connections: ~430 (80%)
├── Resource Waste: High

Dynamic 5-20 Connections (New):
├── Total Connections: 105-420 (traffic dependent)
├── Average Utilization: 60-80%
├── Memory Usage: 840KB-3.4MB (traffic dependent)
├── Idle Connections: ~20-80 (20%)
├── Resource Waste: Low

Efficiency Gains:
├── 80% reduction in idle connections during low traffic
├── 75% memory savings during normal operations
├── Automatic scaling for traffic spikes
├── Priority-based resource allocation
└── Global connection limits prevent resource exhaustion
```

### **Service Discovery & Registration**

#### **Redis-Based Service Registry**
```
Service Registry Structure:
┌─────────────────┬─────────────────┬─────────────────┬─────────────────┐
│ Service Name    │ gRPC Address    │ HTTP Address    │ Health Status   │
├─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ auth-service    │ :9000           │ :9001           │ healthy         │
│ project-service │ :10000          │ :10001          │ healthy         │
│ simulation-svc  │ :11000          │ :11001          │ healthy         │
│ api-gateway-1   │ N/A             │ :8000           │ healthy         │
│ api-gateway-2   │ N/A             │ :8000           │ healthy         │
│ api-gateway-3   │ N/A             │ :8000           │ healthy         │
└─────────────────┴─────────────────┴─────────────────┴─────────────────┘

Redis Keys:
├── services:auth-service → {"grpc_port": 9000, "http_port": 9001, "status": "healthy"}
├── services:project-service → {"grpc_port": 10000, "http_port": 10001, "status": "healthy"}
├── services:simulation-service → {"grpc_port": 11000, "http_port": 11001, "status": "healthy"}
└── services:api-gateway:* → {"http_port": 8000, "status": "healthy", "connections": 50000}
```

### **Event Distribution Architecture**

#### **Redis Pub/Sub Channels**
```
Channel Organization:
├── auth:events:login → User login events
├── auth:events:logout → User logout events
├── auth:events:permission_changed → Permission updates
├── project:events:created → Project creation events
├── project:events:shared → Project sharing events
├── project:events:updated → Project modification events
├── simulation:events:started → Simulation start events
├── simulation:events:completed → Simulation completion events
├── simulation:data:{id} → Real-time simulation data
├── notifications:user:{id} → User-specific notifications
└── system:announcements → System-wide messages
```

#### **Event Publishing Pattern**
```
Event Publishing Flow:
1. Service performs operation (e.g., create project)
2. Service publishes event to Redis: "project:events:created"
3. All API Gateway instances receive event
4. API Gateway filters event for relevant WebSocket connections
5. API Gateway broadcasts to connected clients
6. Other services can subscribe to relevant events if needed
```

## 🏢 **Service Architecture Details**

### **Load Balancer - Port 80/443**
```
Responsibilities:
✅ SSL/TLS termination and certificate management
✅ Traffic distribution across API Gateway instances
✅ Health checks and automatic failover
✅ DDoS protection and rate limiting
✅ Geographic routing (future)
✅ Static content serving (optional)

Technology Stack:
- HAProxy or Nginx
- Let's Encrypt SSL certificates
- Health check monitoring
- Access logging and metrics
- Geographic load balancing
```

### **API Gateway (Multiple Instances) - Port 8000**
```go
Responsibilities:
✅ Full service mesh participant (gRPC server + clients)
✅ Client connection management (HTTP + WebSocket)
✅ JWT token validation via Auth Service gRPC
✅ Request routing to appropriate services via gRPC
✅ Real-time event distribution (Redis pub/sub → WebSocket)
✅ Rate limiting and security (CORS, input validation)
✅ Protocol translation (HTTP/WebSocket ↔ gRPC)
✅ Load balancing across service instances

Technology Stack:
- Go with Gin framework
- gRPC server (8000) for mesh communication
- gRPC clients (20 connections per service instance)
- Gorilla WebSocket for real-time connections
- Redis client for pub/sub event distribution
- Circuit breakers and retry logic
- Service discovery and health monitoring

Mesh Communication:
├── gRPC Server (8000): Receives calls from other services
├── gRPC Clients: 20 connections to each instance of Auth, Project, Simulation
├── Redis Pub/Sub: Subscribes to all event channels for WebSocket broadcasting
└── Service Discovery: Discovers and connects to all service instances

Performance Specifications:
- 100,000+ concurrent WebSocket connections per instance
- 50,000+ HTTP requests/second per instance
- <50ms WebSocket message routing
- <100ms HTTP response time
- <10ms gRPC calls to other services
```

### **Auth Service - Port 9000 (gRPC) / 9001 (HTTP)**
```go
Implementation Status: ✅ COMPLETED (HTTP) + 🔄 gRPC Integration Needed

Responsibilities:
✅ User authentication and authorization
✅ JWT token management (access + refresh)
✅ Session management with Redis
✅ RBAC system (roles, permissions, authorization)
✅ Password security (bcrypt, strength validation)
✅ Account lockout and security features
✅ Email verification and password reset
🔄 gRPC server for service mesh integration
🔄 Service-to-service authentication

Technology Stack:
- Go with Gin framework (HTTP endpoints)
- gRPC server (service mesh integration)
- PostgreSQL auth_schema (users, sessions, roles, permissions)
- Redis for sessions, caching, and pub/sub
- JWT with secure token handling
- bcrypt for password hashing

Mesh Communication:
├── gRPC Server (9000): ValidateToken, GetUserContext, CheckPermission
├── gRPC Clients: 20 connections to each instance of API Gateway, Project, Simulation
├── Redis Pub/Sub Publisher: auth:events:login, auth:events:logout, auth:events:permission_changed
├── Redis Pub/Sub Subscriber: async:email:queue, system:announcements
└── Service Discovery: Discovers and connects to all service instances

gRPC Service Methods (Synchronous):
├── ValidateToken() → Immediate token validation for other services
├── GetUserContext() → Immediate user info with permissions
├── CheckPermission() → Immediate authorization checks
├── ValidateSession() → Immediate session validation
├── GetUserPermissions() → Immediate role-based permissions
└── HealthCheck() → Service health status

Redis Pub/Sub Channels (Asynchronous):
├── Publishes: auth:events:login, auth:events:logout, auth:events:permission_changed
├── Subscribes: async:email:queue (background email processing)
├── Subscribes: system:announcements (system-wide messages)
└── Heavy Operations: Password reset emails, account verification emails

Database Schema (auth_schema):
✅ users → User accounts and profiles
✅ sessions → Active user sessions
✅ roles → System and custom roles
✅ permissions → Granular permissions
✅ user_roles → User role assignments
✅ role_permissions → Role permission mappings

Performance Specifications:
- <5ms token validation (most critical operation)
- 10,000+ authentications/second
- 20 gRPC connections from each service
- Redis-backed session storage for scalability
```

### **Project Service - Port 10000 (gRPC) / 10001 (HTTP)**
```go
Implementation Status: 🔄 PLANNED

Responsibilities:
🔄 Project CRUD operations (create, read, update, delete)
🔄 Template library management (system and user templates)
🔄 Project sharing and permissions (collaboration)
🔄 Version control for projects (history, rollback)
🔄 Project collaboration metadata (real-time editing)
🔄 Project analytics (usage metrics, statistics)
🔄 Project import/export functionality

Technology Stack:
- Go with Gin framework (HTTP endpoints)
- gRPC server (service mesh integration)
- PostgreSQL project_schema (projects, templates, sharing, collaboration)
- Redis for caching and pub/sub
- Object storage for project assets (future)

gRPC Service Methods:
├── CreateProject() → Project creation with validation
├── GetProject() → Project retrieval with permissions
├── UpdateProject() → Project modifications
├── DeleteProject() → Project deletion with cleanup
├── ListProjects() → User's accessible projects
├── ShareProject() → Project sharing and permissions
├── GetTemplate() → Template retrieval
├── ListTemplates() → Available templates
└── HealthCheck() → Service health status

Database Schema (project_schema):
🔄 projects → Project metadata and configuration
🔄 templates → System and user-created templates
🔄 project_sharing → Sharing permissions and access
🔄 project_versions → Version control and history
🔄 collaboration_sessions → Real-time collaboration data

Performance Specifications:
- <10ms project operations
- 5,000+ projects/second throughput
- 20 gRPC connections from each service
- Real-time collaboration support
```

### **Simulation Service - Port 11000 (gRPC) / 11001 (HTTP)**
```go
Implementation Status: 🔄 PLANNED

Responsibilities:
🔄 Simulation engine interface (start, stop, control)
🔄 Real-time simulation processing (live updates)
🔄 Performance calculations (metrics, analytics)
🔄 Bottleneck detection (system analysis)
🔄 Failure injection (chaos engineering)
🔄 Simulation results (storage, retrieval)
🔄 ML-based performance predictions

Technology Stack:
- Go with Gin framework (HTTP endpoints)
- gRPC server (service mesh integration)
- PostgreSQL simulation_schema (simulations, results, metrics, analytics)
- Redis Pub/Sub for real-time data streaming
- Custom simulation algorithms
- ML libraries for performance predictions

gRPC Service Methods:
├── StartSimulation() → Simulation initialization and start
├── StopSimulation() → Simulation termination
├── GetSimulationStatus() → Current simulation state
├── GetSimulationResults() → Historical results
├── GetLiveMetrics() → Real-time performance data
├── GetPerformanceAnalysis() → Analytics and insights
├── ValidateConfiguration() → Pre-simulation validation
├── GetSimulationHistory() → User's simulation history
└── HealthCheck() → Service health status

Database Schema (simulation_schema):
🔄 simulations → Simulation metadata and configuration
🔄 simulation_results → Execution results and metrics
🔄 performance_metrics → Real-time and historical metrics
🔄 simulation_analytics → Analysis and insights
🔄 simulation_history → User simulation tracking

Performance Specifications:
- <15ms simulation control operations
- 100,000+ real-time messages/second
- 20 gRPC connections from each service
- Sub-16ms real-time data streaming
```

### **Redis Cluster - Port 6379**
```
Implementation Status: ✅ CONFIGURED

Responsibilities:
✅ Event distribution (Pub/Sub channels)
✅ Session storage (user authentication state)
✅ Application caching (frequently accessed data)
✅ Service discovery (dynamic service registry)
✅ Real-time data streaming (WebSocket events)

Technology Stack:
- Redis 7+ with cluster configuration
- Master-replica setup for high availability
- Persistence: RDB + AOF for data durability
- Memory optimization for high-throughput pub/sub
- Connection pooling for all services

Configuration:
├── Master Node: Primary read/write operations
├── Replica Node: Failover and read scaling
├── Memory: 8-16GB for high-throughput messaging
├── Persistence: Balanced RDB + AOF configuration
├── Networking: Private network with TLS encryption
└── Monitoring: Memory usage, connection count, message rates

Performance Specifications:
- 1M+ pub/sub messages/second
- <1ms pub/sub latency
- 10,000+ concurrent connections
- 99.9% availability with replica failover
```

### **PostgreSQL Database - Port 5432**
```
Implementation Status: ✅ CONFIGURED (Schema Separation)

Responsibilities:
✅ Primary data storage for all services
✅ ACID transactions and data consistency
✅ Schema separation for service isolation
✅ Connection pooling and performance optimization
✅ Automated backups and point-in-time recovery

Technology Stack:
- PostgreSQL 15+ with optimized configuration
- PgBouncer for connection pooling
- Automated backup with WAL archiving
- Performance monitoring and query optimization
- SSL connections for security

Database Schemas:
├── auth_schema: User accounts, sessions, roles, permissions
├── project_schema: Projects, templates, sharing, collaboration
├── simulation_schema: Simulations, results, metrics, analytics
├── Indexes: Optimized for common query patterns
├── Constraints: Foreign keys and data validation
└── Migrations: Version-controlled schema changes

Performance Specifications:
- 10,000+ transactions/second
- <10ms query response time
- Connection pooling (25 max, 5 idle per service)
- 99.9% availability with backup/recovery
- Horizontal scaling path (read replicas → separate databases)
```

## 📊 **Current Implementation Status**

### **✅ Completed Components**

**Auth Service (Production Ready):**
- Complete HTTP API with Gin framework
- PostgreSQL auth_schema with proper indexing
- Redis session management and caching
- JWT authentication with refresh tokens
- RBAC system with roles and permissions
- Security features (rate limiting, account lockout)
- Password security (bcrypt, strength validation)
- Email verification and password reset
- Docker containerization and health checks

**Infrastructure (Configured):**
- PostgreSQL with schema separation and connection pooling
- Redis cluster with pub/sub and high availability
- Load balancer configuration (HAProxy/Nginx)
- Service discovery and health monitoring
- Monitoring and logging infrastructure

### **🔄 In Progress / Planned**

**Phase 1: gRPC Mesh Implementation (Immediate)**
- Add gRPC server to Auth Service (port 9000)
- Implement protobuf definitions for all services
- Create gRPC clients with connection pooling (20 connections each)
- Service discovery with Redis registry
- Circuit breakers and health monitoring

**Phase 2: Service Development (Short-term)**
- Project Service implementation (ports 10000/10001)
- Simulation Service implementation (ports 11000/11001)
- API Gateway horizontal scaling setup
- Load balancer configuration and SSL termination
- Comprehensive monitoring and alerting

**Phase 3: Production Optimization (Medium-term)**
- Database read replicas for scaling
- Advanced caching strategies
- Performance monitoring and optimization
- Security hardening (mTLS, advanced auth)
- Disaster recovery and backup procedures

## 🚀 **Low-Effort Improvements (Recommended Next Steps)**

### **Phase 1: Health Checks (30 minutes)**
```go
// Extend existing health endpoint in auth service
GET /health
Response: {
  "status": "healthy",
  "service": "auth-service",
  "database": "healthy",
  "redis": "healthy",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### **Phase 2: Basic Monitoring (2-3 hours)**
```go
// Add Prometheus metrics middleware
- Request count by endpoint
- Response time percentiles
- Error rate tracking
- Active connections count
- Database connection pool usage
```

### **Phase 3: Dynamic Connection Pooling (4-5 hours)**
```go
// gRPC connection pool manager
type ConnectionPool struct {
    minConnections int    // 5
    maxConnections int    // 20
    currentLoad    float64
    connections    []*grpc.ClientConn
}

// Scale based on utilization
if utilization > 0.8 { scaleUp() }
if utilization < 0.3 { scaleDown() }
```

## 🔐 **Security Architecture**

### **Authentication Flow**
```
1. Client → Server Service (HTTP/WebSocket)
2. Server Service → Auth Service (gRPC)
3. Auth Service → PostgreSQL (User verification)
4. Auth Service → Redis (Session creation)
5. JWT tokens returned to client
6. Subsequent requests validated via JWT
```

### **Inter-Service Security**
- **mTLS**: Mutual TLS between services (planned)
- **Service Tokens**: Internal JWT tokens for service-to-service auth
- **Network Isolation**: Services communicate only through defined ports
- **Rate Limiting**: Per-service rate limits to prevent abuse

## 📈 **Performance Specifications**

### **System-Wide Performance Targets**
```
Overall System Capacity:
├── Concurrent Users: 500,000+ users
├── WebSocket Connections: 300,000+ concurrent connections
├── HTTP Requests: 50,000+ requests/second
├── Real-time Messages: 200,000+ messages/second
├── Database Transactions: 10,000+ TPS
├── Response Time: <100ms for API calls
└── Real-time Latency: <50ms for WebSocket messages
```

### **Per-Service Performance with Dynamic Scaling**
```
API Gateway (per instance):
├── WebSocket Connections: 100,000+ concurrent
├── HTTP Requests: 50,000+ requests/second
├── Message Routing: <50ms latency
├── gRPC Connections: 20-80 (5-20 per target service, traffic-based)
├── Connection Scaling: <500ms to add new connections
├── Memory Usage: ~8KB per WebSocket + 8KB per gRPC connection
└── CPU Usage: <70% under normal load

Auth Service (Priority 1 - Highest):
├── Token Validation: <5ms (most critical)
├── Authentication: 10,000+ auths/second
├── gRPC Connections: 15-60 (5-20 per target service)
├── Connection Priority: Always gets resources first
├── Scale Up Threshold: 80% utilization (fastest scaling)
├── Database Queries: <5ms average
└── Session Management: Redis-backed for scalability

Project Service (Priority 2):
├── CRUD Operations: <10ms response time
├── Project Queries: 5,000+ projects/second
├── gRPC Connections: 15-60 (5-20 per target service)
├── Connection Scaling: Based on queue depth and utilization
├── Collaboration: Real-time updates <16ms
├── Template Operations: <15ms response time
└── Database Operations: Optimized with indexing

Simulation Service (Priority 3):
├── Simulation Control: <15ms response time
├── Real-time Data: 100,000+ messages/second
├── gRPC Connections: 15-60 (5-20 per target service)
├── Connection Scaling: May be throttled under resource pressure
├── Performance Analysis: <100ms for complex calculations
├── Data Streaming: <16ms latency to clients
└── Engine Integration: High-throughput processing
```

### **Infrastructure Performance**
```
Redis Cluster:
├── Pub/Sub Messages: 1M+ messages/second
├── Pub/Sub Latency: <1ms
├── Cache Operations: <1ms response time
├── Concurrent Connections: 10,000+
└── Memory Usage: 8-16GB optimized

PostgreSQL:
├── Transactions: 10,000+ TPS
├── Query Response: <10ms average
├── Connection Pool: 25 max, 5 idle per service
├── Concurrent Connections: 100+ per service
└── Storage: SSD-optimized for performance

Load Balancer:
├── Request Throughput: 100,000+ requests/second
├── SSL Termination: Hardware-accelerated
├── Health Check Latency: <5ms
├── Failover Time: <2 seconds
└── Geographic Routing: <50ms additional latency
```

---

## 🎯 **Implementation Roadmap**

### **Phase 1: gRPC Mesh Foundation (Week 1-2)**
```
Priority Tasks:
1. Add gRPC server to Auth Service (2 days)
   - Implement protobuf definitions
   - Add gRPC endpoints alongside HTTP
   - Connection pool management

2. API Gateway gRPC Integration (2 days)
   - gRPC client connections to Auth Service
   - Connection pooling (20 connections)
   - Circuit breaker implementation

3. Service Discovery Setup (1 day)
   - Redis-based service registry
   - Health check integration
   - Dynamic endpoint discovery

4. Load Balancer Configuration (1 day)
   - HAProxy/Nginx setup
   - SSL termination
   - Health checks and failover
```

### **Phase 2: Service Development (Week 3-4)**
```
Priority Tasks:
1. Project Service Implementation (1 week)
   - gRPC and HTTP servers
   - Database schema creation
   - Service mesh integration

2. Simulation Service Implementation (1 week)
   - gRPC and HTTP servers
   - Real-time data streaming
   - Service mesh integration

3. API Gateway Scaling (2 days)
   - Multiple instance deployment
   - Load balancer integration
   - WebSocket session affinity
```

### **Phase 3: Production Optimization (Week 5-6)**
```
Priority Tasks:
1. Performance Optimization (1 week)
   - Database query optimization
   - Redis cluster configuration
   - Connection pool tuning

2. Monitoring and Alerting (3 days)
   - Prometheus metrics
   - Grafana dashboards
   - Alert configuration

3. Security Hardening (2 days)
   - mTLS between services
   - Advanced authentication
   - Security audit
```

## 🎯 **Success Metrics**

### **Performance Targets**
- **Response Time**: <100ms for API calls, <50ms for real-time updates
- **Throughput**: 500K+ concurrent users, 50K+ requests/second
- **Availability**: 99.9% uptime with automatic failover
- **Scalability**: Horizontal scaling without service disruption

### **Development Metrics**
- **Service Independence**: Services can be deployed independently
- **Development Velocity**: New features can be added without affecting other services
- **Operational Simplicity**: Clear monitoring and debugging capabilities
- **Cost Efficiency**: Optimal resource utilization with pay-as-you-grow scaling

---

## 🏆 **Architecture Benefits Summary**

### **Performance Benefits**
✅ **High Throughput**: 500K+ concurrent users supported
✅ **Low Latency**: <100ms API responses, <50ms real-time updates
✅ **Efficient Scaling**: Horizontal scaling with clear bottleneck identification
✅ **Optimized Resources**: Shared infrastructure with independent service scaling

### **Reliability Benefits**
✅ **High Availability**: No single points of failure with automatic failover
✅ **Fault Tolerance**: Circuit breakers and graceful degradation
✅ **Data Consistency**: ACID transactions with shared database
✅ **Service Isolation**: Service failures don't cascade through the system

### **Development Benefits**
✅ **Simple to Start**: Shared infrastructure reduces initial complexity
✅ **Clear Scaling Path**: Predictable scaling triggers and solutions
✅ **Service Independence**: Services can be developed and deployed independently
✅ **Operational Clarity**: Clear service boundaries and monitoring

---

*Last Updated: January 2025*
*Version: 2.0 - Complete Backend Architecture*
*Status: Ready for Implementation*
