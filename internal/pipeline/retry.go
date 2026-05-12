package pipeline

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

// RetryConfig defines configuration for pipeline retry logic
type RetryConfig struct {
	MaxAttempts      int           `yaml:"max_attempts"`
	Timeout          time.Duration `yaml:"timeout"`
	RetryDelay       time.Duration `yaml:"retry_delay"`
	CircuitBreaker   bool          `yaml:"circuit_breaker"`
	FailureThreshold int           `yaml:"failure_threshold"`
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:      3,
		Timeout:          10 * time.Second,
		RetryDelay:       100 * time.Millisecond,
		CircuitBreaker:   true,
		FailureThreshold: 5,
	}
}

// RetryResult represents the result of a retry attempt
type RetryResult struct {
	Success    bool
	Technique  string
	Attempts   int
	Duration   time.Duration
	Error      error
	Connection net.Conn
}

// RetryStrategy defines different retry strategies
type RetryStrategy int

const (
	StrategySequential RetryStrategy = iota
	StrategyParallel
	StrategyAdaptive
)

// PipelineEngine interface for dependency injection
type PipelineEngine interface {
	CreatePipeline(target string, port uint16) *Pipeline
}

// PipelineRetry implements retry logic for DPI bypass
type PipelineRetry struct {
	config       *RetryConfig
	engine       PipelineEngine
	strategy     RetryStrategy
	attemptCount map[string]int
	failureCount map[string]int
}

// NewPipelineRetry creates a new pipeline retry instance
func NewPipelineRetry(engine PipelineEngine, config *RetryConfig) *PipelineRetry {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return &PipelineRetry{
		config:       config,
		engine:       engine,
		strategy:     StrategyAdaptive,
		attemptCount: make(map[string]int),
		failureCount: make(map[string]int),
	}
}

// RetryWithPipeline attempts to connect using pipeline techniques when direct connection fails
func (pr *PipelineRetry) RetryWithPipeline(ctx context.Context, req *SOCKS5Request) (*RetryResult, error) {
	start := time.Now()
	result := &RetryResult{
		Success:  false,
		Attempts: 0,
		Duration: 0,
	}

	// Get available pipeline for this target
	pipe := pr.engine.CreatePipeline(req.DstAddr, req.DstPort)
	if pipe == nil || len(pipe.modifiers) == 0 {
		log.Printf("No pipeline available for %s:%d", req.DstAddr, req.DstPort)
		result.Error = fmt.Errorf("no pipeline available")
		return result, fmt.Errorf("no pipeline available for target")
	}

	log.Printf("Starting pipeline retry for %s:%d with %d modifiers",
		req.DstAddr, req.DstPort, len(pipe.modifiers))

	// Try different techniques based on strategy
	switch pr.strategy {
	case StrategySequential:
		result = pr.retrySequential(ctx, req, pipe)
	case StrategyParallel:
		result = pr.retryParallel(ctx, req, pipe)
	case StrategyAdaptive:
		result = pr.retryAdaptive(ctx, req, pipe)
	default:
		result = pr.retrySequential(ctx, req, pipe)
	}

	result.Duration = time.Since(start)

	// Log the result
	if result.Success {
		log.Printf("Pipeline retry successful for %s:%d using %s after %d attempts (%v)",
			req.DstAddr, req.DstPort, result.Technique, result.Attempts, result.Duration)
		return result, nil
	} else {
		log.Printf("Pipeline retry failed for %s:%d after %d attempts: %v",
			req.DstAddr, req.DstPort, result.Attempts, result.Error)
		// Update failure count for circuit breaker
		pr.updateFailureCount(req.DstAddr)
		// Return error on failure for test compatibility
		return result, fmt.Errorf("pipeline retry failed after %d attempts", result.Attempts)
	}
}

// retrySequential tries techniques one by one
func (pr *PipelineRetry) retrySequential(ctx context.Context, req *SOCKS5Request, pipe *Pipeline) *RetryResult {
	result := &RetryResult{Attempts: 0}

	// Try each modifier in sequence
	for i, modifier := range pipe.modifiers {
		if result.Attempts >= pr.config.MaxAttempts {
			break
		}

		technique := modifier.Name()
		log.Printf("Attempting technique %d/%d: %s", i+1, len(pipe.modifiers), technique)

		conn, err := pr.connectWithTechnique(ctx, req, modifier)
		result.Attempts++

		if err == nil && conn != nil {
			result.Success = true
			result.Technique = technique
			result.Connection = conn
			return result
		}

		log.Printf("Technique %s failed: %v", technique, err)

		// Add delay between attempts
		if result.Attempts < pr.config.MaxAttempts {
			time.Sleep(pr.config.RetryDelay)
		}
	}

	result.Error = fmt.Errorf("all %d techniques failed", result.Attempts)
	return result
}

// retryParallel tries multiple techniques in parallel
func (pr *PipelineRetry) retryParallel(ctx context.Context, req *SOCKS5Request, pipe *Pipeline) *RetryResult {
	result := &RetryResult{Attempts: 0}

	// Limit parallel attempts to avoid overwhelming the system
	maxParallel := min(len(pipe.modifiers), 3)
	if maxParallel == 0 {
		result.Error = fmt.Errorf("no modifiers available")
		return result
	}

	type attemptResult struct {
		technique string
		conn      net.Conn
		err       error
		attempt   int
	}

	results := make(chan attemptResult, maxParallel)

	// Start parallel attempts
	for i := 0; i < maxParallel && result.Attempts < pr.config.MaxAttempts; i++ {
		modifier := pipe.modifiers[i]
		go func(mod Modifier, attemptNum int) {
			conn, err := pr.connectWithTechnique(ctx, req, mod)
			results <- attemptResult{
				technique: mod.Name(),
				conn:      conn,
				err:       err,
				attempt:   attemptNum,
			}
		}(modifier, result.Attempts+1)
		result.Attempts++
	}

	// Wait for first successful result or timeout
	timeout := time.After(pr.config.Timeout)

	for {
		select {
		case res := <-results:
			if res.err == nil && res.conn != nil {
				result.Success = true
				result.Technique = res.technique
				result.Connection = res.conn
				return result
			}
			log.Printf("Parallel attempt %s failed: %v", res.technique, res.err)

		case <-timeout:
			result.Error = fmt.Errorf("parallel retry timeout after %v", pr.config.Timeout)
			return result

		case <-ctx.Done():
			result.Error = ctx.Err()
			return result
		}
	}
}

// retryAdaptive uses intelligent technique selection based on success history
func (pr *PipelineRetry) retryAdaptive(ctx context.Context, req *SOCKS5Request, pipe *Pipeline) *RetryResult {
	result := &RetryResult{Attempts: 0}

	// Sort modifiers by success rate (simple implementation - could be enhanced with ML)
	modifiers := pr.sortModifiersBySuccessRate(pipe.modifiers)

	for i, modifier := range modifiers {
		if result.Attempts >= pr.config.MaxAttempts {
			break
		}

		technique := modifier.Name()
		log.Printf("Adaptive attempt %d/%d: %s (success rate: %.2f%%)",
			i+1, len(modifiers), technique, pr.getSuccessRate(technique))

		conn, err := pr.connectWithTechnique(ctx, req, modifier)
		result.Attempts++

		if err == nil && conn != nil {
			result.Success = true
			result.Technique = technique
			result.Connection = conn
			pr.updateSuccessCount(technique)
			return result
		}

		log.Printf("Adaptive technique %s failed: %v", technique, err)
		pr.updateFailureCountForTechnique(technique)

		// Adaptive delay based on technique performance
		delay := pr.calculateAdaptiveDelay(technique)
		if result.Attempts < pr.config.MaxAttempts {
			time.Sleep(delay)
		}
	}

	result.Error = fmt.Errorf("all adaptive attempts failed (%d attempts)", result.Attempts)
	return result
}

// connectWithTechnique attempts connection using a specific modifier
func (pr *PipelineRetry) connectWithTechnique(ctx context.Context, req *SOCKS5Request, modifier Modifier) (net.Conn, error) {
	// Create a copy of the request for modification
	modifiedReq := *req

	// Apply pre-connection modification if supported
	if preConnModifier, ok := modifier.(PreConnectionModifier); ok {
		if newReq := preConnModifier.PreConnection(&modifiedReq); newReq != nil {
			modifiedReq = *newReq
		}
	}

	// Attempt connection with modified request
	conn, err := pr.connectDirectly(ctx, &modifiedReq)
	if err != nil {
		return nil, err
	}

	// If successful, wrap connection with modifier for data processing
	return &ModifiedConnection{
		Conn:     conn,
		Modifier: modifier,
		Request:  &modifiedReq,
	}, nil
}

// connectDirectly performs the actual connection
func (pr *PipelineRetry) connectDirectly(ctx context.Context, req *SOCKS5Request) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: pr.config.Timeout,
	}

	address := fmt.Sprintf("%s:%d", req.DstAddr, req.DstPort)
	return dialer.DialContext(ctx, "tcp", address)
}

// sortModifiersBySuccessRate sorts modifiers by historical success rate
func (pr *PipelineRetry) sortModifiersBySuccessRate(modifiers []Modifier) []Modifier {
	// Simple implementation - could be enhanced with ML-based prediction
	sorted := make([]Modifier, len(modifiers))
	copy(sorted, modifiers)

	// For now, prioritize fragmentation and headers (generally more successful)
	// This could be replaced with ML-based prediction
	priorityOrder := map[string]int{
		"fragmentation": 1,
		"headers":       2,
		"encryption":    3,
		"protocol_mask": 4,
	}

	// Sort by priority
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			priority1 := priorityOrder[sorted[i].Name()]
			priority2 := priorityOrder[sorted[j].Name()]

			if priority1 == 0 {
				priority1 = 999
			}
			if priority2 == 0 {
				priority2 = 999
			}

			if priority1 > priority2 {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// getSuccessRate returns success rate for a technique (placeholder implementation)
func (pr *PipelineRetry) getSuccessRate(technique string) float64 {
	// This could be enhanced with actual statistics
	// For now, return default success rates based on technique
	defaultRates := map[string]float64{
		"fragmentation": 85.0,
		"headers":       80.0,
		"encryption":    75.0,
		"protocol_mask": 70.0,
	}

	if rate, exists := defaultRates[technique]; exists {
		return rate
	}
	return 50.0
}

// updateSuccessCount updates success statistics
func (pr *PipelineRetry) updateSuccessCount(technique string) {
	key := fmt.Sprintf("success_%s", technique)
	pr.attemptCount[key]++
}

// updateFailureCountForTechnique updates failure statistics for a technique
func (pr *PipelineRetry) updateFailureCountForTechnique(technique string) {
	key := fmt.Sprintf("failure_%s", technique)
	pr.attemptCount[key]++
}

// updateFailureCount updates failure count for circuit breaker
func (pr *PipelineRetry) updateFailureCount(target string) {
	pr.failureCount[target]++

	// Check if circuit breaker should be triggered
	if pr.config.CircuitBreaker && pr.failureCount[target] >= pr.config.FailureThreshold {
		log.Printf("Circuit breaker triggered for target %s after %d failures",
			target, pr.failureCount[target])
	}
}

// calculateAdaptiveDelay calculates delay based on technique performance
func (pr *PipelineRetry) calculateAdaptiveDelay(technique string) time.Duration {
	successRate := pr.getSuccessRate(technique)

	// Higher delay for less successful techniques
	if successRate > 80 {
		return pr.config.RetryDelay
	} else if successRate > 60 {
		return pr.config.RetryDelay * 2
	} else {
		return pr.config.RetryDelay * 3
	}
}

// IsCircuitBreakerOpen checks if circuit breaker is open for target
func (pr *PipelineRetry) IsCircuitBreakerOpen(target string) bool {
	if !pr.config.CircuitBreaker {
		return false
	}

	return pr.failureCount[target] >= pr.config.FailureThreshold
}

// ResetCircuitBreaker resets failure count for target
func (pr *PipelineRetry) ResetCircuitBreaker(target string) {
	delete(pr.failureCount, target)
	log.Printf("Circuit breaker reset for target %s", target)
}

// GetStatistics returns retry statistics
func (pr *PipelineRetry) GetStatistics() map[string]interface{} {
	stats := make(map[string]interface{})

	// Calculate success rates for each technique
	techniques := []string{"fragmentation", "headers", "encryption", "protocol_mask"}

	for _, technique := range techniques {
		successKey := fmt.Sprintf("success_%s", technique)
		failureKey := fmt.Sprintf("failure_%s", technique)

		successes := pr.attemptCount[successKey]
		failures := pr.attemptCount[failureKey]
		total := successes + failures

		if total > 0 {
			stats[technique+"_success_rate"] = float64(successes) / float64(total) * 100
			stats[technique+"_total_attempts"] = total
		} else {
			stats[technique+"_success_rate"] = 0.0
			stats[technique+"_total_attempts"] = 0
		}
	}

	stats["circuit_breaker_targets"] = len(pr.failureCount)
	stats["total_circuit_breaker_triggers"] = len(pr.failureCount)

	return stats
}

// PreConnectionModifier interface for modifiers that can modify requests before connection
type PreConnectionModifier interface {
	Modifier
	PreConnection(req *SOCKS5Request) *SOCKS5Request
}

// ModifiedConnection wraps a connection with a modifier
type ModifiedConnection struct {
	net.Conn
	Modifier Modifier
	Request  *SOCKS5Request
}

func (mc *ModifiedConnection) Read(b []byte) (n int, err error) {
	n, err = mc.Conn.Read(b)
	if err != nil {
		return n, err
	}

	// Process inbound data
	modified := mc.Modifier.Process(b[:n], DirectionInbound)
	copy(b, modified)
	return len(modified), nil
}

func (mc *ModifiedConnection) Write(b []byte) (n int, err error) {
	// Process outbound data
	modified := mc.Modifier.Process(b, DirectionOutbound)
	return mc.Conn.Write(modified)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
