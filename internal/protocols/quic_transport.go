package protocols

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"
)

// QUICTransport provides basic QUIC transport layer for Hysteria2 and TUIC
type QUICTransport struct {
	conn         net.Conn
	serverAddr   string
	tlsConfig    *tls.Config
	mu           sync.RWMutex
	connected    bool
	lastActivity  time.Time
}

// NewQUICTransport creates new QUIC transport
func NewQUICTransport(serverAddr string, tlsConfig *tls.Config) *QUICTransport {
	return &QUICTransport{
		serverAddr:   serverAddr,
		tlsConfig:    tlsConfig,
		lastActivity:  time.Now(),
	}
}

// Connect establishes QUIC connection
func (qt *QUICTransport) Connect(ctx context.Context) error {
	qt.mu.Lock()
	defer qt.mu.Unlock()

	if qt.connected {
		return nil
	}

	// For now, use TCP+TLS as fallback for QUIC
	// In real implementation, this would be replaced with actual QUIC
	dialer := &tls.Dialer{
		Config: qt.tlsConfig,
	}

	conn, err := dialer.DialContext(ctx, "tcp", qt.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", qt.serverAddr, err)
	}

	qt.conn = conn
	qt.connected = true
	qt.lastActivity = time.Now()

	return nil
}

// SendData sends data through QUIC connection
func (qt *QUICTransport) SendData(data []byte) error {
	qt.mu.Lock()
	defer qt.mu.Unlock()

	if !qt.connected || qt.conn == nil {
		return fmt.Errorf("not connected")
	}

	qt.lastActivity = time.Now()

	_, err := qt.conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// ReceiveData receives data from QUIC connection
func (qt *QUICTransport) ReceiveData(buffer []byte) (int, error) {
	qt.mu.RLock()
	defer qt.mu.RUnlock()

	if !qt.connected || qt.conn == nil {
		return 0, fmt.Errorf("not connected")
	}

	n, err := qt.conn.Read(buffer)
	if err != nil {
		return 0, fmt.Errorf("failed to read data: %w", err)
	}

	qt.lastActivity = time.Now()
	return n, nil
}

// IsConnected returns connection status
func (qt *QUICTransport) IsConnected() bool {
	qt.mu.RLock()
	defer qt.mu.RUnlock()

	if !qt.connected || qt.conn == nil {
		return false
	}

	// Check if connection is still alive
	return time.Since(qt.lastActivity) < 30*time.Second
}

// Close closes QUIC connection
func (qt *QUICTransport) Close() error {
	qt.mu.Lock()
	defer qt.mu.Unlock()

	if !qt.connected {
		return nil
	}

	var err error
	if qt.conn != nil {
		err = qt.conn.Close()
		qt.conn = nil
	}

	qt.connected = false
	return err
}

// SetDeadline sets read/write deadline
func (qt *QUICTransport) SetDeadline(t time.Time) error {
	qt.mu.RLock()
	defer qt.mu.RUnlock()

	if qt.conn == nil {
		return fmt.Errorf("connection not available")
	}

	return qt.conn.SetDeadline(t)
}

// LocalAddr returns local address
func (qt *QUICTransport) LocalAddr() net.Addr {
	qt.mu.RLock()
	defer qt.mu.RUnlock()

	if qt.conn == nil {
		return nil
	}

	return qt.conn.LocalAddr()
}

// RemoteAddr returns remote address
func (qt *QUICTransport) RemoteAddr() net.Addr {
	qt.mu.RLock()
	defer qt.mu.RUnlock()

	if qt.conn == nil {
		return nil
	}

	return qt.conn.RemoteAddr()
}

// GetStats returns connection statistics
func (qt *QUICTransport) GetStats() *QUICStats {
	qt.mu.RLock()
	defer qt.mu.RUnlock()

	stats := &QUICStats{
		Connected:    qt.connected,
		LastActivity: qt.lastActivity,
	}

	if qt.conn != nil {
		stats.LocalAddr = qt.conn.LocalAddr().String()
		stats.RemoteAddr = qt.conn.RemoteAddr().String()
	}

	return stats
}

// QUICStats represents QUIC connection statistics
type QUICStats struct {
	LocalAddr    string
	RemoteAddr   string
	Connected    bool
	LastActivity time.Time
}

// CreateQUICConfig creates QUIC configuration for protocols
func CreateQUICConfig() map[string]interface{} {
	return map[string]interface{}{
		"initial_stream_receive_window":     1024 * 1024,
		"max_stream_receive_window":         16 * 1024 * 1024,
		"initial_connection_receive_window": 1024 * 1024,
		"max_connection_receive_window":     16 * 1024 * 1024,
		"max_idle_timeout":                30 * time.Second,
		"max_incoming_streams":            100,
		"max_incoming_uni_streams":       100,
		"keep_alive":                     true,
		"versions":                      []string{"v1", "v2"},
	}
}

// CreateTLSConfigForQUIC creates TLS configuration for QUIC
func CreateTLSConfigForQUIC(serverName string) *tls.Config {
	return &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		NextProtos:         []string{"h3"},
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}
}
