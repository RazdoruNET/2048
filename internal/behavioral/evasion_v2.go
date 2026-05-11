package behavioral

import (
	"context"
	"crypto/rand"
	"math/rand"
	"sync"
	"time"

	"socks5-dpi-proxy/internal/pipeline"
)

// EvasionV2 implements Behavioral Evasion 2.0 techniques
type EvasionV2 struct {
	timingEngine   *TimingEngine
	obfuscation    *TrafficObfuscation
	fingerprinting *FingerprintProtection
	mu             sync.RWMutex
	enabled        bool
	stats          *EvasionStats
}

// EvasionStats tracks evasion effectiveness
type EvasionStats struct {
	TechniquesUsed   map[string]int
	SuccessRates     map[string]float64
	DetectionAvoided int64
	TotalRequests    int64
}

// FingerprintProtection implements DPI fingerprinting protection
type FingerprintProtection struct {
	enabled       bool
	randomizeTLS  bool
	randomizeHTTP bool
	rand          *rand.Rand
}

// NewEvasionV2 creates new Behavioral Evasion 2.0 system
func NewEvasionV2() *EvasionV2 {
	return &EvasionV2{
		timingEngine:   NewTimingEngine(),
		obfuscation:    NewTrafficObfuscation(),
		fingerprinting: NewFingerprintProtection(),
		enabled:        true,
		stats: &EvasionStats{
			TechniquesUsed:   make(map[string]int),
			SuccessRates:     make(map[string]float64),
			DetectionAvoided: 0,
			TotalRequests:    0,
		},
	}
}

// Initialize initializes evasion system
func (e *EvasionV2) Initialize() error {
	if err := e.timingEngine.Initialize(); err != nil {
		return err
	}

	e.fingerprinting.Initialize()
	return nil
}

// ApplyEvasion applies comprehensive behavioral evasion
func (e *EvasionV2) ApplyEvasion(ctx context.Context, data []byte, direction pipeline.Direction, target string) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.enabled {
		return data, nil
	}

	// Apply timing randomization
	timingDelay := e.timingEngine.ApplyTiming(ctx, e.detectConnectionType(target))
	if timingDelay > 0 {
		time.Sleep(timingDelay)
	}

	// Apply traffic obfuscation
	obfuscatedData := e.obfuscation.ApplyObfuscation(data, direction)

	// Apply fingerprinting protection
	finalData := e.fingerprinting.ApplyProtection(obfuscatedData, target)

	// Update statistics
	e.updateStats(target, true)

	return finalData, nil
}

// detectConnectionType determines connection type for timing
func (e *EvasionV2) detectConnectionType(target string) string {
	// Simple heuristics for connection type detection
	if containsAny(target, []string{"youtube.com", "netflix.com", "twitch.tv"}) {
		return "stream"
	}
	if containsAny(target, []string{"api.", "graphql", "rest"}) {
		return "api"
	}
	if containsAny(target, []string{"jpg", "png", "gif", "webp"}) {
		return "image"
	}
	return "default"
}

// updateStats updates evasion statistics
func (e *EvasionV2) updateStats(target string, success bool) {
	e.stats.TotalRequests++

	technique := "behavioral_evasion_v2"
	e.stats.TechniquesUsed[technique]++

	// Update success rate
	current := e.stats.SuccessRates[technique]
	alpha := 0.1
	if success {
		e.stats.SuccessRates[technique] = (1-alpha)*current + alpha*1.0
	} else {
		e.stats.SuccessRates[technique] = (1-alpha)*current + alpha*0.0
	}

	// Update detection avoidance
	if success {
		e.stats.DetectionAvoided++
	}
}

// GetStats returns evasion statistics
func (e *EvasionV2) GetStats() *EvasionStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

// SetEnabled enables/disables evasion
func (e *EvasionV2) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// IsEnabled returns if evasion is enabled
func (e *EvasionV2) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// GetRecommendedTechnique returns best technique for target
func (e *EvasionV2) GetRecommendedTechnique(target string) string {
	// Analyze target and recommend best technique
	connectionType := e.detectConnectionType(target)

	switch connectionType {
	case "stream":
		return "aggressive_timing_with_obfuscation"
	case "api":
		return "conservative_timing_with_padding"
	case "image":
		return "burst_like_with_smart_padding"
	default:
		return "human_like_with_fingerprinting"
	}
}

// containsAny checks if string contains any of substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// NewFingerprintProtection creates new fingerprint protection
func NewFingerprintProtection() *FingerprintProtection {
	return &FingerprintProtection{
		enabled:       true,
		randomizeTLS:  true,
		randomizeHTTP: true,
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Initialize initializes fingerprint protection
func (fp *FingerprintProtection) Initialize() {
	// Initialize fingerprint protection
}

// ApplyProtection applies fingerprinting protection to data
func (fp *FingerprintProtection) ApplyProtection(data []byte, target string) []byte {
	if !fp.enabled {
		return data
	}

	// Apply TLS fingerprint randomization if this looks like TLS traffic
	if fp.randomizeTLS && len(data) >= 5 && data[0] == 0x16 { // TLS Handshake
		return fp.randomizeTLSFingerprint(data)
	}

	// Apply HTTP fingerprint randomization
	if fp.randomizeHTTP && fp.isHTTPTraffic(data) {
		return fp.randomizeHTTPHeaders(data)
	}

	return data
}

// randomizeTLSFingerprint randomizes TLS fingerprint
func (fp *FingerprintProtection) randomizeTLSFingerprint(data []byte) []byte {
	if len(data) < 5 {
		return data
	}

	// Randomize TLS version and cipher suites
	result := make([]byte, len(data))
	copy(result, data)

	// Randomize some bytes in TLS handshake
	for i := 1; i < min(10, len(data)); i++ {
		if fp.rand.Intn(10) == 0 {
			result[i] = byte(fp.rand.Intn(256))
		}
	}

	return result
}

// isHTTPTraffic checks if data looks like HTTP traffic
func (fp *FingerprintProtection) isHTTPTraffic(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// Check for common HTTP methods
	methods := [][]byte{
		[]byte("GET"),
		[]byte("POST"),
		[]byte("PUT"),
		[]byte("DELETE"),
		[]byte("HEAD"),
		[]byte("OPTIONS"),
	}

	for _, method := range methods {
		if len(data) >= len(method) && string(data[:len(method)]) == string(method) {
			return true
		}
	}

	return false
}

// randomizeHTTPHeaders randomizes HTTP headers
func (fp *FingerprintProtection) randomizeHTTPHeaders(data []byte) []byte {
	if len(data) < 10 {
		return data
	}

	// Add random variations to HTTP headers
	result := make([]byte, len(data))
	copy(result, data)

	// Randomize some header characters
	for i := 0; i < min(20, len(data)); i++ {
		if fp.rand.Intn(20) == 0 {
			// Randomize case of some letters
			if result[i] >= 'a' && result[i] <= 'z' {
				result[i] = byte(result[i] - 32) // Convert to uppercase
			} else if result[i] >= 'A' && result[i] <= 'Z' {
				result[i] = byte(result[i] + 32) // Convert to lowercase
			}
		}
	}

	return result
}

// min returns minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
