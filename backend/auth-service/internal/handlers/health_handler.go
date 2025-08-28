package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/systemsim/auth-service/internal/health"
	"github.com/systemsim/auth-service/internal/metrics"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	healthChecker         *health.HealthChecker
	enhancedHealthChecker *health.EnhancedHealthChecker
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthChecker *health.HealthChecker, enhancedHealthChecker *health.EnhancedHealthChecker) *HealthHandler {
	return &HealthHandler{
		healthChecker:         healthChecker,
		enhancedHealthChecker: enhancedHealthChecker,
	}
}

// LivenessCheck handles liveness probe requests
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	status := h.healthChecker.CheckLiveness()
	
	if status.Status == "healthy" {
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusServiceUnavailable, status)
	}
}

// ReadinessCheck handles readiness probe requests
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	status := h.healthChecker.CheckReadiness()
	
	if status.Status == "healthy" {
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusServiceUnavailable, status)
	}
}

// DetailedHealthCheck handles detailed health check requests
func (h *HealthHandler) DetailedHealthCheck(c *gin.Context) {
	// Use enhanced health checker for detailed status
	if h.enhancedHealthChecker != nil {
		status := h.enhancedHealthChecker.CheckHealth(c.Request.Context())

		statusCode := http.StatusOK
		if !status.Healthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, status)
		return
	}

	// Fallback to basic health checker
	status := h.healthChecker.CheckDetailed()

	statusCode := http.StatusOK
	switch status.Status {
	case "healthy":
		statusCode = http.StatusOK
	case "degraded":
		statusCode = http.StatusOK // Still OK but with warnings
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, status)
}

// MetricsHandler handles metrics requests
func (h *HealthHandler) MetricsHandler(c *gin.Context) {
	metrics.MetricsHandler(c)
}

// SimpleHealthCheck provides a basic health endpoint for backward compatibility
func (h *HealthHandler) SimpleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "auth-service",
		"timestamp": "2025-07-15T00:00:00Z",
	})
}
