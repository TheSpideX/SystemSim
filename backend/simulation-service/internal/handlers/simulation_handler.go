package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/systemsim/simulation-service/internal/simulation"
)

// SimulationHandler handles simulation-related HTTP requests
type SimulationHandler struct {
	simManager *simulation.Manager
}

// NewSimulationHandler creates a new simulation handler
func NewSimulationHandler(simManager *simulation.Manager) *SimulationHandler {
	return &SimulationHandler{
		simManager: simManager,
	}
}

// CreateSimulation creates a new simulation
func (h *SimulationHandler) CreateSimulation(c *gin.Context) {
	var req simulation.CreateSimulationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	sim, err := h.simManager.CreateSimulation(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create simulation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, sim)
}

// GetSimulation retrieves a simulation by ID
func (h *SimulationHandler) GetSimulation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid simulation ID format",
		})
		return
	}

	sim, err := h.simManager.GetSimulation(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Simulation not found",
		})
		return
	}

	c.JSON(http.StatusOK, sim)
}

// UpdateSimulation updates a simulation
func (h *SimulationHandler) UpdateSimulation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid simulation ID format",
		})
		return
	}

	var req simulation.UpdateSimulationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	sim, err := h.simManager.UpdateSimulation(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update simulation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, sim)
}

// DeleteSimulation deletes a simulation
func (h *SimulationHandler) DeleteSimulation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid simulation ID format",
		})
		return
	}

	if err := h.simManager.DeleteSimulation(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete simulation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// StartSimulation starts a simulation
func (h *SimulationHandler) StartSimulation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid simulation ID format",
		})
		return
	}

	if err := h.simManager.StartSimulation(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start simulation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Simulation started successfully",
	})
}

// StopSimulation stops a simulation
func (h *SimulationHandler) StopSimulation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid simulation ID format",
		})
		return
	}

	if err := h.simManager.StopSimulation(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to stop simulation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Simulation stopped successfully",
	})
}

// PauseSimulation pauses a simulation
func (h *SimulationHandler) PauseSimulation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid simulation ID format",
		})
		return
	}

	if err := h.simManager.PauseSimulation(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to pause simulation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Simulation paused successfully",
	})
}

// ResumeSimulation resumes a simulation
func (h *SimulationHandler) ResumeSimulation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid simulation ID format",
		})
		return
	}

	if err := h.simManager.ResumeSimulation(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to resume simulation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Simulation resumed successfully",
	})
}

// GetSimulationStatus gets the status of a simulation
func (h *SimulationHandler) GetSimulationStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid simulation ID format",
		})
		return
	}

	status, err := h.simManager.GetSimulationStatus(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Simulation not found",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetSimulationMetrics gets metrics for a simulation
func (h *SimulationHandler) GetSimulationMetrics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid simulation ID format",
		})
		return
	}

	metrics, err := h.simManager.GetSimulationMetrics(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Simulation not found",
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// Component-related handlers (placeholder implementations)
func (h *SimulationHandler) CreateComponent(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Component creation not yet implemented",
	})
}

func (h *SimulationHandler) GetComponent(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Component retrieval not yet implemented",
	})
}

func (h *SimulationHandler) UpdateComponent(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Component update not yet implemented",
	})
}

func (h *SimulationHandler) DeleteComponent(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Component deletion not yet implemented",
	})
}

func (h *SimulationHandler) GetComponentMetrics(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Component metrics not yet implemented",
	})
}

// Engine-related handlers (placeholder implementations)
func (h *SimulationHandler) GetEngineProfiles(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Engine profiles not yet implemented",
	})
}

func (h *SimulationHandler) GetEngineTemplates(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Engine templates not yet implemented",
	})
}

// WebSocket handler (placeholder implementation)
func (h *SimulationHandler) HandleWebSocket(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "WebSocket support not yet implemented",
	})
}
