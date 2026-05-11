package pipeline

import (
	"context"
	"net"
)

// ProtocolHandler defines interface for protocol implementations
type ProtocolHandler interface {
	// Name returns protocol name
	Name() string
	
	// Version returns protocol version
	Version() string
	
	// Initialize protocol with configuration
	Initialize(config map[string]interface{}) error
	
	// Create connection to target
	Connect(ctx context.Context, target string, port int) (net.Conn, error)
	
	// Handle connection data
	Handle(conn net.Conn) error
	
	// Close protocol resources
	Close() error
	
	// Get protocol capabilities
	Capabilities() ProtocolCapabilities
	
	// Health check for protocol
	HealthCheck() error
}

// ProtocolCapabilities defines what protocol can do
type ProtocolCapabilities struct {
	SupportsMultiplexing bool
	SupportsEncryption     bool
	SupportsObfuscation   bool
	MaxConnections       int
	ProtocolType         string // "proxy", "tunnel", "direct"
}

// ProtocolRegistry manages protocol handlers
type ProtocolRegistry struct {
	handlers map[string]ProtocolHandler
	factory  map[string]func() ProtocolHandler
}

// NewProtocolRegistry creates new protocol registry
func NewProtocolRegistry() *ProtocolRegistry {
	return &ProtocolRegistry{
		handlers: make(map[string]ProtocolHandler),
		factory:  make(map[string]func() ProtocolHandler),
	}
}

// RegisterProtocol registers a new protocol handler
func (pr *ProtocolRegistry) RegisterProtocol(name string, factory func() ProtocolHandler) {
	pr.factory[name] = factory
}

// GetProtocol returns protocol handler by name
func (pr *ProtocolRegistry) GetProtocol(name string) (ProtocolHandler, error) {
	if factory, exists := pr.factory[name]; exists {
		if handler, handlerExists := pr.handlers[name]; handlerExists {
			return handler, nil
		}
		handler := factory()
		pr.handlers[name] = handler
		return handler, nil
	}
	return nil, ErrProtocolNotFound
}

// ListProtocols returns all registered protocol names
func (pr *ProtocolRegistry) ListProtocols() []string {
	names := make([]string, 0, len(pr.factory))
	for name := range pr.factory {
		names = append(names, name)
	}
	return names
}

// Protocol errors
var (
	ErrProtocolNotFound = &ProtocolError{
		Code:    "PROTOCOL_NOT_FOUND",
		Message: "Protocol handler not found",
	}
	ErrProtocolInitFailed = &ProtocolError{
		Code:    "PROTOCOL_INIT_FAILED", 
		Message: "Protocol initialization failed",
	}
)

// ProtocolError represents protocol-specific errors
type ProtocolError struct {
	Code    string
	Message string
}

func (e *ProtocolError) Error() string {
	return e.Message
}
