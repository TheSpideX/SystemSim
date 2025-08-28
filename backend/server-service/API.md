# API Gateway - API Documentation

This document provides comprehensive API documentation for the System Design Website API Gateway.

## 🌐 Base URL

- **Development**: `https://localhost:8000`
- **Production**: `https://api.systemdesign.example.com`

## 🔐 Authentication

Most endpoints require authentication via JWT tokens in the Authorization header:

```http
Authorization: Bearer <jwt-token>
```

## 📋 Common Response Format

### Success Response
```json
{
  "data": { ... },
  "request_id": "api-gateway-1234567890",
  "timestamp": "2023-12-01T12:00:00Z"
}
```

### Error Response
```json
{
  "type": "validation_error",
  "code": "invalid_input",
  "message": "Input validation failed",
  "details": { ... },
  "request_id": "api-gateway-1234567890",
  "timestamp": "2023-12-01T12:00:00Z",
  "path": "/api/auth/validate"
}
```

## 🏥 System Endpoints

### Health Check
Check the overall health of the API Gateway and its dependencies.

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "services": {
    "grpc_services": {
      "auth_service": true,
      "project_service": true,
      "simulation_service": false
    },
    "redis": true,
    "websocket_hub": {
      "active_connections": 42,
      "total_messages": 1337
    }
  },
  "timestamp": 1701432000
}
```

**Status Codes:**
- `200 OK` - All services healthy
- `503 Service Unavailable` - One or more services unhealthy

### Metrics
Get real-time performance metrics and statistics.

```http
GET /metrics
```

**Response:**
```json
{
  "gateway": {
    "requests_processed": 15420,
    "requests_per_second": 45,
    "avg_response_time_ms": 125
  },
  "circuit_breakers": {
    "auth": {
      "state": "CLOSED",
      "requests": 1000,
      "total_successes": 995,
      "total_failures": 5,
      "consecutive_successes": 50,
      "consecutive_failures": 0
    }
  },
  "websocket_hub": {
    "active_connections": 42,
    "total_messages": 1337
  },
  "grpc_clients": {
    "auth_pool": {
      "active_connections": 5,
      "total_requests": 1000
    }
  }
}
```

## 🔑 Authentication Endpoints

### Validate Token
Validate a JWT token and get user information.

```http
POST /api/auth/validate
Authorization: Bearer <jwt-token>
```

**Response:**
```json
{
  "valid": true,
  "user_id": "user-123",
  "expires_at": 1701518400,
  "request_id": "api-gateway-1234567890"
}
```

**Status Codes:**
- `200 OK` - Token is valid
- `401 Unauthorized` - Token is invalid or missing

### Get User Profile
Get the current user's profile information.

```http
GET /api/auth/profile
Authorization: Bearer <jwt-token>
```

**Response:**
```json
{
  "user_id": "user-123",
  "email": "user@example.com",
  "roles": ["user", "developer"],
  "permissions": ["read:projects", "write:projects", "read:simulations"],
  "request_id": "api-gateway-1234567890"
}
```

**Status Codes:**
- `200 OK` - Profile retrieved successfully
- `401 Unauthorized` - Invalid or missing token
- `500 Internal Server Error` - Failed to retrieve profile

### Get User Permissions
Get the current user's permissions and roles.

```http
GET /api/auth/permissions
Authorization: Bearer <jwt-token>
```

**Response:**
```json
{
  "user_id": "user-123",
  "permissions": [
    "read:projects",
    "write:projects",
    "read:simulations",
    "write:simulations"
  ],
  "roles": ["user", "developer"],
  "request_id": "api-gateway-1234567890"
}
```

### Auth Service Health
Check the health of the authentication service.

```http
GET /api/auth/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "auth-service",
  "request_id": "api-gateway-1234567890"
}
```

**Status Codes:**
- `200 OK` - Auth service is healthy
- `503 Service Unavailable` - Auth service is down

## 📁 Project Endpoints

All project endpoints require authentication.

### List Projects
Get a list of projects accessible to the current user.

```http
GET /api/projects
Authorization: Bearer <jwt-token>
```

**Query Parameters:**
- `limit` (optional): Number of projects to return (default: 20)
- `offset` (optional): Number of projects to skip (default: 0)
- `search` (optional): Search term for project names

**Response:**
```json
{
  "projects": [
    {
      "id": "project-123",
      "name": "E-commerce System",
      "description": "Scalable e-commerce platform design",
      "owner_id": "user-123",
      "created_at": "2023-12-01T12:00:00Z",
      "updated_at": "2023-12-01T15:30:00Z",
      "permissions": ["read", "write", "share"]
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

### Create Project
Create a new project.

```http
POST /api/projects
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "name": "New Project",
  "description": "Project description",
  "template": "microservices"
}
```

**Response:**
```json
{
  "id": "project-456",
  "name": "New Project",
  "description": "Project description",
  "owner_id": "user-123",
  "created_at": "2023-12-01T16:00:00Z",
  "updated_at": "2023-12-01T16:00:00Z"
}
```

**Status Codes:**
- `201 Created` - Project created successfully
- `400 Bad Request` - Invalid input data
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions

### Get Project
Get details of a specific project.

```http
GET /api/projects/{project_id}
Authorization: Bearer <jwt-token>
```

**Response:**
```json
{
  "id": "project-123",
  "name": "E-commerce System",
  "description": "Scalable e-commerce platform design",
  "owner_id": "user-123",
  "created_at": "2023-12-01T12:00:00Z",
  "updated_at": "2023-12-01T15:30:00Z",
  "components": [
    {
      "id": "component-1",
      "type": "load_balancer",
      "name": "Main Load Balancer",
      "config": { ... }
    }
  ],
  "permissions": ["read", "write", "share"]
}
```

### Update Project
Update an existing project.

```http
PUT /api/projects/{project_id}
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "name": "Updated Project Name",
  "description": "Updated description"
}
```

### Delete Project
Delete a project.

```http
DELETE /api/projects/{project_id}
Authorization: Bearer <jwt-token>
```

**Status Codes:**
- `204 No Content` - Project deleted successfully
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Project not found

## 🎯 Simulation Endpoints

All simulation endpoints require authentication.

### List Simulations
Get a list of simulations for the current user.

```http
GET /api/simulations
Authorization: Bearer <jwt-token>
```

**Response:**
```json
{
  "simulations": [
    {
      "id": "sim-123",
      "project_id": "project-123",
      "name": "Load Test Simulation",
      "status": "running",
      "created_at": "2023-12-01T14:00:00Z",
      "started_at": "2023-12-01T14:05:00Z",
      "progress": 75
    }
  ],
  "total": 1
}
```

### Create Simulation
Start a new simulation.

```http
POST /api/simulations
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "project_id": "project-123",
  "name": "Performance Test",
  "config": {
    "duration": "5m",
    "load_pattern": "constant",
    "target_rps": 1000
  }
}
```

### Get Simulation
Get details of a specific simulation.

```http
GET /api/simulations/{simulation_id}
Authorization: Bearer <jwt-token>
```

### Start Simulation
Start a simulation.

```http
POST /api/simulations/{simulation_id}/start
Authorization: Bearer <jwt-token>
```

### Stop Simulation
Stop a running simulation.

```http
POST /api/simulations/{simulation_id}/stop
Authorization: Bearer <jwt-token>
```

### Get Simulation Status
Get the current status of a simulation.

```http
GET /api/simulations/{simulation_id}/status
Authorization: Bearer <jwt-token>
```

**Response:**
```json
{
  "id": "sim-123",
  "status": "running",
  "progress": 75,
  "started_at": "2023-12-01T14:05:00Z",
  "estimated_completion": "2023-12-01T14:10:00Z",
  "metrics": {
    "requests_processed": 45000,
    "average_response_time": 125,
    "error_rate": 0.02
  }
}
```

### Get Simulation Results
Get the results of a completed simulation.

```http
GET /api/simulations/{simulation_id}/results
Authorization: Bearer <jwt-token>
```

## 🔌 WebSocket Connection

### Connect to WebSocket
Establish a WebSocket connection for real-time updates.

```http
GET /ws?user_id={user_id}
Upgrade: websocket
Connection: Upgrade
```

**Query Parameters:**
- `user_id`: User identifier for the connection

### WebSocket Message Format

#### Subscribe to Channel
```json
{
  "type": "subscribe",
  "channel": "project:123:updates"
}
```

#### Unsubscribe from Channel
```json
{
  "type": "unsubscribe",
  "channel": "project:123:updates"
}
```

#### Ping/Pong
```json
{
  "type": "ping",
  "timestamp": 1701432000
}
```

#### Event Notification
```json
{
  "type": "event",
  "channel": "project:123:updates",
  "event": "component_added",
  "data": {
    "component_id": "comp-456",
    "component_type": "database",
    "project_id": "project-123"
  },
  "timestamp": 1701432000
}
```

## ❌ Error Codes

### Error Types
- `validation_error` - Input validation failed
- `authentication_error` - Authentication failed
- `authorization_error` - Insufficient permissions
- `not_found_error` - Resource not found
- `conflict_error` - Resource conflict
- `internal_error` - Internal server error
- `service_unavailable_error` - Service temporarily unavailable
- `timeout_error` - Request timeout
- `rate_limit_error` - Rate limit exceeded
- `circuit_breaker_error` - Circuit breaker open

### HTTP Status Codes
- `200 OK` - Request successful
- `201 Created` - Resource created
- `204 No Content` - Request successful, no content
- `400 Bad Request` - Invalid request
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Access denied
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource conflict
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service unavailable

## 🔄 Rate Limiting

The API Gateway implements rate limiting to ensure fair usage:

- **Default Limit**: 1000 requests per minute per user
- **Burst Limit**: 100 requests per second
- **Headers**: Rate limit information is included in response headers

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1701432060
```

## 📝 Request/Response Examples

### cURL Examples

```bash
# Health check
curl -k https://localhost:8000/health

# Get metrics
curl -k https://localhost:8000/metrics

# Validate token
curl -k -X POST https://localhost:8000/api/auth/validate \
  -H "Authorization: Bearer your-jwt-token"

# List projects
curl -k https://localhost:8000/api/projects \
  -H "Authorization: Bearer your-jwt-token"

# Create project
curl -k -X POST https://localhost:8000/api/projects \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Project","description":"A test project"}'
```

### JavaScript Examples

```javascript
// Fetch with authentication
const response = await fetch('https://localhost:8000/api/projects', {
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  }
});

const projects = await response.json();

// WebSocket connection
const ws = new WebSocket('wss://localhost:8000/ws?user_id=user-123');

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'subscribe',
    channel: 'project:123:updates'
  }));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};
```

This API documentation provides comprehensive coverage of all available endpoints, request/response formats, authentication requirements, and usage examples.
