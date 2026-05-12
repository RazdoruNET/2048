package pipeline

import (
	"context"
	"net"
	"testing"
	"time"
)

// MockModifier implements Modifier interface for testing
type MockModifier struct {
	name       string
	shouldFail bool
	delay      time.Duration
}

func (m *MockModifier) Name() string {
	return m.name
}

func (m *MockModifier) Process(data []byte, direction Direction) []byte {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return data
}

func (m *MockModifier) Configure(config map[string]interface{}) error {
	return nil
}

// MockConnection implements net.Conn for testing
type MockConnection struct {
	data    []byte
	closed  bool
	readPos int
}

func (m *MockConnection) Read(b []byte) (n int, err error) {
	if m.closed {
		return 0, net.ErrClosed
	}
	if m.readPos >= len(m.data) {
		return 0, nil
	}
	n = copy(b, m.data[m.readPos:])
	m.readPos += n
	return n, nil
}

func (m *MockConnection) Write(b []byte) (n int, err error) {
	if m.closed {
		return 0, net.ErrClosed
	}
	m.data = append(m.data, b...)
	return len(b), nil
}

func (m *MockConnection) Close() error {
	m.closed = true
	return nil
}

func (m *MockConnection) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
}

func (m *MockConnection) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 80}
}

func (m *MockConnection) SetDeadline(t time.Time) error {
	return nil
}

func (m *MockConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

// MockEngine implements Engine interface for testing
type MockEngine struct {
	modifiers []Modifier
}

func (m *MockEngine) CreatePipeline(target string, port uint16) *Pipeline {
	if len(m.modifiers) == 0 {
		return nil
	}

	return &Pipeline{
		modifiers: m.modifiers,
		target:    target,
		port:      port,
	}
}

// MockEngineWrapper wraps MockEngine to implement Engine interface
type MockEngineWrapper struct {
	*MockEngine
}

func NewMockEngineWrapper(modifiers []Modifier) *MockEngineWrapper {
	return &MockEngineWrapper{
		MockEngine: &MockEngine{modifiers: modifiers},
	}
}

func TestNewPipelineRetry(t *testing.T) {
	engine := NewMockEngineWrapper([]Modifier{})
	config := DefaultRetryConfig()

	retry := NewPipelineRetry(engine, config)

	if retry == nil {
		t.Fatal("Expected non-nil PipelineRetry")
	}

	if retry.config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts=3, got %d", retry.config.MaxAttempts)
	}

	if retry.strategy != StrategyAdaptive {
		t.Errorf("Expected StrategyAdaptive, got %v", retry.strategy)
	}
}

func TestRetryWithPipeline_NoPipeline(t *testing.T) {
	engine := NewMockEngineWrapper([]Modifier{})
	retry := NewPipelineRetry(engine, DefaultRetryConfig())

	req := &SOCKS5Request{
		DstAddr: "example.com",
		DstPort: 443,
	}

	result, err := retry.RetryWithPipeline(context.Background(), req)

	if err == nil {
		t.Error("Expected error for no pipeline")
	}

	if result.Success {
		t.Error("Expected failure for no pipeline")
	}
}

func TestRetryWithPipeline_Success(t *testing.T) {
	// Create successful modifier
	successModifier := &MockModifier{
		name: "success",
	}

	engine := NewMockEngineWrapper([]Modifier{successModifier})
	retry := NewPipelineRetry(engine, DefaultRetryConfig())

	req := &SOCKS5Request{
		DstAddr: "example.com",
		DstPort: 443,
	}

	result, err := retry.RetryWithPipeline(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if result.Technique != "success" {
		t.Errorf("Expected technique='success', got '%s'", result.Technique)
	}
}

func TestRetryWithPipeline_Failure(t *testing.T) {
	// Create failing modifier
	failModifier := &MockModifier{
		name:       "fail",
		shouldFail: true,
	}

	engine := NewMockEngineWrapper([]Modifier{failModifier})
	retry := NewPipelineRetry(engine, DefaultRetryConfig())

	req := &SOCKS5Request{
		DstAddr: "example.com",
		DstPort: 443,
	}

	result, err := retry.RetryWithPipeline(context.Background(), req)

	if err == nil {
		t.Error("Expected error for failed retry")
	}

	if result.Success {
		t.Error("Expected failure")
	}
}

func TestCircuitBreaker(t *testing.T) {
	engine := NewMockEngineWrapper([]Modifier{})
	config := &RetryConfig{
		MaxAttempts:      3,
		Timeout:          1 * time.Second,
		RetryDelay:       10 * time.Millisecond,
		CircuitBreaker:   true,
		FailureThreshold: 2,
	}

	retry := NewPipelineRetry(engine, config)

	// Trigger circuit breaker
	target := "example.com"
	retry.updateFailureCount(target)
	retry.updateFailureCount(target)

	if !retry.IsCircuitBreakerOpen(target) {
		t.Error("Expected circuit breaker to be open")
	}

	// Reset circuit breaker
	retry.ResetCircuitBreaker(target)

	if retry.IsCircuitBreakerOpen(target) {
		t.Error("Expected circuit breaker to be closed after reset")
	}
}

func TestGetStatistics(t *testing.T) {
	engine := NewMockEngineWrapper([]Modifier{})
	retry := NewPipelineRetry(engine, DefaultRetryConfig())

	// Update some statistics
	retry.updateSuccessCount("fragmentation")
	retry.updateSuccessCount("fragmentation")
	retry.updateFailureCountForTechnique("fragmentation")
	retry.updateFailureCountForTechnique("headers")

	stats := retry.GetStatistics()

	// Check fragmentation stats
	if fragRate, ok := stats["fragmentation_success_rate"].(float64); !ok {
		t.Error("Expected fragmentation_success_rate in stats")
	} else {
		// 2 successes, 1 failure = 66.67%
		expected := 66.66666666666666
		if fragRate < expected-0.1 || fragRate > expected+0.1 {
			t.Errorf("Expected fragmentation success rate ~%.2f, got %.2f", expected, fragRate)
		}
	}

	// Check headers stats
	if headersRate, ok := stats["headers_success_rate"].(float64); !ok {
		t.Error("Expected headers_success_rate in stats")
	} else {
		// 0 successes, 1 failure = 0%
		if headersRate != 0.0 {
			t.Errorf("Expected headers success rate 0.0, got %.2f", headersRate)
		}
	}
}

func TestAdaptiveDelay(t *testing.T) {
	retry := NewPipelineRetry(NewMockEngineWrapper([]Modifier{}), DefaultRetryConfig())

	// Test high success rate technique
	delay1 := retry.calculateAdaptiveDelay("fragmentation") // 85% success rate
	expected1 := DefaultRetryConfig().RetryDelay
	if delay1 != expected1 {
		t.Errorf("Expected delay %v for high success rate, got %v", expected1, delay1)
	}

	// Test medium success rate technique
	delay2 := retry.calculateAdaptiveDelay("protocol_mask") // 70% success rate
	expected2 := DefaultRetryConfig().RetryDelay * 2
	if delay2 != expected2 {
		t.Errorf("Expected delay %v for medium success rate, got %v", expected2, delay2)
	}

	// Test low success rate technique
	delay3 := retry.calculateAdaptiveDelay("unknown") // 50% success rate
	expected3 := DefaultRetryConfig().RetryDelay * 3
	if delay3 != expected3 {
		t.Errorf("Expected delay %v for low success rate, got %v", expected3, delay3)
	}
}

func TestModifiedConnection(t *testing.T) {
	// Create mock connection and modifier
	baseConn := &MockConnection{}
	modifier := &MockModifier{name: "test"}

	modConn := &ModifiedConnection{
		Conn:     baseConn,
		Modifier: modifier,
		Request:  &SOCKS5Request{},
	}

	// Test write
	testData := []byte("test data")
	n, err := modConn.Write(testData)

	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Test read
	readBuf := make([]byte, 100)
	n, err = modConn.Read(readBuf)

	if err != nil && err != net.ErrClosed {
		t.Errorf("Read failed: %v", err)
	}

	expected := "test data"
	if n != len(expected) {
		t.Errorf("Expected to read %d bytes, read %d", len(expected), n)
	}

	if string(readBuf[:n]) != expected {
		t.Errorf("Expected to read '%s', got '%s'", expected, string(readBuf[:n]))
	}
}
