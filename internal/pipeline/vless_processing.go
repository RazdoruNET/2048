package pipeline

import (
	"fmt"
)

// processOutbound processes outbound data through VLESS tunnel
func (v *VLESSModifier) processOutbound(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	// VLESS packet format: [Length:2][Data][Padding]
	packet := make([]byte, 2+len(data))

	// Add length header
	packet[0] = byte(len(data) >> 8)
	packet[1] = byte(len(data) & 0xFF)

	// Add data
	copy(packet[2:], data)

	// Apply simple XOR encryption (in real implementation would use proper encryption)
	encrypted := v.xorEncrypt(packet)

	return encrypted, nil
}

// processInbound processes inbound data from VLESS tunnel
func (v *VLESSModifier) processInbound(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return data, nil
	}

	// Decrypt data
	decrypted := v.xorDecrypt(data)

	// Extract length
	length := int(decrypted[0])<<8 | int(decrypted[1])

	// Validate length
	if length > len(decrypted)-2 || length < 0 {
		return data, nil // Return original if invalid
	}

	// Extract data
	return decrypted[2 : 2+length], nil
}

// xorEncrypt applies simple XOR encryption
func (v *VLESSModifier) xorEncrypt(data []byte) []byte {
	encrypted := make([]byte, len(data))
	uuid := v.GetUUID()
	key := []byte(uuid)

	for i, b := range data {
		encrypted[i] = b ^ key[i%len(key)]
	}

	return encrypted
}

// xorDecrypt applies simple XOR decryption
func (v *VLESSModifier) xorDecrypt(data []byte) []byte {
	// XOR is symmetric, so same function works
	return v.xorEncrypt(data)
}

// sendDataVLESS sends data through VLESS connection (alternative name to avoid conflict)
func (v *VLESSModifier) sendDataVLESS(data []byte) error {
	if v.conn == nil || !v.conn.isConnected() {
		return fmt.Errorf("not connected")
	}

	// Process outbound data
	processed, err := v.processOutbound(data)
	if err != nil {
		return fmt.Errorf("failed to process outbound data: %w", err)
	}

	// Send through WebSocket
	return v.conn.conn.WriteMessage(2, processed) // Binary message
}

// readData reads data from VLESS connection
func (v *VLESSModifier) readData() ([]byte, error) {
	if v.conn == nil || !v.conn.isConnected() {
		return nil, fmt.Errorf("not connected")
	}

	// Read from WebSocket
	messageType, data, err := v.conn.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	if messageType != 2 { // Not binary message
		return nil, fmt.Errorf("unexpected message type: %d", messageType)
	}

	// Process inbound data
	return v.processInbound(data)
}
