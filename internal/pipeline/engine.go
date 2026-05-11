package pipeline

import (
	"net"
	"sync"
)

type Modifier interface {
	Name() string
	Process(data []byte, direction Direction) []byte
	Configure(config map[string]interface{}) error
}

type Direction int

const (
	DirectionInbound Direction = iota
	DirectionOutbound
)

type SOCKS5Request struct {
	Version  byte
	Command  byte
	Reserved byte
	AddrType byte
	DstAddr  string
	DstPort  uint16
}

type Pipeline struct {
	modifiers []Modifier
	target    string
	port      uint16
}

type Engine struct {
	rules     []*Rule
	modifiers map[string]Modifier
	mu        sync.RWMutex
}

type Rule struct {
	Domain    string                 `yaml:"domain"`
	IPRange   string                 `yaml:"ip_range"`
	Port      uint16                 `yaml:"port"`
	Config    map[string]interface{} `yaml:"pipeline"`
	modifiers []Modifier
}

func NewEngine() *Engine {
	return &Engine{
		rules:     make([]*Rule, 0),
		modifiers: make(map[string]Modifier),
	}
}

func (e *Engine) RegisterModifier(modifier Modifier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.modifiers[modifier.Name()] = modifier
}

func (e *Engine) CreatePipeline(target string, port uint16) *Pipeline {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rule := e.findRule(target, port)
	if rule == nil {
		return nil
	}

	return &Pipeline{
		modifiers: rule.modifiers,
		target:    target,
		port:      port,
	}
}

// ShouldUsePipeline determines if DPI bypass mechanisms should be applied
// This method will be used by the proxy server to optimize routing
func (e *Engine) ShouldUsePipeline(target string, port uint16) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rule := e.findRule(target, port)
	if rule == nil {
		return false // No rule means no bypass needed
	}

	// If rule has no modifiers, no need to use pipeline
	return len(rule.modifiers) > 0
}

// GetPipelineModifiers returns the list of modifiers for a target
func (e *Engine) GetPipelineModifiers(target string, port uint16) []Modifier {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rule := e.findRule(target, port)
	if rule == nil {
		return nil
	}

	return rule.modifiers
}

func (e *Engine) findRule(target string, port uint16) *Rule {
	// Check exact domain match first
	for _, rule := range e.rules {
		if rule.Domain != "" && matchesDomain(target, rule.Domain) {
			if rule.Port == 0 || rule.Port == port {
				return rule
			}
		}
	}

	// Check IP range match
	ip := net.ParseIP(target)
	if ip != nil {
		for _, rule := range e.rules {
			if rule.IPRange != "" && matchesIPRange(ip, rule.IPRange) {
				if rule.Port == 0 || rule.Port == port {
					return rule
				}
			}
		}
	}

	// Return default rule if exists
	for _, rule := range e.rules {
		if rule.Domain == "" && rule.IPRange == "" {
			if rule.Port == 0 || rule.Port == port {
				return rule
			}
		}
	}

	return nil
}

func (p *Pipeline) ProcessOutbound(data []byte) []byte {
	return p.processData(data, DirectionOutbound)
}

func (p *Pipeline) ProcessInbound(data []byte) []byte {
	return p.processData(data, DirectionInbound)
}

func (p *Pipeline) processData(data []byte, direction Direction) []byte {
	result := data
	for _, modifier := range p.modifiers {
		result = modifier.Process(result, direction)
	}
	return result
}

func (p *Pipeline) PreConnection(req *SOCKS5Request) *SOCKS5Request {
	// Apply pre-connection modifications if needed
	// This can be used for connection-level DPI bypass techniques
	return req
}

func matchesDomain(target, pattern string) bool {
	// Simple domain matching with wildcard support
	if pattern == "*" {
		return true
	}

	if pattern[0] == '*' && pattern[1] == '.' {
		// Wildcard subdomain matching
		suffix := pattern[2:]
		return len(target) > len(suffix) && target[len(target)-len(suffix):] == suffix
	}

	return target == pattern
}

func matchesIPRange(ip net.IP, cidr string) bool {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ipNet.Contains(ip)
}
