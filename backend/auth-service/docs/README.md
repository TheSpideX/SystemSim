# Auth Service Documentation

Welcome to the Auth Service documentation! This service provides authentication and authorization for the entire microservice mesh.

## 📚 Documentation Index

### For Service Developers
- **[Service Integration Guide](SERVICE_INTEGRATION.md)** - Complete guide for integrating with auth service
- **[Quick Reference](QUICK_REFERENCE.md)** - Common operations and quick examples
- **[gRPC API Reference](GRPC_API_REFERENCE.md)** - Detailed gRPC method documentation

### For Auth Service Developers
- **[Architecture Overview](../README.md)** - Service architecture and design
- **[API Documentation](../api/README.md)** - HTTP and gRPC API specifications
- **[Development Guide](../DEVELOPMENT.md)** - Local development setup

---

## 🚀 Quick Start for Other Services

### 1. Service Information
- **HTTP/2 Port**: `9001` (client-facing with TLS)
- **gRPC Port**: `9000` (inter-service mesh)
- **PostgreSQL Port**: `5432` (persistent data storage)
- **Redis Port**: `6379` (pub/sub, sessions, caching, background services)
- **Health Check**: `https://localhost:9001/health`

### 2. Most Common Operations

#### Validate Token (API Gateway)
```go
resp, err := authClient.ValidateToken(ctx, &auth.ValidateTokenRequest{
    Token:          "Bearer jwt_token",
    CallingService: "api-gateway",
    RequestId:      generateRequestID(),
})
```

#### Check Permission (Any Service)
```go
resp, err := authClient.CheckPermission(ctx, &auth.CheckPermissionRequest{
    UserId:         userID,
    Permission:     "read:project",
    ResourceId:     projectID,
    CallingService: "project-service",
    RequestId:      generateRequestID(),
})
```

#### Get User Context (Any Service)
```go
resp, err := authClient.GetUserContext(ctx, &auth.GetUserContextRequest{
    UserId:         userID,
    CallingService: "simulation-service",
    RequestId:      generateRequestID(),
})
```

### 3. Client Setup
```go
// Basic setup
conn, err := grpc.Dial("localhost:9001", 
    grpc.WithTransportCredentials(insecure.NewCredentials()))
client := auth.NewAuthServiceClient(conn)

// With connection pooling (recommended)
authClient := mesh.NewAuthClient(poolManager)
```

---

## 🏗️ Integration Patterns

### API Gateway Pattern
The API Gateway validates all incoming requests and adds user context:

```go
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

### Service-to-Service Authorization
Services check permissions before operations:

```go
func (ps *ProjectService) DeleteProject(ctx context.Context, projectID string) error {
    userID := getUserIDFromContext(ctx)
    
    resp, err := ps.authClient.CheckPermission(ctx, &auth.CheckPermissionRequest{
        UserId:         userID,
        Permission:     "delete:project",
        ResourceId:     projectID,
        CallingService: "project-service",
        RequestId:      generateRequestID(),
    })
    
    if err != nil || !resp.Allowed {
        return fmt.Errorf("permission denied: %s", resp.Reason)
    }
    
    return ps.deleteProjectFromDB(projectID)
}
```

---

## 🔒 Security Best Practices

### ✅ Always Do
- Validate tokens on every request
- Check permissions before operations
- Include `CallingService` and `RequestId` in requests
- Use connection pooling for performance
- Implement proper error handling
- Add logging and metrics

### ❌ Never Do
- Cache token validation results
- Skip validation for "internal" requests
- Trust tokens without validation
- Hardcode service URLs
- Ignore error responses

---

## 🚀 Environment Configuration

```bash
# Required
AUTH_SERVICE_GRPC_URL=auth-service:9001
AUTH_REQUEST_TIMEOUT=10s
REDIS_URL=redis://redis:6379
AUTH_DB_HOST=postgres
AUTH_DB_READ_USER=auth_readonly
AUTH_DB_READ_PASSWORD=readonly_password

# Optional
AUTH_GRPC_MAX_CONNECTIONS=10
AUTH_GRPC_MIN_CONNECTIONS=2
AUTH_MAX_RETRIES=3
AUTH_RETRY_DELAY=1s
REDIS_PASSWORD=your_redis_password
REDIS_PUBSUB_ENABLED=true
BACKGROUND_TASKS_ENABLED=true
```

---

## 🧪 Testing

### Mock Client for Unit Tests
```go
type MockAuthClient struct {
    ValidateTokenFunc func(context.Context, *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error)
}

func (m *MockAuthClient) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
    if m.ValidateTokenFunc != nil {
        return m.ValidateTokenFunc(ctx, req)
    }
    
    return &auth.ValidateTokenResponse{
        Valid:  true,
        UserId: "test-user-id",
        Email:  "test@example.com",
    }, nil
}
```

---

## 📊 Health Monitoring

### HTTP/2 Health Checks
```bash
curl -k --http2 https://localhost:9001/health          # Basic
curl -k --http2 https://localhost:9001/health/detailed # Detailed
curl -k --http2 https://localhost:9001/health/ready    # Readiness
curl -k --http2 https://localhost:9001/health/live     # Liveness
curl -k --http2 https://localhost:9001/metrics         # Metrics
```

### gRPC Health Check
```go
resp, err := authClient.HealthCheck(ctx, &auth.HealthCheckRequest{
    CallingService: "your-service",
    RequestId:      "health-check",
})
```

---

## ⚠️ Error Handling

### gRPC Errors
```go
resp, err := authClient.ValidateToken(ctx, req)
if err != nil {
    if st, ok := status.FromError(err); ok {
        switch st.Code() {
        case codes.InvalidArgument:
            // Bad request
        case codes.Unauthenticated:
            // Invalid token
        case codes.Unavailable:
            // Service down
        }
    }
}

if !resp.Valid {
    // Application error: resp.ErrorMessage
}
```

### HTTP Status Codes
- `401 Unauthorized`: Invalid/expired token
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: User/resource not found
- `429 Too Many Requests`: Rate limited

---

## 🐛 Troubleshooting

### Common Issues
1. **Connection Refused**: Check if auth service is running on port 9001 with HTTP/2
2. **Token Validation Fails**: Ensure token has "Bearer " prefix
3. **Permission Denied**: Verify user has required permissions
4. **Timeout Errors**: Check network connectivity or increase timeouts
5. **Certificate Errors**: Use `-k` flag for self-signed certificates in development

### Debug Commands
```bash
# Check service health
curl -k --http2 https://localhost:9001/health/detailed

# Test token validation
curl -k --http2 -H "Authorization: Bearer YOUR_TOKEN" \
     https://localhost:9001/api/v1/user/profile

# Check gRPC connectivity
grpcurl -plaintext localhost:9000 auth.AuthService/HealthCheck
```

---

## 📞 Support

### Documentation
- **[Complete Integration Guide](SERVICE_INTEGRATION.md)** - Comprehensive integration documentation
- **[Quick Reference](QUICK_REFERENCE.md)** - Common operations and examples
- **[gRPC API Reference](GRPC_API_REFERENCE.md)** - Detailed API documentation

### Resources
- **Proto Files**: `api/proto/auth.proto`
- **Health Endpoint**: `https://localhost:9001/health/detailed`
- **Metrics**: `https://localhost:9001/metrics`

### Getting Help
1. Check the documentation above
2. Verify service health endpoints
3. Check logs for error details
4. Contact the auth service team

---

## 🔄 Service Mesh Architecture

The auth service is part of a microservice mesh with the following communication patterns:

```
┌─────────────┐    HTTP     ┌─────────────┐    gRPC    ┌─────────────┐
│   Client    │ ────────── │ API Gateway │ ────────── │ Auth Service│
└─────────────┘            └─────────────┘            └─────────────┘
                                  │                           │
                                  │ gRPC                      │ SQL
                                  ▼                           ▼
                           ┌─────────────┐            ┌─────────────┐
                           │   Project   │            │ PostgreSQL  │
                           │   Service   │            │ (Users,     │
                           └─────────────┘            │Roles,Perms) │
                                  │                   └─────────────┘
                                  │ gRPC                      │
                                  ▼                           │ Redis
                           ┌─────────────┐            ┌─────────────┐
                           │ Simulation  │            │    Redis    │
                           │   Service   │            │ (Sessions,  │
                           └─────────────┘            │Cache,PubSub)│
                                  │                   └─────────────┘
                                  │ Redis Pub/Sub              │
                                  └────────────────────────────┘
                                         Background Services
```

### Communication Flow
1. **Client → API Gateway**: HTTP requests with JWT tokens
2. **API Gateway → Auth Service**: gRPC token validation
3. **Services → Auth Service**: gRPC permission checks and user context
4. **Auth Service → PostgreSQL**: Persistent data storage (users, roles, permissions)
5. **Auth Service → Redis**: Session storage, caching, pub/sub events
6. **Services → Redis**: Pub/sub subscription for background services and high-throughput data
7. **Services → PostgreSQL**: Read-only access for user validation (optional)
8. **All Services**: Health monitoring and metrics collection

This architecture ensures centralized authentication while maintaining high performance through gRPC mesh communication.
