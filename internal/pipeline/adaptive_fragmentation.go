package pipeline

import (
	"math"
	"sync"
	"time"
)

type AdaptiveFragmentationModifier struct {
	MinSize        int
	MaxSize        int
	RandomSizes    bool
	MLOptimization bool
	Strategy       string
	rand           *SafeRandom
	lastAdaptation time.Time
	config         *FragmentationConfig
	metrics        *FragmentationMetrics
	mu             sync.RWMutex
	mlEngine       interface{} // Will be typed as MLPipelineEngine when integrated
}

// NewAdaptiveFragmentationModifier creates new adaptive fragmentation modifier
func NewAdaptiveFragmentationModifier(minSize, maxSize int, randomSizes bool) *AdaptiveFragmentationModifier {
	config := DefaultFragmentationConfig()
	config.MinSize = minSize
	config.MaxSize = maxSize
	config.RandomSizes = randomSizes

	return &AdaptiveFragmentationModifier{
		MinSize:        minSize,
		MaxSize:        maxSize,
		RandomSizes:    randomSizes,
		MLOptimization: config.MLOptimization,
		Strategy:       config.AdaptiveStrategy,
		rand:           NewSafeRandom(),
		lastAdaptation: time.Now(),
		config:         config,
		metrics:        NewFragmentationMetrics(),
	}
}

func (a *AdaptiveFragmentationModifier) Name() string {
	return "adaptive_fragmentation"
}

func (a *AdaptiveFragmentationModifier) Configure(config map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create new config if not exists
	if a.config == nil {
		a.config = DefaultFragmentationConfig()
	}

	// Parse configuration
	if minSize, ok := config["min_size"].(int); ok {
		a.config.MinSize = minSize
		a.MinSize = minSize
	}

	if maxSize, ok := config["max_size"].(int); ok {
		a.config.MaxSize = maxSize
		a.MaxSize = maxSize
	}

	if randomSizes, ok := config["random"].(bool); ok {
		a.config.RandomSizes = randomSizes
		a.RandomSizes = randomSizes
	}

	if mlOptimization, ok := config["ml_optimization"].(bool); ok {
		a.config.MLOptimization = mlOptimization
		a.MLOptimization = mlOptimization
	}

	if strategy, ok := config["strategy"].(string); ok {
		a.config.AdaptiveStrategy = strategy
		a.Strategy = strategy
	}

	if antiNagleDelay, ok := config["anti_nagle_delay"].(int); ok {
		a.config.AntiNagleDelay = time.Duration(antiNagleDelay) * time.Millisecond
	}

	// Validate configuration
	if err := ValidateFragmentationConfig(a.config); err != nil {
		return err
	}

	// Re-initialize random generator
	a.rand = NewSafeRandom()
	a.lastAdaptation = time.Now()

	return nil
}

func (a *AdaptiveFragmentationModifier) Process(data []byte, direction Direction) []byte {
	// Process() should NOT reassemble chunks for fragmenting modifiers
	// This prevents fragmentation from being undone at the network level

	// For non-fragmenting cases, return data as-is
	if len(data) <= a.MinSize || direction == DirectionInbound {
		return data
	}

	// For fragmenting modifiers, Process() should be a no-op
	// Real fragmentation happens via ProcessToChunks() in ModifiedConnectionV2
	return data
}

// ProcessToChunks implements ModifierV2 interface
func (a *AdaptiveFragmentationModifier) ProcessToChunks(data []byte, direction Direction) [][]byte {
	start := time.Now()
	defer func() {
		processingTime := time.Since(start)
		a.metrics.RecordFragmentation(len(data), len(data), processingTime, true)
	}()

	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(data) <= a.MinSize {
		return [][]byte{data}
	}

	// For inbound traffic, we don't fragment
	if direction == DirectionInbound {
		return [][]byte{data}
	}

	// Use ML optimization if enabled
	if a.MLOptimization && a.mlEngine != nil {
		return a.mlFragmentation(data)
	}

	return a.adaptiveFragmentation(data)
}

// SupportsFragmentation implements ModifierV2 interface
func (a *AdaptiveFragmentationModifier) SupportsFragmentation() bool {
	return true
}

// GetFragmentationConfig implements FragmentingModifier interface
func (a *AdaptiveFragmentationModifier) GetFragmentationConfig() *FragmentationConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy to prevent external modification
	configCopy := *a.config
	return &configCopy
}

// SetFragmentationConfig implements FragmentingModifier interface
func (a *AdaptiveFragmentationModifier) SetFragmentationConfig(config *FragmentationConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.config = config
	a.MinSize = config.MinSize
	a.MaxSize = config.MaxSize
	a.RandomSizes = config.RandomSizes
	a.MLOptimization = config.MLOptimization
	a.Strategy = config.AdaptiveStrategy
}

// GetMetrics returns fragmentation performance metrics
func (a *AdaptiveFragmentationModifier) GetMetrics() (uint64, float64, time.Duration, float64) {
	return a.metrics.GetMetrics()
}

// adaptiveFragmentation performs adaptive fragmentation with corrected logic
func (a *AdaptiveFragmentationModifier) adaptiveFragmentation(data []byte) [][]byte {
	baseSize := len(data) / 3 // Default to 3 fragments

	if a.RandomSizes {
		// Randomize around base size
		variation := baseSize / 4
		chunks := make([][]byte, 0, 3)
		offset := 0

		for i := 0; i < 3 && offset < len(data); i++ {
			size := baseSize + a.rand.Intn(variation*2) - variation

			// Clamp to min/max bounds
			size = int(math.Max(float64(a.MinSize), math.Min(float64(size), float64(a.MaxSize))))

			// Adjust for remaining data
			if offset+size > len(data) {
				size = len(data) - offset
			}

			if size <= 0 {
				break
			}

			// Create independent chunk
			chunk := make([]byte, size)
			copy(chunk, data[offset:offset+size])
			chunks = append(chunks, chunk)
			offset += size
		}

		return chunks
	}

	// Fixed size fragmentation with corrected logic
	chunks := make([][]byte, 0, 3)
	offset := 0

	for i := 0; i < 3 && offset < len(data); i++ {
		var size int
		if i < 2 {
			size = a.MinSize
		} else {
			// Last chunk gets remaining data
			size = len(data) - offset
		}

		if size <= 0 {
			break
		}

		// Create independent chunk with bounds checking
		end := offset + size
		if end > len(data) {
			end = len(data)
			size = end - offset
		}

		if size <= 0 {
			break
		}

		chunk := make([]byte, size)
		copy(chunk, data[offset:end])
		chunks = append(chunks, chunk)
		offset += size
	}

	return chunks
}

// mlFragmentation performs ML-based fragmentation (placeholder for future ML integration)
func (a *AdaptiveFragmentationModifier) mlFragmentation(data []byte) [][]byte {
	// TODO: Integrate with MLPipelineEngine
	// For now, fall back to adaptive fragmentation
	return a.adaptiveFragmentation(data)
}
