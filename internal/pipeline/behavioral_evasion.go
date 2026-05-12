package pipeline

import (
	"math/rand"
	"time"
)

type BehavioralEvasionModifier struct {
	Enabled                bool
	TimingRandomization    bool
	ConnectionPattern      string
	JitterRange            string
	BurstSimulation        bool
	StreamSimulation       bool
	ImageRequestSimulation bool
	APISimulation          bool
	rand                   *rand.Rand
}

func (b *BehavioralEvasionModifier) Name() string {
	return "behavioral_evasion"
}

func (b *BehavioralEvasionModifier) Configure(config map[string]interface{}) error {
	if enabled, ok := config["enabled"].(bool); ok {
		b.Enabled = enabled
	} else {
		b.Enabled = true
	}

	if timingRandom, ok := config["timing_randomization"].(bool); ok {
		b.TimingRandomization = timingRandom
	} else {
		b.TimingRandomization = true
	}

	if pattern, ok := config["connection_pattern"].(string); ok {
		b.ConnectionPattern = pattern
	} else {
		b.ConnectionPattern = "human_like"
	}

	if jitterRange, ok := config["jitter_range"].(string); ok {
		b.JitterRange = jitterRange
	} else {
		b.JitterRange = "50-200ms"
	}

	if burstSimulation, ok := config["burst_simulation"].(bool); ok {
		b.BurstSimulation = burstSimulation
	} else {
		b.BurstSimulation = false
	}

	if streamSimulation, ok := config["stream_simulation"].(bool); ok {
		b.StreamSimulation = streamSimulation
	} else {
		b.StreamSimulation = false
	}

	if imageSimulation, ok := config["image_request_simulation"].(bool); ok {
		b.ImageRequestSimulation = imageSimulation
	} else {
		b.ImageRequestSimulation = false
	}

	if apiSimulation, ok := config["api_simulation"].(bool); ok {
		b.APISimulation = apiSimulation
	} else {
		b.APISimulation = false
	}

	b.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	return nil
}

func (b *BehavioralEvasionModifier) Process(data []byte, direction Direction) []byte {
	if !b.Enabled {
		return data
	}

	// Apply enhanced behavioral evasion patterns
	if b.TimingRandomization {
		data = b.applyTimingPatterns(data, direction)
	}

	if b.BurstSimulation {
		data = b.applyBurstSimulation(data)
	}

	if b.StreamSimulation {
		data = b.applyStreamSimulation(data)
	}

	// Apply advanced techniques
	data = b.applyAdvancedObfuscation(data, direction)
	data = b.applyFingerprintingProtection(data, direction)

	return data
}

func (b *BehavioralEvasionModifier) applyTimingRandomization() {
	// Parse jitter range
	minJitter := 50 * time.Millisecond
	maxJitter := 200 * time.Millisecond

	// Add random delay
	jitter := time.Duration(b.rand.Intn(int(maxJitter-minJitter))) + minJitter
	time.Sleep(jitter)
}

func (b *BehavioralEvasionModifier) applyHumanLikePatterns(data []byte) []byte {
	// Add human-like variations to traffic patterns
	if len(data) > 0 {
		// Simulate human behavior with random variations
		if b.rand.Intn(100) < 10 { // 10% chance
			// Add small random delays
			time.Sleep(time.Duration(b.rand.Intn(50)) * time.Millisecond)
		}
	}
	return data
}

func (b *BehavioralEvasionModifier) applyAPISimulation() {
	// Simulate API request patterns
	// Add realistic delays between API calls
	time.Sleep(time.Duration(b.rand.Intn(500)) * time.Millisecond)
}

func (b *BehavioralEvasionModifier) applyTimingPatterns(data []byte, direction Direction) []byte {
	// Calculate delay based on data size and direction
	baseDelay := 50 * time.Millisecond

	if direction == DirectionOutbound {
		baseDelay *= 2 // More delay for outbound
	}

	if len(data) > 1024*1024 { // > 1MB
		baseDelay *= 3
	} else if len(data) < 10*1024 { // < 10KB
		baseDelay /= 2
	}

	// Add jitter
	jitter := time.Duration(b.rand.Intn(50)) * time.Millisecond
	time.Sleep(baseDelay + jitter)

	return data
}

func (b *BehavioralEvasionModifier) applyBurstSimulation(data []byte) []byte {
	// Simulate human-like burst patterns
	if b.rand.Float32() < 0.3 { // 30% chance of burst
		// Add small delay to simulate burst
		time.Sleep(time.Duration(b.rand.Intn(20)) * time.Millisecond)
	}

	return data
}

func (b *BehavioralEvasionModifier) applyStreamSimulation(data []byte) []byte {
	// Simulate streaming behavior
	if len(data) > 1024 { // Only for larger data
		// Split into chunks with delays
		chunkSize := 1024
		for i := 0; i < len(data); i += chunkSize {
			if i > 0 {
				time.Sleep(time.Duration(b.rand.Intn(10)) * time.Millisecond)
			}
		}
	}

	return data
}

func (b *BehavioralEvasionModifier) applyAdvancedObfuscation(data []byte, direction Direction) []byte {
	if len(data) == 0 {
		return data
	}

	// Simple XOR obfuscation with rotating key
	obfuscated := make([]byte, len(data))
	key := byte(len(data) % 256)

	for i, b := range data {
		rotatingKey := key ^ byte(i%16)
		obfuscated[i] = b ^ rotatingKey
	}

	return obfuscated
}

func (b *BehavioralEvasionModifier) applyFingerprintingProtection(data []byte, direction Direction) []byte {
	// Simulate fingerprinting protection with timing
	if direction == DirectionOutbound {
		// Add delay to simulate browser fingerprinting protection
		delay := time.Duration(20+b.rand.Intn(30)) * time.Millisecond
		time.Sleep(delay)
	}

	return data
}
