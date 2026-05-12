package ml

import (
	"fmt"
	"reflect"
	"time"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// ConfigValidator validates ML configuration
type ConfigValidator struct {
	rules map[string]ValidationRule
}

// ValidationRule defines validation rules for a field
type ValidationRule struct {
	Required   bool
	Type       reflect.Kind
	Min        *float64
	Max        *float64
	MinLength  *int
	MaxLength  *int
	Enum       []interface{}
	CustomFunc func(interface{}) error
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	validator := &ConfigValidator{
		rules: make(map[string]ValidationRule),
	}

	// Define validation rules for ML config
	validator.defineMLConfigRules()

	return validator
}

// defineMLConfigRules defines validation rules for ML configuration
func (cv *ConfigValidator) defineMLConfigRules() {
	// Enabled field
	cv.rules["enabled"] = ValidationRule{
		Required: true,
		Type:     reflect.Bool,
	}

	// ModelPath field
	cv.rules["model_path"] = ValidationRule{
		Required:  false,
		Type:      reflect.String,
		MinLength: func() *int { i := 1; return &i }(),
		MaxLength: func() *int { i := 255; return &i }(),
	}

	// UpdateInterval field
	cv.rules["update_interval"] = ValidationRule{
		Required: false,
		Type:     reflect.String,
		CustomFunc: func(value interface{}) error {
			if str, ok := value.(string); ok {
				if str == "" {
					return nil // Empty is allowed
				}
				_, err := time.ParseDuration(str)
				if err != nil {
					return fmt.Errorf("invalid duration format: %v", err)
				}
			}
			return nil
		},
	}

	// MaxRetries field
	cv.rules["max_retries"] = ValidationRule{
		Required: false,
		Type:     reflect.Int,
		Min:      func() *float64 { f := 1.0; return &f }(),
		Max:      func() *float64 { f := 100.0; return &f }(),
	}

	// Timeout field
	cv.rules["timeout"] = ValidationRule{
		Required: false,
		Type:     reflect.String,
		CustomFunc: func(value interface{}) error {
			if str, ok := value.(string); ok {
				if str == "" {
					return nil // Empty is allowed
				}
				duration, err := time.ParseDuration(str)
				if err != nil {
					return fmt.Errorf("invalid duration format: %v", err)
				}
				if duration < 1*time.Second {
					return fmt.Errorf("timeout must be at least 1 second")
				}
				if duration > 10*time.Minute {
					return fmt.Errorf("timeout cannot exceed 10 minutes")
				}
			}
			return nil
		},
	}

	// Custom parameters validation
	cv.rules["custom_parameters"] = ValidationRule{
		Required:   false,
		Type:       reflect.Map,
		CustomFunc: cv.validateCustomParameters,
	}
}

// validateCustomParameters validates custom parameters
func (cv *ConfigValidator) validateCustomParameters(value interface{}) error {
	params, ok := value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("custom_parameters must be a map")
	}

	// Validate learning_rate
	if lr, exists := params["learning_rate"]; exists {
		if err := cv.validateLearningRate(lr); err != nil {
			return fmt.Errorf("learning_rate: %v", err)
		}
	}

	// Validate confidence_threshold
	if ct, exists := params["confidence_threshold"]; exists {
		if err := cv.validateConfidenceThreshold(ct); err != nil {
			return fmt.Errorf("confidence_threshold: %v", err)
		}
	}

	// Validate effectiveness_threshold
	if et, exists := params["effectiveness_threshold"]; exists {
		if err := cv.validateEffectivenessThreshold(et); err != nil {
			return fmt.Errorf("effectiveness_threshold: %v", err)
		}
	}

	return nil
}

// validateLearningRate validates learning rate parameter
func (cv *ConfigValidator) validateLearningRate(value interface{}) error {
	var lr float64

	switch v := value.(type) {
	case float64:
		lr = v
	case int:
		lr = float64(v)
	default:
		return fmt.Errorf("must be a number")
	}

	if lr < 0.0 || lr > 1.0 {
		return fmt.Errorf("must be between 0.0 and 1.0")
	}

	if lr == 0.0 {
		return fmt.Errorf("cannot be zero")
	}

	return nil
}

// validateConfidenceThreshold validates confidence threshold parameter
func (cv *ConfigValidator) validateConfidenceThreshold(value interface{}) error {
	var ct float64

	switch v := value.(type) {
	case float64:
		ct = v
	case int:
		ct = float64(v)
	default:
		return fmt.Errorf("must be a number")
	}

	if ct < 0.0 || ct > 1.0 {
		return fmt.Errorf("must be between 0.0 and 1.0")
	}

	return nil
}

// validateEffectivenessThreshold validates effectiveness threshold parameter
func (cv *ConfigValidator) validateEffectivenessThreshold(value interface{}) error {
	var et float64

	switch v := value.(type) {
	case float64:
		et = v
	case int:
		et = float64(v)
	default:
		return fmt.Errorf("must be a number")
	}

	if et < 0.0 || et > 1.0 {
		return fmt.Errorf("must be between 0.0 and 1.0")
	}

	return nil
}

// ValidateConfig validates an ML configuration
func (cv *ConfigValidator) ValidateConfig(config *MLConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Validate each field
	cv.validateField(config.Enabled, "enabled", config, result)
	cv.validateField(config.ModelPath, "model_path", config, result)
	cv.validateField(config.UpdateInterval, "update_interval", config, result)
	cv.validateField(config.MaxRetries, "max_retries", config, result)
	cv.validateField(config.Timeout, "timeout", config, result)
	cv.validateField(config.CustomParameters, "custom_parameters", config, result)

	// Perform cross-field validation
	cv.validateCrossFields(config, result)

	result.Valid = len(result.Errors) == 0
	return result
}

// validateField validates a single field
func (cv *ConfigValidator) validateField(value interface{}, fieldName string, config *MLConfig, result *ValidationResult) {
	rule, exists := cv.rules[fieldName]
	if !exists {
		return // No rule defined for this field
	}

	// Check if required
	if rule.Required && value == nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fieldName,
			Message: "field is required",
		})
		result.Valid = false
		return
	}

	if value == nil {
		return // Optional field with nil value is valid
	}

	// Check type
	if rule.Type != reflect.Invalid && reflect.TypeOf(value).Kind() != rule.Type {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("must be of type %s", rule.Type),
			Value:   value,
		})
		result.Valid = false
		return
	}

	// Check min/max for numeric values
	if rule.Min != nil || rule.Max != nil {
		if err := cv.validateNumericRange(value, rule, fieldName); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldName,
				Message: err.Error(),
				Value:   value,
			})
			result.Valid = false
			return
		}
	}

	// Check min/max length for strings
	if rule.MinLength != nil || rule.MaxLength != nil {
		if err := cv.validateStringLength(value, rule, fieldName); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldName,
				Message: err.Error(),
				Value:   value,
			})
			result.Valid = false
			return
		}
	}

	// Check enum values
	if len(rule.Enum) > 0 {
		if err := cv.validateEnum(value, rule, fieldName); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldName,
				Message: err.Error(),
				Value:   value,
			})
			result.Valid = false
			return
		}
	}

	// Run custom validation
	if rule.CustomFunc != nil {
		if err := rule.CustomFunc(value); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldName,
				Message: err.Error(),
				Value:   value,
			})
			result.Valid = false
		}
	}
}

// validateNumericRange validates numeric range
func (cv *ConfigValidator) validateNumericRange(value interface{}, rule ValidationRule, fieldName string) error {
	var num float64

	switch v := value.(type) {
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case float64:
		num = v
	default:
		return fmt.Errorf("numeric validation failed for type %T", value)
	}

	if rule.Min != nil && num < *rule.Min {
		return fmt.Errorf("must be at least %f", *rule.Min)
	}

	if rule.Max != nil && num > *rule.Max {
		return fmt.Errorf("must be at most %f", *rule.Max)
	}

	return nil
}

// validateStringLength validates string length
func (cv *ConfigValidator) validateStringLength(value interface{}, rule ValidationRule, fieldName string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("string length validation failed for type %T", value)
	}

	length := len(str)

	if rule.MinLength != nil && length < *rule.MinLength {
		return fmt.Errorf("must be at least %d characters", *rule.MinLength)
	}

	if rule.MaxLength != nil && length > *rule.MaxLength {
		return fmt.Errorf("must be at most %d characters", *rule.MaxLength)
	}

	return nil
}

// validateEnum validates enum values
func (cv *ConfigValidator) validateEnum(value interface{}, rule ValidationRule, fieldName string) error {
	for _, enumValue := range rule.Enum {
		if reflect.DeepEqual(value, enumValue) {
			return nil
		}
	}

	return fmt.Errorf("must be one of: %v", rule.Enum)
}

// validateCrossFields performs cross-field validation
func (cv *ConfigValidator) validateCrossFields(config *MLConfig, result *ValidationResult) {
	// If ML is disabled, other fields should be minimal
	if !config.Enabled {
		if config.UpdateInterval < time.Minute {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "update_interval",
				Message: "should be minimal when ML is disabled",
				Value:   config.UpdateInterval,
			})
			result.Valid = false
		}
	}

	// Validate timeout vs update interval
	if config.Timeout > 0 && config.UpdateInterval > 0 {
		if config.Timeout >= config.UpdateInterval {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "timeout",
				Message: "timeout should be less than update interval",
				Value:   config.Timeout,
			})
			result.Valid = false
		}
	}

	// Validate custom parameters consistency
	if config.CustomParameters != nil {
		cv.validateCustomParameterConsistency(config, result)
	}
}

// validateCustomParameterConsistency validates consistency of custom parameters
func (cv *ConfigValidator) validateCustomParameterConsistency(config *MLConfig, result *ValidationResult) {
	params := config.CustomParameters

	// Check learning rate vs confidence threshold
	lr, hasLR := params["learning_rate"].(float64)
	ct, hasCT := params["confidence_threshold"].(float64)

	if hasLR && hasCT && lr >= ct {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "custom_parameters",
			Message: "learning_rate should be less than confidence_threshold",
			Value:   map[string]float64{"learning_rate": lr, "confidence_threshold": ct},
		})
		result.Valid = false
	}

	// Check confidence threshold vs effectiveness threshold
	et, hasET := params["effectiveness_threshold"].(float64)

	if hasCT && hasET && ct <= et {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "custom_parameters",
			Message: "confidence_threshold should be greater than effectiveness_threshold",
			Value:   map[string]float64{"confidence_threshold": ct, "effectiveness_threshold": et},
		})
		result.Valid = false
	}
}

// MLConfigRequest represents configuration from API request
type MLConfigRequest struct {
	LearningRate           float64 `json:"learning_rate"`
	ConfidenceThreshold    float64 `json:"confidence_threshold"`
	ModelUpdateInterval    string  `json:"model_update_interval"`
	EffectivenessThreshold float64 `json:"effectiveness_threshold"`
}

// ValidateConfigRequest validates configuration from API request
func (cv *ConfigValidator) ValidateConfigRequest(req MLConfigRequest) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Validate learning rate
	if req.LearningRate < 0.0 || req.LearningRate > 1.0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "learning_rate",
			Message: "must be between 0.0 and 1.0",
			Value:   req.LearningRate,
		})
		result.Valid = false
	}

	// Validate confidence threshold
	if req.ConfidenceThreshold < 0.0 || req.ConfidenceThreshold > 1.0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "confidence_threshold",
			Message: "must be between 0.0 and 1.0",
			Value:   req.ConfidenceThreshold,
		})
		result.Valid = false
	}

	// Validate effectiveness threshold
	if req.EffectivenessThreshold < 0.0 || req.EffectivenessThreshold > 1.0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "effectiveness_threshold",
			Message: "must be between 0.0 and 1.0",
			Value:   req.EffectivenessThreshold,
		})
		result.Valid = false
	}

	// Validate model update interval
	if req.ModelUpdateInterval != "" {
		if _, err := time.ParseDuration(req.ModelUpdateInterval); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "model_update_interval",
				Message: "invalid duration format",
				Value:   req.ModelUpdateInterval,
			})
			result.Valid = false
		}
	}

	// Cross-field validation
	if req.LearningRate >= req.ConfidenceThreshold {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "learning_rate",
			Message: "must be less than confidence_threshold",
			Value:   req.LearningRate,
		})
		result.Valid = false
	}

	if req.ConfidenceThreshold <= req.EffectivenessThreshold {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "confidence_threshold",
			Message: "must be greater than effectiveness_threshold",
			Value:   req.ConfidenceThreshold,
		})
		result.Valid = false
	}

	return result
}
