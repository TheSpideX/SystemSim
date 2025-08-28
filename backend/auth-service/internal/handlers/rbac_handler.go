package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/services"
)

// RBACHandler handles RBAC-related HTTP requests
type RBACHandler struct {
	rbacService *services.RBACService
}

// NewRBACHandler creates a new RBAC handler
func NewRBACHandler(rbacService *services.RBACService) *RBACHandler {
	return &RBACHandler{
		rbacService: rbacService,
	}
}

// GetMyRoles handles getting current user's roles
func (h *RBACHandler) GetMyRoles(c *gin.Context) {
	userUUID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	roles, err := h.rbacService.GetUserRoles(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "roles_fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"roles": roles,
	})
}

// GetMyPermissions handles getting current user's permissions
func (h *RBACHandler) GetMyPermissions(c *gin.Context) {
	userUUID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	permissions, err := h.rbacService.GetUserPermissions(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "permissions_fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"permissions": permissions,
	})
}

// GetAllRoles handles getting all system roles (admin only)
func (h *RBACHandler) GetAllRoles(c *gin.Context) {
	userUUID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	roles, err := h.rbacService.GetAllRoles(userUUID)
	if err != nil {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "access_denied",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"roles": roles,
	})
}

// GetAllPermissions handles getting all system permissions (admin only)
func (h *RBACHandler) GetAllPermissions(c *gin.Context) {
	userUUID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	permissions, err := h.rbacService.GetAllPermissions(userUUID)
	if err != nil {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "access_denied",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"permissions": permissions,
	})
}

// AssignRoleRequest represents role assignment request
type AssignRoleRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	RoleName string `json:"role_name" binding:"required"`
}

// AssignRole handles assigning a role to a user (admin only)
func (h *RBACHandler) AssignRole(c *gin.Context) {
	adminUUID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	if err := h.rbacService.AssignRoleToUser(adminUUID, targetUserID, req.RoleName); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "role_assignment_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Role assigned successfully",
	})
}

// RemoveRoleRequest represents role removal request
type RemoveRoleRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	RoleName string `json:"role_name" binding:"required"`
}

// RemoveRole handles removing a role from a user (admin only)
func (h *RBACHandler) RemoveRole(c *gin.Context) {
	adminUUID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	var req RemoveRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	if err := h.rbacService.RemoveRoleFromUser(adminUUID, targetUserID, req.RoleName); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "role_removal_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Role removed successfully",
	})
}

// GetUserRoles handles getting roles for a specific user (admin only)
func (h *RBACHandler) GetUserRoles(c *gin.Context) {
	adminUUID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Check admin permission
	if err := h.rbacService.ValidatePermission(adminUUID, models.PermissionUsersRead); err != nil {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "access_denied",
			Message: err.Error(),
		})
		return
	}

	userIDStr := c.Param("userId")
	targetUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	roles, err := h.rbacService.GetUserRoles(targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "roles_fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"user_id": targetUserID,
		"roles":   roles,
	})
}

// Helper function to extract user ID from context
func getUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid user ID format")
	}

	return userUUID, nil
}
