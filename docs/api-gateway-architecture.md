# API Gateway (Server Service) - Architecture Documentation

## 🎯 **Overview**

The API Gateway (Server Service) serves as the **lightweight, high-performance proxy** that handles all external client traffic while backend services communicate directly through the service mesh. It operates on port range **8000-8020** and acts as the single entry point for client applications.

## 🏗️ **Architecture Position**

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                           EXTERNAL CLIENTS                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               │
│  │   Browser   │  │   Mobile    │  │   Desktop   │  │   API       │               │
│  │   Client    │  │    App      │  │    App      │  │  Client     │               │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘               │
└─────────────────────────────────────────────────────────────────────────────────────┘
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
                        │     (Event Distribution)   │
                        └────────────────────────────┘
                                      │
        ┌─────────────────────────────┼─────────────────────────────┐
        │                             │                             │
┌───────▼──────┐            ┌─────────▼──────┐            ┌────────▼──────┐
│ Auth Service │            │ Project Service│            │Simulation Svc │
│ gRPC: 9000   │◄──────────►│ gRPC: 10000    │◄──────────►│ gRPC: 11000   │
│ HTTP: 9001   │   20 conn  │ HTTP: 10001    │   20 conn  │ HTTP: 11001   │
│              │    pool    │                │    pool    │               │
│ Direct gRPC  │            │ Direct gRPC    │            │ Direct gRPC   │
│ Mesh Network │            │ Communication  │            │ Communication │
└──────────────┘            └────────────────┘            └───────────────┘
```

## 🎯 **Core Responsibilities**

### **1. Client Connection Management**
- **HTTP Connections**: Handle REST API requests for CRUD operations, authentication, configuration
- **WebSocket Connections**: Manage 100,000+ concurrent WebSocket connections for real-time updates
- **Connection Pooling**: Efficient WebSocket connection pooling and lifecycle management
- **Protocol Translation**: Bridge between client HTTP/WebSocket and backend gRPC services

### **2. Request Routing & Load Balancing**
- **Service Routing**: Route incoming requests to appropriate backend services
  - `/api/auth/*` → Auth Service (gRPC:9000)
  - `/api/projects/*` → Project Service (gRPC:10000)
  - `/api/simulations/*` → Simulation Service (gRPC:11000)
- **Load Balancing**: Distribute requests across backend service instances
- **Circuit Breakers**: Implement fault tolerance for backend service failures
- **Request Aggregation**: Combine multiple backend service calls for complex client requests

### **3. Authentication & Security Layer**
- **Authentication Middleware**: Validate JWT tokens for all protected endpoints
- **Token Validation**: Call Auth Service via gRPC to validate user tokens and permissions
- **Rate Limiting**: Implement per-user, per-IP, and per-endpoint rate limiting
- **Security Headers**: Apply CORS, XSS protection, CSP, and other security measures
- **Request Sanitization**: Validate and sanitize all incoming requests

### **4. Real-time Communication Hub**
- **WebSocket Hub**: Central hub for managing 100,000+ persistent client connections per instance
- **Redis Pub/Sub Integration**: Subscribe to all event channels from backend services
- **Live Simulation Updates**: Stream high-frequency simulation data to connected clients
- **Notification Broadcasting**: Send targeted notifications to specific users or groups
- **Presence Management**: Track user online/offline status and connection state
- **Event Filtering**: Filter and route events based on user subscriptions and permissions

## 🔄 **Communication Patterns**

### **Client-Facing Communication (Port 8000-8020)**

#### **HTTP Endpoints**
```
GET  /api/auth/profile          → Auth Service (gRPC)
POST /api/projects              → Project Service (gRPC)
GET  /api/simulations/status    → Simulation Service (gRPC)
GET  /health                    → Aggregated health from all services
```

#### **WebSocket Endpoints**
```
/ws/notifications               → Real-time notifications
/ws/simulation/{id}             → Live simulation updates
/ws/collaboration/{project_id}  → Real-time collaboration
```

### **Backend Service Integration (gRPC)**

#### **Service Discovery & Connection**
```
Auth Service:        auth-service:9000      (Token validation, user context)
Project Service:     project-service:10000  (Project CRUD, templates)
Simulation Service:  simulation-service:11000 (Simulation control, monitoring)
```

#### **Connection Pool Management**
- **20 gRPC connections per backend service**
- **Dynamic scaling based on load (5-20 connections)**
- **Health monitoring and automatic reconnection**
- **Round-robin load balancing across connections**

## 🚫 **What API Gateway Does NOT Handle**

### **Business Logic**
- ❌ **No Authentication Logic**: Delegates to Auth Service via gRPC
- ❌ **No Project Management**: Delegates to Project Service via gRPC
- ❌ **No Simulation Processing**: Delegates to Simulation Service via gRPC
- ❌ **No Data Storage**: No direct database connections
- ❌ **No Business Rules**: Pure routing and protocol translation

### **Inter-Service Communication**
- ❌ **No Service-to-Service Routing**: Backend services communicate directly
- ❌ **No Central Hub Role**: Services bypass API Gateway for internal calls
- ❌ **No Single Point of Failure**: Backend services function independently

## 🔐 **Security Architecture**

### **Authentication Flow**
```
1. Client sends HTTP/WebSocket request with JWT token
2. API Gateway extracts token from Authorization header
3. API Gateway calls Auth Service via gRPC: ValidateToken(token)
4. Auth Service validates token and returns user context
5. If valid: Route request to appropriate service with user context
6. If invalid: Return 401 Unauthorized to client
```

### **Security Layers**
- **Perimeter Security**: First line of defense for all external requests
- **DDoS Protection**: Rate limiting and request throttling
- **Input Validation**: Request sanitization and format validation
- **SSL/TLS Termination**: Secure client connections
- **CORS Management**: Cross-origin request handling

## 📊 **Performance Specifications**

### **Connection Handling**
- **100,000+ concurrent WebSocket connections** per instance
- **Sub-10ms message routing** for real-time updates
- **Automatic reconnection handling** for dropped connections
- **Connection health monitoring** and cleanup

### **Request Processing**
- **High-throughput HTTP processing** (10K+ requests/second)
- **Efficient gRPC client connection pooling** (20 connections per service)
- **Request/response caching** for frequently accessed data
- **Graceful degradation** during backend service failures

### **Real-time Performance**
- **<16ms latency** for WebSocket message routing
- **Sub-5ms** for Redis pub/sub message processing
- **Efficient broadcasting** to multiple WebSocket clients
- **Memory-efficient connection management** (~8KB per connection)

## 🔄 **Real-time Data Flow**

### **Simulation Updates**
```
Simulation Service → Redis Pub/Sub → API Gateway → WebSocket Clients

Flow:
1. Simulation Service publishes update to Redis: "simulation_456_update"
2. API Gateway subscribes to simulation channels
3. API Gateway receives update from Redis pub/sub
4. API Gateway finds WebSocket connections subscribed to simulation_456
5. API Gateway broadcasts update to relevant clients via WebSocket
```

### **Notification System**
```
Any Service → Redis Pub/Sub → API Gateway → Targeted WebSocket Clients

Flow:
1. Backend service publishes notification: "user_123_notification"
2. API Gateway receives notification from Redis
3. API Gateway finds WebSocket connections for user_123
4. API Gateway sends notification to user's active connections
```

## 🏗️ **Technology Stack**

### **Framework & Libraries**
- **Go with Gin/Fiber**: High-performance HTTP server
- **Gorilla WebSocket**: WebSocket connection management
- **gRPC Go Client**: Backend service communication
- **Redis Go Client**: Real-time event subscription
- **Prometheus Client**: Metrics and monitoring

### **Infrastructure**
- **Docker**: Containerized deployment
- **Load Balancer**: Multiple API Gateway instances
- **Redis**: Real-time event streaming
- **Service Discovery**: Backend service location

## 🎯 **Design Principles**

### **Lightweight Proxy**
- **Minimal business logic**: Focus on routing and protocol translation
- **Stateless design**: No session storage in API Gateway
- **Fast request processing**: Minimal latency overhead
- **Horizontal scaling**: Multiple instances for high availability

### **Fault Tolerance**
- **Circuit breakers**: Automatic failure detection and recovery
- **Health check aggregation**: Monitor all backend services
- **Graceful degradation**: Continue operating with partial service failures
- **Retry logic**: Exponential backoff for failed requests

### **Real-time Optimization**
- **Connection persistence**: Maintain WebSocket connections across restarts
- **Efficient broadcasting**: Optimized message distribution
- **Low-latency routing**: Minimize message processing time
- **Resource management**: Optimal memory and CPU usage

## 🚀 **Deployment Architecture**

### **Port Allocation**
```
┌─────────────────┬─────────────────┬─────────────────────────────────────┐
│    Service      │   Port Range    │           Purpose                   │
├─────────────────┼─────────────────┼─────────────────────────────────────┤
│ API Gateway     │   8000-8020     │ Client connections (HTTP/WebSocket) │
│ - HTTP Server   │   8000          │ REST API endpoints                  │
│ - WebSocket     │   8001          │ Real-time connections               │
│ - Health Check  │   8002          │ Health monitoring                   │
│ - Metrics       │   8003          │ Prometheus metrics                  │
└─────────────────┴─────────────────┴─────────────────────────────────────┘
```

### **Scaling Strategy**
- **Horizontal Scaling**: Multiple API Gateway instances behind load balancer
- **Session Affinity**: WebSocket connections stick to specific instances
- **Health Monitoring**: Automatic instance replacement on failure
- **Resource Allocation**: CPU and memory optimization per instance

## 📈 **Monitoring & Observability**

### **Key Metrics**
- **Connection Count**: Active HTTP and WebSocket connections
- **Request Latency**: P50, P95, P99 response times
- **Error Rates**: 4xx and 5xx error percentages
- **Backend Health**: gRPC connection status to all services
- **Real-time Performance**: WebSocket message processing times

### **Health Checks**
- **Liveness**: API Gateway process health
- **Readiness**: Backend service connectivity
- **Dependency Health**: Auth, Project, Simulation service status
- **Resource Usage**: Memory, CPU, connection limits

---

## 🎯 **Summary**

The API Gateway serves as a **high-performance, lightweight proxy** that:

✅ **Handles all external client traffic** (HTTP + WebSocket)  
✅ **Provides authentication middleware** (via Auth Service gRPC calls)  
✅ **Routes requests to backend services** (via gRPC service mesh)  
✅ **Manages real-time connections** (WebSocket hub with 100K+ connections)  
✅ **Implements security and rate limiting** (perimeter defense)  
✅ **Does NOT handle business logic** (pure routing and translation)  
✅ **Does NOT route inter-service traffic** (services communicate directly)  

This design ensures the API Gateway remains a **thin, scalable layer** focused on client communication while backend services handle all business logic through direct service mesh communication.

---

*Last Updated: January 2025*  
*Version: 1.0 - API Gateway Architecture*  
*Status: Design Complete, Implementation Planned*
