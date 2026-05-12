package pipeline

import (
	"testing"
	"time"
)

// TestFragmentationModifierV2 tests the enhanced fragmentation modifier
func TestFragmentationModifierV2(t *testing.T) {
	modifier := &FragmentationModifier{}

	// Test configuration
	config := map[string]interface{}{
		"min_size":         64,
		"max_size":         256,
		"random":           true,
		"anti_nagle_delay": 2,
	}

	if err := modifier.Configure(config); err != nil {
		t.Fatalf("Failed to configure modifier: %v", err)
	}

	// Test basic properties
	if modifier.Name() != "fragmentation" {
		t.Errorf("Expected name 'fragmentation', got '%s'", modifier.Name())
	}

	if !modifier.SupportsFragmentation() {
		t.Error("Expected SupportsFragmentation() to return true")
	}

	// Test configuration retrieval
	fragConfig := modifier.GetFragmentationConfig()
	if fragConfig.MinSize != 64 {
		t.Errorf("Expected MinSize 64, got %d", fragConfig.MinSize)
	}

	if fragConfig.MaxSize != 256 {
		t.Errorf("Expected MaxSize 256, got %d", fragConfig.MaxSize)
	}

	if !fragConfig.RandomSizes {
		t.Error("Expected RandomSizes to be true")
	}

	if fragConfig.AntiNagleDelay != 2*time.Millisecond {
		t.Errorf("Expected AntiNagleDelay 2ms, got %v", fragConfig.AntiNagleDelay)
	}
}

// TestFragmentationModifierProcessToChunks tests chunk processing
func TestFragmentationModifierProcessToChunks(t *testing.T) {
	modifier := &FragmentationModifier{}

	config := map[string]interface{}{
		"min_size": 10,
		"max_size": 20,
		"random":   false, // Use deterministic sizes for testing
	}

	if err := modifier.Configure(config); err != nil {
		t.Fatalf("Failed to configure modifier: %v", err)
	}

	// Test data smaller than min_size
	data := []byte("small")
	chunks := modifier.ProcessToChunks(data, DirectionOutbound)

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for small data, got %d", len(chunks))
	}

	if string(chunks[0]) != string(data) {
		t.Errorf("Expected chunk data to match input, got %s", string(chunks[0]))
	}

	// Test inbound direction (should not fragment)
	chunks = modifier.ProcessToChunks(data, DirectionInbound)
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for inbound data, got %d", len(chunks))
	}

	// Test data larger than min_size
	data = []byte("this is a longer test string that should be fragmented")
	chunks = modifier.ProcessToChunks(data, DirectionOutbound)

	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks for large data, got %d", len(chunks))
	}

	// Verify chunks recombine to original data
	var combined []byte
	for _, chunk := range chunks {
		combined = append(combined, chunk...)
	}

	if string(combined) != string(data) {
		t.Error("Combined chunks don't match original data")
	}

	// Verify each chunk size is within bounds
	for _, chunk := range chunks {
		if len(chunk) > modifier.MaxSize {
			t.Errorf("Chunk size %d exceeds MaxSize %d", len(chunk), modifier.MaxSize)
		}
		if len(chunk) < modifier.MinSize && len(chunk) != len(data)%modifier.MaxSize {
			t.Errorf("Chunk size %d below MinSize %d", len(chunk), modifier.MinSize)
		}
	}
}

// TestAdaptiveFragmentationModifierV2 tests the enhanced adaptive fragmentation modifier
func TestAdaptiveFragmentationModifierV2(t *testing.T) {
	modifier := NewAdaptiveFragmentationModifier(50, 200, true)

	// Test basic properties
	if modifier.Name() != "adaptive_fragmentation" {
		t.Errorf("Expected name 'adaptive_fragmentation', got '%s'", modifier.Name())
	}

	if !modifier.SupportsFragmentation() {
		t.Error("Expected SupportsFragmentation() to return true")
	}

	// Test configuration update
	newConfig := &FragmentationConfig{
		MinSize:          100,
		MaxSize:          300,
		RandomSizes:      false,
		AntiNagleDelay:   5 * time.Millisecond,
		MLOptimization:   true,
		AdaptiveStrategy: "ml_based",
	}

	modifier.SetFragmentationConfig(newConfig)

	retrievedConfig := modifier.GetFragmentationConfig()
	if retrievedConfig.MinSize != 100 {
		t.Errorf("Expected MinSize 100, got %d", retrievedConfig.MinSize)
	}

	if retrievedConfig.MaxSize != 300 {
		t.Errorf("Expected MaxSize 300, got %d", retrievedConfig.MaxSize)
	}

	if retrievedConfig.RandomSizes {
		t.Error("Expected RandomSizes to be false")
	}

	if !retrievedConfig.MLOptimization {
		t.Error("Expected MLOptimization to be true")
	}

	if retrievedConfig.AdaptiveStrategy != "ml_based" {
		t.Errorf("Expected AdaptiveStrategy 'ml_based', got '%s'", retrievedConfig.AdaptiveStrategy)
	}
}

// TestAdaptiveFragmentationProcessToChunks tests adaptive chunk processing
func TestAdaptiveFragmentationProcessToChunks(t *testing.T) {
	modifier := NewAdaptiveFragmentationModifier(30, 100, false) // Fixed size for testing

	// Test data
	data := []byte("this is test data for adaptive fragmentation processing")

	// Test outbound fragmentation
	chunks := modifier.ProcessToChunks(data, DirectionOutbound)

	if len(chunks) == 0 {
		t.Error("Expected at least 1 chunk")
	}

	// Verify data integrity
	var combined []byte
	for _, chunk := range chunks {
		combined = append(combined, chunk...)
	}

	if string(combined) != string(data) {
		t.Error("Combined chunks don't match original data")
	}

	// Verify chunk sizes are within bounds
	for i, chunk := range chunks {
		if len(chunk) > modifier.MaxSize {
			t.Errorf("Chunk %d size %d exceeds MaxSize %d", i, len(chunk), modifier.MaxSize)
		}

		// Only check MinSize for chunks that aren't the last one
		if i < len(chunks)-1 && len(chunk) < modifier.MinSize {
			t.Errorf("Chunk %d size %d below MinSize %d", i, len(chunk), modifier.MinSize)
		}
	}

	// Test inbound (should not fragment)
	chunks = modifier.ProcessToChunks(data, DirectionInbound)
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for inbound data, got %d", len(chunks))
	}

	if string(chunks[0]) != string(data) {
		t.Error("Inbound chunk should match original data")
	}
}

// TestLegacyModifierWrapper tests backward compatibility
func TestLegacyModifierWrapper(t *testing.T) {
	// Create a legacy modifier
	legacyMod := &HeadersModifier{}
	legacyMod.Configure(map[string]interface{}{
		"random_headers": true,
	})

	// Wrap it
	wrapper := NewLegacyModifierWrapper(legacyMod)

	// Test basic properties
	if wrapper.Name() != legacyMod.Name() {
		t.Errorf("Expected wrapped name '%s', got '%s'", legacyMod.Name(), wrapper.Name())
	}

	if wrapper.SupportsFragmentation() {
		t.Error("Expected SupportsFragmentation() to return false for legacy wrapper")
	}

	// Test ProcessToChunks (should return single chunk)
	data := []byte("test data")
	chunks := wrapper.ProcessToChunks(data, DirectionOutbound)

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk from legacy wrapper, got %d", len(chunks))
	}

	// Should be processed by legacy modifier
	processed := legacyMod.Process(data, DirectionOutbound)
	if string(chunks[0]) != string(processed) {
		t.Error("Legacy wrapper should return same result as legacy Process")
	}
}

// TestSafeRandom tests thread-safe random number generation
func TestSafeRandom(t *testing.T) {
	rand := NewSafeRandom()

	// Test basic functionality
	for i := 0; i < 100; i++ {
		n := rand.Intn(1000)
		if n < 0 || n >= 1000 {
			t.Errorf("Random number %d out of range [0, 1000)", n)
		}
	}

	// Test edge cases
	n := rand.Intn(1)
	if n != 0 {
		t.Errorf("Expected 0 for Intn(1), got %d", n)
	}

	n = rand.Intn(0)
	if n != 0 {
		t.Errorf("Expected 0 for Intn(0), got %d", n)
	}
}

// TestFragmentationConfigValidation tests configuration validation
func TestFragmentationConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *FragmentationConfig
		wantErr bool
	}{
		{
			name: "Valid config",
			config: &FragmentationConfig{
				MinSize:          64,
				MaxSize:          256,
				RandomSizes:      true,
				AntiNagleDelay:   1 * time.Millisecond,
				MaxChunks:        10,
				MLOptimization:   false,
				AdaptiveStrategy: "simple",
			},
			wantErr: false,
		},
		{
			name: "Invalid min size",
			config: &FragmentationConfig{
				MinSize: 0,
				MaxSize: 256,
			},
			wantErr: true,
		},
		{
			name: "Invalid max size",
			config: &FragmentationConfig{
				MinSize: 64,
				MaxSize: 0,
			},
			wantErr: true,
		},
		{
			name: "Min size greater than max size",
			config: &FragmentationConfig{
				MinSize: 256,
				MaxSize: 64,
			},
			wantErr: true,
		},
		{
			name: "Invalid max chunks",
			config: &FragmentationConfig{
				MinSize:   64,
				MaxSize:   256,
				MaxChunks: 0,
			},
			wantErr: true,
		},
		{
			name: "Invalid anti-nagle delay",
			config: &FragmentationConfig{
				MinSize:        64,
				MaxSize:        256,
				AntiNagleDelay: -1 * time.Millisecond,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFragmentationConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFragmentationConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFragmentationMetrics tests metrics collection
func TestFragmentationMetrics(t *testing.T) {
	metrics := NewFragmentationMetrics()

	// Test initial state
	totalFragments, avgSize, totalTime, successRate := metrics.GetMetrics()
	if totalFragments != 0 {
		t.Errorf("Expected 0 total fragments, got %d", totalFragments)
	}

	if avgSize != 0 {
		t.Errorf("Expected 0 average size, got %f", avgSize)
	}

	if totalTime != 0 {
		t.Errorf("Expected 0 total time, got %v", totalTime)
	}

	// Record some metrics
	metrics.RecordFragmentation(3, 150, 5*time.Millisecond, true)
	metrics.RecordFragmentation(2, 100, 3*time.Millisecond, true)
	metrics.RecordFragmentation(4, 200, 7*time.Millisecond, false)

	// Check updated metrics
	totalFragments, avgSize, totalTime, successRate = metrics.GetMetrics()
	if totalFragments != 9 {
		t.Errorf("Expected 9 total fragments, got %d", totalFragments)
	}

	expectedAvgSize := 50.0 // (150/3 + 100/2 + 200/4) / 3 = (50 + 50 + 50) / 3 = 50
	if avgSize != expectedAvgSize {
		t.Errorf("Expected average size %f, got %f", expectedAvgSize, avgSize)
	}

	expectedTotalTime := 5*time.Millisecond + 3*time.Millisecond + 7*time.Millisecond
	if totalTime != expectedTotalTime {
		t.Errorf("Expected total time %v, got %v", expectedTotalTime, totalTime)
	}

	// Success rate should be less than 1.0 due to one failure
	if successRate >= 1.0 {
		t.Error("Expected success rate less than 1.0 due to failure")
	}
}

// BenchmarkFragmentationModifierProcessToChunks benchmarks chunk processing
func BenchmarkFragmentationModifierProcessToChunks(b *testing.B) {
	modifier := &FragmentationModifier{}
	config := map[string]interface{}{
		"min_size": 64,
		"max_size": 256,
		"random":   true,
	}
	modifier.Configure(config)

	data := make([]byte, 1024) // 1KB of data
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		modifier.ProcessToChunks(data, DirectionOutbound)
	}
}

// BenchmarkAdaptiveFragmentationModifierProcessToChunks benchmarks adaptive chunk processing
func BenchmarkAdaptiveFragmentationModifierProcessToChunks(b *testing.B) {
	modifier := NewAdaptiveFragmentationModifier(64, 256, true)

	data := make([]byte, 1024) // 1KB of data
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		modifier.ProcessToChunks(data, DirectionOutbound)
	}
}
