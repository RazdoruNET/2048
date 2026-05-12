package handlers

import (
	"net/http"

	"socks5-dpi-proxy/internal/health"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	healthMonitor *health.HealthMonitor
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthMonitor *health.HealthMonitor) *HealthHandler {
	return &HealthHandler{
		healthMonitor: healthMonitor,
	}
}

// GetSystemHealth returns the overall system health
func (h *HealthHandler) GetSystemHealth(c *gin.Context) {
	systemHealth := h.healthMonitor.GetSystemHealth()
	c.JSON(http.StatusOK, systemHealth)
}

// GetComponentHealth returns health information for a specific component
func (h *HealthHandler) GetComponentHealth(c *gin.Context) {
	componentName := c.Param("component")
	
	componentHealth, err := h.healthMonitor.GetComponentHealth(componentName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, componentHealth)
}

// ForceHealthCheck forces a health check for a specific component
func (h *HealthHandler) ForceHealthCheck(c *gin.Context) {
	componentName := c.Param("component")
	
	err := h.healthMonitor.ForceCheck(componentName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	componentHealth, err := h.healthMonitor.GetComponentHealth(componentName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Health check forced successfully",
		"health":  componentHealth,
	})
}

// GetHealthStatistics returns health monitoring statistics
func (h *HealthHandler) GetHealthStatistics(c *gin.Context) {
	stats := h.healthMonitor.GetStatistics()
	c.JSON(http.StatusOK, stats)
}

// LivenessProbe returns a simple liveness check
func (h *HealthHandler) LivenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"timestamp": gin.H{},
	})
}

// ReadinessProbe returns readiness status
func (h *HealthHandler) ReadinessProbe(c *gin.Context) {
	systemHealth := h.healthMonitor.GetSystemHealth()
	
	ready := systemHealth.Status == health.StatusHealthy || systemHealth.Status == health.StatusDegraded
	
	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, gin.H{
		"ready":    ready,
		"status":   systemHealth.Status,
		"checks":   systemHealth.Components,
	})
}
