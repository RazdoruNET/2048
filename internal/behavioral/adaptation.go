package behavioral

import (
	"math"
	"sync"
	"time"
)

// RealtimeAdapter provides real-time adaptation capabilities
type RealtimeAdapter struct {
	performanceTracker *PerformanceTracker
	strategySelector   *StrategySelector
	adaptationEngine   *AdaptationEngine
	learningModule     *LearningModule
	mu                 sync.RWMutex
	enabled            bool
	adaptationRate     float64
	lastAdaptation     time.Time
}

// PerformanceTracker tracks performance metrics
type PerformanceTracker struct {
	successRate        float64
	failureRate        float64
	averageLatency     time.Duration
	detectionEvents    int
	totalRequests      int
	performanceHistory []PerformanceMetric
	mu                 sync.RWMutex
	windowSize         int
}

// PerformanceMetric represents a single performance metric
type PerformanceMetric struct {
	Timestamp    time.Time
	Success      bool
	Latency      time.Duration
	DPILevel     int
	StrategyUsed string
	Confidence   float64
}

// StrategySelector selects optimal strategies based on performance
type StrategySelector struct {
	strategies       map[string]*Strategy
	currentStrategy  string
	strategyWeights  map[string]float64
	performanceCache map[string]*StrategyPerformance
	mu               sync.RWMutex
	selectionWindow  time.Duration
}

// Strategy represents an evasion strategy
type Strategy struct {
	Name            string
	Description     string
	Weight          float64
	SuccessRate     float64
	LastUsed        time.Time
	UsageCount      int
	Configuration   map[string]interface{}
}

// StrategyPerformance tracks strategy-specific performance
type StrategyPerformance struct {
	StrategyName    string
	SuccessRate     float64
	AverageLatency  time.Duration
	Effectiveness   float64
	LastUpdated     time.Time
	SampleCount     int
}

// AdaptationEngine handles adaptation logic
type AdaptationEngine struct {
	adaptationRules []AdaptationRule
	learningRate     float64
	minConfidence    float64
	maxAdaptation    int
	mu               sync.RWMutex
}

// AdaptationRule represents an adaptation rule
type AdaptationRule struct {
	Name        string
	Condition   func(PerformanceMetric) bool
	Action      func(*RealtimeAdapter, PerformanceMetric)
	Priority    int
	Enabled     bool
}

// LearningModule provides ML-based learning capabilities
type LearningModule struct {
	neuralNetwork    *SimpleNeuralNetwork
	trainingData     []TrainingSample
	predictionCache  map[string]float64
	learningRate     float64
	epochs           int
	mu               sync.RWMutex
}

// SimpleNeuralNetwork represents a simple neural network for predictions
type SimpleNeuralNetwork struct {
	inputSize    int
	hiddenSize   int
	outputSize   int
	weightsIH    [][]float64 // Input to Hidden
	weightsHO    [][]float64 // Hidden to Output
	biasH        []float64   // Hidden bias
	biasO        []float64   // Output bias
	learningRate  float64
}

// TrainingSample represents a training sample
type TrainingSample struct {
	Input  []float64
	Output float64
}

// NewRealtimeAdapter creates new real-time adapter
func NewRealtimeAdapter() *RealtimeAdapter {
	return &RealtimeAdapter{
		performanceTracker: NewPerformanceTracker(),
		strategySelector:   NewStrategySelector(),
		adaptationEngine:   NewAdaptationEngine(),
		learningModule:     NewLearningModule(),
		enabled:            true,
		adaptationRate:     0.1,
		lastAdaptation:     time.Now(),
	}
}

// NewPerformanceTracker creates new performance tracker
func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		performanceHistory: make([]PerformanceMetric, 0, 1000),
		windowSize:         100,
	}
}

// NewStrategySelector creates new strategy selector
func NewStrategySelector() *StrategySelector {
	strategies := map[string]*Strategy{
		"timing_obfuscation": {
			Name:        "timing_obfuscation",
			Description: "Advanced timing patterns",
			Weight:      0.25,
			SuccessRate: 0.8,
			Configuration: map[string]interface{}{
				"min_delay":    50 * time.Millisecond,
				"max_delay":    500 * time.Millisecond,
				"randomization": 0.3,
			},
		},
		"traffic_obfuscation": {
			Name:        "traffic_obfuscation",
			Description: "ML-optimized traffic obfuscation",
			Weight:      0.3,
			SuccessRate: 0.85,
			Configuration: map[string]interface{}{
				"noise_ratio":    0.1,
				"pattern_complexity": 3,
			},
		},
		"fingerprinting": {
			Name:        "fingerprinting",
			Description: "Fingerprinting protection",
			Weight:      0.2,
			SuccessRate: 0.9,
			Configuration: map[string]interface{}{
				"protection_level": 3,
				"rotation_time":    30 * time.Minute,
			},
		},
		"adaptive_ml": {
			Name:        "adaptive_ml",
			Description: "ML-based adaptive strategies",
			Weight:      0.25,
			SuccessRate: 0.75,
			Configuration: map[string]interface{}{
				"learning_rate": 0.01,
				"prediction_threshold": 0.7,
			},
		},
	}

	return &StrategySelector{
		strategies:       strategies,
		currentStrategy:  "traffic_obfuscation",
		strategyWeights:  map[string]float64{},
		performanceCache: make(map[string]*StrategyPerformance),
		selectionWindow:  5 * time.Minute,
	}
}

// NewAdaptationEngine creates new adaptation engine
func NewAdaptationEngine() *AdaptationEngine {
	return &AdaptationEngine{
		adaptationRules: []AdaptationRule{
			{
				Name: "high_failure_rate",
				Condition: func(metric PerformanceMetric) bool {
					return !metric.Success
				},
				Action: func(adapter *RealtimeAdapter, metric PerformanceMetric) {
					adapter.SwitchStrategy("adaptive_ml")
				},
				Priority: 1,
				Enabled:  true,
			},
			{
				Name: "high_latency",
				Condition: func(metric PerformanceMetric) bool {
					return metric.Latency > 200*time.Millisecond
				},
				Action: func(adapter *RealtimeAdapter, metric PerformanceMetric) {
					adapter.AdjustStrategyWeight(metric.StrategyUsed, -0.1)
				},
				Priority: 2,
				Enabled:  true,
			},
			{
				Name: "low_success_rate",
				Condition: func(metric PerformanceMetric) bool {
					return false // Will be checked in adaptation loop
				},
				Action: func(adapter *RealtimeAdapter, metric PerformanceMetric) {
					adapter.IncreaseAdaptationRate()
				},
				Priority: 3,
				Enabled:  true,
			},
		},
		learningRate:  0.01,
		minConfidence: 0.7,
		maxAdaptation: 10,
	}
}

// NewLearningModule creates new learning module
func NewLearningModule() *LearningModule {
	return &LearningModule{
		neuralNetwork:   NewSimpleNeuralNetwork(5, 10, 1),
		trainingData:    make([]TrainingSample, 0, 1000),
		predictionCache: make(map[string]float64),
		learningRate:    0.01,
		epochs:          100,
	}
}

// NewSimpleNeuralNetwork creates new simple neural network
func NewSimpleNeuralNetwork(inputSize, hiddenSize, outputSize int) *SimpleNeuralNetwork {
	nn := &SimpleNeuralNetwork{
		inputSize:   inputSize,
		hiddenSize:  hiddenSize,
		outputSize:  outputSize,
		learningRate: 0.01,
		weightsIH:    make([][]float64, hiddenSize),
		weightsHO:    make([][]float64, outputSize),
		biasH:        make([]float64, hiddenSize),
		biasO:        make([]float64, outputSize),
	}

	// Initialize weights with random values
	for i := 0; i < hiddenSize; i++ {
		nn.weightsIH[i] = make([]float64, inputSize)
		nn.biasH[i] = (float64(i) - float64(hiddenSize)/2) * 0.1
		for j := 0; j < inputSize; j++ {
			nn.weightsIH[i][j] = (float64(i+j) - float64(hiddenSize+inputSize)/2) * 0.1
		}
	}

	for i := 0; i < outputSize; i++ {
		nn.weightsHO[i] = make([]float64, hiddenSize)
		nn.biasO[i] = (float64(i) - float64(outputSize)/2) * 0.1
		for j := 0; j < hiddenSize; j++ {
			nn.weightsHO[i][j] = (float64(i+j) - float64(outputSize+hiddenSize)/2) * 0.1
		}
	}

	return nn
}

// TrackPerformance tracks performance metrics
func (ra *RealtimeAdapter) TrackPerformance(success bool, latency time.Duration, dpiLevel int, strategy string) {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	if !ra.enabled {
		return
	}

	metric := PerformanceMetric{
		Timestamp:    time.Now(),
		Success:      success,
		Latency:      latency,
		DPILevel:     dpiLevel,
		StrategyUsed: strategy,
		Confidence:   ra.predictSuccess(dpiLevel, strategy),
	}

	ra.performanceTracker.AddMetric(metric)
	ra.strategySelector.UpdateStrategyPerformance(strategy, metric)
	ra.learningModule.AddTrainingSample(metric)

	// Trigger adaptation if needed
	if time.Since(ra.lastAdaptation) > time.Minute {
		ra.adapt()
		ra.lastAdaptation = time.Now()
	}
}

// AddMetric adds performance metric
func (pt *PerformanceTracker) AddMetric(metric PerformanceMetric) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.performanceHistory = append(pt.performanceHistory, metric)
	pt.totalRequests++

	if metric.Success {
		pt.successRate = float64(pt.totalRequests-pt.detectionEvents) / float64(pt.totalRequests)
	} else {
		pt.detectionEvents++
		pt.failureRate = float64(pt.detectionEvents) / float64(pt.totalRequests)
	}

	// Update average latency
	if len(pt.performanceHistory) > 0 {
		var totalLatency time.Duration
		count := len(pt.performanceHistory)
		if count > pt.windowSize {
			count = pt.windowSize
		}

		for i := len(pt.performanceHistory) - count; i < len(pt.performanceHistory); i++ {
			totalLatency += pt.performanceHistory[i].Latency
		}
		pt.averageLatency = totalLatency / time.Duration(count)
	}

	// Maintain window size
	if len(pt.performanceHistory) > pt.windowSize*2 {
		pt.performanceHistory = pt.performanceHistory[len(pt.performanceHistory)-pt.windowSize:]
	}
}

// GetSuccessRate returns current success rate
func (pt *PerformanceTracker) GetSuccessRate() float64 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.successRate
}

// GetAverageLatency returns current average latency
func (pt *PerformanceTracker) GetAverageLatency() time.Duration {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.averageLatency
}

// UpdateStrategyPerformance updates strategy performance
func (ss *StrategySelector) UpdateStrategyPerformance(strategy string, metric PerformanceMetric) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	perf, exists := ss.performanceCache[strategy]
	if !exists {
		perf = &StrategyPerformance{
			StrategyName: strategy,
			SampleCount:   0,
		}
		ss.performanceCache[strategy] = perf
	}

	perf.SampleCount++
	perf.LastUpdated = time.Now()

	// Update success rate
	if metric.Success {
		perf.SuccessRate = (perf.SuccessRate*float64(perf.SampleCount-1) + 1.0) / float64(perf.SampleCount)
	} else {
		perf.SuccessRate = (perf.SuccessRate*float64(perf.SampleCount-1)) / float64(perf.SampleCount)
	}

	// Update average latency
	perf.AverageLatency = (perf.AverageLatency*time.Duration(perf.SampleCount-1) + metric.Latency) / time.Duration(perf.SampleCount)

	// Update effectiveness
	perf.Effectiveness = perf.SuccessRate * (1.0 - math.Min(float64(metric.Latency)/float64(time.Second), 1.0))
}

// SelectOptimalStrategy selects optimal strategy based on current conditions
func (ss *StrategySelector) SelectOptimalStrategy(dpiLevel int) string {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	bestStrategy := ss.currentStrategy
	bestScore := 0.0

	for name, strategy := range ss.strategies {
		perf, exists := ss.performanceCache[name]
		score := strategy.Weight

		if exists {
			score *= perf.Effectiveness
		}

		// Adjust score based on DPI level
		if dpiLevel > 3 && name == "adaptive_ml" {
			score *= 1.2
		} else if dpiLevel <= 2 && name == "timing_obfuscation" {
			score *= 1.1
		}

		if score > bestScore {
			bestScore = score
			bestStrategy = name
		}
	}

	return bestStrategy
}

// SwitchStrategy switches to a new strategy
func (ra *RealtimeAdapter) SwitchStrategy(strategy string) {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	ra.strategySelector.currentStrategy = strategy
}

// AdjustStrategyWeight adjusts strategy weight
func (ra *RealtimeAdapter) AdjustStrategyWeight(strategy string, delta float64) {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	if strat, exists := ra.strategySelector.strategies[strategy]; exists {
		strat.Weight = math.Max(0.1, math.Min(1.0, strat.Weight+delta))
	}
}

// IncreaseAdaptationRate increases adaptation rate
func (ra *RealtimeAdapter) IncreaseAdaptationRate() {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	ra.adaptationRate = math.Min(0.5, ra.adaptationRate*1.2)
}

// adapt performs adaptation based on current performance
func (ra *RealtimeAdapter) adapt() {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	if !ra.enabled {
		return
	}

	// Get recent performance metrics
	recentMetrics := ra.performanceTracker.performanceHistory
	if len(recentMetrics) < 10 {
		return
	}

	// Apply adaptation rules
	for _, rule := range ra.adaptationEngine.adaptationRules {
		if !rule.Enabled {
			continue
		}

		for _, metric := range recentMetrics[len(recentMetrics)-10:] {
			// Special handling for low_success_rate rule
			if rule.Name == "low_success_rate" {
				if ra.performanceTracker.GetSuccessRate() < 0.6 {
					rule.Action(ra, metric)
					break
				}
			} else if rule.Condition(metric) {
				rule.Action(ra, metric)
				break
			}
		}
	}

	// Update strategy selection
	currentDPI := recentMetrics[len(recentMetrics)-1].DPILevel
	optimalStrategy := ra.strategySelector.SelectOptimalStrategy(currentDPI)
	if optimalStrategy != ra.strategySelector.currentStrategy {
		ra.SwitchStrategy(optimalStrategy)
	}

	// Train neural network
	if len(ra.learningModule.trainingData) >= 50 {
		ra.learningModule.Train()
	}
}

// predictSuccess predicts success probability
func (ra *RealtimeAdapter) predictSuccess(dpiLevel int, strategy string) float64 {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	input := []float64{
		float64(dpiLevel) / 10.0,
		ra.performanceTracker.GetSuccessRate(),
		float64(ra.performanceTracker.GetAverageLatency()) / float64(time.Second),
		ra.adaptationRate,
		ra.strategySelector.strategies[strategy].Weight,
	}

	return ra.learningModule.Predict(input)
}

// AddTrainingSample adds training sample
func (lm *LearningModule) AddTrainingSample(metric PerformanceMetric) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	input := []float64{
		float64(metric.DPILevel) / 10.0,
		metric.Confidence,
		float64(metric.Latency) / float64(time.Second),
		0.1, // adaptation rate placeholder
		0.25, // strategy weight placeholder
	}

	output := 0.0
	if metric.Success {
		output = 1.0
	}

	sample := TrainingSample{
		Input:  input,
		Output: output,
	}

	lm.trainingData = append(lm.trainingData, sample)

	// Maintain training data size
	if len(lm.trainingData) > 1000 {
		lm.trainingData = lm.trainingData[len(lm.trainingData)-1000:]
	}
}

// Predict makes prediction using neural network
func (nn *SimpleNeuralNetwork) Predict(input []float64) float64 {
	// Forward propagation
	hidden := make([]float64, nn.hiddenSize)
	for i := 0; i < nn.hiddenSize; i++ {
		sum := nn.biasH[i]
		for j := 0; j < nn.inputSize; j++ {
			sum += nn.weightsIH[i][j] * input[j]
		}
		hidden[i] = sigmoid(sum)
	}

	// Output layer
	output := make([]float64, nn.outputSize)
	for i := 0; i < nn.outputSize; i++ {
		sum := nn.biasO[i]
		for j := 0; j < nn.hiddenSize; j++ {
			sum += nn.weightsHO[i][j] * hidden[j]
		}
		output[i] = sigmoid(sum)
	}

	return output[0]
}

// Train trains the neural network
func (lm *LearningModule) Train() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if len(lm.trainingData) < 10 {
		return
	}

	for epoch := 0; epoch < lm.epochs; epoch++ {
		for _, sample := range lm.trainingData {
			lm.neuralNetwork.Backpropagate(sample.Input, []float64{sample.Output})
		}
	}
}

// Backpropagate performs backpropagation
func (nn *SimpleNeuralNetwork) Backpropagate(input []float64, target []float64) {
	// Forward pass
	hidden := make([]float64, nn.hiddenSize)
	for i := 0; i < nn.hiddenSize; i++ {
		sum := nn.biasH[i]
		for j := 0; j < nn.inputSize; j++ {
			sum += nn.weightsIH[i][j] * input[j]
		}
		hidden[i] = sigmoid(sum)
	}

	output := make([]float64, nn.outputSize)
	for i := 0; i < nn.outputSize; i++ {
		sum := nn.biasO[i]
		for j := 0; j < nn.hiddenSize; j++ {
			sum += nn.weightsHO[i][j] * hidden[j]
		}
		output[i] = sigmoid(sum)
	}

	// Calculate output error
	outputError := make([]float64, nn.outputSize)
	for i := 0; i < nn.outputSize; i++ {
		outputError[i] = (target[i] - output[i]) * sigmoidDerivative(output[i])
	}

	// Calculate hidden error
	hiddenError := make([]float64, nn.hiddenSize)
	for i := 0; i < nn.hiddenSize; i++ {
		errorSum := 0.0
		for j := 0; j < nn.outputSize; j++ {
			errorSum += outputError[j] * nn.weightsHO[j][i]
		}
		hiddenError[i] = errorSum * sigmoidDerivative(hidden[i])
	}

	// Update weights
	for i := 0; i < nn.outputSize; i++ {
		for j := 0; j < nn.hiddenSize; j++ {
			nn.weightsHO[i][j] += nn.learningRate * outputError[i] * hidden[j]
		}
		nn.biasO[i] += nn.learningRate * outputError[i]
	}

	for i := 0; i < nn.hiddenSize; i++ {
		for j := 0; j < nn.inputSize; j++ {
			nn.weightsIH[i][j] += nn.learningRate * hiddenError[i] * input[j]
		}
		nn.biasH[i] += nn.learningRate * hiddenError[i]
	}
}

// sigmoid activation function
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// sigmoidDerivative derivative of sigmoid function
func sigmoidDerivative(x float64) float64 {
	return x * (1.0 - x)
}

// Predict makes prediction using learning module
func (lm *LearningModule) Predict(input []float64) float64 {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return lm.neuralNetwork.Predict(input)
}

// GetAdaptationStats returns adaptation statistics
func (ra *RealtimeAdapter) GetAdaptationStats() map[string]interface{} {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	return map[string]interface{}{
		"success_rate":      ra.performanceTracker.GetSuccessRate(),
		"average_latency":   ra.performanceTracker.GetAverageLatency(),
		"current_strategy":  ra.strategySelector.currentStrategy,
		"adaptation_rate":   ra.adaptationRate,
		"total_requests":    ra.performanceTracker.totalRequests,
		"detection_events":  ra.performanceTracker.detectionEvents,
		"enabled":           ra.enabled,
	}
}

// Enable enables real-time adaptation
func (ra *RealtimeAdapter) Enable() {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	ra.enabled = true
}

// Disable disables real-time adaptation
func (ra *RealtimeAdapter) Disable() {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	ra.enabled = false
}

// SetEnabled enables or disables real-time adaptation
func (ra *RealtimeAdapter) SetEnabled(enabled bool) {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	ra.enabled = enabled
}

// IsEnabled returns whether real-time adaptation is enabled
func (ra *RealtimeAdapter) IsEnabled() bool {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	return ra.enabled
}
