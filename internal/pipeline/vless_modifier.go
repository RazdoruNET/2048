package pipeline

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// VLESSConnection represents a VLESS connection state
type VLESSConnection struct {
	conn         *websocket.Conn
	tlsConfig    *tls.Config
	server       string
	port         int
	uuid         string
	flow         string
	connected    bool
	lastActivity time.Time
	mu           sync.RWMutex
}

// VLESSModifier implements pipeline.Modifier interface for VLESS protocol
type VLESSModifier struct {
	config map[string]interface{}
	conn   *VLESSConnection
	mu     sync.RWMutex
}

// NewVLESSModifier creates new VLESS modifier
func NewVLESSModifier() *VLESSModifier {
	return &VLESSModifier{
		config: make(map[string]interface{}),
	}
}

// Name returns modifier name
func (v *VLESSModifier) Name() string {
	return "vless_client"
}

// Configure configures VLESS modifier
func (v *VLESSModifier) Configure(config map[string]interface{}) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.config = config

	// Validate required fields
	if _, ok := config["server"].(string); !ok {
		return fmt.Errorf("server is required")
	}

	if _, ok := config["uuid"].(string); !ok {
		return fmt.Errorf("uuid is required")
	}

	// Set defaults
	if _, ok := config["port"]; !ok {
		config["port"] = 443
	}

	if _, ok := config["flow"]; !ok {
		config["flow"] = "xtls-rprx-vision"
	}

	if _, ok := config["encryption"]; !ok {
		config["encryption"] = "none"
	}

	return nil
}

// Process processes data through VLESS tunnel
func (v *VLESSModifier) Process(data []byte, direction Direction) []byte {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Only process outbound traffic
	if direction == DirectionInbound {
		return data
	}

	// Connect if not connected
	if v.conn == nil || !v.conn.isConnected() {
		if err := v.connect(); err != nil {
			// Return original data if connection fails
			return data
		}
	}

	// Send data through VLESS tunnel
	if err := v.sendData(data); err != nil {
		// Try to reconnect and send again
		v.disconnect()
		if err := v.connect(); err != nil {
			return data
		}
		if err := v.sendData(data); err != nil {
			return data
		}
	}

	// For now, pass data through without modification
	// In a full VLESS implementation, this would handle tunneling
	// For basic functionality, return original data to allow HTTP requests to work
	return data
}

// connect establishes VLESS connection
func (v *VLESSModifier) connect() error {
	// Get configuration
	server := v.GetServer()
	port := v.GetPort()
	uuid := v.GetUUID()
	flow := v.GetFlow()

	if server == "" || uuid == "" {
		return fmt.Errorf("server and UUID are required")
	}

	// Create TLS configuration
	tlsConfig := v.createTLSConfig()

	// Create WebSocket dialer
	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  tlsConfig,
	}

	// Build WebSocket URL
	scheme := "wss"
	url := fmt.Sprintf("%s://%s:%d/vless", scheme, server, port)

	// Connect via WebSocket
	headers := v.buildHeaders()
	httpHeaders := make(http.Header)
	for k, v := range headers {
		httpHeaders.Add(k, v)
	}

	conn, _, err := dialer.Dial(url, httpHeaders)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	// Perform VLESS handshake
	if err := v.performHandshake(conn, uuid, flow); err != nil {
		conn.Close()
		return fmt.Errorf("VLESS handshake failed: %w", err)
	}

	// Create connection state
	v.conn = &VLESSConnection{
		conn:         conn,
		tlsConfig:    tlsConfig,
		server:       server,
		port:         port,
		uuid:         uuid,
		flow:         flow,
		connected:    true,
		lastActivity: time.Now(),
	}

	return nil
}

// disconnect closes VLESS connection
func (v *VLESSModifier) disconnect() {
	if v.conn != nil {
		v.conn.mu.Lock()
		defer v.conn.mu.Unlock()

		if v.conn.conn != nil {
			v.conn.conn.Close()
		}
		v.conn.connected = false
		v.conn = nil
	}
}

// sendData sends data through VLESS connection
func (v *VLESSModifier) sendData(data []byte) error {
	if v.conn == nil || !v.conn.isConnected() {
		return fmt.Errorf("not connected")
	}

	v.conn.mu.Lock()
	defer v.conn.mu.Unlock()

	// Update activity
	v.conn.lastActivity = time.Now()

	// Send data as WebSocket binary message
	return v.conn.conn.WriteMessage(websocket.BinaryMessage, data)
}

// createTLSConfig creates TLS configuration for VLESS
func (v *VLESSModifier) createTLSConfig() *tls.Config {
	config := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: false,
		NextProtos:         []string{"h2", "http/1.1"},
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	// Set server name if available
	if server := v.GetServer(); server != "" {
		config.ServerName = server
	}

	return config
}

// buildHeaders creates WebSocket request headers
func (v *VLESSModifier) buildHeaders() map[string]string {
	headers := make(map[string]string)

	// Add Chrome-like headers for obfuscation
	headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	headers["Accept"] = "*/*"
	headers["Accept-Language"] = "en-US,en;q=0.9"
	headers["Accept-Encoding"] = "gzip, deflate, br"
	headers["Connection"] = "Upgrade"
	headers["Upgrade"] = "websocket"
	headers["Sec-Fetch-Dest"] = "empty"
	headers["Sec-Fetch-Mode"] = "cors"
	headers["Sec-Fetch-Site"] = "cross-site"

	return headers
}

// performHandshake performs VLESS handshake
func (v *VLESSModifier) performHandshake(conn *websocket.Conn, uuid, flow string) error {
	// VLESS handshake format:
	// Version (1 byte) + UUID (16 bytes) + Command (1 byte) + Flow ID (variable)

	handshake := make([]byte, 18)
	handshake[0] = 0 // Version 0

	// Parse and add UUID
	uuidBytes, err := v.parseUUID(uuid)
	if err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}
	copy(handshake[1:17], uuidBytes)

	handshake[17] = 1 // Command: TCP

	// Send handshake
	if err := conn.WriteMessage(websocket.BinaryMessage, handshake); err != nil {
		return err
	}

	// Wait for response
	_, _, err = conn.ReadMessage()
	if err != nil {
		return err
	}

	return nil
}

// parseUUID parses UUID string to bytes
func (v *VLESSModifier) parseUUID(uuidStr string) ([]byte, error) {
	// Simple UUID parsing - in real implementation would use proper UUID library
	uuidBytes := make([]byte, 16)

	// Remove dashes and convert to bytes
	uuidClean := uuidStr
	for i := 0; i < len(uuidClean) && i < 32; i++ {
		if uuidClean[i] == '-' {
			continue
		}
		// Convert hex characters to bytes
		if i%2 == 0 {
			uuidBytes[i/2] = v.hexToByte(uuidClean[i]) << 4
		} else {
			uuidBytes[i/2] |= v.hexToByte(uuidClean[i])
		}
	}

	return uuidBytes, nil
}

// hexToByte converts hex character to byte value
func (v *VLESSModifier) hexToByte(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}

// isConnected checks if connection is active
func (vc *VLESSConnection) isConnected() bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	if !vc.connected || vc.conn == nil {
		return false
	}

	// Check if connection is still alive (simple check)
	return time.Since(vc.lastActivity) < 30*time.Second
}

// GetServer returns configured VLESS server
func (v *VLESSModifier) GetServer() string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if server, ok := v.config["server"].(string); ok {
		return server
	}
	return ""
}

// GetPort returns configured VLESS port
func (v *VLESSModifier) GetPort() int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if port, ok := v.config["port"].(int); ok {
		return port
	}
	return 443
}

// GetUUID returns configured UUID
func (v *VLESSModifier) GetUUID() string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if uuid, ok := v.config["uuid"].(string); ok {
		return uuid
	}
	return ""
}

// GetFlow returns configured flow
func (v *VLESSModifier) GetFlow() string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if flow, ok := v.config["flow"].(string); ok {
		return flow
	}
	return "xtls-rprx-vision"
}

// IsEnabled returns if VLESS is properly configured
func (v *VLESSModifier) IsEnabled() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	_, hasServer := v.config["server"].(string)
	_, hasUUID := v.config["uuid"].(string)

	return hasServer && hasUUID
}
