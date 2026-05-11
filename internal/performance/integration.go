package performance

import (
	"context"
	"sync"
	"time"
)

// PerformanceManager integrates all performance components
type PerformanceManager struct {
	connectionPool   *ConnectionPool
	loadBalancer     *LoadBalancer
	optimizer        *PerformanceOptimizer
	monitor          *PerformanceMonitor
	config           *PerformanceConfig
	mu               sync.RWMutex
	running          bool
	stopCh           chan struct{}
}

// PerformanceConfig defines the complete performance configuration
type PerformanceConfig struct {
	ConnectionPool   *PoolConfig         `yaml:"connection_pool"`
	LoadBalancer     *BalancerConfig     `yaml:"load_balancer"`
	Optimizer        *OptimizerConfig    `yaml:"optimizer"`
	Monitor          *MonitoringConfig   `yaml:"monitor"`
	Integration      *IntegrationConfig  `yaml:"integration"`
}

// IntegrationConfig defines integration settings
type IntegrationConfig struct {
	AutoStart         bool          `yaml:"auto_start"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	MetricsAggregation  bool          `yaml:"metrics_aggregation"`
	CrossComponentOptimization bool `yaml:"cross_component_optimization"`
}

// PerformanceStats represents aggregated performance statistics
type PerformanceStats struct {
	ConnectionPool   *PoolStats           `json:"connection_pool"`
	LoadBalancer     *BalancerStats       `json:"load_balancer"`
	Optimizer        *PerformanceMetrics  `json:"optimizer"`
	Monitor          *PerformanceMetrics  `json:"monitor"`
	System           *SystemStats         `json:"system"`
	Timestamp        time.Time            `json:"timestamp"`
}

// SystemStats represents system-level statistics
type SystemStats struct {
	MemoryUsage      int64   `json:"memory_usage"`
	CPUUsage         float64 `json:"cpu_usage"`
	Goroutines       int     `json:"goroutines"`
	Uptime           time.Duration `json:"uptime"`
	RequestsPerSecond float64 `json:"requests_per_second"`
	ErrorRate        float64 `json:"error_rate"`
}

// NewPerformanceManager creates a new performance manager
func NewPerformanceManager(config *PerformanceConfig) *PerformanceManager {
	if config == nil {
		config = &PerformanceConfig{
			ConnectionPool: &PoolConfig{
				MaxConnections:     100,
				MaxIdleTime:        30 * time.Second,
				MaxLifetime:        5 * time.Minute,
				HealthCheckInterval: 10 * time.Second,
				EnableMetrics:      true,
			},
			LoadBalancer: &BalancerConfig{
				Strategy:            "round_robin",
				HealthCheckInterval: 30 * time.Second,
				UnhealthyThreshold: 3,
				HealthyThreshold:   2,
				EnableStickySessions: false,
				SessionTimeout:     5 * time.Minute,
				RetryAttempts:      3,
			},
			Optimizer: &OptimizerConfig{
				MemoryConfig: &MemoryConfig{
					MaxMemoryMB:     512,
					BufferPoolSize:  1000,
					CacheSize:       10000,
					GCTargetPercent:  50,
				},
				CPUConfig: &CPUConfig{
					WorkerCount:    4,
					EnableAffinity: true,
					MaxCPUUsage:    80.0,
				},
				IOConfig: &IOConfig{
					BufferSize:      64 * 1024,
					AsyncWorkers:    4,
					MaxConcurrentIO: 100,
				},
				GCConfig: &GCConfig{
					GOGC:            100,
					GOMEMLIMIT:      512 * 1024 * 1024,
					ForceGCInterval: 30 * time.Second,
				},
				MetricsInterval: 10 * time.Second,
				EnableAutoTuning: true,
			},
			Monitor: &MonitoringConfig{
				Enabled:          true,
				MetricsInterval:  10 * time.Second,
				AlertingEnabled:  true,
				DashboardEnabled: true,
				DashboardPort:    8080,
				PrometheusEnabled: true,
				PrometheusPort:   9090,
			},
			Integration: &IntegrationConfig{
				AutoStart:         true,
				HealthCheckInterval: 30 * time.Second,
				MetricsAggregation:  true,
				CrossComponentOptimization: true,
			},
		}
	}

	pm := &PerformanceManager{
		config: config,
		stopCh: make(chan struct{}),
	}

	return pm
}

// Start starts all performance components
func (pm *PerformanceManager) Start(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.running {
		return ErrPerformanceManagerAlreadyRunning
	}

	// Initialize connection pool
	tcpFactory := NewTCPConnectionFactory("localhost:8080", &TCPFactoryConfig{
		Timeout:        10 * time.Second,
		KeepAlive:      30 * time.Second,
		EnableTLS:      false,
		ValidationMode: ValidationBasic,
	})
	pm.connectionPool = NewConnectionPool(tcpFactory, pm.config.ConnectionPool.MaxConnections, pm.config.ConnectionPool)

	// Initialize load balancer
	endpoints := []string{"localhost:8080", "localhost:8081", "localhost:8082"}
	pm.loadBalancer = NewLoadBalancer(endpoints, pm.config.LoadBalancer)

	// Initialize optimizer
	pm.optimizer = NewPerformanceOptimizer(pm.config.Optimizer)

	// Initialize monitor
	pm.monitor = NewPerformanceMonitor(pm.config.Monitor)

	// Start all components
	if err := pm.startComponents(ctx); err != nil {
		return err
	}

	pm.running = true

	// Start integration services
	go pm.integrationLoop(ctx)

	return nil
}

// startComponents starts all performance components
func (pm *PerformanceManager) startComponents(ctx context.Context) error {
	// Start optimizer
	if err := pm.optimizer.Start(); err != nil {
		return err
	}

	// Start monitor
	if err := pm.monitor.Start(); err != nil {
		return err
	}

	return nil
}

// Stop stops all performance components
func (pm *PerformanceManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return ErrPerformanceManagerNotRunning
	}

	pm.running = false
	close(pm.stopCh)

	// Stop components
	if pm.optimizer != nil {
		pm.optimizer.Stop()
	}

	if pm.monitor != nil {
		pm.monitor.Stop()
	}

	if pm.connectionPool != nil {
		pm.connectionPool.Close()
	}

	if pm.loadBalancer != nil {
		pm.loadBalancer.Close()
	}

	return nil
}

// integrationLoop runs the integration loop
func (pm *PerformanceManager) integrationLoop(ctx context.Context) {
	ticker := time.NewTicker(pm.config.Integration.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.performIntegration()
		case <-pm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// performIntegration performs cross-component integration
func (pm *PerformanceManager) performIntegration() {
	if !pm.config.Integration.CrossComponentOptimization {
		return
	}

	// Get metrics from all components
	optimizerMetrics := pm.optimizer.GetMetrics()
	monitorMetrics := pm.monitor.GetMetrics()

	// Optimize based on cross-component metrics
	pm.optimizeCrossComponent(optimizerMetrics, monitorMetrics)
}

// optimizeCrossComponent performs cross-component optimization
func (pm *PerformanceManager) optimizeCrossComponent(optimizerMetrics, monitorMetrics *PerformanceMetrics) {
	// Memory optimization
	if optimizerMetrics.MemoryUsage > 500*1024*1024 { // 500MB
		pm.optimizeMemoryUsage()
	}

	// CPU optimization
	if optimizerMetrics.CPUUsage > 80.0 {
		pm.optimizeCPUUsage()
	}

	// Connection pool optimization
	if pm.connectionPool != nil {
		poolStats := pm.connectionPool.GetStats()
		if poolStats.HitRate < 0.5 {
			pm.optimizeConnectionPool()
		}
	}

	// Load balancer optimization
	if pm.loadBalancer != nil {
		lbStats := pm.loadBalancer.GetStats()
		if lbStats.FailedRequests > lbStats.SuccessfulRequests/10 { // >10% failure rate
			pm.optimizeLoadBalancer()
		}
	}
}

// optimizeMemoryUsage optimizes memory usage
func (pm *PerformanceManager) optimizeMemoryUsage() {
	// Force garbage collection
	// This would trigger the optimizer's memory optimization
}

// optimizeCPUUsage optimizes CPU usage
func (pm *PerformanceManager) optimizeCPUUsage() {
	// Reduce worker count or optimize CPU affinity
	// This would trigger the optimizer's CPU optimization
}

// optimizeConnectionPool optimizes connection pool
func (pm *PerformanceManager) optimizeConnectionPool() {
	// Adjust pool size or timeout settings
	// This would optimize the connection pool parameters
}

// optimizeLoadBalancer optimizes load balancer
func (pm *PerformanceManager) optimizeLoadBalancer() {
	// Change strategy or remove unhealthy endpoints
	// This would optimize the load balancer configuration
}

// GetConnection gets a connection from the pool
func (pm *PerformanceManager) GetConnection() (*PooledConnection, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.connectionPool == nil {
		return nil, ErrConnectionPoolNotInitialized
	}

	return pm.connectionPool.Get()
}

// GetEndpoint gets an endpoint from the load balancer
func (pm *PerformanceManager) GetEndpoint(key string) (*Endpoint, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.loadBalancer == nil {
		return nil, ErrLoadBalancerNotInitialized
	}

	return pm.loadBalancer.GetEndpoint(key)
}

// GetStats returns aggregated performance statistics
func (pm *PerformanceManager) GetStats() *PerformanceStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := &PerformanceStats{
		Timestamp: time.Now(),
		System: &SystemStats{
			Goroutines:       0, // Would get from runtime
			Uptime:          0, // Would calculate from start time
			RequestsPerSecond: 0,
			ErrorRate:        0,
		},
	}

	// Get connection pool stats
	if pm.connectionPool != nil {
		stats.ConnectionPool = pm.connectionPool.GetStats()
	}

	// Get load balancer stats
	if pm.loadBalancer != nil {
		stats.LoadBalancer = pm.loadBalancer.GetStats()
	}

	// Get optimizer metrics
	if pm.optimizer != nil {
		stats.Optimizer = pm.optimizer.GetMetrics()
	}

	// Get monitor metrics
	if pm.monitor != nil {
		stats.Monitor = pm.monitor.GetMetrics()
	}

	return stats
}

// UpdateConfig updates the performance configuration
func (pm *PerformanceManager) UpdateConfig(config *PerformanceConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.config = config

	// Update individual component configurations
	if pm.connectionPool != nil {
		// Update connection pool config
	}

	if pm.loadBalancer != nil {
		// Update load balancer config
		pm.loadBalancer.SetStrategy(config.LoadBalancer.Strategy)
	}

	if pm.optimizer != nil {
		// Update optimizer config
	}

	if pm.monitor != nil {
		// Update monitor config
	}

	return nil
}

// HealthCheck performs a comprehensive health check
func (pm *PerformanceManager) HealthCheck() map[string]bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	health := make(map[string]bool)

	// Check connection pool health
	if pm.connectionPool != nil {
		poolStats := pm.connectionPool.GetStats()
		health["connection_pool"] = poolStats.Errors < poolStats.TotalRequests/10
	}

	// Check load balancer health
	if pm.loadBalancer != nil {
		endpoints := pm.loadBalancer.GetEndpoints()
		healthyCount := 0
		for _, endpoint := range endpoints {
			if endpoint.Healthy {
				healthyCount++
			}
		}
		health["load_balancer"] = healthyCount > 0
	}

	// Check optimizer health
	if pm.optimizer != nil {
		metrics := pm.optimizer.GetMetrics()
		health["optimizer"] = metrics.MemoryUsage < 1024*1024*1024 // < 1GB
	}

	// Check monitor health
	if pm.monitor != nil {
		health["monitor"] = true // Monitor is always healthy if running
	}

	return health
}

// IsRunning returns whether the performance manager is running
func (pm *PerformanceManager) IsRunning() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.running
}

// GetConfig returns the current configuration
func (pm *PerformanceManager) GetConfig() *PerformanceConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.config
}

// Errors
var (
	ErrPerformanceManagerAlreadyRunning = &PerformanceManagerError{Code: "ALREADY_RUNNING", Message: "performance manager is already running"}
	ErrPerformanceManagerNotRunning     = &PerformanceManagerError{Code: "NOT_RUNNING", Message: "performance manager is not running"}
	ErrConnectionPoolNotInitialized     = &PerformanceManagerError{Code: "POOL_NOT_INITIALIZED", Message: "connection pool not initialized"}
	ErrLoadBalancerNotInitialized      = &PerformanceManagerError{Code: "LB_NOT_INITIALIZED", Message: "load balancer not initialized"}
)

// PerformanceManagerError represents a performance manager error
type PerformanceManagerError struct {
	Code    string
	Message string
}

func (e *PerformanceManagerError) Error() string {
	return e.Message + " (" + e.Code + ")"
}
