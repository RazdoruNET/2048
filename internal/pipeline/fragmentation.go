package pipeline

import (
	"sync"
	"time"
)

type FragmentationModifier struct {
	MinSize int
	MaxSize int
	Random  bool
	rand    *SafeRandom
	config  *FragmentationConfig
	metrics *FragmentationMetrics
	mu      sync.RWMutex
}

func (f *FragmentationModifier) Name() string {
	return "fragmentation"
}

func (f *FragmentationModifier) Configure(config map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Create new config if not exists
	if f.config == nil {
		f.config = DefaultFragmentationConfig()
	}

	// Parse configuration
	if size, ok := config["size"].(int); ok {
		f.config.MinSize = size
		f.config.MaxSize = size
		f.MinSize = size
		f.MaxSize = size
	} else {
		if minSize, ok := config["min_size"].(int); ok {
			f.config.MinSize = minSize
			f.MinSize = minSize
		} else {
			f.config.MinSize = 100
			f.MinSize = 100
		}

		if maxSize, ok := config["max_size"].(int); ok {
			f.config.MaxSize = maxSize
			f.MaxSize = maxSize
		} else {
			f.config.MaxSize = 200
			f.MaxSize = 200
		}
	}

	if random, ok := config["random"].(bool); ok {
		f.config.RandomSizes = random
		f.Random = random
	} else {
		f.config.RandomSizes = true
		f.Random = true
	}

	if antiNagleDelay, ok := config["anti_nagle_delay"].(int); ok {
		f.config.AntiNagleDelay = time.Duration(antiNagleDelay) * time.Millisecond
	}

	// Validate configuration
	if err := ValidateFragmentationConfig(f.config); err != nil {
		return err
	}

	// Initialize safe random generator
	f.rand = NewSafeRandom()
	f.metrics = NewFragmentationMetrics()

	return nil
}

func (f *FragmentationModifier) Process(data []byte, direction Direction) []byte {
	// Process() should NOT reassemble chunks for fragmenting modifiers
	// This prevents fragmentation from being undone at the network level
	
	// For non-fragmenting cases, return data as-is
	if len(data) <= f.MinSize || direction == DirectionInbound {
		return data
	}
	
	// For fragmenting modifiers, Process() should be a no-op
	// Real fragmentation happens via ProcessToChunks() in ModifiedConnectionV2
	return data
}

// ProcessToChunks implements ModifierV2 interface
func (f *FragmentationModifier) ProcessToChunks(data []byte, direction Direction) [][]byte {
	start := time.Now()
	defer func() {
		processingTime := time.Since(start)
		f.metrics.RecordFragmentation(len(data), len(data), processingTime, true)
	}()

	f.mu.RLock()
	defer f.mu.RUnlock()

	if len(data) <= f.MinSize {
		return [][]byte{data}
	}

	// For inbound traffic, we don't fragment
	if direction == DirectionInbound {
		return [][]byte{data}
	}

	// Fragment outbound traffic into independent chunks
	var chunks [][]byte
	offset := 0

	for offset < len(data) {
		var chunkSize int
		if f.Random {
			chunkSize = f.MinSize + f.rand.Intn(f.MaxSize-f.MinSize+1)
		} else {
			chunkSize = f.MaxSize
		}

		if offset+chunkSize > len(data) {
			chunkSize = len(data) - offset
		}

		// Create independent chunk to ensure proper network-level fragmentation
		chunk := make([]byte, chunkSize)
		copy(chunk, data[offset:offset+chunkSize])
		chunks = append(chunks, chunk)
		offset += chunkSize
	}

	return chunks
}

// SupportsFragmentation implements ModifierV2 interface
func (f *FragmentationModifier) SupportsFragmentation() bool {
	return true
}

// GetFragmentationConfig implements FragmentingModifier interface
func (f *FragmentationModifier) GetFragmentationConfig() *FragmentationConfig {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Return a copy to prevent external modification
	configCopy := *f.config
	return &configCopy
}

// SetFragmentationConfig implements FragmentingModifier interface
func (f *FragmentationModifier) SetFragmentationConfig(config *FragmentationConfig) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.config = config
	f.MinSize = config.MinSize
	f.MaxSize = config.MaxSize
	f.Random = config.RandomSizes
}

// GetMetrics returns fragmentation performance metrics
func (f *FragmentationModifier) GetMetrics() (uint64, float64, time.Duration, float64) {
	return f.metrics.GetMetrics()
}
