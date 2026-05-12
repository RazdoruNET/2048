package pipeline

import (
	"bytes"
	"testing"
	"time"
)

// TestFragmentationIntegration tests end-to-end fragmentation behavior
func TestFragmentationIntegration(t *testing.T) {
	// Test that Process() is NO-OP and ProcessToChunks() creates real fragments
	modifier := &FragmentationModifier{}
	config := map[string]interface{}{
		"min_size": 10,
		"max_size": 20,
		"random":   false,
	}
	
	if err := modifier.Configure(config); err != nil {
		t.Fatalf("Failed to configure modifier: %v", err)
	}
	
	// Test data
	testData := []byte("this is test data that should be fragmented into multiple chunks for real DPI bypass")
	
	// Test Process() behavior - should be NO-OP for fragmenting modifier
	processed := modifier.Process(testData, DirectionOutbound)
	
	if !bytes.Equal(processed, testData) {
		t.Errorf("Process() should return original data for fragmenting modifier, got %d bytes vs %d expected", 
			len(processed), len(testData))
	}
	
	// Test ProcessToChunks() behavior - should create multiple chunks
	chunks := modifier.ProcessToChunks(testData, DirectionOutbound)
	
	if len(chunks) < 2 {
		t.Errorf("ProcessToChunks() should create multiple chunks, got %d", len(chunks))
	}
	
	// Verify data integrity across chunks
	var combined []byte
	for _, chunk := range chunks {
		combined = append(combined, chunk...)
	}
	
	if !bytes.Equal(combined, testData) {
		t.Error("Combined chunks don't match original data")
	}
	
	// Verify chunks are within size bounds
	for i, chunk := range chunks {
		if len(chunk) > modifier.MaxSize {
			t.Errorf("Chunk %d size %d exceeds MaxSize %d", i, len(chunk), modifier.MaxSize)
		}
		
		if len(chunk) < modifier.MinSize && i < len(chunks)-1 {
			t.Errorf("Chunk %d size %d below MinSize %d", i, len(chunk), modifier.MinSize)
		}
	}
}

// TestAdaptiveFragmentationIntegration tests adaptive fragmentation end-to-end
func TestAdaptiveFragmentationIntegration(t *testing.T) {
	modifier := NewAdaptiveFragmentationModifier(15, 40, false) // Fixed size for testing
	
	// Test data
	testData := []byte("adaptive fragmentation test data for DPI bypass verification")
	
	// Test Process() behavior - should be NO-OP for fragmenting modifier
	processed := modifier.Process(testData, DirectionOutbound)
	
	if !bytes.Equal(processed, testData) {
		t.Errorf("Process() should return original data for adaptive fragmenting modifier, got %d bytes vs %d expected", 
			len(processed), len(testData))
	}
	
	// Test ProcessToChunks() behavior - should create 3 chunks
	chunks := modifier.ProcessToChunks(testData, DirectionOutbound)
	
	expectedChunks := 3 // Adaptive fragmentation creates 3 chunks by default
	if len(chunks) != expectedChunks {
		t.Errorf("ProcessToChunks() should create %d chunks, got %d", expectedChunks, len(chunks))
	}
	
	// Verify data integrity
	var combined []byte
	for _, chunk := range chunks {
		combined = append(combined, chunk...)
	}
	
	if !bytes.Equal(combined, testData) {
		t.Error("Combined adaptive chunks don't match original data")
	}
	
	// Verify chunk sizes are within bounds
	for i, chunk := range chunks {
		if len(chunk) > modifier.MaxSize {
			t.Errorf("Adaptive chunk %d size %d exceeds MaxSize %d", i, len(chunk), modifier.MaxSize)
		}
		
		if len(chunk) < modifier.MinSize && i < len(chunks)-1 {
			t.Errorf("Adaptive chunk %d size %d below MinSize %d", i, len(chunk), modifier.MinSize)
		}
	}
}

// TestModifiedConnectionV2FragmentationFlow tests the complete flow
func TestModifiedConnectionV2FragmentationFlow(t *testing.T) {
	// Create mock connection
	mockConn := NewMockConnectionV2()
	
	// Create fragmentation modifier
	modifier := &FragmentationModifier{}
	config := map[string]interface{}{
		"min_size": 8,
		"max_size": 16,
		"random":   false,
	}
	
	if err := modifier.Configure(config); err != nil {
		t.Fatalf("Failed to configure modifier: %v", err)
	}
	
	// Create connection config
	connConfig := &ConnectionConfig{
		AntiNagleDelay:  0, // No delay for testing
		MaxWriteRetries: 3,
		WriteTimeout:    5 * time.Second,
		EnableMetrics:   true,
	}
	
	// Create modified connection
	modConn := NewModifiedConnectionV2(mockConn, modifier, nil, connConfig)
	
	// Test data that should be fragmented
	testData := []byte("this test data should be sent as separate TCP packets for DPI bypass")
	
	// Write data - this should trigger real fragmentation
	written, err := modConn.Write(testData)
	
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	
	if written != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), written)
	}
	
	// Verify that multiple chunks were written (real network-level fragmentation)
	writtenData := mockConn.GetWrittenData()
	
	if len(writtenData) < 2 {
		t.Errorf("Expected at least 2 separate TCP packets, got %d", len(writtenData))
	}
	
	// Verify data integrity across packets
	var combined []byte
	for _, chunk := range writtenData {
		combined = append(combined, chunk...)
	}
	
	if !bytes.Equal(combined, testData) {
		t.Error("Combined TCP packets don't match original data")
	}
	
	// Verify individual packet sizes are within bounds
	for i, chunk := range writtenData {
		if len(chunk) > modifier.MaxSize {
			t.Errorf("TCP packet %d size %d exceeds MaxSize %d", i, len(chunk), modifier.MaxSize)
		}
		
		if len(chunk) < modifier.MinSize && i < len(writtenData)-1 {
			t.Errorf("TCP packet %d size %d below MinSize %d", i, len(chunk), modifier.MinSize)
		}
	}
	
	// Verify statistics
	stats := modConn.GetWriteStats()
	if stats.FragmentedWrites != 1 {
		t.Errorf("Expected 1 fragmented write, got %d", stats.FragmentedWrites)
	}
	
	if stats.TotalWrites != 1 {
		t.Errorf("Expected 1 total write, got %d", stats.TotalWrites)
	}
	
	if stats.TotalBytes != uint64(len(testData)) {
		t.Errorf("Expected %d total bytes, got %d", len(testData), stats.TotalBytes)
	}
}

// TestLegacyModifierWrapperIntegration tests legacy modifier behavior
func TestLegacyModifierWrapperIntegration(t *testing.T) {
	// Create legacy modifier
	legacyMod := &HeadersModifier{}
	legacyMod.Configure(map[string]interface{}{
		"random_headers": false, // Deterministic for testing
	})
	
	// Wrap with legacy wrapper
	wrapper := NewLegacyModifierWrapper(legacyMod)
	
	// Test data
	testData := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	
	// Test Process() behavior - should work normally for legacy modifiers
	processed := wrapper.Process(testData, DirectionOutbound)
	
	if bytes.Equal(processed, testData) {
		t.Error("Legacy wrapper Process() should modify data, but returned original")
	}
	
	// Test ProcessToChunks() behavior - should return single chunk
	chunks := wrapper.ProcessToChunks(testData, DirectionOutbound)
	
	if len(chunks) != 1 {
		t.Errorf("Legacy wrapper ProcessToChunks() should return 1 chunk, got %d", len(chunks))
	}
	
	// Verify chunk contains processed data
	if !bytes.Equal(chunks[0], processed) {
		t.Error("Legacy wrapper chunk should match Process() output")
	}
	
	// Verify SupportsFragmentation() returns false
	if wrapper.SupportsFragmentation() {
		t.Error("Legacy wrapper should not support fragmentation")
	}
}
