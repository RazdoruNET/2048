package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"socks5-dpi-proxy/internal/behavioral"
)

// BehavioralEvasionIntegration integrates behavioral evasion with pipeline
type BehavioralEvasionIntegration struct {
	evasion *behavioral.SimpleEvasion
	mu      sync.RWMutex
	enabled bool
	stats   *BehavioralStats
}

// BehavioralStats tracks behavioral evasion effectiveness
type BehavioralStats struct {
	RequestsProcessed int64
	SuccessRate      float64
	AverageDelay     time.Duration
	mu              sync.RWMutex
}

// NewBehavioralEvasionIntegration creates new integration
func NewBehavioralEvasionIntegration() *BehavioralEvasionIntegration {
	return &BehavioralEvasionIntegration{
		evasion: behavioral.NewSimpleEvasion(),
		enabled: true,
		stats: &BehavioralStats{
			SuccessRate: 0.8, // Initial success rate
		},
	}
}

// Initialize initializes behavioral evasion system
func (bei *BehavioralEvasionIntegration) Initialize() error {
	if err := bei.evasion.Configure(map[string]interface{}{
		"enabled":              true,
		"timing_randomization": true,
		"jitter_min":           50,
		"jitter_max":           200,
		"burst_chance":         0.1,
	}); err != nil {
		return fmt.Errorf("failed to configure behavioral evasion: %w", err)
	}

	return nil
}

// ApplyBehavioralEvasion applies behavioral evasion techniques
func (bei *BehavioralEvasionIntegration) ApplyBehavioralEvasion(ctx context.Context, data []byte, direction Direction, target string) ([]byte, error) {
	bei.mu.Lock()
	defer bei.mu.Unlock()

	if !bei.enabled || direction == DirectionInbound {
		return data, nil
	}

	// Apply behavioral evasion techniques
	processedData := bei.evasion.ApplyEvasion(ctx, data, direction, target)

	// Update statistics
	bei.updateStats(true)

	return processedData, nil
}

// GetStats returns behavioral evasion statistics
func (bei *BehavioralEvasionIntegration) GetStats() *BehavioralStats {
	bei.mu.RLock()
	defer bei.mu.RUnlock()
	return bei.stats
}

// SetEnabled enables/disables behavioral evasion
func (bei *BehavioralEvasionIntegration) SetEnabled(enabled bool) {
	bei.mu.Lock()
	defer bei.mu.Unlock()
	bei.enabled = enabled
	bei.evasion.SetEnabled(enabled)
}

// IsEnabled returns if behavioral evasion is enabled
func (bei *BehavioralEvasionIntegration) IsEnabled() bool {
	bei.mu.RLock()
	defer bei.mu.RUnlock()
	return bei.enabled
}

// updateStats updates behavioral evasion statistics
func (bei *BehavioralEvasionIntegration) updateStats(success bool) {
	bei.stats.mu.Lock()
	defer bei.stats.mu.Unlock()

	bei.stats.RequestsProcessed++
	
	// Update success rate with exponential moving average
	alpha := 0.1
	if success {
		bei.stats.SuccessRate = (1-alpha)*bei.stats.SuccessRate + alpha*1.0
	} else {
		bei.stats.SuccessRate = (1-alpha)*bei.stats.SuccessRate + alpha*0.0
	}
}
