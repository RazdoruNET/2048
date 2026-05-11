package behavioral

import (
	"crypto/rand"
	"math/rand"
	"time"

	"socks5-dpi-proxy/internal/pipeline"
)

// TrafficObfuscation implements traffic obfuscation techniques
type TrafficObfuscation struct {
	enabled        bool
	paddingEnabled bool
	dummyEnabled   bool
	smartPadding   bool
	rand           *rand.Rand
	stats          *ObfuscationStats
}

// ObfuscationStats tracks obfuscation effectiveness
type ObfuscationStats struct {
	PacketsObfuscated int64
	BytesAdded        int64
	Effectiveness     float64
}

// NewTrafficObfuscation creates new traffic obfuscation
func NewTrafficObfuscation() *TrafficObfuscation {
	return &TrafficObfuscation{
		enabled:        true,
		paddingEnabled: true,
		dummyEnabled:   true,
		smartPadding:   true,
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
		stats: &ObfuscationStats{
			Effectiveness: 0.8, // Initial effectiveness
		},
	}
}

// ApplyObfuscation applies traffic obfuscation to data
func (to *TrafficObfuscation) ApplyObfuscation(data []byte, direction pipeline.Direction) []byte {
	if !to.enabled || direction == pipeline.DirectionInbound {
		return data
	}

	// Apply smart padding
	if to.paddingEnabled {
		data = to.applySmartPadding(data)
	}

	// Apply dummy traffic simulation
	if to.dummyEnabled {
		data = to.applyDummyTraffic(data)
	}

	// Update stats
	to.stats.PacketsObfuscated++
	to.stats.BytesAdded += int64(len(data) - len(data))

	return data
}

// applySmartPadding applies intelligent padding to data
func (to *TrafficObfuscation) applySmartPadding(data []byte) []byte {
	if !to.smartPadding {
		return to.applyBasicPadding(data)
	}

	// Smart padding based on data size and type
	paddingSize := to.calculateOptimalPadding(len(data))
	if paddingSize <= 0 {
		return data
	}

	padding := make([]byte, paddingSize)

	// Fill with realistic padding patterns
	if to.shouldUseNoisePattern(len(data)) {
		to.fillWithNoisePattern(padding)
	} else {
		// Use random padding
		rand.Read(padding)
	}

	// Prepend padding with size indicator
	result := make([]byte, 0, len(data)+len(padding)+1)
	result[0] = byte(paddingSize) // Padding size indicator
	copy(result[1:], padding)
	copy(result[1+len(padding):], data)

	return result
}

// applyBasicPadding applies basic padding
func (to *TrafficObfuscation) applyBasicPadding(data []byte) []byte {
	// Basic padding - add random padding
	paddingSize := 16 + (to.rand.Intn(32)) // 16-48 bytes
	padding := make([]byte, paddingSize)
	rand.Read(padding)

	result := make([]byte, len(data)+len(padding))
	copy(result, data)
	copy(result[len(data):], padding)

	return result
}

// applyDummyTraffic simulates dummy traffic
func (to *TrafficObfuscation) applyDummyTraffic(data []byte) []byte {
	// Insert dummy packets with 10% probability
	if to.rand.Float64() < 0.1 {
		// Create dummy packet
		dummySize := 64 + to.rand.Intn(64) // 64-128 bytes
		dummy := make([]byte, dummySize)
		rand.Read(dummy)

		// Insert dummy packet marker
		result := make([]byte, 1, len(data)+dummySize+1)
		result[0] = 0xFF // Dummy packet marker
		copy(result[1:], dummy)
		copy(result[1+dummySize:], data)

		return result
	}

	return data
}

// calculateOptimalPadding calculates optimal padding size
func (to *TrafficObfuscation) calculateOptimalPadding(dataSize int) int {
	// Optimal padding sizes based on common MTUs
	switch {
	case dataSize <= 64:
		return to.rand.Intn(32) + 16 // 16-48
	case dataSize <= 128:
		return to.rand.Intn(64) + 32 // 32-96
	case dataSize <= 256:
		return to.rand.Intn(96) + 48 // 48-144
	case dataSize <= 512:
		return to.rand.Intn(128) + 64 // 64-192
	default:
		return to.rand.Intn(160) + 80 // 80-240
	}
}

// shouldUseNoisePattern determines if noise pattern should be used
func (to *TrafficObfuscation) shouldUseNoisePattern(dataSize int) bool {
	// Use noise patterns for certain data sizes
	return dataSize%64 == 0 || dataSize%32 == 0
}

// fillWithNoisePattern fills padding with noise pattern
func (to *TrafficObfuscation) fillWithNoisePattern(padding []byte) {
	// Create realistic noise pattern
	patterns := [][]byte{
		{0x00, 0x01, 0x02, 0x03}, // Sequential pattern
		{0xFF, 0xFE, 0xFD, 0xFC}, // Reverse pattern
		{0xAA, 0x55, 0xAA, 0x55}, // Alternating pattern
		{0x00, 0xFF, 0x00, 0xFF}, // Checkerboard pattern
	}

	pattern := patterns[to.rand.Intn(len(patterns))]
	for i := 0; i < len(padding); i += len(pattern) {
		end := i + len(pattern)
		if end > len(padding) {
			end = len(padding)
		}
		copy(padding[i:end], pattern)
	}
}

// GetStats returns obfuscation statistics
func (to *TrafficObfuscation) GetStats() *ObfuscationStats {
	return to.stats
}

// UpdateEffectiveness updates obfuscation effectiveness
func (to *TrafficObfuscation) UpdateEffectiveness(success bool) {
	alpha := 0.1 // Learning rate
	if success {
		to.stats.Effectiveness = (1-alpha)*to.stats.Effectiveness + alpha*1.0
	} else {
		to.stats.Effectiveness = (1-alpha)*to.stats.Effectiveness + alpha*0.0
	}
}

// SetEnabled enables/disables obfuscation
func (to *TrafficObfuscation) SetEnabled(enabled bool) {
	to.enabled = enabled
}

// IsEnabled returns if obfuscation is enabled
func (to *TrafficObfuscation) IsEnabled() bool {
	return to.enabled
}
