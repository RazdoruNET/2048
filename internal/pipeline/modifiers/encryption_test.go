package modifiers

import (
	"testing"
	"time"

	"socks5-dpi-proxy/internal/pipeline"
)

func TestEncryptionModifier_Configure(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]interface{}
		expectError  bool
		expectedType string
	}{
		{
			name: "ChaCha20-Poly1305 configuration",
			config: map[string]interface{}{
				"type": "chacha20poly1305",
				"key":  "test_key_123",
			},
			expectError:  false,
			expectedType: "chacha20poly1305",
		},
		{
			name: "AES-GCM configuration",
			config: map[string]interface{}{
				"type": "aes-gcm",
				"key":  "test_key_456",
			},
			expectError:  false,
			expectedType: "aes-gcm",
		},
		{
			name: "XOR backward compatibility",
			config: map[string]interface{}{
				"type": "xor",
				"key":  "test_key_789",
			},
			expectError:  false,
			expectedType: "xor",
		},
		{
			name:         "Default configuration",
			config:       map[string]interface{}{},
			expectError:  false,
			expectedType: "xor",
		},
		{
			name: "Unsupported encryption type",
			config: map[string]interface{}{
				"type": "unsupported",
				"key":  "test_key",
			},
			expectError:  true,
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod := &EncryptionModifier{}
			err := mod.Configure(tt.config)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && mod.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, mod.Type)
			}
		})
	}
}

func TestEncryptionModifier_Process(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]interface{}
		input     []byte
		direction pipeline.Direction
		wantError bool
	}{
		{
			name: "ChaCha20-Poly1305 outbound",
			config: map[string]interface{}{
				"type": "chacha20poly1305",
				"key":  "test_key_123456789012345678901234",
			},
			input:     []byte("Hello, World!"),
			direction: pipeline.DirectionOutbound,
			wantError: false,
		},
		{
			name: "AES-GCM outbound",
			config: map[string]interface{}{
				"type": "aes-gcm",
				"key":  "test_key_123456789012345678901234",
			},
			input:     []byte("Hello, World!"),
			direction: pipeline.DirectionOutbound,
			wantError: false,
		},
		{
			name: "XOR outbound",
			config: map[string]interface{}{
				"type": "xor",
				"key":  "simple_key",
			},
			input:     []byte("Hello, World!"),
			direction: pipeline.DirectionOutbound,
			wantError: false,
		},
		{
			name: "Empty input",
			config: map[string]interface{}{
				"type": "xor",
				"key":  "test_key",
			},
			input:     []byte{},
			direction: pipeline.DirectionOutbound,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod := &EncryptionModifier{}
			err := mod.Configure(tt.config)
			if err != nil {
				t.Fatalf("Failed to configure modifier: %v", err)
			}

			result := mod.Process(tt.input, tt.direction)

			// For XOR, test round-trip
			if tt.config["type"] == "xor" {
				// Encrypt
				encrypted := mod.Process(tt.input, pipeline.DirectionOutbound)
				// Decrypt
				decrypted := mod.Process(encrypted, pipeline.DirectionInbound)

				if string(decrypted) != string(tt.input) {
					t.Errorf("XOR round-trip failed: got %s, want %s", decrypted, tt.input)
				}
			}

			// For AEAD modes, ensure output is different from input (encrypted)
			if tt.config["type"] == "chacha20poly1305" || tt.config["type"] == "aes-gcm" {
				if string(result) == string(tt.input) {
					t.Errorf("Expected encrypted output to differ from input")
				}

				// Test round-trip for AEAD modes with fresh modifier instance
				// to avoid nonce reuse issues
				freshMod := &EncryptionModifier{}
				err := freshMod.Configure(tt.config)
				if err != nil {
					t.Fatalf("Failed to configure fresh modifier: %v", err)
				}

				encrypted := freshMod.Process(tt.input, pipeline.DirectionOutbound)
				decrypted := freshMod.Process(encrypted, pipeline.DirectionInbound)

				if string(decrypted) != string(tt.input) {
					t.Errorf("AEAD round-trip failed: got %s, want %s", decrypted, tt.input)
				}
			}

			// Empty input should return empty
			if len(tt.input) == 0 && len(result) != 0 {
				t.Errorf("Empty input should return empty output")
			}
		})
	}
}

func TestEncryptionModifier_KeyRotation(t *testing.T) {
	mod := &EncryptionModifier{}
	config := map[string]interface{}{
		"type": "chacha20poly1305",
		"key":  "rotation_test_key",
	}

	err := mod.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure modifier: %v", err)
	}

	// Store initial state
	initialKey := mod.Key
	initialTime := mod.lastRotation

	// Force key rotation
	mod.rotationInterval = time.Millisecond * 10
	time.Sleep(time.Millisecond * 20)

	// Process some data to trigger rotation
	mod.Process([]byte("test"), pipeline.DirectionOutbound)

	// Check that rotation occurred
	if mod.lastRotation.Equal(initialTime) {
		t.Error("Key rotation did not occur")
	}

	// Key should remain the same base key
	if mod.Key != initialKey {
		t.Error("Base key should not change during rotation")
	}
}

func TestEncryptionModifier_DeriveKey(t *testing.T) {
	mod := &EncryptionModifier{}

	tests := []struct {
		name     string
		inputKey []byte
		length   int
		expected int
	}{
		{
			name:     "Short key expansion",
			inputKey: []byte("short"),
			length:   32,
			expected: 32,
		},
		{
			name:     "Long key truncation",
			inputKey: []byte("this_is_a_very_long_key_that_exceeds_the_required_length"),
			length:   16,
			expected: 16,
		},
		{
			name:     "Exact length key",
			inputKey: []byte("exactly_32_bytes_long_key_here!!"),
			length:   32,
			expected: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mod.deriveKey(tt.inputKey, tt.length)
			if len(result) != tt.expected {
				t.Errorf("Expected key length %d, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestEncryptionModifier_ShouldRotateKey(t *testing.T) {
	mod := &EncryptionModifier{}

	// Test with XOR (should not rotate)
	mod.Type = "xor"
	mod.rotationInterval = time.Hour
	mod.lastRotation = time.Now().Add(-time.Hour * 2)

	if mod.shouldRotateKey() {
		t.Error("XOR encryption should not rotate keys")
	}

	// Test with ChaCha20-Poly1305 (should rotate)
	mod.Type = "chacha20poly1305"
	mod.lastRotation = time.Now().Add(-time.Hour * 2)

	if !mod.shouldRotateKey() {
		t.Error("ChaCha20-Poly1305 should rotate keys")
	}
}

// Benchmark tests
func BenchmarkEncryptionModifier_XOR(b *testing.B) {
	mod := &EncryptionModifier{}
	mod.Configure(map[string]interface{}{
		"type": "xor",
		"key":  "benchmark_key",
	})

	data := make([]byte, 1024)
	copy(data, "Hello, World! Benchmark test data that is longer than the original message.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mod.Process(data, pipeline.DirectionOutbound)
	}
}

func BenchmarkEncryptionModifier_ChaCha20Poly1305(b *testing.B) {
	mod := &EncryptionModifier{}
	mod.Configure(map[string]interface{}{
		"type": "chacha20poly1305",
		"key":  "benchmark_key_123456789012345678901234",
	})

	data := make([]byte, 1024)
	copy(data, "Hello, World! Benchmark test data that is longer than the original message.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mod.Process(data, pipeline.DirectionOutbound)
	}
}

func BenchmarkEncryptionModifier_AESGCM(b *testing.B) {
	mod := &EncryptionModifier{}
	mod.Configure(map[string]interface{}{
		"type": "aes-gcm",
		"key":  "benchmark_key_123456789012345678901234",
	})

	data := make([]byte, 1024)
	copy(data, "Hello, World! Benchmark test data that is longer than the original message.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mod.Process(data, pipeline.DirectionOutbound)
	}
}
