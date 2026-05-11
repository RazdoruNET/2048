package behavioral

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"socks5-dpi-proxy/internal/pipeline"
)

// EvasionStats tracks evasion effectiveness
type EvasionStats struct {
	RequestsProcessed int64
	SuccessRate       float64
	AverageDelay      time.Duration
	mu                sync.RWMutex
}

// Evasion implements behavioral evasion techniques
type Evasion struct {
	enabled      bool
	timingRandom bool
	jitterMin    time.Duration
	jitterMax    time.Duration
	burstChance  float64
	mu           sync.RWMutex
	stats        *EvasionStats
}

// NewEvasion creates new evasion system
func NewEvasion() *Evasion {
	return &Evasion{
		enabled:      true,
		timingRandom: true,
		jitterMin:    10 * time.Millisecond, // Reduced from 50ms
		jitterMax:    30 * time.Millisecond, // Reduced from 200ms
		burstChance:  0.05,                  // Reduced from 0.1
		stats: &EvasionStats{
			SuccessRate: 0.8,
		},
	}
}

// ApplyEvasion applies behavioral evasion techniques
func (e *Evasion) ApplyEvasion(ctx context.Context, data []byte, direction pipeline.Direction) []byte {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.enabled || direction == pipeline.DirectionInbound {
		return data
	}

	// Apply minimal timing randomization only for small packets
	if e.timingRandom && len(data) < 512 {
		delay := e.calculateOptimalDelay()
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	// Apply burst simulation with reduced probability
	if rand.Float64() < e.burstChance {
		data = e.simulateBurst(data)
	}

	// Update statistics
	e.updateStats(true)

	return data
}

// calculateOptimalDelay calculates optimal delay based on data size
func (e *Evasion) calculateOptimalDelay() time.Duration {
	if e.jitterMax <= e.jitterMin {
		return 0
	}

	// Small random delay between min and max
	diff := e.jitterMax - e.jitterMin
	delay := e.jitterMin + time.Duration(rand.Int63n(int64(diff)))
	return delay
}

// simulateBurst simulates burst traffic pattern
func (e *Evasion) simulateBurst(data []byte) []byte {
	// Simple burst simulation
	if len(data) <= 64 {
		return data
	}

	// Create burst pattern
	burstSize := 16 + rand.Intn(16) // 16-32 bytes (reduced)
	if burstSize > len(data) {
		burstSize = len(data)
	}

	result := make([]byte, len(data))
	copy(result, data)

	// Add minimal noise to simulate burst
	for i := 0; i < len(result); i++ {
		if rand.Intn(20) == 0 { // Reduced probability
			result[i] = byte(rand.Intn(256))
		}
	}

	return result
}

// updateStats updates evasion statistics
func (e *Evasion) updateStats(success bool) {
	e.stats.mu.Lock()
	defer e.stats.mu.Unlock()

	e.stats.RequestsProcessed++

	// Update success rate with exponential moving average
	alpha := 0.1
	if success {
		e.stats.SuccessRate = (1-alpha)*e.stats.SuccessRate + alpha*1.0
	} else {
		e.stats.SuccessRate = (1-alpha)*e.stats.SuccessRate + alpha*0.0
	}
}

// GetStats returns evasion statistics
func (e *Evasion) GetStats() *EvasionStats {
	e.stats.mu.RLock()
	defer e.stats.mu.RUnlock()
	return e.stats
}

// SetEnabled enables/disables evasion
func (e *Evasion) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// IsEnabled returns if evasion is enabled
func (e *Evasion) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}
