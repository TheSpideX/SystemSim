package services

import (
	"context"
	"fmt"
	"log"
	"time"

	proto "server-service/api/proto"
	"server-service/internal/grpc_clients"
)

// Mock response types (temporary until proto integration is fixed)
type ValidateTokenResponse struct {
	Valid     bool   `json:"valid"`
	UserId    string `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
}

type GetUserContextResponse struct {
	UserId      string   `json:"user_id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type CheckPermissionResponse struct {
	Allowed bool `json:"allowed"`
}

type ValidateSessionResponse struct {
	Valid  bool   `json:"valid"`
	UserId string `json:"user_id"`
}

type GetUserPermissionsResponse struct {
	Permissions []string `json:"permissions"`
	Roles       []string `json:"roles"`
}

type HealthCheckResponse struct {
	Status string `json:"status"`
}

// AuthService handles authentication-related operations
type AuthService struct {
	grpcPool *grpc_clients.ServicePool
}

// NewAuthService creates a new auth service client
func NewAuthService(grpcPool *grpc_clients.ServicePool) *AuthService {
	return &AuthService{
		grpcPool: grpcPool,
	}
}

// ValidateToken validates a JWT token and returns user context via gRPC
func (as *AuthService) ValidateToken(token, callingService, requestID string) (*ValidateTokenResponse, error) {
	// If gRPC pool is available, use real service
	if as.grpcPool != nil {
		// Get connection from pool
		conn := as.grpcPool.GetConnection()
		if conn == nil {
			return nil, fmt.Errorf("no gRPC connection available")
		}

		// Ensure connection is released after use
		defer as.grpcPool.ReleaseConnection()

		// Create gRPC client
		client := proto.NewAuthServiceClient(conn)

		// Create request
		req := &proto.ValidateTokenRequest{
			Token:          token,
			CallingService: callingService,
			RequestId:      requestID,
		}

		// Make gRPC call with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("Making REAL gRPC ValidateToken call to auth service")
		resp, err := client.ValidateToken(ctx, req)
		if err != nil {
			log.Printf("gRPC ValidateToken failed: %v", err)
			return nil, fmt.Errorf("gRPC call failed: %w", err)
		}

		// Convert gRPC response to internal response
		return &ValidateTokenResponse{
			Valid:     resp.Valid,
			UserId:    resp.UserId,
			ExpiresAt: resp.ExpiresAt,
		}, nil
	}

	// Fallback to mock implementation if no gRPC pool
	log.Printf("No gRPC pool available, using mock implementation")
	if token == "" {
		return &ValidateTokenResponse{
			Valid:     false,
			UserId:    "",
			ExpiresAt: 0,
		}, nil
	}

	return &ValidateTokenResponse{
		Valid:     true,
		UserId:    "mock-user-123",
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

// GetUserContext gets user context with permissions via gRPC
func (as *AuthService) GetUserContext(userID, callingService, requestID string) (*GetUserContextResponse, error) {
	// If gRPC pool is available, use real service
	if as.grpcPool != nil {
		// Get connection from pool
		conn := as.grpcPool.GetConnection()
		if conn == nil {
			return nil, fmt.Errorf("no gRPC connection available")
		}

		// Ensure connection is released after use
		defer as.grpcPool.ReleaseConnection()

		// Create gRPC client
		client := proto.NewAuthServiceClient(conn)

		// Create request
		req := &proto.GetUserContextRequest{
			UserId:         userID,
			CallingService: callingService,
			RequestId:      requestID,
		}

		// Make gRPC call with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("Making REAL gRPC GetUserContext call to auth service")
		resp, err := client.GetUserContext(ctx, req)
		if err != nil {
			log.Printf("gRPC GetUserContext failed: %v", err)
			return nil, fmt.Errorf("gRPC call failed: %w", err)
		}

		// Convert gRPC response to internal response
		return &GetUserContextResponse{
			UserId:      resp.UserId,
			Email:       resp.Email,
			Roles:       resp.Roles,
			Permissions: resp.Permissions,
		}, nil
	}

	// Fallback to mock implementation if no gRPC pool
	log.Printf("No gRPC pool available, using mock implementation")
	return &GetUserContextResponse{
		UserId:      userID,
		Email:       "mock.user@example.com",
		Roles:       []string{"user", "developer"},
		Permissions: []string{"read:projects", "write:projects", "read:simulations"},
	}, nil
}

// CheckPermission checks if user has specific permission via gRPC
func (as *AuthService) CheckPermission(userID, permission, resource, callingService, requestID string) (*CheckPermissionResponse, error) {
	// If gRPC pool is available, use real service
	if as.grpcPool != nil {
		// Get connection from pool
		conn := as.grpcPool.GetConnection()
		if conn == nil {
			return nil, fmt.Errorf("no gRPC connection available")
		}

		// Ensure connection is released after use
		defer as.grpcPool.ReleaseConnection()

		// Create gRPC client
		client := proto.NewAuthServiceClient(conn)

		// Create request
		req := &proto.CheckPermissionRequest{
			UserId:         userID,
			Permission:     permission,
			ResourceId:     resource,
			CallingService: callingService,
			RequestId:      requestID,
		}

		// Make gRPC call with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("Making REAL gRPC CheckPermission call to auth service")
		resp, err := client.CheckPermission(ctx, req)
		if err != nil {
			log.Printf("gRPC CheckPermission failed: %v", err)
			return nil, fmt.Errorf("gRPC call failed: %w", err)
		}

		// Convert gRPC response to internal response
		return &CheckPermissionResponse{
			Allowed: resp.Allowed,
		}, nil
	}

	// Fallback to mock implementation if no gRPC pool
	log.Printf("No gRPC pool available, using mock implementation")
	allowed := permission == "read:projects" || permission == "write:projects" || permission == "read:simulations"
	return &CheckPermissionResponse{
		Allowed: allowed,
	}, nil
}

// ValidateSession validates a user session via gRPC
func (as *AuthService) ValidateSession(sessionID, callingService, requestID string) (*ValidateSessionResponse, error) {
	// If gRPC pool is available, use real service
	if as.grpcPool != nil {
		// Get connection from pool
		conn := as.grpcPool.GetConnection()
		if conn == nil {
			return nil, fmt.Errorf("no gRPC connection available")
		}

		// Ensure connection is released after use
		defer as.grpcPool.ReleaseConnection()

		// Create gRPC client
		client := proto.NewAuthServiceClient(conn)

		// Create request
		req := &proto.ValidateSessionRequest{
			SessionId:      sessionID,
			CallingService: callingService,
			RequestId:      requestID,
		}

		// Make gRPC call with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("Making REAL gRPC ValidateSession call to auth service")
		resp, err := client.ValidateSession(ctx, req)
		if err != nil {
			log.Printf("gRPC ValidateSession failed: %v", err)
			return nil, fmt.Errorf("gRPC call failed: %w", err)
		}

		// Convert gRPC response to internal response
		return &ValidateSessionResponse{
			Valid:  resp.Valid,
			UserId: resp.UserId,
		}, nil
	}

	// Fallback to mock implementation if no gRPC pool
	log.Printf("No gRPC pool available, using mock implementation")
	return &ValidateSessionResponse{
		Valid:  sessionID != "",
		UserId: "mock-user-123",
	}, nil
}

// GetUserPermissions gets all user permissions via gRPC
func (as *AuthService) GetUserPermissions(userID, callingService, requestID string) (*GetUserPermissionsResponse, error) {
	// If gRPC pool is available, use real service
	if as.grpcPool != nil {
		// Get connection from pool
		conn := as.grpcPool.GetConnection()
		if conn == nil {
			return nil, fmt.Errorf("no gRPC connection available")
		}

		// Ensure connection is released after use
		defer as.grpcPool.ReleaseConnection()

		// Create gRPC client
		client := proto.NewAuthServiceClient(conn)

		// Create request
		req := &proto.GetUserPermissionsRequest{
			UserId:         userID,
			CallingService: callingService,
			RequestId:      requestID,
			IncludeRoles:   true,
		}

		// Make gRPC call with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("Making REAL gRPC GetUserPermissions call to auth service")
		resp, err := client.GetUserPermissions(ctx, req)
		if err != nil {
			log.Printf("gRPC GetUserPermissions failed: %v", err)
			return nil, fmt.Errorf("gRPC call failed: %w", err)
		}

		// Extract role names from UserRole objects
		roleNames := make([]string, len(resp.Roles))
		for i, role := range resp.Roles {
			roleNames[i] = role.RoleName
		}

		// Convert gRPC response to internal response
		return &GetUserPermissionsResponse{
			Permissions: resp.Permissions,
			Roles:       roleNames,
		}, nil
	}

	// Fallback to mock implementation if no gRPC pool
	log.Printf("No gRPC pool available, using mock implementation")
	return &GetUserPermissionsResponse{
		Permissions: []string{"read:projects", "write:projects", "read:simulations", "write:simulations"},
		Roles:       []string{"user", "developer"},
	}, nil
}

// HealthCheck checks auth service health via gRPC
func (as *AuthService) HealthCheck(callingService, requestID string) (*HealthCheckResponse, error) {
	// If gRPC pool is available, use real service
	if as.grpcPool != nil {
		// Get connection from pool
		conn := as.grpcPool.GetConnection()
		if conn == nil {
			return &HealthCheckResponse{Status: "unavailable"}, fmt.Errorf("no gRPC connection available")
		}

		// Ensure connection is released after use
		defer as.grpcPool.ReleaseConnection()

		// Create gRPC client
		client := proto.NewAuthServiceClient(conn)

		// Create request
		req := &proto.HealthCheckRequest{
			CallingService: callingService,
			RequestId:      requestID,
		}

		// Make gRPC call with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		log.Printf("Making REAL gRPC HealthCheck call to auth service")
		resp, err := client.HealthCheck(ctx, req)
		if err != nil {
			log.Printf("gRPC HealthCheck failed: %v", err)
			return &HealthCheckResponse{Status: "unhealthy"}, fmt.Errorf("gRPC call failed: %w", err)
		}

		// Convert gRPC response to internal response
		return &HealthCheckResponse{Status: resp.Status}, nil
	}

	// Fallback to mock implementation if no gRPC pool
	log.Printf("No gRPC pool available, using mock implementation")
	return &HealthCheckResponse{Status: "healthy"}, nil
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

// GenerateRequestID generates a unique request ID for tracing
func GenerateRequestID() string {
	return fmt.Sprintf("api-gateway-%d", time.Now().UnixNano())
}

// LogAuthOperation logs authentication operations for monitoring
func (as *AuthService) LogAuthOperation(operation, userID, result string, duration time.Duration) {
	log.Printf("Auth Operation: %s | User: %s | Result: %s | Duration: %dms",
		operation, userID, result, duration.Milliseconds())
}
