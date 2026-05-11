package behavioral

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// TimingEngine implements advanced timing randomization for behavioral evasion
type TimingEngine struct {
	rand            *rand.Rand
	mu              sync.RWMutex
	patterns        map[string]*TimingPattern
	activePattern   string
	learningEnabled bool
	stats           *TimingStats
}

// TimingPattern represents a timing behavior pattern
type TimingPattern struct {
	Name          string
	BaseDelay     time.Duration
	JitterRange   time.Duration
	BurstChance   float64
	StreamChance  float64
	APICallChance float64
	ImageChance   float64
	LastUpdate    time.Time
	Effectiveness float64
}

// TimingStats tracks timing effectiveness
type TimingStats struct {
	PatternUsage   map[string]int
	SuccessRates   map[string]float64
	AverageLatency map[string]time.Duration
	TotalRequests  int64
	mu             sync.RWMutex
}

// NewTimingEngine creates new timing engine
func NewTimingEngine() *TimingEngine {
	return &TimingEngine{
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
		patterns:        make(map[string]*TimingPattern),
		learningEnabled: true,
		stats: &TimingStats{
			PatternUsage:   make(map[string]int),
			SuccessRates:   make(map[string]float64),
			AverageLatency: make(map[string]time.Duration),
		},
	}
}

// Initialize initializes timing engine with default patterns
func (te *TimingEngine) Initialize() error {
	te.loadDefaultPatterns()
	te.activePattern = "human_like"
	return nil
}

// ApplyTiming applies timing randomization to connection
func (te *TimingEngine) ApplyTiming(ctx context.Context, connectionType string) time.Duration {
	te.mu.RLock()
	defer te.mu.RUnlock()

	pattern, exists := te.patterns[te.activePattern]
	if !exists {
		pattern = te.patterns["human_like"]
	}

	// Calculate base delay with jitter
	baseDelay := pattern.BaseDelay
	jitter := time.Duration(float64(pattern.JitterRange) * (te.rand.Float64()*2 - 1))

	// Apply connection type specific modifications
	finalDelay := te.applyConnectionTypeDelay(baseDelay+jitter, connectionType)

	// Record pattern usage
	te.stats.RecordPatternUsage(te.activePattern)

	return finalDelay
}

// SetPattern sets active timing pattern
func (te *TimingEngine) SetPattern(patternName string) error {
	te.mu.Lock()
	defer te.mu.Unlock()

	if _, exists := te.patterns[patternName]; !exists {
		return ErrPatternNotFound
	}

	te.activePattern = patternName
	return nil
}

// GetActivePattern returns current active pattern
func (te *TimingEngine) GetActivePattern() string {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.activePattern
}

// LearnFromFeedback updates pattern effectiveness based on feedback
func (te *TimingEngine) LearnFromFeedback(patternName string, success bool, latency time.Duration) {
	if !te.learningEnabled {
		return
	}

	te.mu.Lock()
	defer te.mu.Unlock()

	pattern, exists := te.patterns[patternName]
	if !exists {
		return
	}

	// Update pattern effectiveness using exponential moving average
	alpha := 0.1
	if success {
		newEffectiveness := (1-alpha)*pattern.Effectiveness + alpha*1.0
		pattern.Effectiveness = newEffectiveness
	} else {
		newEffectiveness := (1-alpha)*pattern.Effectiveness + alpha*0.0
		pattern.Effectiveness = newEffectiveness
	}

	pattern.LastUpdate = time.Now()

	// Update stats
	te.stats.UpdatePatternStats(patternName, success, latency)
}

// GetRecommendedPattern returns best pattern for given context
func (te *TimingEngine) GetRecommendedPattern(connectionType string) string {
	te.mu.RLock()
	defer te.mu.RUnlock()

	bestPattern := "human_like"
	bestScore := 0.0

	for name, pattern := range te.patterns {
		score := te.calculatePatternScore(pattern, connectionType)
		if score > bestScore {
			bestScore = score
			bestPattern = name
		}
	}

	return bestPattern
}

// loadDefaultPatterns loads built-in timing patterns
func (te *TimingEngine) loadDefaultPatterns() {
	// Human-like pattern - mimics human browsing behavior
	te.patterns["human_like"] = &TimingPattern{
		Name:          "human_like",
		BaseDelay:     150 * time.Millisecond,
		JitterRange:   100 * time.Millisecond,
		BurstChance:   0.1,
		StreamChance:  0.05,
		APICallChance: 0.15,
		ImageChance:   0.2,
		Effectiveness: 0.8,
		LastUpdate:    time.Now(),
	}

	// Conservative pattern - minimal timing changes
	te.patterns["conservative"] = &TimingPattern{
		Name:          "conservative",
		BaseDelay:     50 * time.Millisecond,
		JitterRange:   25 * time.Millisecond,
		BurstChance:   0.02,
		StreamChance:  0.01,
		APICallChance: 0.05,
		ImageChance:   0.03,
		Effectiveness: 0.6,
		LastUpdate:    time.Now(),
	}

	// Aggressive pattern - maximum randomization
	te.patterns["aggressive"] = &TimingPattern{
		Name:          "aggressive",
		BaseDelay:     300 * time.Millisecond,
		JitterRange:   500 * time.Millisecond,
		BurstChance:   0.3,
		StreamChance:  0.2,
		APICallChance: 0.4,
		ImageChance:   0.35,
		Effectiveness: 0.7,
		LastUpdate:    time.Now(),
	}

	// Burst-like pattern - mimics burst traffic
	te.patterns["burst_like"] = &TimingPattern{
		Name:          "burst_like",
		BaseDelay:     10 * time.Millisecond,
		JitterRange:   50 * time.Millisecond,
		BurstChance:   0.8,
		StreamChance:  0.6,
		APICallChance: 0.3,
		ImageChance:   0.4,
		Effectiveness: 0.75,
		LastUpdate:    time.Now(),
	}

	// Stream-like pattern - mimics streaming behavior
	te.patterns["stream_like"] = &TimingPattern{
		Name:          "stream_like",
		BaseDelay:     200 * time.Millisecond,
		JitterRange:   150 * time.Millisecond,
		BurstChance:   0.05,
		StreamChance:  0.7,
		APICallChance: 0.1,
		ImageChance:   0.6,
		Effectiveness: 0.8,
		LastUpdate:    time.Now(),
	}

	// API-like pattern - mimics API calls
	te.patterns["api_like"] = &TimingPattern{
		Name:          "api_like",
		BaseDelay:     100 * time.Millisecond,
		JitterRange:   75 * time.Millisecond,
		BurstChance:   0.1,
		StreamChance:  0.05,
		APICallChance: 0.8,
		ImageChance:   0.1,
		Effectiveness: 0.7,
		LastUpdate:    time.Now(),
	}

	// Image-like pattern - mimics image loading
	te.patterns["image_like"] = &TimingPattern{
		Name:          "image_like",
		BaseDelay:     250 * time.Millisecond,
		JitterRange:   200 * time.Millisecond,
		BurstChance:   0.15,
		StreamChance:  0.1,
		APICallChance: 0.2,
		ImageChance:   0.9,
		Effectiveness: 0.75,
		LastUpdate:    time.Now(),
	}
}

// applyConnectionTypeDelay applies connection type specific delays
func (te *TimingEngine) applyConnectionTypeDelay(baseDelay time.Duration, connectionType string) time.Duration {
	switch connectionType {
	case "stream":
		// Add streaming-specific delays
		return baseDelay + time.Duration(te.rand.Float64()*50)*time.Millisecond
	case "api":
		// Add API call delays
		return baseDelay + time.Duration(te.rand.Float64()*30)*time.Millisecond
	case "image":
		// Add image loading delays
		return baseDelay + time.Duration(te.rand.Float64()*100)*time.Millisecond
	default:
		return baseDelay
	}
}

// calculatePatternScore calculates effectiveness score for pattern
func (te *TimingEngine) calculatePatternScore(pattern *TimingPattern, connectionType string) float64 {
	baseScore := pattern.Effectiveness

	// Adjust score based on connection type
	switch connectionType {
	case "stream":
		return baseScore * pattern.StreamChance
	case "api":
		return baseScore * pattern.APICallChance
	case "image":
		return baseScore * pattern.ImageChance
	default:
		return baseScore
	}
}

// RecordPatternUsage records pattern usage in stats
func (ts *TimingStats) RecordPatternUsage(patternName string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.PatternUsage[patternName]++
	ts.TotalRequests++
}

// UpdatePatternStats updates pattern statistics
func (ts *TimingStats) UpdatePatternStats(patternName string, success bool, latency time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.SuccessRates[patternName] == 0 {
		ts.SuccessRates[patternName] = 0
	}

	// Update success rate
	alpha := 0.1
	if success {
		ts.SuccessRates[patternName] = (1-alpha)*ts.SuccessRates[patternName] + alpha*1.0
	} else {
		ts.SuccessRates[patternName] = (1-alpha)*ts.SuccessRates[patternName] + alpha*0.0
	}

	// Update average latency
	if ts.AverageLatency[patternName] == 0 {
		ts.AverageLatency[patternName] = latency
	} else {
		// Convert both to milliseconds for calculation, then back
		avgMs := float64(ts.AverageLatency[patternName].Milliseconds())
		latencyMs := float64(latency.Milliseconds())
		newAvgMs := (1-alpha)*avgMs + alpha*latencyMs
		ts.AverageLatency[patternName] = time.Duration(newAvgMs) * time.Millisecond
	}
}

// GetStats returns current timing statistics
func (te *TimingEngine) GetStats() *TimingStats {
	return te.stats
}

// Timing errors
var (
	ErrPatternNotFound = &TimingError{
		Code:    "PATTERN_NOT_FOUND",
		Message: "Timing pattern not found",
	}
)

// TimingError represents timing engine errors
type TimingError struct {
	Code    string
	Message string
}

func (e *TimingError) Error() string {
	return e.Message
}
