# gRPC API Reference

## Service Definition

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

---

## 🔐 ValidateToken

**Use Case**: Validate JWT tokens (most common operation)

### Request
```protobuf
message ValidateTokenRequest {
  string token = 1;           // JWT token with "Bearer " prefix
  string calling_service = 2; // Name of calling service
  string request_id = 3;      // Unique request identifier
}
```

### Response
```protobuf
message ValidateTokenResponse {
  bool valid = 1;             // Token validity
  string user_id = 2;         // User UUID
  string email = 3;           // User email
  bool is_admin = 4;          // Admin status
  string session_id = 5;      // Session UUID
  repeated string permissions = 6; // User permissions
  int64 expires_at = 7;       // Token expiration (Unix timestamp)
  string error_message = 8;   // Error details if invalid
}
```

### Example
```go
req := &auth.ValidateTokenRequest{
    Token:          "Bearer eyJhbGciOiJIUzI1NiIs...",
    CallingService: "api-gateway",
    RequestId:      "req-123",
}

resp, err := client.ValidateToken(ctx, req)
if err != nil {
    // Handle gRPC error
}

if resp.Valid {
    userID := resp.UserId
    permissions := resp.Permissions
    // Proceed with authenticated request
} else {
    // Token invalid: resp.ErrorMessage
}
```

---

## 👤 GetUserContext

**Use Case**: Get complete user information for request processing

### Request
```protobuf
message GetUserContextRequest {
  string user_id = 1;         // User UUID
  string calling_service = 2; // Name of calling service
  string request_id = 3;      // Unique request identifier
}
```

### Response
```protobuf
message GetUserContextResponse {
  string user_id = 1;         // User UUID
  string email = 2;           // User email
  string first_name = 3;      // First name
  string last_name = 4;       // Last name
  string company = 5;         // Company name
  repeated string roles = 6;   // User roles
  repeated string permissions = 7; // User permissions
  bool is_active = 8;         // Account status
  bool is_admin = 9;          // Admin status
  int64 last_login = 10;      // Last login (Unix timestamp)
  string last_login_ip = 11;  // Last login IP address
  bool email_verified = 12;   // Email verification status
  string error_message = 13;  // Error details if failed
}
```

### Example
```go
req := &auth.GetUserContextRequest{
    UserId:         "user-uuid-here",
    CallingService: "project-service",
    RequestId:      "ctx-456",
}

resp, err := client.GetUserContext(ctx, req)
if err != nil {
    // Handle gRPC error
}

if resp.ErrorMessage == "" {
    email := resp.Email
    company := resp.Company
    permissions := resp.Permissions
    // Use user context
} else {
    // User not found or error
}
```

---

## 🔒 CheckPermission

**Use Case**: Check if user has specific permission for resource

### Request
```protobuf
message CheckPermissionRequest {
  string user_id = 1;         // User UUID
  string permission = 2;      // Permission to check (e.g., "read:project")
  string resource_id = 3;     // Resource identifier (optional)
  string calling_service = 4; // Name of calling service
  string request_id = 5;      // Unique request identifier
}
```

### Response
```protobuf
message CheckPermissionResponse {
  bool allowed = 1;           // Permission granted
  string reason = 2;          // Reason for decision
  string user_id = 3;         // User UUID (echo)
  string permission = 4;      // Permission (echo)
  string resource_id = 5;     // Resource ID (echo)
  string error_message = 6;   // Error details if failed
}
```

### Example
```go
req := &auth.CheckPermissionRequest{
    UserId:         "user-uuid-here",
    Permission:     "delete:project",
    ResourceId:     "project-123",
    CallingService: "project-service",
    RequestId:      "perm-789",
}

resp, err := client.CheckPermission(ctx, req)
if err != nil {
    // Handle gRPC error
}

if resp.Allowed {
    // Permission granted
} else {
    // Permission denied: resp.Reason
}
```

---

## 🎫 ValidateSession

**Use Case**: Validate session tokens and check session status

### Request
```protobuf
message ValidateSessionRequest {
  string session_id = 1;      // Session UUID
  string user_id = 2;         // User UUID
  string calling_service = 3; // Name of calling service
  string request_id = 4;      // Unique request identifier
}
```

### Response
```protobuf
message ValidateSessionResponse {
  bool valid = 1;             // Session validity
  string session_id = 2;      // Session UUID (echo)
  string user_id = 3;         // User UUID (echo)
  bool is_active = 4;         // Session active status
  int64 expires_at = 5;       // Session expiration (Unix timestamp)
  int64 last_used_at = 6;     // Last used (Unix timestamp)
  string device_info = 7;     // Device information
  string ip_address = 8;      // IP address
  string error_message = 9;   // Error details if invalid
}
```

---

## 🔑 GetUserPermissions

**Use Case**: Get all permissions and optionally roles for a user

### Request
```protobuf
message GetUserPermissionsRequest {
  string user_id = 1;         // User UUID
  string calling_service = 2; // Name of calling service
  string request_id = 3;      // Unique request identifier
  bool include_roles = 4;     // Include role information
}
```

### Response
```protobuf
message GetUserPermissionsResponse {
  string user_id = 1;         // User UUID (echo)
  repeated string permissions = 2; // User permissions
  repeated UserRole roles = 3; // User roles (if requested)
  bool is_admin = 4;          // Admin status
  string error_message = 5;   // Error details if failed
}

message UserRole {
  string role_id = 1;         // Role UUID
  string role_name = 2;       // Role name
  string description = 3;     // Role description
  bool is_system = 4;         // System role flag
  repeated string permissions = 5; // Role permissions
}
```

---

## 🏥 HealthCheck

**Use Case**: Check service health and connectivity

### Request
```protobuf
message HealthCheckRequest {
  string calling_service = 1; // Name of calling service
  string request_id = 2;      // Unique request identifier
}
```

### Response
```protobuf
message HealthCheckResponse {
  string status = 1;          // Health status ("healthy", "unhealthy")
  string version = 2;         // Service version
  int64 timestamp = 3;        // Response timestamp
  map<string, string> details = 4; // Additional health details
}
```

### Example
```go
req := &auth.HealthCheckRequest{
    CallingService: "monitoring-service",
    RequestId:      "health-check",
}

resp, err := client.HealthCheck(ctx, req)
if err != nil {
    // Service unavailable
}

if resp.Status == "healthy" {
    // Service is healthy
} else {
    // Service has issues
}
```

---

## 🔄 Common Patterns

### 1. API Gateway Authentication Flow
```go
// 1. Extract token from request
token := extractBearerToken(r)

// 2. Validate token
validateResp, err := authClient.ValidateToken(ctx, &auth.ValidateTokenRequest{
    Token:          token,
    CallingService: "api-gateway",
    RequestId:      generateRequestID(),
})

if err != nil || !validateResp.Valid {
    http.Error(w, "Unauthorized", 401)
    return
}

// 3. Add user context to request
ctx = context.WithValue(ctx, "user_id", validateResp.UserId)
ctx = context.WithValue(ctx, "permissions", validateResp.Permissions)
```

### 2. Service Permission Check
```go
// 1. Get user ID from context
userID := getUserIDFromContext(ctx)

// 2. Check specific permission
permResp, err := authClient.CheckPermission(ctx, &auth.CheckPermissionRequest{
    UserId:         userID,
    Permission:     "write:project",
    ResourceId:     projectID,
    CallingService: "project-service",
    RequestId:      generateRequestID(),
})

if err != nil || !permResp.Allowed {
    return fmt.Errorf("access denied")
}

// 3. Proceed with operation
```

### 3. User Context Enrichment
```go
// 1. Get user ID from context
userID := getUserIDFromContext(ctx)

// 2. Get full user context
userResp, err := authClient.GetUserContext(ctx, &auth.GetUserContextRequest{
    UserId:         userID,
    CallingService: "simulation-service",
    RequestId:      generateRequestID(),
})

if err != nil {
    return err
}

// 3. Use user context for business logic
simulation := &Simulation{
    UserID:    userID,
    UserEmail: userResp.Email,
    Company:   userResp.Company,
}
```

---

## ⚠️ Error Codes

### gRPC Status Codes
- `InvalidArgument`: Missing or invalid request parameters
- `Unauthenticated`: Invalid or expired token
- `PermissionDenied`: Insufficient permissions
- `NotFound`: User or resource not found
- `DeadlineExceeded`: Request timeout
- `Unavailable`: Auth service unavailable

### Application Error Messages
- `"token is required"`: Missing token in request
- `"invalid token"`: Token format or signature invalid
- `"token expired"`: Token has expired
- `"user not found"`: User ID doesn't exist
- `"permission denied"`: User lacks required permission
- `"session invalid"`: Session not found or expired

---

## 🚀 Performance Tips

1. **Use Connection Pooling**: Reuse gRPC connections
2. **Set Timeouts**: Always set request timeouts
3. **Implement Retries**: Handle transient failures
4. **Cache Sparingly**: Don't cache token validations
5. **Batch Requests**: Combine multiple permission checks when possible
6. **Monitor Metrics**: Track request latency and error rates
