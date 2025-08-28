# API Gateway (Server Service) - Complete API Documentation

## 🎯 **Overview**

The API Gateway serves as the **single entry point** for all client requests in the System Design Website. It provides HTTP/2-enabled endpoints, WebSocket connections for real-time features, and acts as a proxy to backend microservices through gRPC communication.

**Base URL**: `https://localhost:8000` (HTTP/2 + TLS)  
**Version**: 1.0.0  
**Protocol**: HTTP/2 (strict, no HTTP/1.1 fallback)

---

## 📋 **Table of Contents**

1. [System Endpoints](#-system-endpoints)
2. [Authentication Endpoints](#-authentication-endpoints)
3. [Project Endpoints](#-project-endpoints)
4. [Simulation Endpoints](#-simulation-endpoints)
5. [WebSocket Endpoints](#-websocket-endpoints)
6. [Error Handling](#-error-handling)
7. [Authentication & Authorization](#-authentication--authorization)
8. [Rate Limiting](#-rate-limiting)
9. [Performance & Monitoring](#-performance--monitoring)

---

## 🏥 **System Endpoints**

### Health Check (Aggregated)
Get comprehensive health status of all microservices and dependencies.

```http
GET /health
```

**Response (200 OK):**
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
      "total_messages": 1337,
      "messages_per_second": 25.5
    }
  },
  "response_time_ms": 12,
  "timestamp": 1701432000
}
```

**Response (503 Service Unavailable):**
```json
{
  "status": "degraded",
  "services": {
    "grpc_services": {
      "auth_service": true,
      "project_service": false,
      "simulation_service": true
    },
    "redis": true,
    "websocket_hub": {
      "active_connections": 42,
      "total_messages": 1337
    }
  },
  "errors": [
    "project_service: connection timeout",
    "simulation_service: unhealthy status"
  ],
  "timestamp": 1701432000
}
```

### Individual Service Health
Check health of specific backend services.

```http
GET /health/auth
GET /health/project
GET /health/simulation
```

**Response:**
```json
{
  "status": "healthy",
  "service": "auth-service",
  "response_time_ms": 5,
  "details": {
    "database": "healthy",
    "redis": "healthy",
    "uptime": "2h30m15s"
  },
  "request_id": "api-gateway-1234567890"
}
```

### gRPC Connection Statistics
Monitor gRPC connection pools and performance.

```http
GET /grpc/stats
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": 1701432000,
  "grpc_pools": {
    "auth_service": {
      "active_connections": 8,
      "total_requests": 15420,
      "error_rate": 0.02,
      "avg_latency_ms": 12.5,
      "pool_utilization": 0.4
    },
    "project_service": {
      "active_connections": 5,
      "total_requests": 8930,
      "error_rate": 0.01,
      "avg_latency_ms": 18.2,
      "pool_utilization": 0.25
    },
    "simulation_service": {
      "active_connections": 12,
      "total_requests": 45230,
      "error_rate": 0.05,
      "avg_latency_ms": 45.8,
      "pool_utilization": 0.6
    }
  }
}
```

### Performance Metrics
Get API Gateway performance statistics.

```http
GET /metrics
```

**Response:**
```json
{
  "requests_processed": 125430,
  "requests_per_second": 245.8,
  "avg_response_time_ms": 28.5,
  "error_rate": 0.02,
  "websocket_connections": 1250,
  "uptime_seconds": 86400,
  "memory_usage_mb": 128.5,
  "cpu_usage_percent": 15.2
}
```

---

## 🔐 **Authentication Endpoints**

All authentication endpoints proxy to the Auth Service with additional gateway-level processing.

### User Registration
Register a new user account.

```http
POST /api/auth/register
Content-Type: application/json
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "first_name": "John",
  "last_name": "Doe",
  "company": "Tech Corp"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "User registered successfully",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "email_verified": false
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "token_type": "Bearer"
  }
}
```

### User Login
Authenticate user and receive JWT tokens.

```http
POST /api/auth/login
Content-Type: application/json
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "remember_me": true
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Login successful",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "is_admin": false,
    "last_login": "2023-12-01T12:00:00Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "token_type": "Bearer"
  }
}
```

### Token Refresh
Refresh expired access token using refresh token.

```http
POST /api/auth/refresh
Content-Type: application/json
```

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "token_type": "Bearer"
  }
}
```

### User Logout
Invalidate current session and tokens.

```http
POST /api/auth/logout
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Successfully logged out"
}
```

### Password Reset Request
Request password reset email.

```http
POST /api/auth/forgot-password
Content-Type: application/json
```

**Request Body:**
```json
{
  "email": "user@example.com"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Password reset email sent"
}
```

### Password Reset Confirmation
Reset password using reset token.

```http
POST /api/auth/reset-password
Content-Type: application/json
```

**Request Body:**
```json
{
  "token": "reset_token_here",
  "new_password": "NewSecurePassword123!"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Password reset successfully"
}
```

### Email Verification
Verify user email address.

```http
POST /api/auth/verify-email
Content-Type: application/json
```

**Request Body:**
```json
{
  "token": "verification_token_here"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Email verified successfully"
}
```

---

## 🔒 **Protected Authentication Endpoints**

These endpoints require valid JWT token in Authorization header.

### Token Validation (Gateway-specific)
Validate JWT token and get user context.

```http
POST /api/auth/validate
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK):**
```json
{
  "valid": true,
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "is_admin": false,
  "permissions": ["read:projects", "write:projects"],
  "expires_at": 1701432900
}
```

### Get User Profile
Get current user profile information.

```http
GET /api/auth/profile
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "company": "Tech Corp",
    "email_verified": true,
    "is_admin": false,
    "created_at": "2023-11-01T10:00:00Z",
    "last_login": "2023-12-01T12:00:00Z"
  },
  "preferences": {
    "simulation_preferences": {},
    "ui_preferences": {}
  }
}
```

### Get User Permissions
Get current user's roles and permissions.

```http
GET /api/auth/permissions
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "roles": ["user", "project_member"],
  "permissions": [
    "read:projects",
    "write:projects",
    "read:simulations",
    "write:simulations"
  ],
  "is_admin": false
}
```

### Update User Profile
Update current user profile information.

```http
PUT /api/auth/profile
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "first_name": "John",
  "last_name": "Smith",
  "company": "New Tech Corp"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Profile updated successfully",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Smith",
    "company": "New Tech Corp"
  }
}
```

### Change Password
Change user password.

```http
PUT /api/auth/password
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "current_password": "CurrentPassword123!",
  "new_password": "NewSecurePassword123!"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Password changed successfully"
}
```

### Get User Sessions
Get all active sessions for current user.

```http
GET /api/auth/sessions
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "sessions": [
    {
      "id": "session_id_1",
      "device_info": {
        "browser": "Chrome",
        "os": "Windows",
        "device": "Desktop"
      },
      "ip_address": "192.168.1.100",
      "last_used": "2023-12-01T12:00:00Z",
      "is_current": true
    },
    {
      "id": "session_id_2",
      "device_info": {
        "browser": "Safari",
        "os": "iOS",
        "device": "Mobile"
      },
      "ip_address": "192.168.1.101",
      "last_used": "2023-11-30T18:30:00Z",
      "is_current": false
    }
  ]
}
```

### Revoke Session
Revoke a specific user session.

```http
DELETE /api/auth/sessions/{session_id}
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Session revoked successfully"
}
```

---

## 📁 **Project Endpoints**

Project endpoints require authentication and route to the Project Service.

### List Projects
Get all projects accessible to the current user.

```http
GET /api/projects
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)
- `search` (optional): Search term for project name/description
- `status` (optional): Filter by project status (active, archived, draft)

**Response (200 OK):**
```json
{
  "projects": [
    {
      "id": "project_id_1",
      "name": "E-commerce System Design",
      "description": "Scalable e-commerce platform architecture",
      "status": "active",
      "owner": {
        "id": "user_id_1",
        "name": "John Doe"
      },
      "collaborators": 3,
      "created_at": "2023-11-01T10:00:00Z",
      "updated_at": "2023-12-01T12:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

### Create Project
Create a new project.

```http
POST /api/projects
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "New System Design",
  "description": "Description of the system design project",
  "template": "microservices",
  "visibility": "private"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "Project created successfully",
  "project": {
    "id": "new_project_id",
    "name": "New System Design",
    "description": "Description of the system design project",
    "status": "draft",
    "owner": {
      "id": "user_id",
      "name": "John Doe"
    },
    "visibility": "private",
    "created_at": "2023-12-01T12:00:00Z"
  }
}
```

### Get Project Details
Get detailed information about a specific project.

```http
GET /api/projects/{project_id}
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "project": {
    "id": "project_id",
    "name": "E-commerce System Design",
    "description": "Scalable e-commerce platform architecture",
    "status": "active",
    "owner": {
      "id": "user_id_1",
      "name": "John Doe"
    },
    "collaborators": [
      {
        "id": "user_id_2",
        "name": "Jane Smith",
        "role": "editor",
        "joined_at": "2023-11-15T10:00:00Z"
      }
    ],
    "components": [
      {
        "id": "component_1",
        "type": "database",
        "name": "User Database",
        "position": {"x": 100, "y": 200}
      }
    ],
    "created_at": "2023-11-01T10:00:00Z",
    "updated_at": "2023-12-01T12:00:00Z"
  }
}
```

### Update Project
Update project information.

```http
PUT /api/projects/{project_id}
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Updated Project Name",
  "description": "Updated description",
  "status": "active"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Project updated successfully",
  "project": {
    "id": "project_id",
    "name": "Updated Project Name",
    "description": "Updated description",
    "status": "active",
    "updated_at": "2023-12-01T12:30:00Z"
  }
}
```

### Delete Project
Delete a project (owner only).

```http
DELETE /api/projects/{project_id}
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Project deleted successfully"
}
```

---

## 🎮 **Simulation Endpoints**

Simulation endpoints route to the Simulation Service for running system design simulations.

### List Simulations
Get all simulations for a project.

```http
GET /api/projects/{project_id}/simulations
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "simulations": [
    {
      "id": "simulation_id_1",
      "name": "Load Test Simulation",
      "status": "completed",
      "type": "load_test",
      "created_at": "2023-12-01T10:00:00Z",
      "completed_at": "2023-12-01T10:15:00Z",
      "results": {
        "max_rps": 10000,
        "avg_latency_ms": 45,
        "success_rate": 99.8
      }
    }
  ]
}
```

### Create Simulation
Start a new simulation.

```http
POST /api/projects/{project_id}/simulations
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Performance Test",
  "type": "load_test",
  "parameters": {
    "duration_seconds": 300,
    "concurrent_users": 1000,
    "ramp_up_time": 60
  }
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "Simulation started successfully",
  "simulation": {
    "id": "new_simulation_id",
    "name": "Performance Test",
    "status": "running",
    "type": "load_test",
    "started_at": "2023-12-01T12:00:00Z",
    "estimated_completion": "2023-12-01T12:05:00Z"
  }
}
```

### Get Simulation Status
Get current status and results of a simulation.

```http
GET /api/simulations/{simulation_id}
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "simulation": {
    "id": "simulation_id",
    "name": "Performance Test",
    "status": "running",
    "progress": 65,
    "type": "load_test",
    "started_at": "2023-12-01T12:00:00Z",
    "estimated_completion": "2023-12-01T12:05:00Z",
    "current_metrics": {
      "current_rps": 8500,
      "avg_latency_ms": 42,
      "error_rate": 0.2
    }
  }
}
```

---

## 🔌 **WebSocket Endpoints**

WebSocket endpoints provide real-time communication for collaborative features and live updates.

### Generic WebSocket Connection
General-purpose WebSocket connection for system notifications.

```
WSS /ws
Authorization: Bearer <access_token> (via query param or header)
```

**Connection URL:**
```
wss://localhost:8000/ws?token=<access_token>
```

**Message Format:**
```json
{
  "type": "message_type",
  "channel": "channel_name",
  "data": { ... },
  "timestamp": "2023-12-01T12:00:00Z"
}
```

**Supported Message Types:**
- `notification` - System notifications
- `user_activity` - User activity updates
- `system_announcement` - System-wide announcements

### Project Collaboration WebSocket
Real-time collaboration for project editing.

```
WSS /ws/project/{project_id}
Authorization: Bearer <access_token>
```

**Connection URL:**
```
wss://localhost:8000/ws/project/project_id_123?token=<access_token>
```

**Message Types:**
- `project_update` - Project changes
- `component_added` - New component added
- `component_updated` - Component modified
- `component_deleted` - Component removed
- `user_joined` - User joined project
- `user_left` - User left project
- `cursor_position` - Real-time cursor tracking

**Example Messages:**
```json
{
  "type": "component_updated",
  "channel": "project:project_id_123",
  "data": {
    "component_id": "comp_123",
    "changes": {
      "position": {"x": 150, "y": 250},
      "name": "Updated Component Name"
    },
    "user": {
      "id": "user_id",
      "name": "John Doe"
    }
  },
  "timestamp": "2023-12-01T12:00:00Z"
}
```

---

## ⚠️ **Error Handling**

### Standard Error Response Format
All API endpoints return errors in a consistent format:

```json
{
  "type": "error_type",
  "code": "specific_error_code",
  "message": "Human-readable error message",
  "details": {
    "field": "Additional error details"
  },
  "request_id": "api-gateway-1234567890",
  "timestamp": "2023-12-01T12:00:00Z",
  "path": "/api/endpoint/path"
}
```

### HTTP Status Codes

| Status Code | Description | Usage |
|-------------|-------------|-------|
| 200 | OK | Successful request |
| 201 | Created | Resource created successfully |
| 400 | Bad Request | Invalid request format or parameters |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource not found |
| 409 | Conflict | Resource conflict (e.g., duplicate email) |
| 422 | Unprocessable Entity | Validation errors |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |
| 502 | Bad Gateway | Backend service error |
| 503 | Service Unavailable | Service temporarily unavailable |
| 504 | Gateway Timeout | Backend service timeout |

---

## 🔐 **Authentication & Authorization**

### JWT Token Format
All authenticated requests require a JWT token in the Authorization header:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Permission System
The system uses a permission-based authorization model:

#### Available Permissions
- `read:projects` - View projects
- `write:projects` - Create/edit projects
- `delete:projects` - Delete projects
- `read:simulations` - View simulations
- `write:simulations` - Create/run simulations
- `admin:users` - Manage users (admin only)
- `admin:system` - System administration (admin only)

---

## 🚦 **Rate Limiting**

### Rate Limits
Different endpoints have different rate limits:

| Endpoint Category | Rate Limit | Window |
|------------------|------------|---------|
| Authentication | 10 requests | 1 minute |
| General API | 1000 requests | 1 hour |
| WebSocket connections | 100 connections | 1 hour |
| File uploads | 50 requests | 1 hour |

### Rate Limit Headers
Rate limit information is included in response headers:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1701435600
X-RateLimit-Window: 3600
```

---

## 📊 **Performance & Monitoring**

### Response Time Targets
- Health checks: < 50ms
- Authentication: < 200ms
- Project operations: < 500ms
- Simulation operations: < 1000ms

### Monitoring Endpoints
- `/health` - Overall system health
- `/health/auth` - Auth service health
- `/health/project` - Project service health
- `/health/simulation` - Simulation service health
- `/metrics` - Performance metrics
- `/grpc/stats` - gRPC connection statistics

---

*Last Updated: 2023-12-01*
*API Gateway Version: 1.0.0*
