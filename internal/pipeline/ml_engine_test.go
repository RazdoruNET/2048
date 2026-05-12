package pipeline

import (
	"bytes"
	"testing"
)

func TestNewSimpleDPIDetector(t *testing.T) {
	detector := NewSimpleDPIDetector()

	if detector == nil {
		t.Error("Expected detector to be created")
	}

	if len(detector.techniqueCache) == 0 {
		t.Error("Expected technique cache to be initialized")
	}
}

func TestSimpleDPIDetector_DetectDPIType(t *testing.T) {
	detector := NewSimpleDPIDetector()

	tests := []struct {
		name     string
		data     []byte
		expected DPIType
	}{
		{
			name:     "Normal HTTP data",
			data:     []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
			expected: DPITypeUnknown,
		},
		{
			name:     "Blocked response",
			data:     []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/html\r\n\r\n"),
			expected: DPITypeSignature,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: DPITypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectDPIType(tt.data)

			if result.Type != tt.expected {
				t.Errorf("Expected DPI type %v, got %v", tt.expected, result.Type)
			}

			if result.Confidence <= 0 {
				t.Error("Expected confidence > 0")
			}

			if len(result.Techniques) == 0 {
				t.Error("Expected techniques to be suggested")
			}
		})
	}
}

func TestSimpleDPIDetector_PredictEffectiveness(t *testing.T) {
	detector := NewSimpleDPIDetector()

	tests := []struct {
		technique string
		expected  float64
	}{
		{"encryption", 0.9},
		{"fragmentation", 0.8},
		{"headers", 0.7},
		{"protocol_mask", 0.6},
		{"unknown", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.technique, func(t *testing.T) {
			effectiveness := detector.PredictEffectiveness("example.com", tt.technique)

			if effectiveness != tt.expected {
				t.Errorf("Expected effectiveness %f, got %f", tt.expected, effectiveness)
			}
		})
	}
}

func TestSimpleDPIDetector_LearnFromFeedback(t *testing.T) {
	detector := NewSimpleDPIDetector()

	// Test successful feedback
	detector.LearnFromFeedback("example.com", "encryption", true)

	eff := detector.techniqueCache["encryption"]
	if eff == nil {
		t.Error("Expected technique to be cached")
	}

	if eff.Effectiveness < 0.8 {
		t.Errorf("Expected effectiveness >= 0.8 after success, got %f", eff.Effectiveness)
	}

	// Test failure feedback
	initialEffectiveness := eff.Effectiveness
	detector.LearnFromFeedback("example.com", "encryption", false)

	if eff.Effectiveness >= initialEffectiveness {
		t.Error("Expected effectiveness to decrease after failure")
	}

	if eff.SampleCount < 2 {
		t.Error("Expected sample count to increase")
	}
}

func TestSimpleDPIDetector_GetRecommendedTechniques(t *testing.T) {
	detector := NewSimpleDPIDetector()

	// Add some techniques with different effectiveness
	detector.LearnFromFeedback("example.com", "encryption", true)
	detector.LearnFromFeedback("example.com", "fragmentation", false)
	detector.LearnFromFeedback("example.com", "headers", true)

	techniques := detector.GetRecommendedTechniques("example.com")

	if len(techniques) == 0 {
		t.Error("Expected techniques to be returned")
	}

	// Should be sorted by effectiveness (encryption first, then headers)
	if techniques[0] != "encryption" {
		t.Errorf("Expected first technique to be encryption, got %s", techniques[0])
	}
}

func TestNewMLPipelineEngine(t *testing.T) {
	engine := NewMLPipelineEngine()

	if engine == nil {
		t.Error("Expected engine to be created")
	}

	if engine.detector == nil {
		t.Error("Expected detector to be initialized")
	}

	if !engine.learningEnabled {
		t.Error("Expected learning to be enabled")
	}
}

func TestMLPipelineEngine_AnalyzeTraffic(t *testing.T) {
	engine := NewMLPipelineEngine()

	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	domain := "example.com"

	analysis, err := engine.AnalyzeTraffic(data, domain)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if analysis == nil {
		t.Error("Expected analysis to be returned")
		return
	}

	if analysis.DPIType == DPITypeUnknown {
		// This is expected for normal traffic
	} else {
		t.Errorf("Expected DPI type Unknown, got %v", analysis.DPIType)
	}

	if len(analysis.Techniques) == 0 {
		t.Error("Expected techniques to be suggested")
	}

	if len(analysis.Effectiveness) == 0 {
		t.Error("Expected effectiveness map to be populated")
	}
}

func TestMLPipelineEngine_UpdateLearning(t *testing.T) {
	engine := NewMLPipelineEngine()

	// Test successful learning
	err := engine.UpdateLearning("example.com", "encryption", true)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test with learning disabled
	engine.learningEnabled = false
	err = engine.UpdateLearning("example.com", "encryption", true)
	if err != nil {
		t.Errorf("Unexpected error with disabled learning: %v", err)
	}
}

func TestMLPipelineEngine_GetOptimalTechnique(t *testing.T) {
	engine := NewMLPipelineEngine()

	// Test with default cache
	technique, effectiveness := engine.GetOptimalTechnique("example.com")

	if technique == "" {
		t.Error("Expected technique to be returned")
	}

	if effectiveness <= 0 {
		t.Error("Expected effectiveness > 0")
	}

	// Test with problematic domain
	technique, effectiveness = engine.GetOptimalTechnique("youtube.com")

	if technique == "" {
		t.Error("Expected technique to be returned for problematic domain")
	}
}

func TestMLPipelineEngine_ShouldUseMLPipeline(t *testing.T) {
	engine := NewMLPipelineEngine()

	tests := []struct {
		domain   string
		expected bool
	}{
		{"youtube.com", true},
		{"facebook.com", true},
		{"twitter.com", true},
		{"example.com", false},
		{"google.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			shouldUse := engine.ShouldUseMLPipeline(tt.domain)

			if shouldUse != tt.expected {
				t.Errorf("Expected %v for domain %s, got %v", tt.expected, tt.domain, shouldUse)
			}
		})
	}
}

func TestAdaptiveFragmentationModifier(t *testing.T) {
	modifier := NewAdaptiveFragmentationModifier(50, 200, true)

	if modifier.Name() != "adaptive_fragmentation" {
		t.Errorf("Expected name 'adaptive_fragmentation', got '%s'", modifier.Name())
	}

	// Test configuration
	config := map[string]interface{}{
		"min_size": 100,
		"max_size": 300,
		"random":   false,
	}

	err := modifier.Configure(config)
	if err != nil {
		t.Errorf("Unexpected configuration error: %v", err)
	}

	if modifier.MinSize != 100 {
		t.Errorf("Expected min_size 100, got %d", modifier.MinSize)
	}

	if modifier.MaxSize != 300 {
		t.Errorf("Expected max_size 300, got %d", modifier.MaxSize)
	}

	if modifier.RandomSizes != false {
		t.Errorf("Expected random false, got %v", modifier.RandomSizes)
	}
}

func TestAdaptiveFragmentationModifier_Process(t *testing.T) {
	modifier := NewAdaptiveFragmentationModifier(50, 150, true)

	data := []byte("This is a test message that should be fragmented into multiple pieces")

	// Test outbound fragmentation - Process() should be NO-OP for fragmenting modifiers
	result := modifier.Process(data, DirectionOutbound)

	// Process() should return original data for fragmenting modifiers (NO-OP behavior)
	if !bytes.Equal(result, data) {
		t.Errorf("Process() should return original data for fragmenting modifiers, got %d bytes vs %d expected", len(result), len(data))
	}

	// Test inbound (should return original)
	result = modifier.Process(data, DirectionInbound)

	if !bytes.Equal(result, data) {
		t.Error("Expected original data for inbound direction")
	}

	// Test empty data
	result = modifier.Process([]byte{}, DirectionOutbound)

	if len(result) != 0 {
		t.Error("Expected empty result for empty input")
	}
}

func TestContainsDPIIndicators(t *testing.T) {
	tests := []struct {
		data     string
		expected bool
	}{
		{"This page is blocked by DPI", true},
		{"Access forbidden by network policy", true},
		{"Normal webpage content", false},
		{"Connection reset by peer", true},
		{"Regular HTTP response", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.data, func(t *testing.T) {
			result := containsDPIIndicators(tt.data)

			if result != tt.expected {
				t.Errorf("Expected %v for '%s', got %v", tt.expected, tt.data, result)
			}
		})
	}
}

func TestHasBehavioralPatterns(t *testing.T) {
	tests := []struct {
		data     []byte
		expected bool
	}{
		{[]byte("Normal HTTP request"), false},
		{[]byte{}, false},
		{[]byte("Repeated pattern data"), false}, // Simplified test
	}

	for _, tt := range tests {
		t.Run(string(tt.data), func(t *testing.T) {
			result := hasBehavioralPatterns(tt.data)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFindSubstring(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "test", false},
		{"", "test", false},
		{"test", "", true}, // Empty substring should match at position 0
		{"abc", "c", true},
		{"abc", "d", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			result := findSubstring(tt.s, tt.substr)

			if result != tt.expected {
				t.Errorf("Expected %v for '%s' in '%s', got %v", tt.expected, tt.substr, tt.s, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSimpleDPIDetector_DetectDPIType(b *testing.B) {
	detector := NewSimpleDPIDetector()
	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectDPIType(data)
	}
}

func BenchmarkMLPipelineEngine_AnalyzeTraffic(b *testing.B) {
	engine := NewMLPipelineEngine()
	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	domain := "example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.AnalyzeTraffic(data, domain)
	}
}

func BenchmarkAdaptiveFragmentationModifier_Process(b *testing.B) {
	modifier := NewAdaptiveFragmentationModifier(50, 150, true)
	data := []byte("This is a test message that should be fragmented into multiple pieces")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		modifier.Process(data, DirectionOutbound)
	}
}

// Helper functions for tests

func findSubstring(s, substr string) bool {
	if substr == "" {
		return true // Empty substring matches at position 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
