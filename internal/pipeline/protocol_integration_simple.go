package pipeline

import (
	"fmt"
)

// SimpleProtocolIntegration integrates modern protocols with pipeline system
type SimpleProtocolIntegration struct {
	protocols map[string]ProtocolHandler
}

// NewSimpleProtocolIntegration creates new protocol integration
func NewSimpleProtocolIntegration() *SimpleProtocolIntegration {
	return &SimpleProtocolIntegration{
		protocols: make(map[string]ProtocolHandler),
	}
}

// RegisterProtocol registers a new protocol handler
func (spi *SimpleProtocolIntegration) RegisterProtocol(name string, handler ProtocolHandler) {
	spi.protocols[name] = handler
}

// GetProtocolHandler returns protocol handler by name
func (spi *SimpleProtocolIntegration) GetProtocolHandler(name string) (ProtocolHandler, error) {
	if handler, exists := spi.protocols[name]; exists {
		return handler, nil
	}
	return nil, fmt.Errorf("protocol not found: %s", name)
}

// ListAvailableProtocols returns list of available protocols
func (spi *SimpleProtocolIntegration) ListAvailableProtocols() []string {
	names := make([]string, 0, len(spi.protocols))
	for name := range spi.protocols {
		names = append(names, name)
	}
	return names
}

// CreateProtocolFromConfig creates protocol handler from configuration
func (spi *SimpleProtocolIntegration) CreateProtocolFromConfig(config map[string]interface{}) (ProtocolHandler, error) {
	protocolType, ok := config["type"].(string)
	if !ok {
		return nil, fmt.Errorf("protocol type is required")
	}
	
	return spi.GetProtocolHandler(protocolType)
}

// ValidateProtocolConfig validates protocol configuration
func (spi *SimpleProtocolIntegration) ValidateProtocolConfig(config map[string]interface{}) error {
	protocolType, ok := config["type"].(string)
	if !ok {
		return fmt.Errorf("protocol type is required")
	}
	
	// Check if protocol is supported
	supportedProtocols := spi.ListAvailableProtocols()
	for _, supported := range supportedProtocols {
		if supported == protocolType {
			return nil
		}
	}
	
	return fmt.Errorf("unsupported protocol type: %s", protocolType)
}
