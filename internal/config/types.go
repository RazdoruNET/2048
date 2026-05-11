package config

import (
	"os"
	"sync"
	"time"
)

// FileWatcher interface for file system monitoring
type FileWatcher interface {
	OnChange(handler func())
	Close() error
}

// SimpleFileWatcher implements basic file watching
type SimpleFileWatcher struct {
	filePath string
	lastMod  time.Time
	mu       sync.RWMutex
	handler  func()
	stopCh   chan struct{}
}

// NewFileWatcher creates new file watcher
func NewFileWatcher(filePath string) (FileWatcher, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	return &SimpleFileWatcher{
		filePath: filePath,
		lastMod:  info.ModTime(),
		stopCh:   make(chan struct{}),
	}, nil
}

// OnChange sets change handler
func (fw *SimpleFileWatcher) OnChange(handler func()) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.handler = handler
	go fw.watchLoop()
}

// Close closes file watcher
func (fw *SimpleFileWatcher) Close() error {
	close(fw.stopCh)
	return nil
}

// watchLoop monitors file changes
func (fw *SimpleFileWatcher) watchLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-fw.stopCh:
			return
		case <-ticker.C:
			if fw.hasChanged() {
				fw.mu.RLock()
				if fw.handler != nil {
					fw.handler()
				}
				fw.mu.RUnlock()
			}
		}
	}
}

// hasChanged checks if file has been modified
func (fw *SimpleFileWatcher) hasChanged() bool {
	info, err := os.Stat(fw.filePath)
	if err != nil {
		return false
	}

	if info.ModTime().After(fw.lastMod) {
		fw.lastMod = info.ModTime()
		return true
	}

	return false
}

// Configuration type definitions

// DPIDetectionConfig represents DPI detection settings
type DPIDetectionConfig struct {
	Enabled        bool             `yaml:"enabled"`
	SignatureBased bool             `yaml:"signature_based"`
	Behavioral     bool             `yaml:"behavioral"`
	MLBased        bool             `yaml:"ml_based"`
	Thresholds     ThresholdsConfig `yaml:"thresholds"`
}

// ThresholdsConfig represents detection thresholds
type ThresholdsConfig struct {
	Signature  float64 `yaml:"signature"`
	Behavioral float64 `yaml:"behavioral"`
	MLBased    float64 `yaml:"ml_based"`
}

// FeatureExtractionConfig represents feature extraction settings
type FeatureExtractionConfig struct {
	Enabled        bool `yaml:"enabled"`
	PacketFeatures bool `yaml:"packet_features"`
	FlowFeatures   bool `yaml:"flow_features"`
	TimingFeatures bool `yaml:"timing_features"`
	HTTPFeatures   bool `yaml:"http_features"`
	TLSFeatures    bool `yaml:"tls_features"`
}

// TechniqueCacheConfig represents technique cache settings
type TechniqueCacheConfig struct {
	Enabled bool   `yaml:"enabled"`
	MaxSize int    `yaml:"max_size"`
	TTL     string `yaml:"ttl"`
}

// RLConfig represents reinforcement learning settings
type RLConfig struct {
	Enabled          bool    `yaml:"enabled"`
	ExplorationRate  float64 `yaml:"exploration_rate"`
	DiscountFactor   float64 `yaml:"discount_factor"`
	LearningEpisodes int     `yaml:"learning_episodes"`
}

// RateLimitingConfig represents rate limiting settings
type RateLimitingConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerSecond int  `yaml:"requests_per_second"`
	BurstSize         int  `yaml:"burst_size"`
}

// IPFilteringConfig represents IP filtering settings
type IPFilteringConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Whitelist []string `yaml:"whitelist"`
	Blacklist []string `yaml:"blacklist"`
}

// AntiTamperingConfig represents anti-tampering settings
type AntiTamperingConfig struct {
	Enabled        bool   `yaml:"enabled"`
	IntegrityCheck string `yaml:"integrity_check"`
}

// PrivacyConfig represents privacy settings
type PrivacyConfig struct {
	Anonymization    bool `yaml:"anonymization"`
	DataMinimization bool `yaml:"data_minimization"`
	LogEncryption    bool `yaml:"log_encryption"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Interval string   `yaml:"interval"`
	Include  []string `yaml:"include"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Levels   []string       `yaml:"levels"`
	Format   string         `yaml:"format"`
	Rotation RotationConfig `yaml:"rotation"`
}

// RotationConfig represents log rotation settings
type RotationConfig struct {
	MaxSize  string `yaml:"max_size"`
	MaxFiles int    `yaml:"max_files"`
	Compress bool   `yaml:"compress"`
}

// HealthConfig represents health check configuration
type HealthConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
	Retries  int    `yaml:"retries"`
}

// ConnectionPoolConfig represents connection pool settings
type ConnectionPoolConfig struct {
	MaxConnections  int    `yaml:"max_connections"`
	IdleTimeout     string `yaml:"idle_timeout"`
	ReadBufferSize  int    `yaml:"read_buffer_size"`
	WriteBufferSize int    `yaml:"write_buffer_size"`
}

// MLPerformanceConfig represents ML performance settings
type MLPerformanceConfig struct {
	CacheSize          string `yaml:"cache_size"`
	BatchSize          int    `yaml:"batch_size"`
	ParallelProcessing bool   `yaml:"parallel_processing"`
	ModelQuantization  bool   `yaml:"model_quantization"`
}

// QuantumResistantConfig represents quantum-resistant settings
type QuantumResistantConfig struct {
	Enabled       bool     `yaml:"enabled"`
	Algorithms    []string `yaml:"algorithms"`
	KeySizes      []int    `yaml:"key_sizes"`
	SecurityLevel string   `yaml:"security_level"`
}
