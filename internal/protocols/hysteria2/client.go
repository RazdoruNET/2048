package hysteria2

import (
	"context"
	"fmt"
	"sync"

	"socks5-dpi-proxy/internal/protocols"
)

// Hysteria2Client implements Hysteria2 protocol client
type Hysteria2Client struct {
	transport   *protocols.QUICTransport
	serverAddr  string
	password    string
	obfuscation string
	mu          sync.RWMutex
	connected   bool
}

// Hysteria2Config represents Hysteria2 configuration
type Hysteria2Config struct {
	Server      string `yaml:"server"`
	Port        int    `yaml:"port"`
	Password    string `yaml:"password"`
	Obfuscation string `yaml:"obfuscation"`
	UpMbps      int    `yaml:"up_mbps"`
	DownMbps    int    `yaml:"down_mbps"`
}

// NewHysteria2Client creates new Hysteria2 client
func NewHysteria2Client(config *Hysteria2Config) *Hysteria2Client {
	serverAddr := fmt.Sprintf("%s:%d", config.Server, config.Port)

	// Create TLS configuration for Hysteria2
	tlsConfig := protocols.CreateTLSConfigForQUIC(config.Server)

	// Create QUIC transport
	transport := protocols.NewQUICTransport(serverAddr, tlsConfig)

	return &Hysteria2Client{
		transport:   transport,
		serverAddr:  serverAddr,
		password:    config.Password,
		obfuscation: config.Obfuscation,
		connected:   false,
	}
}

// Connect establishes Hysteria2 connection
func (h *Hysteria2Client) Connect(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connected {
		return nil
	}

	// Connect via QUIC transport
	if err := h.transport.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect Hysteria2: %w", err)
	}

	// Perform Hysteria2 handshake
	if err := h.performHandshake(); err != nil {
		h.transport.Close()
		return fmt.Errorf("Hysteria2 handshake failed: %w", err)
	}

	h.connected = true
	return nil
}

// SendData sends data through Hysteria2 connection
func (h *Hysteria2Client) SendData(data []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	// Apply Hysteria2 congestion control
	data = h.applyCongestionControl(data)

	// Send through QUIC transport
	return h.transport.SendData(data)
}

// ReceiveData receives data from Hysteria2 connection
func (h *Hysteria2Client) ReceiveData(buffer []byte) (int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return 0, fmt.Errorf("not connected")
	}

	// Receive through QUIC transport
	return h.transport.ReceiveData(buffer)
}

// IsConnected returns connection status
func (h *Hysteria2Client) IsConnected() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return false
	}

	// Check QUIC transport status
	return h.transport.IsConnected()
}

// Close closes Hysteria2 connection
func (h *Hysteria2Client) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return nil
	}

	h.connected = false
	return h.transport.Close()
}

// performHandshake performs Hysteria2 protocol handshake
func (h *Hysteria2Client) performHandshake() error {
	// Hysteria2 handshake format:
	// 1. Send client hello with obfuscation
	// 2. Receive server response
	// 3. Send authentication
	// 4. Receive session confirmation

	// For now, simplified handshake
	clientHello := h.buildHandshake()

	// Send client hello
	if err := h.transport.SendData(clientHello); err != nil {
		return fmt.Errorf("failed to send client hello: %w", err)
	}

	buffer := make([]byte, 1024)
	n, err := h.transport.ReceiveData(buffer)
	if err != nil {
		return fmt.Errorf("failed to receive handshake response: %w", err)
	}

	// Process server response
	if !h.validateHandshakeResponse(buffer[:n]) {
		return fmt.Errorf("invalid handshake response")
	}

	// Send authentication
	auth := h.buildAuthentication()
	if err := h.transport.SendData(auth); err != nil {
		return fmt.Errorf("failed to send authentication: %w", err)
	}

	return nil
}

// buildHandshake builds Hysteria2 handshake packet
func (h *Hysteria2Client) buildHandshake() []byte {
	// Simplified Hysteria2 handshake
	handshake := []byte{
		0x48, // 'H'
		0x59, // 'Y'
		0x53, // 'S'
		0x32, // '2'
		0x01, // Version
		0x00, // Reserved
	}

	// Add obfuscation
	if h.obfuscation != "" {
		obfuscated := h.obfuscateData(handshake)
		handshake = obfuscated
	}

	return handshake
}

// buildAuthentication builds Hysteria2 authentication packet
func (h *Hysteria2Client) buildAuthentication() []byte {
	// Simplified authentication
	auth := []byte{
		0x41, // 'A'
		0x55, // 'U'
		0x54, // 'T'
		0x48, // 'H'
	}

	// Add password if available
	if h.password != "" {
		passwordBytes := []byte(h.password)
		auth = append(auth, passwordBytes...)
	}

	return auth
}

// validateHandshakeResponse validates server handshake response
func (h *Hysteria2Client) validateHandshakeResponse(response []byte) bool {
	if len(response) < 4 {
		return false
	}

	// Check for Hysteria2 response magic
	magic := response[:4]
	return string(magic) == "HYS2"
}

// obfuscateData applies Hysteria2 obfuscation
func (h *Hysteria2Client) obfuscateData(data []byte) []byte {
	// Simple XOR obfuscation for now
	obfuscated := make([]byte, len(data))
	key := byte(len(h.obfuscation))

	for i, b := range data {
		obfuscated[i] = b ^ key
	}

	return obfuscated
}

// applyCongestionControl applies Hysteria2 congestion control
func (h *Hysteria2Client) applyCongestionControl(data []byte) []byte {
	// Simplified congestion control
	// In real implementation, this would use BBR or similar algorithm

	maxPacketSize := 1400 // Hysteria2 typical MTU
	if len(data) > maxPacketSize {
		return data[:maxPacketSize]
	}

	return data
}

// GetStats returns connection statistics
func (h *Hysteria2Client) GetStats() *Hysteria2Stats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := &Hysteria2Stats{
		Connected:  h.connected,
		ServerAddr: h.serverAddr,
	}

	if h.transport != nil {
		quicStats := h.transport.GetStats()
		stats.LocalAddr = quicStats.LocalAddr
		stats.RemoteAddr = quicStats.RemoteAddr
	}

	return stats
}

// Hysteria2Stats represents Hysteria2 connection statistics
type Hysteria2Stats struct {
	Connected  bool
	ServerAddr string
	LocalAddr  string
	RemoteAddr string
}

// CreateHysteria2Config creates default Hysteria2 configuration
func CreateHysteria2Config() *Hysteria2Config {
	return &Hysteria2Config{
		Port:        443,
		Obfuscation: "xchacha20-ietf-poly1305",
		UpMbps:      100,
		DownMbps:    100,
	}
}
