package handlers

import (
	"net/http"
	"strconv"
	"time"

	"socks5-dpi-proxy/internal/ml"
	"socks5-dpi-proxy/internal/storage"

	"github.com/gin-gonic/gin"
)

type MLHandler struct {
	mlEngine ml.MLEngine
	storage  *storage.Storage
}

func NewMLHandler(mlEngine ml.MLEngine, storage *storage.Storage) *MLHandler {
	return &MLHandler{
		mlEngine: mlEngine,
		storage:  storage,
	}
}

type MLStatusResponse struct {
	Enabled      bool      `json:"enabled"`
	LastUpdate   time.Time `json:"last_update"`
	ModelVersion string    `json:"model_version"`
	Techniques   []string  `json:"techniques"`
	Health       string    `json:"health"`
	Uptime       string    `json:"uptime"`
}

func (h *MLHandler) GetStatus(c *gin.Context) {
	// Get real status from ML engine
	status := h.mlEngine.GetStatus()

	response := MLStatusResponse{
		Enabled:      status.Enabled,
		LastUpdate:   status.LastUpdate,
		ModelVersion: status.ModelVersion,
		Techniques:   status.Techniques,
		Health:       status.Health,
		Uptime:       status.Uptime.String(),
	}

	c.JSON(http.StatusOK, response)
}

type MLStatisticsResponse struct {
	TotalRequests     int64     `json:"total_requests"`
	SuccessRate       float64   `json:"success_rate"`
	AverageLatency    int64     `json:"average_latency"`
	ThroughputBPS     int64     `json:"throughput_bps"`
	ErrorRate         float64   `json:"error_rate"`
	ActiveConnections int       `json:"active_connections"`
	Timestamp         time.Time `json:"timestamp"`
}

func (h *MLHandler) GetStatistics(c *gin.Context) {
	// Get latest statistics from storage
	stats, err := h.storage.GetLatestMLStatistics()
	if err != nil {
		// Return default values if no statistics exist
		response := MLStatisticsResponse{
			TotalRequests:     0,
			SuccessRate:       0.0,
			AverageLatency:    0,
			ThroughputBPS:     0,
			ErrorRate:         0.0,
			ActiveConnections: 0,
			Timestamp:         time.Now(),
		}
		c.JSON(http.StatusOK, response)
		return
	}

	response := MLStatisticsResponse{
		TotalRequests:     int64(stats.TotalRequests),
		SuccessRate:       stats.SuccessRate,
		AverageLatency:    stats.AverageLatency,
		ThroughputBPS:     stats.ThroughputBPS,
		ErrorRate:         stats.ErrorRate,
		ActiveConnections: stats.ActiveConnections,
		Timestamp:         stats.Timestamp,
	}

	c.JSON(http.StatusOK, response)
}

type TechniqueResponse struct {
	Technique     string    `json:"technique"`
	Effectiveness float64   `json:"effectiveness"`
	SampleCount   int       `json:"sample_count"`
	LastUpdate    time.Time `json:"last_update"`
	Domain        string    `json:"domain,omitempty"`
}

func (h *MLHandler) GetTechniques(c *gin.Context) {
	domain := c.Query("domain")

	techniques, err := h.storage.GetTechniqueEffectiveness(domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]TechniqueResponse, len(techniques))
	for i, tech := range techniques {
		response[i] = TechniqueResponse{
			Technique:     tech.Technique,
			Effectiveness: tech.Effectiveness,
			SampleCount:   tech.SampleCount,
			LastUpdate:    tech.LastUpdate,
			Domain:        tech.Domain,
		}
	}

	c.JSON(http.StatusOK, response)
}

type HistoryResponse struct {
	Hours      int                    `json:"hours"`
	Statistics []storage.MLStatistics `json:"statistics"`
}

func (h *MLHandler) GetHistory(c *gin.Context) {
	hoursStr := c.Param("hours")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hours parameter"})
		return
	}

	if hours > 24 {
		hours = 24 // Limit to 24 hours
	}

	stats, err := h.storage.GetMLStatistics(hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := HistoryResponse{
		Hours:      hours,
		Statistics: stats,
	}

	c.JSON(http.StatusOK, response)
}

type FeedbackRequest struct {
	Domain    string `json:"domain" binding:"required"`
	Technique string `json:"technique" binding:"required"`
	Success   bool   `json:"success" binding:"required"`
}

func (h *MLHandler) PostFeedback(c *gin.Context) {
	var req FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update ML engine with feedback
	err := h.mlEngine.LearnFromFeedback(req.Domain, req.Technique, req.Success)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update technique effectiveness in storage
	effectiveness := 0.2 // Default for failure
	if req.Success {
		effectiveness = 0.9 // Default for success
	}

	techEffectiveness := &storage.TechniqueEffectiveness{
		Technique:     req.Technique,
		Domain:        req.Domain,
		Effectiveness: effectiveness,
		SampleCount:   1,
		LastUpdate:    time.Now(),
	}

	err = h.storage.UpdateTechniqueEffectiveness(techEffectiveness)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feedback recorded successfully"})
}

type UpdateTechniqueRequest struct {
	Effectiveness float64 `json:"effectiveness" binding:"required,min=0,max=1"`
}

func (h *MLHandler) UpdateTechniqueEffectiveness(c *gin.Context) {
	technique := c.Param("technique")
	domain := c.Query("domain")

	var req UpdateTechniqueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current technique effectiveness to update sample count
	techniques, err := h.storage.GetTechniqueEffectiveness(domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	sampleCount := 1
	for _, tech := range techniques {
		if tech.Technique == technique {
			sampleCount = tech.SampleCount + 1
			break
		}
	}

	techEffectiveness := &storage.TechniqueEffectiveness{
		Technique:     technique,
		Domain:        domain,
		Effectiveness: req.Effectiveness,
		SampleCount:   sampleCount,
		LastUpdate:    time.Now(),
	}

	err = h.storage.UpdateTechniqueEffectiveness(techEffectiveness)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Technique effectiveness updated successfully"})
}

type RetrainRequest struct {
	Force bool `json:"force"`
}

func (h *MLHandler) RetrainModel(c *gin.Context) {
	var req RetrainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = RetrainRequest{Force: false} // Default value
	}

	// Start actual model retraining
	err := h.mlEngine.StartRetraining(req.Force)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get retraining progress
	progress := h.mlEngine.GetRetrainingProgress()

	c.JSON(http.StatusOK, gin.H{
		"message":  "Model retraining initiated",
		"force":    req.Force,
		"status":   progress.CurrentStep,
		"progress": progress.Progress,
		"active":   progress.Active,
	})
}

type MLConfigRequest struct {
	LearningRate           float64 `json:"learning_rate" binding:"min=0,max=1"`
	ConfidenceThreshold    float64 `json:"confidence_threshold" binding:"min=0,max=1"`
	ModelUpdateInterval    string  `json:"model_update_interval"`
	EffectivenessThreshold float64 `json:"effectiveness_threshold" binding:"min=0,max=1"`
}

func (h *MLHandler) UpdateConfig(c *gin.Context) {
	var req MLConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create ML config from request
	config := &ml.MLConfig{
		Enabled:        true, // Default to enabled
		UpdateInterval: parseDuration(req.ModelUpdateInterval),
		MaxRetries:     10,               // Default value
		Timeout:        30 * time.Second, // Default value
		CustomParameters: map[string]interface{}{
			"learning_rate":           req.LearningRate,
			"confidence_threshold":    req.ConfidenceThreshold,
			"effectiveness_threshold": req.EffectivenessThreshold,
		},
	}

	// Update ML engine configuration
	err := h.mlEngine.UpdateConfig(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get updated config
	updatedConfig := h.mlEngine.GetConfig()

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
		"config":  updatedConfig,
	})
}

// parseDuration converts string duration to time.Duration
func parseDuration(s string) time.Duration {
	if s == "" {
		return time.Hour // Default
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return time.Hour // Default on error
	}

	return duration
}
