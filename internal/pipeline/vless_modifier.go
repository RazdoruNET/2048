package pipeline

import (
	"fmt"
	"sync"
)

// VLESSModifier implements pipeline.Modifier interface for VLESS protocol
type VLESSModifier struct {
	config map[string]interface{}
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
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	// Only process outbound traffic
	if direction == DirectionInbound {
		return data
	}
	
	// TODO: Implement actual VLESS tunneling
	// For now, return data unchanged
	// In production, this would:
	// 1. Connect to VLESS server
	// 2. Perform handshake
	// 3. Send data through WebSocket with XTLS Vision flow
	// 4. Handle response
	
	return data
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
