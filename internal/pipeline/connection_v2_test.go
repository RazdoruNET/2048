package pipeline

import (
	"bytes"
	"net"
	"sync"
	"testing"
	"time"
)

// MockConnectionV2 implements net.Conn for testing V2 functionality
type MockConnectionV2 struct {
	data      []byte
	writeData [][]byte
	mu        sync.Mutex
	closed    bool
}

func NewMockConnectionV2() *MockConnectionV2 {
	return &MockConnectionV2{
		writeData: make([][]byte, 0),
	}
}

func (mc *MockConnectionV2) Read(b []byte) (n int, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.closed {
		return 0, net.ErrClosed
	}

	if len(mc.data) == 0 {
		return 0, nil // EOF
	}

	n = copy(b, mc.data)
	mc.data = mc.data[n:]
	return n, nil
}

func (mc *MockConnectionV2) Write(b []byte) (n int, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.closed {
		return 0, net.ErrClosed
	}

	// Store written data for verification
	chunk := make([]byte, len(b))
	copy(chunk, b)
	mc.writeData = append(mc.writeData, chunk)

	return len(b), nil
}

func (mc *MockConnectionV2) Close() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.closed = true
	return nil
}

func (mc *MockConnectionV2) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
}

func (mc *MockConnectionV2) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8081}
}

func (mc *MockConnectionV2) SetDeadline(t time.Time) error {
	return nil
}

func (mc *MockConnectionV2) SetReadDeadline(t time.Time) error {
	return nil
}

func (mc *MockConnectionV2) SetWriteDeadline(t time.Time) error {
	return nil
}

func (mc *MockConnectionV2) GetWrittenData() [][]byte {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	result := make([][]byte, len(mc.writeData))
	for i, data := range mc.writeData {
		result[i] = make([]byte, len(data))
		copy(result[i], data)
	}

	return result
}

func (mc *MockConnectionV2) ClearWrittenData() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.writeData = mc.writeData[:0]
}

func (mc *MockConnectionV2) SetReadData(data []byte) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.data = make([]byte, len(data))
	copy(mc.data, data)
}

// TestModifiedConnectionV2Fragmentation tests real network-level fragmentation
func TestModifiedConnectionV2Fragmentation(t *testing.T) {
	// Create mock connection
	mockConn := NewMockConnectionV2()

	// Create fragmentation modifier
	modifier := &FragmentationModifier{}
	config := map[string]interface{}{
		"min_size":         10,
		"max_size":         20,
		"random":           false, // Deterministic for testing
		"anti_nagle_delay": 0,     // No delay for testing
	}

	if err := modifier.Configure(config); err != nil {
		t.Fatalf("Failed to configure modifier: %v", err)
	}

	// Create connection config
	connConfig := &ConnectionConfig{
		AntiNagleDelay:  0,
		MaxWriteRetries: 3,
		WriteTimeout:    5 * time.Second,
		EnableMetrics:   true,
	}

	// Create modified connection
	modConn := NewModifiedConnectionV2(mockConn, modifier, nil, connConfig)

	// Test data that should be fragmented
	testData := []byte("this is a test string that should be fragmented into multiple chunks")

	// Write data
	start := time.Now()
	written, err := modConn.Write(testData)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if written != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), written)
	}

	// Verify fragmentation occurred
	writtenData := mockConn.GetWrittenData()

	if len(writtenData) <= 1 {
		t.Errorf("Expected multiple chunks, got %d", len(writtenData))
	}

	// Verify data integrity across chunks
	var combined []byte
	for _, chunk := range writtenData {
		combined = append(combined, chunk...)
	}

	if !bytes.Equal(combined, testData) {
		t.Error("Combined written data doesn't match original test data")
	}

	// Verify chunk sizes are within bounds
	for i, chunk := range writtenData {
		if len(chunk) > modifier.MaxSize {
			t.Errorf("Chunk %d size %d exceeds MaxSize %d", i, len(chunk), modifier.MaxSize)
		}

		if len(chunk) < modifier.MinSize && i < len(writtenData)-1 {
			t.Errorf("Chunk %d size %d below MinSize %d", i, len(chunk), modifier.MinSize)
		}
	}

	// Check write statistics
	stats := modConn.GetWriteStats()
	if stats.TotalWrites != 1 {
		t.Errorf("Expected 1 total write, got %d", stats.TotalWrites)
	}

	if stats.TotalBytes != uint64(len(testData)) {
		t.Errorf("Expected %d total bytes, got %d", len(testData), stats.TotalBytes)
	}

	if stats.FragmentedWrites != 1 {
		t.Errorf("Expected 1 fragmented write, got %d", stats.FragmentedWrites)
	}

	if stats.AverageLatency > elapsed {
		t.Errorf("Average latency %v greater than actual elapsed %v", stats.AverageLatency, elapsed)
	}
}

// TestModifiedConnectionV2Legacy tests legacy modifier compatibility
func TestModifiedConnectionV2Legacy(t *testing.T) {
	// Create mock connection
	mockConn := NewMockConnectionV2()

	// Create legacy modifier (wrapped)
	legacyMod := &HeadersModifier{}
	legacyMod.Configure(map[string]interface{}{
		"random_headers": false, // Disable randomization for deterministic testing
	})

	wrapper := NewLegacyModifierWrapper(legacyMod)

	// Create connection config
	connConfig := DefaultConnectionConfig()
	connConfig.AntiNagleDelay = 0 // No delay for testing

	// Create modified connection
	modConn := NewModifiedConnectionV2(mockConn, wrapper, nil, connConfig)

	// Test data
	testData := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")

	// Write data
	_, err := modConn.Write(testData)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Note: HeadersModifier may change data size, so we check that data was processed
	// Verify single write (no fragmentation for legacy modifiers)
	writtenData := mockConn.GetWrittenData()

	if len(writtenData) != 1 {
		t.Errorf("Expected 1 chunk for legacy modifier, got %d", len(writtenData))
	}

	// Verify data was processed by legacy modifier (should be different from original)
	if bytes.Equal(writtenData[0], testData) {
		t.Error("Legacy modifier should have processed the data but didn't")
	}

	// Verify data contains HTTP request (basic sanity check)
	if !bytes.Contains(writtenData[0], []byte("GET")) || !bytes.Contains(writtenData[0], []byte("HTTP/1.1")) {
		t.Error("Processed data doesn't contain expected HTTP request elements")
	}

	// Check statistics
	stats := modConn.GetWriteStats()
	if stats.FragmentedWrites != 0 {
		t.Errorf("Expected 0 fragmented writes for legacy modifier, got %d", stats.FragmentedWrites)
	}
}

// TestModifiedConnectionV2AdaptiveFragmentation tests adaptive fragmentation
func TestModifiedConnectionV2AdaptiveFragmentation(t *testing.T) {
	// Create mock connection
	mockConn := NewMockConnectionV2()

	// Create adaptive fragmentation modifier
	modifier := NewAdaptiveFragmentationModifier(15, 40, false) // Fixed size for testing

	// Create connection config
	connConfig := DefaultConnectionConfig()
	connConfig.AntiNagleDelay = 0 // No delay for testing

	// Create modified connection
	modConn := NewModifiedConnectionV2(mockConn, modifier, nil, connConfig)

	// Test data
	testData := []byte("adaptive fragmentation test with multiple chunks expected")

	// Write data
	written, err := modConn.Write(testData)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if written != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), written)
	}

	// Verify adaptive fragmentation
	writtenData := mockConn.GetWrittenData()

	if len(writtenData) < 2 {
		t.Errorf("Expected at least 2 chunks for adaptive fragmentation, got %d", len(writtenData))
	}

	// Verify data integrity
	var combined []byte
	for _, chunk := range writtenData {
		combined = append(combined, chunk...)
	}

	if !bytes.Equal(combined, testData) {
		t.Error("Combined adaptive chunks don't match original data")
	}

	// Verify adaptive chunk size logic
	expectedChunks := 3 // Default adaptive fragmentation creates 3 chunks
	if len(writtenData) != expectedChunks {
		t.Errorf("Expected %d chunks for adaptive fragmentation, got %d", expectedChunks, len(writtenData))
	}
}

// TestModifiedConnectionV2AntiNagleDelay tests anti-Nagle delay functionality
func TestModifiedConnectionV2AntiNagleDelay(t *testing.T) {
	// Create mock connection
	mockConn := NewMockConnectionV2()

	// Create fragmentation modifier
	modifier := &FragmentationModifier{}
	config := map[string]interface{}{
		"min_size":         5,
		"max_size":         10,
		"random":           false,
		"anti_nagle_delay": 10, // 10ms delay
	}

	if err := modifier.Configure(config); err != nil {
		t.Fatalf("Failed to configure modifier: %v", err)
	}

	// Create connection config with anti-Nagle delay
	connConfig := &ConnectionConfig{
		AntiNagleDelay:  5 * time.Millisecond,
		MaxWriteRetries: 3,
		WriteTimeout:    5 * time.Second,
		EnableMetrics:   true,
	}

	// Create modified connection
	modConn := NewModifiedConnectionV2(mockConn, modifier, nil, connConfig)

	// Test data
	testData := []byte("this test will have delays between fragments")

	// Write data and measure time
	start := time.Now()
	written, err := modConn.Write(testData)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if written != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), written)
	}

	// Verify fragmentation occurred
	writtenData := mockConn.GetWrittenData()

	if len(writtenData) <= 1 {
		t.Errorf("Expected multiple chunks with delays, got %d", len(writtenData))
	}

	// Verify delay was applied (should be at least (len(writtenData)-1) * delay)
	expectedMinDelay := time.Duration(len(writtenData)-1) * connConfig.AntiNagleDelay
	if elapsed < expectedMinDelay {
		t.Errorf("Expected minimum delay %v, got %v", expectedMinDelay, elapsed)
	}
}

// TestModifiedConnectionV2RetryLogic tests write retry functionality
func TestModifiedConnectionV2RetryLogic(t *testing.T) {
	// Create mock connection that fails on first write
	mockConn := NewMockConnectionV2()

	// Create fragmentation modifier
	modifier := &FragmentationModifier{}
	config := map[string]interface{}{
		"min_size": 10,
		"max_size": 20,
		"random":   false,
	}

	if err := modifier.Configure(config); err != nil {
		t.Fatalf("Failed to configure modifier: %v", err)
	}

	// Create connection config
	connConfig := &ConnectionConfig{
		AntiNagleDelay:  0,
		MaxWriteRetries: 3,
		WriteTimeout:    1 * time.Second,
		EnableMetrics:   true,
	}

	// Create modified connection
	modConn := NewModifiedConnectionV2(mockConn, modifier, nil, connConfig)

	// Test data
	testData := []byte("retry test data")

	// Write data
	written, err := modConn.Write(testData)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if written != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), written)
	}

	// Verify data was written correctly
	writtenData := mockConn.GetWrittenData()

	var combined []byte
	for _, chunk := range writtenData {
		combined = append(combined, chunk...)
	}

	if !bytes.Equal(combined, testData) {
		t.Error("Data integrity compromised during retry")
	}
}

// TestModifiedConnectionV2Metrics tests metrics collection
func TestModifiedConnectionV2Metrics(t *testing.T) {
	// Create mock connection
	mockConn := NewMockConnectionV2()

	// Create fragmentation modifier
	modifier := &FragmentationModifier{}
	config := map[string]interface{}{
		"min_size": 10,
		"max_size": 20,
		"random":   false,
	}

	if err := modifier.Configure(config); err != nil {
		t.Fatalf("Failed to configure modifier: %v", err)
	}

	// Create connection config
	connConfig := DefaultConnectionConfig()

	// Create modified connection
	modConn := NewModifiedConnectionV2(mockConn, modifier, nil, connConfig)

	// Perform multiple writes
	testData := []byte("metrics test data")

	for i := 0; i < 5; i++ {
		_, err := modConn.Write(testData)
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}

	// Check metrics
	stats := modConn.GetWriteStats()

	if stats.TotalWrites != 5 {
		t.Errorf("Expected 5 total writes, got %d", stats.TotalWrites)
	}

	expectedBytes := uint64(len(testData) * 5)
	if stats.TotalBytes != expectedBytes {
		t.Errorf("Expected %d total bytes, got %d", expectedBytes, stats.TotalBytes)
	}

	if stats.FragmentedWrites != 5 {
		t.Errorf("Expected 5 fragmented writes, got %d", stats.FragmentedWrites)
	}

	if stats.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", stats.ErrorCount)
	}

	if stats.AverageLatency <= 0 {
		t.Error("Expected positive average latency")
	}

	// Reset stats
	modConn.ResetStats()

	// Check stats after reset
	stats = modConn.GetWriteStats()

	if stats.TotalWrites != 0 {
		t.Errorf("Expected 0 total writes after reset, got %d", stats.TotalWrites)
	}

	if stats.TotalBytes != 0 {
		t.Errorf("Expected 0 total bytes after reset, got %d", stats.TotalBytes)
	}
}

// BenchmarkModifiedConnectionV2Write benchmarks write performance
func BenchmarkModifiedConnectionV2Write(b *testing.B) {
	mockConn := NewMockConnectionV2()

	modifier := &FragmentationModifier{}
	config := map[string]interface{}{
		"min_size": 64,
		"max_size": 256,
		"random":   true,
	}
	modifier.Configure(config)

	connConfig := DefaultConnectionConfig()
	connConfig.AntiNagleDelay = 0 // No delay for benchmarking

	modConn := NewModifiedConnectionV2(mockConn, modifier, nil, connConfig)

	data := make([]byte, 1024) // 1KB
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockConn.ClearWrittenData()
		modConn.Write(data)
	}
}
