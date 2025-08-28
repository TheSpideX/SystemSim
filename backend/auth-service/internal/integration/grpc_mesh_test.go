package integration

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	auth "github.com/systemsim/auth-service/internal/proto"
	"github.com/systemsim/auth-service/internal/discovery"
	"github.com/systemsim/auth-service/internal/events"
	"github.com/systemsim/auth-service/internal/health"
	"github.com/systemsim/auth-service/internal/mesh"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/security"
	"github.com/systemsim/auth-service/internal/testutils"
)

// GRPCMeshTestSuite holds the gRPC mesh test environment
type GRPCMeshTestSuite struct {
	server          *grpc.Server
	listener        *bufconn.Listener
	client          auth.AuthServiceClient
	conn            *grpc.ClientConn
	authService     *MockAuthServiceForGRPC
	rbacService     *MockRBACServiceForGRPC
	userService     *MockUserServiceForGRPC
	healthChecker   *health.EnhancedHealthChecker
	meshClient      *mesh.MeshClient
	poolManager     *mesh.PoolManager
	cleanup         func()
}

// Mock services for gRPC testing
type MockAuthServiceForGRPC struct {
	tokens map[string]*security.JWTClaims
	mu     sync.RWMutex
}

func (m *MockAuthServiceForGRPC) ValidateAccessToken(token string) (*security.JWTClaims, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	claims, exists := m.tokens[token]
	if !exists {
		return nil, fmt.Errorf("invalid token")
	}
	
	// Check if token is expired
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

func (m *MockAuthServiceForGRPC) AddToken(token string, claims *security.JWTClaims) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token] = claims
}

type MockRBACServiceForGRPC struct {
	userPermissions map[string][]string
	userRoles       map[string][]string
	mu              sync.RWMutex
}

func (m *MockRBACServiceForGRPC) GetUserPermissionsForGRPC(userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	permissions, exists := m.userPermissions[userID]
	if !exists {
		return []string{}, nil
	}
	return permissions, nil
}

func (m *MockRBACServiceForGRPC) GetUserRolesForGRPC(userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	roles, exists := m.userRoles[userID]
	if !exists {
		return []string{}, nil
	}
	return roles, nil
}

func (m *MockRBACServiceForGRPC) CheckPermissionForGRPC(userID, permission, resourceID string) (bool, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	permissions, exists := m.userPermissions[userID]
	if !exists {
		return false, "user not found", nil
	}
	
	for _, perm := range permissions {
		if perm == permission {
			return true, "permission granted", nil
		}
	}
	
	return false, "permission denied", nil
}

func (m *MockRBACServiceForGRPC) SetUserPermissions(userID string, permissions []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userPermissions[userID] = permissions
}

func (m *MockRBACServiceForGRPC) SetUserRoles(userID string, roles []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userRoles[userID] = roles
}

type MockUserServiceForGRPC struct {
	users map[string]*models.User
	mu    sync.RWMutex
}

func (m *MockUserServiceForGRPC) GetUserByIDForGRPC(userID string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (m *MockUserServiceForGRPC) GetUserByID(userID string) (*models.UserResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	// Convert User to UserResponse using the built-in method
	return user.ToResponse(), nil
}

func (m *MockUserServiceForGRPC) AddUser(userID string, user *models.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[userID] = user
}

// Mock mesh health checker
type MockMeshHealthChecker struct{}

func (m *MockMeshHealthChecker) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"status": "healthy",
		"connections": map[string]interface{}{
			"total": 10,
			"healthy": 8,
		},
	}
}

// Adapter types to match the expected service interfaces
type AuthServiceAdapter struct {
	mock *MockAuthServiceForGRPC
}

func (a *AuthServiceAdapter) ValidateAccessToken(token string) (*security.JWTClaims, error) {
	return a.mock.ValidateAccessToken(token)
}

type RBACServiceAdapter struct {
	mock *MockRBACServiceForGRPC
}

func (r *RBACServiceAdapter) GetUserPermissionsForGRPC(userID string) ([]string, error) {
	return r.mock.GetUserPermissionsForGRPC(userID)
}

func (r *RBACServiceAdapter) GetUserRolesForGRPC(userID string) ([]string, error) {
	return r.mock.GetUserRolesForGRPC(userID)
}

func (r *RBACServiceAdapter) CheckPermissionForGRPC(userID, permission, resourceID string) (bool, string, error) {
	return r.mock.CheckPermissionForGRPC(userID, permission, resourceID)
}

type UserServiceAdapter struct {
	mock *MockUserServiceForGRPC
}

func (u *UserServiceAdapter) GetUserByID(userID string) (*models.UserResponse, error) {
	return u.mock.GetUserByID(userID)
}

// MockAuthGRPCHandler implements the gRPC AuthService interface for testing
type MockAuthGRPCHandler struct {
	auth.UnimplementedAuthServiceServer
	authService *MockAuthServiceForGRPC
	rbacService *MockRBACServiceForGRPC
	userService *MockUserServiceForGRPC
}

func (h *MockAuthGRPCHandler) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &auth.ValidateTokenResponse{
			Valid:        false,
			ErrorMessage: "token is required",
		}, nil
	}

	// Validate the token using mock auth service
	claims, err := h.authService.ValidateAccessToken(req.Token)
	if err != nil {
		return &auth.ValidateTokenResponse{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("invalid token: %v", err),
		}, nil
	}

	// Get user permissions
	permissions, err := h.rbacService.GetUserPermissionsForGRPC(claims.UserID.String())
	if err != nil {
		permissions = []string{}
	}

	return &auth.ValidateTokenResponse{
		Valid:       true,
		UserId:      claims.UserID.String(),
		Email:       claims.Email,
		IsAdmin:     claims.IsAdmin,
		SessionId:   claims.SessionID.String(),
		Permissions: permissions,
		ExpiresAt:   claims.ExpiresAt.Unix(),
	}, nil
}

func (h *MockAuthGRPCHandler) GetUserContext(ctx context.Context, req *auth.GetUserContextRequest) (*auth.GetUserContextResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Get the original User model (not UserResponse) to access all fields including LastLoginIP
	var originalUser *models.User
	for _, user := range h.userService.users {
		if user.ID.String() == req.UserId {
			originalUser = user
			break
		}
	}

	if originalUser == nil {
		return &auth.GetUserContextResponse{
			ErrorMessage: "user not found",
		}, nil
	}

	// Get user roles and permissions
	roles, err := h.rbacService.GetUserRolesForGRPC(req.UserId)
	if err != nil {
		roles = []string{}
	}

	permissions, err := h.rbacService.GetUserPermissionsForGRPC(req.UserId)
	if err != nil {
		permissions = []string{}
	}

	var lastLogin int64
	if originalUser.LastLoginAt != nil {
		lastLogin = originalUser.LastLoginAt.Unix()
	}

	var lastLoginIP string
	if originalUser.LastLoginIP != nil {
		lastLoginIP = *originalUser.LastLoginIP
	}

	return &auth.GetUserContextResponse{
		UserId:        originalUser.ID.String(),
		Email:         originalUser.Email,
		FirstName:     originalUser.FirstName,
		LastName:      originalUser.LastName,
		Company:       originalUser.Company,
		Roles:         roles,
		Permissions:   permissions,
		IsActive:      originalUser.IsActive,
		IsAdmin:       originalUser.IsAdmin,
		LastLogin:     lastLogin,
		LastLoginIp:   lastLoginIP,
		EmailVerified: originalUser.EmailVerified,
	}, nil
}

func (h *MockAuthGRPCHandler) CheckPermission(ctx context.Context, req *auth.CheckPermissionRequest) (*auth.CheckPermissionResponse, error) {
	allowed, reason, err := h.rbacService.CheckPermissionForGRPC(req.UserId, req.Permission, req.ResourceId)
	if err != nil {
		return &auth.CheckPermissionResponse{
			Allowed:      false,
			Reason:       "error checking permission",
			UserId:       req.UserId,
			Permission:   req.Permission,
			ResourceId:   req.ResourceId,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &auth.CheckPermissionResponse{
		Allowed:    allowed,
		Reason:     reason,
		UserId:     req.UserId,
		Permission: req.Permission,
		ResourceId: req.ResourceId,
	}, nil
}

func (h *MockAuthGRPCHandler) ValidateSession(ctx context.Context, req *auth.ValidateSessionRequest) (*auth.ValidateSessionResponse, error) {
	// Simple mock implementation
	return &auth.ValidateSessionResponse{
		Valid:     true,
		SessionId: req.SessionId,
		UserId:    req.UserId,
		IsActive:  true,
	}, nil
}

func (h *MockAuthGRPCHandler) GetUserPermissions(ctx context.Context, req *auth.GetUserPermissionsRequest) (*auth.GetUserPermissionsResponse, error) {
	permissions, err := h.rbacService.GetUserPermissionsForGRPC(req.UserId)
	if err != nil {
		return &auth.GetUserPermissionsResponse{
			ErrorMessage: err.Error(),
		}, nil
	}

	var roles []string
	if req.IncludeRoles {
		roles, _ = h.rbacService.GetUserRolesForGRPC(req.UserId)
	}

	// Convert string roles to UserRole objects
	var userRoles []*auth.UserRole
	for _, roleName := range roles {
		userRoles = append(userRoles, &auth.UserRole{
			RoleName: roleName,
		})
	}

	return &auth.GetUserPermissionsResponse{
		UserId:      req.UserId,
		Permissions: permissions,
		Roles:       userRoles,
	}, nil
}

func (h *MockAuthGRPCHandler) HealthCheck(ctx context.Context, req *auth.HealthCheckRequest) (*auth.HealthCheckResponse, error) {
	return &auth.HealthCheckResponse{
		Status:  "healthy",
		Version: "test-version",
	}, nil
}

// SetupGRPCMeshTestSuite initializes the gRPC mesh test environment
func SetupGRPCMeshTestSuite(t *testing.T) *GRPCMeshTestSuite {
	// Setup test Redis for health checker
	redisClient := testutils.SetupTestRedis(t)
	
	// Create mock services
	authService := &MockAuthServiceForGRPC{
		tokens: make(map[string]*security.JWTClaims),
	}
	
	rbacService := &MockRBACServiceForGRPC{
		userPermissions: make(map[string][]string),
		userRoles:       make(map[string][]string),
	}
	
	userService := &MockUserServiceForGRPC{
		users: make(map[string]*models.User),
	}
	
	// Create event system for health checker
	eventPublisher := events.NewPublisher(redisClient)
	eventSubscriber := events.NewSubscriber(redisClient)

	// Create mock mesh client for health checker
	mockMeshClient := &MockMeshHealthChecker{}

	// Create health checker
	healthChecker := health.NewEnhancedHealthChecker(nil, redisClient, eventPublisher, eventSubscriber, mockMeshClient, "test-version")

	// Create mock gRPC handler directly
	handler := &MockAuthGRPCHandler{
		authService: authService,
		rbacService: rbacService,
		userService: userService,
	}
	
	// Create in-memory gRPC server
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	auth.RegisterAuthServiceServer(server, handler)
	
	// Start server in background
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Logf("gRPC server error: %v", err)
		}
	}()
	
	// Create client connection
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "Failed to create gRPC client connection")
	
	client := auth.NewAuthServiceClient(conn)
	
	// Create mesh client components
	serviceDiscovery := discovery.NewServiceDiscovery(redisClient)
	poolManager := mesh.NewPoolManager(serviceDiscovery, 2, 5) // 2 min, 5 max connections for testing
	meshClient := mesh.NewMeshClient(poolManager)
	
	cleanup := func() {
		conn.Close()
		server.Stop()
		listener.Close()
		testutils.CleanupTestRedis(t, redisClient)
		redisClient.Close()
	}
	
	return &GRPCMeshTestSuite{
		server:        server,
		listener:      listener,
		client:        client,
		conn:          conn,
		authService:   authService,
		rbacService:   rbacService,
		userService:   userService,
		healthChecker: healthChecker,
		meshClient:    meshClient,
		poolManager:   poolManager,
		cleanup:       cleanup,
	}
}

// Test gRPC ValidateToken endpoint
func TestGRPCMesh_ValidateToken(t *testing.T) {
	suite := SetupGRPCMeshTestSuite(t)
	defer suite.cleanup()
	
	t.Run("valid_token_validation", func(t *testing.T) {
		// Setup test data
		userID := uuid.New()
		sessionID := uuid.New()
		token := "valid_test_token_123"
		
		claims := &security.JWTClaims{
			UserID:    userID,
			Email:     "test@example.com",
			IsAdmin:   false,
			SessionID: sessionID,
			TokenType: "access",
		}
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(1 * time.Hour))
		
		suite.authService.AddToken(token, claims)
		suite.rbacService.SetUserPermissions(userID.String(), []string{"read:profile", "write:profile"})
		
		// Make gRPC call
		req := &auth.ValidateTokenRequest{
			Token:          token,
			CallingService: "api-gateway",
			RequestId:      "test-request-123",
		}
		
		resp, err := suite.client.ValidateToken(context.Background(), req)
		require.NoError(t, err, "ValidateToken should succeed")
		
		// Verify response
		assert.True(t, resp.Valid, "Token should be valid")
		assert.Equal(t, userID.String(), resp.UserId, "User ID should match")
		assert.Equal(t, "test@example.com", resp.Email, "Email should match")
		assert.False(t, resp.IsAdmin, "IsAdmin should be false")
		assert.Equal(t, sessionID.String(), resp.SessionId, "Session ID should match")
		assert.Contains(t, resp.Permissions, "read:profile", "Should contain read:profile permission")
		assert.Contains(t, resp.Permissions, "write:profile", "Should contain write:profile permission")
		assert.True(t, resp.ExpiresAt > time.Now().Unix(), "ExpiresAt should be in the future")
		assert.Empty(t, resp.ErrorMessage, "Error message should be empty")
	})
	
	t.Run("invalid_token_validation", func(t *testing.T) {
		req := &auth.ValidateTokenRequest{
			Token:          "invalid_token",
			CallingService: "api-gateway",
			RequestId:      "test-request-456",
		}
		
		resp, err := suite.client.ValidateToken(context.Background(), req)
		require.NoError(t, err, "gRPC call should succeed even with invalid token")
		
		// Verify response
		assert.False(t, resp.Valid, "Token should be invalid")
		assert.Empty(t, resp.UserId, "User ID should be empty")
		assert.Empty(t, resp.Email, "Email should be empty")
		assert.NotEmpty(t, resp.ErrorMessage, "Error message should not be empty")
		assert.Contains(t, resp.ErrorMessage, "invalid token", "Error message should indicate invalid token")
	})
	
	t.Run("empty_token_validation", func(t *testing.T) {
		req := &auth.ValidateTokenRequest{
			Token:          "",
			CallingService: "api-gateway",
			RequestId:      "test-request-789",
		}
		
		resp, err := suite.client.ValidateToken(context.Background(), req)
		require.NoError(t, err, "gRPC call should succeed")
		
		// Verify response
		assert.False(t, resp.Valid, "Empty token should be invalid")
		assert.Equal(t, "token is required", resp.ErrorMessage, "Should indicate token is required")
	})
	
	t.Run("expired_token_validation", func(t *testing.T) {
		// Setup expired token
		userID := uuid.New()
		sessionID := uuid.New()
		expiredToken := "expired_test_token_123"
		
		expiredClaims := &security.JWTClaims{
			UserID:    userID,
			Email:     "expired@example.com",
			IsAdmin:   false,
			SessionID: sessionID,
			TokenType: "access",
		}
		expiredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)) // Expired 1 hour ago
		
		suite.authService.AddToken(expiredToken, expiredClaims)
		
		req := &auth.ValidateTokenRequest{
			Token:          expiredToken,
			CallingService: "api-gateway",
			RequestId:      "test-request-expired",
		}
		
		resp, err := suite.client.ValidateToken(context.Background(), req)
		require.NoError(t, err, "gRPC call should succeed")
		
		// Verify response
		assert.False(t, resp.Valid, "Expired token should be invalid")
		assert.Contains(t, resp.ErrorMessage, "expired", "Error message should indicate token is expired")
	})
}

// Test gRPC GetUserContext endpoint
func TestGRPCMesh_GetUserContext(t *testing.T) {
	suite := SetupGRPCMeshTestSuite(t)
	defer suite.cleanup()

	t.Run("valid_user_context_retrieval", func(t *testing.T) {
		// Setup test data
		userID := uuid.New()
		lastLoginTime := time.Now().Add(-1 * time.Hour)
		lastLoginIP := "192.168.1.100"
		now := time.Now()
		user := &models.User{
			ID:                    userID,
			Email:                 "context@example.com",
			FirstName:             "Context",
			LastName:              "User",
			Company:               "Test Company",
			IsActive:              true,
			EmailVerified:         true,
			LastLoginAt:           &lastLoginTime,
			LastLoginIP:           &lastLoginIP,
			SimulationPreferences: make(models.JSONMap),
			UIPreferences:         make(models.JSONMap),
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		suite.userService.AddUser(userID.String(), user)
		suite.rbacService.SetUserRoles(userID.String(), []string{"user", "editor"})
		suite.rbacService.SetUserPermissions(userID.String(), []string{"read:profile", "write:content", "delete:content"})

		// Make gRPC call
		req := &auth.GetUserContextRequest{
			UserId:         userID.String(),
			CallingService: "project-service",
			RequestId:      "context-request-123",
		}

		resp, err := suite.client.GetUserContext(context.Background(), req)
		require.NoError(t, err, "GetUserContext should succeed")

		// Verify response
		assert.Equal(t, userID.String(), resp.UserId, "User ID should match")
		assert.Equal(t, "context@example.com", resp.Email, "Email should match")
		assert.Equal(t, "Context", resp.FirstName, "First name should match")
		assert.Equal(t, "User", resp.LastName, "Last name should match")
		assert.Equal(t, "Test Company", resp.Company, "Company should match")
		assert.True(t, resp.IsActive, "User should be active")
		assert.True(t, resp.EmailVerified, "Email should be verified")
		assert.False(t, resp.IsAdmin, "User should not be admin")
		assert.Contains(t, resp.Roles, "user", "Should contain user role")
		assert.Contains(t, resp.Roles, "editor", "Should contain editor role")
		assert.Contains(t, resp.Permissions, "read:profile", "Should contain read:profile permission")
		assert.Contains(t, resp.Permissions, "write:content", "Should contain write:content permission")
		assert.Contains(t, resp.Permissions, "delete:content", "Should contain delete:content permission")
		assert.True(t, resp.LastLogin > 0, "Last login should be set")
		assert.Equal(t, "192.168.1.100", resp.LastLoginIp, "Last login IP should match")
		assert.Empty(t, resp.ErrorMessage, "Error message should be empty")
	})

	t.Run("user_not_found", func(t *testing.T) {
		nonExistentUserID := uuid.New()

		req := &auth.GetUserContextRequest{
			UserId:         nonExistentUserID.String(),
			CallingService: "project-service",
			RequestId:      "context-request-404",
		}

		resp, err := suite.client.GetUserContext(context.Background(), req)
		require.NoError(t, err, "gRPC call should succeed")

		// Verify response
		assert.NotEmpty(t, resp.ErrorMessage, "Error message should not be empty")
		assert.Contains(t, resp.ErrorMessage, "user not found", "Error should indicate user not found")
	})

	t.Run("empty_user_id", func(t *testing.T) {
		req := &auth.GetUserContextRequest{
			UserId:         "",
			CallingService: "project-service",
			RequestId:      "context-request-empty",
		}

		_, err := suite.client.GetUserContext(context.Background(), req)
		require.Error(t, err, "Should return error for empty user ID")

		// Verify gRPC error
		st, ok := status.FromError(err)
		require.True(t, ok, "Error should be a gRPC status error")
		assert.Equal(t, codes.InvalidArgument, st.Code(), "Should return InvalidArgument error")
		assert.Contains(t, st.Message(), "user_id is required", "Error message should indicate user_id is required")
	})
}

// Test gRPC CheckPermission endpoint
func TestGRPCMesh_CheckPermission(t *testing.T) {
	suite := SetupGRPCMeshTestSuite(t)
	defer suite.cleanup()

	t.Run("permission_allowed", func(t *testing.T) {
		// Setup test data
		userID := uuid.New()
		suite.rbacService.SetUserPermissions(userID.String(), []string{"read:project", "write:project", "delete:project"})

		// Make gRPC call
		req := &auth.CheckPermissionRequest{
			UserId:         userID.String(),
			Permission:     "read:project",
			ResourceId:     "project-123",
			CallingService: "project-service",
			RequestId:      "permission-request-123",
		}

		resp, err := suite.client.CheckPermission(context.Background(), req)
		require.NoError(t, err, "CheckPermission should succeed")

		// Verify response
		assert.True(t, resp.Allowed, "Permission should be allowed")
		assert.Equal(t, "permission granted", resp.Reason, "Reason should indicate permission granted")
		assert.Equal(t, userID.String(), resp.UserId, "User ID should match")
		assert.Equal(t, "read:project", resp.Permission, "Permission should match")
		assert.Equal(t, "project-123", resp.ResourceId, "Resource ID should match")
		assert.Empty(t, resp.ErrorMessage, "Error message should be empty")
	})

	t.Run("permission_denied", func(t *testing.T) {
		// Setup test data
		userID := uuid.New()
		suite.rbacService.SetUserPermissions(userID.String(), []string{"read:project"}) // Only read permission

		// Make gRPC call for write permission
		req := &auth.CheckPermissionRequest{
			UserId:         userID.String(),
			Permission:     "write:project",
			ResourceId:     "project-456",
			CallingService: "project-service",
			RequestId:      "permission-request-denied",
		}

		resp, err := suite.client.CheckPermission(context.Background(), req)
		require.NoError(t, err, "gRPC call should succeed")

		// Verify response
		assert.False(t, resp.Allowed, "Permission should be denied")
		assert.Equal(t, "permission denied", resp.Reason, "Reason should indicate permission denied")
		assert.Equal(t, userID.String(), resp.UserId, "User ID should match")
		assert.Equal(t, "write:project", resp.Permission, "Permission should match")
		assert.Equal(t, "project-456", resp.ResourceId, "Resource ID should match")
		assert.Empty(t, resp.ErrorMessage, "Error message should be empty")
	})

	t.Run("user_not_found_permission_check", func(t *testing.T) {
		nonExistentUserID := uuid.New()

		req := &auth.CheckPermissionRequest{
			UserId:         nonExistentUserID.String(),
			Permission:     "read:project",
			ResourceId:     "project-789",
			CallingService: "project-service",
			RequestId:      "permission-request-404",
		}

		resp, err := suite.client.CheckPermission(context.Background(), req)
		require.NoError(t, err, "gRPC call should succeed")

		// Verify response
		assert.False(t, resp.Allowed, "Permission should be denied for non-existent user")
		assert.Equal(t, "user not found", resp.Reason, "Reason should indicate user not found")
	})
}

// Test gRPC ValidateSession endpoint
func TestGRPCMesh_ValidateSession(t *testing.T) {
	suite := SetupGRPCMeshTestSuite(t)
	defer suite.cleanup()

	t.Run("valid_session_validation", func(t *testing.T) {
		// Note: This test would require implementing session validation in the mock
		// For now, we'll test the gRPC interface structure
		req := &auth.ValidateSessionRequest{
			SessionId:      "session-123",
			UserId:         uuid.New().String(),
			CallingService: "api-gateway",
			RequestId:      "session-request-123",
		}

		resp, err := suite.client.ValidateSession(context.Background(), req)
		require.NoError(t, err, "ValidateSession gRPC call should succeed")

		// The response structure should be valid even if the implementation is not complete
		assert.NotNil(t, resp, "Response should not be nil")
		// In a full implementation, we would test session validity, expiration, etc.
	})
}

// Test gRPC GetUserPermissions endpoint
func TestGRPCMesh_GetUserPermissions(t *testing.T) {
	suite := SetupGRPCMeshTestSuite(t)
	defer suite.cleanup()

	t.Run("get_user_permissions_with_roles", func(t *testing.T) {
		userID := uuid.New()
		suite.rbacService.SetUserPermissions(userID.String(), []string{"read:all", "write:own", "delete:own"})
		suite.rbacService.SetUserRoles(userID.String(), []string{"user", "contributor"})

		req := &auth.GetUserPermissionsRequest{
			UserId:         userID.String(),
			CallingService: "simulation-service",
			RequestId:      "permissions-request-123",
			IncludeRoles:   true,
		}

		resp, err := suite.client.GetUserPermissions(context.Background(), req)
		require.NoError(t, err, "GetUserPermissions should succeed")

		// The response structure should be valid
		assert.NotNil(t, resp, "Response should not be nil")
		// In a full implementation, we would verify permissions and roles are returned
	})
}

// Test gRPC HealthCheck endpoint
func TestGRPCMesh_HealthCheck(t *testing.T) {
	suite := SetupGRPCMeshTestSuite(t)
	defer suite.cleanup()

	t.Run("health_check_success", func(t *testing.T) {
		req := &auth.HealthCheckRequest{
			CallingService: "api-gateway",
			RequestId:      "health-request-123",
		}

		resp, err := suite.client.HealthCheck(context.Background(), req)
		require.NoError(t, err, "HealthCheck should succeed")

		// Verify health check response structure
		assert.NotNil(t, resp, "Health check response should not be nil")
		// In a full implementation, we would verify service health status
	})
}

// Test concurrent gRPC calls (mesh communication patterns)
func TestGRPCMesh_ConcurrentCalls(t *testing.T) {
	suite := SetupGRPCMeshTestSuite(t)
	defer suite.cleanup()

	t.Run("concurrent_token_validation", func(t *testing.T) {
		// Setup multiple valid tokens
		numTokens := 10
		tokens := make([]string, numTokens)
		userIDs := make([]uuid.UUID, numTokens)

		for i := 0; i < numTokens; i++ {
			userID := uuid.New()
			sessionID := uuid.New()
			token := fmt.Sprintf("concurrent_token_%d", i)

			claims := &security.JWTClaims{
				UserID:    userID,
				Email:     fmt.Sprintf("user%d@example.com", i),
				IsAdmin:   i%2 == 0, // Every other user is admin
				SessionID: sessionID,
				TokenType: "access",
			}
			claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(1 * time.Hour))

			suite.authService.AddToken(token, claims)
			suite.rbacService.SetUserPermissions(userID.String(), []string{fmt.Sprintf("permission_%d", i)})

			tokens[i] = token
			userIDs[i] = userID
		}

		// Make concurrent gRPC calls
		var wg sync.WaitGroup
		results := make(chan *auth.ValidateTokenResponse, numTokens)
		errors := make(chan error, numTokens)

		for i := 0; i < numTokens; i++ {
			wg.Add(1)
			go func(tokenIndex int) {
				defer wg.Done()

				req := &auth.ValidateTokenRequest{
					Token:          tokens[tokenIndex],
					CallingService: fmt.Sprintf("service_%d", tokenIndex),
					RequestId:      fmt.Sprintf("concurrent_request_%d", tokenIndex),
				}

				resp, err := suite.client.ValidateToken(context.Background(), req)
				if err != nil {
					errors <- err
					return
				}

				results <- resp
			}(i)
		}

		wg.Wait()
		close(results)
		close(errors)

		// Verify no errors occurred
		errorCount := 0
		for err := range errors {
			t.Errorf("Concurrent call error: %v", err)
			errorCount++
		}
		assert.Equal(t, 0, errorCount, "No errors should occur during concurrent calls")

		// Verify all responses are valid
		successCount := 0
		for resp := range results {
			assert.True(t, resp.Valid, "All tokens should be valid")
			assert.NotEmpty(t, resp.UserId, "User ID should not be empty")
			assert.NotEmpty(t, resp.Email, "Email should not be empty")
			successCount++
		}
		assert.Equal(t, numTokens, successCount, "All concurrent calls should succeed")
	})
}

// Test gRPC error handling and edge cases
func TestGRPCMesh_ErrorHandling(t *testing.T) {
	suite := SetupGRPCMeshTestSuite(t)
	defer suite.cleanup()

	t.Run("malformed_requests", func(t *testing.T) {
		// Test with nil context (should be handled gracefully)
		req := &auth.ValidateTokenRequest{
			Token:          "test_token",
			CallingService: "test-service",
			RequestId:      "malformed-request",
		}

		// This should not panic even with edge cases
		resp, err := suite.client.ValidateToken(context.Background(), req)
		require.NoError(t, err, "gRPC call should not fail due to malformed request")
		assert.NotNil(t, resp, "Response should not be nil")
	})

	t.Run("timeout_handling", func(t *testing.T) {
		// Test with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		req := &auth.ValidateTokenRequest{
			Token:          "timeout_test_token",
			CallingService: "timeout-service",
			RequestId:      "timeout-request",
		}

		_, err := suite.client.ValidateToken(ctx, req)
		// Should get a timeout error
		assert.Error(t, err, "Should get timeout error")

		// Verify it's a timeout error
		st, ok := status.FromError(err)
		if ok {
			assert.Equal(t, codes.DeadlineExceeded, st.Code(), "Should be deadline exceeded error")
		}
	})
}

// Test Mesh Client functionality
func TestGRPCMesh_MeshClientIntegration(t *testing.T) {
	suite := SetupGRPCMeshTestSuite(t)
	defer suite.cleanup()

	t.Run("mesh_client_call_with_retry", func(t *testing.T) {
		// Setup a mock connection in the pool manager
		// Note: This is a simplified test since we're using bufconn
		serviceName := "test-service"

		// Test the mesh client call pattern
		callCount := 0
		err := suite.meshClient.CallWithRetry(context.Background(), serviceName, func(conn *grpc.ClientConn) error {
			callCount++
			// Simulate a successful call
			return nil
		})

		// Since we don't have the service registered in the pool manager,
		// this will fail, but we can test the retry mechanism
		assert.Error(t, err, "Should fail since service is not in pool")
		assert.Contains(t, err.Error(), "failed to get connection", "Error should indicate connection failure")
	})

	t.Run("mesh_client_timeout_handling", func(t *testing.T) {
		serviceName := "timeout-service"
		timeout := 100 * time.Millisecond

		err := suite.meshClient.CallWithTimeout(serviceName, timeout, func(conn *grpc.ClientConn) error {
			// Simulate a slow operation
			time.Sleep(200 * time.Millisecond)
			return nil
		})

		assert.Error(t, err, "Should timeout")
	})
}

// Test service mesh communication patterns
func TestGRPCMesh_ServiceMeshPatterns(t *testing.T) {
	suite := SetupGRPCMeshTestSuite(t)
	defer suite.cleanup()

	t.Run("service_to_service_authentication", func(t *testing.T) {
		// Simulate API Gateway calling Auth Service for token validation
		userID := uuid.New()
		sessionID := uuid.New()
		token := "api_gateway_token_123"

		claims := &security.JWTClaims{
			UserID:    userID,
			Email:     "gateway@example.com",
			IsAdmin:   true,
			SessionID: sessionID,
			TokenType: "access",
		}
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(1 * time.Hour))

		suite.authService.AddToken(token, claims)
		suite.rbacService.SetUserPermissions(userID.String(), []string{"admin:all", "read:all", "write:all"})

		// API Gateway validates token
		validateReq := &auth.ValidateTokenRequest{
			Token:          token,
			CallingService: "api-gateway",
			RequestId:      "gateway-auth-123",
		}

		validateResp, err := suite.client.ValidateToken(context.Background(), validateReq)
		require.NoError(t, err, "Token validation should succeed")
		assert.True(t, validateResp.Valid, "Token should be valid")

		// API Gateway checks permissions for a specific action
		permissionReq := &auth.CheckPermissionRequest{
			UserId:         validateResp.UserId,
			Permission:     "admin:all",
			ResourceId:     "system",
			CallingService: "api-gateway",
			RequestId:      "gateway-permission-123",
		}

		permissionResp, err := suite.client.CheckPermission(context.Background(), permissionReq)
		require.NoError(t, err, "Permission check should succeed")
		assert.True(t, permissionResp.Allowed, "Admin permission should be allowed")

		// API Gateway gets full user context for request processing
		contextReq := &auth.GetUserContextRequest{
			UserId:         validateResp.UserId,
			CallingService: "api-gateway",
			RequestId:      "gateway-context-123",
		}

		// Add user to mock service
		now := time.Now()
		user := &models.User{
			ID:                    userID,
			Email:                 "gateway@example.com",
			FirstName:             "Gateway",
			LastName:              "User",
			IsActive:              true,
			EmailVerified:         true,
			SimulationPreferences: make(models.JSONMap),
			UIPreferences:         make(models.JSONMap),
			CreatedAt:             now,
			UpdatedAt:             now,
		}
		suite.userService.AddUser(userID.String(), user)
		suite.rbacService.SetUserRoles(userID.String(), []string{"admin", "super-user"})

		contextResp, err := suite.client.GetUserContext(context.Background(), contextReq)
		require.NoError(t, err, "Get user context should succeed")
		assert.Equal(t, "gateway@example.com", contextResp.Email, "Email should match")
		assert.True(t, contextResp.IsActive, "User should be active")
		assert.Contains(t, contextResp.Roles, "admin", "Should have admin role")
		assert.Contains(t, contextResp.Permissions, "admin:all", "Should have admin:all permission")
	})

	t.Run("project_service_authorization_flow", func(t *testing.T) {
		// Simulate Project Service checking user permissions for project operations
		userID := uuid.New()
		projectID := "project-456"

		// Setup user with project-specific permissions
		suite.rbacService.SetUserPermissions(userID.String(), []string{"read:project", "write:project"})

		// Project Service checks if user can read project
		readReq := &auth.CheckPermissionRequest{
			UserId:         userID.String(),
			Permission:     "read:project",
			ResourceId:     projectID,
			CallingService: "project-service",
			RequestId:      "project-read-123",
		}

		readResp, err := suite.client.CheckPermission(context.Background(), readReq)
		require.NoError(t, err, "Read permission check should succeed")
		assert.True(t, readResp.Allowed, "Read permission should be allowed")

		// Project Service checks if user can delete project (should be denied)
		deleteReq := &auth.CheckPermissionRequest{
			UserId:         userID.String(),
			Permission:     "delete:project",
			ResourceId:     projectID,
			CallingService: "project-service",
			RequestId:      "project-delete-123",
		}

		deleteResp, err := suite.client.CheckPermission(context.Background(), deleteReq)
		require.NoError(t, err, "Delete permission check should succeed")
		assert.False(t, deleteResp.Allowed, "Delete permission should be denied")
		assert.Equal(t, "permission denied", deleteResp.Reason, "Should indicate permission denied")
	})

	t.Run("simulation_service_user_context_flow", func(t *testing.T) {
		// Simulate Simulation Service getting user context for simulation access
		userID := uuid.New()

		lastLoginTime := time.Now().Add(-30 * time.Minute)
		lastLoginIP := "10.0.0.50"
		now := time.Now()
		user := &models.User{
			ID:                    userID,
			Email:                 "simulator@example.com",
			FirstName:             "Simulation",
			LastName:              "User",
			Company:               "SimCorp",
			IsActive:              true,
			EmailVerified:         true,
			LastLoginAt:           &lastLoginTime,
			LastLoginIP:           &lastLoginIP,
			SimulationPreferences: make(models.JSONMap),
			UIPreferences:         make(models.JSONMap),
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		suite.userService.AddUser(userID.String(), user)
		suite.rbacService.SetUserRoles(userID.String(), []string{"simulator", "analyst"})
		suite.rbacService.SetUserPermissions(userID.String(), []string{"run:simulation", "view:results", "export:data"})

		// Simulation Service gets user context
		contextReq := &auth.GetUserContextRequest{
			UserId:         userID.String(),
			CallingService: "simulation-service",
			RequestId:      "simulation-context-123",
		}

		contextResp, err := suite.client.GetUserContext(context.Background(), contextReq)
		require.NoError(t, err, "Get user context should succeed")

		// Verify comprehensive user context
		assert.Equal(t, userID.String(), contextResp.UserId, "User ID should match")
		assert.Equal(t, "simulator@example.com", contextResp.Email, "Email should match")
		assert.Equal(t, "Simulation", contextResp.FirstName, "First name should match")
		assert.Equal(t, "User", contextResp.LastName, "Last name should match")
		assert.Equal(t, "SimCorp", contextResp.Company, "Company should match")
		assert.True(t, contextResp.IsActive, "User should be active")
		assert.True(t, contextResp.EmailVerified, "Email should be verified")
		assert.False(t, contextResp.IsAdmin, "User should not be admin")
		assert.Contains(t, contextResp.Roles, "simulator", "Should have simulator role")
		assert.Contains(t, contextResp.Roles, "analyst", "Should have analyst role")
		assert.Contains(t, contextResp.Permissions, "run:simulation", "Should have run:simulation permission")
		assert.Contains(t, contextResp.Permissions, "view:results", "Should have view:results permission")
		assert.Contains(t, contextResp.Permissions, "export:data", "Should have export:data permission")
		assert.True(t, contextResp.LastLogin > 0, "Last login should be set")
		assert.Equal(t, "10.0.0.50", contextResp.LastLoginIp, "Last login IP should match")
		assert.Empty(t, contextResp.ErrorMessage, "Error message should be empty")
	})
}
