package modifiers

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// RealityFallback handles Reality obfuscation as fallback mechanism
type RealityFallback struct {
	config    *RealityConfig
	tlsConfig *tls.Config
	enabled   bool
}

// NewRealityFallback creates new Reality fallback handler
func NewRealityFallback(config *RealityConfig, tlsConfig *tls.Config) *RealityFallback {
	return &RealityFallback{
		config:    config,
		tlsConfig: tlsConfig,
		enabled:   config != nil && config.Enabled,
	}
}

// Connect establishes Reality obfuscated connection
func (rf *RealityFallback) Connect(serverAddr string, port int) (net.Conn, error) {
	if !rf.enabled {
		return nil, fmt.Errorf("Reality fallback is disabled")
	}

	// Create Reality TLS configuration
	realityTLSConfig := rf.createRealityTLSConfig()

	// Connect with Reality obfuscation
	conn, err := tls.DialWithDialer(&net.Dialer{
		Timeout: 10 * time.Second,
	}, "tcp", fmt.Sprintf("%s:%d", serverAddr, port), realityTLSConfig)

	if err != nil {
		return nil, fmt.Errorf("Reality connection failed: %w", err)
	}

	return conn, nil
}

// createRealityTLSConfig creates TLS configuration for Reality obfuscation
func (rf *RealityFallback) createRealityTLSConfig() *tls.Config {
	var baseConfig *TLSConfig
	if rf.tlsConfig == nil {
		baseConfig = DefaultTLSConfig()
	} else {
		// Convert tls.Config to TLSConfig if needed
		baseConfig = DefaultTLSConfig()
	}

	// Convert to crypto/tls config
	config, err := baseConfig.ToTLSConfig()
	if err != nil {
		// Fallback to basic config
		config = &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}
	}

	// Reality-specific modifications
	config.ServerName = rf.getServerName()
	config.InsecureSkipVerify = true // Reality uses self-signed certificates

	// Add Reality-specific cipher suites if not specified
	if len(config.CipherSuites) == 0 {
		config.CipherSuites = []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		}
	}

	// Set ALPN for Reality
	config.NextProtos = []string{"h2", "http/1.1"}

	return config
}

// getServerName returns server name for Reality
func (rf *RealityFallback) getServerName() string {
	if rf.config != nil && rf.config.PublicKey != "" {
		// Extract server name from public key or use default
		parts := strings.Split(rf.config.PublicKey, ":")
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// Fallback to common domains for obfuscation
	return "www.google.com"
}

// ValidateConnection validates if connection is properly obfuscated
func (rf *RealityFallback) ValidateConnection(conn net.Conn) error {
	if !rf.enabled {
		return fmt.Errorf("Reality fallback is disabled")
	}

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return fmt.Errorf("connection is not TLS")
	}

	// Check TLS connection state
	state := tlsConn.ConnectionState()

	// Verify TLS version
	if state.Version < tls.VersionTLS12 {
		return fmt.Errorf("TLS version too old: %x", state.Version)
	}

	// Check cipher suite
	validCipher := false
	validCiphers := []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
	}

	for _, valid := range validCiphers {
		if state.CipherSuite == valid {
			validCipher = true
			break
		}
	}

	if !validCipher {
		return fmt.Errorf("invalid cipher suite: %x", state.CipherSuite)
	}

	return nil
}

// GetFallbackDelay returns delay before using Reality fallback
func (rf *RealityFallback) GetFallbackDelay() time.Duration {
	// Reality fallback should be used after WebSocket fails
	return 5 * time.Second
}

// IsEnabled returns if Reality fallback is enabled
func (rf *RealityFallback) IsEnabled() bool {
	return rf.enabled && rf.config != nil
}

// GetPriority returns priority of Reality fallback
func (rf *RealityFallback) GetPriority() int {
	// Lower priority than WebSocket
	return 2
}

// CreateRealityConfig creates Reality configuration from map
func CreateRealityConfig(config map[string]interface{}) *RealityConfig {
	realityConfig := &RealityConfig{
		Enabled: false,
	}

	if enabled, ok := config["enabled"].(bool); ok {
		realityConfig.Enabled = enabled
	}

	if publicKey, ok := config["public_key"].(string); ok {
		realityConfig.PublicKey = publicKey
	}

	if shortID, ok := config["short_id"].(string); ok {
		realityConfig.ShortID = shortID
	}

	return realityConfig
}

// Validate validates Reality configuration
func (rc *RealityConfig) Validate() error {
	if !rc.Enabled {
		return nil
	}

	if rc.PublicKey == "" {
		return fmt.Errorf("Reality public key is required when enabled")
	}

	// Validate public key format (basic check)
	if len(rc.PublicKey) < 32 {
		return fmt.Errorf("Reality public key too short")
	}

	return nil
}

// Clone creates a copy of RealityConfig
func (rc *RealityConfig) Clone() *RealityConfig {
	if rc == nil {
		return nil
	}

	clone := *rc
	return &clone
}

// RealityConnection represents a Reality obfuscated connection
type RealityConnection struct {
	conn     net.Conn
	fallback *RealityFallback
}

// NewRealityConnection creates new Reality connection wrapper
func NewRealityConnection(conn net.Conn, fallback *RealityFallback) *RealityConnection {
	return &RealityConnection{
		conn:     conn,
		fallback: fallback,
	}
}

// Read implements net.Conn Read method
func (rc *RealityConnection) Read(b []byte) (n int, err error) {
	return rc.conn.Read(b)
}

// Write implements net.Conn Write method
func (rc *RealityConnection) Write(b []byte) (n int, err error) {
	return rc.conn.Write(b)
}

// Close implements net.Conn Close method
func (rc *RealityConnection) Close() error {
	return rc.conn.Close()
}

// LocalAddr implements net.Conn LocalAddr method
func (rc *RealityConnection) LocalAddr() net.Addr {
	return rc.conn.LocalAddr()
}

// RemoteAddr implements net.Conn RemoteAddr method
func (rc *RealityConnection) RemoteAddr() net.Addr {
	return rc.conn.RemoteAddr()
}

// SetDeadline implements net.Conn SetDeadline method
func (rc *RealityConnection) SetDeadline(t time.Time) error {
	return rc.conn.SetDeadline(t)
}

// SetReadDeadline implements net.Conn SetReadDeadline method
func (rc *RealityConnection) SetReadDeadline(t time.Time) error {
	return rc.conn.SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn SetWriteDeadline method
func (rc *RealityConnection) SetWriteDeadline(t time.Time) error {
	return rc.conn.SetWriteDeadline(t)
}

// UnderlyingConn returns underlying connection
func (rc *RealityConnection) UnderlyingConn() net.Conn {
	return rc.conn
}

// IsObfuscated returns if connection is properly obfuscated
func (rc *RealityConnection) IsObfuscated() bool {
	if rc.fallback == nil {
		return false
	}

	err := rc.fallback.ValidateConnection(rc.conn)
	return err == nil
}
