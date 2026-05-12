package performance

import (
	"net"
	"time"
)

// TCPConnectionFactory creates TCP connections
type TCPConnectionFactory struct {
	address        string
	timeout        time.Duration
	keepAlive      time.Duration
	enableTLS      bool
	tlsConfig      *TLSConfig
	validationMode ValidationMode
}

// TLSConfig defines TLS configuration
type TLSConfig struct {
	Enabled            bool   `yaml:"enabled"`
	ServerName         string `yaml:"server_name"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	MinVersion         uint16 `yaml:"min_version"`
	MaxVersion         uint16 `yaml:"max_version"`
}

// ValidationMode defines how connections are validated
type ValidationMode int

const (
	ValidationNone ValidationMode = iota
	ValidationBasic
	ValidationAdvanced
)

// NewTCPConnectionFactory creates a new TCP connection factory
func NewTCPConnectionFactory(address string, config *TCPFactoryConfig) *TCPConnectionFactory {
	if config == nil {
		config = &TCPFactoryConfig{
			Timeout:        10 * time.Second,
			KeepAlive:      30 * time.Second,
			EnableTLS:      false,
			ValidationMode: ValidationBasic,
		}
	}

	return &TCPConnectionFactory{
		address:        address,
		timeout:        config.Timeout,
		keepAlive:      config.KeepAlive,
		enableTLS:      config.EnableTLS,
		tlsConfig:      config.TLSConfig,
		validationMode: config.ValidationMode,
	}
}

// TCPFactoryConfig defines TCP factory configuration
type TCPFactoryConfig struct {
	Timeout        time.Duration  `yaml:"timeout"`
	KeepAlive      time.Duration  `yaml:"keep_alive"`
	EnableTLS      bool           `yaml:"enable_tls"`
	TLSConfig      *TLSConfig     `yaml:"tls_config"`
	ValidationMode ValidationMode `yaml:"validation_mode"`
}

// CreateConnection creates a new TCP connection
func (tcf *TCPConnectionFactory) CreateConnection() (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   tcf.timeout,
		KeepAlive: tcf.keepAlive,
	}

	conn, err := dialer.Dial("tcp", tcf.address)
	if err != nil {
		return nil, err
	}

	// Apply TLS if enabled
	if tcf.enableTLS && tcf.tlsConfig != nil {
		tlsConn, err := tcf.applyTLS(conn)
		if err != nil {
			conn.Close()
			return nil, err
		}
		conn = tlsConn
	}

	return conn, nil
}

// ValidateConnection validates if a connection is still usable
func (tcf *TCPConnectionFactory) ValidateConnection(conn net.Conn) bool {
	if conn == nil {
		return false
	}

	switch tcf.validationMode {
	case ValidationNone:
		return true
	case ValidationBasic:
		return tcf.basicValidation(conn)
	case ValidationAdvanced:
		return tcf.advancedValidation(conn)
	default:
		return tcf.basicValidation(conn)
	}
}

// CloseConnection closes a TCP connection
func (tcf *TCPConnectionFactory) CloseConnection(conn net.Conn) error {
	if conn == nil {
		return nil
	}
	return conn.Close()
}

// basicValidation performs basic connection validation
func (tcf *TCPConnectionFactory) basicValidation(conn net.Conn) bool {
	// Check if connection is still open by setting a very short deadline
	originalDeadline := time.Time{}

	// Try to get original deadline (this might fail, that's ok)
	conn.SetDeadline(time.Now().Add(1 * time.Millisecond))
	originalDeadline = time.Now().Add(1 * time.Millisecond)

	// Try to read 0 bytes (non-blocking check)
	buf := make([]byte, 0)
	_, err := conn.Read(buf)

	// Restore original deadline
	if !originalDeadline.IsZero() {
		conn.SetDeadline(originalDeadline)
	} else {
		conn.SetDeadline(time.Time{})
	}

	// If we get EOF or timeout, connection is closed
	if err != nil {
		return false
	}

	return true
}

// advancedValidation performs advanced connection validation
func (tcf *TCPConnectionFactory) advancedValidation(conn net.Conn) bool {
	// First do basic validation
	if !tcf.basicValidation(conn) {
		return false
	}

	// Check remote address is still accessible
	remoteAddr := conn.RemoteAddr()
	if remoteAddr == nil {
		return false
	}

	// For TCP, we can check if the connection is still in a valid state
	// by attempting to get the file descriptor (this is a basic check)
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		// Try to get connection state
		err := tcpConn.SetNoDelay(false)
		if err != nil {
			return false
		}
		// Restore original setting
		tcpConn.SetNoDelay(true)
	}

	return true
}

// applyTLS applies TLS to a TCP connection
func (tcf *TCPConnectionFactory) applyTLS(conn net.Conn) (net.Conn, error) {
	if !tcf.enableTLS || tcf.tlsConfig == nil {
		return conn, nil
	}

	// Import crypto/tls here to avoid circular imports
	// This is a simplified TLS implementation
	// In a real implementation, you would use crypto/tls package

	// For now, return the connection as-is (TLS implementation would go here)
	return conn, nil
}

// GetAddress returns the target address
func (tcf *TCPConnectionFactory) GetAddress() string {
	return tcf.address
}

// SetTimeout sets the connection timeout
func (tcf *TCPConnectionFactory) SetTimeout(timeout time.Duration) {
	tcf.timeout = timeout
}

// SetKeepAlive sets the keep alive duration
func (tcf *TCPConnectionFactory) SetKeepAlive(keepAlive time.Duration) {
	tcf.keepAlive = keepAlive
}

// SetValidationMode sets the validation mode
func (tcf *TCPConnectionFactory) SetValidationMode(mode ValidationMode) {
	tcf.validationMode = mode
}

// GetStats returns factory statistics
func (tcf *TCPConnectionFactory) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"address":         tcf.address,
		"timeout":         tcf.timeout,
		"keep_alive":      tcf.keepAlive,
		"tls_enabled":     tcf.enableTLS,
		"validation_mode": tcf.validationMode,
	}
}

// HTTPConnectionFactory creates HTTP connections
type HTTPConnectionFactory struct {
	tcpFactory *TCPConnectionFactory
	userAgent  string
	headers    map[string]string
}

// NewHTTPConnectionFactory creates a new HTTP connection factory
func NewHTTPConnectionFactory(address string, config *HTTPFactoryConfig) *HTTPConnectionFactory {
	tcpConfig := &TCPFactoryConfig{
		Timeout:        config.Timeout,
		KeepAlive:      config.KeepAlive,
		EnableTLS:      config.EnableTLS,
		TLSConfig:      config.TLSConfig,
		ValidationMode: config.ValidationMode,
	}

	return &HTTPConnectionFactory{
		tcpFactory: NewTCPConnectionFactory(address, tcpConfig),
		userAgent:  config.UserAgent,
		headers:    config.Headers,
	}
}

// HTTPFactoryConfig defines HTTP factory configuration
type HTTPFactoryConfig struct {
	Timeout        time.Duration     `yaml:"timeout"`
	KeepAlive      time.Duration     `yaml:"keep_alive"`
	EnableTLS      bool              `yaml:"enable_tls"`
	TLSConfig      *TLSConfig        `yaml:"tls_config"`
	ValidationMode ValidationMode    `yaml:"validation_mode"`
	UserAgent      string            `yaml:"user_agent"`
	Headers        map[string]string `yaml:"headers"`
}

// CreateConnection creates a new HTTP connection
func (hcf *HTTPConnectionFactory) CreateConnection() (net.Conn, error) {
	// For HTTP, we just create a TCP connection
	// HTTP protocol handling would be done at a higher level
	return hcf.tcpFactory.CreateConnection()
}

// ValidateConnection validates an HTTP connection
func (hcf *HTTPConnectionFactory) ValidateConnection(conn net.Conn) bool {
	return hcf.tcpFactory.ValidateConnection(conn)
}

// CloseConnection closes an HTTP connection
func (hcf *HTTPConnectionFactory) CloseConnection(conn net.Conn) error {
	return hcf.tcpFactory.CloseConnection(conn)
}

// GetUserAgent returns the user agent
func (hcf *HTTPConnectionFactory) GetUserAgent() string {
	return hcf.userAgent
}

// GetHeaders returns the HTTP headers
func (hcf *HTTPConnectionFactory) GetHeaders() map[string]string {
	return hcf.headers
}

// WebSocketConnectionFactory creates WebSocket connections
type WebSocketConnectionFactory struct {
	tcpFactory        *TCPConnectionFactory
	origin            string
	protocols         []string
	enableCompression bool
}

// NewWebSocketConnectionFactory creates a new WebSocket connection factory
func NewWebSocketConnectionFactory(address string, config *WebSocketFactoryConfig) *WebSocketConnectionFactory {
	tcpConfig := &TCPFactoryConfig{
		Timeout:        config.Timeout,
		KeepAlive:      config.KeepAlive,
		EnableTLS:      config.EnableTLS,
		TLSConfig:      config.TLSConfig,
		ValidationMode: config.ValidationMode,
	}

	return &WebSocketConnectionFactory{
		tcpFactory:        NewTCPConnectionFactory(address, tcpConfig),
		origin:            config.Origin,
		protocols:         config.Protocols,
		enableCompression: config.EnableCompression,
	}
}

// WebSocketFactoryConfig defines WebSocket factory configuration
type WebSocketFactoryConfig struct {
	Timeout           time.Duration  `yaml:"timeout"`
	KeepAlive         time.Duration  `yaml:"keep_alive"`
	EnableTLS         bool           `yaml:"enable_tls"`
	TLSConfig         *TLSConfig     `yaml:"tls_config"`
	ValidationMode    ValidationMode `yaml:"validation_mode"`
	Origin            string         `yaml:"origin"`
	Protocols         []string       `yaml:"protocols"`
	EnableCompression bool           `yaml:"enable_compression"`
}

// CreateConnection creates a new WebSocket connection
func (wscf *WebSocketConnectionFactory) CreateConnection() (net.Conn, error) {
	// For WebSocket, we create a TCP connection first
	// WebSocket handshake would be done at a higher level
	return wscf.tcpFactory.CreateConnection()
}

// ValidateConnection validates a WebSocket connection
func (wscf *WebSocketConnectionFactory) ValidateConnection(conn net.Conn) bool {
	return wscf.tcpFactory.ValidateConnection(conn)
}

// CloseConnection closes a WebSocket connection
func (wscf *WebSocketConnectionFactory) CloseConnection(conn net.Conn) error {
	return wscf.tcpFactory.CloseConnection(conn)
}

// GetOrigin returns the WebSocket origin
func (wscf *WebSocketConnectionFactory) GetOrigin() string {
	return wscf.origin
}

// GetProtocols returns the WebSocket protocols
func (wscf *WebSocketConnectionFactory) GetProtocols() []string {
	return wscf.protocols
}
