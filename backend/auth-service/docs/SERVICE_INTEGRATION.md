# Auth Service Integration Guide

This document provides comprehensive information for developers working on other services (API Gateway, Project Service, Simulation Service) about how to integrate with the Auth Service.

## 🚀 Quick Start

The Auth Service provides authentication and authorization for the entire microservice mesh. It exposes both HTTP/2 and gRPC APIs for different use cases.

### Service Information
- **Service Name**: `auth-service`
- **HTTP/2 Port**: `9001` (client-facing operations with TLS)
- **gRPC Port**: `9000` (inter-service mesh communication)
- **Redis Port**: `6379` (pub/sub, sessions, caching, background services)
- **PostgreSQL Port**: `5432` (persistent data storage)
- **Health Check**: `https://localhost:9001/health`
- **Metrics**: `https://localhost:9001/metrics`

### Dependencies
- **PostgreSQL**: Primary database for persistent data
  - **Default Address**: `localhost:5432`
  - **Database**: `auth_service_db`
  - **Tables**: Separate tables for easy service separation if needed
  - **Used For**: Users, roles, permissions, audit logs

- **Redis**: High-performance operations and background services
  - **Default Address**: `localhost:6379`
  - **Database**: `0` (sessions), `1` (cache), `2` (pub/sub events)
  - **Used For**:
    - **Pub/Sub**: Background service communication, high-throughput data transfer
    - **Sessions**: JWT session storage and validation
    - **Caching**: User context, permissions, role caching
    - **Events**: Real-time auth events and notifications

---

## 🌐 HTTP/2 API (Client-Facing)

### Base URL
```
https://localhost:9001/api/v1
```

> **Note**: The service uses strict HTTP/2-only with mandatory TLS. Use `-k` flag with curl for self-signed certificates in development.

### Authentication Endpoints

#### 1. User Registration
```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "first_name": "John",
  "last_name": "Doe",
  "company": "Example Corp"
}
```

**Response (201 Created):**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe"
  },
  "access_token": "jwt_token_here",
  "refresh_token": "refresh_token_here",
  "expires_in": 900,
  "session_id": "session_uuid"
}
```

#### 2. User Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "remember": false
}
```

#### 3. Token Refresh
```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "refresh_token_here"
}
```

#### 4. Logout
```http
POST /auth/logout
Authorization: Bearer <access_token>
```

### User Management Endpoints

#### Get User Profile
```http
GET /user/profile
Authorization: Bearer <access_token>
```

#### Update User Profile
```http
PUT /user/profile
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "first_name": "Updated",
  "last_name": "Name"
}
```

#### Get User Sessions
```http
GET /user/sessions
Authorization: Bearer <access_token>
```

### RBAC Endpoints

#### Get My Roles
```http
GET /rbac/my-roles
Authorization: Bearer <access_token>
```

#### Get My Permissions
```http
GET /rbac/my-permissions
Authorization: Bearer <access_token>
```

---

## 🔗 gRPC API (Inter-Service Mesh)

### Connection Information
- **Address**: `localhost:9001`
- **Protocol**: gRPC with TLS (production) / Insecure (development)
- **Proto File**: `api/proto/auth.proto`

### Service Definition
```protobuf
service AuthService {
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc GetUserContext(GetUserContextRequest) returns (GetUserContextResponse);
  rpc CheckPermission(CheckPermissionRequest) returns (CheckPermissionResponse);
  rpc ValidateSession(ValidateSessionRequest) returns (ValidateSessionResponse);
  rpc GetUserPermissions(GetUserPermissionsRequest) returns (GetUserPermissionsResponse);
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

### Key gRPC Methods

#### 1. ValidateToken (Most Common)
**Use Case**: API Gateway validates incoming requests

```go
req := &auth.ValidateTokenRequest{
    Token:          "Bearer jwt_token_here",
    CallingService: "api-gateway",
    RequestId:      "unique_request_id",
}

resp, err := authClient.ValidateToken(ctx, req)
if err != nil {
    // Handle gRPC error
}

if resp.Valid {
    userID := resp.UserId
    permissions := resp.Permissions
    isAdmin := resp.IsAdmin
    // Token is valid, proceed with request
} else {
    // Token invalid: resp.ErrorMessage contains details
}
```

#### 2. GetUserContext
**Use Case**: Services need complete user information

```go
req := &auth.GetUserContextRequest{
    UserId:         "user_uuid",
    CallingService: "project-service",
    RequestId:      "unique_request_id",
}

resp, err := authClient.GetUserContext(ctx, req)
if err != nil {
    // Handle gRPC error
}

// Access user details
email := resp.Email
firstName := resp.FirstName
company := resp.Company
roles := resp.Roles
permissions := resp.Permissions
lastLogin := resp.LastLogin
```

#### 3. CheckPermission
**Use Case**: Services check specific permissions

```go
req := &auth.CheckPermissionRequest{
    UserId:         "user_uuid",
    Permission:     "read:project",
    ResourceId:     "project_123",
    CallingService: "project-service",
    RequestId:      "unique_request_id",
}

resp, err := authClient.CheckPermission(ctx, req)
if err != nil {
    // Handle gRPC error
}

if resp.Allowed {
    // User has permission
} else {
    // Permission denied: resp.Reason contains details
}
```

---

## 🏗️ Integration Patterns

### 1. API Gateway Pattern
```go
// API Gateway validates all incoming requests
func (gw *APIGateway) AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractToken(r)
        
        resp, err := gw.authClient.ValidateToken(ctx, &auth.ValidateTokenRequest{
            Token:          token,
            CallingService: "api-gateway",
            RequestId:      generateRequestID(),
        })
        
        if err != nil || !resp.Valid {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Add user context to request
        ctx := context.WithValue(r.Context(), "user_id", resp.UserId)
        ctx = context.WithValue(ctx, "permissions", resp.Permissions)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 2. Service-to-Service Authorization
```go
// Project Service checks permissions before operations
func (ps *ProjectService) GetProject(ctx context.Context, projectID string) (*Project, error) {
    userID := getUserIDFromContext(ctx)
    
    // Check if user can read this project
    resp, err := ps.authClient.CheckPermission(ctx, &auth.CheckPermissionRequest{
        UserId:         userID,
        Permission:     "read:project",
        ResourceId:     projectID,
        CallingService: "project-service",
        RequestId:      generateRequestID(),
    })
    
    if err != nil {
        return nil, fmt.Errorf("auth check failed: %w", err)
    }
    
    if !resp.Allowed {
        return nil, fmt.Errorf("permission denied: %s", resp.Reason)
    }
    
    // User has permission, proceed with operation
    return ps.getProjectFromDB(projectID)
}
```

### 3. User Context Enrichment
```go
// Simulation Service gets full user context for personalization
func (ss *SimulationService) CreateSimulation(ctx context.Context, req *CreateSimulationRequest) (*Simulation, error) {
    userID := getUserIDFromContext(ctx)
    
    // Get complete user context
    userResp, err := ss.authClient.GetUserContext(ctx, &auth.GetUserContextRequest{
        UserId:         userID,
        CallingService: "simulation-service",
        RequestId:      generateRequestID(),
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to get user context: %w", err)
    }
    
    // Use user context for simulation
    simulation := &Simulation{
        UserID:      userID,
        UserEmail:   userResp.Email,
        Company:     userResp.Company,
        Permissions: userResp.Permissions,
        // ... other fields
    }
    
    return ss.createSimulation(simulation)
}
```

---

## 🔧 Connection Management

### gRPC Client Setup
```go
package main

import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    auth "path/to/auth/proto"
)

func setupAuthClient() (auth.AuthServiceClient, error) {
    conn, err := grpc.Dial("localhost:9001", 
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4*1024*1024)),
    )
    if err != nil {
        return nil, err
    }
    
    return auth.NewAuthServiceClient(conn), nil
}
```

### Connection Pooling (Recommended)
```go
// Use connection pooling for better performance
type AuthClientPool struct {
    pool *mesh.ConnectionPool
}

func (p *AuthClientPool) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
    return p.pool.CallWithRetry(ctx, "auth-service", func(conn *grpc.ClientConn) (*auth.ValidateTokenResponse, error) {
        client := auth.NewAuthServiceClient(conn)
        return client.ValidateToken(ctx, req)
    })
}
```

---

## 📊 Health Monitoring

### Health Check Endpoints
```bash
# Basic health check
curl -k --http2 https://localhost:9001/health

# Detailed health check
curl -k --http2 https://localhost:9001/health/detailed

# Readiness check
curl -k --http2 https://localhost:9001/health/ready

# Liveness check
curl -k --http2 https://localhost:9001/health/live

# Metrics (Prometheus format)
curl -k --http2 https://localhost:9001/metrics
```

### gRPC Health Check
```go
resp, err := authClient.HealthCheck(ctx, &auth.HealthCheckRequest{
    CallingService: "your-service",
    RequestId:      "health-check-123",
})

if err != nil {
    // Auth service is down
} else {
    // Check resp.Status for health status
}
```

---

## 🔴 Redis Pub/Sub & Background Services

### Redis Usage by Auth Service
The auth service uses Redis for high-performance operations and background service communication:

```bash
# Database 0: Session Storage
# - JWT session tokens and validation
# - Session metadata (device info, IP, etc.)
# - Session expiration tracking

# Database 1: High-Performance Caching
# - User context caching for fast lookups
# - Permission and role caching
# - Frequently accessed data

# Database 2: Pub/Sub & Background Services
# - Real-time authentication events
# - Background service communication
# - High-throughput data transfer between services
# - Async processing notifications
```

### Redis Pub/Sub for Background Services
Redis pub/sub enables high-throughput, real-time communication between services:

#### 1. Background Service Communication
```go
// Publisher (Auth Service)
func publishUserEvent(redisClient *redis.Client, event string, data map[string]interface{}) error {
    payload, _ := json.Marshal(map[string]interface{}{
        "event":     event,
        "data":      data,
        "timestamp": time.Now().Unix(),
        "service":   "auth-service",
    })

    return redisClient.Publish(context.Background(), "auth.events", payload).Err()
}

// Subscriber (Other Services)
func subscribeToAuthEvents(redisClient *redis.Client) {
    pubsub := redisClient.Subscribe(context.Background(), "auth.events")
    defer pubsub.Close()

    for msg := range pubsub.Channel() {
        var event map[string]interface{}
        json.Unmarshal([]byte(msg.Payload), &event)

        switch event["event"] {
        case "user.login":
            handleUserLogin(event["data"])
        case "user.permissions.changed":
            invalidateUserCache(event["data"])
        case "bulk.user.update":
            processBulkUserUpdate(event["data"])
        }
    }
}
```

#### 2. High-Throughput Data Transfer
```go
// For bulk operations and high-volume data transfer
func publishBulkUserData(redisClient *redis.Client, users []User) error {
    // Split large datasets into chunks for efficient transfer
    chunkSize := 1000
    for i := 0; i < len(users); i += chunkSize {
        end := i + chunkSize
        if end > len(users) {
            end = len(users)
        }

        chunk := users[i:end]
        payload, _ := json.Marshal(map[string]interface{}{
            "event":      "bulk.user.sync",
            "chunk":      i / chunkSize,
            "total":      len(users),
            "data":       chunk,
            "timestamp":  time.Now().Unix(),
        })

        if err := redisClient.Publish(context.Background(), "auth.bulk", payload).Err(); err != nil {
            return err
        }
    }
    return nil
}
```

#### 3. Background Processing Notifications
```go
// Notify background services of processing tasks
func notifyBackgroundTask(redisClient *redis.Client, taskType string, params map[string]interface{}) error {
    task := map[string]interface{}{
        "task_id":    uuid.New().String(),
        "task_type":  taskType,
        "params":     params,
        "created_at": time.Now().Unix(),
        "priority":   "normal",
    }

    payload, _ := json.Marshal(task)
    return redisClient.Publish(context.Background(), "auth.background.tasks", payload).Err()
}

// Background service processes tasks
func processBackgroundTasks(redisClient *redis.Client) {
    pubsub := redisClient.Subscribe(context.Background(), "auth.background.tasks")
    defer pubsub.Close()

    for msg := range pubsub.Channel() {
        var task map[string]interface{}
        json.Unmarshal([]byte(msg.Payload), &task)

        switch task["task_type"] {
        case "user.cleanup":
            cleanupInactiveUsers(task["params"])
        case "permission.audit":
            auditUserPermissions(task["params"])
        case "session.cleanup":
            cleanupExpiredSessions(task["params"])
        }
    }
}
```

### Redis Connection for Other Services
Other services may need Redis access for:

#### 1. Event Subscription (Optional)
```go
// Subscribe to auth events
import "github.com/systemsim/auth-service/internal/events"

subscriber := events.NewSubscriber(redisClient)

// Listen for user login events
subscriber.Subscribe("user.login", func(event events.Event) {
    userID := event.Data["user_id"]
    // Handle user login in your service
})

// Listen for permission changes
subscriber.Subscribe("user.permissions.changed", func(event events.Event) {
    userID := event.Data["user_id"]
    // Invalidate cached permissions
})
```

#### 2. Session Validation (Advanced)
```go
// Direct Redis session validation (not recommended - use gRPC instead)
func validateSessionDirect(redisClient *redis.Client, sessionID string) (bool, error) {
    key := fmt.Sprintf("session:%s", sessionID)
    exists, err := redisClient.Exists(context.Background(), key).Result()
    return exists > 0, err
}

// ⚠️ WARNING: Direct Redis access bypasses auth service logic
// Use gRPC ValidateSession instead for proper validation
```

### Redis Configuration for Services
```go
// Redis client setup for event subscription
func setupRedisForEvents() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     os.Getenv("REDIS_URL"),
        Password: os.Getenv("REDIS_PASSWORD"),
        DB:       2, // Events database
    })
}

// Redis client for caching (if needed)
func setupRedisForCache() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     os.Getenv("REDIS_URL"),
        Password: os.Getenv("REDIS_PASSWORD"),
        DB:       3, // Use different DB for service-specific cache
    })
}
```

### Event Types Published by Auth Service
```go
// Authentication Events
"user.login"           // User logged in
"user.logout"          // User logged out
"user.token.refresh"   // Token refreshed

// User Management Events
"user.created"         // New user registered
"user.updated"         // User profile updated
"user.deleted"         // User account deleted

// Permission Events
"user.permissions.changed"  // User permissions modified
"user.roles.changed"        // User roles modified

// Security Events
"user.password.changed"     // Password changed
"user.locked"              // Account locked
"user.unlocked"            // Account unlocked
```

### Redis Health Monitoring
```go
// Check Redis connectivity in your service
func checkRedisHealth(redisClient *redis.Client) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    return redisClient.Ping(ctx).Err()
}
```

---

## 🐘 PostgreSQL Database Architecture

### Database Design for Service Separation
The auth service uses PostgreSQL with separate tables designed for easy service separation:

```sql
-- Auth Service Database: auth_service_db

-- Core User Management Tables
CREATE TABLE auth_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    company VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    is_admin BOOLEAN DEFAULT false,
    email_verified BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Session Management (can be moved to separate service)
CREATE TABLE auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth_users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    device_info JSONB,
    ip_address INET,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP DEFAULT NOW()
);

-- RBAC Tables (can be moved to separate RBAC service)
CREATE TABLE auth_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE auth_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(resource, action)
);

CREATE TABLE auth_user_roles (
    user_id UUID REFERENCES auth_users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES auth_roles(id) ON DELETE CASCADE,
    granted_at TIMESTAMP DEFAULT NOW(),
    granted_by UUID REFERENCES auth_users(id),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE auth_role_permissions (
    role_id UUID REFERENCES auth_roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES auth_permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Audit and Logging (can be moved to separate audit service)
CREATE TABLE auth_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth_users(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Database Connection for Other Services

#### 1. Read-Only Access (Recommended)
Other services should primarily use gRPC APIs, but may need read-only database access for:

```go
// Database connection for read-only operations
func setupAuthDBReadOnly() *sql.DB {
    dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&default_query_exec_mode=simple_protocol",
        os.Getenv("AUTH_DB_READ_USER"),     // read-only user
        os.Getenv("AUTH_DB_READ_PASSWORD"),
        os.Getenv("AUTH_DB_HOST"),
        os.Getenv("AUTH_DB_PORT"),
        os.Getenv("AUTH_DB_NAME"))

    db, err := sql.Open("postgres", dsn)
    if err != nil {
        log.Fatal("Failed to connect to auth database:", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(10)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(time.Hour)

    return db
}

// Example: Project service checking user existence
func (ps *ProjectService) validateUserExists(userID string) (bool, error) {
    var exists bool
    query := "SELECT EXISTS(SELECT 1 FROM auth_users WHERE id = $1 AND is_active = true)"

    err := ps.authDB.QueryRow(query, userID).Scan(&exists)
    return exists, err
}
```

#### 2. Service Separation Strategy
Tables are designed for easy separation into microservices:

```bash
# Future Service Separation Options:

# 1. User Management Service
# Tables: auth_users, auth_audit_logs (user actions)
# Responsibilities: User CRUD, profile management, user lifecycle

# 2. Session Management Service
# Tables: auth_sessions
# Responsibilities: Session creation, validation, cleanup

# 3. RBAC Service
# Tables: auth_roles, auth_permissions, auth_user_roles, auth_role_permissions
# Responsibilities: Role/permission management, authorization checks

# 4. Audit Service
# Tables: auth_audit_logs
# Responsibilities: Audit logging, compliance reporting, security monitoring
```

### Database Environment Variables
```bash
# PostgreSQL Connection
AUTH_DB_HOST=postgres
AUTH_DB_PORT=5432
AUTH_DB_NAME=auth_service_db
AUTH_DB_USER=auth_service
AUTH_DB_PASSWORD=secure_password

# Read-only access for other services
AUTH_DB_READ_USER=auth_readonly
AUTH_DB_READ_PASSWORD=readonly_password

# Connection Pool Settings
AUTH_DB_MAX_OPEN_CONNS=25
AUTH_DB_MAX_IDLE_CONNS=10
AUTH_DB_CONN_MAX_LIFETIME=1h
```

### Database Migrations and Schema Management
```go
// Schema initialization (auto-created on startup)
func initializeAuthSchema(db *sql.DB) error {
    // Check if schema exists
    var exists bool
    err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'auth_users')").Scan(&exists)
    if err != nil {
        return err
    }

    if !exists {
        // Create schema (tables shown above)
        return createAuthTables(db)
    }

    return nil
}

// Health check for database
func checkDatabaseHealth(db *sql.DB) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    return db.PingContext(ctx)
}
```

---

## ⚠️ Error Handling

### HTTP Error Responses
```json
{
  "error": "invalid_credentials",
  "message": "Email or password is incorrect",
  "details": {
    "field": "password",
    "code": "INVALID"
  }
}
```

### Common HTTP Status Codes
- `400 Bad Request`: Invalid request format or missing required fields
- `401 Unauthorized`: Invalid or expired token
- `403 Forbidden`: Valid token but insufficient permissions
- `404 Not Found`: User or resource not found
- `409 Conflict`: Email already exists (registration)
- `429 Too Many Requests`: Rate limiting
- `500 Internal Server Error`: Server error

### gRPC Error Handling
```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

resp, err := authClient.ValidateToken(ctx, req)
if err != nil {
    st, ok := status.FromError(err)
    if ok {
        switch st.Code() {
        case codes.InvalidArgument:
            // Invalid request parameters
        case codes.Unauthenticated:
            // Invalid token
        case codes.PermissionDenied:
            // Insufficient permissions
        case codes.NotFound:
            // User not found
        case codes.DeadlineExceeded:
            // Request timeout
        case codes.Unavailable:
            // Auth service unavailable
        default:
            // Other gRPC errors
        }
    }
    return err
}

// Check application-level errors
if !resp.Valid {
    // Handle invalid token: resp.ErrorMessage contains details
}
```

---

## 🔒 Security Considerations

### Token Handling
```go
// ✅ DO: Always validate tokens on every request
func (s *Service) protectedHandler(w http.ResponseWriter, r *http.Request) {
    token := extractBearerToken(r)
    if token == "" {
        http.Error(w, "Missing token", http.StatusUnauthorized)
        return
    }

    // Validate with auth service
    resp, err := s.authClient.ValidateToken(ctx, &auth.ValidateTokenRequest{
        Token: token,
        CallingService: "your-service",
        RequestId: generateRequestID(),
    })

    if err != nil || !resp.Valid {
        http.Error(w, "Invalid token", http.StatusUnauthorized)
        return
    }

    // Token is valid, proceed
}

// ❌ DON'T: Cache token validation results
// ❌ DON'T: Skip token validation for "internal" requests
// ❌ DON'T: Trust tokens without validation
```

### Permission Checking
```go
// ✅ DO: Check permissions for every protected operation
func (s *ProjectService) DeleteProject(ctx context.Context, projectID string) error {
    userID := getUserIDFromContext(ctx)

    // Always check permissions
    resp, err := s.authClient.CheckPermission(ctx, &auth.CheckPermissionRequest{
        UserId:     userID,
        Permission: "delete:project",
        ResourceId: projectID,
        CallingService: "project-service",
        RequestId: generateRequestID(),
    })

    if err != nil || !resp.Allowed {
        return fmt.Errorf("permission denied")
    }

    return s.deleteProjectFromDB(projectID)
}
```

### Request Context
```go
// ✅ DO: Always include calling service and request ID
func makeAuthRequest(callingService, requestID string) *auth.ValidateTokenRequest {
    return &auth.ValidateTokenRequest{
        Token:          token,
        CallingService: callingService,  // For audit logs
        RequestId:      requestID,       // For request tracing
    }
}
```

---

## 🚀 Deployment Configuration

### Environment Variables
```bash
# Auth Service Connection
AUTH_SERVICE_HTTP2_URL=https://auth-service:9001
AUTH_SERVICE_GRPC_URL=auth-service:9000

# PostgreSQL Connection (Required)
AUTH_DB_HOST=postgres
AUTH_DB_PORT=5432
AUTH_DB_NAME=auth_service_db
AUTH_DB_USER=auth_service
AUTH_DB_PASSWORD=secure_password
AUTH_DB_READ_USER=auth_readonly
AUTH_DB_READ_PASSWORD=readonly_password

# Redis Connection (Required)
REDIS_URL=redis://redis:6379
REDIS_PASSWORD=your_redis_password
REDIS_DB_SESSIONS=0
REDIS_DB_CACHE=1
REDIS_DB_PUBSUB=2

# Connection Pool Settings
AUTH_GRPC_MAX_CONNECTIONS=10
AUTH_GRPC_MIN_CONNECTIONS=2
AUTH_GRPC_CONNECTION_TIMEOUT=30s
AUTH_DB_MAX_OPEN_CONNS=25
AUTH_DB_MAX_IDLE_CONNS=10

# Request Timeouts
AUTH_REQUEST_TIMEOUT=10s
AUTH_HEALTH_CHECK_INTERVAL=30s

# Retry Settings
AUTH_MAX_RETRIES=3
AUTH_RETRY_DELAY=1s

# Background Service Settings
REDIS_PUBSUB_ENABLED=true
BACKGROUND_TASKS_ENABLED=true
```

### Docker Compose Example
```yaml
version: '3.8'
services:
  your-service:
    build: .
    environment:
      - AUTH_SERVICE_GRPC_URL=auth-service:9001
      - AUTH_REQUEST_TIMEOUT=10s
      - REDIS_URL=redis://redis:6379
      - AUTH_DB_HOST=postgres
      - AUTH_DB_READ_USER=auth_readonly
      - AUTH_DB_READ_PASSWORD=readonly_password
    depends_on:
      - auth-service
      - postgres
      - redis
    networks:
      - mesh-network

  auth-service:
    image: auth-service:latest
    ports:
      - "9001:9001"  # HTTP/2 with TLS (external)
      - "9000:9000"  # gRPC (internal)
    environment:
      - REDIS_URL=redis://redis:6379
      - AUTH_DB_HOST=postgres
      - AUTH_DB_USER=auth_service
      - AUTH_DB_PASSWORD=secure_password
      - REDIS_PUBSUB_ENABLED=true
      - BACKGROUND_TASKS_ENABLED=true
    depends_on:
      - postgres
      - redis
    networks:
      - mesh-network

  postgres:
    image: postgres:15-alpine
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_DB=auth_service_db
      - POSTGRES_USER=auth_service
      - POSTGRES_PASSWORD=secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init-db.sql:/docker-entrypoint-initdb.d/init-db.sql
    networks:
      - mesh-network

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    networks:
      - mesh-network

volumes:
  postgres_data:
  redis_data:

networks:
  mesh-network:
    driver: bridge
```

### Kubernetes Example
```yaml
apiVersion: v1
kind: Service
metadata:
  name: auth-service
spec:
  selector:
    app: auth-service
  ports:
    - name: http2
      port: 9001
      targetPort: 9001
    - name: grpc
      port: 9000
      targetPort: 9000
---
apiVersion: v1
kind: Service
metadata:
  name: redis
spec:
  selector:
    app: redis
  ports:
    - port: 6379
      targetPort: 6379
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: your-service
spec:
  template:
    spec:
      containers:
      - name: your-service
        image: your-service:latest
        env:
        - name: AUTH_SERVICE_GRPC_URL
          value: "auth-service:9001"
        - name: AUTH_REQUEST_TIMEOUT
          value: "10s"
        - name: REDIS_URL
          value: "redis://redis:6379"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
spec:
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
```

---

## 📝 Development Tips

### 1. Use Request IDs for Tracing
```go
func generateRequestID() string {
    return fmt.Sprintf("%s-%d", uuid.New().String()[:8], time.Now().UnixNano())
}
```

### 2. Implement Circuit Breakers
```go
type AuthClient struct {
    client auth.AuthServiceClient
    breaker *circuitbreaker.CircuitBreaker
}

func (c *AuthClient) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
    result, err := c.breaker.Execute(func() (interface{}, error) {
        return c.client.ValidateToken(ctx, req)
    })

    if err != nil {
        return nil, err
    }

    return result.(*auth.ValidateTokenResponse), nil
}
```

### 3. Add Metrics and Logging
```go
func (c *AuthClient) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        metrics.RecordAuthRequest("validate_token", duration)
    }()

    log.Info("Validating token",
        "calling_service", req.CallingService,
        "request_id", req.RequestId)

    resp, err := c.client.ValidateToken(ctx, req)
    if err != nil {
        log.Error("Token validation failed", "error", err)
        return nil, err
    }

    log.Info("Token validation completed",
        "valid", resp.Valid,
        "user_id", resp.UserId)

    return resp, nil
}
```

---

## 🧪 Testing Integration

### Mock Auth Client for Testing
```go
type MockAuthClient struct {
    ValidateTokenFunc func(context.Context, *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error)
}

func (m *MockAuthClient) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
    if m.ValidateTokenFunc != nil {
        return m.ValidateTokenFunc(ctx, req)
    }

    // Default mock response
    return &auth.ValidateTokenResponse{
        Valid:  true,
        UserId: "test-user-id",
        Email:  "test@example.com",
    }, nil
}
```

### Integration Test Example
```go
func TestServiceWithAuth(t *testing.T) {
    // Setup mock auth client
    mockAuth := &MockAuthClient{
        ValidateTokenFunc: func(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
            if req.Token == "valid-token" {
                return &auth.ValidateTokenResponse{
                    Valid:  true,
                    UserId: "user-123",
                    Permissions: []string{"read:project"},
                }, nil
            }
            return &auth.ValidateTokenResponse{Valid: false}, nil
        },
    }

    service := NewYourService(mockAuth)

    // Test with valid token
    result, err := service.GetProject(ctx, "project-123")
    assert.NoError(t, err)
    assert.NotNil(t, result)

    // Test with invalid token
    // ... test invalid scenarios
}
```

---

## 📞 Support and Troubleshooting

### Common Issues

1. **Connection Refused**: Check if auth service is running on correct port
2. **Token Validation Fails**: Ensure token format is correct (Bearer prefix)
3. **Permission Denied**: Verify user has required permissions in auth service
4. **Timeout Errors**: Increase timeout values or check network connectivity

### Debug Commands
```bash
# Check auth service health
curl -k --http2 https://localhost:9001/health/detailed

# Test token validation
curl -k --http2 -H "Authorization: Bearer YOUR_TOKEN" https://localhost:9001/api/v1/user/profile

# Check gRPC connectivity
grpcurl -plaintext localhost:9000 auth.AuthService/HealthCheck
```

### Contact Information
- **Auth Service Team**: auth-team@yourcompany.com
- **Documentation**: [Internal Wiki Link]
- **Issue Tracker**: [JIRA/GitHub Link]
