package pipeline

import (
	"testing"
)

func TestNewVLESSModifier(t *testing.T) {
	modifier := NewVLESSModifier()
	
	if modifier == nil {
		t.Error("Expected modifier to be created")
	}
	
	if modifier.Name() != "vless_client" {
		t.Errorf("Expected name 'vless_client', got '%s'", modifier.Name())
	}
}

func TestVLESSModifier_Configure(t *testing.T) {
	modifier := NewVLESSModifier()
	
	// Test valid configuration
	config := map[string]interface{}{
		"server": "example.com",
		"uuid":   "12345678-1234-1234-1234-123456789abc",
		"port":   443,
	}
	
	err := modifier.Configure(config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Test missing server
	configNoServer := map[string]interface{}{
		"uuid": "12345678-1234-1234-1234-123456789abc",
	}
	
	err = modifier.Configure(configNoServer)
	if err == nil {
		t.Error("Expected error for missing server")
	}
	
	// Test missing UUID
	configNoUUID := map[string]interface{}{
		"server": "example.com",
	}
	
	err = modifier.Configure(configNoUUID)
	if err == nil {
		t.Error("Expected error for missing UUID")
	}
}

func TestVLESSModifier_Process(t *testing.T) {
	modifier := NewVLESSModifier()
	
	config := map[string]interface{}{
		"server": "example.com",
		"uuid":   "12345678-1234-1234-1234-123456789abc",
	}
	
	modifier.Configure(config)
	
	// Test inbound traffic (should pass through)
	data := []byte("test data")
	result := modifier.Process(data, DirectionInbound)
	
	if string(result) != string(data) {
		t.Error("Inbound data should pass through unchanged")
	}
	
	// Test outbound traffic (should be processed)
	result = modifier.Process(data, DirectionOutbound)
	
	// For now, should return unchanged (TODO: implement VLESS tunneling)
	if string(result) != string(data) {
		t.Error("Outbound data should be unchanged for now")
	}
}

func TestVLESSModifier_Getters(t *testing.T) {
	modifier := NewVLESSModifier()
	
	config := map[string]interface{}{
		"server": "example.com",
		"port":   8080,
		"uuid":   "12345678-1234-1234-1234-123456789abc",
		"flow":   "xtls-rprx-vision",
	}
	
	modifier.Configure(config)
	
	if modifier.GetServer() != "example.com" {
		t.Errorf("Expected server 'example.com', got '%s'", modifier.GetServer())
	}
	
	if modifier.GetPort() != 8080 {
		t.Errorf("Expected port 8080, got %d", modifier.GetPort())
	}
	
	if modifier.GetUUID() != "12345678-1234-1234-1234-123456789abc" {
		t.Errorf("Expected UUID '12345678-1234-1234-1234-123456789abc', got '%s'", modifier.GetUUID())
	}
	
	if modifier.GetFlow() != "xtls-rprx-vision" {
		t.Errorf("Expected flow 'xtls-rprx-vision', got '%s'", modifier.GetFlow())
	}
}

func TestVLESSModifier_IsEnabled(t *testing.T) {
	modifier := NewVLESSModifier()
	
	// Test without configuration
	if modifier.IsEnabled() {
		t.Error("Should not be enabled without configuration")
	}
	
	// Test with partial configuration
	config := map[string]interface{}{
		"server": "example.com",
	}
	modifier.Configure(config)
	
	if modifier.IsEnabled() {
		t.Error("Should not be enabled with only server")
	}
	
	// Test with full configuration
	config = map[string]interface{}{
		"server": "example.com",
		"uuid":   "12345678-1234-1234-1234-123456789abc",
	}
	modifier.Configure(config)
	
	if !modifier.IsEnabled() {
		t.Error("Should be enabled with server and UUID")
	}
}

func TestVLESSModifier_Defaults(t *testing.T) {
	modifier := NewVLESSModifier()
	
	config := map[string]interface{}{
		"server": "example.com",
		"uuid":   "12345678-1234-1234-1234-123456789abc",
	}
	
	modifier.Configure(config)
	
	// Check default values
	if modifier.GetPort() != 443 {
		t.Errorf("Expected default port 443, got %d", modifier.GetPort())
	}
	
	if modifier.GetFlow() != "xtls-rprx-vision" {
		t.Errorf("Expected default flow 'xtls-rprx-vision', got '%s'", modifier.GetFlow())
	}
}

// Benchmark tests
func BenchmarkVLESSModifier_Process(b *testing.B) {
	modifier := NewVLESSModifier()
	
	config := map[string]interface{}{
		"server": "example.com",
		"uuid":   "12345678-1234-1234-1234-123456789abc",
	}
	
	modifier.Configure(config)
	
	data := []byte("test data for benchmarking")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		modifier.Process(data, DirectionOutbound)
	}
}
