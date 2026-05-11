package behavioral

import (
	"math/rand"
	"sync"
	"time"
)

// TimingEngine provides advanced timing patterns for behavioral evasion
type TimingEngine struct {
	humanPatterns  *HumanTimingPatterns
	adaptiveDelay  *AdaptiveDelayCalculator
	randomJitter   *RandomJitterGenerator
	burstSimulator *BurstSimulator
	mu             sync.RWMutex
	enabled        bool
}

// HumanTimingPatterns mimics human browsing behavior
type HumanTimingPatterns struct {
	minReadDelay  time.Duration
	maxReadDelay  time.Duration
	minWriteDelay time.Duration
	maxWriteDelay time.Duration
	typingPattern *TypingPattern
	scrollPattern *ScrollPattern
	clickPattern  *ClickPattern
}

// AdaptiveDelayCalculator calculates adaptive delays based on DPI detection
type AdaptiveDelayCalculator struct {
	baseDelay        time.Duration
	currentDelay     time.Duration
	detectionLevel   int
	adjustmentFactor float64
	mu               sync.RWMutex
}

// RandomJitterGenerator adds random jitter to timing
type RandomJitterGenerator struct {
	minJitter time.Duration
	maxJitter time.Duration
	rng       *rand.Rand
}

// BurstSimulator simulates human traffic bursts
type BurstSimulator struct {
	burstSize   int
	burstGap    time.Duration
	burstChance float64
	rng         *rand.Rand
}

// TypingPattern simulates human typing behavior
type TypingPattern struct {
	minTypingSpeed time.Duration
	maxTypingSpeed time.Duration
	pauseChance    float64
	errorChance    float64
	rng            *rand.Rand
}

// ScrollPattern simulates human scrolling behavior
type ScrollPattern struct {
	minScrollSpeed time.Duration
	maxScrollSpeed time.Duration
	scrollChance   float64
	rng            *rand.Rand
}

// ClickPattern simulates human click behavior
type ClickPattern struct {
	minClickDelay     time.Duration
	maxClickDelay     time.Duration
	doubleClickChance float64
	rng               *rand.Rand
}

// NewTimingEngine creates new timing engine
func NewTimingEngine() *TimingEngine {
	return &TimingEngine{
		humanPatterns:  NewHumanTimingPatterns(),
		adaptiveDelay:  NewAdaptiveDelayCalculator(),
		randomJitter:   NewRandomJitterGenerator(),
		burstSimulator: NewBurstSimulator(),
		enabled:        true,
	}
}

// GetReadDelay returns human-like read delay
func (te *TimingEngine) GetReadDelay() time.Duration {
	te.mu.RLock()
	defer te.mu.RUnlock()

	if !te.enabled {
		return 0
	}

	// Base delay from human patterns
	baseDelay := te.humanPatterns.GetRandomReadDelay()

	// Add adaptive delay
	adaptiveDelay := te.adaptiveDelay.GetCurrentDelay()

	// Add random jitter
	jitter := te.randomJitter.GetJitter()

	return baseDelay + adaptiveDelay + jitter
}

// GetWriteDelay returns human-like write delay
func (te *TimingEngine) GetWriteDelay() time.Duration {
	te.mu.RLock()
	defer te.mu.RUnlock()

	if !te.enabled {
		return 0
	}

	// Base delay from human patterns
	baseDelay := te.humanPatterns.GetRandomWriteDelay()

	// Add adaptive delay
	adaptiveDelay := te.adaptiveDelay.GetCurrentDelay()

	// Add random jitter
	jitter := te.randomJitter.GetJitter()

	// Check for burst behavior
	if te.burstSimulator.ShouldBurst() {
		return baseDelay / 2 // Faster during bursts
	}

	return baseDelay + adaptiveDelay + jitter
}

// GetTypingDelay returns typing simulation delay
func (te *TimingEngine) GetTypingDelay() time.Duration {
	te.mu.RLock()
	defer te.mu.RUnlock()

	if !te.enabled {
		return 0
	}

	return te.humanPatterns.typingPattern.GetTypingDelay()
}

// GetScrollDelay returns scrolling simulation delay
func (te *TimingEngine) GetScrollDelay() time.Duration {
	te.mu.RLock()
	defer te.mu.RUnlock()

	if !te.enabled {
		return 0
	}

	if te.humanPatterns.scrollPattern.ShouldScroll() {
		return te.humanPatterns.scrollPattern.GetScrollDelay()
	}

	return 0
}

// GetClickDelay returns click simulation delay
func (te *TimingEngine) GetClickDelay() time.Duration {
	te.mu.RLock()
	defer te.mu.RUnlock()

	if !te.enabled {
		return 0
	}

	return te.humanPatterns.clickPattern.GetClickDelay()
}

// UpdateDetectionLevel updates adaptive delay based on DPI detection
func (te *TimingEngine) UpdateDetectionLevel(level int) {
	te.mu.Lock()
	defer te.mu.Unlock()

	te.adaptiveDelay.UpdateDetectionLevel(level)
}

// SetEnabled enables/disables timing engine
func (te *TimingEngine) SetEnabled(enabled bool) {
	te.mu.Lock()
	defer te.mu.Unlock()

	te.enabled = enabled
}

// IsEnabled returns timing engine status
func (te *TimingEngine) IsEnabled() bool {
	te.mu.RLock()
	defer te.mu.RUnlock()

	return te.enabled
}

// GetHumanizedDelay returns humanized delay for given pattern and DPI level
func (te *TimingEngine) GetHumanizedDelay(pattern string, dpiLevel int) time.Duration {
	te.mu.RLock()
	defer te.mu.RUnlock()

	if !te.enabled {
		return 0
	}

	baseDelay := te.humanPatterns.minReadDelay

	// Adjust delay based on DPI level
	switch dpiLevel {
	case 4, 5: // High DPI detection
		baseDelay = time.Duration(float64(baseDelay) * 2.0)
	case 3: // Medium DPI detection
		baseDelay = time.Duration(float64(baseDelay) * 1.5)
	}

	// Add pattern-specific variation
	switch pattern {
	case "mobile":
		baseDelay = time.Duration(float64(baseDelay) * 0.7)
	case "bot":
		baseDelay = time.Duration(float64(baseDelay) * 0.3)
	}

	// Add jitter
	jitter := te.randomJitter.GetJitter()
	return baseDelay + jitter
}

// UpdateContext updates timing engine context
func (te *TimingEngine) UpdateContext(dpiLevel int, dataSize int) {
	te.mu.Lock()
	defer te.mu.Unlock()

	te.adaptiveDelay.UpdateDetectionLevel(dpiLevel)

	// Adjust timing based on data size
	if dataSize > 1024*1024 { // > 1MB
		te.adaptiveDelay.IncreaseDelay()
	} else if dataSize < 10*1024 { // < 10KB
		te.adaptiveDelay.DecreaseDelay()
	}
}

// GetStats returns timing engine statistics
func (te *TimingEngine) GetStats() map[string]interface{} {
	te.mu.RLock()
	defer te.mu.RUnlock()

	return map[string]interface{}{
		"enabled":         te.enabled,
		"current_delay":   te.adaptiveDelay.GetCurrentDelay(),
		"detection_level": te.adaptiveDelay.detectionLevel,
		"human_patterns": map[string]interface{}{
			"min_read_delay":  te.humanPatterns.minReadDelay,
			"max_read_delay":  te.humanPatterns.maxReadDelay,
			"min_write_delay": te.humanPatterns.minWriteDelay,
			"max_write_delay": te.humanPatterns.maxWriteDelay,
		},
	}
}

// NewHumanTimingPatterns creates new human timing patterns
func NewHumanTimingPatterns() *HumanTimingPatterns {
	return &HumanTimingPatterns{
		minReadDelay:  10 * time.Millisecond,
		maxReadDelay:  100 * time.Millisecond,
		minWriteDelay: 20 * time.Millisecond,
		maxWriteDelay: 200 * time.Millisecond,
		typingPattern: NewTypingPattern(),
		scrollPattern: NewScrollPattern(),
		clickPattern:  NewClickPattern(),
	}
}

// GetRandomReadDelay returns random read delay
func (htp *HumanTimingPatterns) GetRandomReadDelay() time.Duration {
	return time.Duration(rand.Int63n(int64(htp.maxReadDelay-htp.minReadDelay))) + htp.minReadDelay
}

// GetRandomWriteDelay returns random write delay
func (htp *HumanTimingPatterns) GetRandomWriteDelay() time.Duration {
	return time.Duration(rand.Int63n(int64(htp.maxWriteDelay-htp.minWriteDelay))) + htp.minWriteDelay
}

// NewTypingPattern creates new typing pattern
func NewTypingPattern() *TypingPattern {
	return &TypingPattern{
		minTypingSpeed: 50 * time.Millisecond,
		maxTypingSpeed: 200 * time.Millisecond,
		pauseChance:    0.1,
		errorChance:    0.05,
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetTypingDelay returns typing simulation delay
func (tp *TypingPattern) GetTypingDelay() time.Duration {
	// Simulate typing speed variation
	baseDelay := time.Duration(rand.Int63n(int64(tp.maxTypingSpeed-tp.minTypingSpeed))) + tp.minTypingSpeed

	// Add random pauses
	if rand.Float64() < tp.pauseChance {
		baseDelay += time.Duration(rand.Int63n(500)) * time.Millisecond
	}

	// Add typing errors
	if rand.Float64() < tp.errorChance {
		baseDelay += time.Duration(rand.Int63n(1000)) * time.Millisecond
	}

	return baseDelay
}

// NewScrollPattern creates new scroll pattern
func NewScrollPattern() *ScrollPattern {
	return &ScrollPattern{
		minScrollSpeed: 50 * time.Millisecond,
		maxScrollSpeed: 300 * time.Millisecond,
		scrollChance:   0.3,
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ShouldScroll returns if scroll should occur
func (sp *ScrollPattern) ShouldScroll() bool {
	return rand.Float64() < sp.scrollChance
}

// GetScrollDelay returns scroll simulation delay
func (sp *ScrollPattern) GetScrollDelay() time.Duration {
	return time.Duration(rand.Int63n(int64(sp.maxScrollSpeed-sp.minScrollSpeed))) + sp.minScrollSpeed
}

// NewClickPattern creates new click pattern
func NewClickPattern() *ClickPattern {
	return &ClickPattern{
		minClickDelay:     50 * time.Millisecond,
		maxClickDelay:     300 * time.Millisecond,
		doubleClickChance: 0.15,
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetClickDelay returns click simulation delay
func (cp *ClickPattern) GetClickDelay() time.Duration {
	baseDelay := time.Duration(rand.Int63n(int64(cp.maxClickDelay-cp.minClickDelay))) + cp.minClickDelay

	// Add double-click behavior
	if rand.Float64() < cp.doubleClickChance {
		baseDelay /= 2
	}

	return baseDelay
}

// NewAdaptiveDelayCalculator creates new adaptive delay calculator
func NewAdaptiveDelayCalculator() *AdaptiveDelayCalculator {
	return &AdaptiveDelayCalculator{
		baseDelay:        50 * time.Millisecond,
		currentDelay:     50 * time.Millisecond,
		detectionLevel:   0,
		adjustmentFactor: 1.5,
	}
}

// GetCurrentDelay returns current adaptive delay
func (adc *AdaptiveDelayCalculator) GetCurrentDelay() time.Duration {
	adc.mu.RLock()
	defer adc.mu.RUnlock()

	return adc.currentDelay
}

// UpdateDetectionLevel updates detection level
func (adc *AdaptiveDelayCalculator) UpdateDetectionLevel(level int) {
	adc.mu.Lock()
	defer adc.mu.Unlock()

	adc.detectionLevel = level

	// Adjust delay based on detection level
	switch level {
	case 4, 5: // High detection
		adc.currentDelay = time.Duration(float64(adc.baseDelay) * 2.0)
	case 3: // Medium detection
		adc.currentDelay = time.Duration(float64(adc.baseDelay) * 1.5)
	default: // Low detection
		adc.currentDelay = adc.baseDelay
	}
}

// IncreaseDelay increases current delay
func (adc *AdaptiveDelayCalculator) IncreaseDelay() {
	adc.mu.Lock()
	defer adc.mu.Unlock()

	adc.currentDelay = time.Duration(float64(adc.currentDelay) * 1.2)
}

// DecreaseDelay decreases current delay
func (adc *AdaptiveDelayCalculator) DecreaseDelay() {
	adc.mu.Lock()
	defer adc.mu.Unlock()

	adc.currentDelay = time.Duration(float64(adc.currentDelay) * 0.8)
	if adc.currentDelay < adc.baseDelay {
		adc.currentDelay = adc.baseDelay
	}
}

// NewRandomJitterGenerator creates new jitter generator
func NewRandomJitterGenerator() *RandomJitterGenerator {
	return &RandomJitterGenerator{
		minJitter: -20 * time.Millisecond,
		maxJitter: 20 * time.Millisecond,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetJitter returns random jitter
func (rjg *RandomJitterGenerator) GetJitter() time.Duration {
	jitterRange := int64(rjg.maxJitter - rjg.minJitter)
	return time.Duration(rjg.minJitter) + time.Duration(rjg.rng.Int63n(jitterRange))
}

// NewBurstSimulator creates new burst simulator
func NewBurstSimulator() *BurstSimulator {
	return &BurstSimulator{
		burstSize:   5,
		burstGap:    100 * time.Millisecond,
		burstChance: 0.1,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ShouldBurst returns if burst should occur
func (bs *BurstSimulator) ShouldBurst() bool {
	return rand.Float64() < bs.burstChance
}
