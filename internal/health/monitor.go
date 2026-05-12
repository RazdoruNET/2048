package health

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"socks5-dpi-proxy/internal/ml"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusUnknown   HealthStatus = "unknown"
)

// ComponentHealth represents health information for a component
type ComponentHealth struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	LastCheck   time.Time              `json:"last_check"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	CheckPeriod time.Duration          `json:"check_period"`
}

// HealthCheck interface for components that can be health-checked
type HealthCheck interface {
	Name() string
	CheckHealth(ctx context.Context) error
	GetHealthDetails() map[string]interface{}
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status       HealthStatus                `json:"status"`
	LastUpdate   time.Time                   `json:"last_update"`
	Components   map[string]*ComponentHealth `json:"components"`
	Uptime       time.Duration               `json:"uptime"`
	Version      string                      `json:"version"`
	Environment  string                      `json:"environment"`
}

// HealthMonitor monitors health of system components
type HealthMonitor struct {
	mu          sync.RWMutex
	components  map[string]HealthCheck
	health      map[string]*ComponentHealth
	startTime   time.Time
	version     string
	environment string
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(version, environment string) *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HealthMonitor{
		components:  make(map[string]HealthCheck),
		health:      make(map[string]*ComponentHealth),
		startTime:   time.Now(),
		version:     version,
		environment: environment,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// RegisterComponent registers a component for health monitoring
func (hm *HealthMonitor) RegisterComponent(component HealthCheck) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	name := component.Name()
	hm.components[name] = component
	
	// Initialize health status
	hm.health[name] = &ComponentHealth{
		Name:        name,
		Status:      StatusUnknown,
		LastCheck:   time.Time{},
		Details:     make(map[string]interface{}),
		CheckPeriod: 30 * time.Second, // Default check period
	}
	
	log.Printf("Registered health check for component: %s", name)
}

// UnregisterComponent removes a component from health monitoring
func (hm *HealthMonitor) UnregisterComponent(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	delete(hm.components, name)
	delete(hm.health, name)
	
	log.Printf("Unregistered health check for component: %s", name)
}

// StartMonitoring starts the health monitoring process
func (hm *HealthMonitor) StartMonitoring() {
	go hm.monitorLoop()
	log.Printf("Health monitoring started for %d components", len(hm.components))
}

// StopMonitoring stops the health monitoring process
func (hm *HealthMonitor) StopMonitoring() {
	hm.cancel()
	log.Printf("Health monitoring stopped")
}

// monitorLoop runs the continuous health monitoring
func (hm *HealthMonitor) monitorLoop() {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-ticker.C:
			hm.checkAllComponents()
		}
	}
}

// checkAllComponents performs health checks on all registered components
func (hm *HealthMonitor) checkAllComponents() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	for name, component := range hm.components {
		health := hm.health[name]
		
		// Perform health check with timeout
		ctx, cancel := context.WithTimeout(hm.ctx, 5*time.Second)
		err := component.CheckHealth(ctx)
		cancel()
		
		// Update health status
		now := time.Now()
		health.LastCheck = now
		
		if err != nil {
			health.Status = StatusUnhealthy
			health.Message = err.Error()
		} else {
			health.Status = StatusHealthy
			health.Message = ""
		}
		
		// Update health details
		if details := component.GetHealthDetails(); details != nil {
			health.Details = details
		}
		
		// Add basic metrics
		health.Details["last_check"] = now
		health.Details["check_duration"] = time.Since(now)
	}
}

// GetSystemHealth returns the overall system health
func (hm *HealthMonitor) GetSystemHealth() *SystemHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	// Copy component health
	components := make(map[string]*ComponentHealth)
	for name, health := range hm.health {
		healthCopy := *health
		healthCopy.Details = make(map[string]interface{})
		for k, v := range health.Details {
			healthCopy.Details[k] = v
		}
		components[name] = &healthCopy
	}
	
	// Determine overall system status
	systemStatus := hm.calculateSystemStatus()
	
	return &SystemHealth{
		Status:      systemStatus,
		LastUpdate:  time.Now(),
		Components:  components,
		Uptime:      time.Since(hm.startTime),
		Version:     hm.version,
		Environment: hm.environment,
	}
}

// calculateSystemStatus determines the overall system health status
func (hm *HealthMonitor) calculateSystemStatus() HealthStatus {
	if len(hm.health) == 0 {
		return StatusUnknown
	}
	
	unhealthyCount := 0
	degradedCount := 0
	unknownCount := 0
	
	for _, health := range hm.health {
		switch health.Status {
		case StatusUnhealthy:
			unhealthyCount++
		case StatusDegraded:
			degradedCount++
		case StatusUnknown:
			unknownCount++
		}
	}
	
	totalComponents := len(hm.health)
	
	// If any component is unhealthy, system is unhealthy
	if unhealthyCount > 0 {
		return StatusUnhealthy
	}
	
	// If more than 25% of components are degraded, system is degraded
	if float64(degradedCount)/float64(totalComponents) > 0.25 {
		return StatusDegraded
	}
	
	// If more than 50% of components are unknown, system is unknown
	if float64(unknownCount)/float64(totalComponents) > 0.5 {
		return StatusUnknown
	}
	
	return StatusHealthy
}

// GetComponentHealth returns health information for a specific component
func (hm *HealthMonitor) GetComponentHealth(name string) (*ComponentHealth, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	health, exists := hm.health[name]
	if !exists {
		return nil, fmt.Errorf("component %s not found", name)
	}
	
	// Return a copy to prevent external modification
	healthCopy := *health
	healthCopy.Details = make(map[string]interface{})
	for k, v := range health.Details {
		healthCopy.Details[k] = v
	}
	
	return &healthCopy, nil
}

// GetHealthyComponents returns a list of healthy components
func (hm *HealthMonitor) GetHealthyComponents() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	var healthy []string
	for name, health := range hm.health {
		if health.Status == StatusHealthy {
			healthy = append(healthy, name)
		}
	}
	
	return healthy
}

// GetUnhealthyComponents returns a list of unhealthy components
func (hm *HealthMonitor) GetUnhealthyComponents() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	var unhealthy []string
	for name, health := range hm.health {
		if health.Status == StatusUnhealthy {
			unhealthy = append(unhealthy, name)
		}
	}
	
	return unhealthy
}

// ForceCheck forces a health check for a specific component
func (hm *HealthMonitor) ForceCheck(name string) error {
	hm.mu.RLock()
	component, exists := hm.components[name]
	if !exists {
		hm.mu.RUnlock()
		return fmt.Errorf("component %s not found", name)
	}
	hm.mu.RUnlock()
	
	// Perform health check
	ctx, cancel := context.WithTimeout(hm.ctx, 5*time.Second)
	err := component.CheckHealth(ctx)
	cancel()
	
	// Update health status
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	health := hm.health[name]
	now := time.Now()
	health.LastCheck = now
	
	if err != nil {
		health.Status = StatusUnhealthy
		health.Message = err.Error()
	} else {
		health.Status = StatusHealthy
		health.Message = ""
	}
	
	// Update health details
	if details := component.GetHealthDetails(); details != nil {
		health.Details = details
	}
	
	health.Details["last_check"] = now
	health.Details["forced_check"] = true
	
	return err
}

// SetCheckPeriod sets the check period for a component
func (hm *HealthMonitor) SetCheckPeriod(name string, period time.Duration) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	health, exists := hm.health[name]
	if !exists {
		return fmt.Errorf("component %s not found", name)
	}
	
	health.CheckPeriod = period
	return nil
}

// GetStatistics returns health monitoring statistics
func (hm *HealthMonitor) GetStatistics() map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	healthyCount := 0
	unhealthyCount := 0
	degradedCount := 0
	unknownCount := 0
	
	for _, health := range hm.health {
		switch health.Status {
		case StatusHealthy:
			healthyCount++
		case StatusUnhealthy:
			unhealthyCount++
		case StatusDegraded:
			degradedCount++
		case StatusUnknown:
			unknownCount++
		}
	}
	
	stats["total_components"] = len(hm.health)
	stats["healthy_components"] = healthyCount
	stats["unhealthy_components"] = unhealthyCount
	stats["degraded_components"] = degradedCount
	stats["unknown_components"] = unknownCount
	stats["monitoring_uptime"] = time.Since(hm.startTime).String()
	stats["system_status"] = hm.calculateSystemStatus()
	
	return stats
}

// MLHealthCheck implements HealthCheck for ML engine
type MLHealthCheck struct {
	engine ml.MLEngine
}

// NewMLHealthCheck creates a new ML health check
func NewMLHealthCheck(engine ml.MLEngine) *MLHealthCheck {
	return &MLHealthCheck{
		engine: engine,
	}
}

// Name returns the name of the ML health check
func (mlhc *MLHealthCheck) Name() string {
	return "ml_engine"
}

// CheckHealth checks the health of the ML engine
func (mlhc *MLHealthCheck) CheckHealth(ctx context.Context) error {
	if !mlhc.engine.IsEnabled() {
		return fmt.Errorf("ML engine is disabled")
	}
	
	return mlhc.engine.HealthCheck()
}

// GetHealthDetails returns health details for the ML engine
func (mlhc *MLHealthCheck) GetHealthDetails() map[string]interface{} {
	status := mlhc.engine.GetStatus()
	stats := mlhc.engine.GetStatistics()
	
	details := make(map[string]interface{})
	details["enabled"] = status.Enabled
	details["model_version"] = status.ModelVersion
	details["uptime"] = status.Uptime.String()
	details["health"] = status.Health
	details["techniques_count"] = len(status.Techniques)
	details["last_update"] = status.LastUpdate
	
	// Add statistics
	for k, v := range stats {
		details[k] = v
	}
	
	// Add retraining status
	progress := mlhc.engine.GetRetrainingProgress()
	details["retraining_active"] = progress.Active
	details["retraining_progress"] = progress.Progress
	
	return details
}
