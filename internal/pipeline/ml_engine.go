package pipeline

import (
	"fmt"
	"strings"
	"time"
)

// DPIType represents different types of DPI systems
type DPIType int

const (
	DPITypeUnknown DPIType = iota
	DPITypeSignature
	DPITypeBehavioral
	DPITypeMLBased
	DPITypeHybrid
)

// DPIResult represents the result of DPI detection
type DPIResult struct {
	Type       DPIType
	Confidence float64
	Techniques []string
	LastUpdate time.Time
}

// TechniqueEffectiveness represents effectiveness of bypass techniques
type TechniqueEffectiveness struct {
	Technique     string
	Effectiveness float64
	SampleCount   int
	LastUpdate    time.Time
}

// DPIDetector interface for DPI detection and technique prediction
type DPIDetector interface {
	DetectDPIType(data []byte) DPIResult
	PredictEffectiveness(domain, technique string) float64
	LearnFromFeedback(domain, technique string, success bool)
	GetRecommendedTechniques(domain string) []string
}

// SimpleDPIDetector implements a basic DPI detector
type SimpleDPIDetector struct {
	techniqueCache map[string]*TechniqueEffectiveness
	lastUpdate     time.Time
}

// NewSimpleDPIDetector creates a new simple DPI detector
func NewSimpleDPIDetector() *SimpleDPIDetector {
	detector := &SimpleDPIDetector{
		techniqueCache: make(map[string]*TechniqueEffectiveness),
		lastUpdate:     time.Now(),
	}

	// Initialize with default effectiveness values
	defaultTechniques := map[string]float64{
		"fragmentation": 0.8,
		"headers":       0.7,
		"encryption":    0.9,
		"protocol_mask": 0.6,
		"adaptive_frag": 0.85,
		"timing_random": 0.75,
	}

	for technique, effectiveness := range defaultTechniques {
		detector.techniqueCache[technique] = &TechniqueEffectiveness{
			Technique:     technique,
			Effectiveness: effectiveness,
			SampleCount:   1,
			LastUpdate:    time.Now(),
		}
	}

	return detector
}

// DetectDPIType implements DPIDetector interface
func (s *SimpleDPIDetector) DetectDPIType(data []byte) DPIResult {
	// Simple heuristic-based DPI detection
	dpiType := s.heuristicDetection(data)
	confidence := 0.8 // Default confidence

	return DPIResult{
		Type:       dpiType,
		Confidence: confidence,
		Techniques: s.getTechniquesForDPIType(dpiType),
		LastUpdate: time.Now(),
	}
}

// PredictEffectiveness implements DPIDetector interface
func (s *SimpleDPIDetector) PredictEffectiveness(domain, technique string) float64 {
	if eff, exists := s.techniqueCache[technique]; exists {
		return eff.Effectiveness
	}

	// Default effectiveness based on technique
	defaultEffectiveness := map[string]float64{
		"fragmentation": 0.8,
		"headers":       0.7,
		"encryption":    0.9,
		"protocol_mask": 0.6,
		"adaptive_frag": 0.85,
		"timing_random": 0.75,
	}

	if eff, exists := defaultEffectiveness[technique]; exists {
		return eff
	}

	return 0.5 // Default
}

// LearnFromFeedback implements DPIDetector interface
func (s *SimpleDPIDetector) LearnFromFeedback(domain, technique string, success bool) {
	// Update effectiveness based on feedback
	effectiveness := 0.5
	if success {
		effectiveness = 0.9
	} else {
		effectiveness = 0.2
	}

	if existing, exists := s.techniqueCache[technique]; exists {
		// Weighted average
		weight := float64(existing.SampleCount) / float64(existing.SampleCount+1)
		existing.Effectiveness = existing.Effectiveness*weight + effectiveness*(1-weight)
		existing.SampleCount++
		existing.LastUpdate = time.Now()
	} else {
		s.techniqueCache[technique] = &TechniqueEffectiveness{
			Technique:     technique,
			Effectiveness: effectiveness,
			SampleCount:   1,
			LastUpdate:    time.Now(),
		}
	}

	s.lastUpdate = time.Now()
}

// GetRecommendedTechniques implements DPIDetector interface
func (s *SimpleDPIDetector) GetRecommendedTechniques(domain string) []string {
	// Return techniques sorted by effectiveness
	techniques := make([]string, 0, len(s.techniqueCache))

	type techniqueScore struct {
		name  string
		score float64
	}

	scores := make([]techniqueScore, 0, len(s.techniqueCache))
	for name, eff := range s.techniqueCache {
		scores = append(scores, techniqueScore{
			name:  name,
			score: eff.Effectiveness,
		})
	}

	// Sort by effectiveness (descending)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	for _, score := range scores {
		techniques = append(techniques, score.name)
	}

	return techniques
}

// heuristicDetection performs simple heuristic DPI detection
func (s *SimpleDPIDetector) heuristicDetection(data []byte) DPIType {
	// Simple heuristics for DPI detection
	dataStr := string(data)

	// Check for common DPI signatures
	if containsDPIIndicators(dataStr) {
		return DPITypeSignature
	}

	// Check for behavioral patterns
	if hasBehavioralPatterns(data) {
		return DPITypeBehavioral
	}

	// Default to unknown
	return DPITypeUnknown
}

// getTechniquesForDPIType returns techniques for DPI type
func (s *SimpleDPIDetector) getTechniquesForDPIType(dpiType DPIType) []string {
	switch dpiType {
	case DPITypeSignature:
		return []string{"encryption", "protocol_mask", "fragmentation"}
	case DPITypeBehavioral:
		return []string{"fragmentation", "headers", "timing_random"}
	case DPITypeMLBased:
		return []string{"encryption", "adaptive_frag", "behavioral_evasion"}
	default:
		return []string{"encryption", "fragmentation", "headers"}
	}
}

// containsDPIIndicators checks for DPI signature indicators
func containsDPIIndicators(data string) bool {
	indicators := []string{
		"blocked",
		"forbidden",
		"access denied",
		"connection reset",
		"timeout",
		"403",
		"connection reset",
	}

	data = strings.ToLower(data)
	for _, indicator := range indicators {
		if strings.Contains(data, strings.ToLower(indicator)) {
			return true
		}
	}

	return false
}

// hasBehavioralPatterns checks for behavioral DPI patterns
func hasBehavioralPatterns(data []byte) bool {
	// Simple behavioral analysis
	// Check for consistent packet sizes (potential DPI throttling)
	if len(data) > 0 {
		// This is a simplified check
		return false
	}

	return false
}

// MLPipelineEngine integrates ML with DPI bypass pipeline
type MLPipelineEngine struct {
	detector        DPIDetector
	learningEnabled bool
	lastUpdate      time.Time
}

// NewMLPipelineEngine creates a new ML-enhanced pipeline engine
func NewMLPipelineEngine() *MLPipelineEngine {
	detector := NewSimpleDPIDetector()

	return &MLPipelineEngine{
		detector:        detector,
		learningEnabled: true,
		lastUpdate:      time.Now(),
	}
}

// AnalyzeTraffic analyzes traffic and provides ML insights
func (mle *MLPipelineEngine) AnalyzeTraffic(data []byte, domain string) (*TrafficAnalysis, error) {
	if mle.detector == nil {
		return nil, fmt.Errorf("DPI detector not initialized")
	}

	// Detect DPI type
	dpiResult := mle.detector.DetectDPIType(data)

	// Get recommended techniques
	techniques := mle.detector.GetRecommendedTechniques(domain)

	// Predict effectiveness for each technique
	effectiveness := make(map[string]float64)
	for _, technique := range techniques {
		effectiveness[technique] = mle.detector.PredictEffectiveness(domain, technique)
	}

	return &TrafficAnalysis{
		DPIType:       dpiResult.Type,
		Confidence:    dpiResult.Confidence,
		Techniques:    techniques,
		Effectiveness: effectiveness,
		Timestamp:     time.Now(),
	}, nil
}

// UpdateLearning updates ML model with feedback
func (mle *MLPipelineEngine) UpdateLearning(domain, technique string, success bool) error {
	if !mle.learningEnabled || mle.detector == nil {
		return nil
	}

	mle.detector.LearnFromFeedback(domain, technique, success)
	mle.lastUpdate = time.Now()

	return nil
}

// GetRecommendedTechniques returns recommended techniques for given domain
func (mle *MLPipelineEngine) GetRecommendedTechniques(domain string) []string {
	if mle.detector == nil {
		return []string{"fragmentation", "encryption", "headers"} // Fallback
	}

	return mle.detector.GetRecommendedTechniques(domain)
}

// GetOptimalTechnique returns the best technique for given domain
func (mle *MLPipelineEngine) GetOptimalTechnique(domain string) (string, float64) {
	if mle.detector == nil {
		return "fragmentation", 0.7 // Fallback
	}

	techniques := mle.detector.GetRecommendedTechniques(domain)
	if len(techniques) == 0 {
		return "fragmentation", 0.7 // Fallback
	}

	bestTechnique := techniques[0]
	bestEffectiveness := mle.detector.PredictEffectiveness(domain, bestTechnique)

	return bestTechnique, bestEffectiveness
}

// ShouldUseMLPipeline determines if ML should be used for this domain
func (mle *MLPipelineEngine) ShouldUseMLPipeline(domain string) bool {
	// Use ML for known problematic domains
	problematicDomains := []string{
		"youtube.com",
		"facebook.com",
		"twitter.com",
		"instagram.com",
		"tiktok.com",
		"reddit.com",
	}

	for _, pd := range problematicDomains {
		if domain == pd {
			return true
		}
	}

	return false
}

// TrafficAnalysis represents ML analysis results
type TrafficAnalysis struct {
	DPIType       DPIType
	Confidence    float64
	Techniques    []string
	Effectiveness map[string]float64
	Timestamp     time.Time
}
