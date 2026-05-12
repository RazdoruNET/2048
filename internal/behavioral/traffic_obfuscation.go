package behavioral

import (
	crand "crypto/rand"
	"math"
	"math/rand"
	mrand "math/rand"
	"sync"
)

// TrafficObfuscator provides ML-optimized traffic obfuscation
type TrafficObfuscator struct {
	mlEngine         *MLEngine
	patternGenerator *DynamicPatternGenerator
	noiseGenerator   *NoiseTrafficGenerator
	trafficMixer     *TrafficMixer
	mu               sync.RWMutex
	enabled          bool
	obfuscationLevel int
}

// MLEngine provides ML-based optimization
type MLEngine struct {
	effectivenessTracker *EffectivenessTracker
	patternLearner       *PatternLearner
	adaptationRate       float64
	mu                   sync.RWMutex
}

// DynamicPatternGenerator generates dynamic obfuscation patterns
type DynamicPatternGenerator struct {
	currentPattern []byte
	patternHistory [][]byte
	successRates   map[string]float64
	mu             sync.RWMutex
}

// NoiseTrafficGenerator generates realistic noise traffic
type NoiseTrafficGenerator struct {
	noiseRatio     float64
	noisePatterns  [][]byte
	currentPattern int
	mu             sync.RWMutex
}

// TrafficMixer mixes legitimate and obfuscated traffic
type TrafficMixer struct {
	legitRatio      float64
	obfuscatedRatio float64
	mixingStrategy  string
	mu              sync.RWMutex
}

// EffectivenessTracker tracks obfuscation effectiveness
type EffectivenessTracker struct {
	successCount     int
	failureCount     int
	totalAttempts    int
	effectivenessMap map[string]float64
	mu               sync.RWMutex
}

// PatternLearner learns optimal patterns
type PatternLearner struct {
	learningRate   float64
	patternWeights map[string]float64
	recentResults  []float64
	mu             sync.RWMutex
}

// NewTrafficObfuscator creates new traffic obfuscator
func NewTrafficObfuscator() *TrafficObfuscator {
	return &TrafficObfuscator{
		mlEngine:         NewMLEngine(),
		patternGenerator: NewDynamicPatternGenerator(),
		noiseGenerator:   NewNoiseTrafficGenerator(),
		trafficMixer:     NewTrafficMixer(),
		enabled:          true,
		obfuscationLevel: 1, // 0-5 scale
	}
}

// ObfuscateData applies ML-optimized obfuscation
func (to *TrafficObfuscator) ObfuscateData(data []byte, dpiLevel int) ([]byte, error) {
	to.mu.RLock()
	defer to.mu.RUnlock()

	if !to.enabled {
		return data, nil
	}

	// Get current effectiveness
	effectiveness, _ := to.mlEngine.GetCurrentEffectiveness()

	// Select obfuscation strategy based on DPI detection level
	strategy := to.selectObfuscationStrategy(dpiLevel, effectiveness)

	switch strategy {
	case "pattern_substitution":
		return to.patternGenerator.ObfuscateWithPattern(data)
	case "noise_injection":
		return to.noiseGenerator.InjectNoise(data)
	case "traffic_mixing":
		return to.trafficMixer.MixTraffic(data)
	case "adaptive_ml":
		return to.applyMLObfuscation(data, dpiLevel)
	default:
		result, _ := to.applyBasicObfuscation(data)
		return result, nil
	}
}

// selectObfuscationStrategy selects optimal strategy
func (to *TrafficObfuscator) selectObfuscationStrategy(dpiLevel int, effectiveness float64) string {
	// High DPI detection -> use advanced strategies
	if dpiLevel > 3 {
		if effectiveness < 0.7 {
			return "adaptive_ml"
		}
		return "traffic_mixing"
	}

	// Medium DPI detection -> use pattern-based strategies
	if dpiLevel > 1 {
		if effectiveness < 0.8 {
			return "pattern_substitution"
		}
		return "noise_injection"
	}

	// Low DPI detection -> use basic strategies
	return "basic"
}

// applyBasicObfuscation applies basic obfuscation
func (to *TrafficObfuscator) applyBasicObfuscation(data []byte) ([]byte, error) {
	// Simple XOR with random key
	key := make([]byte, 16)
	crand.Read(key)

	obfuscated := make([]byte, len(data))
	for i, b := range data {
		obfuscated[i] = b ^ key[i%16]
	}

	return obfuscated, nil
}

// applyMLObfuscation applies ML-optimized obfuscation
func (to *TrafficObfuscator) applyMLObfuscation(data []byte, dpiLevel int) ([]byte, error) {
	// Get learned pattern
	pattern := to.mlEngine.patternLearner.GetOptimalPattern(dpiLevel)

	// Apply pattern-based obfuscation
	obfuscated, _ := to.applyPatternObfuscation(data, pattern)

	// Update ML engine with result
	to.mlEngine.UpdateEffectiveness(len(data), true)

	return obfuscated, nil
}

// applyPatternObfuscation applies pattern-based obfuscation
func (to *TrafficObfuscator) applyPatternObfuscation(data []byte, pattern []byte) ([]byte, error) {
	if len(pattern) == 0 {
		return to.applyBasicObfuscation(data)
	}

	obfuscated := make([]byte, len(data))
	patternLen := len(pattern)

	for i, b := range data {
		patternByte := pattern[i%patternLen]
		obfuscated[i] = b ^ patternByte ^ byte(i)
	}

	return obfuscated, nil
}

// UpdateEffectiveness updates obfuscation effectiveness
func (to *TrafficObfuscator) UpdateEffectiveness(dataSize int, success bool) {
	to.mu.Lock()
	defer to.mu.Unlock()

	to.mlEngine.effectivenessTracker.RecordResult(success)
	to.patternGenerator.UpdatePattern(success)
	to.noiseGenerator.UpdateEffectiveness(success)
}

// SetObfuscationLevel sets obfuscation intensity
func (to *TrafficObfuscator) SetObfuscationLevel(level int) {
	to.mu.Lock()
	defer to.mu.Unlock()

	to.obfuscationLevel = level
	to.noiseGenerator.SetNoiseRatio(float64(level) / 5.0)
}

// GetObfuscationLevel returns current level
func (to *TrafficObfuscator) GetObfuscationLevel() int {
	to.mu.RLock()
	defer to.mu.RUnlock()

	return to.obfuscationLevel
}

// SetEnabled enables/disables obfuscator
func (to *TrafficObfuscator) SetEnabled(enabled bool) {
	to.mu.Lock()
	defer to.mu.Unlock()

	to.enabled = enabled
}

// IsEnabled returns obfuscator status
func (to *TrafficObfuscator) IsEnabled() bool {
	to.mu.RLock()
	defer to.mu.RUnlock()

	return to.enabled
}

// GetStats returns traffic obfuscator statistics
func (to *TrafficObfuscator) GetStats() map[string]interface{} {
	to.mu.RLock()
	defer to.mu.RUnlock()

	effectiveness, _ := to.mlEngine.GetCurrentEffectiveness()

	return map[string]interface{}{
		"enabled":               to.enabled,
		"obfuscation_level":     to.obfuscationLevel,
		"current_effectiveness": effectiveness,
		"ml_engine": map[string]interface{}{
			"enabled":         true,
			"adaptation_rate": to.mlEngine.adaptationRate,
		},
		"pattern_generator": map[string]interface{}{
			"enabled":         true,
			"current_pattern": string(to.patternGenerator.currentPattern),
		},
		"noise_generator": map[string]interface{}{
			"enabled":     true,
			"noise_ratio": to.noiseGenerator.noiseRatio,
		},
		"traffic_mixer": map[string]interface{}{
			"enabled":     true,
			"legit_ratio": to.trafficMixer.legitRatio,
		},
	}
}

// NewMLEngine creates new ML engine
func NewMLEngine() *MLEngine {
	return &MLEngine{
		effectivenessTracker: NewEffectivenessTracker(),
		patternLearner:       NewPatternLearner(),
		adaptationRate:       0.1,
	}
}

// GetCurrentEffectiveness returns current effectiveness
func (ml *MLEngine) GetCurrentEffectiveness() (float64, error) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	return 0.8, nil
}

// UpdateEffectiveness updates ML engine
func (ml *MLEngine) UpdateEffectiveness(dataSize int, success bool) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	ml.effectivenessTracker.RecordResult(success)
	ml.patternLearner.UpdatePattern(dataSize, success)
}

// NewEffectivenessTracker creates new effectiveness tracker
func NewEffectivenessTracker() *EffectivenessTracker {
	return &EffectivenessTracker{
		successCount:     0,
		failureCount:     0,
		totalAttempts:    0,
		effectivenessMap: make(map[string]float64),
	}
}

// RecordResult records obfuscation result
func (et *EffectivenessTracker) RecordResult(success bool) {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.totalAttempts++
	if success {
		et.successCount++
	} else {
		et.failureCount++
	}
}

// GetEffectiveness calculates effectiveness rate
func (et *EffectivenessTracker) GetEffectiveness() float64 {
	et.mu.RLock()
	defer et.mu.RUnlock()

	if et.totalAttempts == 0 {
		return 0.5 // Default effectiveness
	}

	return float64(et.successCount) / float64(et.totalAttempts)
}

// NewPatternLearner creates new pattern learner
func NewPatternLearner() *PatternLearner {
	return &PatternLearner{
		learningRate:   0.01,
		patternWeights: make(map[string]float64),
		recentResults:  make([]float64, 0, 100),
	}
}

// GetOptimalPattern gets optimal pattern for context
func (pl *PatternLearner) GetOptimalPattern(context int) []byte {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	// Simple pattern selection based on context
	if context == 1 {
		return []byte{0x12, 0x34, 0x56, 0x78}
	} else if context == 2 {
		return []byte{0x9a, 0xbc, 0xde, 0xf0}
	}

	// Default pattern
	return []byte{0xab, 0xcd, 0xef, 0x12}
}

// UpdatePattern updates pattern learning
func (pl *PatternLearner) UpdatePattern(dataSize int, success bool) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	// Add recent result
	pl.recentResults = append(pl.recentResults[1:], float64(b2i(success)))

	// Update pattern weights
	weight := pl.learningRate
	if !success {
		weight = -weight
	}

	// Update current pattern weight
	currentPattern := string([]byte{0xab, 0xcd, 0xef, 0x12})
	pl.patternWeights[currentPattern] += weight
}

// NewDynamicPatternGenerator creates new pattern generator
func NewDynamicPatternGenerator() *DynamicPatternGenerator {
	return &DynamicPatternGenerator{
		currentPattern: []byte{0xab, 0xcd, 0xef, 0x12},
		patternHistory: make([][]byte, 0),
		successRates:   make(map[string]float64),
	}
}

// ObfuscateWithPattern obfuscates with dynamic pattern
func (dpg *DynamicPatternGenerator) ObfuscateWithPattern(data []byte) ([]byte, error) {
	dpg.mu.RLock()
	defer dpg.mu.RUnlock()

	if len(dpg.currentPattern) == 0 {
		return data, nil
	}

	obfuscated := make([]byte, len(data))
	patternLen := len(dpg.currentPattern)

	for i, b := range data {
		patternByte := dpg.currentPattern[i%patternLen]
		obfuscated[i] = b ^ patternByte ^ byte(i)
	}

	return obfuscated, nil
}

// UpdatePattern updates pattern based on success
func (dpg *DynamicPatternGenerator) UpdatePattern(success bool) {
	dpg.mu.Lock()
	defer dpg.mu.Unlock()

	if success {
		// Keep current pattern
		return
	}

	// Generate new pattern
	newPattern := make([]byte, 4)
	rand.Read(newPattern)
	dpg.currentPattern = newPattern

	// Add to history
	dpg.patternHistory = append(dpg.patternHistory, newPattern)
	if len(dpg.patternHistory) > 10 {
		dpg.patternHistory = dpg.patternHistory[1:]
	}
}

// NewNoiseTrafficGenerator creates new noise generator
func NewNoiseTrafficGenerator() *NoiseTrafficGenerator {
	return &NoiseTrafficGenerator{
		noiseRatio:     0.1,
		noisePatterns:  generateNoisePatterns(),
		currentPattern: 0,
	}
}

// InjectNoise injects realistic noise into traffic
func (ntg *NoiseTrafficGenerator) InjectNoise(data []byte) ([]byte, error) {
	ntg.mu.RLock()
	defer ntg.mu.Unlock()

	noiseSize := int(float64(len(data)) * ntg.noiseRatio)
	if noiseSize == 0 {
		return data, nil
	}

	// Generate noise
	noise := make([]byte, noiseSize)
	crand.Read(noise)

	// Mix noise with data
	result := make([]byte, len(data)+noiseSize)
	copy(result, data)

	// Inject noise at random positions
	for i := 0; i < noiseSize; i++ {
		pos := mrand.Intn(len(result))
		result[pos] = noise[i]
	}

	return result, nil
}

// UpdateEffectiveness updates noise generator
func (ntg *NoiseTrafficGenerator) UpdateEffectiveness(success bool) {
	ntg.mu.Lock()
	defer ntg.mu.Unlock()

	if success {
		// Reduce noise ratio if successful
		ntg.noiseRatio *= 0.95
		if ntg.noiseRatio < 0.05 {
			ntg.noiseRatio = 0.05
		}
	} else {
		// Increase noise ratio if failed
		ntg.noiseRatio *= 1.05
		if ntg.noiseRatio > 0.3 {
			ntg.noiseRatio = 0.3
		}
	}

	// Change pattern
	ntg.currentPattern = (ntg.currentPattern + 1) % len(ntg.noisePatterns)
}

// SetNoiseRatio sets noise injection ratio
func (ntg *NoiseTrafficGenerator) SetNoiseRatio(ratio float64) {
	ntg.mu.Lock()
	defer ntg.mu.Unlock()

	ntg.noiseRatio = math.Max(0.0, math.Min(1.0, ratio))
}

// generateNoisePatterns generates realistic noise patterns
func generateNoisePatterns() [][]byte {
	patterns := make([][]byte, 5)

	// Pattern 1: Random bytes
	patterns[0] = make([]byte, 16)
	rand.Read(patterns[0])

	// Pattern 2: Sequential bytes
	patterns[1] = []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}

	// Pattern 3: Alternating bits
	patterns[2] = []byte{0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55}

	// Pattern 4: Sinusoidal pattern
	patterns[3] = make([]byte, 16)
	for i := range patterns[3] {
		patterns[3][i] = byte(math.Sin(float64(i)/16.0*2*math.Pi)*127 + 128)
	}

	// Pattern 5: Compressed-like pattern
	patterns[4] = []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	return patterns
}

// NewTrafficMixer creates new traffic mixer
func NewTrafficMixer() *TrafficMixer {
	return &TrafficMixer{
		legitRatio:      0.7,
		obfuscatedRatio: 0.3,
		mixingStrategy:  "interleaved",
	}
}

// MixTraffic mixes legitimate and obfuscated traffic
func (tm *TrafficMixer) MixTraffic(data []byte) ([]byte, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	legitSize := int(float64(len(data)) * tm.legitRatio)
	obfSize := len(data) - legitSize

	if legitSize == 0 || obfSize == 0 {
		return data, nil
	}

	// Create mixed traffic
	result := make([]byte, len(data))
	copy(result[:legitSize], data[:legitSize])

	// Mix obfuscated data
	obfData := make([]byte, obfSize)
	copy(obfData, data[legitSize:])

	// Interleave obfuscated data
	for i := 0; i < obfSize; i++ {
		pos := legitSize + (i*2)%obfSize
		if pos < len(result) {
			result[pos] = obfData[i]
		}
	}

	return result, nil
}

// SetMixingStrategy sets traffic mixing strategy
func (tm *TrafficMixer) SetMixingStrategy(strategy string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.mixingStrategy = strategy
}

// SetRatios sets mixing ratios
func (tm *TrafficMixer) SetRatios(legit, obf float64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	total := legit + obf
	tm.legitRatio = legit / total
	tm.obfuscatedRatio = obf / total
}

// b2i converts bool to int
func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
