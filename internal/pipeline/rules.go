package pipeline

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

func (e *Engine) LoadConfig(configPath string) error {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		Rules []struct {
			Domain  string                 `yaml:"domain"`
			IPRange string                 `yaml:"ip_range"`
			Port    uint16                 `yaml:"port"`
			Config  map[string]interface{} `yaml:"pipeline"`
		} `yaml:"rules"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules = make([]*Rule, 0, len(config.Rules))

	for _, ruleConfig := range config.Rules {
		rule := &Rule{
			Domain:  ruleConfig.Domain,
			IPRange: ruleConfig.IPRange,
			Port:    ruleConfig.Port,
			Config:  ruleConfig.Config,
		}

		// Initialize modifiers for this rule
		if err := e.initializeRuleModifiers(rule); err != nil {
			log.Printf("Failed to initialize modifiers for rule %s: %v", rule.Domain, err)
			continue
		}

		e.rules = append(e.rules, rule)
	}

	log.Printf("Loaded %d pipeline rules", len(e.rules))
	return nil
}

func (e *Engine) initializeRuleModifiers(rule *Rule) error {
	rule.modifiers = make([]Modifier, 0)

	// Parse pipeline configuration
	for modifierName, config := range rule.Config {
		// Create a new instance of the modifier
		newModifier := e.createModifierInstance(modifierName)
		if newModifier == nil {
			log.Printf("Failed to create modifier instance: %s", modifierName)
			continue
		}

		// Configure the modifier
		if configMap, ok := config.(map[string]interface{}); ok {
			if err := newModifier.Configure(configMap); err != nil {
				log.Printf("Failed to configure modifier %s: %v", modifierName, err)
				continue
			}
		}

		rule.modifiers = append(rule.modifiers, newModifier)
	}

	return nil
}

// initializeRuleModifiersV2 creates ModifierV2 instances for enhanced pipeline
func (e *Engine) initializeRuleModifiersV2(rule *Rule) error {
	rule.modifiers = make([]Modifier, 0)

	// Parse pipeline configuration
	for modifierName, config := range rule.Config {
		// Create a new instance of the modifier
		newModifier := e.createModifierInstanceV2(modifierName)
		if newModifier == nil {
			log.Printf("Failed to create modifier V2 instance: %s", modifierName)
			continue
		}

		// Configure the modifier
		if configMap, ok := config.(map[string]interface{}); ok {
			if err := newModifier.Configure(configMap); err != nil {
				log.Printf("Failed to configure modifier V2 %s: %v", modifierName, err)
				continue
			}
		}

		rule.modifiers = append(rule.modifiers, newModifier)
	}

	return nil
}

func (e *Engine) createModifierInstance(name string) Modifier {
	switch name {
	case "fragmentation":
		return &FragmentationModifier{}
	case "adaptive_fragmentation":
		return &AdaptiveFragmentationModifier{}
	case "headers":
		return &HeadersModifier{}
	case "encryption":
		return &EncryptionModifier{}
	case "protocol_mask":
		return &ProtocolMaskModifier{}
	case "behavioral_evasion":
		return &BehavioralEvasionModifier{}
	case "vless_client":
		return NewVLESSModifier()
	case "hysteria2_client":
		return NewHysteria2Modifier()
	case "tuic_client":
		return NewTUICModifier()
	default:
		return nil
	}
}

// createModifierInstanceV2 creates ModifierV2 instances with enhanced capabilities
func (e *Engine) createModifierInstanceV2(name string) ModifierV2 {
	switch name {
	case "fragmentation":
		return &FragmentationModifier{}
	case "adaptive_fragmentation":
		return &AdaptiveFragmentationModifier{}
	case "headers":
		// Wrap legacy modifier for compatibility
		return NewLegacyModifierWrapper(&HeadersModifier{})
	case "encryption":
		// Wrap legacy modifier for compatibility
		return NewLegacyModifierWrapper(&EncryptionModifier{})
	case "protocol_mask":
		// Wrap legacy modifier for compatibility
		return NewLegacyModifierWrapper(&ProtocolMaskModifier{})
	case "behavioral_evasion":
		// Wrap legacy modifier for compatibility
		return NewLegacyModifierWrapper(&BehavioralEvasionModifier{})
	case "vless_client":
		// Wrap legacy modifier for compatibility
		return NewLegacyModifierWrapper(NewVLESSModifier())
	case "hysteria2_client":
		// Wrap legacy modifier for compatibility
		return NewLegacyModifierWrapper(NewHysteria2Modifier())
	case "tuic_client":
		// Wrap legacy modifier for compatibility
		return NewLegacyModifierWrapper(NewTUICModifier())
	default:
		return nil
	}
}

// CreatePipelineV2 creates an enhanced pipeline with ModifierV2 support
func (e *Engine) CreatePipelineV2(target string, port uint16) *PipelineV2 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rule := e.findRule(target, port)
	if rule == nil {
		return nil
	}

	// Initialize V2 modifiers
	if err := e.initializeRuleModifiersV2(rule); err != nil {
		log.Printf("Failed to initialize V2 modifiers for rule %s: %v", rule.Domain, err)
		return nil
	}

	// Convert to ModifierV2 slice
	modifiersV2 := make([]ModifierV2, 0, len(rule.modifiers))
	for _, modifier := range rule.modifiers {
		if modV2, ok := modifier.(ModifierV2); ok {
			modifiersV2 = append(modifiersV2, modV2)
		} else {
			// Wrap legacy modifier
			modifiersV2 = append(modifiersV2, NewLegacyModifierWrapper(modifier))
		}
	}

	return &PipelineV2{
		modifiers: modifiersV2,
		target:    target,
		port:      port,
		config:    DefaultConnectionConfig(),
	}
}

func (e *Engine) SaveConfig(configPath string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	config := struct {
		Rules []map[string]interface{} `yaml:"rules"`
	}{
		Rules: make([]map[string]interface{}, 0, len(e.rules)),
	}

	for _, rule := range e.rules {
		ruleMap := make(map[string]interface{})

		if rule.Domain != "" {
			ruleMap["domain"] = rule.Domain
		}
		if rule.IPRange != "" {
			ruleMap["ip_range"] = rule.IPRange
		}
		if rule.Port != 0 {
			ruleMap["port"] = rule.Port
		}
		if rule.Config != nil {
			ruleMap["pipeline"] = rule.Config
		}

		config.Rules = append(config.Rules, ruleMap)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
