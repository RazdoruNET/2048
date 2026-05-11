package behavioral

import (
	"context"
	"crypto/rand"
	"sync"
	"time"

	"socks5-dpi-proxy/internal/pipeline"
)

// SimpleEvasion implements basic behavioral evasion techniques
type SimpleEvasion struct {
	enabled        bool
	timingRandom   bool
	jitterMin      time.Duration
	jitterMax      time.Duration
	burstChance    float64
	mu             sync.RWMutex
	stats          *EvasionStats
}

// EvasionStats tracks evasion effectiveness
type EvasionStats struct {
	RequestsProcessed int64
	SuccessRate      float64
	AverageDelay     time.Duration
	mu              sync.RWMutex
}

// NewSimpleEvasion creates new simple evasion system
func NewSimpleEvasion() *SimpleEvasion {
	return &SimpleEvasion{
		enabled:        true,
		timingRandom:   true,
		jitterMin:      50 * time.Millisecond,
		jitterMax:      200 * time.Millisecond,
		burstChance:    0.1,
		stats: &EvasionStats{
			SuccessRate:  0.8,
		},
	}
}

// ApplyEvasion applies behavioral evasion techniques
func (se *SimpleEvasion) ApplyEvasion(ctx context.Context, data []byte, direction pipeline.Direction) []byte {
	se.mu.Lock()
	defer se.mu.Unlock()

	if !se.enabled || direction == pipeline.DirectionInbound {
		return data
	}

	// Apply timing randomization
	if se.timingRandom {
		delay := se.calculateDelay()
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	// Apply burst simulation with small probability
	if rand.Float64() < se.burstChance {
		data = se.simulateBurst(data)
	}

	// Update statistics
	se.updateStats(true)

	return data
}

// calculateDelay calculates random delay
func (se *SimpleEvasion) calculateDelay() time.Duration {
	if se.jitterMax <= se.jitterMin {
		return 0
	}
	
	// Random delay between min and max
	diff := se.jitterMax - se.jitterMin
	delay := se.jitterMin + time.Duration(rand.Int63n(int64(diff)))
	return delay
}

// simulateBurst simulates burst traffic pattern
func (se *SimpleEvasion) simulateBurst(data []byte) []byte {
	// Simple burst simulation - duplicate small chunks
	if len(data) <= 64 {
		return data
	}

	// Create burst pattern
	burstSize := 32 + rand.Intn(32) // 32-64 bytes
	if burstSize > len(data) {
		burstSize = len(data)
	}

	result := make([]byte, 0, len(data)*2) // Max double size for burst
	copy(result, data)
	
	// Add some noise
	for i := len(data); i < len(result); i++ {
		if rand.Intn(10) == 0 {
			result[i] = byte(rand.Intn(256))
		}
	}

	return result[:len(data)]
}

// updateStats updates evasion statistics
func (se *SimpleEvasion) updateStats(success bool) {
	se.stats.mu.Lock()
	defer se.stats.mu.Unlock()

	se.stats.RequestsProcessed++
	
	// Update success rate with exponential moving average
	alpha := 0.1
	if success {
		se.stats.SuccessRate = (1-alpha)*se.stats.SuccessRate + alpha*1.0
	} else {
		se.stats.SuccessRate = (1-alpha)*se.stats.SuccessRate + alpha*0.0
	}
}

// GetStats returns evasion statistics
func (se *SimpleEvasion) GetStats() *EvasionStats {
	se.stats.mu.RLock()
	defer se.stats.mu.RUnlock()
	return se.stats
}

// SetEnabled enables/disables evasion
func (se *SimpleEvasion) SetEnabled(enabled bool) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.enabled = enabled
}

// IsEnabled returns if evasion is enabled
func (se *SimpleEvasion) IsEnabled() bool {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.enabled
}

// Configure configures evasion parameters
func (se *SimpleEvasion) Configure(config map[string]interface{}) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	if enabled, ok := config["enabled"].(bool); ok {
		se.enabled = enabled
	}

	if timingRandom, ok := config["timing_randomization"].(bool); ok {
		se.timingRandom = timingRandom
	}

	if jitterMin, ok := config["jitter_min"].(int); ok {
		se.jitterMin = time.Duration(jitterMin) * time.Millisecond
	}

	if jitterMax, ok := config["jitter_max"].(int); ok {
		se.jitterMax = time.Duration(jitterMax) * time.Millisecond
	}

	if burstChance, ok := config["burst_chance"].(float64); ok {
		se.burstChance = burstChance
	}

	return nil
}
