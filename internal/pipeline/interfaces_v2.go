package pipeline

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"
)

// ModifierV2 extends Modifier interface with fragmentation support
type ModifierV2 interface {
	Modifier
	// ProcessToChunks processes data and returns chunks for network-level fragmentation
	ProcessToChunks(data []byte, direction Direction) [][]byte
	// SupportsFragmentation indicates if this modifier can fragment data
	SupportsFragmentation() bool
}

// IMPORTANT: Process() behavior for fragmenting modifiers
//
// For modifiers that support fragmentation (SupportsFragmentation() == true):
// - Process() should be a NO-OP for outbound traffic
// - Process() should return data unchanged for outbound DirectionOutbound
// - Process() can return processed data for inbound DirectionInbound
// - Real fragmentation happens via ProcessToChunks() in ModifiedConnectionV2
//
// For non-fragmenting modifiers (SupportsFragmentation() == false):
// - Process() works normally, processing and returning modified data
// - ProcessToChunks() returns single chunk with processed data
//
// This architecture ensures that fragmenting modifiers create separate TCP packets
// instead of reassembled data that defeats DPI bypass.

// FragmentingModifier is a specialized interface for modifiers that implement fragmentation
type FragmentingModifier interface {
	ModifierV2
	// GetFragmentationConfig returns current fragmentation configuration
	GetFragmentationConfig() *FragmentationConfig
	// SetFragmentationConfig updates fragmentation configuration
	SetFragmentationConfig(config *FragmentationConfig)
}

// FragmentationConfig contains configuration for fragmentation behavior
type FragmentationConfig struct {
	MinSize          int           `yaml:"min_size"`
	MaxSize          int           `yaml:"max_size"`
	RandomSizes      bool          `yaml:"random_sizes"`
	AntiNagleDelay   time.Duration `yaml:"anti_nagle_delay"`
	MaxChunks        int           `yaml:"max_chunks"`
	MLOptimization   bool          `yaml:"ml_optimization"`
	AdaptiveStrategy string        `yaml:"adaptive_strategy"`
}

// ConnectionConfig contains configuration for connection-level behavior
type ConnectionConfig struct {
	AntiNagleDelay  time.Duration `yaml:"anti_nagle_delay"`
	MaxWriteRetries int           `yaml:"max_write_retries"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	EnableMetrics   bool          `yaml:"enable_metrics"`
}

// LegacyModifierWrapper provides backward compatibility for existing modifiers
type LegacyModifierWrapper struct {
	Modifier
	supportsFragmentation bool
}

func NewLegacyModifierWrapper(modifier Modifier) *LegacyModifierWrapper {
	return &LegacyModifierWrapper{
		Modifier:              modifier,
		supportsFragmentation: false,
	}
}

func (lmw *LegacyModifierWrapper) ProcessToChunks(data []byte, direction Direction) [][]byte {
	// Legacy modifiers don't support fragmentation, return single chunk
	processed := lmw.Modifier.Process(data, direction)
	return [][]byte{processed}
}

func (lmw *LegacyModifierWrapper) SupportsFragmentation() bool {
	return lmw.supportsFragmentation
}

// SafeRandom provides thread-safe random number generation using crypto/rand
type SafeRandom struct {
	mu sync.Mutex
}

func NewSafeRandom() *SafeRandom {
	return &SafeRandom{}
}

func (sr *SafeRandom) Intn(n int) int {
	if n <= 0 {
		return 0
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Use crypto/rand for secure random numbers
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to simple hash-based pseudo-random if crypto/rand fails
		timestamp := time.Now().UnixNano()
		return int(timestamp % int64(n))
	}

	// Convert to int and apply modulo
	randomValue := int(binary.BigEndian.Uint64(b) % uint64(n))
	return randomValue
}

func (sr *SafeRandom) Seed(seed int64) {
	// No-op for crypto/rand - it's always seeded from system entropy
}

// FragmentationMetrics tracks fragmentation performance
type FragmentationMetrics struct {
	TotalFragments      uint64        `json:"total_fragments"`
	AverageFragmentSize float64       `json:"average_fragment_size"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
	SuccessRate         float64       `json:"success_rate"`
	mu                  sync.RWMutex
}

func NewFragmentationMetrics() *FragmentationMetrics {
	return &FragmentationMetrics{}
}

func (fm *FragmentationMetrics) RecordFragmentation(chunkCount int, totalSize int, processingTime time.Duration, success bool) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.TotalFragments += uint64(chunkCount)
	fm.TotalProcessingTime += processingTime

	// Update average fragment size correctly
	if fm.TotalFragments == uint64(chunkCount) {
		// First record
		fm.AverageFragmentSize = float64(totalSize) / float64(chunkCount)
	} else {
		// Update running average
		previousTotalSize := fm.AverageFragmentSize * float64(fm.TotalFragments-uint64(chunkCount))
		newTotalSize := previousTotalSize + float64(totalSize)
		fm.AverageFragmentSize = newTotalSize / float64(fm.TotalFragments)
	}

	// Update success rate (simple moving average)
	if success {
		fm.SuccessRate = (fm.SuccessRate*0.9 + 0.1) // 90% retention of old value
	} else {
		fm.SuccessRate = (fm.SuccessRate * 0.9) // 90% retention of old value
	}
}

func (fm *FragmentationMetrics) GetMetrics() (uint64, float64, time.Duration, float64) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	return fm.TotalFragments, fm.AverageFragmentSize, fm.TotalProcessingTime, fm.SuccessRate
}

// ValidateFragmentationConfig validates fragmentation configuration parameters
func ValidateFragmentationConfig(config *FragmentationConfig) error {
	if config.MinSize <= 0 {
		return ErrInvalidMinSize
	}

	if config.MaxSize <= 0 {
		return ErrInvalidMaxSize
	}

	if config.MinSize > config.MaxSize {
		return ErrMinSizeGreaterThanMaxSize
	}

	if config.MaxChunks <= 0 {
		return ErrInvalidMaxChunks
	}

	if config.AntiNagleDelay < 0 {
		return ErrInvalidAntiNagleDelay
	}

	return nil
}

// DefaultFragmentationConfig returns a safe default configuration
func DefaultFragmentationConfig() *FragmentationConfig {
	return &FragmentationConfig{
		MinSize:          64,
		MaxSize:          256,
		RandomSizes:      true,
		AntiNagleDelay:   1 * time.Millisecond,
		MaxChunks:        10,
		MLOptimization:   false,
		AdaptiveStrategy: "simple",
	}
}

// DefaultConnectionConfig returns a safe default connection configuration
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		AntiNagleDelay:  1 * time.Millisecond,
		MaxWriteRetries: 3,
		WriteTimeout:    30 * time.Second,
		EnableMetrics:   true,
	}
}

// Error definitions
var (
	ErrInvalidMinSize            = &PipelineError{Code: "INVALID_MIN_SIZE", Message: "Minimum size must be positive"}
	ErrInvalidMaxSize            = &PipelineError{Code: "INVALID_MAX_SIZE", Message: "Maximum size must be positive"}
	ErrMinSizeGreaterThanMaxSize = &PipelineError{Code: "MIN_SIZE_GT_MAX_SIZE", Message: "Minimum size cannot be greater than maximum size"}
	ErrInvalidMaxChunks          = &PipelineError{Code: "INVALID_MAX_CHUNKS", Message: "Maximum chunks must be positive"}
	ErrInvalidAntiNagleDelay     = &PipelineError{Code: "INVALID_ANTI_NAGLE_DELAY", Message: "Anti-Nagle delay cannot be negative"}
)

// PipelineError represents a pipeline-specific error
type PipelineError struct {
	Code    string
	Message string
}

func (pe *PipelineError) Error() string {
	return pe.Message
}
