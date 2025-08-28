package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/services"
)

// Helper function to extract user ID from context
func (h *UserHandler) getUserID(c *gin.Context) (uuid.UUID, error) {
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

// Helper function to extract session ID from context
func (h *UserHandler) getSessionID(c *gin.Context) (uuid.UUID, error) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("session ID not found in context")
	}

	sessionUUID, ok := sessionID.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid session ID format")
	}

	return sessionUUID, nil
}

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService *services.UserService
	authService *services.AuthService
	validator   *validator.Validate
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *services.UserService, authService *services.AuthService) *UserHandler {
	return &UserHandler{
		userService: userService,
		authService: authService,
		validator:   validator.New(),
	}
}

// GetProfile handles getting user profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	// Convert user ID to UUID
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	// Get user profile
	profile, err := h.userService.GetProfile(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "profile_fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile handles updating user profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	// Convert user ID to UUID
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request format",
			Details: map[string]string{"json": err.Error()},
		})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_failed",
			Message: "Request validation failed",
			Details: h.getValidationErrors(err),
		})
		return
	}

	// Update profile
	profile, err := h.userService.UpdateProfile(userUUID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "profile_update_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ChangePassword handles password change
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	// Convert user ID to UUID
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request format",
			Details: map[string]string{"json": err.Error()},
		})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_failed",
			Message: "Request validation failed",
			Details: h.getValidationErrors(err),
		})
		return
	}

	// Change password
	if err := h.userService.ChangePassword(userUUID, &req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "password_change_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Password changed successfully",
	})
}

// DeleteAccount handles account deletion
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	// Convert user ID to UUID
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	// Delete account
	if err := h.userService.DeleteAccount(userUUID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "account_deletion_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Account deleted successfully",
	})
}

// GetSessions handles getting user sessions
func (h *UserHandler) GetSessions(c *gin.Context) {
	userUUID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Get user sessions
	sessions, err := h.userService.GetUserSessions(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "sessions_fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"sessions": sessions,
	})
}

// GetStats handles getting user statistics
func (h *UserHandler) GetStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	// Convert user ID to UUID
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	// Get user stats
	stats, err := h.userService.GetUserStats(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "stats_fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// RevokeSession handles revoking a specific session
func (h *UserHandler) RevokeSession(c *gin.Context) {
	userUUID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Get session ID from URL parameter
	sessionIDStr := c.Param("sessionId")
	sessionUUID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_session_id",
			Message: "Invalid session ID format",
		})
		return
	}

	// Revoke the session
	if err := h.userService.RevokeSession(userUUID, sessionUUID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "session_revoke_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Session revoked successfully",
	})
}

// RevokeAllSessions handles revoking all sessions except current
func (h *UserHandler) RevokeAllSessions(c *gin.Context) {
	userUUID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	sessionUUID, err := h.getSessionID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Revoke all sessions except current
	if err := h.userService.RevokeAllSessions(userUUID, sessionUUID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "sessions_revoke_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "All other sessions revoked successfully",
	})
}

// getValidationErrors converts validator errors to a map
func (h *UserHandler) getValidationErrors(err error) map[string]string {
	errors := make(map[string]string)
	
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors[e.Field()] = h.getValidationMessage(e)
		}
	}
	
	return errors
}

// getValidationMessage returns a user-friendly validation message
func (h *UserHandler) getValidationMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return "Must be at least " + e.Param() + " characters long"
	case "max":
		return "Must be no more than " + e.Param() + " characters long"
	default:
		return "Invalid value"
	}
}
