package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/systemsim/simulation-service/internal/health"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	healthChecker *health.HealthChecker
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthChecker *health.HealthChecker) *HealthHandler {
	return &HealthHandler{
		healthChecker: healthChecker,
	}
}

// Health handles the general health check endpoint
func (h *HealthHandler) Health(c *gin.Context) {
	report := h.healthChecker.CheckHealth()
	
	statusCode := http.StatusOK
	if report.Status == health.HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if report.Status == health.HealthStatusDegraded {
		statusCode = http.StatusOK // Still serving requests but with warnings
	}
	
	c.JSON(statusCode, report)
}

// Ready handles the readiness probe endpoint
func (h *HealthHandler) Ready(c *gin.Context) {
	report := h.healthChecker.CheckReadiness()
	
	statusCode := http.StatusOK
	if report.Status != health.HealthStatusHealthy {
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, report)
}

// Live handles the liveness probe endpoint
func (h *HealthHandler) Live(c *gin.Context) {
	report := h.healthChecker.CheckLiveness()
	c.JSON(http.StatusOK, report)
}
