package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// DynamicConfig manages dynamic configuration with hot-reload
type DynamicConfig struct {
	mu             sync.RWMutex
	configPath     string
	config         *Config
	lastModified   time.Time
	watcher        FileWatcher
	changeHandlers []ConfigChangeHandler
}

// Config represents main configuration structure
type Config struct {
	Listen      string            `yaml:"listen"`
	LogLevel    string            `yaml:"log_level"`
	MLEnabled   bool              `yaml:"ml_enabled"`
	Rules       []Rule            `yaml:"rules"`
	MLConfig    MLConfig          `yaml:"ml_engine"`
	Security    SecurityConfig    `yaml:"security"`
	Monitoring  MonitoringConfig  `yaml:"monitoring"`
	Performance PerformanceConfig `yaml:"performance"`
}

// Rule represents pipeline rule
type Rule struct {
	Domain  string                 `yaml:"domain"`
	IPRange string                 `yaml:"ip_range"`
	Port    uint16                 `yaml:"port"`
	Config  map[string]interface{} `yaml:"pipeline"`
}

// MLConfig represents ML engine configuration
type MLConfig struct {
	Enabled                bool                    `yaml:"enabled"`
	LearningRate           float64                 `yaml:"learning_rate"`
	ConfidenceThreshold    float64                 `yaml:"confidence_threshold"`
	ModelUpdateInterval    string                  `yaml:"model_update_interval"`
	EffectivenessThreshold float64                 `yaml:"effectiveness_threshold"`
	DPIDetection           DPIDetectionConfig      `yaml:"dpi_detection"`
	FeatureExtraction      FeatureExtractionConfig `yaml:"feature_extraction"`
	TechniqueCache         TechniqueCacheConfig    `yaml:"technique_cache"`
	ReinforcementLearning  RLConfig                `yaml:"reinforcement_learning"`
}

// SecurityConfig represents security settings
type SecurityConfig struct {
	RateLimiting  RateLimitingConfig  `yaml:"rate_limiting"`
	IPFiltering   IPFilteringConfig   `yaml:"ip_filtering"`
	AntiTampering AntiTamperingConfig `yaml:"anti_tampering"`
	Privacy       PrivacyConfig       `yaml:"privacy"`
}

// MonitoringConfig represents monitoring settings
type MonitoringConfig struct {
	Metrics      MetricsConfig `yaml:"metrics"`
	Logging      LoggingConfig `yaml:"logging"`
	HealthChecks HealthConfig  `yaml:"health_checks"`
}

// PerformanceConfig represents performance settings
type PerformanceConfig struct {
	GoMaxprocs     int                  `yaml:"gomaxprocs"`
	GCPercent      int                  `yaml:"gc_percent"`
	ConnectionPool ConnectionPoolConfig `yaml:"connection_pool"`
	ML             MLPerformanceConfig  `yaml:"ml"`
}

// ConfigChangeHandler handles configuration changes
type ConfigChangeHandler interface {
	OnConfigChange(oldConfig, newConfig *Config) error
}

// NewDynamicConfig creates new dynamic configuration manager
func NewDynamicConfig(configPath string) (*DynamicConfig, error) {
	dc := &DynamicConfig{
		configPath:     configPath,
		config:         &Config{},
		changeHandlers: make([]ConfigChangeHandler, 0),
	}

	// Load initial configuration
	if err := dc.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	// Setup file watcher
	watcher, err := NewFileWatcher(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to setup file watcher: %w", err)
	}

	dc.watcher = watcher
	watcher.OnChange(dc.onFileChange)

	return dc, nil
}

// GetConfig returns current configuration
func (dc *DynamicConfig) GetConfig() *Config {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	// Return copy to prevent external modifications
	configCopy := *dc.config
	return &configCopy
}

// UpdateConfig updates configuration and saves to file
func (dc *DynamicConfig) UpdateConfig(newConfig *Config) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	oldConfig := dc.config
	dc.config = newConfig

	// Save to file
	if err := dc.saveConfig(); err != nil {
		dc.config = oldConfig // Rollback on error
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Notify handlers
	for _, handler := range dc.changeHandlers {
		if err := handler.OnConfigChange(oldConfig, newConfig); err != nil {
			log.Printf("Config change handler error: %v", err)
		}
	}

	return nil
}

// AddChangeHandler adds a configuration change handler
func (dc *DynamicConfig) AddChangeHandler(handler ConfigChangeHandler) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.changeHandlers = append(dc.changeHandlers, handler)
}

// loadConfig loads configuration from file
func (dc *DynamicConfig) loadConfig() error {
	data, err := os.ReadFile(dc.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	dc.applyDefaults(&config)

	// Get file modification time
	if info, err := os.Stat(dc.configPath); err == nil {
		dc.lastModified = info.ModTime()
	}

	dc.config = &config
	return nil
}

// saveConfig saves configuration to file
func (dc *DynamicConfig) saveConfig() error {
	// Ensure directory exists
	dir := filepath.Dir(dc.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(dc.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(dc.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// onFileChange handles file change events
func (dc *DynamicConfig) onFileChange() {
	// Reload configuration
	if err := dc.loadConfig(); err != nil {
		log.Printf("Failed to reload config: %v", err)
		return
	}

	log.Printf("Configuration reloaded from %s", dc.configPath)
}

// applyDefaults applies default values to configuration
func (dc *DynamicConfig) applyDefaults(config *Config) {
	if config.Listen == "" {
		config.Listen = ":1080"
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}

	// Apply ML defaults
	dc.applyMLDefaults(&config.MLConfig)

	// Apply security defaults
	dc.applySecurityDefaults(&config.Security)

	// Apply monitoring defaults
	dc.applyMonitoringDefaults(&config.Monitoring)

	// Apply performance defaults
	dc.applyPerformanceDefaults(&config.Performance)
}

// applyMLDefaults applies default ML configuration
func (dc *DynamicConfig) applyMLDefaults(mlConfig *MLConfig) {
	if mlConfig.LearningRate == 0 {
		mlConfig.LearningRate = 0.1
	}
	if mlConfig.ConfidenceThreshold == 0 {
		mlConfig.ConfidenceThreshold = 0.85
	}
	if mlConfig.ModelUpdateInterval == "" {
		mlConfig.ModelUpdateInterval = "30m"
	}
	if mlConfig.EffectivenessThreshold == 0 {
		mlConfig.EffectivenessThreshold = 0.7
	}
}

// applySecurityDefaults applies default security configuration
func (dc *DynamicConfig) applySecurityDefaults(security *SecurityConfig) {
	if security.RateLimiting.RequestsPerSecond == 0 {
		security.RateLimiting.RequestsPerSecond = 100
	}
	if security.RateLimiting.BurstSize == 0 {
		security.RateLimiting.BurstSize = 1000
	}
}

// applyMonitoringDefaults applies default monitoring configuration
func (dc *DynamicConfig) applyMonitoringDefaults(monitoring *MonitoringConfig) {
	if monitoring.Metrics.Interval == "" {
		monitoring.Metrics.Interval = "30s"
	}
	if monitoring.HealthChecks.Interval == "" {
		monitoring.HealthChecks.Interval = "30s"
	}
	if monitoring.HealthChecks.Timeout == "" {
		monitoring.HealthChecks.Timeout = "5s"
	}
	if monitoring.HealthChecks.Retries == 0 {
		monitoring.HealthChecks.Retries = 3
	}
}

// applyPerformanceDefaults applies default performance configuration
func (dc *DynamicConfig) applyPerformanceDefaults(performance *PerformanceConfig) {
	if performance.GoMaxprocs == 0 {
		performance.GoMaxprocs = 8
	}
	if performance.GCPercent == 0 {
		performance.GCPercent = 20
	}
	if performance.ConnectionPool.MaxConnections == 0 {
		performance.ConnectionPool.MaxConnections = 10000
	}
	if performance.ConnectionPool.IdleTimeout == "" {
		performance.ConnectionPool.IdleTimeout = "30s"
	}
}

// Close closes the dynamic configuration manager
func (dc *DynamicConfig) Close() error {
	if dc.watcher != nil {
		return dc.watcher.Close()
	}
	return nil
}

// Config validation
func (dc *DynamicConfig) ValidateConfig(config *Config) error {
	// Validate basic fields
	if config.Listen == "" {
		return fmt.Errorf("listen address is required")
	}

	// Validate ML config
	if config.MLEnabled {
		if config.MLConfig.LearningRate <= 0 || config.MLConfig.LearningRate > 1 {
			return fmt.Errorf("learning rate must be between 0 and 1")
		}
		if config.MLConfig.ConfidenceThreshold < 0 || config.MLConfig.ConfidenceThreshold > 1 {
			return fmt.Errorf("confidence threshold must be between 0 and 1")
		}
	}

	return nil
}

// ExportConfig exports configuration to JSON format
func (dc *DynamicConfig) ExportConfig() ([]byte, error) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return json.MarshalIndent(dc.config, "", "  ")
}

// ImportConfig imports configuration from JSON format
func (dc *DynamicConfig) ImportConfig(data []byte) error {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse JSON config: %w", err)
	}

	if err := dc.ValidateConfig(&config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return dc.UpdateConfig(&config)
}
