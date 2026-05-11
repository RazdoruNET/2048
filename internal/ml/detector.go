package ml

import (
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
	// DetectDPIType analyzes traffic data to determine DPI type
	DetectDPIType(data []byte) DPIResult

	// PredictEffectiveness predicts bypass technique effectiveness
	PredictEffectiveness(domain, technique string) float64

	// LearnFromFeedback updates model based on success/failure
	LearnFromFeedback(domain, technique string, success bool)

	// GetRecommendedTechniques returns recommended techniques for given domain
	GetRecommendedTechniques(domain string) []string

	// UpdateModel updates the ML model with new data
	UpdateModel(trainingData []TrainingSample) error
}

// TrainingSample represents a training data sample
type TrainingSample struct {
	Features      []float64
	DPIType       DPIType
	Technique     string
	Effectiveness float64
	Success       bool
	Timestamp     time.Time
}

// BaseDPIDetector provides common functionality for DPI detectors
type BaseDPIDetector struct {
	techniqueCache map[string]*TechniqueEffectiveness
	lastUpdate     time.Time
}

// NewBaseDPIDetector creates a new base DPI detector
func NewBaseDPIDetector() *BaseDPIDetector {
	return &BaseDPIDetector{
		techniqueCache: make(map[string]*TechniqueEffectiveness),
		lastUpdate:     time.Now(),
	}
}

// GetRecommendedTechniques returns techniques sorted by effectiveness
func (b *BaseDPIDetector) GetRecommendedTechniques(domain string) []string {
	techniques := make([]string, 0, len(b.techniqueCache))

	// Sort techniques by effectiveness
	type techniqueScore struct {
		name  string
		score float64
	}

	scores := make([]techniqueScore, 0, len(b.techniqueCache))
	for name, eff := range b.techniqueCache {
		scores = append(scores, techniqueScore{
			name:  name,
			score: eff.Effectiveness,
		})
	}

	// Simple sort by effectiveness (descending)
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

// updateTechniqueEffectiveness updates cached effectiveness
func (b *BaseDPIDetector) updateTechniqueEffectiveness(technique string, effectiveness float64) {
	if existing, exists := b.techniqueCache[technique]; exists {
		// Weighted average with new data
		weight := float64(existing.SampleCount) / float64(existing.SampleCount+1)
		existing.Effectiveness = existing.Effectiveness*weight + effectiveness*(1-weight)
		existing.SampleCount++
		existing.LastUpdate = time.Now()
	} else {
		b.techniqueCache[technique] = &TechniqueEffectiveness{
			Technique:     technique,
			Effectiveness: effectiveness,
			SampleCount:   1,
			LastUpdate:    time.Now(),
		}
	}
	b.lastUpdate = time.Now()
}
