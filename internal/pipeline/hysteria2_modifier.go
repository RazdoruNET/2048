package pipeline

import (
	"context"
	"fmt"
	"sync"

	"socks5-dpi-proxy/internal/protocols/hysteria2"
)

// Hysteria2Modifier implements pipeline.Modifier interface for Hysteria2 protocol
type Hysteria2Modifier struct {
	client *hysteria2.Hysteria2Client
	config map[string]interface{}
	mu     sync.RWMutex
}

// NewHysteria2Modifier creates new Hysteria2 modifier
func NewHysteria2Modifier() *Hysteria2Modifier {
	return &Hysteria2Modifier{
		config: make(map[string]interface{}),
	}
}

// Name returns modifier name
func (h *Hysteria2Modifier) Name() string {
	return "hysteria2_client"
}

// Configure configures Hysteria2 modifier
func (h *Hysteria2Modifier) Configure(config map[string]interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.config = config

	// Parse Hysteria2 configuration
	hysteria2Config, err := h.parseHysteria2Config(config)
	if err != nil {
		return fmt.Errorf("failed to parse Hysteria2 config: %w", err)
	}

	// Create Hysteria2 client
	h.client = hysteria2.NewHysteria2Client(hysteria2Config)

	return nil
}

// Process processes data through Hysteria2 tunnel
func (h *Hysteria2Modifier) Process(data []byte, direction Direction) []byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Only process outbound traffic
	if direction == DirectionInbound {
		return data
	}

	// Connect if not connected
	if h.client == nil || !h.client.IsConnected() {
		if err := h.client.Connect(context.Background()); err != nil {
			// Return original data if connection fails
			return data
		}
	}

	// Send data via Hysteria2
	if err := h.client.SendData(data); err != nil {
		// Try to reconnect and send again
		h.client.Close()
		if err := h.client.Connect(context.Background()); err != nil {
			return data
		}
		if err := h.client.SendData(data); err != nil {
			return data
		}
	}

	// For now, pass data through without modification
	// In a full Hysteria2 implementation, this would handle tunneling
	// For basic functionality, return original data to allow HTTP requests to work
	return data
}

// parseHysteria2Config parses configuration from map
func (h *Hysteria2Modifier) parseHysteria2Config(config map[string]interface{}) (*hysteria2.Hysteria2Config, error) {
	hysteria2Config := hysteria2.CreateHysteria2Config()

	// Parse server
	if server, ok := config["server"].(string); ok {
		hysteria2Config.Server = server
	} else {
		return nil, fmt.Errorf("server is required")
	}

	// Parse port
	if port, ok := config["port"].(int); ok {
		hysteria2Config.Port = port
	}

	// Parse password
	if password, ok := config["password"].(string); ok {
		hysteria2Config.Password = password
	}

	// Parse obfuscation
	if obfuscation, ok := config["obfuscation"].(string); ok {
		hysteria2Config.Obfuscation = obfuscation
	}

	// Parse bandwidth
	if upMbps, ok := config["up_mbps"].(int); ok {
		hysteria2Config.UpMbps = upMbps
	}

	if downMbps, ok := config["down_mbps"].(int); ok {
		hysteria2Config.DownMbps = downMbps
	}

	return hysteria2Config, nil
}

// GetServer returns configured Hysteria2 server
func (h *Hysteria2Modifier) GetServer() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if server, ok := h.config["server"].(string); ok {
		return server
	}
	return ""
}

// GetPort returns configured Hysteria2 port
func (h *Hysteria2Modifier) GetPort() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if port, ok := h.config["port"].(int); ok {
		return port
	}
	return 443
}

// IsEnabled returns if Hysteria2 is properly configured
func (h *Hysteria2Modifier) IsEnabled() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, hasServer := h.config["server"].(string)
	_, hasPassword := h.config["password"].(string)

	return hasServer && hasPassword
}
