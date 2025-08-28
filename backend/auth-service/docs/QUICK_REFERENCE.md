# Auth Service Quick Reference

## 🚀 Service Endpoints

### HTTP/2 API (Client-Facing)
- **Base URL**: `https://localhost:9001/api/v1`
- **Health**: `https://localhost:9001/health`
- **Metrics**: `https://localhost:9001/metrics`

### gRPC API (Inter-Service)
- **Address**: `localhost:9000`
- **Proto**: `api/proto/auth.proto`

### PostgreSQL (Primary Database)
- **Address**: `localhost:5432`
- **Database**: `auth_service_db`
- **Used For**: Users, roles, permissions, audit logs

### Redis (High-Performance Operations)
- **Address**: `localhost:6379`
- **Databases**: `0` (sessions), `1` (cache), `2` (pub/sub)
- **Used For**: Session storage, caching, pub/sub, background services

---

## 🔑 Most Common Operations

### 1. Validate Token (API Gateway)
```go
resp, err := authClient.ValidateToken(ctx, &auth.ValidateTokenRequest{
    Token:          "Bearer jwt_token",
    CallingService: "api-gateway",
    RequestId:      generateRequestID(),
})

if err != nil || !resp.Valid {
    // Token invalid
    return
}

userID := resp.UserId
permissions := resp.Permissions
```

### 2. Check Permission (Any Service)
```go
resp, err := authClient.CheckPermission(ctx, &auth.CheckPermissionRequest{
    UserId:         userID,
    Permission:     "read:project",
    ResourceId:     projectID,
    CallingService: "project-service",
    RequestId:      generateRequestID(),
})

if err != nil || !resp.Allowed {
    // Permission denied
    return fmt.Errorf("access denied: %s", resp.Reason)
}
```

### 3. Get User Context (Any Service)
```go
resp, err := authClient.GetUserContext(ctx, &auth.GetUserContextRequest{
    UserId:         userID,
    CallingService: "simulation-service",
    RequestId:      generateRequestID(),
})

if err != nil {
    return err
}

email := resp.Email
company := resp.Company
roles := resp.Roles
permissions := resp.Permissions
```

---

## 🔧 Client Setup

### Basic gRPC Client
```go
conn, err := grpc.Dial("localhost:9001", 
    grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
    return err
}
defer conn.Close()

client := auth.NewAuthServiceClient(conn)
```

### With Connection Pooling (Recommended)
```go
// Use the mesh connection pool
func (s *Service) validateToken(token string) (*auth.ValidateTokenResponse, error) {
    return s.meshClient.CallWithRetry(ctx, "auth-service", func(conn *grpc.ClientConn) (*auth.ValidateTokenResponse, error) {
        client := auth.NewAuthServiceClient(conn)
        return client.ValidateToken(ctx, &auth.ValidateTokenRequest{
            Token:          token,
            CallingService: s.serviceName,
            RequestId:      generateRequestID(),
        })
    })
}
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
    return err
}

if !resp.Valid {
    // Application-level error
    log.Error("Token invalid", "reason", resp.ErrorMessage)
}
```

### HTTP Errors
- `401`: Invalid/expired token
- `403`: Insufficient permissions
- `404`: User/resource not found
- `429`: Rate limited

---

## 🔒 Security Best Practices

### ✅ DO
- Always validate tokens on every request
- Include `CallingService` and `RequestId` in all requests
- Check permissions before operations
- Use connection pooling for performance
- Implement circuit breakers
- Add proper logging and metrics

### ❌ DON'T
- Cache token validation results
- Skip validation for "internal" requests
- Trust tokens without validation
- Hardcode service URLs
- Ignore error responses

---

## 🚀 Environment Variables

```bash
# Required
AUTH_SERVICE_GRPC_URL=auth-service:9000
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

### Mock Client
```go
type MockAuthClient struct {
    responses map[string]*auth.ValidateTokenResponse
}

func (m *MockAuthClient) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
    if resp, ok := m.responses[req.Token]; ok {
        return resp, nil
    }
    return &auth.ValidateTokenResponse{Valid: false}, nil
}

// In tests
mockAuth := &MockAuthClient{
    responses: map[string]*auth.ValidateTokenResponse{
        "valid-token": {
            Valid:  true,
            UserId: "test-user",
            Permissions: []string{"read:project"},
        },
    },
}
```

---

## 📊 Health Checks

### HTTP/2 Health Check
```bash
curl -k --http2 https://localhost:9001/health
```

### gRPC Health Check
```go
resp, err := authClient.HealthCheck(ctx, &auth.HealthCheckRequest{
    CallingService: "your-service",
    RequestId:      "health-check",
})
```

---

## 🐛 Debugging

### Check Service Status
```bash
# HTTP/2 health
curl -k --http2 https://localhost:9001/health/detailed

# gRPC connectivity
grpcurl -plaintext localhost:9000 auth.AuthService/HealthCheck
```

### Common Issues
1. **Connection refused**: Service not running or wrong port
2. **Token validation fails**: Check token format (Bearer prefix)
3. **Permission denied**: User lacks required permissions
4. **Timeouts**: Network issues or service overload

---

## 📝 Request ID Generation

```go
func generateRequestID() string {
    return fmt.Sprintf("%s-%d", 
        uuid.New().String()[:8], 
        time.Now().UnixNano())
}
```

---

## 🔄 Retry Logic

```go
func (c *AuthClient) ValidateTokenWithRetry(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
    var resp *auth.ValidateTokenResponse
    var err error
    
    for i := 0; i < 3; i++ {
        resp, err = c.client.ValidateToken(ctx, req)
        if err == nil {
            return resp, nil
        }
        
        if st, ok := status.FromError(err); ok {
            if st.Code() == codes.Unavailable {
                time.Sleep(time.Duration(i+1) * time.Second)
                continue
            }
        }
        
        return nil, err
    }
    
    return nil, err
}
```

---

## 📞 Support

- **Full Documentation**: `docs/SERVICE_INTEGRATION.md`
- **Proto Files**: `api/proto/auth.proto`
- **Health Endpoint**: `https://localhost:9001/health/detailed`
