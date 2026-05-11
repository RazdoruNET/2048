package tuic

import (
	"context"
	"fmt"
	"sync"

	"socks5-dpi-proxy/internal/protocols"
)

// TUICClient implements TUIC protocol client
type TUICClient struct {
	transport  *protocols.QUICTransport
	serverAddr string
	uuid       string
	password   string
	mu         sync.RWMutex
	connected  bool
}

// TUICConfig represents TUIC configuration
type TUICConfig struct {
	Server     string `yaml:"server"`
	Port       int    `yaml:"port"`
	UUID       string `yaml:"uuid"`
	Password   string `yaml:"password"`
	IPVersion  int    `yaml:"ip_version"`
	Congestion string `yaml:"congestion_control"`
}

// NewTUICClient creates new TUIC client
func NewTUICClient(config *TUICConfig) *TUICClient {
	serverAddr := fmt.Sprintf("%s:%d", config.Server, config.Port)

	// Create TLS configuration for TUIC
	tlsConfig := protocols.CreateTLSConfigForQUIC(config.Server)

	// Create QUIC transport
	transport := protocols.NewQUICTransport(serverAddr, tlsConfig)

	return &TUICClient{
		transport:  transport,
		serverAddr: serverAddr,
		uuid:       config.UUID,
		password:   config.Password,
		connected:  false,
	}
}

// Connect establishes TUIC connection
func (t *TUICClient) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	// Connect via QUIC transport
	if err := t.transport.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect TUIC: %w", err)
	}

	// Perform TUIC handshake
	if err := t.performHandshake(); err != nil {
		t.transport.Close()
		return fmt.Errorf("TUIC handshake failed: %w", err)
	}

	t.connected = true
	return nil
}

// SendData sends data through TUIC connection
func (t *TUICClient) SendData(data []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return fmt.Errorf("not connected")
	}

	// Apply TUIC multiplexing
	data = t.applyMultiplexing(data)

	// Send through QUIC transport
	return t.transport.SendData(data)
}

// ReceiveData receives data from TUIC connection
func (t *TUICClient) ReceiveData(buffer []byte) (int, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return 0, fmt.Errorf("not connected")
	}

	// Receive through QUIC transport
	n, err := t.transport.ReceiveData(buffer)
	if err != nil {
		return 0, fmt.Errorf("failed to receive data: %w", err)
	}

	// Apply TUIC demultiplexing
	return t.applyDemultiplexing(buffer[:n])
}

// IsConnected returns connection status
func (t *TUICClient) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return false
	}

	// Check QUIC transport status
	return t.transport.IsConnected()
}

// Close closes TUIC connection
func (t *TUICClient) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	t.connected = false
	return t.transport.Close()
}

// performHandshake performs TUIC protocol handshake
func (t *TUICClient) performHandshake() error {
	// TUIC handshake format:
	// 1. Send client hello with UUID
	// 2. Receive server challenge
	// 3. Send response with password
	// 4. Receive session confirmation

	// Send client hello
	clientHello := t.buildClientHello()
	if err := t.transport.SendData(clientHello); err != nil {
		return fmt.Errorf("failed to send client hello: %w", err)
	}

	// Receive server response
	buffer := make([]byte, 1024)
	n, err := t.transport.ReceiveData(buffer)
	if err != nil {
		return fmt.Errorf("failed to receive server response: %w", err)
	}

	// Process server challenge
	if !t.validateServerResponse(buffer[:n]) {
		return fmt.Errorf("invalid server response")
	}

	// Send authentication response
	authResponse := t.buildAuthResponse()
	if err := t.transport.SendData(authResponse); err != nil {
		return fmt.Errorf("failed to send auth response: %w", err)
	}

	return nil
}

// buildClientHello builds TUIC client hello packet
func (t *TUICClient) buildClientHello() []byte {
	// Simplified TUIC client hello
	hello := []byte{
		0x54, // 'T'
		0x55, // 'U'
		0x49, // 'I'
		0x43, // 'C'
		0x01, // Version 1
		0x00, // Reserved
	}

	// Add UUID
	uuidBytes := []byte(t.uuid)
	hello = append(hello, uuidBytes...)

	return hello
}

// buildAuthResponse builds TUIC authentication response
func (t *TUICClient) buildAuthResponse() []byte {
	// Simplified TUIC authentication
	auth := []byte{
		0x41, // 'A'
		0x55, // 'U'
		0x54, // 'T'
		0x48, // 'H'
		0x02, // Response type 2
		0x00, // Status success
	}

	// Add password if available
	if t.password != "" {
		passwordBytes := []byte(t.password)
		auth = append(auth, passwordBytes...)
	}

	return auth
}

// validateServerResponse validates TUIC server response
func (t *TUICClient) validateServerResponse(response []byte) bool {
	if len(response) < 4 {
		return false
	}

	// Check for TUIC response magic
	magic := response[:4]
	return string(magic) == "TUIC"
}

// applyMultiplexing applies TUIC multiplexing
func (t *TUICClient) applyMultiplexing(data []byte) []byte {
	// Simplified TUIC multiplexing
	// Add stream ID and multiplexing header

	header := []byte{
		0x4D,                   // Stream header
		0x55,                   // 'U'
		0x49,                   // 'I'
		0x43,                   // 'C'
		byte(len(data) >> 8),   // Length high
		byte(len(data) & 0xFF), // Length low
	}

	result := make([]byte, 0, len(header)+len(data))
	result = append(result, header...)
	result = append(result, data...)

	return result
}

// applyDemultiplexing applies TUIC demultiplexing
func (t *TUICClient) applyDemultiplexing(data []byte) (int, error) {
	if len(data) < 6 {
		return 0, fmt.Errorf("invalid TUIC packet")
	}

	// Check TUIC header
	if data[0] != 0x4D || data[1] != 0x55 || data[2] != 0x49 || data[3] != 0x43 {
		return 0, fmt.Errorf("invalid TUIC header")
	}

	// Extract length
	length := int(data[4])<<8 | int(data[5])
	if len(data) < 6+length {
		return 0, fmt.Errorf("incomplete TUIC packet")
	}

	// Return payload
	copy(data, data[6:6+length])
	return length, nil
}

// GetStats returns connection statistics
func (t *TUICClient) GetStats() *TUICStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := &TUICStats{
		Connected:  t.connected,
		ServerAddr: t.serverAddr,
		UUID:       t.uuid,
	}

	if t.transport != nil {
		quicStats := t.transport.GetStats()
		stats.LocalAddr = quicStats.LocalAddr
		stats.RemoteAddr = quicStats.RemoteAddr
	}

	return stats
}

// TUICStats represents TUIC connection statistics
type TUICStats struct {
	Connected  bool
	ServerAddr string
	LocalAddr  string
	RemoteAddr string
	UUID       string
}

// CreateTUICConfig creates default TUIC configuration
func CreateTUICConfig() *TUICConfig {
	return &TUICConfig{
		Port:       443,
		IPVersion:  4,     // IPv4
		Congestion: "bbr", // BBR congestion control
	}
}
