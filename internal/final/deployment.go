package final

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// DeploymentManager manages production deployment of the system
type DeploymentManager struct {
	finalManager    *FinalManager
	testManager     *TestManager
	config          *DeploymentConfig
	mu              sync.RWMutex
	running         bool
	deploymentState *DeploymentState
}

// DeploymentConfig defines deployment configuration
type DeploymentConfig struct {
	Environment string             `yaml:"environment"`
	Target      *TargetConfig      `yaml:"target"`
	HealthCheck *HealthCheckConfig `yaml:"health_check"`
	Rollback    *RollbackConfig    `yaml:"rollback"`
	Monitoring  *MonitoringConfig  `yaml:"monitoring"`
	Backup      *BackupConfig      `yaml:"backup"`
}

// TargetConfig defines deployment target
type TargetConfig struct {
	Hosts       []string `yaml:"hosts"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
	KeyPath     string   `yaml:"key_path"`
	DeployPath  string   `yaml:"deploy_path"`
	ServiceName string   `yaml:"service_name"`
	Port        int      `yaml:"port"`
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Interval  time.Duration `yaml:"interval"`
	Timeout   time.Duration `yaml:"timeout"`
	Retries   int           `yaml:"retries"`
	Endpoints []string      `yaml:"endpoints"`
}

// RollbackConfig defines rollback configuration
type RollbackConfig struct {
	Enabled        bool          `yaml:"enabled"`
	AutoRollback   bool          `yaml:"auto_rollback"`
	Threshold      float64       `yaml:"threshold"`
	Window         time.Duration `yaml:"window"`
	BackupVersions int           `yaml:"backup_versions"`
}

// MonitoringConfig defines monitoring configuration
type MonitoringConfig struct {
	Enabled         bool   `yaml:"enabled"`
	MetricsPort     int    `yaml:"metrics_port"`
	LogLevel        string `yaml:"log_level"`
	AlertingEnabled bool   `yaml:"alerting_enabled"`
}

// BackupConfig defines backup configuration
type BackupConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Path      string        `yaml:"path"`
	Retention time.Duration `yaml:"retention"`
	Compress  bool          `yaml:"compress"`
}

// DeploymentState represents the current deployment state
type DeploymentState struct {
	Version         string            `json:"version"`
	Status          DeploymentStatus  `json:"status"`
	StartTime       time.Time         `json:"start_time"`
	EndTime         time.Time         `json:"end_time"`
	Duration        time.Duration     `json:"duration"`
	Health          *DeploymentHealth `json:"health"`
	RollbackVersion string            `json:"rollback_version"`
	Errors          []string          `json:"errors"`
}

// DeploymentStatus represents deployment status
type DeploymentStatus int

const (
	DeploymentStatusPending DeploymentStatus = iota
	DeploymentStatusRunning
	DeploymentStatusCompleted
	DeploymentStatusFailed
	DeploymentStatusRollingBack
	DeploymentStatusRolledBack
)

// DeploymentHealth represents deployment health status
type DeploymentHealth struct {
	OverallStatus string                    `json:"overall_status"`
	Services      map[string]*ServiceHealth `json:"services"`
	Metrics       *DeploymentMetrics        `json:"metrics"`
	LastCheck     time.Time                 `json:"last_check"`
}

// ServiceHealth represents health of a service
type ServiceHealth struct {
	Name            string        `json:"name"`
	Status          string        `json:"status"`
	Uptime          time.Duration `json:"uptime"`
	CPUUsage        float64       `json:"cpu_usage"`
	MemoryUsage     float64       `json:"memory_usage"`
	ConnectionCount int           `json:"connection_count"`
	ErrorRate       float64       `json:"error_rate"`
	LastCheck       time.Time     `json:"last_check"`
}

// DeploymentMetrics represents deployment metrics
type DeploymentMetrics struct {
	RequestsPerSecond float64       `json:"requests_per_second"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorRate         float64       `json:"error_rate"`
	ConnectionCount   int           `json:"connection_count"`
	Throughput        int64         `json:"throughput"`
}

// NewDeploymentManager creates a new deployment manager
func NewDeploymentManager(finalManager *FinalManager, testManager *TestManager, config *DeploymentConfig) *DeploymentManager {
	if config == nil {
		config = createDefaultDeploymentConfig()
	}

	dm := &DeploymentManager{
		finalManager: finalManager,
		testManager:  testManager,
		config:       config,
		deploymentState: &DeploymentState{
			Status: DeploymentStatusPending,
			Health: &DeploymentHealth{
				Services: make(map[string]*ServiceHealth),
			},
		},
	}

	return dm
}

// createDefaultDeploymentConfig creates default deployment configuration
func createDefaultDeploymentConfig() *DeploymentConfig {
	return &DeploymentConfig{
		Environment: "production",
		Target: &TargetConfig{
			Hosts:       []string{"localhost"},
			Username:    "deploy",
			KeyPath:     "/home/deploy/.ssh/id_rsa",
			DeployPath:  "/opt/socks5-dpi-proxy",
			ServiceName: "socks5-dpi-proxy",
			Port:        1080,
		},
		HealthCheck: &HealthCheckConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
			Retries:  3,
			Endpoints: []string{
				"http://localhost:8080/health",
				"http://localhost:8085/metrics",
			},
		},
		Rollback: &RollbackConfig{
			Enabled:        true,
			AutoRollback:   true,
			Threshold:      0.1, // 10% error rate
			Window:         5 * time.Minute,
			BackupVersions: 3,
		},
		Monitoring: &MonitoringConfig{
			Enabled:         true,
			MetricsPort:     9090,
			LogLevel:        "info",
			AlertingEnabled: true,
		},
		Backup: &BackupConfig{
			Enabled:   true,
			Path:      "/opt/backups/socks5-dpi-proxy",
			Retention: 30 * 24 * time.Hour, // 30 days
			Compress:  true,
		},
	}
}

// Deploy performs a production deployment
func (dm *DeploymentManager) Deploy(ctx context.Context) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.running {
		return ErrDeploymentAlreadyRunning
	}

	dm.running = true
	dm.deploymentState.Status = DeploymentStatusRunning
	dm.deploymentState.StartTime = time.Now()

	defer func() {
		dm.deploymentState.EndTime = time.Now()
		dm.deploymentState.Duration = dm.deploymentState.EndTime.Sub(dm.deploymentState.StartTime)
		dm.running = false
	}()

	// Step 1: Pre-deployment checks
	if err := dm.preDeploymentChecks(ctx); err != nil {
		dm.deploymentState.Status = DeploymentStatusFailed
		dm.deploymentState.Errors = append(dm.deploymentState.Errors, err.Error())
		return fmt.Errorf("pre-deployment checks failed: %w", err)
	}

	// Step 2: Run tests
	if err := dm.runDeploymentTests(ctx); err != nil {
		dm.deploymentState.Status = DeploymentStatusFailed
		dm.deploymentState.Errors = append(dm.deploymentState.Errors, err.Error())
		return fmt.Errorf("deployment tests failed: %w", err)
	}

	// Step 3: Backup current version
	if err := dm.backupCurrentVersion(ctx); err != nil {
		dm.deploymentState.Status = DeploymentStatusFailed
		dm.deploymentState.Errors = append(dm.deploymentState.Errors, err.Error())
		return fmt.Errorf("backup failed: %w", err)
	}

	// Step 4: Deploy new version
	if err := dm.deployNewVersion(ctx); err != nil {
		dm.deploymentState.Status = DeploymentStatusFailed
		dm.deploymentState.Errors = append(dm.deploymentState.Errors, err.Error())
		return fmt.Errorf("deployment failed: %w", err)
	}

	// Step 5: Post-deployment verification
	if err := dm.postDeploymentVerification(ctx); err != nil {
		if dm.config.Rollback.Enabled && dm.config.Rollback.AutoRollback {
			if rollbackErr := dm.rollbackDeployment(ctx); rollbackErr != nil {
				dm.deploymentState.Errors = append(dm.deploymentState.Errors, rollbackErr.Error())
			}
		}
		dm.deploymentState.Status = DeploymentStatusFailed
		dm.deploymentState.Errors = append(dm.deploymentState.Errors, err.Error())
		return fmt.Errorf("post-deployment verification failed: %w", err)
	}

	dm.deploymentState.Status = DeploymentStatusCompleted
	return nil
}

// preDeploymentChecks performs pre-deployment checks
func (dm *DeploymentManager) preDeploymentChecks(ctx context.Context) error {
	// Check if system is ready for deployment
	if !dm.finalManager.IsRunning() {
		return fmt.Errorf("system is not running")
	}

	// Check health status
	health := dm.finalManager.GetHealth()
	for component, healthy := range health {
		if !healthy {
			return fmt.Errorf("component %s is not healthy", component)
		}
	}

	// Check configuration
	config := dm.finalManager.GetConfig()
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Check disk space
	if err := dm.checkDiskSpace(); err != nil {
		return fmt.Errorf("insufficient disk space: %w", err)
	}

	return nil
}

// runDeploymentTests runs deployment tests
func (dm *DeploymentManager) runDeploymentTests(ctx context.Context) error {
	if dm.testManager == nil {
		return fmt.Errorf("test manager is not available")
	}

	// Run critical test suites
	testConfig := &TestConfig{
		EnabledSuites: []string{"unit", "integration"},
		Timeout:       10 * time.Minute,
		ParallelTests: 2,
		Retries:       1,
	}

	// Create temporary test manager with deployment-specific config
	tempTestManager := NewTestManager(dm.finalManager, testConfig)
	if err := tempTestManager.RunTests(ctx); err != nil {
		return fmt.Errorf("deployment tests failed: %w", err)
	}

	// Check test results
	summary := tempTestManager.GetTestSummary()

	if summary.Failed > 0 {
		return fmt.Errorf("%d tests failed", summary.Failed)
	}

	return nil
}

// backupCurrentVersion backs up the current version
func (dm *DeploymentManager) backupCurrentVersion(ctx context.Context) error {
	if !dm.config.Backup.Enabled {
		return nil
	}

	// Create backup directory
	backupPath := fmt.Sprintf("%s/backup-%s", dm.config.Backup.Path, time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Backup configuration
	// In real implementation, this would backup actual files and configuration

	return nil
}

// deployNewVersion deploys the new version
func (dm *DeploymentManager) deployNewVersion(ctx context.Context) error {
	// Stop current system
	if err := dm.finalManager.Stop(); err != nil {
		return fmt.Errorf("failed to stop system: %w", err)
	}

	// Update configuration if needed
	// In real implementation, this would update configuration files

	// Start system with new configuration
	if err := dm.finalManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start system: %w", err)
	}

	return nil
}

// postDeploymentVerification performs post-deployment verification
func (dm *DeploymentManager) postDeploymentVerification(ctx context.Context) error {
	// Wait for system to stabilize
	time.Sleep(10 * time.Second)

	// Check system health
	health := dm.finalManager.GetHealth()
	for component, healthy := range health {
		if !healthy {
			return fmt.Errorf("component %s is not healthy after deployment", component)
		}
	}

	// Perform health checks
	if dm.config.HealthCheck.Enabled {
		if err := dm.performHealthChecks(ctx); err != nil {
			return fmt.Errorf("health checks failed: %w", err)
		}
	}

	// Verify metrics
	if err := dm.verifyMetrics(ctx); err != nil {
		return fmt.Errorf("metrics verification failed: %w", err)
	}

	return nil
}

// performHealthChecks performs health checks
func (dm *DeploymentManager) performHealthChecks(ctx context.Context) error {
	for range dm.config.HealthCheck.Endpoints {
		// In real implementation, this would perform actual HTTP health checks
		// For now, simulate successful health checks
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// verifyMetrics verifies deployment metrics
func (dm *DeploymentManager) verifyMetrics(ctx context.Context) error {
	// Get system metrics
	metrics := dm.finalManager.GetMetrics()
	if metrics == nil {
		return fmt.Errorf("failed to get system metrics")
	}

	// Verify key metrics
	if metrics.Connections == nil {
		return fmt.Errorf("connection metrics are not available")
	}

	// Check error rates
	if metrics.Connections.ErrorConnections > metrics.Connections.TotalConnections/10 {
		return fmt.Errorf("high error rate detected")
	}

	return nil
}

// rollbackDeployment performs rollback
func (dm *DeploymentManager) rollbackDeployment(ctx context.Context) error {
	dm.deploymentState.Status = DeploymentStatusRollingBack

	// Stop current system
	if err := dm.finalManager.Stop(); err != nil {
		return fmt.Errorf("failed to stop system for rollback: %w", err)
	}

	// Restore previous version
	// In real implementation, this would restore from backup

	// Start system with previous configuration
	if err := dm.finalManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start system after rollback: %w", err)
	}

	dm.deploymentState.Status = DeploymentStatusRolledBack
	return nil
}

// checkDiskSpace checks available disk space
func (dm *DeploymentManager) checkDiskSpace() error {
	// In real implementation, this would check actual disk space
	// For now, assume sufficient space
	return nil
}

// GetDeploymentState returns current deployment state
func (dm *DeploymentManager) GetDeploymentState() *DeploymentState {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.deploymentState
}

// IsDeploying returns whether deployment is in progress
func (dm *DeploymentManager) IsDeploying() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.running
}

// UpdateDeploymentHealth updates deployment health status
func (dm *DeploymentManager) UpdateDeploymentHealth(ctx context.Context) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.deploymentState.Health == nil {
		dm.deploymentState.Health = &DeploymentHealth{
			Services: make(map[string]*ServiceHealth),
		}
	}

	dm.deploymentState.Health.LastCheck = time.Now()

	// Update overall status
	health := dm.finalManager.GetHealth()
	allHealthy := true
	for _, healthy := range health {
		if !healthy {
			allHealthy = false
			break
		}
	}

	if allHealthy {
		dm.deploymentState.Health.OverallStatus = "healthy"
	} else {
		dm.deploymentState.Health.OverallStatus = "unhealthy"
	}

	// Update service health
	dm.updateServiceHealth()

	// Update metrics
	dm.updateDeploymentMetrics()

	return nil
}

// updateServiceHealth updates individual service health
func (dm *DeploymentManager) updateServiceHealth() {
	// Update proxy server health
	if dm.finalManager.proxyServer != nil {
		healthy := dm.finalManager.proxyServer.IsHealthy()
		status := "healthy"
		if !healthy {
			status = "unhealthy"
		}

		dm.deploymentState.Health.Services["proxy_server"] = &ServiceHealth{
			Name:            "proxy_server",
			Status:          status,
			Uptime:          time.Since(dm.deploymentState.StartTime),
			CPUUsage:        0.0, // Would get actual CPU usage
			MemoryUsage:     0.0, // Would get actual memory usage
			ConnectionCount: 0,   // Would get actual connection count
			ErrorRate:       0.0, // Would get actual error rate
			LastCheck:       time.Now(),
		}
	}

	// Update other services
	// This would update health for all other services
}

// updateDeploymentMetrics updates deployment metrics
func (dm *DeploymentManager) updateDeploymentMetrics() {
	metrics := dm.finalManager.GetMetrics()
	if metrics == nil {
		return
	}

	dm.deploymentState.Health.Metrics = &DeploymentMetrics{
		RequestsPerSecond: 0.0, // Would calculate actual RPS
		AverageLatency:    0,   // Would get actual latency
		ErrorRate:         0.0, // Would calculate actual error rate
		ConnectionCount:   0,   // Would get actual connection count
		Throughput:        0,   // Would get actual throughput
	}

	if metrics.Connections != nil {
		dm.deploymentState.Health.Metrics.ConnectionCount = metrics.Connections.ActiveConnections
	}
}

// Errors
var (
	ErrDeploymentAlreadyRunning = &DeploymentError{Code: "ALREADY_RUNNING", Message: "deployment is already running"}
)

// DeploymentError represents a deployment error
type DeploymentError struct {
	Code    string
	Message string
}

func (e *DeploymentError) Error() string {
	return e.Message + " (" + e.Code + ")"
}
