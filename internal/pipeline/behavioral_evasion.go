package pipeline

import (
	"math/rand"
	"time"
)

type BehavioralEvasionModifier struct {
	Enabled               bool
	TimingRandomization    bool
	ConnectionPattern     string
	JitterRange          string
	BurstSimulation      bool
	StreamSimulation     bool
	ImageRequestSimulation bool
	APISimulation       bool
	rand                 *rand.Rand
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

	// Apply timing randomization for outbound traffic
	if direction == DirectionOutbound && b.TimingRandomization {
		b.applyTimingRandomization()
	}

	// Apply behavioral patterns
	if b.ConnectionPattern == "human_like" {
		data = b.applyHumanLikePatterns(data)
	}

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
