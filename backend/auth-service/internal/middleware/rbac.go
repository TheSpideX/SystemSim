package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/services"
)

// RBACMiddleware provides role-based access control middleware
type RBACMiddleware struct {
	rbacService *services.RBACService
}

// NewRBACMiddleware creates a new RBAC middleware
func NewRBACMiddleware(rbacService *services.RBACService) *RBACMiddleware {
	return &RBACMiddleware{
		rbacService: rbacService,
	}
}

// RequirePermission creates middleware that requires a specific permission
func (m *RBACMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User ID not found in context",
			})
			c.Abort()
			return
		}

		userUUID, ok := userID.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid user ID format",
			})
			c.Abort()
			return
		}

		// Check if user has the required permission
		hasPermission, err := m.rbacService.HasPermission(userUUID, permission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to check user permissions",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "insufficient_permissions",
				Message: "You don't have permission to perform this action",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin creates middleware that requires admin role
func (m *RBACMiddleware) RequireAdmin() gin.HandlerFunc {
	return m.RequirePermission(models.PermissionSystemAdmin)
}

// RequireUserManagement creates middleware that requires user management permissions
func (m *RBACMiddleware) RequireUserManagement() gin.HandlerFunc {
	return m.RequirePermission(models.PermissionUsersUpdate)
}

// RequireSessionManagement creates middleware that requires session management permissions
func (m *RBACMiddleware) RequireSessionManagement() gin.HandlerFunc {
	return m.RequirePermission(models.PermissionSessionsRevoke)
}

// AddUserPermissions adds user permissions to the context for use in handlers
func (m *RBACMiddleware) AddUserPermissions() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		userUUID, ok := userID.(uuid.UUID)
		if !ok {
			c.Next()
			return
		}

		// Get user permissions
		permissions, err := m.rbacService.GetUserPermissions(userUUID)
		if err != nil {
			// Log error but don't fail the request
			c.Next()
			return
		}

		// Add permissions to context
		permissionNames := make([]string, len(permissions))
		for i, perm := range permissions {
			permissionNames[i] = perm.Name
		}
		c.Set("user_permissions", permissionNames)

		// Check if user is admin
		isAdmin, err := m.rbacService.IsUserAdmin(userUUID)
		if err == nil {
			c.Set("is_admin", isAdmin)
		}

		c.Next()
	}
}

// CheckPermissionInHandler is a helper function to check permissions within handlers
func (m *RBACMiddleware) CheckPermissionInHandler(c *gin.Context, permission string) bool {
	userID, exists := c.Get("user_id")
	if !exists {
		return false
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		return false
	}

	hasPermission, err := m.rbacService.HasPermission(userUUID, permission)
	if err != nil {
		return false
	}

	return hasPermission
}

// GetUserPermissionsFromContext retrieves user permissions from context
func GetUserPermissionsFromContext(c *gin.Context) []string {
	permissions, exists := c.Get("user_permissions")
	if !exists {
		return []string{}
	}

	permissionSlice, ok := permissions.([]string)
	if !ok {
		return []string{}
	}

	return permissionSlice
}

// IsAdminFromContext checks if user is admin from context
func IsAdminFromContext(c *gin.Context) bool {
	isAdmin, exists := c.Get("is_admin")
	if !exists {
		return false
	}

	adminBool, ok := isAdmin.(bool)
	if !ok {
		return false
	}

	return adminBool
}
