package modifiers

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConfig represents WebSocket transport configuration
type WebSocketConfig struct {
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
	Host    string            `yaml:"host"`
	Origin  string            `yaml:"origin"`
}

// WebSocketTransport handles WebSocket connections for VLESS
type WebSocketTransport struct {
	dialer  *websocket.Dialer
	config  *WebSocketConfig
	tlsConf *tls.Config
}

// NewWebSocketTransport creates a new WebSocket transport
func NewWebSocketTransport(config *WebSocketConfig, tlsConf *tls.Config) *WebSocketTransport {
	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  tlsConf,
	}

	return &WebSocketTransport{
		dialer:  dialer,
		config:  config,
		tlsConf: tlsConf,
	}
}

// Connect establishes WebSocket connection
func (wt *WebSocketTransport) Connect(serverAddr string, port int) (*websocket.Conn, error) {
	// Build WebSocket URL
	scheme := "ws"
	if wt.tlsConf != nil {
		scheme = "wss"
	}

	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", serverAddr, port),
		Path:   wt.config.Path,
	}

	// Prepare request headers
	headers := wt.buildHeaders()

	// Establish connection
	conn, resp, err := wt.dialer.Dial(u.String(), headers)
	if err != nil {
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	// Check response
	if resp.StatusCode != 101 {
		conn.Close()
		return nil, fmt.Errorf("websocket handshake failed: %s", resp.Status)
	}

	return conn, nil
}

// buildHeaders creates WebSocket request headers
func (wt *WebSocketTransport) buildHeaders() http.Header {
	headers := make(http.Header)

	// Add custom headers from config
	for key, value := range wt.config.Headers {
		headers.Set(key, value)
	}

	// Set default headers if not provided
	if headers.Get("User-Agent") == "" {
		headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	}

	if wt.config.Host != "" {
		headers.Set("Host", wt.config.Host)
	}

	if wt.config.Origin != "" {
		headers.Set("Origin", wt.config.Origin)
	}

	// Add standard WebSocket headers
	headers.Set("Connection", "Upgrade")
	headers.Set("Upgrade", "websocket")
	headers.Set("Sec-WebSocket-Version", "13")

	// Add Chrome-like headers for better obfuscation
	if headers.Get("Accept-Language") == "" {
		headers.Set("Accept-Language", "en-US,en;q=0.9")
	}

	if headers.Get("Accept-Encoding") == "" {
		headers.Set("Accept-Encoding", "gzip, deflate, br")
	}

	if headers.Get("Sec-Fetch-Dest") == "" {
		headers.Set("Sec-Fetch-Dest", "empty")
	}

	if headers.Get("Sec-Fetch-Mode") == "" {
		headers.Set("Sec-Fetch-Mode", "cors")
	}

	if headers.Get("Sec-Fetch-Site") == "" {
		headers.Set("Sec-Fetch-Site", "cross-site")
	}

	return headers
}

// WriteMessage writes data to WebSocket connection
func (wt *WebSocketTransport) WriteMessage(conn *websocket.Conn, data []byte) error {
	// Use binary message type for VLESS data
	return conn.WriteMessage(websocket.BinaryMessage, data)
}

// ReadMessage reads data from WebSocket connection
func (wt *WebSocketTransport) ReadMessage(conn *websocket.Conn) ([]byte, error) {
	messageType, data, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	// Only accept binary messages
	if messageType != websocket.BinaryMessage {
		return nil, fmt.Errorf("unexpected message type: %d", messageType)
	}

	return data, nil
}

// SetReadDeadline sets read deadline for connection
func (wt *WebSocketTransport) SetReadDeadline(conn *websocket.Conn, deadline time.Time) error {
	return conn.SetReadDeadline(deadline)
}

// SetWriteDeadline sets write deadline for connection
func (wt *WebSocketTransport) SetWriteDeadline(conn *websocket.Conn, deadline time.Time) error {
	return conn.SetWriteDeadline(deadline)
}

// Close closes WebSocket connection
func (wt *WebSocketTransport) Close(conn *websocket.Conn) error {
	if conn != nil {
		// Send close message
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			return err
		}
		return conn.Close()
	}
	return nil
}

// Ping sends ping to keep connection alive
func (wt *WebSocketTransport) Ping(conn *websocket.Conn) error {
	return conn.WriteMessage(websocket.PingMessage, []byte{})
}

// SetPingHandler sets ping handler for connection
func (wt *WebSocketTransport) SetPingHandler(conn *websocket.Conn, handler func(appData string) error) {
	conn.SetPingHandler(func(appData string) error {
		return handler(appData)
	})
}

// SetPongHandler sets pong handler for connection
func (wt *WebSocketTransport) SetPongHandler(conn *websocket.Conn, handler func(appData string) error) {
	conn.SetPongHandler(func(appData string) error {
		return handler(appData)
	})
}

// DefaultWebSocketConfig returns default WebSocket configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		Path: "/",
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Accept":     "*/*",
			"Connection": "Upgrade",
			"Upgrade":    "websocket",
		},
	}
}

// ChromeWebSocketConfig returns WebSocket config mimicking Chrome browser
func ChromeWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		Path: "/",
		Headers: map[string]string{
			"User-Agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Accept":            "*/*",
			"Accept-Language":   "en-US,en;q=0.9",
			"Accept-Encoding":   "gzip, deflate, br",
			"Connection":        "Upgrade",
			"Upgrade":           "websocket",
			"Sec-Fetch-Dest":    "empty",
			"Sec-Fetch-Mode":    "cors",
			"Sec-Fetch-Site":    "cross-site",
			"Sec-WebSocket-Key":  "dGhlIHNhbXBsZSBub25jZQ==", // Example key
		},
	}
}

// Validate validates WebSocket configuration
func (wc *WebSocketConfig) Validate() error {
	if wc.Path == "" {
		return fmt.Errorf("WebSocket path cannot be empty")
	}

	if !isPathValid(wc.Path) {
		return fmt.Errorf("invalid WebSocket path: %s", wc.Path)
	}

	return nil
}

// isPathValid checks if WebSocket path is valid
func isPathValid(path string) bool {
	if len(path) == 0 {
		return false
	}

	if path[0] != '/' {
		return false
	}

	// Basic path validation
	for _, char := range path {
		if char < 32 || char > 126 {
			return false
		}
	}

	return true
}

// Clone creates a copy of WebSocketConfig
func (wc *WebSocketConfig) Clone() *WebSocketConfig {
	if wc == nil {
		return nil
	}

	clone := *wc
	if wc.Headers != nil {
		clone.Headers = make(map[string]string)
		for k, v := range wc.Headers {
			clone.Headers[k] = v
		}
	}

	return &clone
}
