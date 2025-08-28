package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	auth "github.com/systemsim/auth-service/internal/proto"
	"github.com/systemsim/auth-service/internal/health"
	"github.com/systemsim/auth-service/internal/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthGRPCHandler implements the gRPC AuthService interface
type AuthGRPCHandler struct {
	auth.UnimplementedAuthServiceServer
	authService     *services.AuthService
	rbacService     *services.RBACService
	userService     *services.UserService
	healthChecker   *health.EnhancedHealthChecker
}

// NewAuthGRPCHandler creates a new gRPC auth handler
func NewAuthGRPCHandler(authService *services.AuthService, rbacService *services.RBACService, userService *services.UserService, healthChecker *health.EnhancedHealthChecker) *AuthGRPCHandler {
	return &AuthGRPCHandler{
		authService:   authService,
		rbacService:   rbacService,
		userService:   userService,
		healthChecker: healthChecker,
	}
}

// ValidateToken validates a JWT token and returns user context
func (h *AuthGRPCHandler) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
	log.Printf("gRPC ValidateToken called by %s (request_id: %s)", req.CallingService, req.RequestId)

	if req.Token == "" {
		return &auth.ValidateTokenResponse{
			Valid:        false,
			ErrorMessage: "token is required",
		}, nil
	}

	// Validate the token using existing auth service
	claims, err := h.authService.ValidateAccessToken(req.Token)
	if err != nil {
		log.Printf("Token validation failed: %v", err)
		return &auth.ValidateTokenResponse{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("invalid token: %v", err),
		}, nil
	}

	// Get user permissions
	permissions, err := h.rbacService.GetUserPermissionsForGRPC(claims.UserID.String())
	if err != nil {
		log.Printf("Failed to get user permissions: %v", err)
		// Don't fail the request, just return empty permissions
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

// GetUserContext retrieves comprehensive user context
func (h *AuthGRPCHandler) GetUserContext(ctx context.Context, req *auth.GetUserContextRequest) (*auth.GetUserContextResponse, error) {
	log.Printf("gRPC GetUserContext called by %s for user %s (request_id: %s)", req.CallingService, req.UserId, req.RequestId)

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Get user details
	user, err := h.userService.GetUserByID(req.UserId)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return &auth.GetUserContextResponse{
			ErrorMessage: fmt.Sprintf("user not found: %v", err),
		}, nil
	}

	// Get user roles and permissions
	roles, err := h.rbacService.GetUserRolesForGRPC(req.UserId)
	if err != nil {
		log.Printf("Failed to get user roles: %v", err)
		roles = []string{} // Don't fail, just return empty roles
	}

	permissions, err := h.rbacService.GetUserPermissionsForGRPC(req.UserId)
	if err != nil {
		log.Printf("Failed to get user permissions: %v", err)
		permissions = []string{} // Don't fail, just return empty permissions
	}

	var lastLogin int64
	if user.LastLoginAt != nil {
		lastLogin = user.LastLoginAt.Unix()
	}

	var lastLoginIP string
	if user.LastLoginIP != nil {
		lastLoginIP = *user.LastLoginIP
	}

	return &auth.GetUserContextResponse{
		UserId:        user.ID.String(),
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Company:       user.Company,
		Roles:         roles,
		Permissions:   permissions,
		IsActive:      user.IsActive,
		IsAdmin:       user.IsAdmin,
		LastLogin:     lastLogin,
		LastLoginIp:   lastLoginIP,
		EmailVerified: user.EmailVerified,
	}, nil
}

// CheckPermission checks if a user has a specific permission
func (h *AuthGRPCHandler) CheckPermission(ctx context.Context, req *auth.CheckPermissionRequest) (*auth.CheckPermissionResponse, error) {
	log.Printf("gRPC CheckPermission called by %s for user %s, permission %s (request_id: %s)", 
		req.CallingService, req.UserId, req.Permission, req.RequestId)

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}

	// Check permission using RBAC service
	hasPermission, err := h.rbacService.CheckUserPermission(req.UserId, req.Permission)
	if err != nil {
		log.Printf("Permission check failed: %v", err)
		return &auth.CheckPermissionResponse{
			Allowed:      false,
			Reason:       fmt.Sprintf("permission check failed: %v", err),
			UserId:       req.UserId,
			Permission:   req.Permission,
			ResourceId:   req.ResourceId,
			ErrorMessage: err.Error(),
		}, nil
	}

	reason := "permission granted"
	if !hasPermission {
		reason = "permission denied"
	}

	return &auth.CheckPermissionResponse{
		Allowed:    hasPermission,
		Reason:     reason,
		UserId:     req.UserId,
		Permission: req.Permission,
		ResourceId: req.ResourceId,
	}, nil
}

// ValidateSession validates a user session
func (h *AuthGRPCHandler) ValidateSession(ctx context.Context, req *auth.ValidateSessionRequest) (*auth.ValidateSessionResponse, error) {
	log.Printf("gRPC ValidateSession called by %s for session %s (request_id: %s)", 
		req.CallingService, req.SessionId, req.RequestId)

	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	// Get session details
	session, err := h.userService.GetSessionByID(req.SessionId)
	if err != nil {
		log.Printf("Session validation failed: %v", err)
		return &auth.ValidateSessionResponse{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("session not found: %v", err),
		}, nil
	}

	// Check if session is active and not expired
	isValid := session.IsActive && session.ExpiresAt.After(time.Now())
	if req.UserId != "" && session.UserID.String() != req.UserId {
		isValid = false
	}

	var deviceInfo string
	if session.DeviceInfo != nil {
		// Convert JSONMap to string representation
		deviceInfo = fmt.Sprintf("%v", session.DeviceInfo)
	}

	var ipAddress string
	if session.IPAddress != nil {
		ipAddress = *session.IPAddress
	}

	return &auth.ValidateSessionResponse{
		Valid:      isValid,
		SessionId:  session.ID.String(),
		UserId:     session.UserID.String(),
		IsActive:   session.IsActive,
		ExpiresAt:  session.ExpiresAt.Unix(),
		LastUsedAt: session.LastUsedAt.Unix(),
		DeviceInfo: deviceInfo,
		IpAddress:  ipAddress,
	}, nil
}

// GetUserPermissions retrieves all permissions for a user
func (h *AuthGRPCHandler) GetUserPermissions(ctx context.Context, req *auth.GetUserPermissionsRequest) (*auth.GetUserPermissionsResponse, error) {
	log.Printf("gRPC GetUserPermissions called by %s for user %s (request_id: %s)", 
		req.CallingService, req.UserId, req.RequestId)

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Get user permissions
	permissions, err := h.rbacService.GetUserPermissionsForGRPC(req.UserId)
	if err != nil {
		log.Printf("Failed to get user permissions: %v", err)
		return &auth.GetUserPermissionsResponse{
			ErrorMessage: fmt.Sprintf("failed to get permissions: %v", err),
		}, nil
	}

	response := &auth.GetUserPermissionsResponse{
		UserId:      req.UserId,
		Permissions: permissions,
	}

	// Include roles if requested
	if req.IncludeRoles {
		roles, err := h.rbacService.GetUserRolesWithDetails(req.UserId)
		if err != nil {
			log.Printf("Failed to get user roles: %v", err)
		} else {
			response.Roles = make([]*auth.UserRole, len(roles))
			for i, role := range roles {
				rolePermissions, _ := h.rbacService.GetRolePermissions(role.ID.String())
				response.Roles[i] = &auth.UserRole{
					RoleId:      role.ID.String(),
					RoleName:    role.Name,
					Description: role.Description,
					IsSystem:    role.IsSystem,
					Permissions: rolePermissions,
				}
			}
		}
	}

	// Check if user is admin
	user, err := h.userService.GetUserByID(req.UserId)
	if err == nil {
		response.IsAdmin = user.IsAdmin
	}

	return response, nil
}

// HealthCheck performs a comprehensive health check
func (h *AuthGRPCHandler) HealthCheck(ctx context.Context, req *auth.HealthCheckRequest) (*auth.HealthCheckResponse, error) {
	log.Printf("gRPC HealthCheck called by %s (request_id: %s)", req.CallingService, req.RequestId)

	// Perform comprehensive health checks
	healthStatus := h.healthChecker.CheckHealth(ctx)

	// Convert to gRPC response format
	details := &auth.HealthDetails{
		DatabaseHealthy: healthStatus.Details.Database.Healthy,
		RedisHealthy:    healthStatus.Details.Redis.Healthy,
		ResponseTimeMs:  healthStatus.ResponseTimeMs,
		Uptime:          healthStatus.Uptime,
	}

	return &auth.HealthCheckResponse{
		Healthy:   healthStatus.Healthy,
		Status:    healthStatus.Status,
		Version:   healthStatus.Version,
		Timestamp: healthStatus.Timestamp,
		Details:   details,
	}, nil
}
