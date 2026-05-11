package modifiers

import (
	"crypto/rand"
	mrand "math/rand"
	"time"

	"socks5-dpi-proxy/internal/pipeline"
)

type ProtocolMaskModifier struct {
	Type       string
	SSHPadding bool
	HTTPSNoise bool
	rands      *mrand.Rand
}

func (p *ProtocolMaskModifier) Name() string {
	return "protocol_mask"
}

func (p *ProtocolMaskModifier) Configure(config map[string]interface{}) error {
	if maskType, ok := config["type"].(string); ok {
		p.Type = maskType
	} else {
		p.Type = "https"
	}

	if sshPadding, ok := config["ssh_padding"].(bool); ok {
		p.SSHPadding = sshPadding
	}

	if httpsNoise, ok := config["https_noise"].(bool); ok {
		p.HTTPSNoise = httpsNoise
	}

	p.rands = mrand.New(mrand.NewSource(time.Now().UnixNano()))
	return nil
}

func (p *ProtocolMaskModifier) Process(data []byte, direction pipeline.Direction) []byte {
	if len(data) == 0 {
		return data
	}

	// Only process outbound traffic
	if direction != pipeline.DirectionOutbound {
		return data
	}

	switch p.Type {
	case "ssh":
		return p.maskAsSSH(data)
	case "https":
		return p.maskAsHTTPS(data)
	case "tls":
		return p.maskAsTLS(data)
	default:
		return data
	}
}

func (p *ProtocolMaskModifier) maskAsSSH(data []byte) []byte {
	// SSH protocol masking
	var result []byte

	// Add SSH protocol identification
	result = append(result, []byte("SSH-2.0-OpenSSH_8.9\r\n")...)

	// Add some SSH-like padding
	if p.SSHPadding {
		padding := make([]byte, p.rands.Intn(32)+16)
		rand.Read(padding)
		result = append(result, padding...)
	}

	// Add original data
	result = append(result, data...)

	return result
}

func (p *ProtocolMaskModifier) maskAsHTTPS(data []byte) []byte {
	// HTTPS/TLS masking
	var result []byte

	// Add TLS record layer header
	tlsHeader := []byte{
		0x16, // Handshake type
		0x03, // TLS version major
		0x01, // TLS version minor (TLS 1.0)
		0x00, // Length high byte
		0x00, // Length low byte (will be set later)
	}

	// Create handshake message
	handshake := []byte{
		0x01, // ClientHello
		0x00, // Length high byte
		0x00, // Length low byte (will be set later)
		0x03, // TLS version major
		0x01, // TLS version minor
	}

	// Add random (32 bytes)
	random := make([]byte, 32)
	rand.Read(random)
	handshake = append(handshake, random...)

	// Add session ID (0 length)
	handshake = append(handshake, 0x00)

	// Add cipher suites
	handshake = append(handshake, 0x00, 0x02) // Cipher suites length
	handshake = append(handshake, 0x00, 0x3D) // TLS_RSA_WITH_AES_256_CBC_SHA256

	// Add compression methods
	handshake = append(handshake, 0x01, 0x00) // Only null compression

	// Set handshake length
	handshakeLen := len(handshake) - 4
	handshake[1] = byte(handshakeLen >> 8)
	handshake[2] = byte(handshakeLen & 0xFF)

	// Combine TLS header and handshake
	result = append(result, tlsHeader...)
	result = append(result, handshake...)

	// Set TLS record length
	tlsLen := len(handshake)
	result[3] = byte(tlsLen >> 8)
	result[4] = byte(tlsLen & 0xFF)

	// Add original data as application data
	if p.HTTPSNoise {
		// Add some noise
		noise := make([]byte, p.rands.Intn(64)+32)
		rand.Read(noise)
		result = append(result, noise...)
	}

	result = append(result, data...)

	return result
}

func (p *ProtocolMaskModifier) maskAsTLS(data []byte) []byte {
	// Simple TLS application data masking
	var result []byte

	// TLS application data record
	tlsRecord := []byte{
		0x17, // Application data
		0x03, // TLS version major
		0x03, // TLS version minor (TLS 1.2)
		0x00, // Length high byte
		0x00, // Length low byte (will be set later)
	}

	// Add some padding to make it look like encrypted data
	padding := make([]byte, p.rands.Intn(16)+16)
	rand.Read(padding)

	result = append(result, tlsRecord...)
	result = append(result, padding...)
	result = append(result, data...)

	// Set length
	dataLen := len(padding) + len(data)
	result[3] = byte(dataLen >> 8)
	result[4] = byte(dataLen & 0xFF)

	return result
}
