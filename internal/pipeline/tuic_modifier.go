package pipeline

import (
	"context"
	"fmt"
	"sync"

	"socks5-dpi-proxy/internal/protocols/tuic"
)

// TUICModifier implements pipeline.Modifier interface for TUIC protocol
type TUICModifier struct {
	client *tuic.TUICClient
	config map[string]interface{}
	mu     sync.RWMutex
}

// NewTUICModifier creates new TUIC modifier
func NewTUICModifier() *TUICModifier {
	return &TUICModifier{
		config: make(map[string]interface{}),
	}
}

// Name returns modifier name
func (t *TUICModifier) Name() string {
	return "tuic_client"
}

// Configure configures TUIC modifier
func (t *TUICModifier) Configure(config map[string]interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.config = config

	// Parse TUIC configuration
	tuicConfig, err := t.parseTUICConfig(config)
	if err != nil {
		return fmt.Errorf("failed to parse TUIC config: %w", err)
	}

	// Create TUIC client
	t.client = tuic.NewTUICClient(tuicConfig)

	return nil
}

// Process processes data through TUIC tunnel
func (t *TUICModifier) Process(data []byte, direction Direction) []byte {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Only process outbound traffic
	if direction == DirectionInbound {
		return data
	}

	// Connect if not connected
	if t.client == nil || !t.client.IsConnected() {
		if err := t.client.Connect(context.Background()); err != nil {
			// Return original data if connection fails
			return data
		}
	}

	// Send data via TUIC
	if err := t.client.SendData(data); err != nil {
		// Try to reconnect and send again
		t.client.Close()
		if err := t.client.Connect(context.Background()); err != nil {
			return data
		}
		if err := t.client.SendData(data); err != nil {
			return data
		}
	}

	// For now, pass data through without modification
	// In a full TUIC implementation, this would handle tunneling
	// For basic functionality, return original data to allow HTTP requests to work
	return data
}

// parseTUICConfig parses configuration from map
func (t *TUICModifier) parseTUICConfig(config map[string]interface{}) (*tuic.TUICConfig, error) {
	tuicConfig := tuic.CreateTUICConfig()

	// Parse server
	if server, ok := config["server"].(string); ok {
		tuicConfig.Server = server
	} else {
		return nil, fmt.Errorf("server is required")
	}

	// Parse port
	if port, ok := config["port"].(int); ok {
		tuicConfig.Port = port
	}

	// Parse UUID
	if uuid, ok := config["uuid"].(string); ok {
		tuicConfig.UUID = uuid
	} else {
		return nil, fmt.Errorf("uuid is required")
	}

	// Parse password
	if password, ok := config["password"].(string); ok {
		tuicConfig.Password = password
	}

	// Parse IP version
	if ipVersion, ok := config["ip_version"].(int); ok {
		tuicConfig.IPVersion = ipVersion
	}

	// Parse congestion control
	if congestion, ok := config["congestion_control"].(string); ok {
		tuicConfig.Congestion = congestion
	}

	return tuicConfig, nil
}

// GetServer returns configured TUIC server
func (t *TUICModifier) GetServer() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if server, ok := t.config["server"].(string); ok {
		return server
	}
	return ""
}

// GetPort returns configured TUIC port
func (t *TUICModifier) GetPort() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if port, ok := t.config["port"].(int); ok {
		return port
	}
	return 443
}

// GetUUID returns configured UUID
func (t *TUICModifier) GetUUID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if uuid, ok := t.config["uuid"].(string); ok {
		return uuid
	}
	return ""
}

// IsEnabled returns if TUIC is properly configured
func (t *TUICModifier) IsEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	_, hasServer := t.config["server"].(string)
	_, hasUUID := t.config["uuid"].(string)

	return hasServer && hasUUID
}
