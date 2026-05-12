package behavioral

import (
	"fmt"
	"sync"
	"time"

	"socks5-dpi-proxy/internal/pipeline"
)

// BehavioralEvasionModifier implements unified behavioral evasion as pipeline.Modifier
type BehavioralEvasionModifier struct {
	timingEngine      *TimingEngine
	trafficObfuscator *TrafficObfuscator
	fingerprinting    *FingerprintingProtection
	realtimeAdapter   *RealtimeAdapter
	config            *BehavioralConfig
	enabled           bool
	mu                sync.RWMutex
}

// BehavioralConfig represents behavioral evasion configuration
type BehavioralConfig struct {
	// Timing configuration
	TimingEnabled     bool     `yaml:"timing_enabled"`
	TimingPatterns    []string `yaml:"timing_patterns"`
	HumanizationLevel int      `yaml:"humanization_level"`

	// Traffic obfuscation configuration
	ObfuscationEnabled bool `yaml:"obfuscation_enabled"`
	ObfuscationLevel   int  `yaml:"obfuscation_level"`
	MLOptimization     bool `yaml:"ml_optimization"`

	// Fingerprinting protection configuration
	FingerprintingEnabled bool `yaml:"fingerprinting_enabled"`
	ProtectionLevel       int  `yaml:"protection_level"`
	BrowserMasking        bool `yaml:"browser_masking"`

	// Real-time adaptation configuration
	AdaptationEnabled bool    `yaml:"adaptation_enabled"`
	AdaptationRate    float64 `yaml:"adaptation_rate"`
	LearningEnabled   bool    `yaml:"learning_enabled"`

	// Global settings
	Enabled   bool `yaml:"enabled"`
	DebugMode bool `yaml:"debug_mode"`
}

// NewBehavioralEvasionModifier creates new behavioral evasion modifier
func NewBehavioralEvasionModifier() *BehavioralEvasionModifier {
	config := &BehavioralConfig{
		TimingEnabled:         true,
		TimingPatterns:        []string{"human", "mobile", "bot"},
		HumanizationLevel:     3,
		ObfuscationEnabled:    true,
		ObfuscationLevel:      3,
		MLOptimization:        true,
		FingerprintingEnabled: true,
		ProtectionLevel:       3,
		BrowserMasking:        true,
		AdaptationEnabled:     true,
		AdaptationRate:        0.1,
		LearningEnabled:       true,
		Enabled:               true,
		DebugMode:             false,
	}

	return &BehavioralEvasionModifier{
		timingEngine:      NewTimingEngine(),
		trafficObfuscator: NewTrafficObfuscator(),
		fingerprinting:    NewFingerprintingProtection(),
		realtimeAdapter:   NewRealtimeAdapter(),
		config:            config,
		enabled:           true,
	}
}

// Name returns the modifier name
func (bem *BehavioralEvasionModifier) Name() string {
	return "behavioral_evasion"
}

// Configure configures the behavioral evasion modifier
func (bem *BehavioralEvasionModifier) Configure(config map[string]interface{}) error {
	bem.mu.Lock()
	defer bem.mu.Unlock()

	// Parse configuration
	if err := bem.parseConfig(config); err != nil {
		return fmt.Errorf("failed to parse behavioral evasion config: %w", err)
	}

	// Update component configurations
	bem.updateComponentConfigs()

	return nil
}

// Process processes data through behavioral evasion
func (bem *BehavioralEvasionModifier) Process(data []byte, direction pipeline.Direction) []byte {
	bem.mu.RLock()
	defer bem.mu.RUnlock()

	if !bem.enabled || !bem.config.Enabled {
		return data
	}

	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		bem.trackPerformance(true, latency, direction)
	}()

	// Apply different processing based on direction
	switch direction {
	case pipeline.DirectionOutbound:
		return bem.processOutbound(data)
	case pipeline.DirectionInbound:
		return bem.processInbound(data)
	default:
		return data
	}
}

// processOutbound processes outbound data
func (bem *BehavioralEvasionModifier) processOutbound(data []byte) []byte {
	processedData := data

	// Apply timing patterns
	if bem.config.TimingEnabled {
		dpiLevel := bem.calculateDPILevel(processedData, "outbound")
		delay := bem.timingEngine.GetHumanizedDelay("normal", dpiLevel)
		if delay > 0 {
			time.Sleep(delay)
		}
		bem.timingEngine.UpdateContext(dpiLevel, len(processedData))
	}

	// Apply traffic obfuscation
	if bem.config.ObfuscationEnabled {
		dpiLevel := bem.calculateDPILevel(processedData, "outbound")
		obfuscatedData, err := bem.trafficObfuscator.ObfuscateData(processedData, dpiLevel)
		if err == nil {
			processedData = obfuscatedData
		}
	}

	// Apply fingerprinting protection (simulated for data processing)
	if bem.config.FingerprintingEnabled {
		requestType := bem.getRequestType(processedData)
		delay := bem.fingerprinting.MaskBrowserBehavior(requestType)
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	// Apply real-time adaptation
	if bem.config.AdaptationEnabled {
		bem.applyRealtimeAdaptation(processedData, "outbound")
	}

	return processedData
}

// processInbound processes inbound data
func (bem *BehavioralEvasionModifier) processInbound(data []byte) []byte {
	processedData := data

	// For inbound data, we mainly apply timing and adaptation
	if bem.config.TimingEnabled {
		dpiLevel := bem.calculateDPILevel(processedData, "inbound")
		delay := bem.timingEngine.GetHumanizedDelay("normal", dpiLevel)
		if delay > 0 {
			// Smaller delay for inbound to reduce latency
			delay = time.Duration(float64(delay) * 0.5)
			time.Sleep(delay)
		}
	}

	// Apply real-time adaptation
	if bem.config.AdaptationEnabled {
		bem.applyRealtimeAdaptation(processedData, "inbound")
	}

	return processedData
}

// calculateDPILevel calculates DPI detection level based on data characteristics
func (bem *BehavioralEvasionModifier) calculateDPILevel(data []byte, direction string) int {
	dataSize := len(data)

	// Complex DPI level calculation based on multiple factors
	dpiScore := 0

	// Size-based scoring
	if dataSize > 1024*1024 { // > 1MB
		dpiScore += 3
	} else if dataSize > 100*1024 { // > 100KB
		dpiScore += 2
	} else if dataSize > 10*1024 { // > 10KB
		dpiScore += 1
	}

	// Direction-based scoring
	if direction == "outbound" {
		dpiScore += 1
	}

	// Content-based scoring (simple pattern detection)
	if dataSize > 0 {
		// Check for common patterns that might trigger DPI
		if bem.containsSuspiciousPatterns(data) {
			dpiScore += 2
		}
	}

	// Clamp to 1-5 range
	if dpiScore < 1 {
		return 1
	} else if dpiScore > 5 {
		return 5
	}
	return dpiScore
}

// containsSuspiciousPatterns checks for suspicious patterns in data
func (bem *BehavioralEvasionModifier) containsSuspiciousPatterns(data []byte) bool {
	if len(data) < 10 {
		return false
	}

	// Simple pattern detection for demonstration
	suspiciousPatterns := [][]byte{
		[]byte("GET"),
		[]byte("POST"),
		[]byte("HTTP"),
		[]byte("https"),
		[]byte("torrent"),
	}

	for _, pattern := range suspiciousPatterns {
		if bem.containsPattern(data, pattern) {
			return true
		}
	}

	return false
}

// containsPattern checks if data contains a specific pattern
func (bem *BehavioralEvasionModifier) containsPattern(data []byte, pattern []byte) bool {
	if len(pattern) > len(data) {
		return false
	}

	for i := 0; i <= len(data)-len(pattern); i++ {
		match := true
		for j := 0; j < len(pattern); j++ {
			if data[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

// getRequestType determines request type from data
func (bem *BehavioralEvasionModifier) getRequestType(data []byte) string {
	dataSize := len(data)

	if dataSize < 10*1024 { // < 10KB
		return "mobile"
	} else if dataSize > 100*1024 { // > 100KB
		return "bot"
	}

	return "normal"
}

// trackPerformance tracks performance metrics
func (bem *BehavioralEvasionModifier) trackPerformance(success bool, latency time.Duration, direction pipeline.Direction) {
	dpiLevel := 2 // Default DPI level
	strategy := "behavioral_evasion"

	bem.realtimeAdapter.TrackPerformance(success, latency, dpiLevel, strategy)
}

// applyRealtimeAdaptation applies real-time adaptation
func (bem *BehavioralEvasionModifier) applyRealtimeAdaptation(data []byte, direction string) {
	// Get current strategy
	stats := bem.realtimeAdapter.GetAdaptationStats()
	currentStrategy, _ := stats["current_strategy"].(string)

	// Track performance for adaptation
	dpiLevel := bem.calculateDPILevel(data, direction)
	bem.realtimeAdapter.TrackPerformance(true, 0, dpiLevel, currentStrategy)

	// Update configuration based on adaptation
	bem.updateConfigFromAdaptation(stats)
}

// parseConfig parses configuration from map
func (bem *BehavioralEvasionModifier) parseConfig(config map[string]interface{}) error {
	// Parse timing configuration
	if timingEnabled, ok := config["timing_enabled"].(bool); ok {
		bem.config.TimingEnabled = timingEnabled
	}

	if timingPatterns, ok := config["timing_patterns"].([]string); ok {
		bem.config.TimingPatterns = timingPatterns
	}

	if humanizationLevel, ok := config["humanization_level"].(int); ok {
		bem.config.HumanizationLevel = humanizationLevel
	}

	// Parse obfuscation configuration
	if obfuscationEnabled, ok := config["obfuscation_enabled"].(bool); ok {
		bem.config.ObfuscationEnabled = obfuscationEnabled
	}

	if obfuscationLevel, ok := config["obfuscation_level"].(int); ok {
		bem.config.ObfuscationLevel = obfuscationLevel
	}

	if mlOptimization, ok := config["ml_optimization"].(bool); ok {
		bem.config.MLOptimization = mlOptimization
	}

	// Parse fingerprinting configuration
	if fingerprintingEnabled, ok := config["fingerprinting_enabled"].(bool); ok {
		bem.config.FingerprintingEnabled = fingerprintingEnabled
	}

	if protectionLevel, ok := config["protection_level"].(int); ok {
		bem.config.ProtectionLevel = protectionLevel
	}

	if browserMasking, ok := config["browser_masking"].(bool); ok {
		bem.config.BrowserMasking = browserMasking
	}

	// Parse adaptation configuration
	if adaptationEnabled, ok := config["adaptation_enabled"].(bool); ok {
		bem.config.AdaptationEnabled = adaptationEnabled
	}

	if adaptationRate, ok := config["adaptation_rate"].(float64); ok {
		bem.config.AdaptationRate = adaptationRate
	}

	if learningEnabled, ok := config["learning_enabled"].(bool); ok {
		bem.config.LearningEnabled = learningEnabled
	}

	// Parse global settings
	if enabled, ok := config["enabled"].(bool); ok {
		bem.config.Enabled = enabled
	}

	if debugMode, ok := config["debug_mode"].(bool); ok {
		bem.config.DebugMode = debugMode
	}

	return nil
}

// updateComponentConfigs updates individual component configurations
func (bem *BehavioralEvasionModifier) updateComponentConfigs() {
	// Update timing engine
	bem.timingEngine.SetEnabled(bem.config.TimingEnabled)

	// Update traffic obfuscator
	bem.trafficObfuscator.SetEnabled(bem.config.ObfuscationEnabled)

	// Update fingerprinting protection
	bem.fingerprinting.SetEnabled(bem.config.FingerprintingEnabled)
	bem.fingerprinting.SetProtectionLevel(bem.config.ProtectionLevel)

	// Update real-time adapter
	bem.realtimeAdapter.SetEnabled(bem.config.AdaptationEnabled)
}

// updateConfigFromAdaptation updates configuration based on adaptation stats
func (bem *BehavioralEvasionModifier) updateConfigFromAdaptation(stats map[string]interface{}) {
	successRate, _ := stats["success_rate"].(float64)

	// Adjust configuration based on performance
	if successRate < 0.6 {
		// Increase obfuscation level if success rate is low
		if bem.config.ObfuscationLevel < 5 {
			bem.config.ObfuscationLevel++
		}
		if bem.config.ProtectionLevel < 5 {
			bem.config.ProtectionLevel++
		}
	} else if successRate > 0.9 {
		// Decrease obfuscation level if success rate is very high (optimization)
		if bem.config.ObfuscationLevel > 1 {
			bem.config.ObfuscationLevel--
		}
	}
}

// GetConfig returns current configuration
func (bem *BehavioralEvasionModifier) GetConfig() *BehavioralConfig {
	bem.mu.RLock()
	defer bem.mu.RUnlock()

	return bem.config
}

// GetStats returns comprehensive statistics
func (bem *BehavioralEvasionModifier) GetStats() map[string]interface{} {
	bem.mu.RLock()
	defer bem.mu.RUnlock()

	stats := make(map[string]interface{})

	// Basic stats
	stats["enabled"] = bem.enabled
	stats["config"] = bem.config

	// Component stats
	stats["timing_stats"] = bem.timingEngine.GetStats()
	stats["obfuscation_stats"] = bem.trafficObfuscator.GetStats()
	stats["fingerprinting_stats"] = map[string]interface{}{
		"enabled":          bem.fingerprinting.IsEnabled(),
		"protection_level": bem.fingerprinting.GetProtectionLevel(),
	}
	stats["adaptation_stats"] = bem.realtimeAdapter.GetAdaptationStats()

	return stats
}

// Enable enables behavioral evasion
func (bem *BehavioralEvasionModifier) Enable() {
	bem.mu.Lock()
	defer bem.mu.Unlock()
	bem.enabled = true
	bem.config.Enabled = true
	bem.updateComponentConfigs()
}

// Disable disables behavioral evasion
func (bem *BehavioralEvasionModifier) Disable() {
	bem.mu.Lock()
	defer bem.mu.Unlock()
	bem.enabled = false
	bem.config.Enabled = false
	bem.updateComponentConfigs()
}

// IsEnabled returns whether behavioral evasion is enabled
func (bem *BehavioralEvasionModifier) IsEnabled() bool {
	bem.mu.RLock()
	defer bem.mu.RUnlock()
	return bem.enabled
}
